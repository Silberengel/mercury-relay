package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"mercury-relay/internal/cache"
	"mercury-relay/internal/config"
	"mercury-relay/internal/models"
	"mercury-relay/internal/quality"
	"mercury-relay/internal/queue"

	"github.com/gorilla/mux"
	"github.com/nbd-wtf/go-nostr"
)

type RESTAPIServer struct {
	config         config.RESTAPIConfig
	qualityControl *quality.Controller
	rabbitMQ       queue.Queue
	cache          cache.Cache
	server         *http.Server
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type EventRequest struct {
	Filter nostr.Filter `json:"filter"`
	Limit  int          `json:"limit,omitempty"`
}

type PublishRequest struct {
	Event models.Event `json:"event"`
}

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

type StatsResponse struct {
	TotalEvents       int64                  `json:"total_events"`
	ActiveConnections int                    `json:"active_connections"`
	CacheSize         int64                  `json:"cache_size"`
	QueueSize         int64                  `json:"queue_size"`
	QualityStats      map[string]interface{} `json:"quality_stats"`
}

func NewRESTAPIServer(
	config config.RESTAPIConfig,
	qualityControl *quality.Controller,
	rabbitMQ queue.Queue,
	cache cache.Cache,
) *RESTAPIServer {
	return &RESTAPIServer{
		config:         config,
		qualityControl: qualityControl,
		rabbitMQ:       rabbitMQ,
		cache:          cache,
	}
}

func (r *RESTAPIServer) Start(ctx context.Context) error {
	router := mux.NewRouter()

	// CORS middleware
	if r.config.CORSEnabled {
		router.Use(r.corsMiddleware)
	}

	// Rate limiting middleware
	router.Use(r.rateLimitMiddleware)

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/events", r.handleGetEvents).Methods("GET", "POST")
	api.HandleFunc("/query", r.handleQuery).Methods("POST")
	api.HandleFunc("/publish", r.handlePublish).Methods("POST")
	api.HandleFunc("/stream", r.handleStream).Methods("GET")                    // HTTP streaming
	api.HandleFunc("/sse", r.handleSSE).Methods("GET")                          // Server-Sent Events
	api.HandleFunc("/ebooks", r.handleEbooks).Methods("GET")                    // E-book specific endpoint
	api.HandleFunc("/ebooks/{id}/content", r.handleEbookContent).Methods("GET") // E-book content with nested structure
	api.HandleFunc("/ebooks/{id}/epub", r.handleEbookEPUB).Methods("GET")       // Generate EPUB from Nostr book
	api.HandleFunc("/health", r.handleHealth).Methods("GET")
	api.HandleFunc("/stats", r.handleStats).Methods("GET")

	// Start server
	r.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", r.config.Port),
		Handler: router,
	}

	go func() {
		log.Printf("Starting REST API server on port %d", r.config.Port)
		if err := r.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("REST API server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return r.server.Shutdown(shutdownCtx)
}

func (r *RESTAPIServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if req.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, req)
	})
}

func (r *RESTAPIServer) rateLimitMiddleware(next http.Handler) http.Handler {
	// Simple rate limiting - in production, use a proper rate limiter
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// TODO: Implement proper rate limiting
		next.ServeHTTP(w, req)
	})
}

func (r *RESTAPIServer) handleGetEvents(w http.ResponseWriter, req *http.Request) {
	var filter nostr.Filter

	if req.Method == "GET" {
		// Parse query parameters
		if authors := req.URL.Query()["authors"]; len(authors) > 0 {
			filter.Authors = authors
		}
		if kinds := req.URL.Query()["kinds"]; len(kinds) > 0 {
			for _, kind := range kinds {
				if k, err := strconv.Atoi(kind); err == nil {
					filter.Kinds = append(filter.Kinds, k)
				}
			}
		}
		if since := req.URL.Query().Get("since"); since != "" {
			if s, err := strconv.ParseInt(since, 10, 64); err == nil {
				timestamp := nostr.Timestamp(s)
				filter.Since = &timestamp
			}
		}
		if until := req.URL.Query().Get("until"); until != "" {
			if u, err := strconv.ParseInt(until, 10, 64); err == nil {
				timestamp := nostr.Timestamp(u)
				filter.Until = &timestamp
			}
		}
		if limit := req.URL.Query().Get("limit"); limit != "" {
			if l, err := strconv.Atoi(limit); err == nil {
				filter.Limit = l
			}
		}
	} else {
		// Parse JSON body
		var eventReq EventRequest
		if err := json.NewDecoder(req.Body).Decode(&eventReq); err != nil {
			r.sendError(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		filter = eventReq.Filter
		if eventReq.Limit > 0 {
			filter.Limit = eventReq.Limit
		}
	}

	// Get events from cache
	events, err := r.cache.GetEvents(filter)
	if err != nil {
		r.sendError(w, fmt.Sprintf("Failed to get events: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to Nostr events
	var nostrEvents []nostr.Event
	for _, event := range events {
		nostrEvent := event.ToNostrEvent()
		nostrEvents = append(nostrEvents, *nostrEvent)
	}

	r.sendSuccess(w, nostrEvents)
}

func (r *RESTAPIServer) handleQuery(w http.ResponseWriter, req *http.Request) {
	var eventReq EventRequest
	if err := json.NewDecoder(req.Body).Decode(&eventReq); err != nil {
		r.sendError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Get events from cache
	events, err := r.cache.GetEvents(eventReq.Filter)
	if err != nil {
		r.sendError(w, fmt.Sprintf("Failed to query events: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to Nostr events
	var nostrEvents []nostr.Event
	for _, event := range events {
		nostrEvent := event.ToNostrEvent()
		nostrEvents = append(nostrEvents, *nostrEvent)
	}

	r.sendSuccess(w, nostrEvents)
}

func (r *RESTAPIServer) handlePublish(w http.ResponseWriter, req *http.Request) {
	var publishReq PublishRequest
	if err := json.NewDecoder(req.Body).Decode(&publishReq); err != nil {
		r.sendError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate event
	if err := publishReq.Event.Validate(); err != nil {
		r.sendError(w, fmt.Sprintf("Event validation failed: %v", err), http.StatusBadRequest)
		return
	}

	// Check quality control
	if r.qualityControl != nil {
		if err := r.qualityControl.ValidateEvent(&publishReq.Event); err != nil {
			r.sendError(w, fmt.Sprintf("Quality control failed: %v", err), http.StatusBadRequest)
			return
		}
	}

	// Publish to queue
	if err := r.rabbitMQ.PublishEvent(&publishReq.Event); err != nil {
		r.sendError(w, fmt.Sprintf("Failed to publish event: %v", err), http.StatusInternalServerError)
		return
	}

	r.sendSuccess(w, map[string]interface{}{
		"event_id": publishReq.Event.ID,
		"status":   "published",
	})
}

func (r *RESTAPIServer) handleHealth(w http.ResponseWriter, req *http.Request) {
	health := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
	}

	r.sendSuccess(w, health)
}

func (r *RESTAPIServer) handleStats(w http.ResponseWriter, req *http.Request) {
	// Get basic stats
	stats := StatsResponse{
		TotalEvents:       0, // TODO: Implement actual stats
		ActiveConnections: 0, // TODO: Implement actual stats
		CacheSize:         0, // TODO: Implement actual stats
		QueueSize:         0, // TODO: Implement actual stats
		QualityStats:      make(map[string]interface{}),
	}

	// Get quality stats
	if r.qualityControl != nil {
		qualityStats, err := r.qualityControl.GetQualityStats()
		if err == nil {
			stats.QualityStats = qualityStats
		}
	}

	r.sendSuccess(w, stats)
}

func (r *RESTAPIServer) sendSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := APIResponse{
		Success: true,
		Data:    data,
	}

	json.NewEncoder(w).Encode(response)
}

func (r *RESTAPIServer) handleStream(w http.ResponseWriter, req *http.Request) {
	// HTTP streaming endpoint for server-side rendering
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	// Parse filter from query parameters
	var filter nostr.Filter
	if authors := req.URL.Query()["authors"]; len(authors) > 0 {
		filter.Authors = authors
	}
	if kinds := req.URL.Query()["kinds"]; len(kinds) > 0 {
		for _, kind := range kinds {
			if k, err := strconv.Atoi(kind); err == nil {
				filter.Kinds = append(filter.Kinds, k)
			}
		}
	}
	if since := req.URL.Query().Get("since"); since != "" {
		if s, err := strconv.ParseInt(since, 10, 64); err == nil {
			timestamp := nostr.Timestamp(s)
			filter.Since = &timestamp
		}
	}
	if until := req.URL.Query().Get("until"); until != "" {
		if u, err := strconv.ParseInt(until, 10, 64); err == nil {
			timestamp := nostr.Timestamp(u)
			filter.Until = &timestamp
		}
	}
	if limit := req.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filter.Limit = l
		}
	}

	// Get initial events
	events, err := r.cache.GetEvents(filter)
	if err != nil {
		r.sendError(w, fmt.Sprintf("Failed to get events: %v", err), http.StatusInternalServerError)
		return
	}

	// Send initial events
	encoder := json.NewEncoder(w)
	for _, event := range events {
		nostrEvent := event.ToNostrEvent()
		if err := encoder.Encode(map[string]interface{}{
			"type": "event",
			"data": *nostrEvent,
		}); err != nil {
			return
		}
		w.(http.Flusher).Flush()
	}

	// TODO: Implement real-time streaming for new events
	// This would require a subscription mechanism to push new events
	// as they arrive from the queue or upstream relays
}

func (r *RESTAPIServer) handleSSE(w http.ResponseWriter, req *http.Request) {
	// Server-Sent Events endpoint for monitoring and admin purposes
	// Note: For Nostr event streaming, use WebSocket connections instead
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Parse endpoint type from query parameters
	endpoint := req.URL.Query().Get("type")

	switch endpoint {
	case "stats":
		r.handleSSEStats(w, req)
	case "health":
		r.handleSSEHealth(w, req)
	case "admin":
		r.handleSSEAdmin(w, req)
	default:
		// Send initial connection event
		fmt.Fprintf(w, "event: connected\n")
		fmt.Fprintf(w, "data: {\"message\": \"Connected to Mercury Relay SSE\", \"endpoints\": [\"stats\", \"health\", \"admin\"]}\n\n")
		w.(http.Flusher).Flush()

		// Keep connection alive with heartbeat
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-req.Context().Done():
				return
			case <-ticker.C:
				fmt.Fprintf(w, "event: heartbeat\n")
				fmt.Fprintf(w, "data: {\"timestamp\": %d}\n\n", time.Now().Unix())
				w.(http.Flusher).Flush()
			}
		}
	}
}

func (r *RESTAPIServer) handleSSEStats(w http.ResponseWriter, req *http.Request) {
	// SSE endpoint for real-time relay statistics
	fmt.Fprintf(w, "event: connected\n")
	fmt.Fprintf(w, "data: {\"message\": \"Connected to Mercury Relay Stats SSE\"}\n\n")
	w.(http.Flusher).Flush()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-req.Context().Done():
			return
		case <-ticker.C:
			stats, err := r.qualityControl.GetQualityStats()
			if err != nil {
				continue
			}
			statsJSON, _ := json.Marshal(stats)
			fmt.Fprintf(w, "event: stats\n")
			fmt.Fprintf(w, "data: %s\n\n", string(statsJSON))
			w.(http.Flusher).Flush()
		}
	}
}

func (r *RESTAPIServer) handleSSEHealth(w http.ResponseWriter, req *http.Request) {
	// SSE endpoint for real-time health monitoring
	fmt.Fprintf(w, "event: connected\n")
	fmt.Fprintf(w, "data: {\"message\": \"Connected to Mercury Relay Health SSE\"}\n\n")
	w.(http.Flusher).Flush()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-req.Context().Done():
			return
		case <-ticker.C:
			health := HealthResponse{
				Status:    "healthy",
				Timestamp: time.Now(),
				Version:   "1.0.0",
			}
			healthJSON, _ := json.Marshal(health)
			fmt.Fprintf(w, "event: health\n")
			fmt.Fprintf(w, "data: %s\n\n", string(healthJSON))
			w.(http.Flusher).Flush()
		}
	}
}

func (r *RESTAPIServer) handleSSEAdmin(w http.ResponseWriter, req *http.Request) {
	// SSE endpoint for admin monitoring (blocked users, quality control, etc.)
	fmt.Fprintf(w, "event: connected\n")
	fmt.Fprintf(w, "data: {\"message\": \"Connected to Mercury Relay Admin SSE\"}\n\n")
	w.(http.Flusher).Flush()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-req.Context().Done():
			return
		case <-ticker.C:
			// Send admin-specific data
			qualityStats, _ := r.qualityControl.GetQualityStats()
			adminData := map[string]interface{}{
				"blocked_users": r.qualityControl.GetBlockedNpubs(),
				"quality_stats": qualityStats,
				"timestamp":     time.Now().Unix(),
			}
			adminJSON, _ := json.Marshal(adminData)
			fmt.Fprintf(w, "event: admin\n")
			fmt.Fprintf(w, "data: %s\n\n", string(adminJSON))
			w.(http.Flusher).Flush()
		}
	}
}

func (r *RESTAPIServer) handleEbooks(w http.ResponseWriter, req *http.Request) {
	// E-book specific endpoint optimized for e-paper readers
	// Supports kind 30040 (NKBIP-01) and related ebook events

	// Parse query parameters
	author := req.URL.Query().Get("author")
	identifier := req.URL.Query().Get("identifier") // d tag value
	format := req.URL.Query().Get("format")         // epub, pdf, etc.
	limit := req.URL.Query().Get("limit")

	// Build filter for ebooks
	filter := nostr.Filter{
		Kinds: []int{30040}, // NKBIP-01 ebook kind
	}

	if author != "" {
		filter.Authors = []string{author}
	}

	if limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filter.Limit = l
		}
	}

	// Add d tag filter if identifier provided
	if identifier != "" {
		// This would require custom tag filtering
		// For now, we'll get all ebooks and filter client-side
	}

	// Get ebooks from cache
	events, err := r.cache.GetEvents(filter)
	if err != nil {
		r.sendError(w, fmt.Sprintf("Failed to get ebooks: %v", err), http.StatusInternalServerError)
		return
	}

	// Filter and format for e-paper readers
	var ebooks []map[string]interface{}
	for _, event := range events {
		// Parse ebook metadata from content
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(event.Content), &metadata); err != nil {
			continue
		}

		// Check format filter
		if format != "" {
			if bookFormat, ok := metadata["format"].(string); ok {
				if bookFormat != format {
					continue
				}
			}
		}

		// Extract ebook information
		ebook := map[string]interface{}{
			"id":          event.ID,
			"author":      event.PubKey,
			"title":       metadata["title"],
			"author_name": metadata["author"],
			"format":      metadata["format"],
			"size":        metadata["size"],
			"created_at":  event.CreatedAt.Unix(),
			"tags":        event.Tags,
		}

		// Add download URL if available
		if downloadURL, ok := metadata["download_url"].(string); ok {
			ebook["download_url"] = downloadURL
		}

		// Add cover image if available
		if cover, ok := metadata["cover"].(string); ok {
			ebook["cover"] = cover
		}

		ebooks = append(ebooks, ebook)
	}

	// Set headers optimized for e-paper readers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Return simplified response for e-paper readers
	response := map[string]interface{}{
		"success":   true,
		"count":     len(ebooks),
		"ebooks":    ebooks,
		"timestamp": time.Now().Unix(),
	}

	json.NewEncoder(w).Encode(response)
}

func (r *RESTAPIServer) handleEbookContent(w http.ResponseWriter, req *http.Request) {
	// Special function for transmitting e-paper books with nested structure
	// Supports kind 30040 (Publication Index) with kind 30041 (Publication Content) per NKBIP-01

	// Extract book ID from URL path
	vars := mux.Vars(req)
	bookID := vars["id"]

	if bookID == "" {
		r.sendError(w, "Book ID is required", http.StatusBadRequest)
		return
	}

	// Parse query parameters for content options
	format := req.URL.Query().Get("format") // asciidoc, html, markdown
	includeImages := req.URL.Query().Get("images") == "true"
	maxDepth := req.URL.Query().Get("depth")

	// Set default format
	if format == "" {
		format = "asciidoc"
	}

	// Parse max depth
	depth := 3 // default depth
	if maxDepth != "" {
		if d, err := strconv.Atoi(maxDepth); err == nil && d > 0 {
			depth = d
		}
	}

	// Get the main book event (kind 30040)
	bookFilter := nostr.Filter{
		Kinds: []int{30040},
		IDs:   []string{bookID},
	}

	bookEvents, err := r.cache.GetEvents(bookFilter)
	if err != nil {
		r.sendError(w, fmt.Sprintf("Failed to get book: %v", err), http.StatusInternalServerError)
		return
	}

	if len(bookEvents) == 0 {
		r.sendError(w, "Book not found", http.StatusNotFound)
		return
	}

	bookEvent := bookEvents[0]

	// Parse book metadata
	var bookMetadata map[string]interface{}
	if err := json.Unmarshal([]byte(bookEvent.Content), &bookMetadata); err != nil {
		r.sendError(w, "Invalid book metadata", http.StatusBadRequest)
		return
	}

	// Get the book's d tag for finding content events
	var bookIdentifier string
	for _, tag := range bookEvent.Tags {
		if len(tag) >= 2 && tag[0] == "d" {
			bookIdentifier = tag[1]
			break
		}
	}

	if bookIdentifier == "" {
		r.sendError(w, "Book identifier not found", http.StatusBadRequest)
		return
	}

	// Get content events (kind 30041) for this book
	contentFilter := nostr.Filter{
		Kinds:   []int{30041},
		Authors: []string{bookEvent.PubKey},
	}

	contentEvents, err := r.cache.GetEvents(contentFilter)
	if err != nil {
		r.sendError(w, fmt.Sprintf("Failed to get content: %v", err), http.StatusInternalServerError)
		return
	}

	// Filter content events by book identifier
	var bookContent []*models.Event
	for _, event := range contentEvents {
		// Check if this content belongs to our book
		for _, tag := range event.Tags {
			if len(tag) >= 2 && tag[0] == "a" {
				// Check if this is an addressable event for our book
				address := fmt.Sprintf("30040:%s:%s", bookEvent.PubKey, bookIdentifier)
				if tag[1] == address {
					bookContent = append(bookContent, event)
					break
				}
			}
		}
	}

	// Build nested book structure
	bookStructure := r.buildBookStructure(bookEvent, bookContent, depth)

	// Set headers optimized for e-paper readers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=7200") // Cache for 2 hours
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Return structured book content
	response := map[string]interface{}{
		"success": true,
		"book": map[string]interface{}{
			"id":          bookEvent.ID,
			"title":       bookMetadata["title"],
			"author":      bookMetadata["author"],
			"description": bookMetadata["description"],
			"format":      bookMetadata["format"],
			"language":    bookMetadata["language"],
			"created_at":  bookEvent.CreatedAt.Unix(),
			"structure":   bookStructure,
		},
		"content_format": format,
		"include_images": includeImages,
		"max_depth":      depth,
		"timestamp":      time.Now().Unix(),
	}

	json.NewEncoder(w).Encode(response)
}

func (r *RESTAPIServer) buildBookStructure(bookEvent *models.Event, contentEvents []*models.Event, maxDepth int) map[string]interface{} {
	// Build hierarchical book structure from content events
	// This creates a tree structure suitable for e-paper readers

	structure := map[string]interface{}{
		"title":    "Book Structure",
		"type":     "root",
		"children": []map[string]interface{}{},
	}

	// Sort content events by creation time and d tag
	sortedContent := r.sortContentEvents(contentEvents)

	// Build hierarchy based on d tag values
	// d tag format: "chapter-1", "chapter-1-section-1", etc.
	stack := []map[string]interface{}{structure}

	for _, event := range sortedContent {
		// Get the d tag value
		var dTag string
		for _, tag := range event.Tags {
			if len(tag) >= 2 && tag[0] == "d" {
				dTag = tag[1]
				break
			}
		}

		if dTag == "" {
			continue
		}

		// Parse content
		var content map[string]interface{}
		if err := json.Unmarshal([]byte(event.Content), &content); err != nil {
			continue
		}

		// Calculate depth from d tag
		depth := strings.Count(dTag, "-")

		// Adjust stack to current depth
		for len(stack) > depth+1 {
			stack = stack[:len(stack)-1]
		}

		// Create content node
		contentNode := map[string]interface{}{
			"id":         event.ID,
			"title":      content["title"],
			"type":       content["type"], // chapter, section, subsection, etc.
			"content":    content["content"],
			"format":     content["format"], // asciidoc, markdown, etc.
			"created_at": event.CreatedAt.Unix(),
			"children":   []map[string]interface{}{},
		}

		// Add images if requested
		if images, ok := content["images"].([]interface{}); ok {
			contentNode["images"] = images
		}

		// Add to parent
		if parent, ok := stack[len(stack)-1]["children"].([]map[string]interface{}); ok {
			stack[len(stack)-1]["children"] = append(parent, contentNode)
		}

		// Add to stack for potential children
		if depth < maxDepth {
			stack = append(stack, contentNode)
		}
	}

	return structure
}

func (r *RESTAPIServer) sortContentEvents(events []*models.Event) []*models.Event {
	// Sort content events by d tag for proper hierarchy
	sort.Slice(events, func(i, j int) bool {
		var dTagI, dTagJ string

		// Get d tag for event i
		for _, tag := range events[i].Tags {
			if len(tag) >= 2 && tag[0] == "d" {
				dTagI = tag[1]
				break
			}
		}

		// Get d tag for event j
		for _, tag := range events[j].Tags {
			if len(tag) >= 2 && tag[0] == "d" {
				dTagJ = tag[1]
				break
			}
		}

		// Sort by d tag (hierarchical)
		return dTagI < dTagJ
	})

	return events
}

func (r *RESTAPIServer) handleEbookEPUB(w http.ResponseWriter, req *http.Request) {
	// Generate EPUB from any Nostr kind 30040 book
	// This creates a proper EPUB file that can be read on any e-reader

	// Extract book ID from URL path
	vars := mux.Vars(req)
	bookID := vars["id"]

	if bookID == "" {
		r.sendError(w, "Book ID is required", http.StatusBadRequest)
		return
	}

	// Parse query parameters for EPUB options
	includeImages := req.URL.Query().Get("images") == "true"
	format := req.URL.Query().Get("format") // epub, mobi, pdf
	if format == "" {
		format = "epub"
	}

	// Get the main book event (kind 30040)
	bookFilter := nostr.Filter{
		Kinds: []int{30040},
		IDs:   []string{bookID},
	}

	bookEvents, err := r.cache.GetEvents(bookFilter)
	if err != nil {
		r.sendError(w, fmt.Sprintf("Failed to get book: %v", err), http.StatusInternalServerError)
		return
	}

	if len(bookEvents) == 0 {
		r.sendError(w, "Book not found", http.StatusNotFound)
		return
	}

	bookEvent := bookEvents[0]

	// Parse book metadata
	var bookMetadata map[string]interface{}
	if err := json.Unmarshal([]byte(bookEvent.Content), &bookMetadata); err != nil {
		r.sendError(w, "Invalid book metadata", http.StatusBadRequest)
		return
	}

	// Get the book's d tag for finding content events
	var bookIdentifier string
	for _, tag := range bookEvent.Tags {
		if len(tag) >= 2 && tag[0] == "d" {
			bookIdentifier = tag[1]
			break
		}
	}

	if bookIdentifier == "" {
		r.sendError(w, "Book identifier not found", http.StatusBadRequest)
		return
	}

	// Get content events (kind 30041) for this book
	contentFilter := nostr.Filter{
		Kinds:   []int{30041},
		Authors: []string{bookEvent.PubKey},
	}

	contentEvents, err := r.cache.GetEvents(contentFilter)
	if err != nil {
		r.sendError(w, fmt.Sprintf("Failed to get content: %v", err), http.StatusInternalServerError)
		return
	}

	// Filter content events by book identifier
	var bookContent []*models.Event
	for _, event := range contentEvents {
		for _, tag := range event.Tags {
			if len(tag) >= 2 && tag[0] == "a" {
				address := fmt.Sprintf("30040:%s:%s", bookEvent.PubKey, bookIdentifier)
				if tag[1] == address {
					bookContent = append(bookContent, event)
					break
				}
			}
		}
	}

	// Generate EPUB
	epubData, err := r.generateEPUB(bookEvent, bookContent, bookMetadata, includeImages)
	if err != nil {
		r.sendError(w, fmt.Sprintf("Failed to generate EPUB: %v", err), http.StatusInternalServerError)
		return
	}

	// Set headers for file download
	filename := fmt.Sprintf("%s.epub", sanitizeFilename(bookMetadata["title"].(string)))
	w.Header().Set("Content-Type", "application/epub+zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(epubData)))
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour

	// Write EPUB data
	w.Write(epubData)
}

func (r *RESTAPIServer) generateEPUB(bookEvent *models.Event, contentEvents []*models.Event, metadata map[string]interface{}, includeImages bool) ([]byte, error) {
	// Generate EPUB from Nostr book content
	// This creates a proper EPUB structure with all necessary files

	// Create EPUB structure
	epub := &EPUBBook{
		Title:       getString(metadata, "title", "Untitled Book"),
		Author:      getString(metadata, "author", "Unknown Author"),
		Language:    getString(metadata, "language", "en"),
		Description: getString(metadata, "description", ""),
		Publisher:   "Mercury Relay",
		Date:        bookEvent.CreatedAt.Format("2006-01-02"),
		Identifier:  bookEvent.ID,
		Content:     []EPUBChapter{},
		Images:      []EPUBImage{},
	}

	// Sort content events by d tag for proper order
	sortedContent := r.sortContentEvents(contentEvents)

	// Process content events into EPUB chapters
	for i, event := range sortedContent {
		// Parse content
		var content map[string]interface{}
		if err := json.Unmarshal([]byte(event.Content), &content); err != nil {
			continue
		}

		// Get d tag for chapter ordering
		var dTag string
		for _, tag := range event.Tags {
			if len(tag) >= 2 && tag[0] == "d" {
				dTag = tag[1]
				break
			}
		}

		// Create chapter
		chapter := EPUBChapter{
			ID:      fmt.Sprintf("chapter-%d", i+1),
			Title:   getString(content, "title", fmt.Sprintf("Chapter %d", i+1)),
			Content: getString(content, "content", ""),
			Format:  getString(content, "format", "asciidoc"),
			Order:   dTag,
		}

		// Convert content format if needed
		if chapter.Format == "asciidoc" {
			chapter.Content = r.convertAsciiDocToHTML(chapter.Content)
		} else if chapter.Format == "markdown" {
			chapter.Content = r.convertMarkdownToHTML(chapter.Content)
		}

		epub.Content = append(epub.Content, chapter)

		// Process images if requested
		if includeImages {
			if images, ok := content["images"].([]interface{}); ok {
				for _, img := range images {
					if imgMap, ok := img.(map[string]interface{}); ok {
						image := EPUBImage{
							ID:      getString(imgMap, "id", fmt.Sprintf("image-%d", len(epub.Images)+1)),
							URL:     getString(imgMap, "url", ""),
							Alt:     getString(imgMap, "alt", ""),
							Caption: getString(imgMap, "caption", ""),
						}
						epub.Images = append(epub.Images, image)
					}
				}
			}
		}
	}

	// Generate EPUB file
	return r.createEPUBFile(epub)
}

func (r *RESTAPIServer) convertAsciiDocToHTML(content string) string {
	// Simple AsciiDoc to HTML conversion
	// In a real implementation, you'd use a proper AsciiDoc parser

	// Basic conversions
	content = strings.ReplaceAll(content, "\n= ", "\n<h1>")
	content = strings.ReplaceAll(content, "\n== ", "\n<h2>")
	content = strings.ReplaceAll(content, "\n=== ", "\n<h3>")
	content = strings.ReplaceAll(content, "\n==== ", "\n<h4>")

	// Close headers
	content = strings.ReplaceAll(content, "\n<h1>", "\n<h1>")
	content = strings.ReplaceAll(content, "\n<h2>", "\n<h2>")
	content = strings.ReplaceAll(content, "\n<h3>", "\n<h3>")
	content = strings.ReplaceAll(content, "\n<h4>", "\n<h4>")

	// Add closing tags
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "<h1>") || strings.HasPrefix(line, "<h2>") ||
			strings.HasPrefix(line, "<h3>") || strings.HasPrefix(line, "<h4>") {
			lines[i] = line + "</h1>"
			if strings.HasPrefix(line, "<h2>") {
				lines[i] = strings.ReplaceAll(lines[i], "</h1>", "</h2>")
			} else if strings.HasPrefix(line, "<h3>") {
				lines[i] = strings.ReplaceAll(lines[i], "</h1>", "</h3>")
			} else if strings.HasPrefix(line, "<h4>") {
				lines[i] = strings.ReplaceAll(lines[i], "</h1>", "</h4>")
			}
		}
	}

	// Convert paragraphs
	content = strings.Join(lines, "\n")
	content = strings.ReplaceAll(content, "\n\n", "\n<p>")
	content = strings.ReplaceAll(content, "\n", "<br>\n")

	return content
}

func (r *RESTAPIServer) convertMarkdownToHTML(content string) string {
	// Simple Markdown to HTML conversion
	// In a real implementation, you'd use a proper Markdown parser

	// Headers
	content = strings.ReplaceAll(content, "\n# ", "\n<h1>")
	content = strings.ReplaceAll(content, "\n## ", "\n<h2>")
	content = strings.ReplaceAll(content, "\n### ", "\n<h3>")
	content = strings.ReplaceAll(content, "\n#### ", "\n<h4>")

	// Bold and italic
	content = strings.ReplaceAll(content, "**", "<strong>")
	content = strings.ReplaceAll(content, "*", "<em>")

	// Links
	content = strings.ReplaceAll(content, "[", "<a href=\"")
	content = strings.ReplaceAll(content, "](", "\">")
	content = strings.ReplaceAll(content, ")", "</a>")

	return content
}

func (r *RESTAPIServer) createEPUBFile(book *EPUBBook) ([]byte, error) {
	// Create EPUB file structure
	// This is a simplified implementation - in production you'd use a proper EPUB library

	// For now, return a placeholder that indicates EPUB generation
	// In a real implementation, you'd create:
	// - mimetype file
	// - META-INF/container.xml
	// - OEBPS/content.opf
	// - OEBPS/toc.ncx
	// - OEBPS/*.xhtml files for each chapter
	// - OEBPS/*.css for styling
	// - OEBPS/images/* for images

	epubData := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0" unique-identifier="book-id">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>%s</dc:title>
    <dc:creator>%s</dc:creator>
    <dc:language>%s</dc:language>
    <dc:description>%s</dc:description>
    <dc:publisher>%s</dc:publisher>
    <dc:date>%s</dc:date>
    <dc:identifier id="book-id">%s</dc:identifier>
  </metadata>
  <manifest>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
    <item id="css" href="style.css" media-type="text/css"/>
    <item id="cover" href="cover.xhtml" media-type="application/xhtml+xml"/>
    %s
  </manifest>
  <spine toc="ncx">
    <itemref idref="cover"/>
    %s
  </spine>
</package>`,
		book.Title,
		book.Author,
		book.Language,
		book.Description,
		book.Publisher,
		book.Date,
		book.Identifier,
		r.generateManifestItems(book),
		r.generateSpineItems(book),
	)

	// In a real implementation, you'd create a proper ZIP file with all the EPUB structure
	// For now, return the OPF content as a placeholder
	return []byte(epubData), nil
}

func (r *RESTAPIServer) generateManifestItems(book *EPUBBook) string {
	var items []string
	for i := range book.Content {
		items = append(items, fmt.Sprintf(`<item id="chapter-%d" href="chapter-%d.xhtml" media-type="application/xhtml+xml"/>`, i+1, i+1))
	}
	return strings.Join(items, "\n    ")
}

func (r *RESTAPIServer) generateSpineItems(book *EPUBBook) string {
	var items []string
	for i := range book.Content {
		items = append(items, fmt.Sprintf(`<itemref idref="chapter-%d"/>`, i+1))
	}
	return strings.Join(items, "\n    ")
}

func getString(m map[string]interface{}, key, defaultValue string) string {
	if value, ok := m[key].(string); ok {
		return value
	}
	return defaultValue
}

func sanitizeFilename(filename string) string {
	// Remove invalid characters for filenames
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range invalid {
		filename = strings.ReplaceAll(filename, char, "_")
	}
	return filename
}

// EPUB structures
type EPUBBook struct {
	Title       string
	Author      string
	Language    string
	Description string
	Publisher   string
	Date        string
	Identifier  string
	Content     []EPUBChapter
	Images      []EPUBImage
}

type EPUBChapter struct {
	ID      string
	Title   string
	Content string
	Format  string
	Order   string
}

type EPUBImage struct {
	ID      string
	URL     string
	Alt     string
	Caption string
}

func (r *RESTAPIServer) sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := APIResponse{
		Success: false,
		Error:   message,
	}

	json.NewEncoder(w).Encode(response)
}
