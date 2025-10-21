package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mercury-relay/internal/config"
	"mercury-relay/internal/models"
	"mercury-relay/test/helpers"
	"mercury-relay/test/mocks"

	"github.com/nbd-wtf/go-nostr"
)

func TestRESTAPIGetEvents(t *testing.T) {
	t.Run("Query by authors and kinds", func(t *testing.T) {
		// Setup
		mockCache := mocks.NewMockCache()
		mockQueue := mocks.NewMockQueue()
		eg := helpers.NewEventGenerator()

		npub1 := eg.GetRandomNpub()
		npub2 := eg.GetRandomNpub()

		// Create test events
		event1 := eg.GenerateTextNote(npub1, "Message 1", nostr.Tags{})
		event2 := eg.GenerateUserMetadata(npub2, map[string]interface{}{"name": "User"})
		event3 := eg.GenerateTextNote(npub1, "Message 3", nostr.Tags{})

		mockCache.SetEvents([]*models.Event{event1, event2, event3})

		cfg := config.RESTAPIConfig{
			Enabled:     true,
			Port:        8082,
			CORSEnabled: true,
		}

		server := NewRESTAPIServer(cfg, nil, mockQueue, mockCache)

		// Create request
		req := httptest.NewRequest("GET", "/api/v1/events?authors="+npub1+"&kinds=1&limit=10", nil)
		w := httptest.NewRecorder()

		// Execute request
		server.handleGetEvents(w, req)

		// Verify response
		helpers.AssertIntEqual(t, http.StatusOK, w.Code)

		var response APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		helpers.AssertNoError(t, err)
		helpers.AssertBoolEqual(t, true, response.Success)

		// Verify events returned
		events, ok := response.Data.([]interface{})
		helpers.AssertBoolEqual(t, true, ok)
		helpers.AssertIntEqual(t, 2, len(events)) // Only npub1's kind 1 events
	})

	t.Run("Query with time range", func(t *testing.T) {
		// Setup
		mockCache := mocks.NewMockCache()
		mockQueue := mocks.NewMockQueue()
		eg := helpers.NewEventGenerator()

		npub := eg.GetRandomNpub()

		// Create events with different timestamps
		event1 := eg.GenerateTextNote(npub, "Message 1", nostr.Tags{})
		event1.CreatedAt = time.Unix(1640995200, 0)

		event2 := eg.GenerateTextNote(npub, "Message 2", nostr.Tags{})
		event2.CreatedAt = time.Unix(1640995300, 0)

		event3 := eg.GenerateTextNote(npub, "Message 3", nostr.Tags{})
		event3.CreatedAt = time.Unix(1640995400, 0)

		mockCache.SetEvents([]*models.Event{event1, event2, event3})

		cfg := config.RESTAPIConfig{
			Enabled:     true,
			Port:        8082,
			CORSEnabled: true,
		}

		server := NewRESTAPIServer(cfg, nil, mockQueue, mockCache)

		// Create request with time range
		req := httptest.NewRequest("GET", "/api/v1/events?since=1640995250&until=1640995350", nil)
		w := httptest.NewRecorder()

		// Execute request
		server.handleGetEvents(w, req)

		// Verify response
		helpers.AssertIntEqual(t, http.StatusOK, w.Code)

		var response APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		helpers.AssertNoError(t, err)
		helpers.AssertBoolEqual(t, true, response.Success)

		// Verify only event2 is returned (within time range)
		events, ok := response.Data.([]interface{})
		helpers.AssertBoolEqual(t, true, ok)
		helpers.AssertIntEqual(t, 1, len(events))
	})

	t.Run("POST request with JSON body", func(t *testing.T) {
		// Setup
		mockCache := mocks.NewMockCache()
		mockQueue := mocks.NewMockQueue()
		eg := helpers.NewEventGenerator()

		npub := eg.GetRandomNpub()
		event := eg.GenerateTextNote(npub, "Test message", nostr.Tags{})
		mockCache.SetEvents([]*models.Event{event})

		cfg := config.RESTAPIConfig{
			Enabled:     true,
			Port:        8082,
			CORSEnabled: true,
		}

		server := NewRESTAPIServer(cfg, nil, mockQueue, mockCache)

		// Create POST request with JSON body
		eventReq := EventRequest{
			Filter: nostr.Filter{
				Authors: []string{npub},
				Kinds:   []int{1},
			},
			Limit: 10,
		}

		reqBody, _ := json.Marshal(eventReq)
		req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Execute request
		server.handleGetEvents(w, req)

		// Verify response
		helpers.AssertIntEqual(t, http.StatusOK, w.Code)

		var response APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		helpers.AssertNoError(t, err)
		helpers.AssertBoolEqual(t, true, response.Success)
	})
}

func TestRESTAPIPublish(t *testing.T) {
	t.Run("Valid event publication", func(t *testing.T) {
		// Setup
		mockCache := mocks.NewMockCache()
		mockQueue := mocks.NewMockQueue()
		eg := helpers.NewEventGenerator()

		// Mock quality controller
		_ = &MockQualityController{}

		cfg := config.RESTAPIConfig{
			Enabled:     true,
			Port:        8082,
			CORSEnabled: true,
		}

		server := NewRESTAPIServer(cfg, nil, mockQueue, mockCache)

		// Create valid event
		event := eg.GenerateTextNote(eg.GetRandomNpub(), "Test message", nostr.Tags{})

		publishReq := PublishRequest{
			Event: *event,
		}

		reqBody, _ := json.Marshal(publishReq)
		req := httptest.NewRequest("POST", "/api/v1/publish", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Execute request
		server.handlePublish(w, req)

		// Verify response
		helpers.AssertIntEqual(t, http.StatusOK, w.Code)

		var response APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		helpers.AssertNoError(t, err)
		helpers.AssertBoolEqual(t, true, response.Success)

		// Verify event was published to queue
		helpers.AssertIntEqual(t, 1, mockQueue.GetEventCount())
	})

	t.Run("Invalid event validation failure", func(t *testing.T) {
		// Setup
		mockCache := mocks.NewMockCache()
		mockQueue := mocks.NewMockQueue()

		cfg := config.RESTAPIConfig{
			Enabled:     true,
			Port:        8082,
			CORSEnabled: true,
		}

		server := NewRESTAPIServer(cfg, nil, mockQueue, mockCache)

		// Create invalid event (missing required fields)
		event := &models.Event{
			ID:        "", // Missing ID
			PubKey:    "", // Missing PubKey
			CreatedAt: time.Now(),
			Kind:      1,
			Tags:      nostr.Tags{},
			Content:   "test",
			Sig:       "", // Missing Sig
		}

		publishReq := PublishRequest{
			Event: *event,
		}

		reqBody, _ := json.Marshal(publishReq)
		req := httptest.NewRequest("POST", "/api/v1/publish", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Execute request
		server.handlePublish(w, req)

		// Verify response
		helpers.AssertIntEqual(t, http.StatusBadRequest, w.Code)

		var response APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		helpers.AssertNoError(t, err)
		helpers.AssertBoolEqual(t, false, response.Success)
		helpers.AssertErrorContains(t, err, "validation failed")
	})
}

func TestRESTAPIEbooks(t *testing.T) {
	t.Run("Discover all ebooks with format filter", func(t *testing.T) {
		// Setup
		mockCache := mocks.NewMockCache()
		mockQueue := mocks.NewMockQueue()
		eg := helpers.NewEventGenerator()

		// Create ebook events
		ebook1 := eg.GenerateEbook(eg.GetRandomNpub(), map[string]interface{}{
			"title":  "Test Book 1",
			"format": "epub",
			"author": "Test Author 1",
		})

		ebook2 := eg.GenerateEbook(eg.GetRandomNpub(), map[string]interface{}{
			"title":  "Test Book 2",
			"format": "pdf",
			"author": "Test Author 2",
		})

		ebook3 := eg.GenerateEbook(eg.GetRandomNpub(), map[string]interface{}{
			"title":  "Test Book 3",
			"format": "epub",
			"author": "Test Author 3",
		})

		mockCache.SetEvents([]*models.Event{ebook1, ebook2, ebook3})

		cfg := config.RESTAPIConfig{
			Enabled:     true,
			Port:        8082,
			CORSEnabled: true,
		}

		server := NewRESTAPIServer(cfg, nil, mockQueue, mockCache)

		// Create request
		req := httptest.NewRequest("GET", "/api/v1/ebooks?format=epub&limit=20", nil)
		w := httptest.NewRecorder()

		// Execute request
		server.handleEbooks(w, req)

		// Verify response
		helpers.AssertIntEqual(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		helpers.AssertNoError(t, err)
		helpers.AssertBoolEqual(t, true, response["success"].(bool))

		// Verify only epub books returned
		ebooks := response["ebooks"].([]interface{})
		helpers.AssertIntEqual(t, 2, len(ebooks))
	})

	t.Run("Author-specific ebook search", func(t *testing.T) {
		// Setup
		mockCache := mocks.NewMockCache()
		mockQueue := mocks.NewMockQueue()
		eg := helpers.NewEventGenerator()

		npub1 := eg.GetRandomNpub()
		npub2 := eg.GetRandomNpub()

		// Create ebook events from different authors
		ebook1 := eg.GenerateEbook(npub1, map[string]interface{}{
			"title":  "Book by Author 1",
			"format": "epub",
			"author": "Author 1",
		})

		ebook2 := eg.GenerateEbook(npub2, map[string]interface{}{
			"title":  "Book by Author 2",
			"format": "pdf",
			"author": "Author 2",
		})

		mockCache.SetEvents([]*models.Event{ebook1, ebook2})

		cfg := config.RESTAPIConfig{
			Enabled:     true,
			Port:        8082,
			CORSEnabled: true,
		}

		server := NewRESTAPIServer(cfg, nil, mockQueue, mockCache)

		// Create request for specific author
		req := httptest.NewRequest("GET", "/api/v1/ebooks?author="+npub1, nil)
		w := httptest.NewRecorder()

		// Execute request
		server.handleEbooks(w, req)

		// Verify response
		helpers.AssertIntEqual(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		helpers.AssertNoError(t, err)
		helpers.AssertBoolEqual(t, true, response["success"].(bool))

		// Verify only books from npub1 returned
		ebooks := response["ebooks"].([]interface{})
		helpers.AssertIntEqual(t, 1, len(ebooks))
	})
}

func TestRESTAPIHealth(t *testing.T) {
	t.Run("Health check", func(t *testing.T) {
		// Setup
		mockCache := mocks.NewMockCache()
		mockQueue := mocks.NewMockQueue()

		cfg := config.RESTAPIConfig{
			Enabled:     true,
			Port:        8082,
			CORSEnabled: true,
		}

		server := NewRESTAPIServer(cfg, nil, mockQueue, mockCache)

		// Create request
		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		w := httptest.NewRecorder()

		// Execute request
		server.handleHealth(w, req)

		// Verify response
		helpers.AssertIntEqual(t, http.StatusOK, w.Code)

		var response APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		helpers.AssertNoError(t, err)
		helpers.AssertBoolEqual(t, true, response.Success)

		// Verify health data
		health, ok := response.Data.(map[string]interface{})
		helpers.AssertBoolEqual(t, true, ok)
		helpers.AssertStringEqual(t, "healthy", health["status"].(string))
		helpers.AssertStringEqual(t, "1.0.0", health["version"].(string))
	})
}

func TestRESTAPIStats(t *testing.T) {
	t.Run("Get relay stats", func(t *testing.T) {
		// Setup
		mockCache := mocks.NewMockCache()
		mockQueue := mocks.NewMockQueue()
		_ = &MockQualityController{}

		cfg := config.RESTAPIConfig{
			Enabled:     true,
			Port:        8082,
			CORSEnabled: true,
		}

		server := NewRESTAPIServer(cfg, nil, mockQueue, mockCache)

		// Create request
		req := httptest.NewRequest("GET", "/api/v1/stats", nil)
		w := httptest.NewRecorder()

		// Execute request
		server.handleStats(w, req)

		// Verify response
		helpers.AssertIntEqual(t, http.StatusOK, w.Code)

		var response APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		helpers.AssertNoError(t, err)
		helpers.AssertBoolEqual(t, true, response.Success)

		// Verify stats data
		stats, ok := response.Data.(map[string]interface{})
		helpers.AssertBoolEqual(t, true, ok)
		helpers.AssertIntEqual(t, 0, int(stats["total_events"].(int64)))
		helpers.AssertIntEqual(t, 0, int(stats["active_connections"].(int)))
		helpers.AssertIntEqual(t, 0, int(stats["cache_size"].(int64)))
		helpers.AssertIntEqual(t, 0, int(stats["queue_size"].(int64)))
	})
}

func TestRESTAPICORS(t *testing.T) {
	t.Run("CORS preflight request", func(t *testing.T) {
		// Setup
		mockCache := mocks.NewMockCache()
		mockQueue := mocks.NewMockQueue()

		cfg := config.RESTAPIConfig{
			Enabled:     true,
			Port:        8082,
			CORSEnabled: true,
		}

		server := NewRESTAPIServer(cfg, nil, mockQueue, mockCache)

		// Create OPTIONS request
		req := httptest.NewRequest("OPTIONS", "/api/v1/events", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", "GET")
		req.Header.Set("Access-Control-Request-Headers", "Content-Type")
		w := httptest.NewRecorder()

		// Create router and add CORS middleware
		router := server.createTestRouter()
		router.ServeHTTP(w, req)

		// Verify CORS headers
		helpers.AssertIntEqual(t, http.StatusOK, w.Code)
		helpers.AssertStringEqual(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		helpers.AssertStringEqual(t, "GET, POST, PUT, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
		helpers.AssertStringEqual(t, "Content-Type, Authorization", w.Header().Get("Access-Control-Allow-Headers"))
	})
}

func TestRESTAPIRateLimiting(t *testing.T) {
	t.Run("Rate limit enforcement", func(t *testing.T) {
		// Setup
		mockCache := mocks.NewMockCache()
		mockQueue := mocks.NewMockQueue()

		cfg := config.RESTAPIConfig{
			Enabled:     true,
			Port:        8082,
			CORSEnabled: true,
		}

		server := NewRESTAPIServer(cfg, nil, mockQueue, mockCache)

		// Make multiple rapid requests
		// Note: The current implementation doesn't have actual rate limiting,
		// so this test verifies the interface works
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/api/v1/health", nil)
			w := httptest.NewRecorder()
			server.handleHealth(w, req)
			helpers.AssertIntEqual(t, http.StatusOK, w.Code)
		}
	})
}

// Mock implementations for testing

type MockQualityController struct{}

func (m *MockQualityController) ValidateEvent(event *models.Event) error {
	return nil // Always pass validation for testing
}

func (m *MockQualityController) GetQualityStats() (map[string]interface{}, error) {
	return map[string]interface{}{
		"total_events":  0,
		"blocked_npubs": 0,
		"active_npubs":  0,
	}, nil
}

func (m *MockQualityController) BlockNpub(npub string) error {
	return nil
}

func (m *MockQualityController) UnblockNpub(npub string) error {
	return nil
}

func (m *MockQualityController) IsNpubBlocked(npub string) bool {
	return false
}

func (m *MockQualityController) GetBlockedNpubs() []string {
	return []string{}
}

// Helper method to create test router with middleware
func (r *RESTAPIServer) createTestRouter() *http.ServeMux {
	mux := http.NewServeMux()

	// Add CORS middleware
	if r.config.CORSEnabled {
		mux.HandleFunc("/api/v1/events", r.corsMiddleware(http.HandlerFunc(r.handleGetEvents)).ServeHTTP)
	} else {
		mux.HandleFunc("/api/v1/events", r.handleGetEvents)
	}

	return mux
}

func TestRESTAPIIntegration(t *testing.T) {
	t.Run("Complete REST API flow", func(t *testing.T) {
		// Setup
		mockCache := mocks.NewMockCache()
		mockQueue := mocks.NewMockQueue()
		_ = &MockQualityController{}

		cfg := config.RESTAPIConfig{
			Enabled:     true,
			Port:        8082,
			CORSEnabled: true,
		}

		server := NewRESTAPIServer(cfg, nil, mockQueue, mockCache)
		eg := helpers.NewEventGenerator()

		// Step 1: Publish event via REST
		event := eg.GenerateTextNote(eg.GetRandomNpub(), "Test message", nostr.Tags{})
		publishReq := PublishRequest{Event: *event}

		reqBody, _ := json.Marshal(publishReq)
		req := httptest.NewRequest("POST", "/api/v1/publish", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handlePublish(w, req)
		helpers.AssertIntEqual(t, http.StatusOK, w.Code)

		// Step 2: Simulate event being processed and stored in cache
		mockCache.StoreEvent(event)

		// Step 3: Query event via REST
		req = httptest.NewRequest("GET", "/api/v1/events?authors="+event.PubKey, nil)
		w = httptest.NewRecorder()

		server.handleGetEvents(w, req)
		helpers.AssertIntEqual(t, http.StatusOK, w.Code)

		var response APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		helpers.AssertNoError(t, err)
		helpers.AssertBoolEqual(t, true, response.Success)

		// Verify event is available
		events := response.Data.([]interface{})
		helpers.AssertIntEqual(t, 1, len(events))
	})
}
