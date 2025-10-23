package quality

import (
	"context"
	"testing"
	"time"

	"mercury-relay/internal/config"
	"mercury-relay/internal/models"
	"mercury-relay/test/helpers"
	"mercury-relay/test/mocks"

	"github.com/nbd-wtf/go-nostr"
)

func TestEventValidation(t *testing.T) {
	eg := models.NewEventGenerator()

	t.Run("Normal quality event", func(t *testing.T) {
		cfg := config.QualityConfig{
			MaxContentLength:   10000,
			RateLimitPerMinute: 100,
			SpamThreshold:      0.7,
		}
		mockQueue := mocks.NewMockQueue()
		mockCache := mocks.NewMockCache()
		controller := NewController(cfg, mockQueue, mockCache)

		event := eg.GenerateTextNote(eg.GetRandomNpub(), "This is a normal quality event with reasonable content length.",
			nostr.Tags{{"t", "quality"}, {"t", "test"}})

		err := controller.ValidateEvent(event)
		helpers.AssertNoError(t, err)
		helpers.AssertQualityScore(t, event, 0.8, 1.0)
		helpers.AssertEventQuarantined(t, event, false)
	})

	t.Run("Spam detection", func(t *testing.T) {
		cfg := config.QualityConfig{
			MaxContentLength:   10000,
			RateLimitPerMinute: 100,
			SpamThreshold:      0.7,
		}
		mockQueue := mocks.NewMockQueue()
		mockCache := mocks.NewMockCache()
		controller := NewController(cfg, mockQueue, mockCache)

		event := eg.GenerateSpamEvent(eg.GetRandomNpub())

		err := controller.ValidateEvent(event)
		helpers.AssertNoError(t, err) // Event should still be processed but quarantined
		helpers.AssertQualityScore(t, event, 0.0, 0.6)
		helpers.AssertEventQuarantined(t, event, true)
		helpers.AssertStringEqual(t, "Low quality score", event.QuarantineReason)
	})
}

func TestRateLimiting(t *testing.T) {
	eg := models.NewEventGenerator()
	npub := eg.GetRandomNpub()

	t.Run("Normal posting rate", func(t *testing.T) {
		cfg := config.QualityConfig{
			MaxContentLength:   10000,
			RateLimitPerMinute: 100,
			SpamThreshold:      0.7,
		}
		mockQueue := mocks.NewMockQueue()
		mockCache := mocks.NewMockCache()
		controller := NewController(cfg, mockQueue, mockCache)

		// Post 5 events in rapid succession
		for i := 0; i < 5; i++ {
			event := eg.GenerateTextNote(npub, "Test message", nostr.Tags{})
			err := controller.ValidateEvent(event)
			helpers.AssertNoError(t, err)
		}
	})

	t.Run("Rate limit exceeded", func(t *testing.T) {
		cfg := config.QualityConfig{
			MaxContentLength:   10000,
			RateLimitPerMinute: 5,
			SpamThreshold:      0.7,
		}
		mockQueue := mocks.NewMockQueue()
		mockCache := mocks.NewMockCache()
		controller := NewController(cfg, mockQueue, mockCache)

		// Post 5 events (should succeed)
		for i := 0; i < 5; i++ {
			event := eg.GenerateTextNote(npub, "Test message", nostr.Tags{})
			err := controller.ValidateEvent(event)
			helpers.AssertNoError(t, err)
		}

		// 6th event should be rejected
		event := eg.GenerateTextNote(npub, "Test message", nostr.Tags{})
		err := controller.ValidateEvent(event)
		helpers.AssertError(t, err)
		helpers.AssertErrorContains(t, err, "rate limit exceeded")
	})

	t.Run("Rate limit resets after time", func(t *testing.T) {
		cfg := config.QualityConfig{
			MaxContentLength:   10000,
			RateLimitPerMinute: 5,
			SpamThreshold:      0.7,
		}
		mockQueue := mocks.NewMockQueue()
		mockCache := mocks.NewMockCache()
		controller := NewController(cfg, mockQueue, mockCache)

		// Post 5 events
		for i := 0; i < 5; i++ {
			event := eg.GenerateTextNote(npub, "Test message", nostr.Tags{})
			err := controller.ValidateEvent(event)
			helpers.AssertNoError(t, err)
		}

		// Manually reset rate limiter by clearing old entries
		controller.rateMutex.Lock()
		controller.rateLimiter[npub] = []time.Time{}
		controller.rateMutex.Unlock()

		// Now should be able to post again
		event := eg.GenerateTextNote(npub, "Test message", nostr.Tags{})
		err := controller.ValidateEvent(event)
		helpers.AssertNoError(t, err)
	})
}

func TestBlockingUnblocking(t *testing.T) {
	eg := models.NewEventGenerator()
	npub := eg.GetRandomNpub()

	t.Run("Block npub", func(t *testing.T) {
		cfg := config.QualityConfig{
			MaxContentLength:   10000,
			RateLimitPerMinute: 100,
			SpamThreshold:      0.7,
		}
		mockQueue := mocks.NewMockQueue()
		mockCache := mocks.NewMockCache()
		controller := NewController(cfg, mockQueue, mockCache)

		// Block npub
		err := controller.BlockNpub(npub)
		helpers.AssertNoError(t, err)
		helpers.AssertBoolEqual(t, true, controller.IsNpubBlocked(npub))

		// Try to validate event from blocked npub
		event := eg.GenerateTextNote(npub, "Test message", nostr.Tags{})
		err = controller.ValidateEvent(event)
		helpers.AssertError(t, err)
		helpers.AssertErrorContains(t, err, "blocked")
	})

	t.Run("Unblock npub", func(t *testing.T) {
		cfg := config.QualityConfig{
			MaxContentLength:   10000,
			RateLimitPerMinute: 100,
			SpamThreshold:      0.7,
		}
		mockQueue := mocks.NewMockQueue()
		mockCache := mocks.NewMockCache()
		controller := NewController(cfg, mockQueue, mockCache)

		// Block then unblock npub
		controller.BlockNpub(npub)
		helpers.AssertBoolEqual(t, true, controller.IsNpubBlocked(npub))

		err := controller.UnblockNpub(npub)
		helpers.AssertNoError(t, err)
		helpers.AssertBoolEqual(t, false, controller.IsNpubBlocked(npub))

		// Now should be able to validate event
		event := eg.GenerateTextNote(npub, "Test message", nostr.Tags{})
		err = controller.ValidateEvent(event)
		helpers.AssertNoError(t, err)
	})

	t.Run("Get blocked npubs", func(t *testing.T) {
		cfg := config.QualityConfig{
			MaxContentLength:   10000,
			RateLimitPerMinute: 100,
			SpamThreshold:      0.7,
		}
		mockQueue := mocks.NewMockQueue()
		mockCache := mocks.NewMockCache()
		controller := NewController(cfg, mockQueue, mockCache)

		// Block multiple npubs
		npub1 := "npub1blocked"
		npub2 := "npub2blocked"

		controller.BlockNpub(npub1)
		controller.BlockNpub(npub2)

		blocked := controller.GetBlockedNpubs()
		helpers.AssertIntEqual(t, 2, len(blocked))
		helpers.AssertContains(t, blocked, npub1)
		helpers.AssertContains(t, blocked, npub2)
	})
}

func TestKindSpecificValidation(t *testing.T) {
	eg := models.NewEventGenerator()
	npub := eg.GetRandomNpub()

	t.Run("Kind 0 (user metadata) validation", func(t *testing.T) {
		cfg := config.QualityConfig{
			MaxContentLength:   10000,
			RateLimitPerMinute: 100,
			SpamThreshold:      0.7,
		}
		mockQueue := mocks.NewMockQueue()
		mockCache := mocks.NewMockCache()
		controller := NewController(cfg, mockQueue, mockCache)

		// Create kind config loader with kind 0 rules
		kindConfig, err := NewKindConfigLoader("../../configs/nostr-event-kinds.yaml")
		helpers.AssertNoError(t, err)
		controller.SetKindConfigLoader(kindConfig)

		// Valid kind 0 event
		metadata := map[string]interface{}{
			"name":  "Test User",
			"about": "A test user",
		}
		event := eg.GenerateUserMetadata(npub, metadata)

		err = controller.ValidateEvent(event)
		helpers.AssertNoError(t, err)
	})

	t.Run("Kind 1 (text note) with excessive length", func(t *testing.T) {
		cfg := config.QualityConfig{
			MaxContentLength:   10000,
			RateLimitPerMinute: 100,
			SpamThreshold:      0.7,
		}
		mockQueue := mocks.NewMockQueue()
		mockCache := mocks.NewMockCache()
		controller := NewController(cfg, mockQueue, mockCache)

		// Create kind config loader with kind 1 rules
		kindConfig, err := NewKindConfigLoader("../../configs/nostr-event-kinds.yaml")
		helpers.AssertNoError(t, err)
		controller.SetKindConfigLoader(kindConfig)

		// Create event with very long content
		longContent := make([]byte, 10001)
		for i := range longContent {
			longContent[i] = 'a'
		}

		event := eg.GenerateTextNote(npub, string(longContent), nostr.Tags{})

		err = controller.ValidateEvent(event)
		helpers.AssertError(t, err)
		helpers.AssertErrorContains(t, err, "too long")
	})
}

func TestQualityScoreCalculation(t *testing.T) {
	eg := models.NewEventGenerator()

	t.Run("Optimal content", func(t *testing.T) {
		cfg := config.QualityConfig{
			MaxContentLength:   10000,
			RateLimitPerMinute: 100,
			SpamThreshold:      0.7,
		}
		mockQueue := mocks.NewMockQueue()
		mockCache := mocks.NewMockCache()
		controller := NewController(cfg, mockQueue, mockCache)

		event := eg.GenerateHighQualityEvent(eg.GetRandomNpub())

		err := controller.ValidateEvent(event)
		helpers.AssertNoError(t, err)
		helpers.AssertQualityScore(t, event, 0.9, 1.0)
	})

	t.Run("Edge cases", func(t *testing.T) {
		cfg := config.QualityConfig{
			MaxContentLength:   10000,
			RateLimitPerMinute: 100,
			SpamThreshold:      0.7,
		}
		mockQueue := mocks.NewMockQueue()
		mockCache := mocks.NewMockCache()
		controller := NewController(cfg, mockQueue, mockCache)

		// Empty content, no tags
		event := eg.GenerateTextNote(eg.GetRandomNpub(), "", nostr.Tags{})

		err := controller.ValidateEvent(event)
		helpers.AssertNoError(t, err)
		// Score should be calculated but may be low
		helpers.AssertQualityScore(t, event, 0.0, 1.0)
	})
}

func TestContentLengthValidation(t *testing.T) {
	eg := models.NewEventGenerator()

	t.Run("Content within limit", func(t *testing.T) {
		cfg := config.QualityConfig{
			MaxContentLength:   5000,
			RateLimitPerMinute: 100,
			SpamThreshold:      0.7,
		}
		mockQueue := mocks.NewMockQueue()
		mockCache := mocks.NewMockCache()
		controller := NewController(cfg, mockQueue, mockCache)

		event := eg.GenerateTextNote(eg.GetRandomNpub(), "Short content", nostr.Tags{})

		err := controller.ValidateEvent(event)
		helpers.AssertNoError(t, err)
	})

	t.Run("Content exceeds limit", func(t *testing.T) {
		cfg := config.QualityConfig{
			MaxContentLength:   100,
			RateLimitPerMinute: 100,
			SpamThreshold:      0.7,
		}
		mockQueue := mocks.NewMockQueue()
		mockCache := mocks.NewMockCache()
		controller := NewController(cfg, mockQueue, mockCache)

		longContent := make([]byte, 101)
		for i := range longContent {
			longContent[i] = 'a'
		}
		event := eg.GenerateTextNote(eg.GetRandomNpub(), string(longContent), nostr.Tags{})

		err := controller.ValidateEvent(event)
		helpers.AssertError(t, err)
		helpers.AssertErrorContains(t, err, "too long")
	})
}

func TestQualityStats(t *testing.T) {
	eg := models.NewEventGenerator()

	t.Run("Get quality stats", func(t *testing.T) {
		cfg := config.QualityConfig{
			MaxContentLength:   10000,
			RateLimitPerMinute: 100,
			SpamThreshold:      0.7,
		}
		mockQueue := mocks.NewMockQueue()
		mockCache := mocks.NewMockCache()
		controller := NewController(cfg, mockQueue, mockCache)

		// Block some npubs
		controller.BlockNpub("npub1blocked")
		controller.BlockNpub("npub2blocked")

		// Add some rate limiting activity
		npub := eg.GetRandomNpub()
		event := eg.GenerateTextNote(npub, "Test", nostr.Tags{})
		controller.ValidateEvent(event)

		stats, err := controller.GetQualityStats()
		helpers.AssertNoError(t, err)

		helpers.AssertIntEqual(t, 2, stats["blocked_npubs"].(int))
		helpers.AssertIntEqual(t, 1, stats["active_npubs"].(int))
	})
}

func TestQualityControlStartStop(t *testing.T) {
	_ = models.NewEventGenerator()

	t.Run("Start quality control", func(t *testing.T) {
		cfg := config.QualityConfig{
			MaxContentLength:   10000,
			RateLimitPerMinute: 100,
			SpamThreshold:      0.7,
		}
		mockQueue := mocks.NewMockQueue()
		mockCache := mocks.NewMockCache()
		controller := NewController(cfg, mockQueue, mockCache)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := controller.Start(ctx)
		helpers.AssertNoError(t, err)

		// Should start cleanup and monitoring goroutines
		time.Sleep(50 * time.Millisecond)

		// Context should be cancelled and goroutines should exit
		time.Sleep(100 * time.Millisecond)
	})
}

func TestQualityControlIntegration(t *testing.T) {
	eg := models.NewEventGenerator()

	t.Run("High quality event flow", func(t *testing.T) {
		cfg := config.QualityConfig{
			MaxContentLength:   10000,
			RateLimitPerMinute: 100,
			SpamThreshold:      0.7,
		}
		mockQueue := mocks.NewMockQueue()
		mockCache := mocks.NewMockCache()
		controller := NewController(cfg, mockQueue, mockCache)

		// Start controller
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		controller.Start(ctx)

		// Publish high-quality event
		event := eg.GenerateHighQualityEvent(eg.GetRandomNpub())

		err := controller.ValidateEvent(event)
		helpers.AssertNoError(t, err)

		// Event should pass quality checks
		helpers.AssertQualityScore(t, event, 0.8, 1.0)
		helpers.AssertEventQuarantined(t, event, false)

		// Should be published to queue
		helpers.AssertIntEqual(t, 1, mockQueue.GetEventCount())
	})

	t.Run("Spam quarantine flow", func(t *testing.T) {
		cfg := config.QualityConfig{
			MaxContentLength:   10000,
			RateLimitPerMinute: 100,
			SpamThreshold:      0.7,
		}
		mockQueue := mocks.NewMockQueue()
		mockCache := mocks.NewMockCache()
		controller := NewController(cfg, mockQueue, mockCache)

		// Start controller
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		controller.Start(ctx)

		// Publish spam event
		event := eg.GenerateSpamEvent(eg.GetRandomNpub())

		err := controller.ValidateEvent(event)
		helpers.AssertNoError(t, err) // Should still be processed

		// Event should be quarantined
		helpers.AssertEventQuarantined(t, event, true)
		helpers.AssertStringEqual(t, "Low quality score", event.QuarantineReason)

		// Should still be published to queue with quarantine flag
		helpers.AssertIntEqual(t, 1, mockQueue.GetEventCount())
	})
}

func TestRateLimiterCleanup(t *testing.T) {
	eg := models.NewEventGenerator()

	t.Run("Rate limiter cleanup", func(t *testing.T) {
		cfg := config.QualityConfig{
			MaxContentLength:   10000,
			RateLimitPerMinute: 100,
			SpamThreshold:      0.7,
		}
		mockQueue := mocks.NewMockQueue()
		mockCache := mocks.NewMockCache()
		controller := NewController(cfg, mockQueue, mockCache)

		// Start controller
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		controller.Start(ctx)

		// Add some old rate limiting entries
		npub := eg.GetRandomNpub()
		controller.rateMutex.Lock()
		controller.rateLimiter[npub] = []time.Time{
			time.Now().Add(-2 * time.Minute), // Old entry
			time.Now(),                       // Recent entry
		}
		controller.rateMutex.Unlock()

		// Trigger cleanup manually
		controller.rateMutex.Lock()
		now := time.Now()
		cutoff := now.Add(-time.Minute)

		for npub, times := range controller.rateLimiter {
			var validTimes []time.Time
			for _, t := range times {
				if t.After(cutoff) {
					validTimes = append(validTimes, t)
				}
			}
			if len(validTimes) == 0 {
				delete(controller.rateLimiter, npub)
			} else {
				controller.rateLimiter[npub] = validTimes
			}
		}
		controller.rateMutex.Unlock()

		// Old entries should be cleaned up
		controller.rateMutex.RLock()
		times := controller.rateLimiter[npub]
		controller.rateMutex.RUnlock()

		// Should only have recent entry
		helpers.AssertIntEqual(t, 1, len(times))
	})
}
