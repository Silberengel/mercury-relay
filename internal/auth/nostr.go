package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

// NostrAuthenticator handles NIP-42 authentication
type NostrAuthenticator struct {
	challenges        map[string]*Challenge
	authenticated     map[string]*AuthenticatedUser
	mu                sync.RWMutex
	RelayURL          string
	authorizedPubkeys []string
}

// Challenge represents a pending authentication challenge
type Challenge struct {
	Challenge string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// AuthenticatedUser represents an authenticated user
type AuthenticatedUser struct {
	Pubkey          string
	AuthenticatedAt time.Time
	ExpiresAt       time.Time
}

// NewNostrAuthenticator creates a new Nostr authenticator
func NewNostrAuthenticator(relayURL string, authorizedPubkeys []string) *NostrAuthenticator {
	return &NostrAuthenticator{
		challenges:        make(map[string]*Challenge),
		authenticated:     make(map[string]*AuthenticatedUser),
		RelayURL:          relayURL,
		authorizedPubkeys: authorizedPubkeys,
	}
}

// GenerateChallenge creates a new authentication challenge
func (na *NostrAuthenticator) GenerateChallenge() (string, error) {
	// Generate a random challenge string
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate challenge: %w", err)
	}

	challenge := hex.EncodeToString(bytes)

	na.mu.Lock()
	defer na.mu.Unlock()

	// Store challenge with expiration (10 minutes)
	now := time.Now()
	na.challenges[challenge] = &Challenge{
		Challenge: challenge,
		CreatedAt: now,
		ExpiresAt: now.Add(10 * time.Minute),
	}

	// Clean up expired challenges
	na.cleanupExpiredChallenges()

	return challenge, nil
}

// VerifyAuthentication verifies a NIP-42 authentication event
func (na *NostrAuthenticator) VerifyAuthentication(event *nostr.Event) error {
	na.mu.Lock()
	defer na.mu.Unlock()

	// Check if this is a valid authentication event (kind 22242)
	if event.Kind != 22242 {
		return fmt.Errorf("invalid event kind: expected 22242, got %d", event.Kind)
	}

	// Check if event is recent (within 10 minutes)
	now := time.Now()
	if event.CreatedAt.Time().Before(now.Add(-10 * time.Minute)) {
		return fmt.Errorf("event too old")
	}

	// Find challenge in tags
	var challenge string
	var relayURL string

	for _, tag := range event.Tags {
		if len(tag) >= 2 {
			switch tag[0] {
			case "challenge":
				challenge = tag[1]
			case "relay":
				relayURL = tag[1]
			}
		}
	}

	if challenge == "" {
		return fmt.Errorf("missing challenge tag")
	}

	if relayURL == "" {
		return fmt.Errorf("missing relay tag")
	}

	// Verify challenge exists and is valid
	challengeObj, exists := na.challenges[challenge]
	if !exists {
		return fmt.Errorf("invalid or expired challenge")
	}

	if time.Now().After(challengeObj.ExpiresAt) {
		delete(na.challenges, challenge)
		return fmt.Errorf("challenge expired")
	}

	// Verify relay URL matches
	if relayURL != na.RelayURL {
		return fmt.Errorf("relay URL mismatch")
	}

	// Verify signature
	valid, err := event.CheckSignature()
	if err != nil || !valid {
		return fmt.Errorf("invalid signature: %w", err)
	}

	// Check if pubkey is authorized
	if len(na.authorizedPubkeys) > 0 {
		authorized := false
		for _, authorizedPubkey := range na.authorizedPubkeys {
			if event.PubKey == authorizedPubkey {
				authorized = true
				break
			}
		}
		if !authorized {
			return fmt.Errorf("pubkey not authorized")
		}
	}

	// Mark user as authenticated
	na.authenticated[event.PubKey] = &AuthenticatedUser{
		Pubkey:          event.PubKey,
		AuthenticatedAt: now,
		ExpiresAt:       now.Add(24 * time.Hour), // Authentication valid for 24 hours
	}

	// Remove used challenge
	delete(na.challenges, challenge)

	log.Printf("User %s authenticated successfully", event.PubKey)
	return nil
}

// IsAuthenticated checks if a pubkey is currently authenticated
func (na *NostrAuthenticator) IsAuthenticated(pubkey string) bool {
	na.mu.RLock()
	defer na.mu.RUnlock()

	user, exists := na.authenticated[pubkey]
	if !exists {
		return false
	}

	// Check if authentication is still valid
	if time.Now().After(user.ExpiresAt) {
		// Authentication expired, remove it
		na.mu.RUnlock()
		na.mu.Lock()
		delete(na.authenticated, pubkey)
		na.mu.Unlock()
		na.mu.RLock()
		return false
	}

	return true
}

// GetAuthenticatedUser returns the authenticated user info
func (na *NostrAuthenticator) GetAuthenticatedUser(pubkey string) (*AuthenticatedUser, bool) {
	na.mu.RLock()
	defer na.mu.RUnlock()

	user, exists := na.authenticated[pubkey]
	if !exists {
		return nil, false
	}

	// Check if authentication is still valid
	if time.Now().After(user.ExpiresAt) {
		// Authentication expired, remove it
		na.mu.RUnlock()
		na.mu.Lock()
		delete(na.authenticated, pubkey)
		na.mu.Unlock()
		na.mu.RLock()
		return nil, false
	}

	return user, true
}

// cleanupExpiredChallenges removes expired challenges
func (na *NostrAuthenticator) cleanupExpiredChallenges() {
	now := time.Now()
	for challenge, challengeObj := range na.challenges {
		if now.After(challengeObj.ExpiresAt) {
			delete(na.challenges, challenge)
		}
	}
}

// cleanupExpiredAuthentications removes expired authentications
func (na *NostrAuthenticator) cleanupExpiredAuthentications() {
	now := time.Now()
	for pubkey, user := range na.authenticated {
		if now.After(user.ExpiresAt) {
			delete(na.authenticated, pubkey)
		}
	}
}

// Cleanup removes expired challenges and authentications
func (na *NostrAuthenticator) Cleanup() {
	na.mu.Lock()
	defer na.mu.Unlock()

	na.cleanupExpiredChallenges()
	na.cleanupExpiredAuthentications()
}

// GetStats returns authentication statistics
func (na *NostrAuthenticator) GetStats() map[string]interface{} {
	na.mu.RLock()
	defer na.mu.RUnlock()

	return map[string]interface{}{
		"active_challenges":   len(na.challenges),
		"authenticated_users": len(na.authenticated),
		"authorized_pubkeys":  len(na.authorizedPubkeys),
	}
}
