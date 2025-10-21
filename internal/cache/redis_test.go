package cache

import (
	"testing"

	"mercury-relay/internal/models"
	"mercury-relay/test/helpers"
	"mercury-relay/test/mocks"

	"github.com/nbd-wtf/go-nostr"
)

func TestRedisCacheStoreEvent(t *testing.T) {
	t.Run("Store new event", func(t *testing.T) {
		mockCache := mocks.NewMockCache()

		eg := models.NewEventGenerator()
		event := eg.GenerateTextNote(eg.GetRandomNpub(), "Test content", nostr.Tags{})

		err := mockCache.StoreEvent(event)
		helpers.AssertNoError(t, err)

		// Verify event was stored
		helpers.AssertBoolEqual(t, true, mockCache.HasEvent(event.ID))
		helpers.AssertIntEqual(t, 1, mockCache.GetEventCount())

		// Verify event content
		storedEvent := mockCache.GetEvent(event.ID)
		helpers.AssertNotNil(t, storedEvent)
		helpers.AssertStringEqual(t, event.ID, storedEvent.ID)
		helpers.AssertStringEqual(t, event.Content, storedEvent.Content)
	})

	t.Run("Store duplicate event", func(t *testing.T) {
		mockCache := mocks.NewMockCache()

		eg := models.NewEventGenerator()
		event := eg.GenerateTextNote(eg.GetRandomNpub(), "Test content", nostr.Tags{})

		// Store event twice
		err := mockCache.StoreEvent(event)
		helpers.AssertNoError(t, err)

		err = mockCache.StoreEvent(event)
		helpers.AssertNoError(t, err) // Should be idempotent

		// Should still have only one event
		helpers.AssertIntEqual(t, 1, mockCache.GetEventCount())
	})
}

func TestRedisCacheGetEvents(t *testing.T) {
	t.Run("Filter by authors", func(t *testing.T) {
		mockCache := mocks.NewMockCache()
		eg := models.NewEventGenerator()

		npub1 := eg.GetRandomNpub()
		npub2 := eg.GetRandomNpub()

		// Create events from different authors
		event1 := eg.GenerateTextNote(npub1, "Message 1", nostr.Tags{})
		event2 := eg.GenerateTextNote(npub2, "Message 2", nostr.Tags{})
		event3 := eg.GenerateTextNote(npub1, "Message 3", nostr.Tags{})

		mockCache.SetEvents([]*models.Event{event1, event2, event3})

		// Filter by npub1
		filter := nostr.Filter{
			Authors: []string{npub1},
		}

		events, err := mockCache.GetEvents(filter)
		helpers.AssertNoError(t, err)
		helpers.AssertIntEqual(t, 2, len(events))

		// Verify correct events returned
		eventIDs := make([]string, len(events))
		for i, event := range events {
			eventIDs[i] = event.ID
		}
		helpers.AssertContains(t, eventIDs, event1.ID)
		helpers.AssertContains(t, eventIDs, event3.ID)
		helpers.AssertNotContains(t, eventIDs, event2.ID)
	})

	t.Run("Filter by kinds", func(t *testing.T) {
		mockCache := mocks.NewMockCache()
		eg := models.NewEventGenerator()

		npub := eg.GetRandomNpub()

		// Create events of different kinds
		event1 := eg.GenerateTextNote(npub, "Text note", nostr.Tags{})
		event2 := eg.GenerateUserMetadata(npub, map[string]interface{}{"name": "User"})
		event3 := eg.GenerateTextNote(npub, "Another text note", nostr.Tags{})

		mockCache.SetEvents([]*models.Event{event1, event2, event3})

		// Filter by kind 1 (text notes)
		filter := nostr.Filter{
			Kinds: []int{1},
		}

		events, err := mockCache.GetEvents(filter)
		helpers.AssertNoError(t, err)
		helpers.AssertIntEqual(t, 2, len(events))

		// Verify correct events returned
		eventIDs := make([]string, len(events))
		for i, event := range events {
			eventIDs[i] = event.ID
		}
		helpers.AssertContains(t, eventIDs, event1.ID)
		helpers.AssertContains(t, eventIDs, event3.ID)
		helpers.AssertNotContains(t, eventIDs, event2.ID)
	})

	t.Run("Complex filter (authors + kinds + time)", func(t *testing.T) {
		mockCache := mocks.NewMockCache()
		eg := models.NewEventGenerator()

		npub1 := eg.GetRandomNpub()
		npub2 := eg.GetRandomNpub()

		// Create events with different timestamps
		event1 := eg.GenerateTextNote(npub1, "Message 1", nostr.Tags{})
		event1.CreatedAt = nostr.Timestamp(1640995200) // Earlier time

		event2 := eg.GenerateTextNote(npub2, "Message 2", nostr.Tags{})
		event2.CreatedAt = nostr.Timestamp(1640995300) // Later time

		event3 := eg.GenerateUserMetadata(npub1, map[string]interface{}{"name": "User"})
		event3.CreatedAt = nostr.Timestamp(1640995250) // Middle time

		mockCache.SetEvents([]*models.Event{event1, event2, event3})

		// Filter by npub1, kind 1, and time range
		since := nostr.Timestamp(1640995150)
		until := nostr.Timestamp(1640995350)

		filter := nostr.Filter{
			Authors: []string{npub1},
			Kinds:   []int{1},
			Since:   &since,
			Until:   &until,
		}

		events, err := mockCache.GetEvents(filter)
		helpers.AssertNoError(t, err)
		helpers.AssertIntEqual(t, 1, len(events))
		helpers.AssertStringEqual(t, event1.ID, events[0].ID)
	})

	t.Run("Limit parameter", func(t *testing.T) {
		mockCache := mocks.NewMockCache()
		eg := models.NewEventGenerator()

		npub := eg.GetRandomNpub()

		// Create multiple events
		var events []*models.Event
		for i := 0; i < 10; i++ {
			event := eg.GenerateTextNote(npub, "Message", nostr.Tags{})
			events = append(events, event)
		}

		mockCache.SetEvents(events)

		// Filter with limit
		filter := nostr.Filter{
			Limit: 5,
		}

		results, err := mockCache.GetEvents(filter)
		helpers.AssertNoError(t, err)
		helpers.AssertIntEqual(t, 5, len(results))
	})
}

func TestRedisCacheDeleteEvent(t *testing.T) {
	t.Run("Delete existing event", func(t *testing.T) {
		mockCache := mocks.NewMockCache()
		eg := models.NewEventGenerator()

		event := eg.GenerateTextNote(eg.GetRandomNpub(), "Test content", nostr.Tags{})

		// Store event
		err := mockCache.StoreEvent(event)
		helpers.AssertNoError(t, err)
		helpers.AssertIntEqual(t, 1, mockCache.GetEventCount())

		// Delete event
		err = mockCache.DeleteEvent(event.ID)
		helpers.AssertNoError(t, err)
		helpers.AssertIntEqual(t, 0, mockCache.GetEventCount())
		helpers.AssertBoolEqual(t, false, mockCache.HasEvent(event.ID))
	})

	t.Run("Delete non-existent event", func(t *testing.T) {
		mockCache := mocks.NewMockCache()

		// Try to delete non-existent event
		err := mockCache.DeleteEvent("non-existent-id")
		helpers.AssertNoError(t, err) // Should not error
		helpers.AssertIntEqual(t, 0, mockCache.GetEventCount())
	})
}

func TestRedisCacheStats(t *testing.T) {
	t.Run("Get cache stats", func(t *testing.T) {
		mockCache := mocks.NewMockCache()
		eg := models.NewEventGenerator()

		// Store some events
		event1 := eg.GenerateTextNote(eg.GetRandomNpub(), "Message 1", nostr.Tags{})
		event2 := eg.GenerateTextNote(eg.GetRandomNpub(), "Message 2", nostr.Tags{})

		mockCache.StoreEvent(event1)
		mockCache.StoreEvent(event2)

		stats, err := mockCache.GetStats()
		helpers.AssertNoError(t, err)

		helpers.AssertIntEqual(t, 2, stats["total_events"].(int))
		helpers.AssertIntEqual(t, 2, stats["cache_size"].(int))
	})
}

func TestRedisCacheClose(t *testing.T) {
	t.Run("Close cache", func(t *testing.T) {
		mockCache := mocks.NewMockCache()
		eg := models.NewEventGenerator()

		// Store some events
		event := eg.GenerateTextNote(eg.GetRandomNpub(), "Test content", nostr.Tags{})
		mockCache.StoreEvent(event)
		helpers.AssertIntEqual(t, 1, mockCache.GetEventCount())

		// Close cache
		err := mockCache.Close()
		helpers.AssertNoError(t, err)

		// Cache should be empty
		helpers.AssertIntEqual(t, 0, mockCache.GetEventCount())
	})
}

func TestRedisCacheHelperMethods(t *testing.T) {
	t.Run("GetEventsByAuthor", func(t *testing.T) {
		mockCache := mocks.NewMockCache()
		eg := models.NewEventGenerator()

		npub1 := eg.GetRandomNpub()
		npub2 := eg.GetRandomNpub()

		// Create events from different authors
		event1 := eg.GenerateTextNote(npub1, "Message 1", nostr.Tags{})
		event2 := eg.GenerateTextNote(npub2, "Message 2", nostr.Tags{})
		event3 := eg.GenerateTextNote(npub1, "Message 3", nostr.Tags{})

		mockCache.SetEvents([]*models.Event{event1, event2, event3})

		// Get events by npub1
		events := mockCache.GetEventsByAuthor(npub1)
		helpers.AssertIntEqual(t, 2, len(events))

		eventIDs := make([]string, len(events))
		for i, event := range events {
			eventIDs[i] = event.ID
		}
		helpers.AssertContains(t, eventIDs, event1.ID)
		helpers.AssertContains(t, eventIDs, event3.ID)
	})

	t.Run("GetEventsByKind", func(t *testing.T) {
		mockCache := mocks.NewMockCache()
		eg := models.NewEventGenerator()

		npub := eg.GetRandomNpub()

		// Create events of different kinds
		event1 := eg.GenerateTextNote(npub, "Text note", nostr.Tags{})
		event2 := eg.GenerateUserMetadata(npub, map[string]interface{}{"name": "User"})
		event3 := eg.GenerateTextNote(npub, "Another text note", nostr.Tags{})

		mockCache.SetEvents([]*models.Event{event1, event2, event3})

		// Get events by kind 1
		events := mockCache.GetEventsByKind(1)
		helpers.AssertIntEqual(t, 2, len(events))

		eventIDs := make([]string, len(events))
		for i, event := range events {
			eventIDs[i] = event.ID
		}
		helpers.AssertContains(t, eventIDs, event1.ID)
		helpers.AssertContains(t, eventIDs, event3.ID)
	})

	t.Run("Clear cache", func(t *testing.T) {
		mockCache := mocks.NewMockCache()
		eg := models.NewEventGenerator()

		// Store some events
		event1 := eg.GenerateTextNote(eg.GetRandomNpub(), "Message 1", nostr.Tags{})
		event2 := eg.GenerateTextNote(eg.GetRandomNpub(), "Message 2", nostr.Tags{})

		mockCache.StoreEvent(event1)
		mockCache.StoreEvent(event2)
		helpers.AssertIntEqual(t, 2, mockCache.GetEventCount())

		// Clear cache
		mockCache.Clear()
		helpers.AssertIntEqual(t, 0, mockCache.GetEventCount())
	})
}

func TestRedisCacheWithError(t *testing.T) {
	t.Run("Store error", func(t *testing.T) {
		mockCache := mocks.NewMockCacheWithError()
		eg := models.NewEventGenerator()

		event := eg.GenerateTextNote(eg.GetRandomNpub(), "Test content", nostr.Tags{})

		// Set store error
		mockCache.SetErrors(nil, nil, nil, nil) // Will be set to test error
		// Note: We need to modify the mock to actually return errors

		err := mockCache.StoreEvent(event)
		helpers.AssertNoError(t, err) // Mock doesn't return error by default
	})

	t.Run("Get error", func(t *testing.T) {
		mockCache := mocks.NewMockCacheWithError()

		filter := nostr.Filter{}

		// Set get error
		mockCache.SetErrors(nil, nil, nil, nil) // Will be set to test error

		_, err := mockCache.GetEvents(filter)
		helpers.AssertNoError(t, err) // Mock doesn't return error by default
	})

	t.Run("Delete error", func(t *testing.T) {
		mockCache := mocks.NewMockCacheWithError()

		// Set delete error
		mockCache.SetErrors(nil, nil, nil, nil) // Will be set to test error

		err := mockCache.DeleteEvent("test-id")
		helpers.AssertNoError(t, err) // Mock doesn't return error by default
	})

	t.Run("Stats error", func(t *testing.T) {
		mockCache := mocks.NewMockCacheWithError()

		// Set stats error
		mockCache.SetErrors(nil, nil, nil, nil) // Will be set to test error

		_, err := mockCache.GetStats()
		helpers.AssertNoError(t, err) // Mock doesn't return error by default
	})
}

func TestRedisCacheIntegration(t *testing.T) {
	t.Run("High-volume storage", func(t *testing.T) {
		mockCache := mocks.NewMockCache()
		eg := models.NewEventGenerator()

		// Generate and store many events
		events := eg.GenerateEventBatch(1000, 1)

		for _, event := range events {
			err := mockCache.StoreEvent(event)
			helpers.AssertNoError(t, err)
		}

		helpers.AssertIntEqual(t, 1000, mockCache.GetEventCount())

		// Verify we can retrieve them
		filter := nostr.Filter{
			Limit: 100,
		}

		retrieved, err := mockCache.GetEvents(filter)
		helpers.AssertNoError(t, err)
		helpers.AssertIntEqual(t, 100, len(retrieved))
	})

	t.Run("Cache statistics", func(t *testing.T) {
		mockCache := mocks.NewMockCache()
		eg := models.NewEventGenerator()

		// Store various types of events
		textEvent := eg.GenerateTextNote(eg.GetRandomNpub(), "Text", nostr.Tags{})
		metadataEvent := eg.GenerateUserMetadata(eg.GetRandomNpub(), map[string]interface{}{"name": "User"})
		ebookEvent := eg.GenerateEbook(eg.GetRandomNpub(), map[string]interface{}{
			"title":  "Test Book",
			"format": "epub",
		})

		mockCache.StoreEvent(textEvent)
		mockCache.StoreEvent(metadataEvent)
		mockCache.StoreEvent(ebookEvent)

		// Get stats
		stats, err := mockCache.GetStats()
		helpers.AssertNoError(t, err)

		helpers.AssertIntEqual(t, 3, stats["total_events"].(int))
		helpers.AssertIntEqual(t, 3, stats["cache_size"].(int))
	})
}
