// Package rule defines TagLock's stable diagnostic catalog.
package rule

import (
	"fmt"
	"sort"
	"strings"
)

// Severity controls reporting and failure thresholds.
type Severity int

const (
	SeverityOff Severity = iota
	SeverityInfo
	SeverityWarning
	SeverityError
)

func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	default:
		return "off"
	}
}

// ParseSeverity converts a configuration or CLI value to a severity.
func ParseSeverity(value string) (Severity, error) {
	switch strings.ToLower(value) {
	case "off":
		return SeverityOff, nil
	case "info", "informational", "note":
		return SeverityInfo, nil
	case "warning", "warn":
		return SeverityWarning, nil
	case "error":
		return SeverityError, nil
	default:
		return SeverityOff, fmt.Errorf("invalid severity %q (want off, info, warning, or error)", value)
	}
}

// FixSafety separates behavior-preserving fixes from review-required edits.
type FixSafety int

const (
	FixNone FixSafety = iota
	FixSafe
	FixReview
)

func (s FixSafety) String() string {
	switch s {
	case FixSafe:
		return "safe"
	case FixReview:
		return "review"
	default:
		return "none"
	}
}

// Definition is the single source of truth for diagnostics and explanations.
type Definition struct {
	ID              string
	Name            string
	Category        string
	Summary         string
	Explanation     string
	Remediation     []string
	Incorrect       string
	Correct         string
	DefaultSeverity Severity
	DefaultEnabled  bool
	CanFix          bool
	FixSafety       FixSafety
}

var definitions = []Definition{
	def("TAG001", "Malformed struct tag", "syntax", "The struct tag is not valid Go tag syntax.", SeverityError, true, false, FixNone),
	def("TAG002", "Duplicate tag namespace", "syntax", "A field declares the same tag namespace more than once.", SeverityError, true, false, FixNone),
	def("TAG003", "Duplicate tag option", "syntax", "A namespace option is repeated and has no additional effect.", SeverityWarning, true, true, FixSafe),
	def("TAG004", "Unknown tag option", "syntax", "A tag option is not recognized by the configured namespace.", SeverityWarning, true, false, FixNone),
	def("TAG005", "Empty or suspicious tag name", "syntax", "A tag name is empty, contains whitespace or control characters, or uses a malformed separator.", SeverityWarning, true, false, FixNone),
	def("TAG101", "Ineffective tag on unexported field", "field", "Serialization tags on unexported fields are ignored by common Go encoders.", SeverityWarning, true, true, FixSafe),
	def("TAG102", "Redundant explicit name", "field", "An explicit external name repeats the namespace default.", SeverityInfo, false, true, FixSafe),
	def("TAG103", "Ignored field has additional options", "field", "Options after an ignore marker have no effect.", SeverityWarning, true, true, FixSafe),
	def("TAG104", "Duplicate external field name", "field", "Multiple direct fields resolve to the same external name.", SeverityError, true, true, FixReview),
	def("TAG105", "Promoted-field collision", "field", "Embedded or inline fields introduce an ambiguous external name.", SeverityError, true, true, FixReview),
	def("TAG106", "Suspicious case-only collision", "field", "External names differ only by case in a case-insensitive contract.", SeverityWarning, true, false, FixNone),
	def("TAG201", "Option incompatible with field type", "types", "A tag option cannot operate on the field's Go type.", SeverityError, true, false, FixNone),
	def("TAG202", "Omission option has no practical effect", "types", "The selected omission behavior is ineffective for this Go type.", SeverityWarning, true, true, FixReview),
	def("TAG203", "Invalid inline target", "types", "Inline behavior requires a struct, pointer-to-struct, or supported map.", SeverityError, true, false, FixNone),
	def("TAG204", "Conflicting representation options", "types", "Two options request incompatible representations.", SeverityError, true, false, FixNone),
	def("TAG301", "External naming drift", "consistency", "A field uses unexpectedly different external names across namespaces.", SeverityWarning, true, true, FixReview),
	def("TAG302", "Inconsistent ignore policy", "consistency", "A field is hidden in one output namespace and exposed in another.", SeverityWarning, true, true, FixReview),
	def("TAG303", "Inconsistent omission policy", "consistency", "A field uses materially different omission behavior across output namespaces.", SeverityInfo, true, true, FixReview),
	def("TAG304", "Missing required namespace", "consistency", "An exported field lacks a tag required by project policy.", SeverityError, true, false, FixNone),
	def("TAG305", "Inconsistent external identity", "consistency", "External names appear to describe different concepts rather than naming-style variants.", SeverityWarning, true, false, FixNone),
	def("TAG401", "Sensitive field exposed", "security", "A security-sensitive field is publicly serialized.", SeverityError, true, true, FixReview),
	def("TAG402", "Sensitive field renamed deceptively", "security", "A sensitive field is exposed under a name unrelated to its Go identity.", SeverityWarning, true, false, FixNone),
	def("TAG403", "Write-only contract mismatch", "security", "A configured write-only sensitive field is exposed by an output namespace.", SeverityError, false, true, FixReview),
	def("TAG901", "Invalid suppression", "suppression", "A taglock suppression is malformed or references an unknown rule.", SeverityWarning, true, false, FixNone),
	def("TAG902", "Unused suppression", "suppression", "A valid suppression did not match any diagnostic on its declaration.", SeverityInfo, true, false, FixNone),
}

func def(id, name, category, summary string, severity Severity, enabled, canFix bool, safety FixSafety) Definition {
	return Definition{
		ID: id, Name: name, Category: category, Summary: summary,
		Explanation:     summary + " TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.",
		Remediation:     []string{"Make the contract explicit and unambiguous.", "Suppress the rule locally with a documented reason only when the behavior is intentional."},
		Incorrect:       "type Payload struct { Field string `json:\"field\"` }",
		Correct:         "type Payload struct { Field string `json:\"field\"` }",
		DefaultSeverity: severity, DefaultEnabled: enabled, CanFix: canFix, FixSafety: safety,
	}
}

var byID = func() map[string]Definition {
	result := make(map[string]Definition, len(definitions))
	for _, definition := range definitions {
		result[definition.ID] = definition
	}
	return result
}()

// Lookup returns metadata for a stable rule identifier.
func Lookup(id string) (Definition, bool) {
	definition, ok := byID[strings.ToUpper(id)]
	return definition, ok
}

// All returns the rule catalog in stable identifier order.
func All() []Definition {
	result := append([]Definition(nil), definitions...)
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

// Known reports whether id is registered.
func Known(id string) bool { _, ok := Lookup(id); return ok }
