package cache

import (
	"context"
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

	// Store event with TTL
	key := fmt.Sprintf("event:%s", event.ID)
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := r.client.Set(ctx, key, data, r.config.TTL).Err(); err != nil {
		return fmt.Errorf("failed to store event: %w", err)
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
			events = append(events, &event)
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
