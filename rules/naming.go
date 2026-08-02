package rules

import (
	"strings"
	"unicode"
)

// ConvertName converts Go/camel/snake/kebab names to a configured style.
func ConvertName(value, style string) string {
	words := SplitWords(value)
	if len(words) == 0 {
		return value
	}
	for index := range words {
		words[index] = strings.ToLower(words[index])
	}
	switch style {
	case "snake_case":
		return strings.Join(words, "_")
	case "kebab-case":
		return strings.Join(words, "-")
	case "camelCase":
		for index := 1; index < len(words); index++ {
			words[index] = upperFirst(words[index])
		}
		return strings.Join(words, "")
	case "PascalCase":
		for index := range words {
			words[index] = upperFirst(words[index])
		}
		return strings.Join(words, "")
	default:
		return value
	}
}

// SplitWords normalizes common identifier casing without regular expressions.
func SplitWords(value string) []string {
	value = strings.Trim(value, "_- ")
	if value == "" {
		return nil
	}
	runes := []rune(value)
	start := 0
	var words []string
	flush := func(end int) {
		if end > start {
			words = append(words, string(runes[start:end]))
		}
	}
	for index, current := range runes {
		if current == '_' || current == '-' || unicode.IsSpace(current) {
			flush(index)
			start = index + 1
			continue
		}
		if index > start && unicode.IsUpper(current) {
			previous := runes[index-1]
			nextLower := index+1 < len(runes) && unicode.IsLower(runes[index+1])
			if unicode.IsLower(previous) || unicode.IsDigit(previous) || (unicode.IsUpper(previous) && nextLower) {
				flush(index)
				start = index
			}
		}
	}
	flush(len(runes))
	return words
}

func upperFirst(value string) string {
	runes := []rune(value)
	if len(runes) > 0 {
		runes[0] = unicode.ToUpper(runes[0])
	}
	return string(runes)
}
func normalized(value string) string { return strings.Join(lowerWords(value), "_") }
func lowerWords(value string) []string {
	words := SplitWords(value)
	for index := range words {
		words[index] = strings.ToLower(words[index])
	}
	return words
}
