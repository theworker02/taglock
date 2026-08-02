// Package namespace describes tag dialect syntax and field-surface behavior.
package namespace

import "go/types"

// OptionSpec documents one recognized comma option.
type OptionSpec struct {
	Name string
}

// NamingBehavior controls the default name of an unrenamed field.
type NamingBehavior int

const (
	GoFieldName NamingBehavior = iota
	ExplicitOnly
)

// Spec defines one tag namespace.
type Spec struct {
	Name            string
	KnownOptions    map[string]OptionSpec
	IgnoreValue     string
	DefaultNaming   NamingBehavior
	Serialized      bool
	Public          bool
	SupportsInline  bool
	SupportsOmit    bool
	InlineDefault   bool
	CaseInsensitive bool
	CanInline       func(types.Type) bool
}

// Registry is an immutable collection of namespace specifications.
type Registry struct{ specs map[string]Spec }

// NewRegistry builds the built-in registry and overlays configured generic namespaces.
func NewRegistry(generic map[string][]string) Registry {
	specs := builtins()
	for name, options := range generic {
		if _, exists := specs[name]; exists {
			continue
		}
		known := make(map[string]OptionSpec, len(options))
		for _, option := range options {
			known[option] = OptionSpec{Name: option}
		}
		specs[name] = Spec{Name: name, KnownOptions: known, IgnoreValue: "-", DefaultNaming: ExplicitOnly, Serialized: true}
	}
	return Registry{specs: specs}
}

// Lookup returns a namespace specification.
func (r Registry) Lookup(name string) (Spec, bool) { spec, ok := r.specs[name]; return spec, ok }

func builtins() map[string]Spec {
	inlineTarget := func(t types.Type) bool {
		if pointer, ok := t.(*types.Pointer); ok {
			t = pointer.Elem()
		}
		switch t.Underlying().(type) {
		case *types.Struct, *types.Map:
			return true
		}
		return false
	}
	return map[string]Spec{
		"json": {Name: "json", KnownOptions: options("omitempty", "omitzero", "string"), IgnoreValue: "-", DefaultNaming: GoFieldName, Serialized: true, Public: true, SupportsOmit: true, InlineDefault: true, CaseInsensitive: true},
		"yaml": {Name: "yaml", KnownOptions: options("omitempty", "flow", "inline"), IgnoreValue: "-", DefaultNaming: GoFieldName, Serialized: true, Public: true, SupportsInline: true, SupportsOmit: true, CanInline: inlineTarget},
		"xml":  {Name: "xml", KnownOptions: options("omitempty", "attr", "chardata", "cdata", "innerxml", "comment", "any"), IgnoreValue: "-", DefaultNaming: GoFieldName, Serialized: true, Public: true, SupportsOmit: true, InlineDefault: true},
		"db":   generic("db"), "sql": generic("sql"), "form": publicGeneric("form"), "query": publicGeneric("query"), "mapstructure": publicGeneric("mapstructure"),
		"validate": {Name: "validate", KnownOptions: map[string]OptionSpec{}, IgnoreValue: "-"},
	}
}

func generic(name string) Spec {
	return Spec{Name: name, KnownOptions: map[string]OptionSpec{}, IgnoreValue: "-", DefaultNaming: ExplicitOnly, Serialized: true}
}
func publicGeneric(name string) Spec { spec := generic(name); spec.Public = true; return spec }
func options(values ...string) map[string]OptionSpec {
	result := make(map[string]OptionSpec, len(values))
	for _, value := range values {
		result[value] = OptionSpec{Name: value}
	}
	return result
}
