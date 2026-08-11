package triage

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// SnapshotFormatVersion is the supported on-disk triage snapshot schema version.
const SnapshotFormatVersion = 1

// Snapshot is a durable triage summary used for CI delta comparisons.
type Snapshot struct {
	FormatVersion int       `json:"format_version"`
	CreatedAt     time.Time `json:"created_at"`
	Summary       Summary   `json:"summary"`
}

// WriteSnapshot persists summary as a versioned triage snapshot file.
func WriteSnapshot(writer io.Writer, summary Summary) error {
	snapshot := Snapshot{
		FormatVersion: SnapshotFormatVersion,
		CreatedAt:     time.Now().UTC(),
		Summary:       summary,
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(snapshot)
}

// ReadSnapshot loads a snapshot file written by WriteSnapshot.
func ReadSnapshot(reader io.Reader) (Snapshot, error) {
	var snapshot Snapshot
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&snapshot); err != nil {
		return Snapshot{}, fmt.Errorf("decode triage snapshot: %w", err)
	}
	if snapshot.FormatVersion != SnapshotFormatVersion {
		return Snapshot{}, fmt.Errorf("unsupported triage snapshot version %d", snapshot.FormatVersion)
	}
	return snapshot, nil
}
