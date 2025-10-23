package queue

import (
	"os"
	"testing"
)

func TestLoadKindsFromConfig(t *testing.T) {
	// Test with existing config file
	kinds, err := loadKindsFromConfig("configs/nostr-event-kinds.yaml")
	if err != nil {
		t.Fatalf("Failed to load kinds from config: %v", err)
	}

	// Should have many kinds from the config file
	if len(kinds) == 0 {
		t.Error("Expected to load kinds from config file, got empty list")
	}

	// Check for some expected kinds (not all, since config may have more)
	expectedKinds := []int{0, 1, 10002} // Basic kinds that should definitely be there
	foundKinds := make(map[int]bool)
	for _, kind := range kinds {
		foundKinds[kind] = true
	}

	for _, expectedKind := range expectedKinds {
		if !foundKinds[expectedKind] {
			t.Errorf("Expected kind %d to be loaded from config", expectedKind)
		}
	}

	t.Logf("Loaded %d kinds from config: %v", len(kinds), kinds)
}

func TestLoadKindsFromConfigFallback(t *testing.T) {
	// Test with non-existent file (should fallback to hardcoded)
	kinds, err := loadKindsFromConfig("non-existent-file.yaml")
	if err != nil {
		t.Fatalf("Expected no error for non-existent file, got: %v", err)
	}

	// Should fallback to hardcoded kinds
	expectedKinds := []int{0, 1, 3, 7, 10002}
	if len(kinds) != len(expectedKinds) {
		t.Errorf("Expected %d fallback kinds, got %d", len(expectedKinds), len(kinds))
	}

	for _, expectedKind := range expectedKinds {
		found := false
		for _, kind := range kinds {
			if kind == expectedKind {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected fallback kind %d not found", expectedKind)
		}
	}
}

func TestGetCommonKinds(t *testing.T) {
	kinds := getCommonKinds()

	if len(kinds) == 0 {
		t.Error("Expected to get some kinds, got empty list")
	}

	t.Logf("getCommonKinds() returned: %v", kinds)
}

func TestIsCommonKind(t *testing.T) {
	tests := []struct {
		kind     int
		expected bool
	}{
		{0, true},      // User metadata
		{1, true},      // Text note
		{3, true},      // Follow list
		{7, true},      // Reaction
		{10002, true},  // User's relay list
		{4, false},     // Enrypted direct message
		{10000, false}, // Mute list
		{999, false},   // Not in config
	}

	for _, test := range tests {
		result := isCommonKind(test.kind)
		if result != test.expected {
			t.Errorf("isCommonKind(%d) = %v, expected %v", test.kind, result, test.expected)
		}
	}
}

func TestDynamicKindIntegration(t *testing.T) {
	// Test that adding a new kind to the config file would be picked up
	// This is more of an integration test

	// First, let's see what kinds are currently configured
	kinds := getCommonKinds()
	t.Logf("Currently configured kinds: %v", kinds)

	// Test that we can handle a kind that's in the config
	if len(kinds) > 0 {
		testKind := kinds[0]
		if !isCommonKind(testKind) {
			t.Errorf("Kind %d should be considered common", testKind)
		}
	}

	// Test that undefined kinds are handled correctly
	undefinedKind := 99999
	if isCommonKind(undefinedKind) {
		t.Errorf("Kind %d should not be considered common", undefinedKind)
	}
}

func TestConfigFileExists(t *testing.T) {
	// Test that the config file actually exists
	if _, err := os.Stat("configs/nostr-event-kinds.yaml"); os.IsNotExist(err) {
		t.Skip("Config file configs/nostr-event-kinds.yaml does not exist, skipping test")
	}

	// Test loading from the actual config file
	kinds, err := loadKindsFromConfig("configs/nostr-event-kinds.yaml")
	if err != nil {
		t.Fatalf("Failed to load from actual config file: %v", err)
	}

	if len(kinds) < 5 {
		t.Errorf("Expected at least 5 kinds from config file, got %d", len(kinds))
	}

	t.Logf("Successfully loaded %d kinds from config file", len(kinds))
}
