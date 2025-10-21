package mocks

import (
	"sync"

	"mercury-relay/internal/models"
)

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
