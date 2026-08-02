// Package snapshot creates canonical serialized contract manifests.
package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/types"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/magnexis/taglock/config"
	"github.com/magnexis/taglock/engine"
	"github.com/magnexis/taglock/internal/version"
	"github.com/magnexis/taglock/semantics"
)

const FormatVersion = 1

type Snapshot struct {
	FormatVersion    int               `json:"format_version"`
	TagLockVersion   string            `json:"taglock_version"`
	GoVersion        string            `json:"go_version"`
	ModulePath       string            `json:"module_path"`
	ModuleVersion    string            `json:"module_version,omitempty"`
	GeneratedAt      string            `json:"generated_at,omitempty"`
	SemanticProfiles []string          `json:"semantic_profiles"`
	Packages         []PackageSnapshot `json:"packages"`
	Fingerprint      string            `json:"fingerprint"`
}
type PackageSnapshot struct {
	ImportPath string             `json:"import_path"`
	Contracts  []ContractSnapshot `json:"contracts"`
}
type ContractSnapshot struct {
	TypeName       string            `json:"type"`
	TypeParameters []string          `json:"type_parameters,omitempty"`
	Source         string            `json:"source,omitempty"`
	Documentation  string            `json:"documentation,omitempty"`
	Level          string            `json:"level"`
	Exported       bool              `json:"exported"`
	Profiles       []SurfaceSnapshot `json:"profiles"`
	Fingerprint    string            `json:"fingerprint"`
}
type SurfaceSnapshot struct {
	Profile             string                      `json:"profile"`
	Namespace           string                      `json:"namespace"`
	Certainty           semantics.ContractCertainty `json:"certainty"`
	CertaintyReason     string                      `json:"certainty_reason,omitempty"`
	Available           bool                        `json:"available"`
	CaseSensitiveDecode bool                        `json:"case_sensitive_decode"`
	CustomMethods       semantics.CustomMethods     `json:"custom_methods"`
	Fields              []FieldSnapshot             `json:"fields"`
}
type FieldSnapshot struct {
	GoName       string               `json:"go_name"`
	ExternalName string               `json:"external_name"`
	GoType       string               `json:"go_type"`
	WireType     string               `json:"wire_type"`
	Path         string               `json:"path"`
	Required     bool                 `json:"required"`
	Nullable     bool                 `json:"nullable"`
	OmitEmpty    bool                 `json:"omitempty,omitempty"`
	OmitZero     bool                 `json:"omitzero,omitempty"`
	Embedded     bool                 `json:"embedded,omitempty"`
	Ignored      bool                 `json:"ignored,omitempty"`
	Deprecated   *DeprecationSnapshot `json:"deprecated,omitempty"`
	Schema       map[string]string    `json:"schema,omitempty"`
}
type DeprecationSnapshot struct {
	Since       string `json:"since,omitempty"`
	RemoveAfter string `json:"remove_after,omitempty"`
	Replacement string `json:"replacement,omitempty"`
	Note        string `json:"note,omitempty"`
}

type BuildOptions struct {
	Semantics    string
	Reproducible bool
}

func Build(result engine.Result, cfg config.Config, options BuildOptions) (Snapshot, error) {
	registry := semantics.NewRegistry()
	profiles, err := registry.Select(options.Semantics)
	if err != nil {
		return Snapshot{}, err
	}
	document := Snapshot{FormatVersion: FormatVersion, TagLockVersion: version.Value, GoVersion: result.GoVersion, ModulePath: result.ModulePath, ModuleVersion: result.ModuleVersion}
	if !options.Reproducible {
		document.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	}
	for _, profile := range profiles {
		document.SemanticProfiles = append(document.SemanticProfiles, profile.ID())
	}
	packages := map[string][]ContractSnapshot{}
	for _, item := range result.Contracts {
		position := result.FileSet.Position(item.Position)
		if !cfg.SelectContract(item.Package, position.Filename, item.Exported) {
			continue
		}
		level := cfg.ContractLevel(item.Package, item.Level)
		if !config.ValidContractLevel(level) {
			return Snapshot{}, fmt.Errorf("%s.%s uses invalid contract level %q", item.Package, item.TypeName, level)
		}
		contractSnapshot := ContractSnapshot{TypeName: item.TypeName, Source: relativeSource(result.ModuleDir, position.Filename), Documentation: firstParagraph(item.Documentation), Level: level, Exported: item.Exported, TypeParameters: typeParameters(item.DeclaredType)}
		for _, profile := range profiles {
			surface, err := profile.ResolveStruct(item)
			if err != nil {
				return Snapshot{}, fmt.Errorf("resolve %s.%s with %s: %w", item.Package, item.TypeName, profile.ID(), err)
			}
			converted := SurfaceSnapshot{Profile: surface.Profile, Namespace: surface.Namespace, Certainty: surface.Certainty, CertaintyReason: surface.Reason, Available: surface.Available, CaseSensitiveDecode: surface.CaseSensitiveDecode, CustomMethods: surface.CustomMethods}
			for _, field := range surface.Fields {
				if err := validateSchemaAnnotations(field.Schema); err != nil {
					return Snapshot{}, fmt.Errorf("%s.%s.%s: %w", item.Package, item.TypeName, field.GoName, err)
				}
				deprecation := (*DeprecationSnapshot)(nil)
				if field.Deprecated != nil {
					deprecation = &DeprecationSnapshot{Since: field.Deprecated.Since, RemoveAfter: field.Deprecated.RemoveAfter, Replacement: field.Deprecated.Replacement, Note: firstParagraph(field.Deprecated.Note)}
				}
				converted.Fields = append(converted.Fields, FieldSnapshot{GoName: field.GoName, ExternalName: field.Name, GoType: field.TypeString, WireType: wireType(field.GoType), Path: strings.Join(field.Path, "."), Required: field.Required, Nullable: field.Nullable, OmitEmpty: field.OmitEmpty, OmitZero: field.OmitZero, Embedded: field.Embedded, Ignored: field.Ignored, Deprecated: deprecation, Schema: cloneMap(field.Schema)})
			}
			sort.Slice(converted.Fields, func(i, j int) bool {
				if converted.Fields[i].ExternalName != converted.Fields[j].ExternalName {
					return converted.Fields[i].ExternalName < converted.Fields[j].ExternalName
				}
				return converted.Fields[i].Path < converted.Fields[j].Path
			})
			contractSnapshot.Profiles = append(contractSnapshot.Profiles, converted)
		}
		sort.Slice(contractSnapshot.Profiles, func(i, j int) bool {
			return contractSnapshot.Profiles[i].Profile < contractSnapshot.Profiles[j].Profile
		})
		contractSnapshot.Fingerprint = fingerprint(contractSemantic(contractSnapshot))
		packages[item.Package] = append(packages[item.Package], contractSnapshot)
	}
	paths := make([]string, 0, len(packages))
	for path := range packages {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		contracts := packages[path]
		sort.Slice(contracts, func(i, j int) bool { return contracts[i].TypeName < contracts[j].TypeName })
		document.Packages = append(document.Packages, PackageSnapshot{ImportPath: path, Contracts: contracts})
	}
	document.Fingerprint = fingerprint(snapshotSemantic(document))
	return document, nil
}

func validateSchemaAnnotations(values map[string]string) error {
	for key, value := range values {
		switch key {
		case "format":
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("schema format cannot be empty")
			}
		case "minimum", "maximum":
			if _, err := strconv.ParseFloat(value, 64); err != nil {
				return fmt.Errorf("schema %s must be numeric: %w", key, err)
			}
		default:
			return fmt.Errorf("unknown taglock:schema annotation %q", key)
		}
	}
	return nil
}

func Write(writer io.Writer, document Snapshot) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(document)
}
func Read(reader io.Reader) (Snapshot, error) {
	var document Snapshot
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return Snapshot{}, fmt.Errorf("decode contract snapshot: %w", err)
	}
	if document.FormatVersion != FormatVersion {
		return Snapshot{}, fmt.Errorf("unsupported snapshot format %d", document.FormatVersion)
	}
	return document, nil
}
func Canonical(document Snapshot) ([]byte, error) {
	document.GeneratedAt = ""
	document.Fingerprint = ""
	return json.Marshal(document)
}
func fingerprint(value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:16])
}
func contractSemantic(value ContractSnapshot) any {
	return struct {
		Type     string
		Level    string
		Profiles []SurfaceSnapshot
	}{value.TypeName, value.Level, value.Profiles}
}
func snapshotSemantic(value Snapshot) any {
	return struct {
		Module   string
		Profiles []string
		Packages []PackageSnapshot
	}{value.ModulePath, value.SemanticProfiles, value.Packages}
}
func relativeSource(root, filename string) string {
	if root != "" {
		if value, err := filepath.Rel(root, filename); err == nil && !strings.HasPrefix(value, "..") {
			return filepath.ToSlash(value)
		}
	}
	return filepath.Base(filename)
}
func firstParagraph(value string) string {
	value = strings.TrimSpace(value)
	if index := strings.Index(value, "\n\n"); index >= 0 {
		return strings.TrimSpace(value[:index])
	}
	return value
}
func cloneMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
func typeParameters(value types.Type) []string {
	named, ok := value.(*types.Named)
	if !ok || named.TypeParams() == nil {
		return nil
	}
	result := make([]string, named.TypeParams().Len())
	for index := range result {
		result[index] = named.TypeParams().At(index).Obj().Name()
	}
	return result
}

func wireType(value types.Type) string {
	if value == nil {
		return "unknown"
	}
	if pointer, ok := value.(*types.Pointer); ok {
		return wireType(pointer.Elem())
	}
	if named, ok := value.(*types.Named); ok {
		if named.Obj() != nil && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == "time" && named.Obj().Name() == "Time" {
			return "date-time"
		}
		return wireType(named.Underlying())
	}
	switch current := value.Underlying().(type) {
	case *types.Basic:
		info := current.Info()
		switch {
		case info&types.IsBoolean != 0:
			return "boolean"
		case info&(types.IsInteger|types.IsUnsigned) != 0:
			return "integer"
		case info&types.IsFloat != 0:
			return "number"
		case info&types.IsString != 0:
			return "string"
		}
	case *types.Slice:
		if basic, ok := current.Elem().Underlying().(*types.Basic); ok && basic.Kind() == types.Uint8 {
			return "bytes"
		}
		return "array:" + wireType(current.Elem())
	case *types.Array:
		return fmt.Sprintf("array[%d]:%s", current.Len(), wireType(current.Elem()))
	case *types.Map:
		return "object:" + wireType(current.Elem())
	case *types.Struct:
		return "object"
	case *types.Interface:
		return "any"
	}
	return "unknown"
}
