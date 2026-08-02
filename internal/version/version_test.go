package version

import "testing"

func TestCurrentPrefersExplicitValue(t *testing.T) {
	original := Value
	t.Cleanup(func() { Value = original })
	Value = "v0.1.0-test"
	if got := Current(); got != Value {
		t.Fatalf("Current() = %q, want %q", got, Value)
	}
}
