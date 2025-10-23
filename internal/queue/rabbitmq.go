package queue

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"mercury-relay/internal/config"
	"mercury-relay/internal/models"

	"github.com/rabbitmq/amqp091-go"
)

// KindConfig represents the structure of the quality control configuration
type KindConfig struct {
	EventKinds map[string]interface{} `yaml:"event_kinds"`
}

// loadKindsFromConfig loads kind configurations from individual YAML files in configs/kinds/
func loadKindsFromConfig(configPath string) ([]int, error) {
	// Default to the kinds directory if no path provided
	if configPath == "" {
		configPath = "configs/kinds"
	}

	// Check if directory exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Fallback to hardcoded kinds if directory doesn't exist
		return []int{0, 1, 3, 7, 10002}, nil
	}

	// Read all .yml files in the kinds directory
	files, err := os.ReadDir(configPath)
	if err != nil {
		// Fallback to hardcoded kinds if directory can't be read
		return []int{0, 1, 3, 7, 10002}, nil
	}

	var kinds []int
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".yml") {
			// Extract kind number from filename (e.g., "0.yml" -> 0)
			kindStr := strings.TrimSuffix(file.Name(), ".yml")
			if kind, err := strconv.Atoi(kindStr); err == nil {
				kinds = append(kinds, kind)
			}
		}
	}

	// If no kinds were loaded, fallback to hardcoded
	if len(kinds) == 0 {
		return []int{0, 1, 3, 7, 10002}, nil
	}

	return kinds, nil
}

// getCommonKinds returns the list of Nostr event kinds from quality control configuration
func getCommonKinds() []int {
	kinds, err := loadKindsFromConfig("")
	if err != nil {
		// Fallback to hardcoded kinds
		return []int{0, 1, 3, 7, 10002}
	}
	return kinds
}

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

	// Get the common Nostr event types that have dedicated topics
	commonKinds := getCommonKinds()

	// Always create an "undefined" queue for unknown kinds and "moderation" queue for invalid events
	allKinds := append(commonKinds, -1, -2) // -1 represents undefined/unknown kinds, -2 represents moderation

	for _, kind := range allKinds {
		var queueName string
		var routingKey string

		if kind == -1 {
			queueName = "nostr_kind_undefined"
			routingKey = "kind.undefined"
		} else if kind == -2 {
			queueName = "nostr_kind_moderation"
			routingKey = "kind.moderation"
		} else {
			queueName = fmt.Sprintf("nostr_kind_%d", kind)
			routingKey = fmt.Sprintf("kind.%d", kind)
		}

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

// PublishToKindTopic routes an event to the appropriate kind-based topic with quality control
func (r *RabbitMQ) PublishToKindTopic(event *models.Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Determine routing key based on kind and quality control
	var routingKey string

	// First, check if the event is valid (basic validation)
	if !r.isValidEvent(event) {
		// Invalid events go to moderation topic
		routingKey = "kind.moderation"
	} else {
		// Check if this is a known kind from configuration
		commonKinds := getCommonKinds()
		isKnown := false
		for _, knownKind := range commonKinds {
			if event.Kind == knownKind {
				isKnown = true
				break
			}
		}

		if isKnown {
			// Known kinds go to their specific topic
			routingKey = fmt.Sprintf("kind.%d", event.Kind)
		} else {
			// Unknown kinds go to undefined topic
			routingKey = "kind.undefined"
		}
	}

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

// isValidEvent performs basic validation on an event
func (r *RabbitMQ) isValidEvent(event *models.Event) bool {
	// Basic validation checks
	if event == nil {
		return false
	}

	// Check required fields
	if event.ID == "" || event.PubKey == "" || event.Sig == "" {
		return false
	}

	// Check timestamp is reasonable (not too far in past/future)
	now := int64(time.Now().Unix())
	createdAt := int64(event.CreatedAt)
	if createdAt < now-86400*365 || createdAt > now+86400 { // Within 1 year
		return false
	}

	// Check kind is reasonable
	if event.Kind < 0 || event.Kind > 65535 {
		return false
	}

	// Additional validation can be added here
	// For now, we'll consider events with basic structure as valid

	return true
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
	var queueName string

	// Check if this is a known kind from configuration
	commonKinds := getCommonKinds()
	isKnown := false
	for _, knownKind := range commonKinds {
		if kind == knownKind {
			isKnown = true
			break
		}
	}

	if isKnown {
		queueName = fmt.Sprintf("nostr_kind_%d", kind)
	} else {
		// For undefined kinds, use the undefined queue
		queueName = "nostr_kind_undefined"
	}

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
	var queueName string

	// Handle special cases first
	if kind == -1 {
		queueName = "nostr_kind_undefined"
	} else if kind == -2 {
		queueName = "nostr_kind_moderation"
	} else {
		// Check if this is a known kind from configuration
		commonKinds := getCommonKinds()
		isKnown := false
		for _, knownKind := range commonKinds {
			if kind == knownKind {
				isKnown = true
				break
			}
		}

		if isKnown {
			// Known kinds have dedicated queues
			queueName = fmt.Sprintf("nostr_kind_%d", kind)
		} else {
			// Unknown kinds use the undefined queue
			queueName = "nostr_kind_undefined"
		}
	}

	queue, err := r.channel.QueueInspect(queueName)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect kind queue %s: %w", queueName, err)
	}
	return queue.Messages, nil
}

// GetAllKindQueueStats returns stats for all kind queues
func (r *RabbitMQ) GetAllKindQueueStats() (map[int]int, error) {
	stats := make(map[int]int)
	commonKinds := getCommonKinds()

	// Get stats for common kinds
	for _, kind := range commonKinds {
		count, err := r.GetKindQueueStats(kind)
		if err != nil {
			return nil, fmt.Errorf("failed to get stats for kind %d: %w", kind, err)
		}
		stats[kind] = count
	}

	// Get stats for undefined kinds
	undefinedCount, err := r.GetKindQueueStats(-1) // -1 represents undefined
	if err != nil {
		return nil, fmt.Errorf("failed to get stats for undefined kinds: %w", err)
	}
	stats[-1] = undefinedCount

	// Get stats for moderation queue
	moderationCount, err := r.GetKindQueueStats(-2) // -2 represents moderation
	if err != nil {
		return nil, fmt.Errorf("failed to get stats for moderation queue: %w", err)
	}
	stats[-2] = moderationCount

	return stats, nil
}
