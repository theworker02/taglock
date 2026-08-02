// Package contract builds normalized semantic representations of Go structs.
package contract

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/theworker02/taglock/namespace"
	"github.com/theworker02/taglock/tag"
)

// StructContract is the normalized contract for one Go struct declaration.
type StructContract struct {
	TypeName      string
	Package       string
	Position      token.Pos
	End           token.Pos
	File          *ast.File
	GoType        *types.Struct
	DeclaredType  types.Type
	Exported      bool
	Documentation string
	Level         string
	Fields        []*FieldContract
	Namespaces    map[string]*NamespaceContract
	Suppression   *Suppression
}

// FieldContract describes one source field and its parsed tags.
type FieldContract struct {
	GoName        string
	GoType        types.Type
	Position      token.Pos
	End           token.Pos
	TagPosition   token.Pos
	TagEnd        token.Pos
	TagLiteral    string
	Exported      bool
	Anonymous     bool
	EmbeddedPath  []string
	Tags          map[string]*TagContract
	Parsed        tag.Parsed
	Suppression   *Suppression
	Documentation string
	Deprecation   *Deprecation
	Schema        map[string]string
}

type Deprecation struct {
	Since       string
	RemoveAfter string
	Replacement string
	Note        string
}

// TagContract is one normalized namespace value.
type TagContract struct {
	Namespace string
	Raw       string
	Name      string
	Ignored   bool
	Options   []string
	Position  token.Pos
	End       token.Pos
	Explicit  bool
}

func (t *TagContract) HasOption(option string) bool {
	for _, candidate := range t.Options {
		if candidate == option {
			return true
		}
	}
	return false
}

// NamespaceContract is the complete effective external field surface.
type NamespaceContract struct {
	Spec   namespace.Spec
	Fields []*EffectiveField
}

// EffectiveField records a field after promotion and naming resolution.
type EffectiveField struct {
	Field    *FieldContract
	Name     string
	Path     []string
	Depth    int
	Explicit bool
	Ignored  bool
	Shadowed bool
	Tag      *TagContract
}

func (f *EffectiveField) PathString(typeName string) string {
	result := typeName
	for _, segment := range f.Path {
		if result == "" {
			result = segment
		} else {
			result += "." + segment
		}
	}
	return result
}

// Suppression applies rule IDs to its declaration.
type Suppression struct {
	Position token.Pos
	End      token.Pos
	IDs      map[string]bool
	Used     map[string]bool
	Reason   string
	Raw      string
}

// Issue is a contract-building diagnostic, currently used by suppression parsing.
type Issue struct {
	Position, End token.Pos
	Message       string
	Suppression   *Suppression
}
