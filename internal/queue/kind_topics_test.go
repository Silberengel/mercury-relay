package queue

import (
	"testing"

	"mercury-relay/internal/models"
	"mercury-relay/test/mocks"
)

func TestMockQueueKindFiltering(t *testing.T) {
	mockQueue := mocks.NewMockQueue()

	// Create test events with different kinds
	events := []*models.Event{
		{ID: "1", Kind: 0, Content: "Profile metadata"},
		{ID: "2", Kind: 1, Content: "Short note"},
		{ID: "3", Kind: 1, Content: "Another note"},
		{ID: "4", Kind: 7, Content: "Reaction"},
		{ID: "5", Kind: 999, Content: "Unknown kind"},
		{ID: "6", Kind: 42, Content: "Another unknown kind"},
	}

	// Add events to queue
	for _, event := range events {
		err := mockQueue.PublishEvent(event)
		if err != nil {
			t.Fatalf("Failed to publish event: %v", err)
		}
	}

	// Test consuming kind 1 events
	kind1Events, err := mockQueue.ConsumeEventsByKind(1)
	if err != nil {
		t.Fatalf("Failed to consume kind 1 events: %v", err)
	}

	if len(kind1Events) != 2 {
		t.Errorf("Expected 2 kind 1 events, got %d", len(kind1Events))
	}

	// Test consuming undefined kind events
	undefinedEvents, err := mockQueue.ConsumeEventsByKind(999)
	if err != nil {
		t.Fatalf("Failed to consume undefined events: %v", err)
	}

	if len(undefinedEvents) != 2 {
		t.Errorf("Expected 2 undefined events, got %d", len(undefinedEvents))
	}

	// Test consuming kind 0 events
	kind0Events, err := mockQueue.ConsumeEventsByKind(0)
	if err != nil {
		t.Fatalf("Failed to consume kind 0 events: %v", err)
	}

	if len(kind0Events) != 1 {
		t.Errorf("Expected 1 kind 0 event, got %d", len(kind0Events))
	}

	// Test consuming kind 7 events
	kind7Events, err := mockQueue.ConsumeEventsByKind(7)
	if err != nil {
		t.Fatalf("Failed to consume kind 7 events: %v", err)
	}

	if len(kind7Events) != 1 {
		t.Errorf("Expected 1 kind 7 event, got %d", len(kind7Events))
	}
}

func TestMockQueueKindStats(t *testing.T) {
	mockQueue := mocks.NewMockQueue()

	// Create test events with different kinds
	events := []*models.Event{
		{ID: "1", Kind: 0, Content: "Profile metadata"},
		{ID: "2", Kind: 1, Content: "Short note"},
		{ID: "3", Kind: 1, Content: "Another note"},
		{ID: "4", Kind: 7, Content: "Reaction"},
		{ID: "5", Kind: 999, Content: "Unknown kind"},
		{ID: "6", Kind: 42, Content: "Another unknown kind"},
	}

	// Add events to queue
	for _, event := range events {
		err := mockQueue.PublishEvent(event)
		if err != nil {
			t.Fatalf("Failed to publish event: %v", err)
		}
	}

	// Test individual kind stats
	kind0Stats, err := mockQueue.GetKindQueueStats(0)
	if err != nil {
		t.Fatalf("Failed to get kind 0 stats: %v", err)
	}
	if kind0Stats != 1 {
		t.Errorf("Expected 1 kind 0 event, got %d", kind0Stats)
	}

	kind1Stats, err := mockQueue.GetKindQueueStats(1)
	if err != nil {
		t.Fatalf("Failed to get kind 1 stats: %v", err)
	}
	if kind1Stats != 2 {
		t.Errorf("Expected 2 kind 1 events, got %d", kind1Stats)
	}

	// Test undefined kind stats
	undefinedStats, err := mockQueue.GetKindQueueStats(999)
	if err != nil {
		t.Fatalf("Failed to get undefined stats: %v", err)
	}
	if undefinedStats != 2 {
		t.Errorf("Expected 2 undefined events, got %d", undefinedStats)
	}

	// Test all kind stats
	allStats, err := mockQueue.GetAllKindQueueStats()
	if err != nil {
		t.Fatalf("Failed to get all kind stats: %v", err)
	}

	expectedStats := map[int]int{
		0:     1,
		1:     2,
		3:     0,
		7:     1,
		10002: 0,
		-1:    2, // undefined kinds
	}

	for kind, expectedCount := range expectedStats {
		if allStats[kind] != expectedCount {
			t.Errorf("Expected %d events for kind %d, got %d", expectedCount, kind, allStats[kind])
		}
	}
}

func TestMockQueueWithErrorKindMethods(t *testing.T) {
	mockQueue := mocks.NewMockQueueWithError()

	// Test error handling for ConsumeEventsByKind
	mockQueue.SetErrors(nil, mocks.ErrConsumeFailed, nil)

	_, err := mockQueue.ConsumeEventsByKind(1)
	if err != mocks.ErrConsumeFailed {
		t.Errorf("Expected consume error, got %v", err)
	}

	// Test error handling for GetKindQueueStats
	mockQueue.SetErrors(nil, nil, mocks.ErrStatsFailed)

	_, err = mockQueue.GetKindQueueStats(1)
	if err != mocks.ErrStatsFailed {
		t.Errorf("Expected stats error, got %v", err)
	}

	// Test error handling for GetAllKindQueueStats
	_, err = mockQueue.GetAllKindQueueStats()
	if err != mocks.ErrStatsFailed {
		t.Errorf("Expected stats error, got %v", err)
	}
}

func TestKindTopicRouting(t *testing.T) {
	// Test that events are routed to correct topics based on kind
	mockQueue := mocks.NewMockQueue()

	// Create events with different kinds
	commonKindEvents := []*models.Event{
		{ID: "1", Kind: 0, Content: "Profile"},
		{ID: "2", Kind: 1, Content: "Note"},
		{ID: "3", Kind: 7, Content: "Reaction"},
	}

	undefinedKindEvents := []*models.Event{
		{ID: "4", Kind: 999, Content: "Unknown"},
		{ID: "5", Kind: 42, Content: "Another unknown"},
	}

	// Add all events
	allEvents := append(commonKindEvents, undefinedKindEvents...)
	for _, event := range allEvents {
		err := mockQueue.PublishEvent(event)
		if err != nil {
			t.Fatalf("Failed to publish event: %v", err)
		}
	}

	// Test that common kinds are routed to their specific topics
	for _, event := range commonKindEvents {
		events, err := mockQueue.ConsumeEventsByKind(event.Kind)
		if err != nil {
			t.Fatalf("Failed to consume kind %d events: %v", event.Kind, err)
		}
		if len(events) != 1 {
			t.Errorf("Expected 1 event for kind %d, got %d", event.Kind, len(events))
		}
		if events[0].ID != event.ID {
			t.Errorf("Expected event ID %s, got %s", event.ID, events[0].ID)
		}
	}

	// Test that undefined kinds are routed to undefined topic
	undefinedEvents, err := mockQueue.ConsumeEventsByKind(999)
	if err != nil {
		t.Fatalf("Failed to consume undefined events: %v", err)
	}
	if len(undefinedEvents) != 2 {
		t.Errorf("Expected 2 undefined events, got %d", len(undefinedEvents))
	}
}
