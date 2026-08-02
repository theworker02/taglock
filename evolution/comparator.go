package evolution

import (
	"fmt"
	"sort"
	"strings"

	"github.com/magnexis/taglock/config"
	"github.com/magnexis/taglock/snapshot"
	"golang.org/x/mod/semver"
)

func Compare(base, head snapshot.Snapshot, cfg config.Config) Report {
	report := Report{FormatVersion: FormatVersion, Base: metadata(base), Head: metadata(head), Changes: []Change{}}
	baseContracts := contractIndex(base)
	headContracts := contractIndex(head)
	keys := unionKeys(baseContracts, headContracts)
	for _, key := range keys {
		before, beforeOK := baseContracts[key]
		after, afterOK := headContracts[key]
		switch {
		case beforeOK && !afterOK:
			report.Changes = append(report.Changes, newChange("EVOL001", "contract-removed", before.Package, before.Contract.TypeName, "", "", mode(before.Contract.Level, cfg), Breaking, before.Contract, nil, "public serialized contract was removed", "restore the contract or publish a documented major-version migration"))
		case !beforeOK && afterOK:
			report.Changes = append(report.Changes, newChange("EVOL100", "contract-added", after.Package, after.Contract.TypeName, "", "", mode(after.Contract.Level, cfg), Compatible, nil, after.Contract, "serialized contract was added", "no migration is normally required"))
		default:
			report.Changes = append(report.Changes, compareContract(before.Package, before.Contract, after.Contract, base.ModuleVersion, head.ModuleVersion, cfg)...)
		}
	}
	sortChanges(report.Changes)
	for index := range report.Changes {
		report.Changes[index].ChangeFingerprint = fingerprint(report.Changes[index])
	}
	report.Summary = summarize(report.Changes)
	return report
}

type indexedContract struct {
	Package  string
	Contract snapshot.ContractSnapshot
}

func contractIndex(document snapshot.Snapshot) map[string]indexedContract {
	result := map[string]indexedContract{}
	for _, pkg := range document.Packages {
		for _, item := range pkg.Contracts {
			result[pkg.ImportPath+"\x00"+item.TypeName] = indexedContract{pkg.ImportPath, item}
		}
	}
	return result
}
func compareContract(packagePath string, before, after snapshot.ContractSnapshot, baseVersion, headVersion string, cfg config.Config) []Change {
	var result []Change
	beforeProfiles := surfaceIndex(before)
	afterProfiles := surfaceIndex(after)
	profiles := unionKeys(beforeProfiles, afterProfiles)
	for _, profile := range profiles {
		oldSurface, oldOK := beforeProfiles[profile]
		newSurface, newOK := afterProfiles[profile]
		if !oldOK || !newOK {
			result = append(result, newChange("EVOL013", "semantic-profile-changed", packagePath, before.TypeName, "", profile, mode(after.Level, cfg), Unknown, optionalSurface(oldSurface, oldOK), optionalSurface(newSurface, newOK), "semantic profile set changed between snapshots", "compare contracts under the same explicit profile"))
			continue
		}
		if certaintyRank(newSurface.Certainty) < certaintyRank(oldSurface.Certainty) {
			result = append(result, newChange("EVOL012", "certainty-reduced", packagePath, before.TypeName, "", profile, mode(after.Level, cfg), Unknown, oldSurface.Certainty, newSurface.Certainty, "contract certainty was reduced by custom serialization behavior", "review custom marshalers and add runtime fixtures"))
		}
		if !oldSurface.CustomMethods.MarshalJSON && newSurface.CustomMethods.MarshalJSON {
			result = append(result, newChange("EVOL010", "custom-marshaler-added", packagePath, before.TypeName, "", profile, mode(after.Level, cfg), Unknown, oldSurface.CustomMethods, newSurface.CustomMethods, "custom marshaling was introduced", "document the wire format and add runtime verification"))
		}
		if oldSurface.CustomMethods.MarshalJSON && !newSurface.CustomMethods.MarshalJSON {
			result = append(result, newChange("EVOL011", "custom-marshaler-removed", packagePath, before.TypeName, "", profile, mode(after.Level, cfg), PotentiallyBreaking, oldSurface.CustomMethods, newSurface.CustomMethods, "custom marshaling was removed", "compare the previous custom output with reflection-based output"))
		}
		result = append(result, compareFields(packagePath, before, after, profile, oldSurface, newSurface, baseVersion, headVersion, cfg)...)
	}
	return result
}
func compareFields(packagePath string, before, after snapshot.ContractSnapshot, profile string, oldSurface, newSurface snapshot.SurfaceSnapshot, baseVersion, headVersion string, cfg config.Config) []Change {
	var result []Change
	oldByGo := fieldByGo(oldSurface.Fields)
	newByGo := fieldByGo(newSurface.Fields)
	matchedOld, matchedNew := map[string]bool{}, map[string]bool{}
	for oldName, oldField := range oldByGo {
		if _, exists := newByGo[oldName]; exists || oldField.Deprecated == nil || oldField.Deprecated.Replacement == "" {
			continue
		}
		for newName, newField := range newByGo {
			if _, existed := oldByGo[newName]; existed || matchedNew[newName] {
				continue
			}
			if oldField.Deprecated.Replacement != newName && oldField.Deprecated.Replacement != newField.ExternalName {
				continue
			}
			matchedOld[oldName], matchedNew[newName] = true, true
			result = append(result, compareFieldPair(packagePath, after, profile, mode(after.Level, cfg), after.TypeName+"."+newName, oldField, newField, "confirmed")...)
			break
		}
	}
	names := unionKeys(oldByGo, newByGo)
	for _, goName := range names {
		if matchedOld[goName] || matchedNew[goName] {
			continue
		}
		oldField, oldOK := oldByGo[goName]
		newField, newOK := newByGo[goName]
		direction := mode(after.Level, cfg)
		fieldPath := after.TypeName + "." + goName
		switch {
		case oldOK && !newOK:
			change := newChange("EVOL002", "field-removed", packagePath, after.TypeName, fieldPath, profile, direction, Breaking, oldField, nil, fmt.Sprintf("serialized field %q was removed", oldField.ExternalName), "retain the field through a deprecation window or provide a migration")
			result = append(result, change)
			if policyChange := deprecationPolicyChange(packagePath, after, fieldPath, profile, direction, oldField, baseVersion, headVersion, cfg); policyChange != nil {
				result = append(result, *policyChange)
			}
		case !oldOK && newOK:
			result = append(result, newChange("EVOL101", "field-added", packagePath, after.TypeName, fieldPath, profile, direction, addedSeverity(newField, direction), nil, newField, fmt.Sprintf("serialized field %q was added", newField.ExternalName), "ensure strict consumers tolerate the new field"))
		default:
			result = append(result, compareFieldPair(packagePath, after, profile, direction, fieldPath, oldField, newField, "confirmed")...)
		}
	}
	if after.Level == "storage" {
		for _, change := range result {
			if change.Severity == Breaking {
				result = append(result, newChange("EVOL015", "storage-migration-required", packagePath, after.TypeName, change.FieldPath, profile, Storage, Breaking, change.Before, change.After, "persistent serialized representation changed incompatibly", "create and test a storage migration"))
				break
			}
		}
	}
	return result
}

func compareFieldPair(packagePath string, after snapshot.ContractSnapshot, profile string, direction Direction, fieldPath string, oldField, newField snapshot.FieldSnapshot, renameConfidence string) []Change {
	var result []Change
	if oldField.Ignored && !newField.Ignored {
		result = append(result, newChange("EVOL007", "field-exposed", packagePath, after.TypeName, fieldPath, profile, direction, PotentiallyBreaking, oldField, newField, "previously ignored field became exposed", "review information disclosure and downstream schema impact"))
	}
	if !oldField.Ignored && newField.Ignored {
		result = append(result, newChange("EVOL008", "field-hidden", packagePath, after.TypeName, fieldPath, profile, direction, Breaking, oldField, newField, "previously exposed field became ignored", "retain an alias during migration"))
	}
	if !oldField.Ignored && !newField.Ignored && oldField.ExternalName != newField.ExternalName {
		change := newChange("EVOL003", "field-renamed", packagePath, after.TypeName, fieldPath, profile, direction, Breaking, oldField.ExternalName, newField.ExternalName, fmt.Sprintf("serialized field %q was renamed to %q", oldField.ExternalName, newField.ExternalName), "preserve the old name through an alias or coordinate all producers and consumers")
		change.RenameConfidence = renameConfidence
		result = append(result, change)
	}
	if oldField.GoType != newField.GoType {
		result = append(result, newChange("EVOL004", "field-type-changed", packagePath, after.TypeName, fieldPath, profile, direction, Breaking, oldField.GoType, newField.GoType, "serialized field type changed", "introduce a new field or versioned contract"))
	}
	if !oldField.Required && newField.Required {
		result = append(result, newChange("EVOL005", "field-required", packagePath, after.TypeName, fieldPath, profile, direction, Breaking, false, true, "optional field became required", "accept missing legacy data during a migration window"))
	}
	if oldField.Required && !newField.Required {
		result = append(result, newChange("EVOL006", "field-optional", packagePath, after.TypeName, fieldPath, profile, direction, PotentiallyBreaking, true, false, "required field became optional", "ensure consumers handle omission"))
	}
	if oldField.Path != newField.Path {
		result = append(result, newChange("EVOL009", "embedded-path-changed", packagePath, after.TypeName, fieldPath, profile, direction, PotentiallyBreaking, oldField.Path, newField.Path, "embedded field path changed", "add explicit tags to stabilize promotion behavior"))
	}
	return result
}

func deprecationPolicyChange(packagePath string, after snapshot.ContractSnapshot, fieldPath, profile string, direction Direction, field snapshot.FieldSnapshot, baseVersion, headVersion string, cfg config.Config) *Change {
	if field.Deprecated == nil {
		if !cfg.Evolution.RequireDeprecationBeforeRemoval && cfg.Evolution.MinimumDeprecationReleases == 0 {
			return nil
		}
		change := newChange("EVOL014", "deprecation-window-violated", packagePath, after.TypeName, fieldPath, profile, direction, Breaking, nil, nil, "field was removed without a recorded deprecation", "restore the field and deprecate it before removal")
		return &change
	}
	if cfg.Evolution.RequireReplacementForPublicFields && after.Level == "public" && field.Deprecated.Replacement == "" {
		change := newChange("EVOL014", "deprecation-replacement-missing", packagePath, after.TypeName, fieldPath, profile, direction, Breaking, field.Deprecated, nil, "public field was removed without a documented replacement", "record a replacement in the deprecation annotation before removal")
		return &change
	}
	if field.Deprecated.RemoveAfter != "" && validSemver(field.Deprecated.RemoveAfter) && validSemver(headVersion) && semver.Compare(normalizeVersion(headVersion), normalizeVersion(field.Deprecated.RemoveAfter)) < 0 {
		change := newChange("EVOL014", "deprecation-window-violated", packagePath, after.TypeName, fieldPath, profile, direction, Breaking, field.Deprecated.RemoveAfter, headVersion, "field was removed before its remove-after release", "restore the field until the declared removal release")
		return &change
	}
	minimum := cfg.Evolution.MinimumDeprecationReleases
	if minimum == 0 {
		return nil
	}
	if field.Deprecated.Since == "" || !validSemver(field.Deprecated.Since) || !validSemver(headVersion) || len(cfg.Evolution.ReleaseHistory) == 0 {
		change := newChange("EVOL014", "deprecation-window-unverifiable", packagePath, after.TypeName, fieldPath, profile, direction, Unknown, field.Deprecated, map[string]any{"base_version": baseVersion, "head_version": headVersion}, "deprecation release window cannot be verified from the supplied release history", "record semantic module versions and an ordered evolution.release_history")
		return &change
	}
	count := 0
	since, head := normalizeVersion(field.Deprecated.Since), normalizeVersion(headVersion)
	for _, release := range cfg.Evolution.ReleaseHistory {
		candidate := normalizeVersion(release)
		if semver.Compare(candidate, since) >= 0 && semver.Compare(candidate, head) < 0 {
			count++
		}
	}
	if count < minimum {
		change := newChange("EVOL014", "deprecation-window-violated", packagePath, after.TypeName, fieldPath, profile, direction, Breaking, count, minimum, fmt.Sprintf("field was deprecated for %d release(s), fewer than the required %d", count, minimum), "retain the field for the configured number of releases")
		return &change
	}
	return nil
}

func normalizeVersion(value string) string {
	value = strings.TrimSpace(value)
	if value != "" && !strings.HasPrefix(value, "v") {
		value = "v" + value
	}
	return value
}

func validSemver(value string) bool { return semver.IsValid(normalizeVersion(value)) }
func newChange(id, kind, packagePath, typeName, fieldPath, profile string, direction Direction, severity Severity, before, after any, message, recommendation string) Change {
	return Change{ID: id, Kind: kind, Package: packagePath, TypeName: typeName, FieldPath: fieldPath, Profile: profile, Direction: direction, Severity: severity, Before: before, After: after, Message: message, Recommendation: recommendation}
}
func metadata(document snapshot.Snapshot) SnapshotMetadata {
	return SnapshotMetadata{Module: document.ModulePath, Fingerprint: document.Fingerprint, GoVersion: document.GoVersion, Profiles: append([]string(nil), document.SemanticProfiles...)}
}
func surfaceIndex(value snapshot.ContractSnapshot) map[string]snapshot.SurfaceSnapshot {
	result := map[string]snapshot.SurfaceSnapshot{}
	for _, surface := range value.Profiles {
		result[surface.Profile] = surface
	}
	return result
}
func fieldByGo(values []snapshot.FieldSnapshot) map[string]snapshot.FieldSnapshot {
	result := map[string]snapshot.FieldSnapshot{}
	for _, field := range values {
		result[field.GoName] = field
	}
	return result
}
func unionKeys[T any](first, second map[string]T) []string {
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
func mode(level string, cfg config.Config) Direction {
	value := cfg.Compatibility.DefaultMode
	if item, ok := cfg.Compatibility.ContractLevels[level]; ok {
		switch item.Policy {
		case "storage":
			value = "storage"
		case "advisory":
			value = "advisory"
		}
	}
	switch value {
	case "producer":
		return Producer
	case "consumer":
		return Consumer
	case "storage":
		return Storage
	case "configuration":
		return Configuration
	case "message":
		return Message
	case "advisory":
		return Advisory
	default:
		return Bidirectional
	}
}
func addedSeverity(field snapshot.FieldSnapshot, direction Direction) Severity {
	if field.Required {
		return PotentiallyBreaking
	}
	if direction == Consumer {
		return PotentiallyBreaking
	}
	return Compatible
}
func certaintyRank(value any) int {
	switch fmt.Sprint(value) {
	case "exact":
		return 3
	case "partial":
		return 2
	default:
		return 1
	}
}
func optionalSurface(value snapshot.SurfaceSnapshot, ok bool) any {
	if !ok {
		return nil
	}
	return value.Profile
}
func sortChanges(values []Change) {
	sort.Slice(values, func(i, j int) bool {
		left, right := values[i], values[j]
		if left.Package != right.Package {
			return left.Package < right.Package
		}
		if left.TypeName != right.TypeName {
			return left.TypeName < right.TypeName
		}
		if left.FieldPath != right.FieldPath {
			return left.FieldPath < right.FieldPath
		}
		if left.Profile != right.Profile {
			return left.Profile < right.Profile
		}
		if left.ID != right.ID {
			return left.ID < right.ID
		}
		return left.Message < right.Message
	})
}
func summarize(values []Change) Summary {
	var result Summary
	for _, change := range values {
		switch change.Severity {
		case Compatible:
			result.Compatible++
		case PotentiallyBreaking:
			result.PotentiallyBreaking++
		case Breaking:
			result.Breaking++
		case Unknown:
			result.Unknown++
		default:
			result.Informational++
		}
	}
	return result
}

var _ = strings.Builder{}
