package mocks

import (
	"errors"
	"sync"

	"mercury-relay/internal/models"
)

// Error constants for testing
var (
	ErrPublishFailed = errors.New("publish failed")
	ErrConsumeFailed = errors.New("consume failed")
	ErrStatsFailed   = errors.New("stats failed")
)

// getCommonKinds returns the list of Nostr event kinds from quality control configuration
func getCommonKinds() []int {
	// For testing, use a reasonable set of common kinds
	// In production, this would load from the actual config file
	return []int{0, 1, 3, 4, 7, 10000, 10001, 20000, 30000, 30001, 30040, 30041, 1040}
}

// isCommonKind checks if a kind is in the configured kinds list
func isCommonKind(kind int) bool {
	commonKinds := getCommonKinds()
	for _, commonKind := range commonKinds {
		if kind == commonKind {
			return true
		}
	}
	return false
}

// MockQueue implements the queue interface for testing
type MockQueue struct {
	events []*models.Event
	stats  map[string]interface{}
	mutex  sync.RWMutex
}

// NewMockQueue creates a new mock queue
func NewMockQueue() *MockQueue {
	return &MockQueue{
		events: make([]*models.Event, 0),
		stats:  make(map[string]interface{}),
	}
}

// PublishEvent adds an event to the queue
func (m *MockQueue) PublishEvent(event *models.Event) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.events = append(m.events, event)
	m.updateStats()
	return nil
}

// ConsumeEvents removes and returns events from the queue
func (m *MockQueue) ConsumeEvents() ([]*models.Event, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if len(m.events) == 0 {
		return []*models.Event{}, nil
	}

	// Return all events and clear the queue
	result := make([]*models.Event, len(m.events))
	copy(result, m.events)
	m.events = make([]*models.Event, 0)
	m.updateStats()

	return result, nil
}

// GetQueueStats returns queue statistics
func (m *MockQueue) GetQueueStats() (int, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return len(m.events), nil
}

// Close closes the mock queue
func (m *MockQueue) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.events = make([]*models.Event, 0)
	m.stats = make(map[string]interface{})
	return nil
}

// Helper methods for testing

// GetEventCount returns the number of queued events
func (m *MockQueue) GetEventCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.events)
}

// GetEvents returns all queued events without removing them
func (m *MockQueue) GetEvents() []*models.Event {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make([]*models.Event, len(m.events))
	copy(result, m.events)
	return result
}

// Clear removes all events from the queue
func (m *MockQueue) Clear() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.events = make([]*models.Event, 0)
	m.updateStats()
}

// Peek returns the first event without removing it
func (m *MockQueue) Peek() *models.Event {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if len(m.events) == 0 {
		return nil
	}
	return m.events[0]
}

// GetStats returns detailed queue statistics
func (m *MockQueue) GetStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.stats
}

// Private methods

func (m *MockQueue) updateStats() {
	m.stats["queue_size"] = len(m.events)
	m.stats["total_events"] = len(m.events)
}

// MockQueueWithError is a queue that returns errors for testing
type MockQueueWithError struct {
	*MockQueue
	publishError error
	consumeError error
	statsError   error
}

// NewMockQueueWithError creates a mock queue that can return errors
func NewMockQueueWithError() *MockQueueWithError {
	return &MockQueueWithError{
		MockQueue: NewMockQueue(),
	}
}

// SetErrors sets the errors to return
func (m *MockQueueWithError) SetErrors(publishError, consumeError, statsError error) {
	m.publishError = publishError
	m.consumeError = consumeError
	m.statsError = statsError
}

// PublishEvent returns configured error
func (m *MockQueueWithError) PublishEvent(event *models.Event) error {
	if m.publishError != nil {
		return m.publishError
	}
	return m.MockQueue.PublishEvent(event)
}

// ConsumeEvents returns configured error
func (m *MockQueueWithError) ConsumeEvents() ([]*models.Event, error) {
	if m.consumeError != nil {
		return nil, m.consumeError
	}
	return m.MockQueue.ConsumeEvents()
}

// GetQueueStats returns configured error
func (m *MockQueueWithError) GetQueueStats() (int, error) {
	if m.statsError != nil {
		return 0, m.statsError
	}
	return m.MockQueue.GetQueueStats()
}

// Kind-based topic methods for MockQueue

// ConsumeEventsByKind returns events filtered by kind
func (m *MockQueue) ConsumeEventsByKind(kind int) ([]*models.Event, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var kindEvents []*models.Event
	var remainingEvents []*models.Event

	// Handle undefined kinds
	for _, event := range m.events {
		if isCommonKind(kind) && event.Kind == kind {
			kindEvents = append(kindEvents, event)
		} else if !isCommonKind(kind) && !isCommonKind(event.Kind) {
			// For undefined kinds, collect all non-common kind events
			kindEvents = append(kindEvents, event)
		} else {
			remainingEvents = append(remainingEvents, event)
		}
	}

	// Update the queue to remove consumed kind events
	m.events = remainingEvents
	m.updateStats()

	return kindEvents, nil
}

// GetKindQueueStats returns the number of events of a specific kind
func (m *MockQueue) GetKindQueueStats(kind int) (int, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	count := 0

	for _, event := range m.events {
		if isCommonKind(kind) && event.Kind == kind {
			count++
		} else if !isCommonKind(kind) && !isCommonKind(event.Kind) {
			// For undefined kinds, count all non-common kind events
			count++
		}
	}
	return count, nil
}

// GetAllKindQueueStats returns stats for all kinds
func (m *MockQueue) GetAllKindQueueStats() (map[int]int, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := make(map[int]int)
	commonKinds := getCommonKinds()

	// Initialize stats for common kinds
	for _, kind := range commonKinds {
		stats[kind] = 0
	}
	stats[-1] = 0 // undefined kinds

	// Count events
	for _, event := range m.events {
		if isCommonKind(event.Kind) {
			stats[event.Kind]++
		} else {
			stats[-1]++ // undefined kinds
		}
	}

	return stats, nil
}

// Kind-based topic methods for MockQueueWithError

// ConsumeEventsByKind returns configured error or filtered events
func (m *MockQueueWithError) ConsumeEventsByKind(kind int) ([]*models.Event, error) {
	if m.consumeError != nil {
		return nil, m.consumeError
	}
	return m.MockQueue.ConsumeEventsByKind(kind)
}

// GetKindQueueStats returns configured error or kind stats
func (m *MockQueueWithError) GetKindQueueStats(kind int) (int, error) {
	if m.statsError != nil {
		return 0, m.statsError
	}
	return m.MockQueue.GetKindQueueStats(kind)
}

// GetAllKindQueueStats returns configured error or all kind stats
func (m *MockQueueWithError) GetAllKindQueueStats() (map[int]int, error) {
	if m.statsError != nil {
		return nil, m.statsError
	}
	return m.MockQueue.GetAllKindQueueStats()
}
