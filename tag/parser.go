// Package tag parses Go struct tags once into a lossless normalized form.
package tag

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ProblemKind classifies parser-level failures.
type ProblemKind int

const (
	ProblemMalformed ProblemKind = iota + 1
	ProblemDuplicateNamespace
)

// Problem is a recoverable parser diagnostic.
type Problem struct {
	Kind      ProblemKind
	Namespace string
	Message   string
}

// Value is one namespace value with lossless raw offsets.
type Value struct {
	Namespace string
	Raw       string
	Name      string
	Options   []string
	Ignored   bool
	start     int
	end       int
}

// HasOption reports whether an option appears at least once.
func (v Value) HasOption(option string) bool {
	for _, candidate := range v.Options {
		if candidate == option {
			return true
		}
	}
	return false
}

// Parsed is a complete normalized struct tag.
type Parsed struct {
	Raw      string
	Values   []Value
	ByName   map[string][]int
	Problems []Problem
}

// Parse parses reflect.StructTag syntax without panicking on malformed input.
func Parse(raw string) Parsed {
	parsed := Parsed{Raw: raw, ByName: make(map[string][]int)}
	for index := 0; index < len(raw); {
		for index < len(raw) && raw[index] == ' ' {
			index++
		}
		if index == len(raw) {
			break
		}
		keyStart := index
		for index < len(raw) && raw[index] > ' ' && raw[index] != ':' && raw[index] != '"' && raw[index] != 0x7f {
			index++
		}
		if keyStart == index || index+1 >= len(raw) || raw[index] != ':' || raw[index+1] != '"' {
			parsed.Problems = append(parsed.Problems, Problem{Kind: ProblemMalformed, Message: fmt.Sprintf("malformed struct tag near %q", raw[keyStart:])})
			break
		}
		key := raw[keyStart:index]
		index++
		valueStart := index
		index++
		closed := false
		for index < len(raw) {
			if raw[index] == '\\' {
				index += 2
				continue
			}
			if index < len(raw) && raw[index] == '"' {
				index++
				closed = true
				break
			}
			index++
		}
		if !closed || index > len(raw) {
			parsed.Problems = append(parsed.Problems, Problem{Kind: ProblemMalformed, Namespace: key, Message: fmt.Sprintf("unterminated value for tag %q", key)})
			break
		}
		quoted := raw[valueStart:index]
		value, err := strconv.Unquote(quoted)
		if err != nil {
			parsed.Problems = append(parsed.Problems, Problem{Kind: ProblemMalformed, Namespace: key, Message: fmt.Sprintf("invalid value for tag %q: %v", key, err)})
		} else {
			parts := strings.Split(value, ",")
			entry := Value{Namespace: key, Raw: value, Name: parts[0], Ignored: parts[0] == "-", start: valueStart, end: index}
			if len(parts) > 1 {
				entry.Options = append([]string(nil), parts[1:]...)
			}
			valueIndex := len(parsed.Values)
			parsed.Values = append(parsed.Values, entry)
			if len(parsed.ByName[key]) > 0 {
				parsed.Problems = append(parsed.Problems, Problem{Kind: ProblemDuplicateNamespace, Namespace: key, Message: fmt.Sprintf("duplicate struct tag namespace %q", key)})
			}
			parsed.ByName[key] = append(parsed.ByName[key], valueIndex)
		}
		if index < len(raw) && raw[index] != ' ' {
			parsed.Problems = append(parsed.Problems, Problem{Kind: ProblemMalformed, Namespace: key, Message: "struct tag values must be separated by spaces"})
			break
		}
	}
	return parsed
}

// First returns the first value for namespace.
func (p Parsed) First(namespace string) (Value, bool) {
	indices := p.ByName[namespace]
	if len(indices) == 0 {
		return Value{}, false
	}
	return p.Values[indices[0]], true
}

// Namespaces returns declared namespaces in sorted order.
func (p Parsed) Namespaces() []string {
	result := make([]string, 0, len(p.ByName))
	for name := range p.ByName {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

// ReplaceValue rewrites one namespace while preserving all unrelated values.
func (p Parsed) ReplaceValue(namespace, value string) (string, bool) {
	indices := p.ByName[namespace]
	if len(indices) != 1 {
		return p.Raw, false
	}
	entry := p.Values[indices[0]]
	return p.Raw[:entry.start] + strconv.Quote(value) + p.Raw[entry.end:], true
}

// Literal quotes raw using the original literal style where possible.
func Literal(raw, original string) string {
	if strings.HasPrefix(original, "`") && strings.HasSuffix(original, "`") && !strings.ContainsRune(raw, '`') {
		return "`" + raw + "`"
	}
	return strconv.Quote(raw)
}

// RemoveDuplicateOptions returns a value with later duplicates removed.
func RemoveDuplicateOptions(value Value) (string, bool) {
	seen := make(map[string]bool)
	options := make([]string, 0, len(value.Options))
	changed := false
	for _, option := range value.Options {
		if seen[option] {
			changed = true
			continue
		}
		seen[option] = true
		options = append(options, option)
	}
	result := value.Name
	if len(options) > 0 {
		result += "," + strings.Join(options, ",")
	}
	return result, changed
}
