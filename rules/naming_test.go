package rules_test

import (
	"testing"

	"github.com/magnexis/taglock/rules"
)

func TestConvertName(t *testing.T) {
	tests := map[string]string{"DisplayName": "display_name", "APIKey": "api_key", "already_snake": "already_snake", "access-token": "access_token"}
	for input, expected := range tests {
		if got := rules.ConvertName(input, "snake_case"); got != expected {
			t.Errorf("ConvertName(%q)=%q, want %q", input, got, expected)
		}
	}
}
