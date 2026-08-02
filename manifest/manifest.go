// Package manifest creates compact publishable contract indexes.
package manifest

import (
	"encoding/json"
	"io"
	"sort"
	"strings"

	"github.com/magnexis/taglock/semantics"
	"github.com/magnexis/taglock/snapshot"
)

type Manifest struct {
	FormatVersion int       `json:"format_version"`
	Module        string    `json:"module"`
	Profiles      []string  `json:"profiles"`
	Packages      []Package `json:"packages"`
}
type Package struct {
	ImportPath   string     `json:"import_path"`
	Contracts    []Contract `json:"contracts"`
	Dependencies []string   `json:"dependencies,omitempty"`
}
type Contract struct {
	Type             string                      `json:"type"`
	Level            string                      `json:"level"`
	Fingerprint      string                      `json:"fingerprint"`
	Certainty        semantics.ContractCertainty `json:"certainty"`
	DeprecatedFields []string                    `json:"deprecated_fields,omitempty"`
}

func Build(document snapshot.Snapshot) Manifest {
	result := Manifest{FormatVersion: 1, Module: document.ModulePath, Profiles: append([]string(nil), document.SemanticProfiles...)}
	knownPackages := make([]string, 0, len(document.Packages))
	for _, pkg := range document.Packages {
		knownPackages = append(knownPackages, pkg.ImportPath)
	}
	for _, pkg := range document.Packages {
		entry := Package{ImportPath: pkg.ImportPath}
		dependencies := map[string]bool{}
		for _, item := range pkg.Contracts {
			certainty := semantics.CertaintyExact
			deprecated := []string{}
			for _, surface := range item.Profiles {
				if surface.Certainty == semantics.CertaintyOpaque {
					certainty = semantics.CertaintyOpaque
				} else if surface.Certainty == semantics.CertaintyPartial && certainty == semantics.CertaintyExact {
					certainty = semantics.CertaintyPartial
				}
				for _, field := range surface.Fields {
					for _, candidate := range knownPackages {
						if candidate != pkg.ImportPath && strings.Contains(field.GoType, candidate+".") {
							dependencies[candidate] = true
						}
					}
					if field.Deprecated != nil {
						deprecated = append(deprecated, field.ExternalName)
					}
				}
			}
			sort.Strings(deprecated)
			entry.Contracts = append(entry.Contracts, Contract{Type: item.TypeName, Level: item.Level, Fingerprint: item.Fingerprint, Certainty: certainty, DeprecatedFields: deprecated})
		}
		for dependency := range dependencies {
			entry.Dependencies = append(entry.Dependencies, dependency)
		}
		sort.Strings(entry.Dependencies)
		result.Packages = append(result.Packages, entry)
	}
	return result
}
func Write(writer io.Writer, value Manifest) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
