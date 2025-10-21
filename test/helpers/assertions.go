package helpers

import (
	"reflect"
	"testing"
	"time"

	"mercury-relay/internal/models"

	"github.com/nbd-wtf/go-nostr"
)

// AssertEventEqual compares two events for equality
func AssertEventEqual(t *testing.T, expected, actual *models.Event, msgAndArgs ...interface{}) {
	if expected == nil && actual == nil {
		return
	}
	if expected == nil || actual == nil {
		t.Fatalf("One event is nil: expected=%v, actual=%v", expected != nil, actual != nil)
	}

	if expected.ID != actual.ID {
		t.Errorf("Event ID mismatch: expected=%s, actual=%s", expected.ID, actual.ID)
	}
	if expected.PubKey != actual.PubKey {
		t.Errorf("Event PubKey mismatch: expected=%s, actual=%s", expected.PubKey, actual.PubKey)
	}
	if expected.Kind != actual.Kind {
		t.Errorf("Event Kind mismatch: expected=%d, actual=%d", expected.Kind, actual.Kind)
	}
	if expected.Content != actual.Content {
		t.Errorf("Event Content mismatch: expected=%s, actual=%s", expected.Content, actual.Content)
	}
	if expected.Sig != actual.Sig {
		t.Errorf("Event Sig mismatch: expected=%s, actual=%s", expected.Sig, actual.Sig)
	}

	// Compare tags
	if !reflect.DeepEqual(expected.Tags, actual.Tags) {
		t.Errorf("Event Tags mismatch: expected=%v, actual=%v", expected.Tags, actual.Tags)
	}

	// Compare timestamps (allow 1 second difference)
	if !timesEqual(expected.CreatedAt, actual.CreatedAt, time.Second) {
		t.Errorf("Event CreatedAt mismatch: expected=%v, actual=%v", expected.CreatedAt, actual.CreatedAt)
	}
}

// AssertEventsEqual compares two slices of events
func AssertEventsEqual(t *testing.T, expected, actual []*models.Event, msgAndArgs ...interface{}) {
	if len(expected) != len(actual) {
		t.Fatalf("Event count mismatch: expected=%d, actual=%d", len(expected), len(actual))
	}

	for i := range expected {
		AssertEventEqual(t, expected[i], actual[i], msgAndArgs...)
	}
}

// AssertFilterMatches checks if events match a filter
func AssertFilterMatches(t *testing.T, events []*models.Event, filter nostr.Filter, expectedCount int, msgAndArgs ...interface{}) {
	matched := 0
	for _, event := range events {
		if matchesFilter(event, filter) {
			matched++
		}
	}

	if matched != expectedCount {
		t.Errorf("Filter match count mismatch: expected=%d, actual=%d, filter=%+v", expectedCount, matched, filter)
	}
}

// AssertQualityScore checks if event quality score is within expected range
func AssertQualityScore(t *testing.T, event *models.Event, min, max float64, msgAndArgs ...interface{}) {
	if event.QualityScore < min || event.QualityScore > max {
		t.Errorf("Quality score out of range: expected=[%.2f, %.2f], actual=%.2f", min, max, event.QualityScore)
	}
}

// AssertEventIsSpam checks if event is correctly identified as spam
func AssertEventIsSpam(t *testing.T, event *models.Event, threshold float64, expected bool, msgAndArgs ...interface{}) {
	actual := event.IsSpam(threshold)
	if actual != expected {
		t.Errorf("Spam detection mismatch: expected=%v, actual=%v, threshold=%.2f, score=%.2f",
			expected, actual, threshold, event.QualityScore)
	}
}

// AssertEventQuarantined checks if event is quarantined
func AssertEventQuarantined(t *testing.T, event *models.Event, expected bool, msgAndArgs ...interface{}) {
	if event.IsQuarantined != expected {
		t.Errorf("Quarantine status mismatch: expected=%v, actual=%v", expected, event.IsQuarantined)
	}
}

// AssertErrorContains checks if error contains expected text
func AssertErrorContains(t *testing.T, err error, expectedText string, msgAndArgs ...interface{}) {
	if err == nil {
		t.Errorf("Expected error containing '%s', but got nil", expectedText)
		return
	}

	if err.Error() != expectedText && !contains(err.Error(), expectedText) {
		t.Errorf("Error message mismatch: expected to contain '%s', actual='%s'", expectedText, err.Error())
	}
}

// AssertNoError checks that error is nil
func AssertNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}

// AssertError checks that error is not nil
func AssertError(t *testing.T, err error, msgAndArgs ...interface{}) {
	if err == nil {
		t.Errorf("Expected error, but got nil")
	}
}

// AssertIntEqual checks integer equality
func AssertIntEqual(t *testing.T, expected, actual int, msgAndArgs ...interface{}) {
	if expected != actual {
		t.Errorf("Integer mismatch: expected=%d, actual=%d", expected, actual)
	}
}

// AssertInt64Equal checks int64 equality
func AssertInt64Equal(t *testing.T, expected, actual int64, msgAndArgs ...interface{}) {
	if expected != actual {
		t.Errorf("Int64 mismatch: expected=%d, actual=%d", expected, actual)
	}
}

// AssertStringEqual checks string equality
func AssertStringEqual(t *testing.T, expected, actual string, msgAndArgs ...interface{}) {
	if expected != actual {
		t.Errorf("String mismatch: expected='%s', actual='%s'", expected, actual)
	}
}

// AssertBoolEqual checks boolean equality
func AssertBoolEqual(t *testing.T, expected, actual bool, msgAndArgs ...interface{}) {
	if expected != actual {
		t.Errorf("Boolean mismatch: expected=%v, actual=%v", expected, actual)
	}
}

// AssertFloat64Equal checks float64 equality with tolerance
func AssertFloat64Equal(t *testing.T, expected, actual, tolerance float64, msgAndArgs ...interface{}) {
	diff := expected - actual
	if diff < 0 {
		diff = -diff
	}
	if diff > tolerance {
		t.Errorf("Float64 mismatch: expected=%.6f, actual=%.6f, tolerance=%.6f", expected, actual, tolerance)
	}
}

// AssertNotNil checks that value is not nil
func AssertNotNil(t *testing.T, value interface{}, msgAndArgs ...interface{}) {
	if value == nil {
		t.Errorf("Expected non-nil value, but got nil")
	}
}

// AssertNil checks that value is nil
func AssertNil(t *testing.T, value interface{}, msgAndArgs ...interface{}) {
	if value != nil {
		t.Errorf("Expected nil value, but got %v", value)
	}
}

// AssertContains checks if slice contains element
func AssertContains(t *testing.T, slice interface{}, element interface{}, msgAndArgs ...interface{}) {
	sliceValue := reflect.ValueOf(slice)
	if sliceValue.Kind() != reflect.Slice {
		t.Errorf("Expected slice, but got %T", slice)
		return
	}

	for i := 0; i < sliceValue.Len(); i++ {
		if reflect.DeepEqual(sliceValue.Index(i).Interface(), element) {
			return
		}
	}

	t.Errorf("Expected slice to contain element %v, but it didn't", element)
}

// AssertNotContains checks if slice does not contain element
func AssertNotContains(t *testing.T, slice interface{}, element interface{}, msgAndArgs ...interface{}) {
	sliceValue := reflect.ValueOf(slice)
	if sliceValue.Kind() != reflect.Slice {
		t.Errorf("Expected slice, but got %T", slice)
		return
	}

	for i := 0; i < sliceValue.Len(); i++ {
		if reflect.DeepEqual(sliceValue.Index(i).Interface(), element) {
			t.Errorf("Expected slice to not contain element %v, but it did", element)
			return
		}
	}
}

// AssertMapContains checks if map contains key
func AssertMapContains(t *testing.T, m interface{}, key interface{}, msgAndArgs ...interface{}) {
	mapValue := reflect.ValueOf(m)
	if mapValue.Kind() != reflect.Map {
		t.Errorf("Expected map, but got %T", m)
		return
	}

	keyValue := reflect.ValueOf(key)
	if !mapValue.MapIndex(keyValue).IsValid() {
		t.Errorf("Expected map to contain key %v, but it didn't", key)
	}
}

// Helper functions

func timesEqual(t1, t2 time.Time, tolerance time.Duration) bool {
	diff := t1.Sub(t2)
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}

func matchesFilter(event *models.Event, filter nostr.Filter) bool {
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
		if nostr.Timestamp(event.CreatedAt.Unix()) < *filter.Since {
			return false
		}
	}

	// Check until
	if filter.Until != nil && *filter.Until > 0 {
		if nostr.Timestamp(event.CreatedAt.Unix()) > *filter.Until {
			return false
		}
	}

	return true
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && contains(s[1:], substr)
}
