package triage

import (
	"reflect"
	"testing"

	"github.com/theworker02/taglock/rule"
	"github.com/theworker02/taglock/rules"
)

func TestSummarize(t *testing.T) {
	tag003, _ := rule.Lookup("TAG003")
	tag104, _ := rule.Lookup("TAG104")
	tag401, _ := rule.Lookup("TAG401")

	summary := Summarize([]rules.Diagnostic{
		{Rule: tag003, Severity: rule.SeverityWarning, Namespace: "json"},
		{Rule: tag104, Severity: rule.SeverityError, Namespace: "json"},
		{Rule: tag104, Severity: rule.SeverityError, Namespace: "yaml"},
		{Rule: tag401, Severity: rule.SeverityError, Namespace: "json"},
	})

	if summary.Total != 4 || summary.Score != 40 {
		t.Fatalf("unexpected totals: %#v", summary)
	}
	if summary.Fixable != 4 || summary.SafeFixes != 1 || summary.ReviewFixes != 3 {
		t.Fatalf("unexpected remediation counts: %#v", summary)
	}
	if summary.ByNamespace["json"] != 3 || summary.ByCategory["security"] != 1 {
		t.Fatalf("unexpected grouping: %#v", summary)
	}
	if !summary.FailsAt(rule.SeverityError) || summary.FailsAt(rule.SeverityOff) {
		t.Fatalf("unexpected threshold result")
	}

	want := []RuleCount{{Rule: "TAG104", Count: 2}, {Rule: "TAG003", Count: 1}}
	if got := summary.TopRules(2); !reflect.DeepEqual(got, want) {
		t.Fatalf("TopRules() = %#v, want %#v", got, want)
	}
}

func TestTopRulesUsesStableTieBreaks(t *testing.T) {
	summary := Summary{ByRule: map[string]int{"TAG401": 1, "TAG003": 1}}
	want := []RuleCount{{Rule: "TAG003", Count: 1}, {Rule: "TAG401", Count: 1}}
	if got := summary.TopRules(10); !reflect.DeepEqual(got, want) {
		t.Fatalf("TopRules() = %#v, want %#v", got, want)
	}
	if got := summary.TopRules(0); got != nil {
		t.Fatalf("TopRules(0) = %#v, want nil", got)
	}
}
