package contract

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"github.com/theworker02/taglock/config"
	"github.com/theworker02/taglock/namespace"
	"github.com/theworker02/taglock/rule"
	"github.com/theworker02/taglock/tag"
)

// BuildPackage parses every struct declaration once and resolves all surfaces.
func BuildPackage(files []*ast.File, info *types.Info, pkg *types.Package, cfg config.Config, registry namespace.Registry) ([]*StructContract, []Issue) {
	var contracts []*StructContract
	var issues []Issue
	byType := make(map[*types.Struct]*StructContract)
	for _, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			structNode, ok := node.(*ast.StructType)
			if !ok {
				return true
			}
			structType, ok := info.TypeOf(structNode).Underlying().(*types.Struct)
			if !ok {
				return true
			}
			name, comments, declared := structMetadata(file, structNode, info)
			suppression, suppressionIssues := parseSuppression(comments)
			issues = append(issues, suppressionIssues...)
			contract := &StructContract{TypeName: name, Package: pkg.Path(), Position: structNode.Pos(), End: structNode.End(), File: file, GoType: structType, DeclaredType: declared, Exported: ast.IsExported(name), Documentation: commentText(comments), Level: contractLevel(comments), Namespaces: map[string]*NamespaceContract{}, Suppression: suppression}
			contract.Fields, suppressionIssues = buildFields(structNode, structType)
			issues = append(issues, suppressionIssues...)
			contracts = append(contracts, contract)
			byType[structType] = contract
			return true
		})
	}
	for _, item := range contracts {
		resolveContract(item, byType, cfg, registry)
	}
	return contracts, issues
}

func structMetadata(file *ast.File, target *ast.StructType, info *types.Info) (string, *ast.CommentGroup, types.Type) {
	name := "<anonymous>"
	var comments *ast.CommentGroup
	for _, declaration := range file.Decls {
		generic, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range generic.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Type != target {
				continue
			}
			name = typeSpec.Name.Name
			comments = typeSpec.Doc
			if comments == nil {
				comments = generic.Doc
			}
			var declared types.Type
			if object, ok := info.Defs[typeSpec.Name].(*types.TypeName); ok {
				declared = object.Type()
			}
			return name, comments, declared
		}
	}
	return name, comments, info.TypeOf(target)
}

func buildFields(node *ast.StructType, structType *types.Struct) ([]*FieldContract, []Issue) {
	fields := make([]*FieldContract, 0, structType.NumFields())
	var issues []Issue
	typeIndex := 0
	for _, source := range node.Fields.List {
		count := len(source.Names)
		if count == 0 {
			count = 1
		}
		var raw, literal string
		var tagPos, tagEnd token.Pos
		if source.Tag != nil {
			literal = source.Tag.Value
			tagPos, tagEnd = source.Tag.Pos(), source.Tag.End()
			if unquoted, err := strconv.Unquote(literal); err == nil {
				raw = unquoted
			} else {
				raw = literal
			}
		}
		parsed := tag.Parse(raw)
		suppression, suppressionIssues := parseSuppression(source.Doc)
		issues = append(issues, suppressionIssues...)
		for range count {
			if typeIndex >= structType.NumFields() {
				break
			}
			variable := structType.Field(typeIndex)
			typeIndex++
			field := &FieldContract{GoName: variable.Name(), GoType: variable.Type(), Position: variable.Pos(), End: source.End(), TagPosition: tagPos, TagEnd: tagEnd, TagLiteral: literal, Exported: variable.Exported(), Anonymous: variable.Embedded(), Parsed: parsed, Tags: map[string]*TagContract{}, Suppression: suppression, Documentation: commentText(source.Doc), Deprecation: parseDeprecation(source.Doc), Schema: parseSchema(source.Doc)}
			for _, value := range parsed.Values {
				if _, exists := field.Tags[value.Namespace]; exists {
					continue
				}
				field.Tags[value.Namespace] = &TagContract{Namespace: value.Namespace, Raw: value.Raw, Name: value.Name, Ignored: value.Ignored, Options: append([]string(nil), value.Options...), Position: tagPos, End: tagEnd, Explicit: true}
			}
			fields = append(fields, field)
		}
	}
	return fields, issues
}

func commentText(group *ast.CommentGroup) string {
	if group == nil {
		return ""
	}
	return strings.TrimSpace(group.Text())
}
func contractLevel(group *ast.CommentGroup) string {
	if group == nil {
		return ""
	}
	for _, comment := range group.List {
		text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
		if strings.HasPrefix(text, "taglock:contract ") {
			return strings.TrimSpace(strings.TrimPrefix(text, "taglock:contract "))
		}
	}
	return ""
}
func parseDeprecation(group *ast.CommentGroup) *Deprecation {
	if group == nil {
		return nil
	}
	text := group.Text()
	var result *Deprecation
	if strings.Contains(text, "Deprecated:") {
		result = &Deprecation{Note: strings.TrimSpace(text)}
	}
	for _, comment := range group.List {
		value := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
		if !strings.HasPrefix(value, "taglock:deprecated") {
			continue
		}
		if result == nil {
			result = &Deprecation{}
		}
		for _, part := range strings.Fields(strings.TrimSpace(strings.TrimPrefix(value, "taglock:deprecated"))) {
			key, val, ok := strings.Cut(part, "=")
			if !ok {
				continue
			}
			switch key {
			case "since":
				result.Since = val
			case "remove-after":
				result.RemoveAfter = val
			case "replacement":
				result.Replacement = val
			}
		}
	}
	return result
}
func parseSchema(group *ast.CommentGroup) map[string]string {
	result := map[string]string{}
	if group == nil {
		return result
	}
	for _, comment := range group.List {
		value := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
		if !strings.HasPrefix(value, "taglock:schema") {
			continue
		}
		for _, part := range strings.Fields(strings.TrimSpace(strings.TrimPrefix(value, "taglock:schema"))) {
			key, val, ok := strings.Cut(part, "=")
			if ok {
				result[key] = val
			}
		}
	}
	return result
}

func parseSuppression(group *ast.CommentGroup) (*Suppression, []Issue) {
	if group == nil {
		return nil, nil
	}
	var found *Suppression
	var issues []Issue
	for _, comment := range group.List {
		text := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(comment.Text, "//"), "/*"))
		text = strings.TrimSuffix(text, "*/")
		if !strings.HasPrefix(text, "taglock:") {
			continue
		}
		if strings.HasPrefix(text, "taglock:contract") || strings.HasPrefix(text, "taglock:deprecated") || strings.HasPrefix(text, "taglock:schema") {
			continue
		}
		if !strings.HasPrefix(text, "taglock:ignore") {
			issues = append(issues, Issue{Position: comment.Pos(), End: comment.End(), Message: fmt.Sprintf("unknown suppression directive %q", text)})
			continue
		}
		remainder := strings.TrimSpace(strings.TrimPrefix(text, "taglock:ignore"))
		idPart, reason, _ := strings.Cut(remainder, "--")
		tokens := strings.Fields(idPart)
		candidate := &Suppression{Position: comment.Pos(), End: comment.End(), IDs: map[string]bool{}, Used: map[string]bool{}, Reason: strings.TrimSpace(reason), Raw: text}
		if len(tokens) == 0 {
			issues = append(issues, Issue{Position: comment.Pos(), End: comment.End(), Message: "suppression must name at least one rule identifier", Suppression: candidate})
			continue
		}
		valid := true
		for _, id := range tokens {
			id = strings.TrimSuffix(id, ",")
			id = strings.ToUpper(id)
			if !rule.Known(id) {
				issues = append(issues, Issue{Position: comment.Pos(), End: comment.End(), Message: fmt.Sprintf("suppression references unknown rule %q", id), Suppression: candidate})
				valid = false
				continue
			}
			candidate.IDs[id] = true
		}
		if valid {
			found = candidate
		}
	}
	return found, issues
}

func resolveContract(item *StructContract, byType map[*types.Struct]*StructContract, cfg config.Config, registry namespace.Registry) {
	for _, namespaceName := range cfg.SortedNamespaces() {
		spec, ok := registry.Lookup(namespaceName)
		if !ok {
			continue
		}
		surface := &NamespaceContract{Spec: spec}
		for _, field := range item.Fields {
			surface.Fields = append(surface.Fields, resolveField(field, namespaceName, spec, byType, []string{field.GoName}, 0, map[*types.Struct]bool{})...)
		}
		markShadowed(surface.Fields, namespaceName)
		item.Namespaces[namespaceName] = surface
	}
}

func resolveField(field *FieldContract, namespaceName string, spec namespace.Spec, byType map[*types.Struct]*StructContract, path []string, depth int, visiting map[*types.Struct]bool) []*EffectiveField {
	tagContract, tagged := field.Tags[namespaceName]
	if tagged && tagContract.Ignored {
		return []*EffectiveField{{Field: field, Name: "-", Path: append([]string(nil), path...), Depth: depth, Explicit: true, Ignored: true, Tag: tagContract}}
	}
	explicitName := tagged && tagContract.Name != ""
	inline := field.Anonymous && !explicitName && spec.InlineDefault
	if tagged && spec.SupportsInline && tagContract.HasOption("inline") && !explicitName {
		inline = true
	}
	if inline {
		if nestedType := underlyingStruct(field.GoType); nestedType != nil && !visiting[nestedType] {
			visiting[nestedType] = true
			defer delete(visiting, nestedType)
			nested := byType[nestedType]
			nestedFields := []*FieldContract(nil)
			if nested != nil {
				nestedFields = nested.Fields
			} else {
				nestedFields = fieldsFromTypes(nestedType)
			}
			var result []*EffectiveField
			for _, child := range nestedFields {
				childPath := append(append([]string(nil), path...), child.GoName)
				child.EmbeddedPath = append([]string(nil), childPath[:len(childPath)-1]...)
				result = append(result, resolveField(child, namespaceName, spec, byType, childPath, depth+1, visiting)...)
			}
			return result
		}
	}
	name := field.GoName
	if explicitName {
		name = tagContract.Name
	} else if spec.DefaultNaming == namespace.ExplicitOnly && !tagged {
		return nil
	}
	return []*EffectiveField{{Field: field, Name: name, Path: append([]string(nil), path...), Depth: depth, Explicit: explicitName, Tag: tagContract}}
}

func fieldsFromTypes(structType *types.Struct) []*FieldContract {
	result := make([]*FieldContract, 0, structType.NumFields())
	for index := 0; index < structType.NumFields(); index++ {
		variable := structType.Field(index)
		parsed := tag.Parse(structType.Tag(index))
		field := &FieldContract{GoName: variable.Name(), GoType: variable.Type(), Position: variable.Pos(), Exported: variable.Exported(), Anonymous: variable.Embedded(), Parsed: parsed, Tags: map[string]*TagContract{}}
		for _, value := range parsed.Values {
			if _, exists := field.Tags[value.Namespace]; !exists {
				field.Tags[value.Namespace] = &TagContract{Namespace: value.Namespace, Raw: value.Raw, Name: value.Name, Ignored: value.Ignored, Options: append([]string(nil), value.Options...), Position: variable.Pos(), End: variable.Pos(), Explicit: true}
			}
		}
		result = append(result, field)
	}
	return result
}

func underlyingStruct(fieldType types.Type) *types.Struct {
	if pointer, ok := fieldType.(*types.Pointer); ok {
		fieldType = pointer.Elem()
	}
	result, _ := fieldType.Underlying().(*types.Struct)
	return result
}

func markShadowed(fields []*EffectiveField, namespaceName string) {
	groups := map[string][]*EffectiveField{}
	for _, field := range fields {
		if !field.Ignored {
			groups[field.Name] = append(groups[field.Name], field)
		}
	}
	for _, group := range groups {
		minimum := int(^uint(0) >> 1)
		for _, field := range group {
			if field.Depth < minimum {
				minimum = field.Depth
			}
		}
		var atDepth []*EffectiveField
		for _, field := range group {
			if field.Depth == minimum {
				atDepth = append(atDepth, field)
			} else {
				field.Shadowed = true
			}
		}
		if namespaceName == "json" {
			explicit := []*EffectiveField{}
			for _, field := range atDepth {
				if field.Explicit {
					explicit = append(explicit, field)
				}
			}
			if len(explicit) == 1 {
				for _, field := range atDepth {
					if field != explicit[0] {
						field.Shadowed = true
					}
				}
			}
		}
	}
}

// FormatPath returns a stable promoted field identity.
func FormatPath(typeName string, field *EffectiveField) string { return field.PathString(typeName) }

// TypeString provides stable type text for diagnostics and fingerprints.
func TypeString(field *FieldContract) string {
	return types.TypeString(field.GoType, func(pkg *types.Package) string {
		if pkg == nil {
			return ""
		}
		return pkg.Path()
	})
}
