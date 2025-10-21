package models

import (
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

// Event represents a Nostr event with additional metadata
type Event struct {
	ID               string          `json:"id" db:"id"`
	PubKey           string          `json:"pubkey" db:"pubkey"`
	CreatedAt        nostr.Timestamp `json:"created_at" db:"created_at"`
	Kind             int             `json:"kind" db:"kind"`
	Tags             nostr.Tags      `json:"tags" db:"tags"`
	Content          string          `json:"content" db:"content"`
	Sig              string          `json:"sig" db:"sig"`
	QualityScore     float64         `json:"quality_score" db:"quality_score"`
	IsQuarantined    bool            `json:"is_quarantined" db:"is_quarantined"`
	QuarantineReason string          `json:"quarantine_reason" db:"quarantine_reason"`
	CreatedAtDB      time.Time       `json:"created_at_db" db:"created_at_db"`
}

// ToNostrEvent converts our Event to a nostr.Event
func (e *Event) ToNostrEvent() *nostr.Event {
	return &nostr.Event{
		ID:        e.ID,
		PubKey:    e.PubKey,
		CreatedAt: e.CreatedAt,
		Kind:      e.Kind,
		Tags:      e.Tags,
		Content:   e.Content,
		Sig:       e.Sig,
	}
}

// FromNostrEvent creates an Event from a nostr.Event
func FromNostrEvent(ne *nostr.Event) *Event {
	return &Event{
		ID:          ne.ID,
		PubKey:      ne.PubKey,
		CreatedAt:   ne.CreatedAt,
		Kind:        ne.Kind,
		Tags:        ne.Tags,
		Content:     ne.Content,
		Sig:         ne.Sig,
		CreatedAtDB: time.Now(),
	}
}

// Validate performs basic validation on the event
func (e *Event) Validate() error {
	// Check if event is not too old (1 hour tolerance)
	if time.Since(e.CreatedAt.Time()) > time.Hour {
		return ErrEventTooOld
	}

	// Check if event is not in the future (5 minutes tolerance)
	if e.CreatedAt.Time().After(time.Now().Add(5 * time.Minute)) {
		return ErrEventInFuture
	}

	// Check content length
	if len(e.Content) > 10000 {
		return ErrContentTooLong
	}

	// Check required fields
	if e.ID == "" || e.PubKey == "" || e.Sig == "" {
		return ErrMissingRequiredFields
	}

	return nil
}

// CalculateQualityScore calculates a quality score for the event
func (e *Event) CalculateQualityScore() float64 {
	score := 1.0

	// Penalize very short content
	if len(e.Content) < 10 {
		score -= 0.2
	}

	// Penalize very long content
	if len(e.Content) > 5000 {
		score -= 0.1
	}

	// Bonus for reasonable content length
	if len(e.Content) >= 50 && len(e.Content) <= 1000 {
		score += 0.1
	}

	// Penalize events with too many tags (potential spam)
	if len(e.Tags) > 20 {
		score -= 0.3
	}

	// Bonus for events with reasonable tag count
	if len(e.Tags) >= 1 && len(e.Tags) <= 5 {
		score += 0.1
	}

	// Ensure score is between 0 and 1
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}

// IsSpam checks if the event appears to be spam
func (e *Event) IsSpam(threshold float64) bool {
	return e.CalculateQualityScore() < threshold
}

// MarshalJSON and UnmarshalJSON are no longer needed since nostr.Timestamp handles JSON serialization

// Error definitions
var (
	ErrEventTooOld           = fmt.Errorf("event is too old")
	ErrEventInFuture         = fmt.Errorf("event is in the future")
	ErrContentTooLong        = fmt.Errorf("content is too long")
	ErrMissingRequiredFields = fmt.Errorf("missing required fields")
)
