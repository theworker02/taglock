// Package analyzer exposes TagLock as a reusable go/analysis checker.
package analyzer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/types"
	"os"
	"strings"

	"github.com/theworker02/taglock/config"
	"github.com/theworker02/taglock/contract"
	"github.com/theworker02/taglock/namespace"
	"github.com/theworker02/taglock/rules"
	"github.com/theworker02/taglock/semantics"
	"golang.org/x/tools/go/analysis"
)

// Analyzer is the filesystem-configured analyzer used by singlechecker and vet.
var Analyzer = newAnalyzer(nil, "")

// New preserves the Phase 1 constructor for embedders with an in-memory policy.
func New(cfg config.Config) *analysis.Analyzer { return newAnalyzer(&cfg, "") }

type options struct {
	cfg  *config.Config
	path string
}

// Option configures NewWithOptions.
type Option func(*options) error

// WithConfig uses an in-memory policy.
func WithConfig(cfg config.Config) Option {
	return func(options *options) error { clone := cfg.Clone(); options.cfg = &clone; return nil }
}

// WithConfigPath loads a fixed policy path for every analyzed package.
func WithConfigPath(path string) Option {
	return func(options *options) error {
		if path == "" {
			return fmt.Errorf("config path cannot be empty")
		}
		options.path = path
		return nil
	}
}

// NewWithOptions constructs a configurable analyzer without exposing internals.
func NewWithOptions(values ...Option) (*analysis.Analyzer, error) {
	state := &options{}
	for _, apply := range values {
		if apply == nil {
			return nil, fmt.Errorf("nil analyzer option")
		}
		if err := apply(state); err != nil {
			return nil, err
		}
	}
	if state.cfg != nil && state.path != "" {
		return nil, fmt.Errorf("WithConfig and WithConfigPath are mutually exclusive")
	}
	if state.cfg != nil {
		if err := state.cfg.Validate(); err != nil {
			return nil, err
		}
	}
	return newAnalyzer(state.cfg, state.path), nil
}

func newAnalyzer(fixed *config.Config, fixedPath string) *analysis.Analyzer {
	var flagPath string
	checker := &analysis.Analyzer{Name: "taglock", Doc: "checks semantic consistency and safety across Go struct-tag contracts"}
	checker.FactTypes = []analysis.Fact{new(ExportedContractFact)}
	checker.Run = func(pass *analysis.Pass) (any, error) {
		cfg, err := resolveConfig(fixed, firstNonEmpty(fixedPath, flagPath))
		if err != nil {
			return nil, err
		}
		registry := namespace.NewRegistry(genericOptions(cfg))
		contracts, issues := contract.BuildPackage(pass.Files, pass.TypesInfo, pass.Pkg, cfg, registry)
		for index, item := range contracts {
			filename := pass.Fset.Position(item.Position).Filename
			effective := cfg.Effective(filename, pass.Pkg.Path())
			itemIssues := []contract.Issue(nil)
			if index == 0 {
				itemIssues = issues
			}
			for _, diagnostic := range rules.Evaluate(item, effective, itemIssues) {
				report(pass, diagnostic)
			}
			exportFact(pass, item, effective)
		}
		return nil, nil
	}
	if fixed == nil && fixedPath == "" {
		checker.Flags.StringVar(&flagPath, "config", "", "path to a TagLock policy file")
	}
	return checker
}

func exportFact(pass *analysis.Pass, item *contract.StructContract, cfg config.Config) {
	if !item.Exported {
		return
	}
	object, ok := pass.Pkg.Scope().Lookup(item.TypeName).(*types.TypeName)
	if !ok {
		return
	}
	profile := semantics.NewJSONV1("analysis")
	surface, err := profile.ResolveStruct(item)
	if err != nil {
		return
	}
	parts := []string{item.Package, item.TypeName, string(surface.Certainty)}
	for _, field := range surface.Fields {
		parts = append(parts, field.Name, field.TypeString)
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	profiles := []string{"json/v1"}
	if cfg.JSON.Semantics == "v2" || cfg.JSON.Semantics == "both" {
		profiles = append(profiles, "json/v2")
	}
	pass.ExportObjectFact(object, &ExportedContractFact{Version: 1, TypeName: item.TypeName, Fingerprint: hex.EncodeToString(sum[:16]), Profiles: profiles, Certainty: string(surface.Certainty)})
}

func resolveConfig(fixed *config.Config, path string) (config.Config, error) {
	if fixed != nil {
		clone := fixed.Clone()
		if err := clone.Validate(); err != nil {
			return config.Config{}, err
		}
		return clone, nil
	}
	if path == "" {
		workingDirectory, err := os.Getwd()
		if err != nil {
			return config.Config{}, fmt.Errorf("get working directory: %w", err)
		}
		path, err = config.Discover(workingDirectory)
		if err != nil {
			return config.Config{}, fmt.Errorf("discover TagLock config: %w", err)
		}
	}
	if path == "" {
		return config.Default(), nil
	}
	cfg, err := config.Load(path)
	if err != nil {
		return config.Config{}, fmt.Errorf("load TagLock config: %w", err)
	}
	return cfg, nil
}

func report(pass *analysis.Pass, finding rules.Diagnostic) {
	diagnostic := analysis.Diagnostic{Pos: finding.Pos, End: finding.End, Category: finding.Rule.ID, Message: finding.FullMessage()}
	for _, related := range finding.Related {
		diagnostic.Related = append(diagnostic.Related, analysis.RelatedInformation{Pos: related.Pos, End: related.End, Message: related.Message})
	}
	for _, suggestion := range finding.Fixes {
		converted := analysis.SuggestedFix{Message: suggestion.Message}
		for _, edit := range suggestion.Edits {
			converted.TextEdits = append(converted.TextEdits, analysis.TextEdit{Pos: edit.Pos, End: edit.End, NewText: edit.NewText})
		}
		diagnostic.SuggestedFixes = append(diagnostic.SuggestedFixes, converted)
	}
	pass.Report(diagnostic)
}

func genericOptions(cfg config.Config) map[string][]string {
	result := map[string][]string{}
	for name, policy := range cfg.NamespacePolicies {
		result[name] = append([]string(nil), policy.KnownOptions...)
	}
	return result
}
func firstNonEmpty(first, second string) string {
	if first != "" {
		return first
	}
	return second
}
