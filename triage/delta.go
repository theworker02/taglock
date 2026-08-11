package triage

import (
	"sort"

	"github.com/theworker02/taglock/rule"
)

// Delta describes how a triage summary changed between two scans.
type Delta struct {
	ScoreDelta    int            `json:"score_delta"`
	AddedRules    []RuleCount    `json:"added_rules"`
	ResolvedRules []RuleCount    `json:"resolved_rules"`
	SeverityDelta map[string]int `json:"severity_delta"`
}

// Compare returns a deterministic delta between previous and current summaries.
func Compare(previous, current Summary) Delta {
	delta := Delta{
		ScoreDelta:    current.Score - previous.Score,
		SeverityDelta: map[string]int{},
	}

	severities := unionSeverityKeys(previous.BySeverity, current.BySeverity)
	for _, severity := range severities {
		change := current.BySeverity[severity] - previous.BySeverity[severity]
		if change != 0 {
			delta.SeverityDelta[severity] = change
		}
	}

	for ruleID, count := range current.ByRule {
		if previous.ByRule[ruleID] == 0 && count > 0 {
			delta.AddedRules = append(delta.AddedRules, RuleCount{Rule: ruleID, Count: count})
		}
	}
	for ruleID, count := range previous.ByRule {
		if current.ByRule[ruleID] == 0 && count > 0 {
			delta.ResolvedRules = append(delta.ResolvedRules, RuleCount{Rule: ruleID, Count: count})
		}
	}

	sortRuleCounts(delta.AddedRules)
	sortRuleCounts(delta.ResolvedRules)
	return delta
}

// Clean reports whether the current summary matches the previous summary.
func (d Delta) Clean() bool {
	return d.ScoreDelta == 0 && len(d.AddedRules) == 0 && len(d.ResolvedRules) == 0 && len(d.SeverityDelta) == 0
}

// IntroducesAt reports whether the delta adds diagnostics at or above threshold.
func (d Delta) IntroducesAt(threshold rule.Severity) bool {
	if threshold == rule.SeverityOff {
		return false
	}
	for severity := threshold; severity <= rule.SeverityError; severity++ {
		if d.SeverityDelta[severity.String()] > 0 {
			return true
		}
	}
	return false
}

func unionSeverityKeys(left, right map[string]int) []string {
	keys := map[string]bool{}
	for key := range left {
		keys[key] = true
	}
	for key := range right {
		keys[key] = true
	}
	result := make([]string, 0, len(keys))
	for key := range keys {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func sortRuleCounts(counts []RuleCount) {
	sort.Slice(counts, func(i, j int) bool {
		if counts[i].Count != counts[j].Count {
			return counts[i].Count > counts[j].Count
		}
		return counts[i].Rule < counts[j].Rule
	})
}
