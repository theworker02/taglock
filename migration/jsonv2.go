// Package migration compares implementation semantic profiles.
package migration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/theworker02/taglock/contract"
	"github.com/theworker02/taglock/semantics"
)

type Category string

const (
	Unchanged              Category = "unchanged"
	BehavioralDifference   Category = "behavioral-difference"
	SourceChangeRequired   Category = "source-change-required"
	WireFormatChange       Category = "wire-format-change"
	DecodeOnlyChange       Category = "decode-only-change"
	EncodeOnlyChange       Category = "encode-only-change"
	UnknownCustomMarshaler Category = "unknown-custom-marshaler"
)

type Finding struct {
	ID             string   `json:"id"`
	Fingerprint    string   `json:"fingerprint"`
	Category       Category `json:"category"`
	Package        string   `json:"package"`
	TypeName       string   `json:"type"`
	FieldPath      string   `json:"field,omitempty"`
	Message        string   `json:"message"`
	V1             any      `json:"json_v1,omitempty"`
	V2             any      `json:"json_v2,omitempty"`
	Impact         string   `json:"impact"`
	Recommendation string   `json:"recommendation"`
}
type Report struct {
	FormatVersion int              `json:"format_version"`
	Toolchain     string           `json:"toolchain"`
	V2Available   bool             `json:"v2_available"`
	Summary       map[Category]int `json:"summary"`
	Findings      []Finding        `json:"findings"`
}

func Analyze(contracts []*contract.StructContract, registry semantics.Registry) Report {
	v1, _ := registry.Lookup("json/v1")
	v2, _ := registry.Lookup("json/v2")
	report := Report{FormatVersion: 1, Toolchain: registry.Toolchain, V2Available: registry.JSONV2Active, Summary: map[Category]int{}}
	for _, item := range contracts {
		first, _ := v1.ResolveStruct(item)
		second, _ := v2.ResolveStruct(item)
		if first.CustomMethods != second.CustomMethods || first.CustomMethods.MarshalJSON || first.CustomMethods.UnmarshalJSON {
			report.Findings = append(report.Findings, newFinding("JSONMIG007", UnknownCustomMarshaler, item, "", "custom JSON method interaction requires runtime verification", first.CustomMethods, second.CustomMethods, "method dispatch or options may interact differently", "run explicit verification fixtures under both implementations"))
		}
		if first.Certainty == semantics.CertaintyOpaque || second.Certainty == semantics.CertaintyOpaque {
			report.Findings = append(report.Findings, newFinding("JSONMIG009", UnknownCustomMarshaler, item, "", "custom JSON methods make the migration outcome unknown", first.Certainty, second.Certainty, "encoded or decoded behavior is controlled by custom code", "add explicit runtime verification fixtures"))
			continue
		}
		for _, field := range item.Fields {
			for _, finding := range v2.ValidateField(field) {
				report.Findings = append(report.Findings, newFinding(finding.ID, SourceChangeRequired, item, finding.FieldPath, finding.Message, nil, nil, "source does not satisfy json/v2 tag semantics", "replace the unsupported option with a verified v2 equivalent"))
			}
			if value := field.Tags["json"]; value != nil && value.HasOption("inline") {
				report.Findings = append(report.Findings, newFinding("JSONMIG002", BehavioralDifference, item, field.GoName, "json/v2 interprets explicit inline while json/v1 does not", field.GoName, "promoted fields", "encoded and decoded field surfaces may change", "use explicit names or retain v1 behavior"))
			}
		}
		oldFields := surfaceByGo(first)
		newFields := surfaceByGo(second)
		surfaceChanged := surfaceSignature(first) != surfaceSignature(second)
		keys := union(oldFields, newFields)
		for _, key := range keys {
			old, oldOK := oldFields[key]
			newValue, newOK := newFields[key]
			if oldOK != newOK || old.Name != newValue.Name {
				category := WireFormatChange
				id := "JSONMIG003"
				if (oldOK && old.Embedded) || (newOK && newValue.Embedded) {
					id = "JSONMIG004"
				}
				report.Findings = append(report.Findings, newFinding(id, category, item, key, "effective JSON field resolution differs between v1 and v2", fieldIdentity(old, oldOK), fieldIdentity(newValue, newOK), "encoded output or accepted input may change", "add explicit field tags or preserve json/v1 semantics"))
				continue
			}
			if old.OmitEmpty != newValue.OmitEmpty || old.OmitZero != newValue.OmitZero {
				report.Findings = append(report.Findings, newFinding("JSONMIG005", EncodeOnlyChange, item, key, "omission behavior differs between profiles", old, newValue, "encoded field presence may change", "choose an explicit omission option verified under both profiles"))
			}
		}
		if surfaceChanged && hasDuplicateExplicitName(item) {
			report.Findings = append(report.Findings, newFinding("JSONMIG008", WireFormatChange, item, "", "duplicate-name resolution has a different effective outcome", surfaceSignature(first), surfaceSignature(second), "a selected, ignored, or ambiguous field may change", "remove ambiguity with unique explicit names"))
		}
		if !first.CaseSensitiveDecode && second.CaseSensitiveDecode {
			report.Findings = append(report.Findings, newFinding("JSONMIG006", DecodeOnlyChange, item, "", "default field-name matching changes from case-insensitive to case-sensitive", "case-insensitive", "case-sensitive", "legacy payload names with case differences may stop decoding", "use case:ignore where verified or normalize producers"))
			report.Findings = append(report.Findings, newFinding("JSONMIG010", SourceChangeRequired, item, "", "an explicit compatibility option can preserve case-insensitive name matching", nil, "case:ignore", "decode compatibility can be preserved explicitly", "review and add case:ignore only to fields requiring legacy matching"))
		}
	}
	sort.Slice(report.Findings, func(i, j int) bool {
		left, right := report.Findings[i], report.Findings[j]
		if left.Package != right.Package {
			return left.Package < right.Package
		}
		if left.TypeName != right.TypeName {
			return left.TypeName < right.TypeName
		}
		if left.FieldPath != right.FieldPath {
			return left.FieldPath < right.FieldPath
		}
		return left.ID < right.ID
	})
	for index := range report.Findings {
		report.Findings[index].Fingerprint = migrationFingerprint(report.Findings[index])
		report.Summary[report.Findings[index].Category]++
	}
	return report
}
func surfaceSignature(surface *semantics.ResolvedSurface) string {
	values := []string{}
	for _, field := range surface.Fields {
		if !field.Ignored {
			values = append(values, field.GoName+"="+field.Name)
		}
	}
	sort.Strings(values)
	return strings.Join(values, ",")
}
func hasDuplicateExplicitName(item *contract.StructContract) bool {
	seen := map[string]bool{}
	for _, field := range item.Fields {
		value := field.Tags["json"]
		if value == nil || value.Ignored || value.Name == "" {
			continue
		}
		if seen[value.Name] {
			return true
		}
		seen[value.Name] = true
	}
	return false
}
func newFinding(id string, category Category, item *contract.StructContract, field, message string, v1, v2 any, impact, recommendation string) Finding {
	return Finding{ID: id, Category: category, Package: item.Package, TypeName: item.TypeName, FieldPath: field, Message: message, V1: v1, V2: v2, Impact: impact, Recommendation: recommendation}
}
func surfaceByGo(surface *semantics.ResolvedSurface) map[string]semantics.ResolvedField {
	result := map[string]semantics.ResolvedField{}
	for _, field := range surface.Fields {
		if !field.Ignored {
			result[field.GoName] = field
		}
	}
	return result
}
func union(first, second map[string]semantics.ResolvedField) []string {
	seen := map[string]bool{}
	for key := range first {
		seen[key] = true
	}
	for key := range second {
		seen[key] = true
	}
	result := make([]string, 0, len(seen))
	for key := range seen {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}
func fieldIdentity(value semantics.ResolvedField, ok bool) any {
	if !ok {
		return nil
	}
	return map[string]any{"name": value.Name, "path": strings.Join(value.Path, "."), "type": value.TypeString}
}
func migrationFingerprint(value Finding) string {
	data, _ := json.Marshal(struct{ ID, Package, Type, Field, Message string }{value.ID, value.Package, value.TypeName, value.FieldPath, value.Message})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:16])
}
func WriteJSON(report Report) ([]byte, error) { return json.MarshalIndent(report, "", "  ") }
func WriteText(report Report) string {
	var builder strings.Builder
	for _, finding := range report.Findings {
		fmt.Fprintf(&builder, "%s %s.%s", finding.ID, finding.Package, finding.TypeName)
		if finding.FieldPath != "" {
			fmt.Fprintf(&builder, ".%s", finding.FieldPath)
		}
		fmt.Fprintf(&builder, ": %s\n  impact: %s\n  recommendation: %s\n", finding.Message, finding.Impact, finding.Recommendation)
	}
	return builder.String()
}
func RuleDefinitions() map[string]string {
	return map[string]string{"JSONMIG001": "Unsupported v1 option under v2", "JSONMIG002": "New v2 interpretation", "JSONMIG003": "Field resolution changed", "JSONMIG004": "Embedded-field behavior changed", "JSONMIG005": "Omission behavior changed", "JSONMIG006": "Name matching changed", "JSONMIG007": "Custom marshaler interaction changed", "JSONMIG008": "Duplicate-name outcome changed", "JSONMIG009": "Unknown migration outcome", "JSONMIG010": "Explicit compatibility option recommended"}
}
