package queue

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"mercury-relay/internal/config"
	"mercury-relay/internal/models"

	"github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
	config  config.RabbitMQConfig
}

func NewRabbitMQ(config config.RabbitMQConfig) (*RabbitMQ, error) {
	conn, err := amqp091.Dial(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare exchange
	if err := channel.ExchangeDeclare(
		config.ExchangeName,
		"fanout",
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,   // arguments
	); err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare dead letter exchange
	if err := channel.ExchangeDeclare(
		config.DLXName,
		"fanout",
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,   // arguments
	); err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare DLX: %w", err)
	}

	// Declare queue
	args := amqp091.Table{
		"x-message-ttl":          int64(config.TTL.Seconds() * 1000), // TTL in milliseconds
		"x-dead-letter-exchange": config.DLXName,
	}

	_, err = channel.QueueDeclare(
		config.QueueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		args,  // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	if err := channel.QueueBind(
		config.QueueName,
		"", // routing key
		config.ExchangeName,
		false, // no-wait
		nil,   // arguments
	); err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	return &RabbitMQ{
		conn:    conn,
		channel: channel,
		config:  config,
	}, nil
}

func (r *RabbitMQ) PublishEvent(event *models.Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return r.channel.Publish(
		r.config.ExchangeName,
		"",    // routing key
		false, // mandatory
		false, // immediate
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        body,
			Timestamp:   time.Now(),
			MessageId:   event.ID,
		},
	)
}

func (r *RabbitMQ) ConsumeEvents() ([]*models.Event, error) {
	// Use Get method to get messages one at a time
	msg, ok, err := r.channel.Get(r.config.QueueName, false) // false = no auto-ack
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}
	if !ok {
		// No messages available
		return []*models.Event{}, nil
	}

	var event models.Event
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("Failed to unmarshal event: %v", err)
		msg.Nack(false, false) // Reject and don't requeue
		return []*models.Event{}, nil
	}

	// Acknowledge the message after successful processing
	msg.Ack(false)

	return []*models.Event{&event}, nil
}

func (r *RabbitMQ) Close() error {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

func (r *RabbitMQ) GetQueueStats() (int, error) {
	queue, err := r.channel.QueueInspect(r.config.QueueName)
	if err != nil {
		return 0, err
	}
	return queue.Messages, nil
}
