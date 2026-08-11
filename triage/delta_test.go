package triage

import (
	"reflect"
	"testing"

	"github.com/theworker02/taglock/rule"
)

func TestCompare(t *testing.T) {
	previous := Summary{
		Score:      5,
		BySeverity: map[string]int{"warning": 1, "error": 0},
		ByRule:     map[string]int{"TAG003": 1},
	}
	current := Summary{
		Score:      17,
		BySeverity: map[string]int{"warning": 1, "error": 1},
		ByRule:     map[string]int{"TAG003": 1, "TAG104": 1},
	}

	delta := Compare(previous, current)
	if delta.ScoreDelta != 12 {
		t.Fatalf("score delta = %d, want 12", delta.ScoreDelta)
	}
	if delta.SeverityDelta["error"] != 1 {
		t.Fatalf("severity delta = %#v", delta.SeverityDelta)
	}
	if len(delta.AddedRules) != 1 || delta.AddedRules[0].Rule != "TAG104" {
		t.Fatalf("added rules = %#v", delta.AddedRules)
	}
	if len(delta.ResolvedRules) != 0 {
		t.Fatalf("resolved rules = %#v", delta.ResolvedRules)
	}
	if !delta.IntroducesAt(rule.SeverityError) {
		t.Fatal("expected error introduction")
	}
	if delta.Clean() {
		t.Fatal("non-empty delta reported clean")
	}
}

func TestCompareResolvedRules(t *testing.T) {
	previous := Summary{ByRule: map[string]int{"TAG003": 2, "TAG104": 1}}
	current := Summary{ByRule: map[string]int{"TAG003": 2}}

	delta := Compare(previous, current)
	want := []RuleCount{{Rule: "TAG104", Count: 1}}
	if got := delta.ResolvedRules; !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolvedRules = %#v, want %#v", got, want)
	}
}

func TestCompareClean(t *testing.T) {
	summary := Summary{
		Score:      4,
		BySeverity: map[string]int{"warning": 1},
		ByRule:     map[string]int{"TAG003": 1},
	}
	delta := Compare(summary, summary)
	if !delta.Clean() {
		t.Fatalf("identical summaries produced %#v", delta)
	}
	if delta.IntroducesAt(rule.SeverityWarning) {
		t.Fatal("identical summaries introduced findings")
	}
}
