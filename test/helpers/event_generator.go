package helpers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"mercury-relay/internal/models"

	"github.com/nbd-wtf/go-nostr"
)

// EventGenerator provides utilities for generating test Nostr events
type EventGenerator struct {
	PrivateKeys map[string]string // npub -> private key mapping
	PublicKeys  map[string]string // npub -> public key mapping
}

// NewEventGenerator creates a new event generator with test keys
func NewEventGenerator() *EventGenerator {
	eg := &EventGenerator{
		PrivateKeys: make(map[string]string),
		PublicKeys:  make(map[string]string),
	}

	// Generate some test keys
	eg.generateTestKeys(5)
	return eg
}

// generateTestKeys creates test key pairs
func (eg *EventGenerator) generateTestKeys(count int) {
	for i := 0; i < count; i++ {
		privateKey := eg.generatePrivateKey()
		publicKey := eg.privateKeyToPublicKey(privateKey)
		npub := eg.publicKeyToNpub(publicKey)

		eg.PrivateKeys[npub] = privateKey
		eg.PublicKeys[npub] = publicKey
	}
}

// generatePrivateKey generates a random 32-byte private key
func (eg *EventGenerator) generatePrivateKey() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// privateKeyToPublicKey converts private key to public key (simplified)
func (eg *EventGenerator) privateKeyToPublicKey(privateKey string) string {
	// This is a simplified implementation for testing
	// In reality, you'd use proper secp256k1 operations
	hash := sha256.Sum256([]byte(privateKey))
	return hex.EncodeToString(hash[:])
}

// publicKeyToNpub converts public key to npub format (simplified)
func (eg *EventGenerator) publicKeyToNpub(publicKey string) string {
	return "npub" + publicKey[:32] // Simplified for testing
}

// GenerateTextNote creates a kind 1 (text note) event
func (eg *EventGenerator) GenerateTextNote(npub string, content string, tags nostr.Tags) *models.Event {
	event := &models.Event{
		ID:        eg.generateEventID(npub, content),
		PubKey:    npub,
		CreatedAt: time.Now(),
		Kind:      1,
		Tags:      tags,
		Content:   content,
		Sig:       eg.generateSignature(npub, content),
	}
	event.QualityScore = event.CalculateQualityScore()
	return event
}

// GenerateUserMetadata creates a kind 0 (user metadata) event
func (eg *EventGenerator) GenerateUserMetadata(npub string, metadata map[string]interface{}) *models.Event {
	content, _ := json.Marshal(metadata)
	event := &models.Event{
		ID:        eg.generateEventID(npub, string(content)),
		PubKey:    npub,
		CreatedAt: time.Now(),
		Kind:      0,
		Tags:      nostr.Tags{},
		Content:   string(content),
		Sig:       eg.generateSignature(npub, string(content)),
	}
	event.QualityScore = event.CalculateQualityScore()
	return event
}

// GenerateFollowList creates a kind 3 (follow list) event
func (eg *EventGenerator) GenerateFollowList(npub string, followedNpubs []string) *models.Event {
	var tags nostr.Tags
	for _, followed := range followedNpubs {
		tags = append(tags, []string{"p", followed, "", "follow"})
	}

	event := &models.Event{
		ID:        eg.generateEventID(npub, "follows"),
		PubKey:    npub,
		CreatedAt: time.Now(),
		Kind:      3,
		Tags:      tags,
		Content:   "",
		Sig:       eg.generateSignature(npub, "follows"),
	}
	event.QualityScore = event.CalculateQualityScore()
	return event
}

// GenerateEbook creates a kind 30040 (ebook) event
func (eg *EventGenerator) GenerateEbook(npub string, bookMetadata map[string]interface{}) *models.Event {
	content, _ := json.Marshal(bookMetadata)
	event := &models.Event{
		ID:        eg.generateEventID(npub, string(content)),
		PubKey:    npub,
		CreatedAt: time.Now(),
		Kind:      30040,
		Tags:      nostr.Tags{[]string{"d", bookMetadata["identifier"].(string)}},
		Content:   string(content),
		Sig:       eg.generateSignature(npub, string(content)),
	}
	event.QualityScore = event.CalculateQualityScore()
	return event
}

// GenerateEbookContent creates a kind 30041 (ebook content) event
func (eg *EventGenerator) GenerateEbookContent(npub string, bookIdentifier string, chapterData map[string]interface{}) *models.Event {
	content, _ := json.Marshal(chapterData)
	event := &models.Event{
		ID:        eg.generateEventID(npub, string(content)),
		PubKey:    npub,
		CreatedAt: time.Now(),
		Kind:      30041,
		Tags:      nostr.Tags{[]string{"a", fmt.Sprintf("30040:%s:%s", npub, bookIdentifier)}, []string{"d", chapterData["identifier"].(string)}},
		Content:   string(content),
		Sig:       eg.generateSignature(npub, string(content)),
	}
	event.QualityScore = event.CalculateQualityScore()
	return event
}

// GenerateSpamEvent creates a low-quality event for spam testing
func (eg *EventGenerator) GenerateSpamEvent(npub string) *models.Event {
	// Generate event with very short content and many tags
	var tags nostr.Tags
	for i := 0; i < 25; i++ {
		tags = append(tags, []string{"t", fmt.Sprintf("spam%d", i)})
	}

	event := &models.Event{
		ID:        eg.generateEventID(npub, "spam"),
		PubKey:    npub,
		CreatedAt: time.Now(),
		Kind:      1,
		Tags:      tags,
		Content:   "spam", // Very short content
		Sig:       eg.generateSignature(npub, "spam"),
	}
	event.QualityScore = event.CalculateQualityScore()
	return event
}

// GenerateHighQualityEvent creates a high-quality event
func (eg *EventGenerator) GenerateHighQualityEvent(npub string) *models.Event {
	content := "This is a high-quality post with meaningful content that provides value to the community. " +
		"It contains thoughtful insights and relevant information that contributes to the conversation."

	tags := nostr.Tags{
		[]string{"t", "quality"},
		[]string{"t", "meaningful"},
	}

	event := &models.Event{
		ID:        eg.generateEventID(npub, content),
		PubKey:    npub,
		CreatedAt: time.Now(),
		Kind:      1,
		Tags:      tags,
		Content:   content,
		Sig:       eg.generateSignature(npub, content),
	}
	event.QualityScore = event.CalculateQualityScore()
	return event
}

// generateEventID creates a deterministic event ID for testing
func (eg *EventGenerator) generateEventID(npub, content string) string {
	data := fmt.Sprintf("%s%s%d", npub, content, time.Now().Unix())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// generateSignature creates a mock signature for testing
func (eg *EventGenerator) generateSignature(npub, content string) string {
	data := fmt.Sprintf("%s%s", npub, content)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// GetRandomNpub returns a random npub from the generator
func (eg *EventGenerator) GetRandomNpub() string {
	var npubs []string
	for npub := range eg.PrivateKeys {
		npubs = append(npubs, npub)
	}

	if len(npubs) == 0 {
		return ""
	}

	idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(npubs))))
	return npubs[idx.Int64()]
}

// GetOwnerNpub returns the first npub as owner
func (eg *EventGenerator) GetOwnerNpub() string {
	for npub := range eg.PrivateKeys {
		return npub
	}
	return ""
}

// GetFollowerNpub returns a non-owner npub
func (eg *EventGenerator) GetFollowerNpub() string {
	owner := eg.GetOwnerNpub()
	for npub := range eg.PrivateKeys {
		if npub != owner {
			return npub
		}
	}
	return ""
}

// GenerateEventBatch creates multiple events for testing
func (eg *EventGenerator) GenerateEventBatch(count int, kind int) []*models.Event {
	var events []*models.Event

	for i := 0; i < count; i++ {
		npub := eg.GetRandomNpub()

		switch kind {
		case 0:
			metadata := map[string]interface{}{
				"name": fmt.Sprintf("Test User %d", i),
			}
			events = append(events, eg.GenerateUserMetadata(npub, metadata))
		case 1:
			content := fmt.Sprintf("Test message %d with some content", i)
			events = append(events, eg.GenerateTextNote(npub, content, nostr.Tags{}))
		case 3:
			followed := []string{eg.GetRandomNpub()}
			events = append(events, eg.GenerateFollowList(npub, followed))
		default:
			content := fmt.Sprintf("Test content for kind %d, message %d", kind, i)
			events = append(events, eg.GenerateTextNote(npub, content, nostr.Tags{}))
		}
	}

	return events
}
