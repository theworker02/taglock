// Package schema exports snapshot contracts as JSON Schema or OpenAPI components.
package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/magnexis/taglock/semantics"
	"github.com/magnexis/taglock/snapshot"
)

type Options struct {
	Format  string
	Profile string
}

func Generate(document snapshot.Snapshot, options Options) (map[string]any, error) {
	if options.Format == "" {
		options.Format = "json-schema"
	}
	if options.Format != "json-schema" && options.Format != "openapi" {
		return nil, fmt.Errorf("unsupported schema format %q", options.Format)
	}
	names := contractNames(document)
	definitions := map[string]any{}
	refPrefix := "#/$defs/"
	if options.Format == "openapi" {
		refPrefix = "#/components/schemas/"
	}
	for _, pkg := range document.Packages {
		for _, item := range pkg.Contracts {
			surface, ok := chooseSurface(item, options.Profile)
			if !ok {
				continue
			}
			name := names[pkg.ImportPath+"\x00"+item.TypeName]
			definitions[name] = contractSchema(surface, names, refPrefix)
		}
	}
	if options.Format == "openapi" {
		return map[string]any{"components": map[string]any{"schemas": definitions}, "x-taglock-profiles": document.SemanticProfiles}, nil
	}
	return map[string]any{"$schema": "https://json-schema.org/draft/2020-12/schema", "$id": "taglock://" + document.ModulePath, "x-taglock-format-version": 1, "x-taglock-profiles": document.SemanticProfiles, "$defs": definitions}, nil
}

func Write(writer io.Writer, value map[string]any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}
func Canonical(value map[string]any) ([]byte, error) { return json.Marshal(value) }
func EqualJSON(first, second []byte) (bool, error) {
	var left, right any
	if err := json.Unmarshal(first, &left); err != nil {
		return false, err
	}
	if err := json.Unmarshal(second, &right); err != nil {
		return false, err
	}
	a, _ := json.Marshal(left)
	b, _ := json.Marshal(right)
	return bytes.Equal(a, b), nil
}

func contractNames(document snapshot.Snapshot) map[string]string {
	counts := map[string]int{}
	for _, pkg := range document.Packages {
		for _, item := range pkg.Contracts {
			counts[item.TypeName]++
		}
	}
	result := map[string]string{}
	for _, pkg := range document.Packages {
		for _, item := range pkg.Contracts {
			name := item.TypeName
			if counts[name] > 1 {
				name = path.Base(pkg.ImportPath) + "_" + name
			}
			result[pkg.ImportPath+"\x00"+item.TypeName] = name
			result[item.TypeName] = name
			result[pkg.ImportPath+"."+item.TypeName] = name
		}
	}
	return result
}
func chooseSurface(item snapshot.ContractSnapshot, profile string) (snapshot.SurfaceSnapshot, bool) {
	if profile != "" {
		for _, surface := range item.Profiles {
			if surface.Profile == profile {
				return surface, true
			}
		}
	}
	for _, surface := range item.Profiles {
		if surface.Profile == "json/v1" {
			return surface, true
		}
	}
	if len(item.Profiles) > 0 {
		return item.Profiles[0], true
	}
	return snapshot.SurfaceSnapshot{}, false
}

func contractSchema(surface snapshot.SurfaceSnapshot, names map[string]string, refPrefix string) map[string]any {
	if surface.Certainty == semantics.CertaintyOpaque {
		return map[string]any{"x-taglock-certainty": "opaque", "x-taglock-reason": surface.CertaintyReason}
	}
	properties := map[string]any{}
	var required []string
	for _, field := range surface.Fields {
		if field.Ignored {
			continue
		}
		value := typeSchema(field.GoType, field.WireType, names, refPrefix)
		for key, annotation := range field.Schema {
			switch key {
			case "format":
				value["format"] = annotation
			case "minimum", "maximum":
				if number, err := strconv.ParseFloat(annotation, 64); err == nil {
					value[key] = number
				}
			}
		}
		if field.Nullable {
			value = map[string]any{"anyOf": []any{value, map[string]any{"type": "null"}}}
		}
		properties[field.ExternalName] = value
		if field.Required {
			required = append(required, field.ExternalName)
		}
	}
	sort.Strings(required)
	result := map[string]any{"type": "object", "properties": properties, "additionalProperties": false, "x-taglock-certainty": surface.Certainty, "x-taglock-profile": surface.Profile}
	if len(required) > 0 {
		result["required"] = required
	}
	return result
}

func typeSchema(value, wire string, names map[string]string, refPrefix string) map[string]any {
	value = strings.TrimSpace(value)
	for strings.HasPrefix(value, "*") {
		value = strings.TrimPrefix(value, "*")
	}
	if strings.HasPrefix(value, "[]") {
		element := strings.TrimPrefix(value, "[]")
		if element == "byte" || element == "uint8" {
			return map[string]any{"type": "string", "contentEncoding": "base64"}
		}
		return map[string]any{"type": "array", "items": typeSchema(element, "", names, refPrefix)}
	}
	if strings.HasPrefix(value, "[") {
		if close := strings.Index(value, "]"); close > 1 {
			length, _ := strconv.Atoi(value[1:close])
			return map[string]any{"type": "array", "items": typeSchema(value[close+1:], "", names, refPrefix), "minItems": length, "maxItems": length}
		}
	}
	if strings.HasPrefix(value, "map[") {
		if close := strings.Index(value, "]"); close > 0 {
			return map[string]any{"type": "object", "additionalProperties": typeSchema(value[close+1:], "", names, refPrefix)}
		}
	}
	switch value {
	case "string":
		return map[string]any{"type": "string"}
	case "bool":
		return map[string]any{"type": "boolean"}
	case "float32", "float64":
		return map[string]any{"type": "number"}
	case "time.Time":
		return map[string]any{"type": "string", "format": "date-time"}
	case "any", "interface{}":
		return map[string]any{}
	}
	if strings.HasPrefix(value, "int") || strings.HasPrefix(value, "uint") || value == "byte" || value == "rune" {
		return map[string]any{"type": "integer"}
	}
	if name, ok := names[value]; ok {
		return map[string]any{"$ref": refPrefix + name}
	}
	if dot := strings.LastIndex(value, "."); dot >= 0 {
		if name, ok := names[value[dot+1:]]; ok {
			return map[string]any{"$ref": refPrefix + name}
		}
	}
	switch {
	case wire == "string":
		return map[string]any{"type": "string"}
	case wire == "integer":
		return map[string]any{"type": "integer"}
	case wire == "number":
		return map[string]any{"type": "number"}
	case wire == "boolean":
		return map[string]any{"type": "boolean"}
	case wire == "date-time":
		return map[string]any{"type": "string", "format": "date-time"}
	case wire == "bytes":
		return map[string]any{"type": "string", "contentEncoding": "base64"}
	case wire == "object":
		return map[string]any{"type": "object"}
	case wire == "any":
		return map[string]any{}
	case strings.HasPrefix(wire, "array:"):
		return map[string]any{"type": "array", "items": typeSchema("", strings.TrimPrefix(wire, "array:"), names, refPrefix)}
	case strings.HasPrefix(wire, "object:"):
		return map[string]any{"type": "object", "additionalProperties": typeSchema("", strings.TrimPrefix(wire, "object:"), names, refPrefix)}
	}
	return map[string]any{"x-taglock-go-type": value, "x-taglock-certainty": "partial"}
}
