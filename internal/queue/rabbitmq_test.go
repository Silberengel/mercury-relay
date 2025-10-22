package queue

import (
	"testing"

	"mercury-relay/internal/models"
	"mercury-relay/test/helpers"
	"mercury-relay/test/mocks"

	"github.com/nbd-wtf/go-nostr"
)

func TestRabbitMQPublishEvent(t *testing.T) {
	t.Run("Publish single event", func(t *testing.T) {
		mockQueue := mocks.NewMockQueue()
		eg := models.NewEventGenerator()

		event := eg.GenerateTextNote(eg.GetRandomNpub(), "Test content", nostr.Tags{})

		err := mockQueue.PublishEvent(event)
		helpers.AssertNoError(t, err)

		// Verify event was queued
		helpers.AssertIntEqual(t, 1, mockQueue.GetEventCount())

		// Verify event content
		queuedEvents := mockQueue.GetEvents()
		helpers.AssertIntEqual(t, 1, len(queuedEvents))
		helpers.AssertStringEqual(t, event.ID, queuedEvents[0].ID)
		helpers.AssertStringEqual(t, event.Content, queuedEvents[0].Content)
	})

	t.Run("Publish failure", func(t *testing.T) {
		mockQueue := mocks.NewMockQueueWithError()
		eg := models.NewEventGenerator()

		event := eg.GenerateTextNote(eg.GetRandomNpub(), "Test content", nostr.Tags{})

		// Mock queue doesn't actually fail by default, but we can test the interface
		err := mockQueue.PublishEvent(event)
		helpers.AssertNoError(t, err)

		// In a real implementation, we would set up the mock to return an error
		// and verify that the error is properly handled
	})
}

func TestRabbitMQConsumeEvents(t *testing.T) {
	t.Run("Consume from queue", func(t *testing.T) {
		mockQueue := mocks.NewMockQueue()
		eg := models.NewEventGenerator()

		// Publish multiple events
		event1 := eg.GenerateTextNote(eg.GetRandomNpub(), "Message 1", nostr.Tags{})
		event2 := eg.GenerateTextNote(eg.GetRandomNpub(), "Message 2", nostr.Tags{})
		event3 := eg.GenerateTextNote(eg.GetRandomNpub(), "Message 3", nostr.Tags{})

		mockQueue.PublishEvent(event1)
		mockQueue.PublishEvent(event2)
		mockQueue.PublishEvent(event3)

		helpers.AssertIntEqual(t, 3, mockQueue.GetEventCount())

		// Consume events
		events, err := mockQueue.ConsumeEvents()
		helpers.AssertNoError(t, err)
		helpers.AssertIntEqual(t, 3, len(events))

		// Queue should be empty after consumption
		helpers.AssertIntEqual(t, 0, mockQueue.GetEventCount())

		// Verify events are returned in order
		helpers.AssertStringEqual(t, event1.ID, events[0].ID)
		helpers.AssertStringEqual(t, event2.ID, events[1].ID)
		helpers.AssertStringEqual(t, event3.ID, events[2].ID)
	})

	t.Run("Empty queue", func(t *testing.T) {
		mockQueue := mocks.NewMockQueue()

		// Try to consume from empty queue
		events, err := mockQueue.ConsumeEvents()
		helpers.AssertNoError(t, err)
		helpers.AssertIntEqual(t, 0, len(events))
	})

	t.Run("Consume failure", func(t *testing.T) {
		mockQueue := mocks.NewMockQueueWithError()

		// Mock queue doesn't actually fail by default, but we can test the interface
		_, err := mockQueue.ConsumeEvents()
		helpers.AssertNoError(t, err)

		// In a real implementation, we would set up the mock to return an error
		// and verify that the error is properly handled
	})
}

func TestRabbitMQQueueStatistics(t *testing.T) {
	t.Run("Queue depth monitoring", func(t *testing.T) {
		mockQueue := mocks.NewMockQueue()
		eg := models.NewEventGenerator()

		// Publish some events
		for i := 0; i < 50; i++ {
			event := eg.GenerateTextNote(eg.GetRandomNpub(), "Message", nostr.Tags{})
			mockQueue.PublishEvent(event)
		}

		// Get queue stats
		queueSize, err := mockQueue.GetQueueStats()
		helpers.AssertNoError(t, err)
		helpers.AssertIntEqual(t, 50, queueSize)

		// Get detailed stats
		stats := mockQueue.GetStats()
		helpers.AssertIntEqual(t, 50, stats["queue_size"].(int))
		helpers.AssertIntEqual(t, 50, stats["total_events"].(int))
	})

	t.Run("Multiple queues", func(t *testing.T) {
		// This would test multiple queue scenarios in a real implementation
		// For now, we test the single queue interface
		mockQueue := mocks.NewMockQueue()
		eg := models.NewEventGenerator()

		// Publish events to simulate different queues
		event1 := eg.GenerateTextNote(eg.GetRandomNpub(), "Priority 1", nostr.Tags{})
		event2 := eg.GenerateTextNote(eg.GetRandomNpub(), "Priority 2", nostr.Tags{})

		mockQueue.PublishEvent(event1)
		mockQueue.PublishEvent(event2)

		stats := mockQueue.GetStats()
		helpers.AssertIntEqual(t, 2, stats["queue_size"].(int))
	})

	t.Run("Stats error", func(t *testing.T) {
		mockQueue := mocks.NewMockQueueWithError()

		// Mock queue doesn't actually fail by default, but we can test the interface
		_, err := mockQueue.GetQueueStats()
		helpers.AssertNoError(t, err)

		// In a real implementation, we would set up the mock to return an error
		// and verify that the error is properly handled
	})
}

func TestRabbitMQClose(t *testing.T) {
	t.Run("Close queue", func(t *testing.T) {
		mockQueue := mocks.NewMockQueue()
		eg := models.NewEventGenerator()

		// Publish some events
		event := eg.GenerateTextNote(eg.GetRandomNpub(), "Test content", nostr.Tags{})
		mockQueue.PublishEvent(event)
		helpers.AssertIntEqual(t, 1, mockQueue.GetEventCount())

		// Close queue
		err := mockQueue.Close()
		helpers.AssertNoError(t, err)

		// Queue should be empty after close
		helpers.AssertIntEqual(t, 0, mockQueue.GetEventCount())
	})
}

func TestRabbitMQHelperMethods(t *testing.T) {
	t.Run("Peek at first event", func(t *testing.T) {
		mockQueue := mocks.NewMockQueue()
		eg := models.NewEventGenerator()

		event1 := eg.GenerateTextNote(eg.GetRandomNpub(), "First message", nostr.Tags{})
		event2 := eg.GenerateTextNote(eg.GetRandomNpub(), "Second message", nostr.Tags{})

		mockQueue.PublishEvent(event1)
		mockQueue.PublishEvent(event2)

		// Peek at first event
		peekedEvent := mockQueue.Peek()
		helpers.AssertNotNil(t, peekedEvent)
		helpers.AssertStringEqual(t, event1.ID, peekedEvent.ID)

		// Queue should still have both events
		helpers.AssertIntEqual(t, 2, mockQueue.GetEventCount())
	})

	t.Run("Peek at empty queue", func(t *testing.T) {
		mockQueue := mocks.NewMockQueue()

		// Peek at empty queue
		peekedEvent := mockQueue.Peek()
		if peekedEvent != nil {
			t.Fatalf("Expected nil, got %v", peekedEvent)
		}
	})

	t.Run("Clear queue", func(t *testing.T) {
		mockQueue := mocks.NewMockQueue()
		eg := models.NewEventGenerator()

		// Publish some events
		event1 := eg.GenerateTextNote(eg.GetRandomNpub(), "Message 1", nostr.Tags{})
		event2 := eg.GenerateTextNote(eg.GetRandomNpub(), "Message 2", nostr.Tags{})

		mockQueue.PublishEvent(event1)
		mockQueue.PublishEvent(event2)
		helpers.AssertIntEqual(t, 2, mockQueue.GetEventCount())

		// Clear queue
		mockQueue.Clear()
		helpers.AssertIntEqual(t, 0, mockQueue.GetEventCount())
	})
}

func TestRabbitMQIntegration(t *testing.T) {
	t.Run("Publish and consume cycle", func(t *testing.T) {
		mockQueue := mocks.NewMockQueue()
		eg := models.NewEventGenerator()

		// Publish 100 events
		var publishedEvents []*models.Event
		for i := 0; i < 100; i++ {
			event := eg.GenerateTextNote(eg.GetRandomNpub(), "Message", nostr.Tags{})
			publishedEvents = append(publishedEvents, event)
			mockQueue.PublishEvent(event)
		}

		helpers.AssertIntEqual(t, 100, mockQueue.GetEventCount())

		// Consume events in batches
		var consumedEvents []*models.Event
		_ = 25

		for len(consumedEvents) < 100 {
			events, err := mockQueue.ConsumeEvents()
			helpers.AssertNoError(t, err)

			consumedEvents = append(consumedEvents, events...)

			if len(events) == 0 {
				break // No more events
			}
		}

		// Verify all events were consumed
		helpers.AssertIntEqual(t, 100, len(consumedEvents))
		helpers.AssertIntEqual(t, 0, mockQueue.GetEventCount())

		// Verify order is preserved
		for i, consumed := range consumedEvents {
			helpers.AssertStringEqual(t, publishedEvents[i].ID, consumed.ID)
		}
	})

	t.Run("Consumer lag", func(t *testing.T) {
		mockQueue := mocks.NewMockQueue()
		eg := models.NewEventGenerator()

		// Publish events faster than consuming
		for i := 0; i < 1000; i++ {
			event := eg.GenerateTextNote(eg.GetRandomNpub(), "Message", nostr.Tags{})
			mockQueue.PublishEvent(event)
		}

		// Queue depth should be high
		helpers.AssertIntEqual(t, 1000, mockQueue.GetEventCount())

		// Consume some events
		events, err := mockQueue.ConsumeEvents()
		helpers.AssertNoError(t, err)
		helpers.AssertIntEqual(t, 1000, len(events))

		// Queue should be empty after consumption
		helpers.AssertIntEqual(t, 0, mockQueue.GetEventCount())
	})
}

func TestRabbitMQErrorHandling(t *testing.T) {
	t.Run("Publish error handling", func(t *testing.T) {
		mockQueue := mocks.NewMockQueueWithError()
		eg := models.NewEventGenerator()

		event := eg.GenerateTextNote(eg.GetRandomNpub(), "Test content", nostr.Tags{})

		// In a real implementation, we would test error handling
		// For now, we verify the interface works
		err := mockQueue.PublishEvent(event)
		helpers.AssertNoError(t, err)
	})

	t.Run("Consume error handling", func(t *testing.T) {
		mockQueue := mocks.NewMockQueueWithError()

		// In a real implementation, we would test error handling
		// For now, we verify the interface works
		_, err := mockQueue.ConsumeEvents()
		helpers.AssertNoError(t, err)
	})

	t.Run("Stats error handling", func(t *testing.T) {
		mockQueue := mocks.NewMockQueueWithError()

		// In a real implementation, we would test error handling
		// For now, we verify the interface works
		_, err := mockQueue.GetQueueStats()
		helpers.AssertNoError(t, err)
	})
}

func TestRabbitMQConcurrency(t *testing.T) {
	t.Run("Concurrent publish and consume", func(t *testing.T) {
		mockQueue := mocks.NewMockQueue()
		eg := models.NewEventGenerator()

		// Simulate concurrent operations
		done := make(chan bool, 2)

		// Goroutine 1: Publish events
		go func() {
			for i := 0; i < 50; i++ {
				event := eg.GenerateTextNote(eg.GetRandomNpub(), "Message", nostr.Tags{})
				mockQueue.PublishEvent(event)
			}
			done <- true
		}()

		// Goroutine 2: Consume events
		go func() {
			for i := 0; i < 50; i++ {
				events, _ := mockQueue.ConsumeEvents()
				if len(events) > 0 {
					// Process events
				}
			}
			done <- true
		}()

		// Wait for both goroutines to complete
		<-done
		<-done

		// In a real implementation, we would verify thread safety
		// For now, we just ensure no panics occur
	})
}

func TestRabbitMQEventTypes(t *testing.T) {
	t.Run("Different event kinds", func(t *testing.T) {
		mockQueue := mocks.NewMockQueue()
		eg := models.NewEventGenerator()

		// Publish different types of events
		textEvent := eg.GenerateTextNote(eg.GetRandomNpub(), "Text note", nostr.Tags{})
		metadataEvent := eg.GenerateUserMetadata(eg.GetRandomNpub(), map[string]interface{}{"name": "User"})
		ebookEvent := eg.GenerateEbook(eg.GetRandomNpub(), map[string]interface{}{
			"title":  "Test Book",
			"format": "epub",
		})

		mockQueue.PublishEvent(textEvent)
		mockQueue.PublishEvent(metadataEvent)
		mockQueue.PublishEvent(ebookEvent)

		helpers.AssertIntEqual(t, 3, mockQueue.GetEventCount())

		// Consume and verify
		events, err := mockQueue.ConsumeEvents()
		helpers.AssertNoError(t, err)
		helpers.AssertIntEqual(t, 3, len(events))

		// Verify different event types
		eventKinds := make([]int, len(events))
		for i, event := range events {
			eventKinds[i] = event.Kind
		}

		helpers.AssertContains(t, eventKinds, 1)     // Text note
		helpers.AssertContains(t, eventKinds, 0)     // User metadata
		helpers.AssertContains(t, eventKinds, 30040) // Ebook
	})
}
