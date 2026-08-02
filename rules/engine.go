// Package rules evaluates normalized struct contracts using stable rule IDs.
package rules

import (
	"fmt"
	"go/token"
	"go/types"
	"sort"
	"strings"
	"unicode"

	"github.com/theworker02/taglock/config"
	"github.com/theworker02/taglock/contract"
	"github.com/theworker02/taglock/fix"
	"github.com/theworker02/taglock/rule"
	"github.com/theworker02/taglock/tag"
)

// Evaluate runs the complete registry against one normalized contract.
func Evaluate(item *contract.StructContract, cfg config.Config, buildIssues []contract.Issue) []Diagnostic {
	var diagnostics []Diagnostic
	for _, issue := range buildIssues {
		diagnostics = append(diagnostics, makeDiagnostic(item, nil, cfg, "TAG901", "", issue.Position, issue.End, issue.Message))
	}
	for _, field := range item.Fields {
		diagnostics = append(diagnostics, checkField(item, field, cfg)...)
	}
	for namespaceName, surface := range item.Namespaces {
		diagnostics = append(diagnostics, checkSurface(item, namespaceName, surface, cfg)...)
	}
	diagnostics = append(diagnostics, checkCrossNamespace(item, cfg)...)
	diagnostics = applySuppressions(item, diagnostics, cfg)
	diagnostics = append(diagnostics, unusedSuppressions(item, cfg)...)
	sort.SliceStable(diagnostics, func(i, j int) bool {
		if diagnostics[i].Pos != diagnostics[j].Pos {
			return diagnostics[i].Pos < diagnostics[j].Pos
		}
		if diagnostics[i].Rule.ID != diagnostics[j].Rule.ID {
			return diagnostics[i].Rule.ID < diagnostics[j].Rule.ID
		}
		return diagnostics[i].Message < diagnostics[j].Message
	})
	return suppressOverlappingSafeFixes(diagnostics)
}

func checkField(item *contract.StructContract, field *contract.FieldContract, cfg config.Config) []Diagnostic {
	var result []Diagnostic
	pos, end := fieldPosition(field)
	for _, problem := range field.Parsed.Problems {
		id := "TAG001"
		if problem.Kind == tag.ProblemDuplicateNamespace {
			id = "TAG002"
		}
		result = append(result, makeDiagnostic(item, field, cfg, id, problem.Namespace, pos, end, problem.Message))
	}
	if len(field.Parsed.Problems) > 0 {
		return result
	}
	if !field.Exported && len(field.Parsed.Values) > 0 {
		diagnostic := makeDiagnostic(item, field, cfg, "TAG101", "", pos, end, fmt.Sprintf("tag on unexported field %s has no effect in common serializers", field.GoName))
		if field.TagPosition != token.NoPos {
			diagnostic.Fixes = []fix.Suggestion{{Message: "remove the ineffective struct tag", Safety: rule.FixSafe, Edits: []fix.Edit{{Pos: field.TagPosition, End: field.TagEnd}}}}
		}
		result = append(result, diagnostic)
	}
	for _, namespaceName := range cfg.SortedNamespaces() {
		value, ok := field.Tags[namespaceName]
		policy := cfg.Namespace(namespaceName)
		if !ok {
			if policy.RequireExplicitTags && field.Exported && !field.Anonymous {
				result = append(result, makeDiagnostic(item, field, cfg, "TAG304", namespaceName, field.Position, field.End, fmt.Sprintf("exported field %s is missing required %s tag", field.GoName, namespaceName)))
			}
			continue
		}
		seen := map[string]bool{}
		duplicate := false
		for _, option := range value.Options {
			if seen[option] {
				duplicate = true
			}
			seen[option] = true
		}
		if duplicate {
			diagnostic := makeDiagnostic(item, field, cfg, "TAG003", namespaceName, pos, end, fmt.Sprintf("%s tag on %s repeats an option", namespaceName, field.GoName))
			if parsedValue, exists := field.Parsed.First(namespaceName); exists {
				if replacement, changed := tag.RemoveDuplicateOptions(parsedValue); changed {
					addTagRewrite(&diagnostic, field, namespaceName, replacement, rule.FixSafe, "remove duplicate tag options")
				}
			}
			result = append(result, diagnostic)
		}
		if value.Ignored && len(value.Options) > 0 {
			diagnostic := makeDiagnostic(item, field, cfg, "TAG103", namespaceName, pos, end, fmt.Sprintf("ignored %s field %s has meaningless options %q", namespaceName, field.GoName, strings.Join(value.Options, ",")))
			addTagRewrite(&diagnostic, field, namespaceName, "-", rule.FixSafe, "remove options after the ignore marker")
			result = append(result, diagnostic)
			continue
		}
		if value.Name == "" && len(value.Options) == 0 {
			result = append(result, makeDiagnostic(item, field, cfg, "TAG005", namespaceName, pos, end, fmt.Sprintf("%s tag on %s has an empty name and no options", namespaceName, field.GoName)))
		}
		if suspiciousName(value.Name, namespaceName) {
			result = append(result, makeDiagnostic(item, field, cfg, "TAG005", namespaceName, pos, end, fmt.Sprintf("%s tag name %q contains whitespace or control characters", namespaceName, value.Name)))
		}
		if policy.Naming != "" && value.Name != "" && !value.Ignored {
			expected := ConvertName(value.Name, policy.Naming)
			if expected != value.Name {
				diagnostic := makeDiagnostic(item, field, cfg, "TAG301", namespaceName, pos, end, fmt.Sprintf("%s name %q does not follow configured %s naming; expected %q", namespaceName, value.Name, policy.Naming, expected))
				replacement := expected
				if len(value.Options) > 0 {
					replacement += "," + strings.Join(value.Options, ",")
				}
				addTagRewrite(&diagnostic, field, namespaceName, replacement, rule.FixSafe, "apply configured naming convention")
				result = append(result, diagnostic)
			}
		}
		result = append(result, checkOptionsAndTypes(item, field, value, namespaceName, cfg)...)
	}
	result = append(result, checkSensitive(item, field, cfg)...)
	return result
}

func checkOptionsAndTypes(item *contract.StructContract, field *contract.FieldContract, value *contract.TagContract, namespaceName string, cfg config.Config) []Diagnostic {
	var result []Diagnostic
	pos, end := fieldPosition(field)
	known := builtinOptions(namespaceName)
	for _, option := range cfg.Namespace(namespaceName).KnownOptions {
		known[option] = true
	}
	if cfg.DisallowUnknownOptions && namespaceName != "validate" {
		for _, option := range value.Options {
			if option == "" || !optionKnown(namespaceName, option, known, cfg) {
				result = append(result, makeDiagnostic(item, field, cfg, "TAG004", namespaceName, pos, end, fmt.Sprintf("unknown %s tag option %q", namespaceName, option)))
			}
		}
	}
	underlying := field.GoType
	wasPointer := false
	if pointer, ok := underlying.(*types.Pointer); ok {
		wasPointer = true
		underlying = pointer.Elem()
	}
	underlying = underlying.Underlying()
	if namespaceName == "json" {
		if value.HasOption("string") && !jsonStringCompatible(underlying) {
			result = append(result, makeDiagnostic(item, field, cfg, "TAG201", namespaceName, pos, end, fmt.Sprintf("json option %q is incompatible with %s field %s", "string", contract.TypeString(field), field.GoName)))
		}
		if value.HasOption("omitempty") && !wasPointer {
			switch underlying.(type) {
			case *types.Struct, *types.Array:
				diagnostic := makeDiagnostic(item, field, cfg, "TAG202", namespaceName, pos, end, fmt.Sprintf("json option %q cannot omit zero-valued %s field %s; use omitzero or a pointer when appropriate", "omitempty", contract.TypeString(field), field.GoName))
				addTagRewrite(&diagnostic, field, namespaceName, rewriteOption(value, "omitempty", "omitzero"), rule.FixReview, "replace omitempty with omitzero (review wire behavior)")
				result = append(result, diagnostic)
			}
		}
	}
	if value.HasOption("inline") {
		switch underlying.(type) {
		case *types.Struct, *types.Map:
		default:
			result = append(result, makeDiagnostic(item, field, cfg, "TAG203", namespaceName, pos, end, fmt.Sprintf("%s inline option requires a struct, pointer-to-struct, or map; %s is %s", namespaceName, field.GoName, contract.TypeString(field))))
		}
	}
	if namespaceName == "xml" {
		representation := 0
		for _, option := range []string{"attr", "chardata", "cdata", "innerxml", "comment"} {
			if value.HasOption(option) {
				representation++
			}
		}
		if representation > 1 {
			result = append(result, makeDiagnostic(item, field, cfg, "TAG204", namespaceName, pos, end, fmt.Sprintf("xml field %s selects multiple incompatible representation options", field.GoName)))
		}
	}
	if namespaceName == "validate" && !value.Ignored {
		if containsRule(value.Raw, "email") && !basicKind(underlying, types.String) {
			result = append(result, makeDiagnostic(item, field, cfg, "TAG201", namespaceName, pos, end, fmt.Sprintf("validation rule %q requires a string, but %s is %s", "email", field.GoName, contract.TypeString(field))))
		}
		if containsRule(value.Raw, "dive") {
			switch underlying.(type) {
			case *types.Array, *types.Slice, *types.Map:
			default:
				result = append(result, makeDiagnostic(item, field, cfg, "TAG201", namespaceName, pos, end, fmt.Sprintf("validation rule %q requires a collection, but %s is %s", "dive", field.GoName, contract.TypeString(field))))
			}
		}
	}
	return result
}

func optionKnown(namespaceName, option string, known map[string]bool, cfg config.Config) bool {
	if known[option] {
		return true
	}
	if namespaceName == "json" && (cfg.JSON.Semantics == "v2" || cfg.JSON.Semantics == "both") {
		return option == "inline" || option == "unknown" || strings.HasPrefix(option, "case:") || strings.HasPrefix(option, "format:")
	}
	return false
}

func checkSurface(item *contract.StructContract, namespaceName string, surface *contract.NamespaceContract, cfg config.Config) []Diagnostic {
	groups := map[string][]*contract.EffectiveField{}
	caseGroups := map[string][]*contract.EffectiveField{}
	for _, field := range surface.Fields {
		if field.Ignored || field.Shadowed {
			continue
		}
		groups[field.Name] = append(groups[field.Name], field)
		if cfg.Namespace(namespaceName).CaseInsensitive || surface.Spec.CaseInsensitive {
			caseGroups[strings.ToLower(field.Name)] = append(caseGroups[strings.ToLower(field.Name)], field)
		}
	}
	var result []Diagnostic
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fields := groups[name]
		if len(fields) < 2 {
			continue
		}
		id := "TAG104"
		for _, field := range fields {
			if field.Depth > 0 {
				id = "TAG105"
			}
		}
		owner := fields[len(fields)-1]
		paths := make([]string, len(fields))
		related := make([]Location, 0, len(fields))
		for index, field := range fields {
			paths[index] = field.PathString(item.TypeName)
			related = append(related, Location{Pos: field.Field.Position, End: field.Field.End, Message: paths[index] + " also resolves here"})
		}
		diagnostic := makeDiagnostic(item, owner.Field, cfg, id, namespaceName, owner.Field.Position, owner.Field.End, fmt.Sprintf("duplicate %s name %q: %s all resolve to the same external field", namespaceName, name, strings.Join(paths, ", ")))
		diagnostic.FieldPath = owner.PathString(item.TypeName)
		diagnostic.Related = related
		if owner.Tag != nil && owner.Tag.Name != "" {
			replacement := ConvertName(owner.Field.GoName, "snake_case")
			if replacement != name {
				if len(owner.Tag.Options) > 0 {
					replacement += "," + strings.Join(owner.Tag.Options, ",")
				}
				addTagRewrite(&diagnostic, owner.Field, namespaceName, replacement, rule.FixReview, "rename one side of the collision (review API compatibility)")
			}
		}
		result = append(result, diagnostic)
	}
	for folded, fields := range caseGroups {
		if len(fields) < 2 {
			continue
		}
		exact := map[string]bool{}
		for _, field := range fields {
			exact[field.Name] = true
		}
		if len(exact) < 2 {
			continue
		}
		owner := fields[len(fields)-1]
		names := make([]string, 0, len(exact))
		for name := range exact {
			names = append(names, name)
		}
		sort.Strings(names)
		diagnostic := makeDiagnostic(item, owner.Field, cfg, "TAG106", namespaceName, owner.Field.Position, owner.Field.End, fmt.Sprintf("%s names %s differ only by case and may collide for case-insensitive consumers (%s)", namespaceName, strings.Join(names, ", "), folded))
		for _, field := range fields {
			diagnostic.Related = append(diagnostic.Related, Location{Pos: field.Field.Position, End: field.Field.End, Message: field.PathString(item.TypeName)})
		}
		result = append(result, diagnostic)
	}
	return result
}

func checkCrossNamespace(item *contract.StructContract, cfg config.Config) []Diagnostic {
	var result []Diagnostic
	for _, field := range item.Fields {
		var states []crossState
		for _, namespaceName := range cfg.CrossNamespace.Compare {
			if !cfg.Enabled(namespaceName) {
				continue
			}
			tagValue, tagged := field.Tags[namespaceName]
			state := crossState{namespace: namespaceName, name: field.GoName, tagged: tagged}
			if tagged {
				state.ignored = tagValue.Ignored
				if tagValue.Name != "" && !tagValue.Ignored {
					state.name = tagValue.Name
				}
				state.omit = tagValue.HasOption("omitempty") || tagValue.HasOption("omitzero")
			}
			states = append(states, state)
		}
		var exposed []crossState
		var explicitlyExposed []crossState
		ignoredCount := 0
		for _, state := range states {
			if state.ignored {
				ignoredCount++
			} else {
				exposed = append(exposed, state)
				if state.tagged {
					explicitlyExposed = append(explicitlyExposed, state)
				}
			}
		}
		if ignoredCount > 0 && len(exposed) > 0 {
			pos, end := fieldPosition(field)
			result = append(result, makeDiagnostic(item, field, cfg, "TAG302", "", pos, end, fmt.Sprintf("field %s is ignored in %d namespace(s) but exposed by %s", field.GoName, ignoredCount, joinNamespaces(exposed))))
		}
		omitCount := 0
		for _, state := range explicitlyExposed {
			if state.omit {
				omitCount++
			}
		}
		// Omission drift is meaningful only when at least two namespaces are
		// explicitly part of the field contract. Treating an absent tag as an
		// explicit non-omitting policy creates noise for ordinary Go structs.
		if len(explicitlyExposed) > 1 && omitCount > 0 && omitCount < len(explicitlyExposed) {
			pos, end := fieldPosition(field)
			result = append(result, makeDiagnostic(item, field, cfg, "TAG303", "", pos, end, fmt.Sprintf("field %s has inconsistent omission behavior across compared namespaces", field.GoName)))
		}
		var named []crossState
		for _, state := range exposed {
			if state.tagged {
				named = append(named, state)
			}
		}
		if len(named) > 1 {
			first := named[0].name
			drift := false
			identities := map[string]bool{}
			pairs := []string{}
			for _, state := range named {
				if state.name != first {
					drift = true
				}
				identities[normalized(state.name)] = true
				pairs = append(pairs, fmt.Sprintf("%s=%q", state.namespace, state.name))
			}
			if drift {
				pos, end := fieldPosition(field)
				diagnostic := makeDiagnostic(item, field, cfg, "TAG301", "", pos, end, "external naming drift: "+strings.Join(pairs, ", "))
				if value := field.Tags[named[1].namespace]; value != nil {
					replacement := named[0].name
					if len(value.Options) > 0 {
						replacement += "," + strings.Join(value.Options, ",")
					}
					addTagRewrite(&diagnostic, field, named[1].namespace, replacement, rule.FixReview, "unify external names (review compatibility)")
				}
				result = append(result, diagnostic)
			}
			if len(identities) > 1 && !sameWords(named) {
				pos, end := fieldPosition(field)
				result = append(result, makeDiagnostic(item, field, cfg, "TAG305", "", pos, end, "external names may describe different identities: "+strings.Join(pairs, ", ")))
			}
		}
		if validation, ok := field.Tags["validate"]; ok && containsRule(validation.Raw, "required") {
			for _, state := range states {
				if state.omit && !state.ignored {
					pos, end := fieldPosition(field)
					result = append(result, makeDiagnostic(item, field, cfg, "TAG204", state.namespace, pos, end, fmt.Sprintf("%q in %s conflicts with validation rule %q on %s", omissionName(field.Tags[state.namespace]), state.namespace, "required", field.GoName)))
					break
				}
			}
		}
	}
	return result
}

func checkSensitive(item *contract.StructContract, field *contract.FieldContract, cfg config.Config) []Diagnostic {
	if !field.Exported || !isSensitive(field.GoName, cfg.Security.SensitiveFields) {
		return nil
	}
	var exposures []string
	deceptive := false
	for _, namespaceName := range cfg.Security.OutputNamespaces {
		if !cfg.Enabled(namespaceName) {
			continue
		}
		value, tagged := field.Tags[namespaceName]
		if tagged && value.Ignored {
			continue
		}
		external := field.GoName
		if tagged && value.Name != "" {
			external = value.Name
		}
		exposures = append(exposures, fmt.Sprintf("%s=%q", namespaceName, external))
		if normalized(external) != normalized(field.GoName) && !containsSensitiveWord(external, cfg.Security.SensitiveFields) {
			deceptive = true
		}
	}
	if len(exposures) == 0 {
		return nil
	}
	pos, end := fieldPosition(field)
	diagnostic := makeDiagnostic(item, field, cfg, "TAG401", "", pos, end, fmt.Sprintf("sensitive field %s is publicly serialized (%s); explicitly ignore it in output namespaces or document a narrow suppression", field.GoName, strings.Join(exposures, ", ")))
	for _, namespaceName := range cfg.Security.OutputNamespaces {
		if value := field.Tags[namespaceName]; value != nil && !value.Ignored {
			addTagRewrite(&diagnostic, field, namespaceName, "-", rule.FixReview, "hide the sensitive field (review API behavior)")
			break
		}
	}
	result := []Diagnostic{diagnostic}
	if deceptive {
		result = append(result, makeDiagnostic(item, field, cfg, "TAG402", "", pos, end, fmt.Sprintf("sensitive field %s is exposed under a potentially misleading external name (%s)", field.GoName, strings.Join(exposures, ", "))))
	}
	if cfg.Security.RequireWriteOnly {
		result = append(result, makeDiagnostic(item, field, cfg, "TAG403", "", pos, end, fmt.Sprintf("write-only policy requires sensitive field %s to be ignored by all output namespaces", field.GoName)))
	}
	return result
}

func makeDiagnostic(item *contract.StructContract, field *contract.FieldContract, cfg config.Config, id, namespaceName string, pos token.Pos, rest ...any) Diagnostic {
	end := pos
	message := ""
	if len(rest) == 2 {
		end, _ = rest[0].(token.Pos)
		message, _ = rest[1].(string)
	} else if len(rest) == 1 {
		message, _ = rest[0].(string)
	}
	definition, _ := rule.Lookup(id)
	fieldPath := ""
	if field != nil {
		fieldPath = item.TypeName + "." + field.GoName
	}
	return Diagnostic{Rule: definition, Severity: cfg.RuleSeverity(id), Message: message, Namespace: namespaceName, Pos: pos, End: end, Package: item.Package, TypeName: item.TypeName, FieldPath: fieldPath, Contract: item, Field: field}
}

func fieldPosition(field *contract.FieldContract) (token.Pos, token.Pos) {
	if field.TagPosition != token.NoPos {
		return field.TagPosition, field.TagEnd
	}
	return field.Position, field.End
}

type crossState struct {
	namespace, name string
	ignored, omit   bool
	tagged          bool
}

func addTagRewrite(diagnostic *Diagnostic, field *contract.FieldContract, namespaceName, replacement string, safety rule.FixSafety, message string) {
	raw, ok := field.Parsed.ReplaceValue(namespaceName, replacement)
	if !ok || field.TagPosition == token.NoPos {
		return
	}
	literal := tag.Literal(raw, field.TagLiteral)
	diagnostic.Fixes = append(diagnostic.Fixes, fix.Suggestion{Message: message, Safety: safety, Edits: []fix.Edit{{Pos: field.TagPosition, End: field.TagEnd, NewText: []byte(literal)}}})
}

func applySuppressions(item *contract.StructContract, diagnostics []Diagnostic, cfg config.Config) []Diagnostic {
	result := diagnostics[:0]
	for _, diagnostic := range diagnostics {
		if !cfg.RuleEnabled(diagnostic.Rule.ID) || diagnostic.Severity == rule.SeverityOff {
			continue
		}
		suppression := item.Suppression
		if diagnostic.Field != nil && diagnostic.Field.Suppression != nil {
			suppression = diagnostic.Field.Suppression
		}
		if suppression != nil && suppression.IDs[diagnostic.Rule.ID] {
			suppression.Used[diagnostic.Rule.ID] = true
			continue
		}
		result = append(result, diagnostic)
	}
	return result
}
func unusedSuppressions(item *contract.StructContract, cfg config.Config) []Diagnostic {
	var result []Diagnostic
	suppressions := []*contract.Suppression{item.Suppression}
	for _, field := range item.Fields {
		suppressions = append(suppressions, field.Suppression)
	}
	seen := map[*contract.Suppression]bool{}
	for _, suppression := range suppressions {
		if suppression == nil || seen[suppression] {
			continue
		}
		seen[suppression] = true
		ids := []string{}
		for id := range suppression.IDs {
			if !suppression.Used[id] {
				ids = append(ids, id)
			}
		}
		sort.Strings(ids)
		for _, id := range ids {
			result = append(result, makeDiagnostic(item, nil, cfg, "TAG902", "", suppression.Position, suppression.End, fmt.Sprintf("suppression for %s is unused", id)))
		}
	}
	return result
}

func suppressOverlappingSafeFixes(diagnostics []Diagnostic) []Diagnostic {
	counts := map[[2]token.Pos]int{}
	for _, diagnostic := range diagnostics {
		for _, suggestion := range diagnostic.Fixes {
			if suggestion.Safety != rule.FixSafe {
				continue
			}
			for _, edit := range suggestion.Edits {
				counts[[2]token.Pos{edit.Pos, edit.End}]++
			}
		}
	}
	for index := range diagnostics {
		filtered := diagnostics[index].Fixes[:0]
		for _, suggestion := range diagnostics[index].Fixes {
			conflict := false
			if suggestion.Safety == rule.FixSafe {
				for _, edit := range suggestion.Edits {
					if counts[[2]token.Pos{edit.Pos, edit.End}] > 1 {
						conflict = true
					}
				}
			}
			if !conflict {
				filtered = append(filtered, suggestion)
			}
		}
		diagnostics[index].Fixes = filtered
	}
	return diagnostics
}

func builtinOptions(namespaceName string) map[string]bool {
	result := map[string]bool{}
	var values []string
	switch namespaceName {
	case "json":
		values = []string{"omitempty", "omitzero", "string"}
	case "yaml":
		values = []string{"omitempty", "flow", "inline"}
	case "xml":
		values = []string{"omitempty", "attr", "chardata", "cdata", "innerxml", "comment", "any"}
	}
	for _, value := range values {
		result[value] = true
	}
	return result
}
func suspiciousName(name, namespaceName string) bool {
	if name == "" || namespaceName == "xml" {
		return false
	}
	for _, character := range name {
		if unicode.IsControl(character) || unicode.IsSpace(character) {
			return true
		}
	}
	return false
}
func jsonStringCompatible(value types.Type) bool {
	basic, ok := value.(*types.Basic)
	if !ok {
		return false
	}
	return basic.Info()&(types.IsBoolean|types.IsInteger|types.IsUnsigned|types.IsFloat|types.IsString) != 0
}
func basicKind(value types.Type, kind types.BasicKind) bool {
	basic, ok := value.(*types.Basic)
	return ok && basic.Kind() == kind
}
func containsRule(value, expected string) bool {
	for _, candidate := range strings.Split(value, ",") {
		name, _, _ := strings.Cut(candidate, "=")
		if name == expected {
			return true
		}
	}
	return false
}

func omissionName(value *contract.TagContract) string {
	if value != nil && value.HasOption("omitzero") {
		return "omitzero"
	}
	return "omitempty"
}
func rewriteOption(value *contract.TagContract, from, to string) string {
	options := append([]string(nil), value.Options...)
	for index, option := range options {
		if option == from {
			options[index] = to
			break
		}
	}
	result := value.Name
	if len(options) > 0 {
		result += "," + strings.Join(options, ",")
	}
	return result
}
func isSensitive(name string, patterns []string) bool {
	normalizedName := normalized(name)
	for _, pattern := range patterns {
		candidate := normalized(pattern)
		if normalizedName == candidate || strings.HasPrefix(normalizedName, candidate+"_") || strings.HasSuffix(normalizedName, "_"+candidate) {
			return true
		}
	}
	return false
}
func containsSensitiveWord(name string, patterns []string) bool {
	normalizedName := normalized(name)
	for _, pattern := range patterns {
		if strings.Contains(normalizedName, normalized(pattern)) {
			return true
		}
	}
	return false
}
func joinNamespaces(states []crossState) string {
	values := make([]string, len(states))
	for index, state := range states {
		values[index] = state.namespace
	}
	return strings.Join(values, ", ")
}
func sameWords(states []crossState) bool {
	if len(states) < 2 {
		return true
	}
	first := strings.Join(lowerWords(states[0].name), "|")
	for _, state := range states[1:] {
		if strings.Join(lowerWords(state.name), "|") != first {
			return false
		}
	}
	return true
}

// BenchmarkEvaluateContract is intentionally exercised through package tests.
