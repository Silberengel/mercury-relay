package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"mercury-relay/internal/cache"
	"mercury-relay/internal/config"
	"mercury-relay/internal/quality"
	"mercury-relay/internal/queue"
	"mercury-relay/internal/storage"
)

type AdminAPI struct {
	config         config.AdminConfig
	qualityControl *quality.Controller
	rabbitMQ       queue.Queue
	cache          cache.Cache
	storage        storage.Storage
	server         *http.Server
}

func NewAdminAPI(
	config config.AdminConfig,
	qualityControl *quality.Controller,
	rabbitMQ queue.Queue,
	cache cache.Cache,
	storage storage.Storage,
) *AdminAPI {
	return &AdminAPI{
		config:         config,
		qualityControl: qualityControl,
		rabbitMQ:       rabbitMQ,
		cache:          cache,
		storage:        storage,
	}
}

func (a *AdminAPI) Start() error {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/stats", a.handleStats)
	mux.HandleFunc("/api/block", a.handleBlock)
	mux.HandleFunc("/api/unblock", a.handleUnblock)
	mux.HandleFunc("/api/blocked", a.handleBlocked)
	mux.HandleFunc("/api/events", a.handleEvents)

	// Health check
	mux.HandleFunc("/health", a.handleHealth)

	a.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", a.config.Port),
		Handler: a.authenticate(mux),
	}

	log.Printf("Starting admin API on port %d", a.config.Port)
	return a.server.ListenAndServe()
}

func (a *AdminAPI) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simple API key authentication
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != a.config.APIKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (a *AdminAPI) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := make(map[string]interface{})

	// Get RabbitMQ stats
	if queueStats, err := a.rabbitMQ.GetQueueStats(); err == nil {
		stats["queue_depth"] = queueStats
	}

	// Get Redis stats
	if cacheStats, err := a.cache.GetStats(); err == nil {
		stats["cache"] = cacheStats
	}

	// Get XFTP stats
	if a.storage != nil {
		if storageStats, err := a.storage.GetStats(); err == nil {
			stats["storage"] = storageStats
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (a *AdminAPI) handleBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Npub string `json:"npub"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := a.qualityControl.BlockNpub(req.Npub); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "blocked"})
}

func (a *AdminAPI) handleUnblock(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Npub string `json:"npub"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := a.qualityControl.UnblockNpub(req.Npub); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "unblocked"})
}

func (a *AdminAPI) handleBlocked(w http.ResponseWriter, r *http.Request) {
	blocked := a.qualityControl.GetBlockedNpubs()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]string{"blocked": blocked})
}

func (a *AdminAPI) handleEvents(w http.ResponseWriter, r *http.Request) {
	// This would return recent events for moderation
	// For now, return empty list
	events := []interface{}{}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"events": events})
}

func (a *AdminAPI) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (a *AdminAPI) Stop(ctx context.Context) error {
	if a.server != nil {
		return a.server.Shutdown(ctx)
	}
	return nil
}
