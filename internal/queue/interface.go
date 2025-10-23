package queue

import "mercury-relay/internal/models"

// Queue defines the interface for message queuing
type Queue interface {
	PublishEvent(event *models.Event) error
	ConsumeEvents() ([]*models.Event, error)
	GetQueueStats() (int, error)
	Close() error

	// Kind-based topic methods
	ConsumeEventsByKind(kind int) ([]*models.Event, error)
	GetKindQueueStats(kind int) (int, error)
	GetAllKindQueueStats() (map[int]int, error)
}
