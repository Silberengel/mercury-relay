package access

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mercury-relay/internal/config"
	"mercury-relay/internal/models"
	"mercury-relay/test/helpers"
)

func TestWritePermissionCheck(t *testing.T) {
	eg := models.NewEventGenerator()
	ownerNpub := eg.GetOwnerNpub()
	followerNpub := eg.GetFollowerNpub()
	
	t.Run("Owner write access", func(t *testing.T) {
		cfg := config.AccessConfig{
			OwnerNpub:        ownerNpub,
			AllowPublicWrite: false,
			AllowPublicRead:  true,
		}
		controller := NewController(cfg)
		
		// Owner should always be able to write
		canWrite := controller.CanWrite(ownerNpub)
		helpers.AssertBoolEqual(t, true, canWrite)
	})

	t.Run("Follow list member write access", func(t *testing.T) {
		cfg := config.AccessConfig{
			OwnerNpub:        ownerNpub,
			AllowPublicWrite: false,
			AllowPublicRead:  true,
		}
		controller := NewController(cfg)
		
		// Manually add follower to allowed list
		controller.allowedNpubs[followerNpub] = true
		
		canWrite := controller.CanWrite(followerNpub)
		helpers.AssertBoolEqual(t, true, canWrite)
		
		// Non-follower should not be able to write
		otherNpub := "npub1other"
		canWrite = controller.CanWrite(otherNpub)
		helpers.AssertBoolEqual(t, false, canWrite)
	})

	t.Run("Public write enabled", func(t *testing.T) {
		cfg := config.AccessConfig{
			OwnerNpub:        ownerNpub,
			AllowPublicWrite: true,
			AllowPublicRead:  true,
		}
		controller := NewController(cfg)
		
		// Anyone should be able to write when public write is enabled
		canWrite := controller.CanWrite("npub1anyone")
		helpers.AssertBoolEqual(t, true, canWrite)
	})
}

func TestReadPermissionCheck(t *testing.T) {
	eg := models.NewEventGenerator()
	ownerNpub := eg.GetOwnerNpub()
	followerNpub := eg.GetFollowerNpub()
	
	t.Run("Public read enabled", func(t *testing.T) {
		cfg := config.AccessConfig{
			OwnerNpub:        ownerNpub,
			AllowPublicWrite: false,
			AllowPublicRead:  true,
		}
		controller := NewController(cfg)
		
		// Anyone should be able to read when public read is enabled
		canRead := controller.CanRead("npub1anyone")
		helpers.AssertBoolEqual(t, true, canRead)
	})

	t.Run("Restricted read access", func(t *testing.T) {
		cfg := config.AccessConfig{
			OwnerNpub:        ownerNpub,
			AllowPublicWrite: false,
			AllowPublicRead:  false,
		}
		controller := NewController(cfg)
		
		// Owner should always be able to read
		canRead := controller.CanRead(ownerNpub)
		helpers.AssertBoolEqual(t, true, canRead)
		
		// Manually add follower to allowed list
		controller.allowedNpubs[followerNpub] = true
		canRead = controller.CanRead(followerNpub)
		helpers.AssertBoolEqual(t, true, canRead)
		
		// Non-follower should not be able to read
		otherNpub := "npub1other"
		canRead = controller.CanRead(otherNpub)
		helpers.AssertBoolEqual(t, false, canRead)
	})
}

func TestFollowListLoading(t *testing.T) {
	t.Run("Successful follow list fetch", func(t *testing.T) {
		eg := models.NewEventGenerator()
		ownerNpub := eg.GetOwnerNpub()
		followerNpub := eg.GetFollowerNpub()
		
		// Create mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate Nostr relay response with kind 3 event
			response := []interface{}{
				[]interface{}{
					"EVENT",
					"follow-list",
					map[string]interface{}{
						"id":         "follow_event_id",
						"pubkey":     ownerNpub,
						"created_at": 1640995200,
						"kind":       3,
						"tags": []interface{}{
							[]interface{}{"p", followerNpub, "", "follow"},
						},
						"content": "",
						"sig":     "signature",
					},
				},
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()
		
		cfg := config.AccessConfig{
			OwnerNpub:        ownerNpub,
			AllowPublicWrite: false,
			AllowPublicRead:  true,
			RelayURL:         server.URL,
		}
		controller := NewController(cfg)
		
		err := controller.loadFollowList()
		helpers.AssertNoError(t, err)
		
		// Check that follower was added to allowed list
		helpers.AssertBoolEqual(t, true, controller.allowedNpubs[followerNpub])
	})

	t.Run("Relay unavailable", func(t *testing.T) {
		eg := models.NewEventGenerator()
		ownerNpub := eg.GetOwnerNpub()
		
		cfg := config.AccessConfig{
			OwnerNpub:        ownerNpub,
			AllowPublicWrite: false,
			AllowPublicRead:  true,
			RelayURL:         "http://nonexistent-relay.example.com",
		}
		controller := NewController(cfg)
		
		// Set some initial allowed npubs
		controller.allowedNpubs["npub1existing"] = true
		
		err := controller.loadFollowList()
		helpers.AssertError(t, err)
		
		// Existing allowed list should be retained
		helpers.AssertBoolEqual(t, true, controller.allowedNpubs["npub1existing"])
	})

	t.Run("Invalid JSON response", func(t *testing.T) {
		eg := models.NewEventGenerator()
		ownerNpub := eg.GetOwnerNpub()
		
		// Create mock HTTP server returning invalid JSON
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("invalid json"))
		}))
		defer server.Close()
		
		cfg := config.AccessConfig{
			OwnerNpub:        ownerNpub,
			AllowPublicWrite: false,
			AllowPublicRead:  true,
			RelayURL:         server.URL,
		}
		controller := NewController(cfg)
		
		err := controller.loadFollowList()
		helpers.AssertError(t, err)
	})
}

func TestPeriodicUpdate(t *testing.T) {
	t.Run("Follow list auto-update", func(t *testing.T) {
		eg := models.NewEventGenerator()
		ownerNpub := eg.GetOwnerNpub()
		followerNpub := eg.GetFollowerNpub()
		
		updateCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			updateCount++
			response := []interface{}{
				[]interface{}{
					"EVENT",
					"follow-list",
					map[string]interface{}{
						"id":         "follow_event_id",
						"pubkey":     ownerNpub,
						"created_at": 1640995200,
						"kind":       3,
						"tags": []interface{}{
							[]interface{}{"p", followerNpub, "", "follow"},
						},
						"content": "",
						"sig":     "signature",
					},
				},
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()
		
		cfg := config.AccessConfig{
			OwnerNpub:        ownerNpub,
			AllowPublicWrite: false,
			AllowPublicRead:  true,
			RelayURL:         server.URL,
			UpdateInterval:   100 * time.Millisecond, // Fast updates for testing
		}
		controller := NewController(cfg)
		
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		
		err := controller.Start(ctx)
		helpers.AssertNoError(t, err)
		
		// Wait for at least one update
		time.Sleep(200 * time.Millisecond)
		
		// Should have made at least one update call
		if updateCount == 0 {
			t.Errorf("Expected at least one update call, got %d", updateCount)
		}
		
		// Check that follower was added
		helpers.AssertBoolEqual(t, true, controller.allowedNpubs[followerNpub])
	})

	t.Run("Update during context cancellation", func(t *testing.T) {
		eg := models.NewEventGenerator()
		ownerNpub := eg.GetOwnerNpub()
		
		cfg := config.AccessConfig{
			OwnerNpub:        ownerNpub,
			AllowPublicWrite: false,
			AllowPublicRead:  true,
			RelayURL:         "http://example.com",
			UpdateInterval:   100 * time.Millisecond,
		}
		controller := NewController(cfg)
		
		ctx, cancel := context.WithCancel(context.Background())
		
		err := controller.Start(ctx)
		helpers.AssertNoError(t, err)
		
		// Cancel context immediately
		cancel()
		
		// Wait a bit to ensure cleanup
		time.Sleep(50 * time.Millisecond)
		
		// Should not panic or hang
		controller.Stop()
	})
}

func TestAccessControlMethods(t *testing.T) {
	eg := models.NewEventGenerator()
	ownerNpub := eg.GetOwnerNpub()
	followerNpub := eg.GetFollowerNpub()
	
	t.Run("GetAllowedNpubs", func(t *testing.T) {
		cfg := config.AccessConfig{
			OwnerNpub:        ownerNpub,
			AllowPublicWrite: false,
			AllowPublicRead:  true,
		}
		controller := NewController(cfg)
		
		// Add some followers
		controller.allowedNpubs[followerNpub] = true
		controller.allowedNpubs["npub1another"] = true
		
		allowed := controller.GetAllowedNpubs()
		helpers.AssertIntEqual(t, 2, len(allowed))
		helpers.AssertContains(t, allowed, followerNpub)
		helpers.AssertContains(t, allowed, "npub1another")
	})

	t.Run("IsOwner", func(t *testing.T) {
		cfg := config.AccessConfig{
			OwnerNpub:        ownerNpub,
			AllowPublicWrite: false,
			AllowPublicRead:  true,
		}
		controller := NewController(cfg)
		
		helpers.AssertBoolEqual(t, true, controller.IsOwner(ownerNpub))
		helpers.AssertBoolEqual(t, false, controller.IsOwner(followerNpub))
	})

	t.Run("GetStats", func(t *testing.T) {
		cfg := config.AccessConfig{
			OwnerNpub:        ownerNpub,
			AllowPublicWrite: false,
			AllowPublicRead:  true,
		}
		controller := NewController(cfg)
		
		// Add some followers
		controller.allowedNpubs[followerNpub] = true
		
		stats := controller.GetStats()
		
		helpers.AssertStringEqual(t, ownerNpub, stats["owner_npub"].(string))
		helpers.AssertIntEqual(t, 1, stats["allowed_count"].(int))
		helpers.AssertBoolEqual(t, false, stats["public_write"].(bool))
		helpers.AssertBoolEqual(t, true, stats["public_read"].(bool))
	})
}

func TestAccessControlEdgeCases(t *testing.T) {
	eg := models.NewEventGenerator()
	ownerNpub := eg.GetOwnerNpub()
	
	t.Run("Empty follow list", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Return empty event list
			response := []interface{}{}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()
		
		cfg := config.AccessConfig{
			OwnerNpub:        ownerNpub,
			AllowPublicWrite: false,
			AllowPublicRead:  true,
			RelayURL:         server.URL,
		}
		controller := NewController(cfg)
		
		err := controller.loadFollowList()
		helpers.AssertNoError(t, err)
		
		// Allowed list should be empty
		helpers.AssertIntEqual(t, 0, len(controller.allowedNpubs))
	})

	t.Run("Follow list with no p tags", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := []interface{}{
				[]interface{}{
					"EVENT",
					"follow-list",
					map[string]interface{}{
						"id":         "follow_event_id",
						"pubkey":     ownerNpub,
						"created_at": 1640995200,
						"kind":       3,
						"tags": []interface{}{
							[]interface{}{"t", "follows"},
						},
						"content": "",
						"sig":     "signature",
					},
				},
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()
		
		cfg := config.AccessConfig{
			OwnerNpub:        ownerNpub,
			AllowPublicWrite: false,
			AllowPublicRead:  true,
			RelayURL:         server.URL,
		}
		controller := NewController(cfg)
		
		err := controller.loadFollowList()
		helpers.AssertNoError(t, err)
		
		// No p tags, so no followers added
		helpers.AssertIntEqual(t, 0, len(controller.allowedNpubs))
	})

	t.Run("HTTP server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()
		
		cfg := config.AccessConfig{
			OwnerNpub:        ownerNpub,
			AllowPublicWrite: false,
			AllowPublicRead:  true,
			RelayURL:         server.URL,
		}
		controller := NewController(cfg)
		
		err := controller.loadFollowList()
		helpers.AssertError(t, err)
	})
}

// Test integration scenarios
func TestAccessControlIntegration(t *testing.T) {
	eg := models.NewEventGenerator()
	ownerNpub := eg.GetOwnerNpub()
	followerNpub := eg.GetFollowerNpub()
	
	t.Run("Dynamic follow list updates", func(t *testing.T) {
		updateCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			updateCount++
			
			// First update: follower is included
			// Second update: follower is removed
			tags := []interface{}{}
			if updateCount == 1 {
				tags = []interface{}{
					[]interface{}{"p", followerNpub, "", "follow"},
				}
			}
			
			response := []interface{}{
				[]interface{}{
					"EVENT",
					"follow-list",
					map[string]interface{}{
						"id":         "follow_event_id",
						"pubkey":     ownerNpub,
						"created_at": 1640995200,
						"kind":       3,
						"tags":       tags,
						"content":    "",
						"sig":        "signature",
					},
				},
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()
		
		cfg := config.AccessConfig{
			OwnerNpub:        ownerNpub,
			AllowPublicWrite: false,
			AllowPublicRead:  true,
			RelayURL:         server.URL,
			UpdateInterval:   100 * time.Millisecond,
		}
		controller := NewController(cfg)
		
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		
		err := controller.Start(ctx)
		helpers.AssertNoError(t, err)
		
		// Wait for first update
		time.Sleep(150 * time.Millisecond)
		
		// Follower should be allowed
		helpers.AssertBoolEqual(t, true, controller.CanWrite(followerNpub))
		
		// Wait for second update
		time.Sleep(150 * time.Millisecond)
		
		// Follower should no longer be allowed
		helpers.AssertBoolEqual(t, false, controller.CanWrite(followerNpub))
	})

	t.Run("Owner always has access", func(t *testing.T) {
		cfg := config.AccessConfig{
			OwnerNpub:        ownerNpub,
			AllowPublicWrite: false,
			AllowPublicRead:  false,
		}
		controller := NewController(cfg)
		
		// Even with empty follow list and no public access
		helpers.AssertBoolEqual(t, true, controller.CanWrite(ownerNpub))
		helpers.AssertBoolEqual(t, true, controller.CanRead(ownerNpub))
	})
}
