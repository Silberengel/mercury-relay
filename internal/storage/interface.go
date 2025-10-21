package storage

import "mercury-relay/internal/models"

// Storage defines the interface for event storage
type Storage interface {
	StoreEvent(event *models.Event) error
	GetEvent(eventID string) (*models.Event, error)
	DeleteEvent(eventID string) error
	GetStats() (map[string]interface{}, error)
	Close() error
}
