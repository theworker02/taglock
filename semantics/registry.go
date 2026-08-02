package semantics

import (
	"fmt"
	"go/build"
	"runtime"
	"strconv"
	"strings"

	"github.com/magnexis/taglock/contract"
)

type Registry struct {
	profiles        map[string]Profile
	Toolchain       string
	JSONV2Installed bool
	JSONV2Active    bool
}

func NewRegistry() Registry {
	installed := goAtLeast(runtime.Version(), 1, 25)
	active := false
	for _, tag := range build.Default.ToolTags {
		if tag == "goexperiment.jsonv2" {
			active = true
		}
	}
	v1 := NewJSONV1(runtime.Version())
	v2 := NewJSONV2(runtime.Version(), installed && active)
	return Registry{profiles: map[string]Profile{"json/v1": v1, "json/v2": v2, "yaml/generic": newGeneric("yaml/generic", "yaml", runtime.Version()), "xml/stdlib": newGeneric("xml/stdlib", "xml", runtime.Version())}, Toolchain: runtime.Version(), JSONV2Installed: installed, JSONV2Active: active}
}
func (r Registry) Lookup(id string) (Profile, bool) {
	profile, ok := r.profiles[id]
	return profile, ok
}
func (r Registry) Select(value string) ([]Profile, error) {
	var ids []string
	switch value {
	case "", "auto", "v1":
		ids = []string{"json/v1"}
	case "v2":
		ids = []string{"json/v2"}
	case "both":
		ids = []string{"json/v1", "json/v2"}
	default:
		if _, ok := r.profiles[value]; ok {
			ids = []string{value}
		} else {
			return nil, fmt.Errorf("unknown semantic profile %q", value)
		}
	}
	result := make([]Profile, 0, len(ids))
	for _, id := range ids {
		result = append(result, r.profiles[id])
	}
	return result, nil
}

type GenericProfile string

func (p GenericProfile) ID() string { return strings.Split(string(p), "|")[0] }
func (p GenericProfile) Namespace() string {
	parts := strings.Split(string(p), "|")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}
func (p GenericProfile) Version() string { return "generic" }
func (p GenericProfile) Available() bool { return true }
func (p GenericProfile) ResolveStruct(*contract.StructContract) (*ResolvedSurface, error) {
	return &ResolvedSurface{Profile: p.ID(), Namespace: p.Namespace(), Certainty: CertaintyExact, Available: true, Toolchain: runtime.Version()}, nil
}
func (p GenericProfile) ValidateField(*contract.FieldContract) []Finding { return nil }
func (p GenericProfile) Compare(Profile) []SemanticDifference            { return nil }
func newGeneric(id, namespace, toolchain string) GenericProfile {
	return GenericProfile(id + "|" + namespace + "|" + toolchain)
}

func goAtLeast(version string, major, minor int) bool {
	value := strings.TrimPrefix(version, "go")
	parts := strings.Split(value, ".")
	if len(parts) < 2 {
		return false
	}
	gotMajor, _ := strconv.Atoi(parts[0])
	gotMinor, _ := strconv.Atoi(parts[1])
	return gotMajor > major || (gotMajor == major && gotMinor >= minor)
}

// keep constructor expression readable.
var _ = newGeneric
