// Package triage turns TagLock diagnostics into deterministic, dashboard-ready summaries.
package triage

import (
	"sort"

	"github.com/theworker02/taglock/rule"
	"github.com/theworker02/taglock/rules"
)

// RuleCount records how often a stable TagLock rule appeared.
type RuleCount struct {
	Rule  string `json:"rule"`
	Count int    `json:"count"`
}

// Summary is an aggregate view of a set of TagLock diagnostics.
type Summary struct {
	Total       int            `json:"total"`
	Score       int            `json:"score"`
	Fixable     int            `json:"fixable"`
	SafeFixes   int            `json:"safe_fixes"`
	ReviewFixes int            `json:"review_fixes"`
	BySeverity  map[string]int `json:"by_severity"`
	ByRule      map[string]int `json:"by_rule"`
	ByCategory  map[string]int `json:"by_category"`
	ByNamespace map[string]int `json:"by_namespace"`
}

// Summarize aggregates diagnostics without changing their order or content.
func Summarize(diagnostics []rules.Diagnostic) Summary {
	summary := Summary{
		BySeverity:  map[string]int{},
		ByRule:      map[string]int{},
		ByCategory:  map[string]int{},
		ByNamespace: map[string]int{},
	}

	for _, diagnostic := range diagnostics {
		summary.Total++
		summary.Score += severityWeight(diagnostic.Severity)
		summary.BySeverity[diagnostic.Severity.String()]++
		summary.ByRule[diagnostic.Rule.ID]++
		summary.ByCategory[diagnostic.Rule.Category]++
		if diagnostic.Namespace != "" {
			summary.ByNamespace[diagnostic.Namespace]++
		}
		if diagnostic.Rule.CanFix {
			summary.Fixable++
		}
		switch diagnostic.Rule.FixSafety {
		case rule.FixSafe:
			summary.SafeFixes++
		case rule.FixReview:
			summary.ReviewFixes++
		}
	}

	return summary
}

// FailsAt reports whether the summary contains a diagnostic at or above threshold.
func (s Summary) FailsAt(threshold rule.Severity) bool {
	if threshold == rule.SeverityOff {
		return false
	}
	for severity := threshold; severity <= rule.SeverityError; severity++ {
		if s.BySeverity[severity.String()] > 0 {
			return true
		}
	}
	return false
}

// TopRules returns the most frequent rule identifiers, with stable tie-breaking.
func (s Summary) TopRules(limit int) []RuleCount {
	if limit <= 0 {
		return nil
	}
	result := make([]RuleCount, 0, len(s.ByRule))
	for id, count := range s.ByRule {
		result = append(result, RuleCount{Rule: id, Count: count})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}
		return result[i].Rule < result[j].Rule
	})
	if len(result) > limit {
		result = result[:limit]
	}
	return result
}

func severityWeight(severity rule.Severity) int {
	switch severity {
	case rule.SeverityInfo:
		return 1
	case rule.SeverityWarning:
		return 4
	case rule.SeverityError:
		return 12
	default:
		return 0
	}
}
