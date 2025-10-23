package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"mercury-relay/internal/config"
	"mercury-relay/internal/models"

	"github.com/nbd-wtf/go-nostr"
	"github.com/redis/go-redis/v9"
)

type Redis struct {
	client *redis.Client
	config config.RedisConfig
}

func NewRedis(config config.RedisConfig) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Host,
		Password: config.Password,
		DB:       config.DB,
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Redis{
		client: client,
		config: config,
	}, nil
}

func (r *Redis) StoreEvent(event *models.Event) error {
	ctx := context.Background()

	// Check if event already exists (prevent duplicates)
	key := fmt.Sprintf("event:%s", event.ID)
	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to check event existence: %w", err)
	}
	if exists > 0 {
		// Event already exists, don't store duplicate
		return nil
	}

	// Store event with TTL
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := r.client.Set(ctx, key, data, r.config.TTL).Err(); err != nil {
		return fmt.Errorf("failed to store event: %w", err)
	}

	// Handle replaceable events
	if r.isReplaceableEvent(event.Kind) {
		if err := r.storeReplaceableEvent(event); err != nil {
			return fmt.Errorf("failed to store replaceable event: %w", err)
		}
	}

	// Index by author
	authorKey := fmt.Sprintf("author:%s", event.PubKey)
	if err := r.client.SAdd(ctx, authorKey, event.ID).Err(); err != nil {
		return fmt.Errorf("failed to index by author: %w", err)
	}
	r.client.Expire(ctx, authorKey, r.config.TTL)

	// Index by kind
	kindKey := fmt.Sprintf("kind:%d", event.Kind)
	if err := r.client.SAdd(ctx, kindKey, event.ID).Err(); err != nil {
		return fmt.Errorf("failed to index by kind: %w", err)
	}
	r.client.Expire(ctx, kindKey, r.config.TTL)

	// Index by tags
	for _, tag := range event.Tags {
		if len(tag) >= 2 {
			tagKey := fmt.Sprintf("tag:%s:%s", tag[0], tag[1])
			if err := r.client.SAdd(ctx, tagKey, event.ID).Err(); err != nil {
				return fmt.Errorf("failed to index by tag: %w", err)
			}
			r.client.Expire(ctx, tagKey, r.config.TTL)
		}
	}

	return nil
}

func (r *Redis) GetEvents(filter nostr.Filter) ([]*models.Event, error) {
	ctx := context.Background()
	var eventIDs []string

	// Get event IDs based on filter
	if len(filter.Authors) > 0 {
		for _, author := range filter.Authors {
			authorKey := fmt.Sprintf("author:%s", author)
			ids, err := r.client.SMembers(ctx, authorKey).Result()
			if err != nil {
				continue
			}
			eventIDs = append(eventIDs, ids...)
		}
	} else if len(filter.Kinds) > 0 {
		for _, kind := range filter.Kinds {
			kindKey := fmt.Sprintf("kind:%d", kind)
			ids, err := r.client.SMembers(ctx, kindKey).Result()
			if err != nil {
				continue
			}
			eventIDs = append(eventIDs, ids...)
		}
	} else {
		// Get all events (limited)
		keys, err := r.client.Keys(ctx, "event:*").Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get event keys: %w", err)
		}
		for _, key := range keys {
			eventIDs = append(eventIDs, key[6:]) // Remove "event:" prefix
		}
	}

	// Get events
	var events []*models.Event
	for _, id := range eventIDs {
		key := fmt.Sprintf("event:%s", id)
		data, err := r.client.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var event models.Event
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		// Apply additional filters
		if r.eventMatchesFilter(&event, filter) {
			// For replaceable events, only return the latest version
			if r.isReplaceableEvent(event.Kind) {
				latestEvent, err := r.getLatestReplaceableEvent(&event)
				if err != nil {
					continue
				}
				events = append(events, latestEvent)
			} else {
				events = append(events, &event)
			}
		}
	}

	return events, nil
}

func (r *Redis) eventMatchesFilter(event *models.Event, filter nostr.Filter) bool {
	// Check since
	if filter.Since != nil && *filter.Since > 0 {
		if nostr.Timestamp(int64(event.CreatedAt)) < *filter.Since {
			return false
		}
	}

	// Check until
	if filter.Until != nil && *filter.Until > 0 {
		if nostr.Timestamp(int64(event.CreatedAt)) > *filter.Until {
			return false
		}
	}

	// Note: Limit is applied in the calling function

	return true
}

// isReplaceableEvent checks if an event kind is replaceable
func (r *Redis) isReplaceableEvent(kind int) bool {
	replaceableKinds := map[int]bool{
		0:     true, // User metadata
		3:     true, // Contacts
		5:     true, // Event deletion
		6:     true, // Repost
		7:     true, // Reaction
		8:     true, // Badge award
		40:    true, // Channel creation
		41:    true, // Channel metadata
		42:    true, // Channel message
		43:    true, // Hide message
		44:    true, // Mute user
		10002: true, // Relay list
		30000: true, // Follow sets
		30001: true, // Follow sets
		30008: true, // Profile badges
		30009: true, // Badge definition
		30078: true, // Application-specific data
	}
	return replaceableKinds[kind]
}

// storeReplaceableEvent stores a replaceable event with version tracking
func (r *Redis) storeReplaceableEvent(event *models.Event) error {
	ctx := context.Background()

	// Generate replaceable event key (kind:pubkey:d-tag)
	key := r.getReplaceableEventKey(event)

	// Get existing versions
	versionsKey := fmt.Sprintf("replaceable:%s", key)
	existingVersions, err := r.client.LRange(ctx, versionsKey, 0, -1).Result()
	if err != nil {
		return fmt.Errorf("failed to get existing versions: %w", err)
	}

	// Create new version
	version := len(existingVersions) + 1
	eventVersion := map[string]interface{}{
		"event_id":   event.ID,
		"version":    version,
		"created_at": event.CreatedAt,
		"hash":       r.getEventHash(event),
	}

	// Store version data
	versionData, err := json.Marshal(eventVersion)
	if err != nil {
		return fmt.Errorf("failed to marshal version data: %w", err)
	}

	// Add to versions list
	if err := r.client.LPush(ctx, versionsKey, versionData).Err(); err != nil {
		return fmt.Errorf("failed to store version: %w", err)
	}

	// Set TTL for versions
	r.client.Expire(ctx, versionsKey, r.config.TTL)

	// Update latest version pointer
	latestKey := fmt.Sprintf("latest:%s", key)
	if err := r.client.Set(ctx, latestKey, event.ID, r.config.TTL).Err(); err != nil {
		return fmt.Errorf("failed to update latest version: %w", err)
	}

	return nil
}

// getReplaceableEventKey generates the key for replaceable events
func (r *Redis) getReplaceableEventKey(event *models.Event) string {
	// Find d-tag
	var dTag string
	for _, tag := range event.Tags {
		if len(tag) >= 2 && tag[0] == "d" {
			dTag = tag[1]
			break
		}
	}

	return fmt.Sprintf("%d:%s:%s", event.Kind, event.PubKey, dTag)
}

// getEventHash generates a hash for event comparison
func (r *Redis) getEventHash(event *models.Event) string {
	content := fmt.Sprintf("%s:%s:%d:%s", event.ID, event.PubKey, event.Kind, event.Content)
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// GetReplaceableEventHistory returns the history of a replaceable event
func (r *Redis) GetReplaceableEventHistory(kind int, pubkey, dTag string) ([]map[string]interface{}, error) {
	ctx := context.Background()
	key := fmt.Sprintf("%d:%s:%s", kind, pubkey, dTag)
	versionsKey := fmt.Sprintf("replaceable:%s", key)

	versions, err := r.client.LRange(ctx, versionsKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get versions: %w", err)
	}

	var history []map[string]interface{}
	for _, versionData := range versions {
		var version map[string]interface{}
		if err := json.Unmarshal([]byte(versionData), &version); err != nil {
			continue
		}
		history = append(history, version)
	}

	return history, nil
}

// GetLatestReplaceableEvent returns the latest version of a replaceable event
func (r *Redis) GetLatestReplaceableEvent(kind int, pubkey, dTag string) (*models.Event, error) {
	ctx := context.Background()
	key := fmt.Sprintf("%d:%s:%s", kind, pubkey, dTag)
	latestKey := fmt.Sprintf("latest:%s", key)

	eventID, err := r.client.Get(ctx, latestKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest event ID: %w", err)
	}

	// Get the actual event
	eventKey := fmt.Sprintf("event:%s", eventID)
	eventData, err := r.client.Get(ctx, eventKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	var event models.Event
	if err := json.Unmarshal([]byte(eventData), &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	return &event, nil
}

// getLatestReplaceableEvent gets the latest version of a replaceable event
func (r *Redis) getLatestReplaceableEvent(event *models.Event) (*models.Event, error) {
	// Find d-tag
	var dTag string
	for _, tag := range event.Tags {
		if len(tag) >= 2 && tag[0] == "d" {
			dTag = tag[1]
			break
		}
	}

	return r.GetLatestReplaceableEvent(event.Kind, event.PubKey, dTag)
}

func (r *Redis) DeleteEvent(eventID string) error {
	ctx := context.Background()

	// Delete event
	key := fmt.Sprintf("event:%s", eventID)
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	return nil
}

func (r *Redis) GetStats() (map[string]interface{}, error) {
	ctx := context.Background()

	_, err := r.client.Info(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis info: %w", err)
	}

	// Parse info string (simplified)
	stats := map[string]interface{}{
		"connected_clients": "unknown",
		"used_memory":       "unknown",
		"keyspace_hits":     "unknown",
		"keyspace_misses":   "unknown",
	}

	// Count events
	eventKeys, err := r.client.Keys(ctx, "event:*").Result()
	if err == nil {
		stats["total_events"] = len(eventKeys)
	}

	return stats, nil
}

func (r *Redis) Close() error {
	return r.client.Close()
}
