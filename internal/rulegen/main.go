package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/magnexis/taglock/evolution"
	"github.com/magnexis/taglock/migration"
	"github.com/magnexis/taglock/rule"
)

func main() {
	root, err := findRoot()
	if err != nil {
		panic(err)
	}
	var builder strings.Builder
	builder.WriteString("# TagLock rule reference\n\nThis file is generated from the executable rule registries.\n\n## Source contract rules\n\n")
	for _, definition := range rule.All() {
		fmt.Fprintf(&builder, "### %s — %s\n\n%s\n\n- Default severity: `%s`\n- Default enabled: `%t`\n- Fix safety: `%s`\n\n", definition.ID, definition.Name, definition.Explanation, definition.DefaultSeverity, definition.DefaultEnabled, definition.FixSafety)
	}
	builder.WriteString("## Evolution rules\n\n")
	for _, definition := range evolution.RuleDefinitions() {
		fmt.Fprintf(&builder, "### %s — %s\n\n%s\n\n", definition.ID, definition.Name, definition.Summary)
	}
	builder.WriteString("## JSON v1-to-v2 migration rules\n\n")
	migrationRules := migration.RuleDefinitions()
	ids := make([]string, 0, len(migrationRules))
	for id := range migrationRules {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		fmt.Fprintf(&builder, "### %s — %s\n\n", id, migrationRules[id])
	}
	directory := filepath.Join(root, "docs")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "RULES.md"), []byte(builder.String()), 0o644); err != nil {
		panic(err)
	}
}
func findRoot() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			return directory, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("go.mod not found")
		}
		directory = parent
	}
}
