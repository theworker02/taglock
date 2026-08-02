// Package evolution compares deterministic contract snapshots.
package evolution

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

const FormatVersion = 1

type Severity string

const (
	Compatible          Severity = "compatible"
	PotentiallyBreaking Severity = "potentially-breaking"
	Breaking            Severity = "breaking"
	Unknown             Severity = "unknown"
	Informational       Severity = "informational"
)

type Direction string

const (
	Producer      Direction = "producer"
	Consumer      Direction = "consumer"
	Bidirectional Direction = "bidirectional"
	Storage       Direction = "storage"
	Configuration Direction = "configuration"
	Message       Direction = "message"
	Advisory      Direction = "advisory"
)

type Change struct {
	ID                string         `json:"id"`
	ChangeFingerprint string         `json:"change_fingerprint"`
	Kind              string         `json:"kind"`
	Package           string         `json:"package"`
	TypeName          string         `json:"type"`
	FieldPath         string         `json:"field,omitempty"`
	Profile           string         `json:"profile"`
	Direction         Direction      `json:"direction"`
	Severity          Severity       `json:"severity"`
	Before            any            `json:"before,omitempty"`
	After             any            `json:"after,omitempty"`
	Message           string         `json:"message"`
	Recommendation    string         `json:"recommendation"`
	RenameConfidence  string         `json:"rename_confidence,omitempty"`
	Acknowledged      bool           `json:"acknowledged,omitempty"`
	Approval          *ApprovalMatch `json:"approval,omitempty"`
}
type ApprovalMatch struct {
	ID        string `json:"id"`
	Reason    string `json:"reason"`
	Migration string `json:"migration"`
}
type Summary struct {
	Compatible          int `json:"compatible"`
	PotentiallyBreaking int `json:"potentially_breaking"`
	Breaking            int `json:"breaking"`
	Unknown             int `json:"unknown"`
	Informational       int `json:"informational"`
}
type Report struct {
	FormatVersion int              `json:"format_version"`
	Base          SnapshotMetadata `json:"base"`
	Head          SnapshotMetadata `json:"head"`
	Summary       Summary          `json:"summary"`
	Changes       []Change         `json:"changes"`
}
type SnapshotMetadata struct {
	Module      string   `json:"module"`
	Fingerprint string   `json:"fingerprint"`
	GoVersion   string   `json:"go_version"`
	Profiles    []string `json:"profiles"`
}

func fingerprint(change Change) string {
	value := struct{ ID, Package, Type, Field, Profile, Message, RenameConfidence string }{change.ID, change.Package, change.TypeName, change.FieldPath, change.Profile, change.Message, change.RenameConfidence}
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:16])
}
