// Package baseline stores durable fingerprints for incremental adoption.
package baseline

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"

	"github.com/theworker02/taglock/rules"
)

const Version = 1

type File struct {
	Version  int      `json:"version"`
	Findings []Record `json:"findings"`
}
type Record struct {
	Fingerprint string `json:"fingerprint"`
	RuleID      string `json:"rule_id"`
	Package     string `json:"package"`
	Type        string `json:"type"`
	FieldPath   string `json:"field_path,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	Profile     string `json:"profile,omitempty"`
	Identity    string `json:"identity"`
}

// Fingerprint deliberately excludes source line numbers.
func Fingerprint(diagnostic rules.Diagnostic) string {
	identity := normalizedIdentity(diagnostic.Message)
	value := strings.Join([]string{diagnostic.Rule.ID, diagnostic.Package, diagnostic.TypeName, diagnostic.FieldPath, diagnostic.Namespace, diagnostic.Profile, identity}, "\x00")
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:16])
}

func RecordFor(diagnostic rules.Diagnostic) Record {
	return Record{Fingerprint: Fingerprint(diagnostic), RuleID: diagnostic.Rule.ID, Package: diagnostic.Package, Type: diagnostic.TypeName, FieldPath: diagnostic.FieldPath, Namespace: diagnostic.Namespace, Profile: diagnostic.Profile, Identity: normalizedIdentity(diagnostic.Message)}
}

func Create(diagnostics []rules.Diagnostic) File {
	records := make([]Record, 0, len(diagnostics))
	seen := map[string]bool{}
	for _, diagnostic := range diagnostics {
		record := RecordFor(diagnostic)
		if !seen[record.Fingerprint] {
			records = append(records, record)
			seen[record.Fingerprint] = true
		}
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Fingerprint < records[j].Fingerprint })
	return File{Version: Version, Findings: records}
}

func Read(reader io.Reader) (File, error) {
	var file File
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return File{}, fmt.Errorf("decode baseline: %w", err)
	}
	if file.Version != Version {
		return File{}, fmt.Errorf("unsupported baseline version %d", file.Version)
	}
	return file, nil
}
func Write(writer io.Writer, file File) error {
	sort.Slice(file.Findings, func(i, j int) bool { return file.Findings[i].Fingerprint < file.Findings[j].Fingerprint })
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(file)
}

// Filter returns new diagnostics and stale baseline records.
func Filter(diagnostics []rules.Diagnostic, file File) ([]rules.Diagnostic, []Record) {
	existing := map[string]Record{}
	for _, record := range file.Findings {
		existing[record.Fingerprint] = record
	}
	matched := map[string]bool{}
	var active []rules.Diagnostic
	for _, diagnostic := range diagnostics {
		fingerprint := Fingerprint(diagnostic)
		if _, ok := existing[fingerprint]; ok {
			matched[fingerprint] = true
			continue
		}
		active = append(active, diagnostic)
	}
	var stale []Record
	for fingerprint, record := range existing {
		if !matched[fingerprint] {
			stale = append(stale, record)
		}
	}
	sort.Slice(stale, func(i, j int) bool { return stale[i].Fingerprint < stale[j].Fingerprint })
	return active, stale
}

func normalizedIdentity(message string) string {
	var builder strings.Builder
	space := false
	for _, character := range strings.ToLower(message) {
		if unicode.IsSpace(character) {
			space = true
			continue
		}
		if space && builder.Len() > 0 {
			builder.WriteByte(' ')
		}
		space = false
		builder.WriteRune(character)
	}
	return builder.String()
}
