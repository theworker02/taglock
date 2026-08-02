// Package registry describes the struct-tag dialects understood by TagLock.
package registry

// Namespace describes a tag namespace and its recognized comma options.
type Namespace struct {
	Name          string
	Serialized    bool
	Public        bool
	AllowedOption map[string]bool
}

var namespaces = map[string]Namespace{
	"json": {
		Name: "json", Serialized: true, Public: true,
		AllowedOption: options("omitempty", "omitzero", "string"),
	},
	"yaml": {
		Name: "yaml", Serialized: true, Public: true,
		AllowedOption: options("omitempty", "flow", "inline"),
	},
	"db": {
		Name: "db", Serialized: true,
		AllowedOption: options(),
	},
	"validate": {
		Name:          "validate",
		AllowedOption: options(),
	},
	"form": {
		Name: "form", Serialized: true, Public: true,
		AllowedOption: options("omitempty"),
	},
}

func options(values ...string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

// Lookup returns the built-in description for name.
func Lookup(name string) (Namespace, bool) {
	namespace, ok := namespaces[name]
	return namespace, ok
}

// Known reports whether TagLock understands a namespace.
func Known(name string) bool {
	_, ok := namespaces[name]
	return ok
}
