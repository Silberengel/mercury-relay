package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// HandleEventHistory handles requests for replaceable event history
func (r *RESTAPIServer) HandleEventHistory(w http.ResponseWriter, req *http.Request) {
	// Extract parameters from URL
	vars := mux.Vars(req)
	kindStr := vars["kind"]
	pubkey := vars["pubkey"]
	dTag := vars["d_tag"]

	// Parse kind
	kind, err := strconv.Atoi(kindStr)
	if err != nil {
		http.Error(w, "Invalid kind parameter", http.StatusBadRequest)
		return
	}

	// Get history from cache
	history, err := r.cache.GetReplaceableEventHistory(kind, pubkey, dTag)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get event history: %v", err), http.StatusInternalServerError)
		return
	}

	// Return history
	response := map[string]interface{}{
		"success": true,
		"key":     fmt.Sprintf("%d:%s:%s", kind, pubkey, dTag),
		"history": history,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleEventDiff handles requests for comparing event versions
func (r *RESTAPIServer) HandleEventDiff(w http.ResponseWriter, req *http.Request) {
	// Extract parameters from URL
	vars := mux.Vars(req)
	kindStr := vars["kind"]
	pubkey := vars["pubkey"]
	dTag := vars["d_tag"]
	fromVersionStr := vars["from_version"]
	toVersionStr := vars["to_version"]

	// Parse parameters
	kind, err := strconv.Atoi(kindStr)
	if err != nil {
		http.Error(w, "Invalid kind parameter", http.StatusBadRequest)
		return
	}

	fromVersion, err := strconv.Atoi(fromVersionStr)
	if err != nil {
		http.Error(w, "Invalid from_version parameter", http.StatusBadRequest)
		return
	}

	toVersion, err := strconv.Atoi(toVersionStr)
	if err != nil {
		http.Error(w, "Invalid to_version parameter", http.StatusBadRequest)
		return
	}

	// Get history
	history, err := r.cache.GetReplaceableEventHistory(kind, pubkey, dTag)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get event history: %v", err), http.StatusInternalServerError)
		return
	}

	// Find the specific versions
	var fromEvent, toEvent map[string]interface{}
	for _, version := range history {
		if version["version"] == float64(fromVersion) {
			fromEvent = version
		}
		if version["version"] == float64(toVersion) {
			toEvent = version
		}
	}

	if fromEvent == nil || toEvent == nil {
		http.Error(w, "Version not found", http.StatusNotFound)
		return
	}

	// Calculate diff
	diff := r.calculateEventDiff(fromEvent, toEvent)

	// Return diff
	response := map[string]interface{}{
		"success":      true,
		"key":          fmt.Sprintf("%d:%s:%s", kind, pubkey, dTag),
		"from_version": fromVersion,
		"to_version":   toVersion,
		"diff":         diff,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleEventVersion handles requests for specific event versions
func (r *RESTAPIServer) HandleEventVersion(w http.ResponseWriter, req *http.Request) {
	// Extract parameters from URL
	vars := mux.Vars(req)
	kindStr := vars["kind"]
	pubkey := vars["pubkey"]
	dTag := vars["d_tag"]
	versionStr := vars["version"]

	// Parse parameters
	kind, err := strconv.Atoi(kindStr)
	if err != nil {
		http.Error(w, "Invalid kind parameter", http.StatusBadRequest)
		return
	}

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		http.Error(w, "Invalid version parameter", http.StatusBadRequest)
		return
	}

	// Get history
	history, err := r.cache.GetReplaceableEventHistory(kind, pubkey, dTag)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get event history: %v", err), http.StatusInternalServerError)
		return
	}

	// Find the specific version
	var targetVersion map[string]interface{}
	for _, v := range history {
		if v["version"] == float64(version) {
			targetVersion = v
			break
		}
	}

	if targetVersion == nil {
		http.Error(w, "Version not found", http.StatusNotFound)
		return
	}

	// Get the actual event
	_, ok := targetVersion["event_id"].(string)
	if !ok {
		http.Error(w, "Invalid event ID in version", http.StatusInternalServerError)
		return
	}

	// Get event from cache (this would need to be implemented in cache interface)
	// For now, return the version metadata
	response := map[string]interface{}{
		"success": true,
		"key":     fmt.Sprintf("%d:%s:%s", kind, pubkey, dTag),
		"version": targetVersion,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// calculateEventDiff calculates the difference between two event versions
func (r *RESTAPIServer) calculateEventDiff(fromEvent, toEvent map[string]interface{}) map[string]interface{} {
	diff := map[string]interface{}{
		"changes":  make(map[string]interface{}),
		"added":    []string{},
		"removed":  []string{},
		"modified": []string{},
	}

	// Compare fields
	fields := []string{"event_id", "created_at", "hash"}

	for _, field := range fields {
		fromVal, fromExists := fromEvent[field]
		toVal, toExists := toEvent[field]

		if !fromExists && toExists {
			// Added
			diff["added"] = append(diff["added"].([]string), field)
			diff["changes"].(map[string]interface{})[field] = map[string]interface{}{
				"from": nil,
				"to":   toVal,
			}
		} else if fromExists && !toExists {
			// Removed
			diff["removed"] = append(diff["removed"].([]string), field)
			diff["changes"].(map[string]interface{})[field] = map[string]interface{}{
				"from": fromVal,
				"to":   nil,
			}
		} else if fromExists && toExists {
			// Check if modified
			if fmt.Sprintf("%v", fromVal) != fmt.Sprintf("%v", toVal) {
				diff["modified"] = append(diff["modified"].([]string), field)
				diff["changes"].(map[string]interface{})[field] = map[string]interface{}{
					"from": fromVal,
					"to":   toVal,
				}
			}
		}
	}

	return diff
}

// HandleEventHistoryByID handles requests for event history by event ID
func (r *RESTAPIServer) HandleEventHistoryByID(w http.ResponseWriter, req *http.Request) {
	// Extract event ID from URL
	vars := mux.Vars(req)
	_ = vars["event_id"] // TODO: Use event ID to determine kind, pubkey, and d-tag

	// Get event from cache to determine kind, pubkey, and d-tag
	// This would need to be implemented in the cache interface
	// For now, return an error
	http.Error(w, "Event history by ID not yet implemented", http.StatusNotImplemented)
}

// HandleEventDiffByID handles requests for event diff by event IDs
func (r *RESTAPIServer) HandleEventDiffByID(w http.ResponseWriter, req *http.Request) {
	// Extract event IDs from URL
	vars := mux.Vars(req)
	_ = vars["from_event_id"] // TODO: Use to get source event
	_ = vars["to_event_id"]   // TODO: Use to get target event

	// Get events from cache
	// This would need to be implemented in the cache interface
	// For now, return an error
	http.Error(w, "Event diff by ID not yet implemented", http.StatusNotImplemented)
}
