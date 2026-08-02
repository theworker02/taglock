package config_test

import (
	"strings"
	"testing"

	"github.com/magnexis/taglock/config"
)

func TestDecodeOverlaysDefaults(t *testing.T) {
	cfg, err := config.Decode(strings.NewReader(`
namespaces: [json, yaml]
naming:
  json: snake_case
require_explicit_json_tags: true
disallow_unknown_options: false
`))
	if err != nil {
		t.Fatalf("Decode returned an error: %v", err)
	}
	if !cfg.RequireExplicitJSON || cfg.DisallowUnknownOptions {
		t.Fatalf("boolean overrides were not preserved: %#v", cfg)
	}
	if got := cfg.SensitiveFields; len(got) == 0 {
		t.Fatal("default sensitive fields were not retained")
	}
}

func TestDecodeRejectsUnknownKeys(t *testing.T) {
	_, err := config.Decode(strings.NewReader("namespaces: [json]\nmade_up: true\n"))
	if err == nil || !strings.Contains(err.Error(), "made_up") {
		t.Fatalf("expected an unknown-key error, got %v", err)
	}
}

func TestDecodeRejectsInvalidNamespace(t *testing.T) {
	_, err := config.Decode(strings.NewReader("namespaces: [json, 'bad name']\n"))
	if err == nil || !strings.Contains(err.Error(), "invalid namespace") {
		t.Fatalf("expected an invalid-namespace error, got %v", err)
	}
}

func TestDecodePhase3PolicyAndOverrides(t *testing.T) {
	cfg, err := config.Decode(strings.NewReader(`
version: 1
namespaces:
  json:
    enabled: true
json:
  semantics: both
contracts:
  default_level: public
  visibility: [exported]
compatibility:
  default_mode: bidirectional
  fail_on: [breaking, unknown]
evolution:
  require_deprecation_before_removal: true
snapshot:
  output: .taglock/contracts.json
  reproducible: true
schema:
  enabled: true
  format: json-schema
  output: schemas/contracts.json
overrides:
  - files: ["internal/legacy/**"]
    rules:
      disable: [TAG301]
`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.JSON.Semantics != "both" || !cfg.Snapshot.Reproducible {
		t.Fatalf("policy not decoded: %#v", cfg)
	}
	effective := cfg.Effective("internal/legacy/model.go", "example/internal/legacy")
	if effective.RuleEnabled("TAG301") {
		t.Fatal("file override did not disable TAG301")
	}
}

func TestDecodeRejectsUnknownProfile(t *testing.T) {
	_, err := config.Decode(strings.NewReader("version: 1\njson:\n  semantics: future\n"))
	if err == nil || !strings.Contains(err.Error(), "json.semantics") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecodeValidatesReleaseHistoryAndGenericArguments(t *testing.T) {
	_, err := config.Decode(strings.NewReader(`version: 1
evolution:
  release_history: [1.5.0, 1.4.0]
`))
	if err == nil || !strings.Contains(err.Error(), "strictly increasing") {
		t.Fatalf("unexpected release history error: %v", err)
	}
	_, err = config.Decode(strings.NewReader(`version: 1
verification:
  fixtures:
    Box:
      type_arguments: ["[]"]
`))
	if err == nil || !strings.Contains(err.Error(), "type argument") {
		t.Fatalf("unexpected type argument error: %v", err)
	}
}
