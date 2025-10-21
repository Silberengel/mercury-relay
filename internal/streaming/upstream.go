package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	"mercury-relay/internal/cache"
	"mercury-relay/internal/config"
	"mercury-relay/internal/models"
	"mercury-relay/internal/quality"
	"mercury-relay/internal/queue"

	"github.com/gorilla/websocket"
	"github.com/nbd-wtf/go-nostr"
)

type UpstreamManager struct {
	config         config.StreamingConfig
	qualityControl *quality.Controller
	rabbitMQ       queue.Queue
	cache          cache.Cache
	connections    map[string]*UpstreamConnection
	connMutex      sync.RWMutex
	transportMgr   *TransportManager
}

type UpstreamConnection struct {
	URL           string
	Conn          *websocket.Conn
	Active        bool
	LastPing      time.Time
	Subscriptions map[string]*UpstreamSubscription
	subMutex      sync.RWMutex
}

type UpstreamSubscription struct {
	ID     string
	Filter nostr.Filter
	Active bool
}

type TransportManager struct {
	torEnabled    bool
	i2pEnabled    bool
	httpStreaming bool
	sseEnabled    bool
}

func NewUpstreamManager(
	config config.StreamingConfig,
	qualityControl *quality.Controller,
	rabbitMQ queue.Queue,
	cache cache.Cache,
) *UpstreamManager {
	return &UpstreamManager{
		config:         config,
		qualityControl: qualityControl,
		rabbitMQ:       rabbitMQ,
		cache:          cache,
		connections:    make(map[string]*UpstreamConnection),
		transportMgr: &TransportManager{
			torEnabled:    config.TransportMethods.Tor,
			i2pEnabled:    config.TransportMethods.I2P,
			httpStreaming: config.TransportMethods.HTTPStreaming,
			sseEnabled:    config.TransportMethods.SSE,
		},
	}
}

func (u *UpstreamManager) Start(ctx context.Context) error {
	if !u.config.Enabled {
		log.Println("Streaming is disabled")
		return nil
	}

	// Start connections to upstream relays
	for _, relay := range u.config.UpstreamRelays {
		if relay.Enabled {
			go u.connectToRelay(ctx, relay)
		}
	}

	// Start connection health monitoring
	go u.monitorConnections(ctx)

	return nil
}

func (u *UpstreamManager) connectToRelay(ctx context.Context, relay config.UpstreamRelay) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := u.establishConnection(ctx, relay); err != nil {
				log.Printf("Failed to connect to relay %s: %v", relay.URL, err)
				time.Sleep(u.config.ReconnectInterval)
				continue
			}
		}
	}
}

func (u *UpstreamManager) establishConnection(ctx context.Context, relay config.UpstreamRelay) error {
	// Parse relay URL
	relayURL, err := url.Parse(relay.URL)
	if err != nil {
		return fmt.Errorf("invalid relay URL: %w", err)
	}

	// Determine connection method based on URL scheme
	switch relayURL.Scheme {
	case "wss", "ws":
		return u.establishWebSocketConnection(ctx, relay)
	case "https", "http":
		if u.transportMgr.httpStreaming {
			return u.establishHTTPStreamingConnection(ctx, relay)
		}
		return fmt.Errorf("HTTP streaming not enabled")
	case "sse":
		if u.transportMgr.sseEnabled {
			return u.establishSSEConnection(ctx, relay)
		}
		return fmt.Errorf("SSE not enabled")
	default:
		return fmt.Errorf("unsupported relay URL scheme: %s", relayURL.Scheme)
	}
}

func (u *UpstreamManager) establishWebSocketConnection(ctx context.Context, relay config.UpstreamRelay) error {
	// Determine transport method
	var dialer websocket.Dialer
	if u.transportMgr.torEnabled {
		dialer = u.getTorDialer()
	} else if u.transportMgr.i2pEnabled {
		dialer = u.getI2PDialer()
	}

	// Connect to relay
	conn, _, err := dialer.Dial(relay.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to dial relay: %w", err)
	}

	// Create connection object
	upstreamConn := &UpstreamConnection{
		URL:           relay.URL,
		Conn:          conn,
		Active:        true,
		LastPing:      time.Now(),
		Subscriptions: make(map[string]*UpstreamSubscription),
	}

	// Store connection
	u.connMutex.Lock()
	u.connections[relay.URL] = upstreamConn
	u.connMutex.Unlock()

	log.Printf("Connected to upstream relay: %s", relay.URL)

	// Start message handling
	go u.handleUpstreamMessages(ctx, upstreamConn)

	// Start subscription to all events
	go u.subscribeToAllEvents(ctx, upstreamConn)

	// Keep connection alive
	u.keepAlive(ctx, upstreamConn)

	return nil
}

func (u *UpstreamManager) handleUpstreamMessages(ctx context.Context, conn *UpstreamConnection) {
	defer conn.Conn.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, message, err := conn.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("Upstream connection error: %v", err)
				}
				u.removeConnection(conn.URL)
				return
			}

			if err := u.handleUpstreamMessage(conn, message); err != nil {
				log.Printf("Error handling upstream message: %v", err)
			}
		}
	}
}

func (u *UpstreamManager) handleUpstreamMessage(conn *UpstreamConnection, message []byte) error {
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
	case "EVENT":
		return u.handleUpstreamEvent(conn, msg[1:])
	case "EOSE":
		return u.handleUpstreamEOSE(conn, msg[1:])
	case "NOTICE":
		return u.handleUpstreamNotice(conn, msg[1:])
	default:
		log.Printf("Unknown upstream message type: %s", msgType)
	}

	return nil
}

func (u *UpstreamManager) handleUpstreamEvent(conn *UpstreamConnection, args []interface{}) error {
	if len(args) < 2 {
		return fmt.Errorf("EVENT requires subscription ID and event data")
	}

	_, ok := args[0].(string)
	if !ok {
		return fmt.Errorf("invalid subscription ID")
	}

	eventData, ok := args[1].(map[string]interface{})
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

	// Parse tags
	if tags, ok := eventData["tags"].([]interface{}); ok {
		for _, tag := range tags {
			if tagArray, ok := tag.([]interface{}); ok && len(tagArray) >= 2 {
				if tagType, ok := tagArray[0].(string); ok {
					if tagValue, ok := tagArray[1].(string); ok {
						event.Tags = append(event.Tags, []string{tagType, tagValue})
					}
				}
			}
		}
	}

	// Validate event
	if err := event.Validate(); err != nil {
		log.Printf("Invalid upstream event: %v", err)
		return nil
	}

	// Check quality control
	if err := u.qualityControl.ValidateEvent(event); err != nil {
		log.Printf("Upstream event failed quality control: %v", err)
		return nil
	}

	// Store in cache
	if err := u.cache.StoreEvent(event); err != nil {
		log.Printf("Failed to store upstream event in cache: %v", err)
	}

	// Publish to queue
	if err := u.rabbitMQ.PublishEvent(event); err != nil {
		log.Printf("Failed to publish upstream event: %v", err)
	}

	return nil
}

func (u *UpstreamManager) handleUpstreamEOSE(conn *UpstreamConnection, args []interface{}) error {
	if len(args) < 1 {
		return fmt.Errorf("EOSE requires subscription ID")
	}

	subID, ok := args[0].(string)
	if !ok {
		return fmt.Errorf("invalid subscription ID")
	}

	log.Printf("End of stored events for subscription %s from relay %s", subID, conn.URL)
	return nil
}

func (u *UpstreamManager) handleUpstreamNotice(conn *UpstreamConnection, args []interface{}) error {
	if len(args) < 1 {
		return fmt.Errorf("NOTICE requires message")
	}

	message, ok := args[0].(string)
	if !ok {
		return fmt.Errorf("invalid notice message")
	}

	log.Printf("Notice from relay %s: %s", conn.URL, message)
	return nil
}

func (u *UpstreamManager) subscribeToAllEvents(ctx context.Context, conn *UpstreamConnection) {
	// Subscribe to all events
	subID := fmt.Sprintf("all-events-%d", time.Now().Unix())

	req := []interface{}{
		"REQ",
		subID,
		map[string]interface{}{
			"limit": 1000,
		},
	}

	if err := conn.Conn.WriteJSON(req); err != nil {
		log.Printf("Failed to subscribe to all events: %v", err)
		return
	}

	// Store subscription
	conn.subMutex.Lock()
	conn.Subscriptions[subID] = &UpstreamSubscription{
		ID:     subID,
		Filter: nostr.Filter{Limit: 1000},
		Active: true,
	}
	conn.subMutex.Unlock()

	log.Printf("Subscribed to all events from relay %s with subscription ID %s", conn.URL, subID)
}

func (u *UpstreamManager) keepAlive(ctx context.Context, conn *UpstreamConnection) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := conn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Failed to ping upstream relay %s: %v", conn.URL, err)
				u.removeConnection(conn.URL)
				return
			}
			conn.LastPing = time.Now()
		}
	}
}

func (u *UpstreamManager) monitorConnections(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			u.connMutex.RLock()
			for url, conn := range u.connections {
				if time.Since(conn.LastPing) > u.config.Timeout {
					log.Printf("Upstream connection %s timed out", url)
					u.removeConnection(url)
				}
			}
			u.connMutex.RUnlock()
		}
	}
}

func (u *UpstreamManager) removeConnection(url string) {
	u.connMutex.Lock()
	defer u.connMutex.Unlock()

	if conn, exists := u.connections[url]; exists {
		conn.Conn.Close()
		delete(u.connections, url)
		log.Printf("Removed connection to relay: %s", url)
	}
}

func (u *UpstreamManager) getTorDialer() websocket.Dialer {
	// TODO: Implement Tor dialer
	return websocket.Dialer{}
}

func (u *UpstreamManager) getI2PDialer() websocket.Dialer {
	// TODO: Implement I2P dialer
	return websocket.Dialer{}
}

func (u *UpstreamManager) establishHTTPStreamingConnection(ctx context.Context, relay config.UpstreamRelay) error {
	// HTTP streaming connection for server-side rendering
	// This would typically involve:
	// 1. Making HTTP requests to the relay's REST API
	// 2. Polling for new events at regular intervals
	// 3. Handling chunked responses for streaming

	log.Printf("HTTP streaming connection to %s not yet implemented", relay.URL)
	return fmt.Errorf("HTTP streaming not yet implemented")
}

func (u *UpstreamManager) establishSSEConnection(ctx context.Context, relay config.UpstreamRelay) error {
	// Server-Sent Events connection
	// This would involve:
	// 1. Opening an SSE connection to the relay
	// 2. Listening for real-time event updates
	// 3. Parsing SSE events and converting to Nostr events

	log.Printf("SSE connection to %s not yet implemented", relay.URL)
	return fmt.Errorf("SSE connection not yet implemented")
}

func (u *UpstreamManager) GetActiveConnections() []string {
	u.connMutex.RLock()
	defer u.connMutex.RUnlock()

	var connections []string
	for url := range u.connections {
		connections = append(connections, url)
	}
	return connections
}

func (u *UpstreamManager) GetConnectionStats() map[string]interface{} {
	u.connMutex.RLock()
	defer u.connMutex.RUnlock()

	stats := map[string]interface{}{
		"total_connections": len(u.connections),
		"connections":       make([]map[string]interface{}, 0),
	}

	for url, conn := range u.connections {
		connStats := map[string]interface{}{
			"url":           url,
			"active":        conn.Active,
			"last_ping":     conn.LastPing,
			"subscriptions": len(conn.Subscriptions),
		}
		stats["connections"] = append(stats["connections"].([]map[string]interface{}), connStats)
	}

	return stats
}
