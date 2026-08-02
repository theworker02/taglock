package semantics

import (
	"fmt"
	"go/types"
	"sort"
	"strings"

	"github.com/magnexis/taglock/contract"
	"github.com/magnexis/taglock/tag"
)

type jsonV1 struct{ toolchain string }
type jsonV2 struct {
	toolchain string
	available bool
}

func NewJSONV1(toolchain string) Profile                 { return jsonV1{toolchain} }
func NewJSONV2(toolchain string, available bool) Profile { return jsonV2{toolchain, available} }
func (jsonV1) ID() string                                { return "json/v1" }
func (jsonV1) Namespace() string                         { return "json" }
func (jsonV1) Version() string                           { return "v1" }
func (jsonV1) Available() bool                           { return true }
func (jsonV2) ID() string                                { return "json/v2" }
func (jsonV2) Namespace() string                         { return "json" }
func (jsonV2) Version() string                           { return "v2" }
func (p jsonV2) Available() bool                         { return p.available }

func (p jsonV1) ResolveStruct(item *contract.StructContract) (*ResolvedSurface, error) {
	surface := &ResolvedSurface{Profile: p.ID(), Namespace: "json", Certainty: CertaintyExact, CaseSensitiveDecode: false, Available: true, Toolchain: p.toolchain}
	applyCustomCertainty(surface, item)
	namespaceSurface := item.Namespaces["json"]
	if namespaceSurface == nil {
		return surface, nil
	}
	groups := map[string][]*contract.EffectiveField{}
	for _, field := range namespaceSurface.Fields {
		if field.Ignored || field.Shadowed {
			continue
		}
		groups[field.Name] = append(groups[field.Name], field)
	}
	names := sortedKeys(groups)
	for _, name := range names {
		fields := groups[name]
		selected := selectDominant(fields)
		if selected == nil {
			continue
		}
		surface.Fields = append(surface.Fields, resolvedFromContract(selected))
	}
	appendIgnored(surface, item)
	return surface, nil
}
func (p jsonV1) ValidateField(field *contract.FieldContract) []Finding {
	var result []Finding
	if value := field.Tags["json"]; value != nil && value.HasOption("inline") {
		result = append(result, Finding{ID: "JSONMIG001", FieldPath: field.GoName, Message: "json/v1 does not recognize the inline option"})
	}
	return result
}
func (p jsonV1) Compare(other Profile) []SemanticDifference {
	if other.ID() == "json/v2" {
		return []SemanticDifference{{Kind: "name-matching", Before: "case-insensitive", After: "case-sensitive", Message: "json/v1 decodes field names case-insensitively while json/v2 is strict by default"}}
	}
	return nil
}

func (p jsonV2) ResolveStruct(item *contract.StructContract) (*ResolvedSurface, error) {
	surface := &ResolvedSurface{Profile: p.ID(), Namespace: "json", Certainty: CertaintyExact, CaseSensitiveDecode: true, Available: p.available, Toolchain: p.toolchain}
	if !p.available {
		surface.Reason = "encoding/json/v2 exists in this toolchain but GOEXPERIMENT=jsonv2 is not active"
	}
	applyCustomCertainty(surface, item)
	candidates := resolveV2(item)
	groups := map[string][]v2Candidate{}
	for _, candidate := range candidates {
		groups[candidate.name] = append(groups[candidate.name], candidate)
	}
	for _, name := range sortedCandidateKeys(groups) {
		selected := selectV2(groups[name])
		if selected == nil {
			continue
		}
		surface.Fields = append(surface.Fields, selected.field)
	}
	appendIgnored(surface, item)
	return surface, nil
}

func appendIgnored(surface *ResolvedSurface, item *contract.StructContract) {
	for _, field := range item.Fields {
		if value := field.Tags["json"]; value != nil && value.Ignored {
			surface.Fields = append(surface.Fields, ResolvedField{GoName: field.GoName, Name: "-", GoType: field.GoType, TypeString: contract.TypeString(field), Path: []string{field.GoName}, Ignored: true, Deprecated: field.Deprecation, Schema: field.Schema})
		}
	}
}
func (p jsonV2) ValidateField(field *contract.FieldContract) []Finding {
	value := field.Tags["json"]
	if value == nil {
		return nil
	}
	known := func(option string) bool {
		return option == "omitempty" || option == "omitzero" || option == "string" || option == "inline" || option == "unknown" || strings.HasPrefix(option, "case:") || strings.HasPrefix(option, "format:")
	}
	var result []Finding
	for _, option := range value.Options {
		if !known(option) {
			result = append(result, Finding{ID: "JSONMIG001", FieldPath: field.GoName, Message: fmt.Sprintf("json/v2 does not recognize option %q", option)})
		}
	}
	if value.HasOption("inline") && len(value.Options) > 1 {
		result = append(result, Finding{ID: "JSONMIG001", FieldPath: field.GoName, Message: "json/v2 inline cannot be combined with other options"})
	}
	return result
}
func (p jsonV2) Compare(other Profile) []SemanticDifference { return other.Compare(p) }

func applyCustomCertainty(surface *ResolvedSurface, item *contract.StructContract) {
	methods := detectMethods(item.DeclaredType)
	surface.CustomMethods = methods
	if methods.MarshalJSON || methods.UnmarshalJSON {
		surface.Certainty = CertaintyOpaque
		surface.Reason = "custom MarshalJSON or UnmarshalJSON controls representation"
	} else if methods.MarshalText || methods.UnmarshalText {
		surface.Certainty = CertaintyPartial
		surface.Reason = "text marshaling methods affect scalar representation"
	}
}
func detectMethods(value types.Type) CustomMethods {
	var result CustomMethods
	if value == nil {
		return result
	}
	sets := []*types.MethodSet{types.NewMethodSet(value)}
	if _, ok := value.(*types.Pointer); !ok {
		sets = append(sets, types.NewMethodSet(types.NewPointer(value)))
	}
	for _, set := range sets {
		for index := 0; index < set.Len(); index++ {
			switch set.At(index).Obj().Name() {
			case "MarshalJSON":
				result.MarshalJSON = true
			case "UnmarshalJSON":
				result.UnmarshalJSON = true
			case "MarshalText":
				result.MarshalText = true
			case "UnmarshalText":
				result.UnmarshalText = true
			}
		}
	}
	return result
}

func selectDominant(fields []*contract.EffectiveField) *contract.EffectiveField {
	if len(fields) == 1 {
		return fields[0]
	}
	minimum := fields[0].Depth
	for _, field := range fields[1:] {
		if field.Depth < minimum {
			minimum = field.Depth
		}
	}
	var shallow []*contract.EffectiveField
	for _, field := range fields {
		if field.Depth == minimum {
			shallow = append(shallow, field)
		}
	}
	if len(shallow) == 1 {
		return shallow[0]
	}
	var explicit []*contract.EffectiveField
	for _, field := range shallow {
		if field.Explicit {
			explicit = append(explicit, field)
		}
	}
	if len(explicit) == 1 {
		return explicit[0]
	}
	return nil
}
func resolvedFromContract(field *contract.EffectiveField) ResolvedField {
	result := ResolvedField{GoName: field.Field.GoName, Name: field.Name, GoType: field.Field.GoType, TypeString: contract.TypeString(field.Field), Path: append([]string(nil), field.Path...), Required: true, Nullable: isNullable(field.Field.GoType), Embedded: field.Depth > 0, Deprecated: field.Field.Deprecation, Schema: field.Field.Schema}
	if field.Tag != nil {
		result.OmitEmpty = field.Tag.HasOption("omitempty")
		result.OmitZero = field.Tag.HasOption("omitzero")
		result.Required = !result.OmitEmpty && !result.OmitZero
	}
	return result
}
func isNullable(value types.Type) bool {
	switch value.Underlying().(type) {
	case *types.Pointer, *types.Slice, *types.Map, *types.Interface, *types.Signature, *types.Chan:
		return true
	}
	return false
}

type v2Candidate struct {
	name     string
	depth    int
	explicit bool
	field    ResolvedField
}
type v2Queue struct {
	structType *types.Struct
	path       []string
	depth      int
	visiting   map[*types.Struct]bool
}

func resolveV2(item *contract.StructContract) []v2Candidate {
	var result []v2Candidate
	top := map[string]*contract.FieldContract{}
	for _, field := range item.Fields {
		top[field.GoName] = field
	}
	var walk func(*types.Struct, []string, int, map[*types.Struct]bool)
	walk = func(structType *types.Struct, prefix []string, depth int, visiting map[*types.Struct]bool) {
		if visiting[structType] {
			return
		}
		next := cloneVisited(visiting)
		next[structType] = true
		for index := 0; index < structType.NumFields(); index++ {
			variable := structType.Field(index)
			if !variable.Exported() {
				continue
			}
			parsed := tag.Parse(structType.Tag(index))
			value, tagged := parsed.First("json")
			if tagged && value.Ignored {
				continue
			}
			path := append(append([]string(nil), prefix...), variable.Name())
			explicit := tagged && value.Name != ""
			inline := variable.Embedded() && !explicit
			if tagged && value.HasOption("inline") {
				inline = true
			}
			if inline {
				if nested := underlyingStruct(variable.Type()); nested != nil {
					walk(nested, path, depth+1, next)
					continue
				}
			}
			name := variable.Name()
			if explicit {
				name = value.Name
			}
			source := top[variable.Name()]
			if depth > 0 || source == nil {
				source = &contract.FieldContract{GoName: variable.Name(), GoType: variable.Type(), Exported: variable.Exported(), Anonymous: variable.Embedded(), Tags: map[string]*contract.TagContract{}, Schema: map[string]string{}}
				if tagged {
					source.Tags["json"] = &contract.TagContract{Namespace: "json", Name: value.Name, Raw: value.Raw, Ignored: value.Ignored, Options: value.Options, Explicit: true}
				}
			}
			effective := &contract.EffectiveField{Field: source, Name: name, Path: path, Depth: depth, Explicit: explicit, Tag: source.Tags["json"]}
			result = append(result, v2Candidate{name: name, depth: depth, explicit: explicit, field: resolvedFromContract(effective)})
		}
	}
	walk(item.GoType, nil, 0, map[*types.Struct]bool{})
	return result
}
func selectV2(fields []v2Candidate) *v2Candidate {
	if len(fields) == 1 {
		return &fields[0]
	}
	minimum := fields[0].depth
	for _, field := range fields[1:] {
		if field.depth < minimum {
			minimum = field.depth
		}
	}
	var shallow []v2Candidate
	for _, field := range fields {
		if field.depth == minimum {
			shallow = append(shallow, field)
		}
	}
	if len(shallow) == 1 {
		return &shallow[0]
	}
	var explicit []v2Candidate
	for _, field := range shallow {
		if field.explicit {
			explicit = append(explicit, field)
		}
	}
	if len(explicit) == 1 {
		return &explicit[0]
	}
	return nil
}
func underlyingStruct(value types.Type) *types.Struct {
	if pointer, ok := value.(*types.Pointer); ok {
		value = pointer.Elem()
	}
	result, _ := value.Underlying().(*types.Struct)
	return result
}
func cloneVisited(source map[*types.Struct]bool) map[*types.Struct]bool {
	result := make(map[*types.Struct]bool, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
func sortedKeys(values map[string][]*contract.EffectiveField) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}
func sortedCandidateKeys(values map[string][]v2Candidate) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}
