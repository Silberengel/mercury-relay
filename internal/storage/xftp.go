package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"mercury-relay/internal/config"
	"mercury-relay/internal/models"
)

type XFTPStorage struct {
	config     config.XFTPConfig
	httpClient *http.Client
	baseURL    string
}

func NewXFTP(config config.XFTPConfig) (*XFTPStorage, error) {
	// Parse XFTP server URL
	baseURL := config.ServerURL
	if baseURL == "" {
		baseURL = "http://localhost:443"
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &XFTPStorage{
		config:     config,
		httpClient: client,
		baseURL:    baseURL,
	}, nil
}

func (x *XFTPStorage) StoreEvent(event *models.Event) error {
	// Convert event to JSON
	_, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create XFTP upload request
	req, err := http.NewRequest("POST", x.baseURL+"/upload", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Event-ID", event.ID)
	req.Header.Set("X-Event-Kind", fmt.Sprintf("%d", event.Kind))
	req.Header.Set("X-Event-Author", event.PubKey)

	// Send request
	resp, err := x.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upload failed with status: %d", resp.StatusCode)
	}

	log.Printf("Event %s stored in XFTP", event.ID)
	return nil
}

func (x *XFTPStorage) GetEvent(eventID string) (*models.Event, error) {
	// Create XFTP download request
	req, err := http.NewRequest("GET", x.baseURL+"/download/"+eventID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Send request
	resp, err := x.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("event not found")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Parse response
	var event models.Event
	if err := json.NewDecoder(resp.Body).Decode(&event); err != nil {
		return nil, fmt.Errorf("failed to decode event: %w", err)
	}

	return &event, nil
}

func (x *XFTPStorage) DeleteEvent(eventID string) error {
	// Create XFTP delete request
	req, err := http.NewRequest("DELETE", x.baseURL+"/delete/"+eventID, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Send request
	resp, err := x.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete failed with status: %d", resp.StatusCode)
	}

	log.Printf("Event %s deleted from XFTP", eventID)
	return nil
}

func (x *XFTPStorage) GetStats() (map[string]interface{}, error) {
	// Create XFTP stats request
	req, err := http.NewRequest("GET", x.baseURL+"/stats", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Send request
	resp, err := x.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stats request failed with status: %d", resp.StatusCode)
	}

	// Parse response
	var stats map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	return stats, nil
}

func (x *XFTPStorage) Close() error {
	// XFTP storage doesn't need explicit cleanup
	return nil
}
