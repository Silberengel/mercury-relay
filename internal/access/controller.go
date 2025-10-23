package access

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"mercury-relay/internal/config"
)

type Controller struct {
	config       config.AccessConfig
	ownerNpub    string
	allowedNpubs map[string]bool
	npubMutex    sync.RWMutex
	lastUpdate   time.Time
	updateTicker *time.Ticker
	httpClient   *http.Client
}

type AccessConfig struct {
	OwnerNpub        string        `yaml:"owner_npub"`
	UpdateInterval   time.Duration `yaml:"update_interval"`
	RelayURL         string        `yaml:"relay_url"`
	AllowPublicRead  bool          `yaml:"allow_public_read"`
	AllowPublicWrite bool          `yaml:"allow_public_write"`
}

func NewController(config config.AccessConfig) *Controller {
	// Use the first admin npub as the primary owner for backward compatibility
	ownerNpub := ""
	if len(config.AdminNpubs) > 0 {
		ownerNpub = config.AdminNpubs[0]
	}

	return &Controller{
		config:       config,
		ownerNpub:    ownerNpub,
		allowedNpubs: make(map[string]bool),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (a *Controller) Start(ctx context.Context) error {
	// Load initial follow list
	if err := a.loadFollowList(); err != nil {
		log.Printf("Failed to load initial follow list: %v", err)
	}

	// Start periodic updates
	a.updateTicker = time.NewTicker(a.config.UpdateInterval)
	go a.updateLoop(ctx)

	return nil
}

func (a *Controller) Stop() {
	if a.updateTicker != nil {
		a.updateTicker.Stop()
	}
}

func (a *Controller) CanWrite(npub string) bool {
	// Owner can always write
	if npub == a.ownerNpub {
		return true
	}

	// Check if public write is allowed
	if a.config.AllowPublicWrite {
		return true
	}

	// Check if npub is in allowed list
	a.npubMutex.RLock()
	defer a.npubMutex.RUnlock()

	return a.allowedNpubs[npub]
}

func (a *Controller) CanRead(npub string) bool {
	// Public read is always allowed if configured
	if a.config.AllowPublicRead {
		return true
	}

	// Owner can always read
	if npub == a.ownerNpub {
		return true
	}

	// Check if npub is in allowed list
	a.npubMutex.RLock()
	defer a.npubMutex.RUnlock()

	return a.allowedNpubs[npub]
}

func (a *Controller) GetAllowedNpubs() []string {
	a.npubMutex.RLock()
	defer a.npubMutex.RUnlock()

	var npubs []string
	for npub := range a.allowedNpubs {
		npubs = append(npubs, npub)
	}
	return npubs
}

func (a *Controller) IsOwner(npub string) bool {
	return npub == a.ownerNpub
}

func (a *Controller) loadFollowList() error {
	// Query the owner's Kind 3 (follow list) event
	req := map[string]interface{}{
		"ids":   []string{a.ownerNpub},
		"kinds": []int{3},
		"limit": 1,
	}

	reqBody, err := json.Marshal([]interface{}{
		"REQ",
		"follow-list",
		req,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send request to relay
	resp, err := a.httpClient.Post(
		a.config.RelayURL,
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return fmt.Errorf("failed to query relay: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("relay returned status: %d", resp.StatusCode)
	}

	// Parse response
	var events []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract p tags from Kind 3 event
	var allowedNpubs = make(map[string]bool)

	for _, eventData := range events {
		if eventArray, ok := eventData.([]interface{}); ok && len(eventArray) >= 3 {
			if eventType, ok := eventArray[0].(string); ok && eventType == "EVENT" {
				if event, ok := eventArray[2].(map[string]interface{}); ok {
					if tags, ok := event["tags"].([]interface{}); ok {
						for _, tag := range tags {
							if tagArray, ok := tag.([]interface{}); ok && len(tagArray) >= 2 {
								if tagType, ok := tagArray[0].(string); ok && tagType == "p" {
									if npub, ok := tagArray[1].(string); ok {
										allowedNpubs[npub] = true
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Update allowed npubs
	a.npubMutex.Lock()
	a.allowedNpubs = allowedNpubs
	a.lastUpdate = time.Now()
	a.npubMutex.Unlock()

	log.Printf("Loaded %d allowed npubs from follow list", len(allowedNpubs))
	return nil
}

func (a *Controller) updateLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-a.updateTicker.C:
			if err := a.loadFollowList(); err != nil {
				log.Printf("Failed to update follow list: %v", err)
			}
		}
	}
}

func (a *Controller) GetStats() map[string]interface{} {
	a.npubMutex.RLock()
	defer a.npubMutex.RUnlock()

	return map[string]interface{}{
		"owner_npub":    a.ownerNpub,
		"allowed_count": len(a.allowedNpubs),
		"last_update":   a.lastUpdate,
		"public_read":   a.config.AllowPublicRead,
		"public_write":  a.config.AllowPublicWrite,
	}
}
