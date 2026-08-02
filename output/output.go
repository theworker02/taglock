// Package output renders deterministic TagLock diagnostics.
package output

import (
	"encoding/json"
	"fmt"
	"go/token"
	"io"
	"sort"
	"strings"

	"github.com/magnexis/taglock/baseline"
	"github.com/magnexis/taglock/rules"
)

const JSONSchemaVersion = 1

type Position struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}
type Related struct {
	Message  string   `json:"message"`
	Location Position `json:"location"`
}
type Fix struct {
	Message string `json:"message"`
	Safety  string `json:"safety"`
}
type Finding struct {
	RuleID    string    `json:"rule_id"`
	Severity  string    `json:"severity"`
	Message   string    `json:"message"`
	Namespace string    `json:"namespace,omitempty"`
	Profile   string    `json:"profile,omitempty"`
	Package   string    `json:"package"`
	Type      string    `json:"type"`
	FieldPath string    `json:"field_path,omitempty"`
	Location  Position  `json:"location"`
	Related   []Related `json:"related,omitempty"`
	Fixes     []Fix     `json:"fixes,omitempty"`
}
type JSONDocument struct {
	SchemaVersion int       `json:"schema_version"`
	Findings      []Finding `json:"findings"`
}

func Flatten(fileSet *token.FileSet, diagnostics []rules.Diagnostic) []Finding {
	result := make([]Finding, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		position := fileSet.Position(diagnostic.Pos)
		finding := Finding{RuleID: diagnostic.Rule.ID, Severity: diagnostic.Severity.String(), Message: diagnostic.Message, Namespace: diagnostic.Namespace, Profile: diagnostic.Profile, Package: diagnostic.Package, Type: diagnostic.TypeName, FieldPath: diagnostic.FieldPath, Location: Position{File: position.Filename, Line: position.Line, Column: position.Column}}
		for _, related := range diagnostic.Related {
			at := fileSet.Position(related.Pos)
			finding.Related = append(finding.Related, Related{Message: related.Message, Location: Position{File: at.Filename, Line: at.Line, Column: at.Column}})
		}
		for _, suggestion := range diagnostic.Fixes {
			finding.Fixes = append(finding.Fixes, Fix{Message: suggestion.Message, Safety: suggestion.Safety.String()})
		}
		result = append(result, finding)
	}
	return result
}

func Text(writer io.Writer, fileSet *token.FileSet, diagnostics []rules.Diagnostic) error {
	for _, diagnostic := range diagnostics {
		position := fileSet.Position(diagnostic.Pos)
		if _, err := fmt.Fprintf(writer, "%s:%d:%d: %s %s [%s]\n", position.Filename, position.Line, position.Column, diagnostic.Rule.ID, diagnostic.Message, diagnostic.Severity); err != nil {
			return err
		}
		for _, related := range diagnostic.Related {
			at := fileSet.Position(related.Pos)
			if _, err := fmt.Fprintf(writer, "  related: %s:%d:%d: %s\n", at.Filename, at.Line, at.Column, related.Message); err != nil {
				return err
			}
		}
	}
	return nil
}

func JSON(writer io.Writer, fileSet *token.FileSet, diagnostics []rules.Diagnostic) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(JSONDocument{SchemaVersion: JSONSchemaVersion, Findings: Flatten(fileSet, diagnostics)})
}

type sarifLog struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []sarifRun `json:"runs"`
}
type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}
type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}
type sarifDriver struct {
	Name           string      `json:"name"`
	InformationURI string      `json:"informationUri,omitempty"`
	Rules          []sarifRule `json:"rules"`
}
type sarifRule struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	ShortDescription sarifMessage   `json:"shortDescription"`
	FullDescription  sarifMessage   `json:"fullDescription"`
	Help             sarifMessage   `json:"help"`
	Properties       map[string]any `json:"properties,omitempty"`
}
type sarifMessage struct {
	Text string `json:"text"`
}
type sarifResult struct {
	RuleID           string            `json:"ruleId"`
	Level            string            `json:"level"`
	Message          sarifMessage      `json:"message"`
	Locations        []sarifLocation   `json:"locations"`
	RelatedLocations []sarifRelated    `json:"relatedLocations,omitempty"`
	Fixes            []sarifFix        `json:"fixes,omitempty"`
	Fingerprints     map[string]string `json:"fingerprints,omitempty"`
}
type sarifLocation struct {
	PhysicalLocation sarifPhysical `json:"physicalLocation"`
}
type sarifRelated struct {
	ID               int           `json:"id"`
	Message          sarifMessage  `json:"message"`
	PhysicalLocation sarifPhysical `json:"physicalLocation"`
}
type sarifPhysical struct {
	ArtifactLocation sarifArtifact `json:"artifactLocation"`
	Region           sarifRegion   `json:"region"`
}
type sarifArtifact struct {
	URI string `json:"uri"`
}
type sarifRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
}
type sarifFix struct {
	Description sarifMessage   `json:"description"`
	Properties  map[string]any `json:"properties,omitempty"`
}

func SARIF(writer io.Writer, fileSet *token.FileSet, diagnostics []rules.Diagnostic) error {
	used := map[string]rules.Diagnostic{}
	results := make([]sarifResult, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		used[diagnostic.Rule.ID] = diagnostic
		position := fileSet.Position(diagnostic.Pos)
		result := sarifResult{RuleID: diagnostic.Rule.ID, Level: sarifLevel(diagnostic.Severity.String()), Message: sarifMessage{Text: diagnostic.Message}, Locations: []sarifLocation{{PhysicalLocation: physical(position)}}, Fingerprints: map[string]string{"taglock/v1": baseline.Fingerprint(diagnostic)}}
		for index, related := range diagnostic.Related {
			at := fileSet.Position(related.Pos)
			result.RelatedLocations = append(result.RelatedLocations, sarifRelated{ID: index + 1, Message: sarifMessage{Text: related.Message}, PhysicalLocation: physical(at)})
		}
		for _, suggestion := range diagnostic.Fixes {
			result.Fixes = append(result.Fixes, sarifFix{Description: sarifMessage{Text: suggestion.Message}, Properties: map[string]any{"taglockFixSafety": suggestion.Safety.String(), "semanticProfile": diagnostic.Profile}})
		}
		results = append(results, result)
	}
	ids := make([]string, 0, len(used))
	for id := range used {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	catalog := make([]sarifRule, 0, len(ids))
	for _, id := range ids {
		definition := used[id].Rule
		catalog = append(catalog, sarifRule{ID: id, Name: definition.Name, ShortDescription: sarifMessage{Text: definition.Summary}, FullDescription: sarifMessage{Text: definition.Explanation}, Help: sarifMessage{Text: strings.Join(definition.Remediation, " ")}, Properties: map[string]any{"defaultSeverity": definition.DefaultSeverity.String(), "fixSafety": definition.FixSafety.String()}})
	}
	document := sarifLog{Version: "2.1.0", Schema: "https://json.schemastore.org/sarif-2.1.0.json", Runs: []sarifRun{{Tool: sarifTool{Driver: sarifDriver{Name: "TagLock", Rules: catalog}}, Results: results}}}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(document)
}
func physical(position token.Position) sarifPhysical {
	return sarifPhysical{ArtifactLocation: sarifArtifact{URI: strings.ReplaceAll(position.Filename, "\\", "/")}, Region: sarifRegion{StartLine: position.Line, StartColumn: position.Column}}
}
func sarifLevel(severity string) string {
	switch severity {
	case "error":
		return "error"
	case "warning":
		return "warning"
	default:
		return "note"
	}
}
