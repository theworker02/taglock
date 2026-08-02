package evolution

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

func WriteJSON(writer io.Writer, report Report) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
func WriteText(writer io.Writer, report Report) error {
	for _, change := range report.Changes {
		if _, err := fmt.Fprintf(writer, "%s %s %s.%s %s: %s\n", change.ID, change.Severity, change.Package, change.TypeName, change.FieldPath, change.Message); err != nil {
			return err
		}
	}
	return nil
}
func WriteMarkdown(writer io.Writer, report Report) error {
	result := "Compatible"
	if report.Summary.Breaking > 0 {
		result = "Breaking"
	} else if report.Summary.Unknown > 0 {
		result = "Unknown"
	} else if report.Summary.PotentiallyBreaking > 0 {
		result = "Potentially breaking"
	}
	fmt.Fprintln(writer, "# TagLock Contract Report")
	fmt.Fprintln(writer)
	fmt.Fprintf(writer, "Compatibility result: **%s**\n\n", result)
	groups := []struct {
		title    string
		severity Severity
	}{{"Breaking changes", Breaking}, {"Potentially breaking changes", PotentiallyBreaking}, {"Unknown changes", Unknown}, {"Compatible changes", Compatible}}
	for _, group := range groups {
		var changes []Change
		for _, change := range report.Changes {
			if change.Severity == group.severity {
				changes = append(changes, change)
			}
		}
		if len(changes) == 0 {
			continue
		}
		fmt.Fprintln(writer, "## "+group.title+"\n")
		sort.Slice(changes, func(i, j int) bool { return changes[i].ChangeFingerprint < changes[j].ChangeFingerprint })
		for _, change := range changes {
			field := change.FieldPath
			if field == "" {
				field = change.TypeName
			}
			fmt.Fprintf(writer, "- **%s** `%s`: %s", change.ID, field, change.Message)
			if change.Acknowledged {
				fmt.Fprint(writer, " _(approved)_")
			}
			if change.RenameConfidence != "" {
				fmt.Fprintf(writer, " _(rename: %s)_", change.RenameConfidence)
			}
			fmt.Fprintln(writer)
		}
	}
	return nil
}
func WriteSARIF(writer io.Writer, report Report) error {
	type message struct {
		Text string `json:"text"`
	}
	type region struct {
		StartLine int `json:"startLine"`
	}
	type result struct {
		RuleID       string            `json:"ruleId"`
		Level        string            `json:"level"`
		Message      message           `json:"message"`
		Fingerprints map[string]string `json:"fingerprints"`
		Properties   map[string]any    `json:"properties"`
	}
	var results []result
	for _, change := range report.Changes {
		level := "note"
		if change.Severity == Breaking {
			level = "error"
		} else if change.Severity == PotentiallyBreaking || change.Severity == Unknown {
			level = "warning"
		}
		properties := map[string]any{"package": change.Package, "type": change.TypeName, "field": change.FieldPath, "profile": change.Profile, "direction": change.Direction, "severity": change.Severity}
		if change.RenameConfidence != "" {
			properties["renameConfidence"] = change.RenameConfidence
		}
		results = append(results, result{RuleID: change.ID, Level: level, Message: message{change.Message}, Fingerprints: map[string]string{"taglock/v1": change.ChangeFingerprint}, Properties: properties})
	}
	document := map[string]any{"version": "2.1.0", "$schema": "https://json.schemastore.org/sarif-2.1.0.json", "runs": []any{map[string]any{"tool": map[string]any{"driver": map[string]any{"name": "TagLock Evolution"}}, "results": results}}}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(document)
}

var _ = strings.Builder{}
