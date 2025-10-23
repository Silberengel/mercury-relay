package cache

import (
	"mercury-relay/internal/models"

	"github.com/nbd-wtf/go-nostr"
)

// Cache defines the interface for caching
type Cache interface {
	StoreEvent(event *models.Event) error
	GetEvents(filter nostr.Filter) ([]*models.Event, error)
	DeleteEvent(eventID string) error
	GetStats() (map[string]interface{}, error)
	Close() error
	
	// Replaceable event history methods
	GetReplaceableEventHistory(kind int, pubkey, dTag string) ([]map[string]interface{}, error)
	GetLatestReplaceableEvent(kind int, pubkey, dTag string) (*models.Event, error)
}
