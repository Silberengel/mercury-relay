package relay

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"mercury-relay/internal/models"
)

// PrivacyFilter handles privacy-aware event filtering
type PrivacyFilter struct {
	requesterPubkey string
}

// NewPrivacyFilter creates a new privacy filter for a specific requester
func NewPrivacyFilter(requesterPubkey string) *PrivacyFilter {
	return &PrivacyFilter{
		requesterPubkey: requesterPubkey,
	}
}

// CanAccessEvent determines if the requester can access a specific event
func (pf *PrivacyFilter) CanAccessEvent(event *models.Event) bool {
	// Always allow access to events authored by the requester
	if event.PubKey == pf.requesterPubkey {
		return true
	}

	// Handle DMs (kind 4) - only return to sender or recipient
	if event.Kind == 4 {
		return pf.canAccessDM(event)
	}

	// Handle encrypted events (kind 1059) - only return to author
	if event.Kind == 1059 {
		return event.PubKey == pf.requesterPubkey
	}

	// Handle other encrypted events (kinds 1060, 1061, etc.) - only return to author
	if event.Kind >= 1059 && event.Kind <= 1999 {
		return event.PubKey == pf.requesterPubkey
	}

	// For all other events, allow access
	return true
}

// canAccessDM checks if the requester can access a DM
func (pf *PrivacyFilter) canAccessDM(event *models.Event) bool {
	// Extract recipient from 'p' tag
	for _, tag := range event.Tags {
		if len(tag) >= 2 && tag[0] == "p" {
			recipient := tag[1]
			// Allow access if requester is the sender or recipient
			if recipient == pf.requesterPubkey {
				return true
			}
		}
	}

	// If no 'p' tag found, only allow access to the author
	return event.PubKey == pf.requesterPubkey
}

// IsReplaceableEvent checks if an event is replaceable
func IsReplaceableEvent(kind int) bool {
	// Replaceable events according to NIP-16
	replaceableKinds := map[int]bool{
		0:     true, // User metadata
		3:     true, // Contacts
		5:     true, // Event deletion
		6:     true, // Repost
		7:     true, // Reaction
		8:     true, // Badge award
		40:    true, // Channel creation
		41:    true, // Channel metadata
		42:    true, // Channel message
		43:    true, // Hide message
		44:    true, // Mute user
		10002: true, // Relay list
		30000: true, // Follow sets
		30001: true, // Follow sets
		30008: true, // Profile badges
		30009: true, // Badge definition
		30078: true, // Application-specific data
	}

	return replaceableKinds[kind]
}

// GetReplaceableEventKey generates the key for replaceable events (kind:pubkey:d-tag)
func GetReplaceableEventKey(event *models.Event) string {
	if !IsReplaceableEvent(event.Kind) {
		return ""
	}

	// Find d-tag
	var dTag string
	for _, tag := range event.Tags {
		if len(tag) >= 2 && tag[0] == "d" {
			dTag = tag[1]
			break
		}
	}

	// If no d-tag, use empty string
	if dTag == "" {
		dTag = ""
	}

	return fmt.Sprintf("%d:%s:%s", event.Kind, event.PubKey, dTag)
}

// IsEncryptedEvent checks if an event is encrypted
func IsEncryptedEvent(event *models.Event) bool {
	// Check if content starts with nostr: (encrypted content indicator)
	return strings.HasPrefix(event.Content, "nostr:")
}

// GetEventHash generates a hash for event comparison
func GetEventHash(event *models.Event) string {
	// Create a hash of the event content for comparison
	content := fmt.Sprintf("%s:%s:%d:%s", event.ID, event.PubKey, event.Kind, event.Content)
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// EventVersion represents a version of a replaceable event
type EventVersion struct {
	Event     *models.Event `json:"event"`
	Version   int           `json:"version"`
	CreatedAt int64         `json:"created_at"`
	Hash      string        `json:"hash"`
}

// EventHistory represents the history of a replaceable event
type EventHistory struct {
	Key      string         `json:"key"`      // kind:pubkey:d-tag
	Versions []EventVersion `json:"versions"` // All versions in chronological order
	Latest   *EventVersion  `json:"latest"`   // Latest version
}

// EventDiff represents the difference between two event versions
type EventDiff struct {
	FromVersion int                    `json:"from_version"`
	ToVersion   int                    `json:"to_version"`
	Changes     map[string]interface{} `json:"changes"`
	Added       []string               `json:"added"`
	Removed     []string               `json:"removed"`
	Modified    []string               `json:"modified"`
}
