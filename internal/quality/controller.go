package quality

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"mercury-relay/internal/cache"
	"mercury-relay/internal/config"
	"mercury-relay/internal/models"
	"mercury-relay/internal/queue"
)

type Controller struct {
	config           config.QualityConfig
	rabbitMQ         queue.Queue
	cache            cache.Cache
	kindConfigLoader *KindConfigLoader

	// Rate limiting
	rateLimiter map[string][]time.Time
	rateMutex   sync.RWMutex

	// Blocked npubs
	blockedNpubs map[string]bool
	blockMutex   sync.RWMutex
}

func NewController(
	config config.QualityConfig,
	rabbitMQ queue.Queue,
	cache cache.Cache,
) *Controller {
	return &Controller{
		config:       config,
		rabbitMQ:     rabbitMQ,
		cache:        cache,
		rateLimiter:  make(map[string][]time.Time),
		blockedNpubs: make(map[string]bool),
	}
}

func (c *Controller) Start(ctx context.Context) error {
	// Start rate limiter cleanup
	go c.cleanupRateLimiter(ctx)

	// Start quality monitoring
	go c.monitorQuality(ctx)

	return nil
}

func (c *Controller) ValidateEvent(event *models.Event) error {
	// Check if npub is blocked
	c.blockMutex.RLock()
	if c.blockedNpubs[event.PubKey] {
		c.blockMutex.RUnlock()
		return fmt.Errorf("npub is blocked")
	}
	c.blockMutex.RUnlock()

	// Check rate limiting
	if err := c.checkRateLimit(event.PubKey); err != nil {
		return fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Check content length
	if len(event.Content) > c.config.MaxContentLength {
		return fmt.Errorf("content too long")
	}

	// Use kind-specific validation if available
	if c.kindConfigLoader != nil {
		// Convert nostr.Tags to [][]string
		tags := make([][]string, len(event.Tags))
		for i, tag := range event.Tags {
			tags[i] = make([]string, len(tag))
			copy(tags[i], tag)
		}

		if err := c.kindConfigLoader.ValidateEventKind(event.Kind, event.Content, tags); err != nil {
			return fmt.Errorf("kind-specific validation failed: %w", err)
		}

		// Calculate quality score using kind config
		if score, err := c.kindConfigLoader.CalculateQualityScore(event.Kind, event.Content, tags); err == nil {
			event.QualityScore = score
		} else {
			event.QualityScore = event.CalculateQualityScore()
		}
	} else {
		// Fallback to default quality calculation
		event.QualityScore = event.CalculateQualityScore()
	}

	if event.QualityScore < c.config.SpamThreshold {
		event.IsQuarantined = true
		event.QuarantineReason = "Low quality score"
	}

	// Publish event to queue
	if err := c.rabbitMQ.PublishEvent(event); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	log.Printf("Quality controller published event %s to queue", event.ID)
	return nil
}

func (c *Controller) checkRateLimit(npub string) error {
	c.rateMutex.Lock()
	defer c.rateMutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-time.Minute)

	// Clean old entries
	if times, exists := c.rateLimiter[npub]; exists {
		var validTimes []time.Time
		for _, t := range times {
			if t.After(cutoff) {
				validTimes = append(validTimes, t)
			}
		}
		c.rateLimiter[npub] = validTimes
	}

	// Check rate limit
	if len(c.rateLimiter[npub]) >= c.config.RateLimitPerMinute {
		return fmt.Errorf("rate limit exceeded")
	}

	// Add current time
	c.rateLimiter[npub] = append(c.rateLimiter[npub], now)
	return nil
}

func (c *Controller) BlockNpub(npub string) error {
	c.blockMutex.Lock()
	defer c.blockMutex.Unlock()

	c.blockedNpubs[npub] = true
	log.Printf("Blocked npub: %s", npub)
	return nil
}

func (c *Controller) UnblockNpub(npub string) error {
	c.blockMutex.Lock()
	defer c.blockMutex.Unlock()

	delete(c.blockedNpubs, npub)
	log.Printf("Unblocked npub: %s", npub)
	return nil
}

func (c *Controller) IsNpubBlocked(npub string) bool {
	c.blockMutex.RLock()
	defer c.blockMutex.RUnlock()

	return c.blockedNpubs[npub]
}

func (c *Controller) GetBlockedNpubs() []string {
	c.blockMutex.RLock()
	defer c.blockMutex.RUnlock()

	var npubs []string
	for npub := range c.blockedNpubs {
		npubs = append(npubs, npub)
	}
	return npubs
}

func (c *Controller) cleanupRateLimiter(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.rateMutex.Lock()
			now := time.Now()
			cutoff := now.Add(-time.Minute)

			for npub, times := range c.rateLimiter {
				var validTimes []time.Time
				for _, t := range times {
					if t.After(cutoff) {
						validTimes = append(validTimes, t)
					}
				}
				if len(validTimes) == 0 {
					delete(c.rateLimiter, npub)
				} else {
					c.rateLimiter[npub] = validTimes
				}
			}
			c.rateMutex.Unlock()
		}
	}
}

func (c *Controller) monitorQuality(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Get quality statistics
			stats, err := c.GetQualityStats()
			if err != nil {
				log.Printf("Failed to get quality stats: %v", err)
				continue
			}

			log.Printf("Quality stats: %+v", stats)
		}
	}
}

func (c *Controller) SetKindConfigLoader(loader *KindConfigLoader) {
	c.kindConfigLoader = loader
}

func (c *Controller) GetQualityStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get rate limiter stats
	c.rateMutex.RLock()
	activeNpubs := len(c.rateLimiter)
	c.rateMutex.RUnlock()
	stats["active_npubs"] = activeNpubs

	// Get blocked npubs count
	c.blockMutex.RLock()
	blockedCount := len(c.blockedNpubs)
	c.blockMutex.RUnlock()
	stats["blocked_npubs"] = blockedCount

	return stats, nil
}
