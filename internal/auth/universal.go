package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"mercury-relay/internal/cache"
	"mercury-relay/internal/config"
	"mercury-relay/internal/models"
	"mercury-relay/internal/queue"

	"github.com/nbd-wtf/go-nostr"
)

// UniversalAuthenticator handles authentication for all relay endpoints
type UniversalAuthenticator struct {
	config         *config.Config
	nostrAuth      *NostrAuthenticator
	cache          cache.Cache
	queue          queue.Queue
	adminNpubs     map[string]bool
	whitelist      map[string]bool
	whitelistMutex sync.RWMutex
	kind3Cache     map[string]bool
	kind3CacheTime map[string]time.Time
	kind3Mutex     sync.RWMutex
}

// NewUniversalAuthenticator creates a new universal authenticator
func NewUniversalAuthenticator(
	config *config.Config,
	relayURL string,
	cache cache.Cache,
	queue queue.Queue,
) *UniversalAuthenticator {
	// Initialize admin npubs from config
	adminNpubs := make(map[string]bool)
	for _, npub := range config.Access.AdminNpubs {
		adminNpubs[npub] = true
	}

	nostrAuth := NewNostrAuthenticator(relayURL, config.Access.AdminNpubs)

	return &UniversalAuthenticator{
		config:         config,
		nostrAuth:      nostrAuth,
		cache:          cache,
		queue:          queue,
		adminNpubs:     adminNpubs,
		whitelist:      make(map[string]bool),
		kind3Cache:     make(map[string]bool),
		kind3CacheTime: make(map[string]time.Time),
	}
}

// AuthenticateRequest checks if a request is authorized
func (ua *UniversalAuthenticator) AuthenticateRequest(r *http.Request) bool {
	// Get Nostr pubkey from header
	npub := r.Header.Get("X-Nostr-Pubkey")
	if npub == "" {
		return false
	}

	// Check if user is authenticated via NIP-42
	if !ua.nostrAuth.IsAuthenticated(npub) {
		return false
	}

	// Check if user is an admin
	if ua.IsAdmin(npub) {
		return true
	}

	// Check if user is on whitelist
	ua.whitelistMutex.RLock()
	whitelisted := ua.whitelist[npub]
	ua.whitelistMutex.RUnlock()

	if whitelisted {
		return true
	}

	// Check if any admin follows this user (kind 3 event)
	// This means the admin has explicitly added this user to their contact list
	if ua.checkAdminFollowsUser(npub) {
		// Add to whitelist for future requests
		ua.whitelistMutex.Lock()
		ua.whitelist[npub] = true
		ua.whitelistMutex.Unlock()
		return true
	}

	return false
}

// checkAdminFollowsUser checks if any admin follows this user via kind 3 event
func (ua *UniversalAuthenticator) checkAdminFollowsUser(userNpub string) bool {
	// Check cache first
	cacheKey := fmt.Sprintf("admin_follows_%s", userNpub)
	ua.kind3Mutex.RLock()
	if cached, exists := ua.kind3Cache[cacheKey]; exists {
		if time.Since(ua.kind3CacheTime[cacheKey]) < time.Hour {
			ua.kind3Mutex.RUnlock()
			return cached
		}
	}
	ua.kind3Mutex.RUnlock()

	// Check each admin's kind 3 events
	for adminNpub := range ua.adminNpubs {
		// Create filter for kind 3 events from this admin
		filter := nostr.Filter{
			Authors: []string{adminNpub},
			Kinds:   []int{3}, // kind 3 = contact list
			Limit:   1,
		}

		// Query for kind 3 events from this admin
		events, err := ua.queryEvents(filter)
		if err != nil {
			log.Printf("Error querying kind 3 events for admin %s: %v", adminNpub, err)
			continue
		}

		// Check if any kind 3 event from this admin contains the user's npub
		for _, event := range events {
			if ua.containsUserNpub(event, userNpub) {
				// Cache the result for 1 hour
				ua.kind3Mutex.Lock()
				ua.kind3Cache[cacheKey] = true
				ua.kind3CacheTime[cacheKey] = time.Now()
				ua.kind3Mutex.Unlock()
				return true
			}
		}
	}

	// Cache negative result for 10 minutes
	ua.kind3Mutex.Lock()
	ua.kind3Cache[cacheKey] = false
	ua.kind3CacheTime[cacheKey] = time.Now()
	ua.kind3Mutex.Unlock()
	return false
}

// containsUserNpub checks if a kind 3 event contains the specific user's npub
func (ua *UniversalAuthenticator) containsUserNpub(event *models.Event, userNpub string) bool {
	// Parse the content as JSON to get the contact list
	var contacts map[string]interface{}
	if err := json.Unmarshal([]byte(event.Content), &contacts); err != nil {
		// If not JSON, check tags for npub
		for _, tag := range event.Tags {
			if len(tag) >= 2 && tag[0] == "p" && tag[1] == userNpub {
				return true
			}
		}
		return false
	}

	// Check if the user's npub is in the contacts
	for _, contact := range contacts {
		if contactMap, ok := contact.(map[string]interface{}); ok {
			if pubkey, exists := contactMap["pubkey"]; exists {
				if pubkey == userNpub {
					return true
				}
			}
		}
	}

	return false
}

// queryEvents queries for events using the queue
func (ua *UniversalAuthenticator) queryEvents(filter nostr.Filter) ([]*models.Event, error) {
	// This would need to be implemented to query the event store
	// For now, return empty slice
	return []*models.Event{}, nil
}

// AddToWhitelist adds a user to the whitelist (admin only)
func (ua *UniversalAuthenticator) AddToWhitelist(adminNpub, userNpub string) error {
	if !ua.IsAdmin(adminNpub) {
		return fmt.Errorf("only admin can modify whitelist")
	}

	ua.whitelistMutex.Lock()
	ua.whitelist[userNpub] = true
	ua.whitelistMutex.Unlock()

	log.Printf("Added %s to whitelist by admin %s", userNpub, adminNpub)
	return nil
}

// RemoveFromWhitelist removes a user from the whitelist (admin only)
func (ua *UniversalAuthenticator) RemoveFromWhitelist(adminNpub, userNpub string) error {
	if !ua.IsAdmin(adminNpub) {
		return fmt.Errorf("only admin can modify whitelist")
	}

	ua.whitelistMutex.Lock()
	delete(ua.whitelist, userNpub)
	ua.whitelistMutex.Unlock()

	log.Printf("Removed %s from whitelist by admin %s", userNpub, adminNpub)
	return nil
}

// GetWhitelist returns the current whitelist (admin only)
func (ua *UniversalAuthenticator) GetWhitelist(adminNpub string) ([]string, error) {
	if !ua.IsAdmin(adminNpub) {
		return nil, fmt.Errorf("only admin can view whitelist")
	}

	ua.whitelistMutex.RLock()
	defer ua.whitelistMutex.RUnlock()

	whitelist := make([]string, 0, len(ua.whitelist))
	for npub := range ua.whitelist {
		whitelist = append(whitelist, npub)
	}

	return whitelist, nil
}

// IsAdmin checks if a pubkey is an admin
func (ua *UniversalAuthenticator) IsAdmin(npub string) bool {
	ua.whitelistMutex.RLock()
	defer ua.whitelistMutex.RUnlock()
	return ua.adminNpubs[npub]
}

// GetAdminNpubs returns all admin npubs
func (ua *UniversalAuthenticator) GetAdminNpubs() []string {
	ua.whitelistMutex.RLock()
	defer ua.whitelistMutex.RUnlock()

	admins := make([]string, 0, len(ua.adminNpubs))
	for npub := range ua.adminNpubs {
		admins = append(admins, npub)
	}
	return admins
}

// RequireAuth middleware for HTTP handlers
func (ua *UniversalAuthenticator) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !ua.AuthenticateRequest(r) {
			http.Error(w, "Unauthorized: Nostr authentication required", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// RequireAdmin middleware for admin-only operations
func (ua *UniversalAuthenticator) RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		npub := r.Header.Get("X-Nostr-Pubkey")
		if npub == "" || !ua.IsAdmin(npub) {
			http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

// GetAuthenticatedNpub extracts the authenticated npub from request
func (ua *UniversalAuthenticator) GetAuthenticatedNpub(r *http.Request) string {
	return r.Header.Get("X-Nostr-Pubkey")
}
