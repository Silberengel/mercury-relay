package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"mercury-relay/internal/access"
	"mercury-relay/internal/api"
	"mercury-relay/internal/cache"
	"mercury-relay/internal/config"
	"mercury-relay/internal/models"
	"mercury-relay/internal/quality"
	"mercury-relay/internal/queue"
	"mercury-relay/internal/storage"
	"mercury-relay/internal/streaming"
	"mercury-relay/internal/transport"

	"github.com/gorilla/websocket"
	"github.com/nbd-wtf/go-nostr"
)

type Server struct {
	config         config.ServerConfig
	transportMgr   *transport.Manager
	rabbitMQ       queue.Queue
	cache          cache.Cache
	storage        storage.Storage
	qualityControl *quality.Controller
	accessControl  *access.Controller
	upstreamMgr    *streaming.UpstreamManager
	restAPI        *api.RESTAPIServer

	// WebSocket upgrader
	upgrader websocket.Upgrader

	// SSH tunnel for WebSocket over SSH
	sshTunnel *transport.WebSocketSSHTunnel

	// Active connections
	connections map[*websocket.Conn]*Connection
	connMutex   sync.RWMutex

	// Event handlers
	eventHandlers map[string]EventHandler
}

type Connection struct {
	conn     *websocket.Conn
	subs     map[string]*Subscription
	subMutex sync.RWMutex
	lastPing time.Time
	pubkey   string // Authenticated user's public key
}

type Subscription struct {
	ID     string
	Filter nostr.Filter
	Active bool
}

type EventHandler func(*models.Event) error

func NewServer(
	cfg config.ServerConfig,
	transportMgr *transport.Manager,
	rabbitMQ queue.Queue,
	cache cache.Cache,
	storage storage.Storage,
	qualityControl *quality.Controller,
	accessControl *access.Controller,
	upstreamMgr *streaming.UpstreamManager,
	restAPI *api.RESTAPIServer,
) *Server {
	server := &Server{
		config:         cfg,
		transportMgr:   transportMgr,
		rabbitMQ:       rabbitMQ,
		cache:          cache,
		storage:        storage,
		qualityControl: qualityControl,
		accessControl:  accessControl,
		upstreamMgr:    upstreamMgr,
		restAPI:        restAPI,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
		connections:   make(map[*websocket.Conn]*Connection),
		eventHandlers: make(map[string]EventHandler),
	}

	// Initialize SSH tunnel if SSH transport is available
	if transportMgr != nil {
		if sshTransport := transportMgr.GetSSHTransport(); sshTransport != nil {
			server.sshTunnel = transport.NewWebSocketSSHTunnel(sshTransport)
		}
	}

	return server
}

func (s *Server) Start(ctx context.Context) error {
	// Start transport manager
	if err := s.transportMgr.Start(ctx); err != nil {
		return fmt.Errorf("failed to start transport manager: %w", err)
	}

	// Start quality control
	if err := s.qualityControl.Start(ctx); err != nil {
		return fmt.Errorf("failed to start quality control: %w", err)
	}

	// Start upstream streaming
	if s.upstreamMgr != nil {
		if err := s.upstreamMgr.Start(ctx); err != nil {
			return fmt.Errorf("failed to start upstream manager: %w", err)
		}
	}

	// Start REST API
	if s.restAPI != nil {
		go func() {
			if err := s.restAPI.Start(ctx); err != nil {
				log.Printf("REST API error: %v", err)
			}
		}()
	}

	// Start event processing
	go s.processEvents(ctx)

	// Start HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleWebSocket)

	// Add WebSocket over SSH endpoint if SSH tunnel is available
	if s.sshTunnel != nil {
		mux.HandleFunc("/ssh", s.handleWebSocketOverSSH)
		log.Println("WebSocket over SSH endpoint available at /ssh")
	}

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.Host, s.config.Port),
		Handler:      mux,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting Mercury Relay on %s:%d", s.config.Host, s.config.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return server.Shutdown(ctx)
}

func (s *Server) handleWebSocketOverSSH(w http.ResponseWriter, r *http.Request) {
	if s.sshTunnel == nil {
		http.Error(w, "SSH tunnel not available", http.StatusServiceUnavailable)
		return
	}

	log.Printf("WebSocket over SSH connection attempt from %s", r.RemoteAddr)

	// Handle WebSocket over SSH tunnel
	if err := s.sshTunnel.HandleWebSocketOverSSH(w, r); err != nil {
		log.Printf("WebSocket over SSH failed: %v", err)
		http.Error(w, "SSH tunnel connection failed", http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Printf("=== WebSocket handler called ===")
	log.Printf("Request from: %s", r.RemoteAddr)
	log.Printf("Request method: %s", r.Method)
	log.Printf("Request URL: %s", r.URL.String())

	// Check if this is a WebSocket upgrade request by examining headers manually
	upgrade := r.Header.Get("Upgrade")
	connection := r.Header.Get("Connection")

	log.Printf("Upgrade header: %s", upgrade)
	log.Printf("Connection header: %s", connection)

	// Check if this is a proper WebSocket upgrade request
	if upgrade != "websocket" || !strings.Contains(strings.ToLower(connection), "upgrade") {
		// For regular HTTP requests, return a simple response
		log.Printf("Regular HTTP request, returning info page")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Mercury Relay - WebSocket endpoint\nUse ws:// or wss:// to connect"))
		return
	}

	log.Printf("Attempting WebSocket upgrade...")
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	log.Printf("WebSocket upgrade successful! Connection established.")
	defer conn.Close()

	// Create connection
	wsConnection := &Connection{
		conn:     conn,
		subs:     make(map[string]*Subscription),
		lastPing: time.Now(),
		pubkey:   "", // Will be extracted from first EVENT message
	}

	// Register connection
	s.connMutex.Lock()
	s.connections[conn] = wsConnection
	s.connMutex.Unlock()

	// Cleanup on disconnect
	defer func() {
		s.connMutex.Lock()
		delete(s.connections, conn)
		s.connMutex.Unlock()
	}()

	// Handle messages
	log.Printf("Starting message handling loop for connection from %s", r.RemoteAddr)
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			log.Printf("WebSocket connection closed: %v", err)
			break
		}

		log.Printf("Received message from %s: %s", r.RemoteAddr, string(message))
		if err := s.handleMessage(wsConnection, message); err != nil {
			log.Printf("Error handling message: %v", err)
			s.sendError(conn, "error", err.Error())
		}
	}
	log.Printf("Message handling loop ended for connection from %s", r.RemoteAddr)
}

func (s *Server) handleMessage(conn *Connection, message []byte) error {
	var msg []interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	if len(msg) < 2 {
		return fmt.Errorf("message too short")
	}

	msgType, ok := msg[0].(string)
	if !ok {
		return fmt.Errorf("invalid message type")
	}

	switch msgType {
	case "REQ":
		return s.handleREQ(conn, msg[1:])
	case "EVENT":
		return s.handleEVENT(conn, msg[1:])
	case "CLOSE":
		return s.handleCLOSE(conn, msg[1:])
	default:
		return fmt.Errorf("unknown message type: %s", msgType)
	}
}

func (s *Server) handleREQ(conn *Connection, args []interface{}) error {
	if len(args) < 2 {
		return fmt.Errorf("REQ requires subscription ID and filter")
	}

	subID, ok := args[0].(string)
	if !ok {
		return fmt.Errorf("invalid subscription ID")
	}

	filterData, ok := args[1].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid filter")
	}

	// Parse filter
	filter := nostr.Filter{}
	if authors, ok := filterData["authors"].([]interface{}); ok {
		for _, author := range authors {
			if authorStr, ok := author.(string); ok {
				filter.Authors = append(filter.Authors, authorStr)
			}
		}
	}
	if kinds, ok := filterData["kinds"].([]interface{}); ok {
		for _, kind := range kinds {
			if kindInt, ok := kind.(float64); ok {
				filter.Kinds = append(filter.Kinds, int(kindInt))
			}
		}
	}
	if since, ok := filterData["since"].(float64); ok {
		timestamp := nostr.Timestamp(since)
		filter.Since = &timestamp
	}
	if until, ok := filterData["until"].(float64); ok {
		timestamp := nostr.Timestamp(until)
		filter.Until = &timestamp
	}
	if limit, ok := filterData["limit"].(float64); ok {
		filter.Limit = int(limit)
	}

	// Create subscription
	sub := &Subscription{
		ID:     subID,
		Filter: filter,
		Active: true,
	}

	conn.subMutex.Lock()
	conn.subs[subID] = sub
	conn.subMutex.Unlock()

	// Send matching events
	go s.sendMatchingEvents(conn, sub)

	return nil
}

func (s *Server) handleEVENT(conn *Connection, args []interface{}) error {
	if len(args) < 1 {
		return fmt.Errorf("EVENT requires event data")
	}

	eventData, ok := args[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid event data")
	}

	// Parse event
	event := &models.Event{}
	if id, ok := eventData["id"].(string); ok {
		event.ID = id
	}
	if pubkey, ok := eventData["pubkey"].(string); ok {
		event.PubKey = pubkey
		// Store the pubkey in the connection for future use
		if conn.pubkey == "" {
			conn.pubkey = pubkey
			log.Printf("Authenticated user: %s", pubkey)
		}
	}
	if createdAt, ok := eventData["created_at"].(float64); ok {
		event.CreatedAt = nostr.Timestamp(createdAt)
	}
	if kind, ok := eventData["kind"].(float64); ok {
		event.Kind = int(kind)
	}
	if content, ok := eventData["content"].(string); ok {
		event.Content = content
	}
	if sig, ok := eventData["sig"].(string); ok {
		event.Sig = sig
	}

	// Check access control
	log.Printf("Checking write access for npub: %s", event.PubKey)
	canWrite := s.accessControl.CanWrite(event.PubKey)
	log.Printf("Access control result: %v", canWrite)

	if !canWrite {
		log.Printf("Write access denied for npub: %s", event.PubKey)
		s.sendError(conn.conn, "restricted", "Write access denied")
		return fmt.Errorf("write access denied for npub: %s", event.PubKey)
	}

	// Validate event
	if err := event.Validate(); err != nil {
		return fmt.Errorf("event validation failed: %w", err)
	}

	// Calculate quality score
	event.QualityScore = event.CalculateQualityScore()

	// Check for spam
	if event.IsSpam(0.7) {
		event.IsQuarantined = true
		event.QuarantineReason = "Low quality score"
	}

	// Publish to queue
	if err := s.rabbitMQ.PublishEvent(event); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	// Send OK response
	s.sendOK(conn.conn, event.ID, true, "")

	return nil
}

func (s *Server) handleCLOSE(conn *Connection, args []interface{}) error {
	if len(args) < 1 {
		return fmt.Errorf("CLOSE requires subscription ID")
	}

	subID, ok := args[0].(string)
	if !ok {
		return fmt.Errorf("invalid subscription ID")
	}

	conn.subMutex.Lock()
	if sub, exists := conn.subs[subID]; exists {
		sub.Active = false
		delete(conn.subs, subID)
	}
	conn.subMutex.Unlock()

	return nil
}

func (s *Server) sendMatchingEvents(conn *Connection, sub *Subscription) {
	// Get events from cache first
	events, err := s.cache.GetEvents(sub.Filter)
	if err != nil {
		log.Printf("Error getting events from cache: %v", err)
	}

	// Create privacy filter for the connection
	privacyFilter := NewPrivacyFilter(conn.pubkey)

	// Send events
	for _, event := range events {
		if !sub.Active {
			break
		}

		// Check if event matches filter
		if s.eventMatchesFilter(event, sub.Filter) {
			// Apply privacy filtering
			if privacyFilter.CanAccessEvent(event) {
				s.sendEvent(conn.conn, sub.ID, event)
			}
		}
	}
}

func (s *Server) eventMatchesFilter(event *models.Event, filter nostr.Filter) bool {
	// Check authors
	if len(filter.Authors) > 0 {
		found := false
		for _, author := range filter.Authors {
			if event.PubKey == author {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check kinds
	if len(filter.Kinds) > 0 {
		found := false
		for _, kind := range filter.Kinds {
			if event.Kind == kind {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check since
	if filter.Since != nil && *filter.Since > 0 {
		if nostr.Timestamp(int64(event.CreatedAt)) < *filter.Since {
			return false
		}
	}

	// Check until
	if filter.Until != nil && *filter.Until > 0 {
		if nostr.Timestamp(int64(event.CreatedAt)) > *filter.Until {
			return false
		}
	}

	return true
}

func (s *Server) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Process events from queue
			events, err := s.rabbitMQ.ConsumeEvents()
			if err != nil {
				log.Printf("Error consuming events: %v", err)
				time.Sleep(time.Second)
				continue
			}

			for _, event := range events {
				// Store in cache
				if err := s.cache.StoreEvent(event); err != nil {
					log.Printf("Error storing event in cache: %v", err)
				}

				// Store in XFTP if enabled
				if s.storage != nil {
					if err := s.storage.StoreEvent(event); err != nil {
						log.Printf("Error storing event in XFTP: %v", err)
					}
				}

				// Broadcast to subscribers
				s.broadcastEvent(event)
			}

			// Add delay to prevent tight loop and reduce consumer count
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (s *Server) broadcastEvent(event *models.Event) {
	s.connMutex.RLock()
	defer s.connMutex.RUnlock()

	for conn, connection := range s.connections {
		connection.subMutex.RLock()
		for _, sub := range connection.subs {
			if sub.Active && s.eventMatchesFilter(event, sub.Filter) {
				s.sendEvent(conn, sub.ID, event)
			}
		}
		connection.subMutex.RUnlock()
	}
}

func (s *Server) sendEvent(conn *websocket.Conn, subID string, event *models.Event) {
	msg := []interface{}{
		"EVENT",
		subID,
		event.ToNostrEvent(),
	}

	if err := conn.WriteJSON(msg); err != nil {
		log.Printf("Error sending event: %v", err)
	}
}

func (s *Server) sendOK(conn *websocket.Conn, eventID string, ok bool, message string) {
	msg := []interface{}{
		"OK",
		eventID,
		ok,
		message,
	}

	if err := conn.WriteJSON(msg); err != nil {
		log.Printf("Error sending OK: %v", err)
	}
}

func (s *Server) sendError(conn *websocket.Conn, errorType, message string) {
	msg := []interface{}{
		"NOTICE",
		fmt.Sprintf("[%s] %s", errorType, message),
	}

	if err := conn.WriteJSON(msg); err != nil {
		log.Printf("Error sending error: %v", err)
	}
}
