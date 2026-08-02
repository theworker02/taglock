// Package cli implements TagLock's standalone command surface.
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/magnexis/taglock/baseline"
	"github.com/magnexis/taglock/config"
	"github.com/magnexis/taglock/engine"
	"github.com/magnexis/taglock/evolution"
	"github.com/magnexis/taglock/gitdiff"
	"github.com/magnexis/taglock/manifest"
	"github.com/magnexis/taglock/migration"
	"github.com/magnexis/taglock/output"
	"github.com/magnexis/taglock/rule"
	"github.com/magnexis/taglock/rules"
	"github.com/magnexis/taglock/schema"
	"github.com/magnexis/taglock/semantics"
	"github.com/magnexis/taglock/snapshot"
	"github.com/magnexis/taglock/verify"
)

const (
	ExitOK         = 0
	ExitViolations = 1
	ExitUsage      = 2
	ExitAnalysis   = 3
)

// Run executes one command and returns a stable process exit code.
func Run(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 {
		return runCheck(nil, stdout, stderr)
	}
	command := arguments[0]
	rest := arguments[1:]
	switch command {
	case "check":
		return runCheck(rest, stdout, stderr)
	case "fix":
		return runFix(rest, stdout, stderr)
	case "rules":
		return runRules(rest, stdout, stderr)
	case "explain":
		return runExplain(rest, stdout, stderr)
	case "init":
		return runInit(rest, stdout, stderr)
	case "config":
		return runConfig(rest, stdout, stderr)
	case "baseline":
		return runBaseline(rest, stdout, stderr)
	case "snapshot":
		return runSnapshot(rest, stdout, stderr)
	case "compare":
		return runCompare(rest, stdout, stderr)
	case "migrate":
		return runMigrate(rest, stdout, stderr)
	case "manifest":
		return runManifest(rest, stdout, stderr)
	case "schema":
		return runSchema(rest, stdout, stderr)
	case "verify":
		return runVerify(rest, stdout, stderr)
	case "changes":
		return runChanges(rest, stdout, stderr)
	case "help", "-h", "--help":
		writeUsage(stdout)
		return ExitOK
	default:
		if strings.HasPrefix(command, "-") || strings.Contains(command, "/") || command == "." || command == "..." {
			return runCheck(arguments, stdout, stderr)
		}
		fmt.Fprintf(stderr, "taglock: unknown command %q\n", command)
		writeUsage(stderr)
		return ExitUsage
	}
}

func runCheck(arguments []string, stdout, stderr io.Writer) int {
	set := newFlagSet("check", stderr)
	format := set.String("format", "text", "output format: text, json, or sarif")
	failOn := set.String("fail-on", "warning", "minimum severity that fails")
	configPath := set.String("config", "", "configuration path")
	baselinePath := set.String("baseline", "", "baseline file")
	jsonSemantics := set.String("json-semantics", "", "JSON semantics: auto, v1, v2, or both")
	if err := set.Parse(arguments); err != nil {
		return ExitUsage
	}
	threshold, err := rule.ParseSeverity(*failOn)
	if err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitUsage
	}
	if *format != "text" && *format != "json" && *format != "sarif" {
		fmt.Fprintf(stderr, "taglock: invalid format %q\n", *format)
		return ExitUsage
	}
	cfg, code := loadConfig(*configPath, stderr)
	if code != ExitOK {
		return code
	}
	if *jsonSemantics != "" {
		cfg.JSON.Semantics = *jsonSemantics
		if err := cfg.Validate(); err != nil {
			fmt.Fprintln(stderr, "taglock:", err)
			return ExitUsage
		}
		registry := semantics.NewRegistry()
		if (*jsonSemantics == "v2" || *jsonSemantics == "both") && !registry.JSONV2Active {
			fmt.Fprintf(stderr, "taglock: warning: json/v2 is installed in %s but unavailable without GOEXPERIMENT=jsonv2; static migration modeling remains available\n", registry.Toolchain)
		}
	}
	result, err := engine.Analyze(context.Background(), set.Args(), cfg)
	if err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitAnalysis
	}
	diagnostics := result.Diagnostics
	if *baselinePath != "" {
		file, err := os.Open(*baselinePath)
		if err != nil {
			fmt.Fprintln(stderr, "taglock: open baseline:", err)
			return ExitUsage
		}
		stored, err := baseline.Read(file)
		file.Close()
		if err != nil {
			fmt.Fprintln(stderr, "taglock:", err)
			return ExitUsage
		}
		var stale []baseline.Record
		diagnostics, stale = baseline.Filter(diagnostics, stored)
		if len(stale) > 0 {
			fmt.Fprintf(stderr, "taglock: baseline contains %d stale entr", len(stale))
			if len(stale) == 1 {
				fmt.Fprintln(stderr, "y")
			} else {
				fmt.Fprintln(stderr, "ies")
			}
		}
	}
	switch *format {
	case "text":
		err = output.Text(stdout, result.FileSet, diagnostics)
	case "json":
		err = output.JSON(stdout, result.FileSet, diagnostics)
	case "sarif":
		err = output.SARIF(stdout, result.FileSet, diagnostics)
	}
	if err != nil {
		fmt.Fprintln(stderr, "taglock: write output:", err)
		return ExitAnalysis
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity >= threshold && threshold != rule.SeverityOff {
			return ExitViolations
		}
	}
	return ExitOK
}

func runFix(arguments []string, stdout, stderr io.Writer) int {
	set := newFlagSet("fix", stderr)
	safe := set.Bool("safe", true, "apply safe fixes (default)")
	review := set.Bool("review", false, "also apply review-required fixes")
	ruleID := set.String("rule", "", "limit fixes to one rule ID")
	diffOnly := set.Bool("diff", false, "print a diff without writing files")
	configPath := set.String("config", "", "configuration path")
	if err := set.Parse(arguments); err != nil {
		return ExitUsage
	}
	if *ruleID != "" && !rule.Known(strings.ToUpper(*ruleID)) {
		fmt.Fprintf(stderr, "taglock: unknown rule %q\n", *ruleID)
		return ExitUsage
	}
	cfg, code := loadConfig(*configPath, stderr)
	if code != ExitOK {
		return code
	}
	result, err := engine.Analyze(context.Background(), set.Args(), cfg)
	if err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitAnalysis
	}
	edits, err := collectEdits(result, *safe, *review, strings.ToUpper(*ruleID))
	if err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitAnalysis
	}
	if len(edits) == 0 {
		fmt.Fprintln(stdout, "TagLock: no applicable fixes")
		return ExitOK
	}
	files := make([]string, 0, len(edits))
	for filename := range edits {
		files = append(files, filename)
	}
	sort.Strings(files)
	for _, filename := range files {
		original, err := os.ReadFile(filename)
		if err != nil {
			fmt.Fprintln(stderr, "taglock: read source:", err)
			return ExitAnalysis
		}
		updated, err := applyEdits(original, edits[filename])
		if err != nil {
			fmt.Fprintln(stderr, "taglock:", err)
			return ExitAnalysis
		}
		if *diffOnly {
			writeSimpleDiff(stdout, filename, original, updated)
			continue
		}
		if err := os.WriteFile(filename, updated, 0o644); err != nil {
			fmt.Fprintln(stderr, "taglock: write source:", err)
			return ExitAnalysis
		}
		fmt.Fprintln(stdout, "fixed", filename)
	}
	return ExitOK
}

type sourceEdit struct {
	start, end int
	text       []byte
	ruleID     string
}

func collectEdits(result engine.Result, safe, review bool, onlyRule string) (map[string][]sourceEdit, error) {
	collected := map[string][]sourceEdit{}
	seen := map[string]bool{}
	for _, diagnostic := range result.Diagnostics {
		if onlyRule != "" && diagnostic.Rule.ID != onlyRule {
			continue
		}
		for _, suggestion := range diagnostic.Fixes {
			if suggestion.Safety == rule.FixSafe && !safe {
				continue
			}
			if suggestion.Safety == rule.FixReview && !review {
				continue
			}
			for _, edit := range suggestion.Edits {
				start, end := result.FileSet.Position(edit.Pos), result.FileSet.Position(edit.End)
				if start.Filename == "" || start.Filename != end.Filename {
					return nil, fmt.Errorf("invalid cross-file edit for %s", diagnostic.Rule.ID)
				}
				key := fmt.Sprintf("%s:%d:%d:%s", start.Filename, start.Offset, end.Offset, edit.NewText)
				if seen[key] {
					continue
				}
				seen[key] = true
				collected[start.Filename] = append(collected[start.Filename], sourceEdit{start: start.Offset, end: end.Offset, text: edit.NewText, ruleID: diagnostic.Rule.ID})
			}
		}
	}
	for filename := range collected {
		sort.Slice(collected[filename], func(i, j int) bool { return collected[filename][i].start > collected[filename][j].start })
		for index := 1; index < len(collected[filename]); index++ {
			if collected[filename][index-1].start < collected[filename][index].end {
				return nil, fmt.Errorf("overlapping fixes in %s", filename)
			}
		}
	}
	return collected, nil
}
func applyEdits(source []byte, edits []sourceEdit) ([]byte, error) {
	result := append([]byte(nil), source...)
	for _, edit := range edits {
		if edit.start < 0 || edit.end < edit.start || edit.end > len(result) {
			return nil, fmt.Errorf("fix range %d:%d is outside source", edit.start, edit.end)
		}
		result = append(result[:edit.start], append(append([]byte(nil), edit.text...), result[edit.end:]...)...)
	}
	return result, nil
}
func writeSimpleDiff(writer io.Writer, filename string, before, after []byte) {
	fmt.Fprintf(writer, "--- %s\n+++ %s\n", filename, filename)
	fmt.Fprintln(writer, "@@ proposed TagLock edits @@")
	for _, line := range strings.Split(strings.TrimSuffix(string(before), "\n"), "\n") {
		fmt.Fprintln(writer, "-"+line)
	}
	for _, line := range strings.Split(strings.TrimSuffix(string(after), "\n"), "\n") {
		fmt.Fprintln(writer, "+"+line)
	}
}

func runRules(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) != 0 {
		fmt.Fprintln(stderr, "taglock rules takes no arguments")
		return ExitUsage
	}
	for _, definition := range rule.All() {
		fmt.Fprintf(stdout, "%s  %-7s %-11s %s\n", definition.ID, definition.DefaultSeverity, definition.FixSafety, definition.Name)
	}
	return ExitOK
}
func runExplain(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) != 1 {
		fmt.Fprintln(stderr, "usage: taglock explain <rule-id>")
		return ExitUsage
	}
	definition, ok := rule.Lookup(arguments[0])
	if !ok {
		fmt.Fprintf(stderr, "taglock: unknown rule %q\n", arguments[0])
		return ExitUsage
	}
	fmt.Fprintf(stdout, "%s — %s\n\n%s\n\n%s\n\nIncorrect:\n  %s\n\nCorrected:\n  %s\n\nDefault severity: %s\nFix availability: %s\n", definition.ID, definition.Name, definition.Summary, definition.Explanation, definition.Incorrect, definition.Correct, definition.DefaultSeverity, definition.FixSafety)
	if len(definition.Remediation) > 0 {
		fmt.Fprintln(stdout, "\nPossible resolutions:")
		for _, item := range definition.Remediation {
			fmt.Fprintln(stdout, "  -", item)
		}
	}
	return ExitOK
}

func runInit(arguments []string, stdout, stderr io.Writer) int {
	set := newFlagSet("init", stderr)
	filename := set.String("file", ".taglock.yaml", "configuration filename")
	if err := set.Parse(arguments); err != nil || len(set.Args()) != 0 {
		return ExitUsage
	}
	for _, name := range config.Filenames {
		if _, err := os.Stat(name); err == nil {
			fmt.Fprintf(stderr, "taglock: configuration already exists at %s; refusing to overwrite\n", name)
			return ExitUsage
		} else if !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(stderr, "taglock:", err)
			return ExitUsage
		}
	}
	data, err := config.Marshal(config.Default())
	if err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitAnalysis
	}
	header := []byte("# TagLock policy. Run `taglock config validate` after editing.\n")
	if err := os.WriteFile(*filename, append(header, data...), 0o644); err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitAnalysis
	}
	fmt.Fprintln(stdout, "created", *filename)
	return ExitOK
}

func runConfig(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 {
		fmt.Fprintln(stderr, "usage: taglock config <validate|print> [--config path]")
		return ExitUsage
	}
	command := arguments[0]
	set := newFlagSet("config "+command, stderr)
	pathValue := set.String("config", "", "configuration path")
	if err := set.Parse(arguments[1:]); err != nil || len(set.Args()) != 0 {
		return ExitUsage
	}
	cfg, code := loadConfig(*pathValue, stderr)
	if code != ExitOK {
		return code
	}
	switch command {
	case "validate":
		fmt.Fprintln(stdout, "TagLock configuration is valid")
		return ExitOK
	case "print":
		data, err := config.Marshal(cfg)
		if err != nil {
			fmt.Fprintln(stderr, "taglock:", err)
			return ExitAnalysis
		}
		_, err = stdout.Write(data)
		if err != nil {
			return ExitAnalysis
		}
		return ExitOK
	default:
		fmt.Fprintf(stderr, "taglock: unknown config command %q\n", command)
		return ExitUsage
	}
}

func runBaseline(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 {
		fmt.Fprintln(stderr, "usage: taglock baseline <create|update> [packages]")
		return ExitUsage
	}
	command := arguments[0]
	if command != "create" && command != "update" {
		fmt.Fprintf(stderr, "taglock: unknown baseline command %q\n", command)
		return ExitUsage
	}
	set := newFlagSet("baseline "+command, stderr)
	filename := set.String("output", ".taglock-baseline.json", "baseline path")
	configPath := set.String("config", "", "configuration path")
	if err := set.Parse(arguments[1:]); err != nil {
		return ExitUsage
	}
	if command == "create" {
		if _, err := os.Stat(*filename); err == nil {
			fmt.Fprintf(stderr, "taglock: baseline %s already exists; use baseline update\n", *filename)
			return ExitUsage
		} else if !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(stderr, "taglock:", err)
			return ExitUsage
		}
	}
	cfg, code := loadConfig(*configPath, stderr)
	if code != ExitOK {
		return code
	}
	result, err := engine.Analyze(context.Background(), set.Args(), cfg)
	if err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitAnalysis
	}
	file, err := os.Create(*filename)
	if err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitAnalysis
	}
	err = baseline.Write(file, baseline.Create(result.Diagnostics))
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitAnalysis
	}
	fmt.Fprintf(stdout, "wrote %d baseline findings to %s\n", len(baseline.Create(result.Diagnostics).Findings), *filename)
	return ExitOK
}

func runSnapshot(arguments []string, stdout, stderr io.Writer) int {
	set := newFlagSet("snapshot", stderr)
	outputPath := set.String("output", "", "snapshot path")
	semanticsValue := set.String("semantics", "", "auto, v1, v2, or both")
	reproducible := set.Bool("reproducible", true, "omit nonsemantic metadata")
	configPath := set.String("config", "", "configuration path")
	buildTags := set.String("build-tags", "", "comma-separated Go build tags")
	if err := set.Parse(arguments); err != nil {
		return ExitUsage
	}
	cfg, code := loadConfig(*configPath, stderr)
	if code != ExitOK {
		return code
	}
	selected := *semanticsValue
	if selected == "" {
		selected = cfg.JSON.Semantics
	}
	result, err := engine.AnalyzeWithOptions(context.Background(), set.Args(), cfg, engine.Options{BuildTags: splitComma(*buildTags)})
	if err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitAnalysis
	}
	document, err := snapshot.Build(result, cfg, snapshot.BuildOptions{Semantics: selected, Reproducible: *reproducible})
	if err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitUsage
	}
	destination := *outputPath
	if destination == "" {
		destination = cfg.Snapshot.Output
	}
	if destination == "" {
		destination = ".taglock/contracts.json"
	}
	if err := writeFileOrOutput(destination, stdout, func(writer io.Writer) error { return snapshot.Write(writer, document) }); err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitAnalysis
	}
	if destination != "-" {
		fmt.Fprintf(stdout, "wrote %d package contract snapshot to %s\n", len(document.Packages), destination)
	}
	return ExitOK
}

func runCompare(arguments []string, stdout, stderr io.Writer) int {
	set := newFlagSet("compare", stderr)
	baseValue := set.String("base", "", "base Git revision or snapshot")
	headValue := set.String("head", "", "head Git revision or snapshot")
	formatValue := set.String("format", "text", "text, json, markdown, or sarif")
	outputPath := set.String("output", "-", "output path or -")
	semanticsValue := set.String("semantics", "", "auto, v1, v2, or both")
	configPath := set.String("config", "", "configuration path")
	buildTags := set.String("build-tags", "", "comma-separated Go build tags")
	if err := set.Parse(arguments); err != nil {
		return ExitUsage
	}
	positional := set.Args()
	if *baseValue == "" && *headValue == "" {
		if len(positional) != 2 {
			fmt.Fprintln(stderr, "usage: taglock compare <base-snapshot> <head-snapshot> or --base REV --head REV")
			return ExitUsage
		}
		*baseValue, *headValue = positional[0], positional[1]
		positional = nil
	} else if *baseValue == "" || *headValue == "" {
		fmt.Fprintln(stderr, "taglock compare requires both --base and --head")
		return ExitUsage
	}
	cfg, code := loadConfig(*configPath, stderr)
	if code != ExitOK {
		return code
	}
	selected := *semanticsValue
	if selected == "" {
		selected = cfg.JSON.Semantics
	}
	var baseSnapshot, headSnapshot snapshot.Snapshot
	var err error
	if gitdiff.IsSnapshot(*baseValue) && gitdiff.IsSnapshot(*headValue) {
		baseSnapshot, err = readSnapshot(*baseValue)
		if err == nil {
			headSnapshot, err = readSnapshot(*headValue)
		}
	} else {
		working, _ := os.Getwd()
		baseSnapshot, headSnapshot, err = gitdiff.CompareRevisions(context.Background(), working, *baseValue, *headValue, cfg, gitdiff.Options{Patterns: positional, Semantics: selected, Reproducible: true, BuildTags: splitComma(*buildTags)})
	}
	if err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitAnalysis
	}
	report := evolution.Compare(baseSnapshot, headSnapshot, cfg)
	if cfg.Evolution.ApprovalsFile != "" {
		if file, openErr := os.Open(cfg.Evolution.ApprovalsFile); openErr == nil {
			approvals, readErr := evolution.ReadApprovals(file)
			file.Close()
			if readErr != nil {
				fmt.Fprintln(stderr, "taglock:", readErr)
				return ExitUsage
			}
			issues := evolution.ApplyApprovals(&report, approvals, time.Now())
			for _, issue := range issues {
				fmt.Fprintf(stderr, "%s %s: %s\n", issue.ID, issue.ApprovalID, issue.Message)
				if issue.ID == "EVOL901" || issue.ID == "EVOL902" {
					code = ExitUsage
				}
			}
		} else if !errors.Is(openErr, os.ErrNotExist) {
			fmt.Fprintln(stderr, "taglock:", openErr)
			return ExitUsage
		}
	}
	if code == ExitUsage {
		return code
	}
	writerFn := func(writer io.Writer) error {
		switch *formatValue {
		case "text":
			return evolution.WriteText(writer, report)
		case "json":
			return evolution.WriteJSON(writer, report)
		case "markdown":
			return evolution.WriteMarkdown(writer, report)
		case "sarif":
			return evolution.WriteSARIF(writer, report)
		default:
			return fmt.Errorf("unsupported compare format %q", *formatValue)
		}
	}
	if err := writeFileOrOutput(*outputPath, stdout, writerFn); err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitUsage
	}
	if report.Summary.Breaking > 0 || report.Summary.Unknown > 0 {
		return ExitViolations
	}
	return ExitOK
}

func runMigrate(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 || arguments[0] != "json-v2" {
		fmt.Fprintln(stderr, "usage: taglock migrate json-v2 [flags] [packages]")
		return ExitUsage
	}
	set := newFlagSet("migrate json-v2", stderr)
	formatValue := set.String("format", "text", "text or json")
	outputPath := set.String("output", "-", "output path or -")
	configPath := set.String("config", "", "configuration path")
	if err := set.Parse(arguments[1:]); err != nil {
		return ExitUsage
	}
	cfg, code := loadConfig(*configPath, stderr)
	if code != ExitOK {
		return code
	}
	result, err := engine.Analyze(context.Background(), set.Args(), cfg)
	if err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitAnalysis
	}
	report := migration.Analyze(result.Contracts, semantics.NewRegistry())
	err = writeFileOrOutput(*outputPath, stdout, func(writer io.Writer) error {
		if *formatValue == "text" {
			_, err := io.WriteString(writer, migration.WriteText(report))
			return err
		}
		if *formatValue == "json" {
			data, err := migration.WriteJSON(report)
			if err == nil {
				_, err = writer.Write(append(data, '\n'))
			}
			return err
		}
		return fmt.Errorf("unsupported migration format %q", *formatValue)
	})
	if err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitUsage
	}
	if len(report.Findings) > 0 {
		return ExitViolations
	}
	return ExitOK
}

func runManifest(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 || (arguments[0] != "package" && arguments[0] != "module") {
		fmt.Fprintln(stderr, "usage: taglock manifest <package|module> [flags] [packages]")
		return ExitUsage
	}
	set := newFlagSet("manifest "+arguments[0], stderr)
	outputPath := set.String("output", "-", "output path or -")
	semanticsValue := set.String("semantics", "", "semantic profile selection")
	configPath := set.String("config", "", "configuration path")
	if err := set.Parse(arguments[1:]); err != nil {
		return ExitUsage
	}
	cfg, code := loadConfig(*configPath, stderr)
	if code != ExitOK {
		return code
	}
	selected := *semanticsValue
	if selected == "" {
		selected = cfg.JSON.Semantics
	}
	result, err := engine.Analyze(context.Background(), set.Args(), cfg)
	if err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitAnalysis
	}
	document, err := snapshot.Build(result, cfg, snapshot.BuildOptions{Semantics: selected, Reproducible: true})
	if err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitAnalysis
	}
	value := manifest.Build(document)
	if err := writeFileOrOutput(*outputPath, stdout, func(writer io.Writer) error { return manifest.Write(writer, value) }); err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitAnalysis
	}
	return ExitOK
}

func runSchema(arguments []string, stdout, stderr io.Writer) int {
	check := len(arguments) > 0 && arguments[0] == "check"
	if check {
		arguments = arguments[1:]
	}
	set := newFlagSet("schema", stderr)
	formatValue := set.String("format", "", "json-schema or openapi")
	outputPath := set.String("output", "", "output path")
	profile := set.String("profile", "json/v1", "semantic profile")
	configPath := set.String("config", "", "configuration path")
	if err := set.Parse(arguments); err != nil {
		return ExitUsage
	}
	cfg, code := loadConfig(*configPath, stderr)
	if code != ExitOK {
		return code
	}
	formatName := *formatValue
	if formatName == "" {
		formatName = cfg.Schema.Format
	}
	result, err := engine.Analyze(context.Background(), set.Args(), cfg)
	if err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitAnalysis
	}
	document, err := snapshot.Build(result, cfg, snapshot.BuildOptions{Semantics: strings.TrimPrefix(*profile, "json/"), Reproducible: true})
	if err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitAnalysis
	}
	value, err := schema.Generate(document, schema.Options{Format: formatName, Profile: *profile})
	if err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitUsage
	}
	var generated bytes.Buffer
	if err := schema.Write(&generated, value); err != nil {
		return ExitAnalysis
	}
	destination := *outputPath
	if destination == "" {
		if check {
			destination = cfg.Schema.Output
		} else {
			destination = "-"
		}
	}
	if check {
		existing, err := os.ReadFile(destination)
		if err != nil {
			fmt.Fprintln(stderr, "taglock: schema check:", err)
			return ExitViolations
		}
		equal, err := schema.EqualJSON(existing, generated.Bytes())
		if err != nil {
			fmt.Fprintln(stderr, "taglock: schema check:", err)
			return ExitUsage
		}
		if !equal {
			fmt.Fprintf(stderr, "taglock: generated schema differs from %s; regenerate with `taglock schema --output %s`\n", destination, destination)
			return ExitViolations
		}
		fmt.Fprintln(stdout, "TagLock schema is current")
		return ExitOK
	}
	if err := writeFileOrOutput(destination, stdout, func(writer io.Writer) error { _, err := writer.Write(generated.Bytes()); return err }); err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitAnalysis
	}
	return ExitOK
}

func runVerify(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 || (arguments[0] != "generate" && arguments[0] != "run") {
		fmt.Fprintln(stderr, "usage: taglock verify <generate|run> [packages]")
		return ExitUsage
	}
	command := arguments[0]
	set := newFlagSet("verify "+command, stderr)
	configPath := set.String("config", "", "configuration path")
	if err := set.Parse(arguments[1:]); err != nil {
		return ExitUsage
	}
	cfg, code := loadConfig(*configPath, stderr)
	if code != ExitOK {
		return code
	}
	if command == "run" {
		patterns := set.Args()
		if len(patterns) == 0 {
			patterns = []string{"./..."}
		}
		args := append([]string{"test"}, patterns...)
		process := exec.Command("go", args...)
		process.Stdout = stdout
		process.Stderr = stderr
		if err := process.Run(); err != nil {
			return ExitViolations
		}
		return ExitOK
	}
	result, err := engine.Analyze(context.Background(), set.Args(), cfg)
	if err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitAnalysis
	}
	files, err := verify.Generate(result, cfg)
	if err == nil {
		err = verify.Write(files)
	}
	if err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitAnalysis
	}
	for _, file := range files {
		fmt.Fprintln(stdout, "generated", file.Path)
	}
	return ExitOK
}

func runChanges(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 || arguments[0] != "validate" {
		fmt.Fprintln(stderr, "usage: taglock changes validate [--file path]")
		return ExitUsage
	}
	set := newFlagSet("changes validate", stderr)
	filename := set.String("file", ".taglock/changes.yml", "approval file")
	if err := set.Parse(arguments[1:]); err != nil {
		return ExitUsage
	}
	file, err := os.Open(*filename)
	if err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitUsage
	}
	approvals, err := evolution.ReadApprovals(file)
	file.Close()
	if err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return ExitUsage
	}
	empty := evolution.Report{}
	issues := evolution.ApplyApprovals(&empty, approvals, time.Now())
	invalid := false
	for _, issue := range issues {
		if issue.ID != "EVOL903" {
			invalid = true
			fmt.Fprintf(stderr, "%s %s: %s\n", issue.ID, issue.ApprovalID, issue.Message)
		}
	}
	if invalid {
		return ExitUsage
	}
	fmt.Fprintln(stdout, "TagLock change approvals are valid")
	return ExitOK
}

func readSnapshot(filename string) (snapshot.Snapshot, error) {
	file, err := os.Open(filename)
	if err != nil {
		return snapshot.Snapshot{}, err
	}
	defer file.Close()
	return snapshot.Read(file)
}
func writeFileOrOutput(destination string, stdout io.Writer, write func(io.Writer) error) error {
	if destination == "" || destination == "-" {
		return write(stdout)
	}
	directory := filepath.Dir(destination)
	if directory != "." {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return err
		}
	}
	file, err := os.Create(destination)
	if err != nil {
		return err
	}
	writeErr := write(file)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}
func splitComma(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := parts[:0]
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func loadConfig(explicit string, stderr io.Writer) (config.Config, int) {
	pathValue := explicit
	if pathValue == "" {
		working, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(stderr, "taglock:", err)
			return config.Config{}, ExitUsage
		}
		pathValue, err = config.Discover(working)
		if err != nil {
			fmt.Fprintln(stderr, "taglock:", err)
			return config.Config{}, ExitUsage
		}
	}
	if pathValue == "" {
		cfg := config.Default()
		return cfg, ExitOK
	}
	cfg, err := config.Load(pathValue)
	if err != nil {
		fmt.Fprintln(stderr, "taglock:", err)
		return config.Config{}, ExitUsage
	}
	return cfg, ExitOK
}
func newFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	set := flag.NewFlagSet("taglock "+name, flag.ContinueOnError)
	set.SetOutput(stderr)
	return set
}
func writeUsage(writer io.Writer) {
	fmt.Fprintln(writer, "TagLock — compile-time confidence for Go's runtime metadata")
	fmt.Fprintln(writer, "\nUsage:\n  taglock check [flags] [packages]\n  taglock fix [flags] [packages]\n  taglock snapshot [flags] [packages]\n  taglock compare <base.json> <head.json>\n  taglock compare --base REV --head REV [packages]\n  taglock migrate json-v2 [packages]\n  taglock manifest <package|module> [packages]\n  taglock schema [check] [packages]\n  taglock verify <generate|run> [packages]\n  taglock changes validate\n  taglock rules\n  taglock explain <rule-id>\n  taglock init\n  taglock config <validate|print>\n  taglock baseline <create|update> [packages]")
}

// MarshalDiagnostics is retained for small CLI integration helpers.
func MarshalDiagnostics(fileSet *token.FileSet, diagnostics []rules.Diagnostic) ([]byte, error) {
	var buffer bytes.Buffer
	if err := output.JSON(&buffer, fileSet, diagnostics); err != nil {
		return nil, err
	}
	var compact any
	if err := json.Unmarshal(buffer.Bytes(), &compact); err != nil {
		return nil, err
	}
	return json.Marshal(compact)
}
