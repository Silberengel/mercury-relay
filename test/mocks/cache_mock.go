package mocks

import (
	"sync"

	"mercury-relay/internal/models"

	"github.com/nbd-wtf/go-nostr"
)

// MockCache implements the cache interface for testing
type MockCache struct {
	events map[string]*models.Event
	stats  map[string]interface{}
	mutex  sync.RWMutex
}

// NewMockCache creates a new mock cache
func NewMockCache() *MockCache {
	return &MockCache{
		events: make(map[string]*models.Event),
		stats:  make(map[string]interface{}),
	}
}

// StoreEvent stores an event in the mock cache
func (m *MockCache) StoreEvent(event *models.Event) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.events[event.ID] = event
	m.updateStats()
	return nil
}

// GetEvents retrieves events matching the filter
func (m *MockCache) GetEvents(filter nostr.Filter) ([]*models.Event, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var result []*models.Event

	for _, event := range m.events {
		if m.eventMatchesFilter(event, filter) {
			result = append(result, event)
		}
	}

	// Apply limit
	if filter.Limit > 0 && len(result) > filter.Limit {
		result = result[:filter.Limit]
	}

	return result, nil
}

// DeleteEvent removes an event from the mock cache
func (m *MockCache) DeleteEvent(eventID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.events, eventID)
	m.updateStats()
	return nil
}

// GetStats returns cache statistics
func (m *MockCache) GetStats() (map[string]interface{}, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.stats, nil
}

// Close closes the mock cache
func (m *MockCache) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.events = make(map[string]*models.Event)
	m.stats = make(map[string]interface{})
	return nil
}

// Helper methods for testing

// SetEvents sets multiple events at once
func (m *MockCache) SetEvents(events []*models.Event) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.events = make(map[string]*models.Event)
	for _, event := range events {
		m.events[event.ID] = event
	}
	m.updateStats()
}

// GetEventCount returns the number of stored events
func (m *MockCache) GetEventCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.events)
}

// GetEvent returns a specific event by ID
func (m *MockCache) GetEvent(eventID string) *models.Event {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.events[eventID]
}

// Clear removes all events
func (m *MockCache) Clear() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.events = make(map[string]*models.Event)
	m.updateStats()
}

// HasEvent checks if an event exists
func (m *MockCache) HasEvent(eventID string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	_, exists := m.events[eventID]
	return exists
}

// GetEventsByAuthor returns all events from a specific author
func (m *MockCache) GetEventsByAuthor(pubkey string) []*models.Event {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var result []*models.Event
	for _, event := range m.events {
		if event.PubKey == pubkey {
			result = append(result, event)
		}
	}
	return result
}

// GetEventsByKind returns all events of a specific kind
func (m *MockCache) GetEventsByKind(kind int) []*models.Event {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var result []*models.Event
	for _, event := range m.events {
		if event.Kind == kind {
			result = append(result, event)
		}
	}
	return result
}

// Private methods

func (m *MockCache) eventMatchesFilter(event *models.Event, filter nostr.Filter) bool {
	// Check authors
	if len(filter.Authors) > 0 {
		found := false
		for _, author := range filter.Authors {
			if event.PubKey == author {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check kinds
	if len(filter.Kinds) > 0 {
		found := false
		for _, kind := range filter.Kinds {
			if event.Kind == kind {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

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

	// Check IDs
	if len(filter.IDs) > 0 {
		found := false
		for _, id := range filter.IDs {
			if event.ID == id {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func (m *MockCache) updateStats() {
	m.stats["total_events"] = len(m.events)
	m.stats["cache_size"] = len(m.events)
}

// MockCacheWithError is a cache that returns errors for testing
type MockCacheWithError struct {
	*MockCache
	storeError  error
	getError    error
	deleteError error
	statsError  error
}

// NewMockCacheWithError creates a mock cache that can return errors
func NewMockCacheWithError() *MockCacheWithError {
	return &MockCacheWithError{
		MockCache: NewMockCache(),
	}
}

// SetErrors sets the errors to return
func (m *MockCacheWithError) SetErrors(storeError, getError, deleteError, statsError error) {
	m.storeError = storeError
	m.getError = getError
	m.deleteError = deleteError
	m.statsError = statsError
}

// StoreEvent returns configured error
func (m *MockCacheWithError) StoreEvent(event *models.Event) error {
	if m.storeError != nil {
		return m.storeError
	}
	return m.MockCache.StoreEvent(event)
}

// GetEvents returns configured error
func (m *MockCacheWithError) GetEvents(filter nostr.Filter) ([]*models.Event, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	return m.MockCache.GetEvents(filter)
}

// DeleteEvent returns configured error
func (m *MockCacheWithError) DeleteEvent(eventID string) error {
	if m.deleteError != nil {
		return m.deleteError
	}
	return m.MockCache.DeleteEvent(eventID)
}

// GetStats returns configured error
func (m *MockCacheWithError) GetStats() (map[string]interface{}, error) {
	if m.statsError != nil {
		return nil, m.statsError
	}
	return m.MockCache.GetStats()
}

// GetReplaceableEventHistory returns mock history for replaceable events
func (m *MockCache) GetReplaceableEventHistory(kind int, pubkey, dTag string) ([]map[string]interface{}, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Mock implementation - return empty history
	return []map[string]interface{}{}, nil
}

// GetLatestReplaceableEvent returns the latest version of a replaceable event
func (m *MockCache) GetLatestReplaceableEvent(kind int, pubkey, dTag string) (*models.Event, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Mock implementation - return nil (not found)
	return nil, nil
}

// GetReplaceableEventHistory returns configured error
func (m *MockCacheWithError) GetReplaceableEventHistory(kind int, pubkey, dTag string) ([]map[string]interface{}, error) {
	if m.storeError != nil {
		return nil, m.storeError
	}
	return m.MockCache.GetReplaceableEventHistory(kind, pubkey, dTag)
}

// GetLatestReplaceableEvent returns configured error
func (m *MockCacheWithError) GetLatestReplaceableEvent(kind int, pubkey, dTag string) (*models.Event, error) {
	if m.storeError != nil {
		return nil, m.storeError
	}
	return m.MockCache.GetLatestReplaceableEvent(kind, pubkey, dTag)
}
