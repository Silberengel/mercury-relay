package models

import (
	"encoding/json"
	"fmt"
	"mercury-relay/test/helpers"
	"strings"
	"testing"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

// Test helper functions
func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func assertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

func assertErrorContains(t *testing.T, err error, substring string) {
	t.Helper()
	if err == nil {
		t.Fatalf("Expected error containing '%s', got nil", substring)
	}
	if !strings.Contains(err.Error(), substring) {
		t.Fatalf("Expected error to contain '%s', got: %v", substring, err)
	}
}

func assertEqual(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if expected != actual {
		t.Fatalf("Expected %v, got %v", expected, actual)
	}
}

func assertTrue(t *testing.T, condition bool) {
	t.Helper()
	if !condition {
		t.Fatal("Expected true, got false")
	}
}

func assertFalse(t *testing.T, condition bool) {
	t.Helper()
	if condition {
		t.Fatal("Expected false, got true")
	}
}

func assertQualityScore(t *testing.T, event *Event, min, max float64) {
	t.Helper()
	if event.QualityScore < min || event.QualityScore > max {
		t.Fatalf("Expected quality score between %f and %f, got %f", min, max, event.QualityScore)
	}
}

func assertEventIsSpam(t *testing.T, event *Event, threshold float64, expected bool) {
	t.Helper()
	isSpam := event.QualityScore < threshold
	if isSpam != expected {
		t.Fatalf("Expected spam=%t, got spam=%t (quality score: %f, threshold: %f)", expected, isSpam, event.QualityScore, threshold)
	}
}

func assertEventQuarantined(t *testing.T, event *Event, expected bool) {
	t.Helper()
	quarantined := event.IsQuarantined
	if quarantined != expected {
		t.Fatalf("Expected quarantined=%t, got quarantined=%t", expected, quarantined)
	}
}

func TestEventValidation(t *testing.T) {
	eg := NewEventGenerator()

	t.Run("Valid complete event", func(t *testing.T) {
		event := eg.GenerateTextNote(eg.GetRandomNpub(), "Test content", nostr.Tags{})
		err := event.Validate()
		assertNoError(t, err)
	})

	t.Run("Event too old", func(t *testing.T) {
		event := eg.GenerateTextNote(eg.GetRandomNpub(), "Test content", nostr.Tags{})
		event.CreatedAt = time.Now().Add(-2 * time.Hour) // 2 hours ago
		err := event.Validate()
		assertError(t, err)
		assertErrorContains(t, err, "too old")
	})

	t.Run("Event in future", func(t *testing.T) {
		event := eg.GenerateTextNote(eg.GetRandomNpub(), "Test content", nostr.Tags{})
		event.CreatedAt = time.Now().Add(10 * time.Minute) // 10 minutes in future
		err := event.Validate()
		assertError(t, err)
		assertErrorContains(t, err, "future")
	})

	t.Run("Content too long", func(t *testing.T) {
		event := eg.GenerateTextNote(eg.GetRandomNpub(), "Test content", nostr.Tags{})
		// Create very long content
		longContent := make([]byte, 10001)
		for i := range longContent {
			longContent[i] = 'a'
		}
		event.Content = string(longContent)
		err := event.Validate()
		assertError(t, err)
		assertErrorContains(t, err, "too long")
	})

	t.Run("Missing required fields", func(t *testing.T) {
		event := &Event{
			ID:        "",
			PubKey:    "",
			CreatedAt: time.Now(),
			Kind:      1,
			Tags:      nostr.Tags{},
			Content:   "test",
			Sig:       "",
		}
		err := event.Validate()
		assertError(t, err)
		assertErrorContains(t, err, "required fields")
	})
}

func TestQualityScoreCalculation(t *testing.T) {
	eg := NewEventGenerator()

	t.Run("Optimal content", func(t *testing.T) {
		content := "This is a well-written post with meaningful content that provides value to the community. " +
			"It contains thoughtful insights and relevant information that contributes to the conversation."
		event := eg.GenerateTextNote(eg.GetRandomNpub(), content, nostr.Tags{
			[]string{"t", "quality"},
			[]string{"t", "meaningful"},
		})

		assertQualityScore(t, event, 0.9, 1.0)
	})

	t.Run("Poor quality event", func(t *testing.T) {
		// Create event with very short content and many tags
		var tags nostr.Tags
		for i := 0; i < 25; i++ {
			tags = append(tags, []string{"t", "spam"})
		}

		event := eg.GenerateTextNote(eg.GetRandomNpub(), "spam", tags)

		assertQualityScore(t, event, 0.0, 0.5)
	})

	t.Run("Edge cases", func(t *testing.T) {
		// Empty content, no tags
		event := eg.GenerateTextNote(eg.GetRandomNpub(), "", nostr.Tags{})
		assertQualityScore(t, event, 0.0, 1.0)

		// Very long content
		longContent := make([]byte, 6000)
		for i := range longContent {
			longContent[i] = 'a'
		}
		event = eg.GenerateTextNote(eg.GetRandomNpub(), string(longContent), nostr.Tags{})
		assertQualityScore(t, event, 0.0, 1.0)
	})
}

func TestEventSerialization(t *testing.T) {
	eg := NewEventGenerator()

	t.Run("To/From Nostr event", func(t *testing.T) {
		original := eg.GenerateTextNote(eg.GetRandomNpub(), "Test content", nostr.Tags{})

		// Convert to nostr.Event
		nostrEvent := original.ToNostrEvent()

		// Convert back to models.Event
		converted := FromNostrEvent(nostrEvent)

		// Compare key fields
		helpers.AssertStringEqual(t, original.ID, converted.ID)
		helpers.AssertStringEqual(t, original.PubKey, converted.PubKey)
		helpers.AssertIntEqual(t, original.Kind, converted.Kind)
		helpers.AssertStringEqual(t, original.Content, converted.Content)
		helpers.AssertStringEqual(t, original.Sig, converted.Sig)
	})

	t.Run("JSON marshaling", func(t *testing.T) {
		original := eg.GenerateTextNote(eg.GetRandomNpub(), "Test content", nostr.Tags{})

		// Marshal to JSON
		jsonData, err := json.Marshal(original)
		assertNoError(t, err)

		// Unmarshal back
		var unmarshaled Event
		err = json.Unmarshal(jsonData, &unmarshaled)
		assertNoError(t, err)

		// Compare timestamps (should be handled correctly)
		helpers.AssertInt64Equal(t, original.CreatedAt.Unix(), unmarshaled.CreatedAt.Unix())
		helpers.AssertStringEqual(t, original.ID, unmarshaled.ID)
		helpers.AssertStringEqual(t, original.PubKey, unmarshaled.PubKey)
	})
}

func TestSpamDetection(t *testing.T) {
	eg := NewEventGenerator()

	t.Run("Normal content", func(t *testing.T) {
		event := eg.GenerateHighQualityEvent(eg.GetRandomNpub())
		assertEventIsSpam(t, event, 0.7, false)
	})

	t.Run("Spam content", func(t *testing.T) {
		event := eg.GenerateSpamEvent(eg.GetRandomNpub())
		assertEventIsSpam(t, event, 0.7, true)
	})

	t.Run("Borderline content", func(t *testing.T) {
		// Create event with medium quality - short content with many tags to lower quality score
		content := "OK"
		var tags nostr.Tags
		for i := 0; i < 10; i++ {
			tags = append(tags, []string{"t", fmt.Sprintf("tag%d", i)})
		}
		event := eg.GenerateTextNote(eg.GetRandomNpub(), content, tags)

		// Test with different thresholds
		assertEventIsSpam(t, event, 0.9, true)  // High threshold
		assertEventIsSpam(t, event, 0.1, false) // Low threshold
	})
}

func TestEventQuarantine(t *testing.T) {
	eg := NewEventGenerator()

	t.Run("High quality event not quarantined", func(t *testing.T) {
		event := eg.GenerateHighQualityEvent(eg.GetRandomNpub())
		event.QualityScore = 0.9
		event.IsQuarantined = event.IsSpam(0.7)

		assertEventQuarantined(t, event, false)
		helpers.AssertStringEqual(t, event.QuarantineReason, "")
	})

	t.Run("Low quality event quarantined", func(t *testing.T) {
		event := eg.GenerateSpamEvent(eg.GetRandomNpub())
		event.QualityScore = 0.3
		event.IsQuarantined = event.IsSpam(0.7)
		event.QuarantineReason = "Low quality score"

		assertEventQuarantined(t, event, true)
		helpers.AssertStringEqual(t, event.QuarantineReason, "Low quality score")
	})
}

func TestEventErrorDefinitions(t *testing.T) {
	t.Run("Error types", func(t *testing.T) {
		// Test that error variables are properly defined
		helpers.AssertNotNil(t, ErrEventTooOld)
		helpers.AssertNotNil(t, ErrEventInFuture)
		helpers.AssertNotNil(t, ErrContentTooLong)
		helpers.AssertNotNil(t, ErrMissingRequiredFields)

		// Test error messages
		helpers.AssertStringEqual(t, "event is too old", ErrEventTooOld.Error())
		helpers.AssertStringEqual(t, "event is in the future", ErrEventInFuture.Error())
		helpers.AssertStringEqual(t, "content is too long", ErrContentTooLong.Error())
		helpers.AssertStringEqual(t, "missing required fields", ErrMissingRequiredFields.Error())
	})
}

func TestEventQualityScoreEdgeCases(t *testing.T) {
	eg := NewEventGenerator()

	t.Run("Very short content penalty", func(t *testing.T) {
		event := eg.GenerateTextNote(eg.GetRandomNpub(), "hi", nostr.Tags{})
		score := event.CalculateQualityScore()

		// Should be penalized for very short content
		if score >= 1.0 {
			t.Errorf("Expected penalty for short content, got score: %.2f", score)
		}
	})

	t.Run("Very long content penalty", func(t *testing.T) {
		longContent := make([]byte, 6000)
		for i := range longContent {
			longContent[i] = 'a'
		}
		event := eg.GenerateTextNote(eg.GetRandomNpub(), string(longContent), nostr.Tags{})
		score := event.CalculateQualityScore()

		// Should be penalized for very long content
		if score >= 1.0 {
			t.Errorf("Expected penalty for long content, got score: %.2f", score)
		}
	})

	t.Run("Too many tags penalty", func(t *testing.T) {
		var tags nostr.Tags
		for i := 0; i < 25; i++ {
			tags = append(tags, []string{"t", "tag"})
		}
		event := eg.GenerateTextNote(eg.GetRandomNpub(), "Normal content", tags)
		score := event.CalculateQualityScore()

		// Should be penalized for too many tags
		if score >= 1.0 {
			t.Errorf("Expected penalty for too many tags, got score: %.2f", score)
		}
	})

	t.Run("Reasonable tag count bonus", func(t *testing.T) {
		tags := nostr.Tags{
			[]string{"t", "quality"},
			[]string{"t", "test"},
		}
		event := eg.GenerateTextNote(eg.GetRandomNpub(), "Good content", tags)
		score := event.CalculateQualityScore()

		// Should get bonus for reasonable tag count
		if score < 1.0 {
			t.Errorf("Expected bonus for reasonable tags, got score: %.2f", score)
		}
	})

	t.Run("Score bounds", func(t *testing.T) {
		// Test that score is always between 0 and 1
		event := eg.GenerateTextNote(eg.GetRandomNpub(), "Test", nostr.Tags{})
		score := event.CalculateQualityScore()

		if score < 0 || score > 1 {
			t.Errorf("Score should be between 0 and 1, got: %.2f", score)
		}
	})
}
