package helpers

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// AssertNoError checks that err is nil
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

// AssertError checks that err is not nil
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

// AssertErrorContains checks that err contains the specified substring
func AssertErrorContains(t *testing.T, err error, substring string) {
	t.Helper()
	if err == nil {
		t.Fatalf("Expected error containing '%s', got nil", substring)
	}
	if !strings.Contains(err.Error(), substring) {
		t.Fatalf("Expected error to contain '%s', got: %v", substring, err)
	}
}

// AssertEqual checks that expected equals actual
func AssertEqual(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if expected != actual {
		t.Fatalf("Expected %v, got %v", expected, actual)
	}
}

// AssertNotEqual checks that expected does not equal actual
func AssertNotEqual(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if expected == actual {
		t.Fatalf("Expected %v to not equal %v", expected, actual)
	}
}

// AssertTrue checks that condition is true
func AssertTrue(t *testing.T, condition bool) {
	t.Helper()
	if !condition {
		t.Fatal("Expected true, got false")
	}
}

// AssertFalse checks that condition is false
func AssertFalse(t *testing.T, condition bool) {
	t.Helper()
	if condition {
		t.Fatal("Expected false, got true")
	}
}

// AssertIntEqual checks that two integers are equal
func AssertIntEqual(t *testing.T, expected, actual int) {
	t.Helper()
	if expected != actual {
		t.Fatalf("Expected %d, got %d", expected, actual)
	}
}

// AssertInt64Equal checks that two int64s are equal
func AssertInt64Equal(t *testing.T, expected, actual int64) {
	t.Helper()
	if expected != actual {
		t.Fatalf("Expected %d, got %d", expected, actual)
	}
}

// AssertFloat64Equal checks that two float64s are equal within epsilon
func AssertFloat64Equal(t *testing.T, expected, actual, epsilon float64) {
	t.Helper()
	if abs(expected-actual) > epsilon {
		t.Fatalf("Expected %f, got %f (epsilon: %f)", expected, actual, epsilon)
	}
}

// AssertStringEqual checks that two strings are equal
func AssertStringEqual(t *testing.T, expected, actual string) {
	t.Helper()
	if expected != actual {
		t.Fatalf("Expected %s, got %s", expected, actual)
	}
}

// AssertDurationEqual checks that two durations are equal
func AssertDurationEqual(t *testing.T, expected, actual time.Duration) {
	t.Helper()
	if expected != actual {
		t.Fatalf("Expected %v, got %v", expected, actual)
	}
}

// AssertStringContains checks that str contains substr
func AssertStringContains(t *testing.T, str, substr string) {
	t.Helper()
	if !strings.Contains(str, substr) {
		t.Fatalf("Expected '%s' to contain '%s'", str, substr)
	}
}

// AssertBoolEqual checks that two booleans are equal
func AssertBoolEqual(t *testing.T, expected, actual bool) {
	t.Helper()
	if expected != actual {
		t.Fatalf("Expected %t, got %t", expected, actual)
	}
}

// AssertNil checks that value is nil
func AssertNil(t *testing.T, value interface{}) {
	t.Helper()
	if value != nil {
		t.Fatalf("Expected nil, got %v", value)
	}
}

// AssertNotNil checks that value is not nil
func AssertNotNil(t *testing.T, value interface{}) {
	t.Helper()
	if value == nil {
		t.Fatal("Expected non-nil value, got nil")
	}
}

// AssertLen checks that slice/array has expected length
func AssertLen(t *testing.T, slice interface{}, expected int) {
	t.Helper()
	actual := reflect.ValueOf(slice).Len()
	if actual != expected {
		t.Fatalf("Expected length %d, got %d", expected, actual)
	}
}

// AssertEmpty checks that slice/array is empty
func AssertEmpty(t *testing.T, slice interface{}) {
	t.Helper()
	length := reflect.ValueOf(slice).Len()
	if length != 0 {
		t.Fatalf("Expected empty slice, got length %d", length)
	}
}

// AssertNotEmpty checks that slice/array is not empty
func AssertNotEmpty(t *testing.T, slice interface{}) {
	t.Helper()
	length := reflect.ValueOf(slice).Len()
	if length == 0 {
		t.Fatal("Expected non-empty slice, got empty")
	}
}

// AssertContains checks that slice contains the specified element
func AssertContains(t *testing.T, slice interface{}, element interface{}) {
	t.Helper()
	sliceValue := reflect.ValueOf(slice)
	if sliceValue.Kind() != reflect.Slice && sliceValue.Kind() != reflect.Array {
		t.Fatalf("Expected slice or array, got %T", slice)
	}

	for i := 0; i < sliceValue.Len(); i++ {
		if reflect.DeepEqual(sliceValue.Index(i).Interface(), element) {
			return
		}
	}

	t.Fatalf("Expected slice to contain %v", element)
}

// AssertNotContains checks that slice does not contain the specified element
func AssertNotContains(t *testing.T, slice interface{}, element interface{}) {
	t.Helper()
	sliceValue := reflect.ValueOf(slice)
	if sliceValue.Kind() != reflect.Slice && sliceValue.Kind() != reflect.Array {
		t.Fatalf("Expected slice or array, got %T", slice)
	}

	for i := 0; i < sliceValue.Len(); i++ {
		if reflect.DeepEqual(sliceValue.Index(i).Interface(), element) {
			t.Fatalf("Expected slice to not contain %v", element)
		}
	}
}

// AssertQualityScore checks that event's quality score is within expected range
func AssertQualityScore(t *testing.T, event interface{}, min, max float64) {
	t.Helper()
	// Use reflection to access QualityScore field
	eventValue := reflect.ValueOf(event)
	if eventValue.Kind() == reflect.Ptr {
		eventValue = eventValue.Elem()
	}

	qualityScoreField := eventValue.FieldByName("QualityScore")
	if !qualityScoreField.IsValid() {
		t.Fatalf("Event does not have QualityScore field")
	}

	qualityScore := qualityScoreField.Float()
	if qualityScore < min || qualityScore > max {
		t.Fatalf("Expected quality score between %f and %f, got %f", min, max, qualityScore)
	}
}

// AssertEventQuarantined checks that event's quarantine status matches expected value
func AssertEventQuarantined(t *testing.T, event interface{}, expected bool) {
	t.Helper()
	// Use reflection to access IsQuarantined field
	eventValue := reflect.ValueOf(event)
	if eventValue.Kind() == reflect.Ptr {
		eventValue = eventValue.Elem()
	}

	quarantinedField := eventValue.FieldByName("IsQuarantined")
	if !quarantinedField.IsValid() {
		t.Fatalf("Event does not have IsQuarantined field")
	}

	quarantined := quarantinedField.Bool()
	if quarantined != expected {
		t.Fatalf("Expected quarantined=%t, got quarantined=%t", expected, quarantined)
	}
}

// Helper functions
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func timesEqual(t1, t2 time.Time, tolerance time.Duration) bool {
	diff := t1.Sub(t2)
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}
