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
	conn         *amqp091.Connection
	channel      *amqp091.Channel
	config       config.RabbitMQConfig
	kindExchange string
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

	// Create kind-based topic exchange for routing by event kind
	kindExchangeName := "nostr_kinds"
	if err := channel.ExchangeDeclare(
		kindExchangeName,
		"topic", // topic exchange for routing by kind
		true,    // durable
		false,   // auto-delete
		false,   // internal
		false,   // no-wait
		nil,     // arguments
	); err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare kind exchange: %w", err)
	}

	// Create kind-based queues for common Nostr event types
	commonKinds := []int{0, 1, 3, 7, 10002}
	for _, kind := range commonKinds {
		queueName := fmt.Sprintf("nostr_kind_%d", kind)

		// Declare kind-specific queue
		_, err = channel.QueueDeclare(
			queueName,
			true,  // durable
			false, // auto-delete
			false, // exclusive
			false, // no-wait
			nil,   // arguments
		)
		if err != nil {
			channel.Close()
			conn.Close()
			return nil, fmt.Errorf("failed to declare kind queue %s: %w", queueName, err)
		}

		// Bind kind queue to kind exchange
		routingKey := fmt.Sprintf("kind.%d", kind)
		if err := channel.QueueBind(
			queueName,
			routingKey,
			kindExchangeName,
			false, // no-wait
			nil,   // arguments
		); err != nil {
			channel.Close()
			conn.Close()
			return nil, fmt.Errorf("failed to bind kind queue %s: %w", queueName, err)
		}
	}

	return &RabbitMQ{
		conn:         conn,
		channel:      channel,
		config:       config,
		kindExchange: kindExchangeName,
	}, nil
}

func (r *RabbitMQ) PublishEvent(event *models.Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Publish to main exchange
	if err := r.channel.Publish(
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
	); err != nil {
		return fmt.Errorf("failed to publish to main exchange: %w", err)
	}

	// Also route to kind-based topic
	return r.PublishToKindTopic(event)
}

// PublishToKindTopic routes an event to the appropriate kind-based topic
func (r *RabbitMQ) PublishToKindTopic(event *models.Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Route to kind-based exchange
	routingKey := fmt.Sprintf("kind.%d", event.Kind)
	return r.channel.Publish(
		r.kindExchange,
		routingKey,
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

// ConsumeEventsByKind consumes events from a specific kind queue
func (r *RabbitMQ) ConsumeEventsByKind(kind int) ([]*models.Event, error) {
	queueName := fmt.Sprintf("nostr_kind_%d", kind)

	// Use Get method to get messages one at a time
	msg, ok, err := r.channel.Get(queueName, false) // false = no auto-ack
	if err != nil {
		return nil, fmt.Errorf("failed to get message from kind queue %s: %w", queueName, err)
	}
	if !ok {
		// No messages available
		return []*models.Event{}, nil
	}

	var event models.Event
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("Failed to unmarshal event from kind queue: %v", err)
		msg.Nack(false, false) // Reject and don't requeue
		return []*models.Event{}, nil
	}

	// Acknowledge the message after successful processing
	msg.Ack(false)

	return []*models.Event{&event}, nil
}

// GetKindQueueStats returns the number of messages in a specific kind queue
func (r *RabbitMQ) GetKindQueueStats(kind int) (int, error) {
	queueName := fmt.Sprintf("nostr_kind_%d", kind)
	queue, err := r.channel.QueueInspect(queueName)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect kind queue %s: %w", queueName, err)
	}
	return queue.Messages, nil
}

// GetAllKindQueueStats returns stats for all kind queues
func (r *RabbitMQ) GetAllKindQueueStats() (map[int]int, error) {
	stats := make(map[int]int)
	commonKinds := []int{0, 1, 3, 7, 10002}

	for _, kind := range commonKinds {
		count, err := r.GetKindQueueStats(kind)
		if err != nil {
			return nil, fmt.Errorf("failed to get stats for kind %d: %w", kind, err)
		}
		stats[kind] = count
	}

	return stats, nil
}
