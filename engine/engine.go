// Package engine runs the shared contract/rule engine across Go packages.
package engine

import (
	"context"
	"fmt"
	"go/token"
	"runtime"
	"sort"
	"strings"

	"github.com/magnexis/taglock/config"
	"github.com/magnexis/taglock/contract"
	"github.com/magnexis/taglock/namespace"
	"github.com/magnexis/taglock/rules"
	"golang.org/x/tools/go/packages"
)

// Result contains deterministic diagnostics and the source position table.
type Result struct {
	FileSet       *token.FileSet
	Diagnostics   []rules.Diagnostic
	Contracts     []*contract.StructContract
	ModulePath    string
	ModuleDir     string
	ModuleVersion string
	GoVersion     string
}

type Options struct {
	Dir       string
	BuildTags []string
}

// Analyze loads package patterns and evaluates every discovered struct.
func Analyze(ctx context.Context, patterns []string, cfg config.Config) (Result, error) {
	return AnalyzeWithOptions(ctx, patterns, cfg, Options{})
}

// AnalyzeWithOptions supports isolated worktrees and explicit build tags.
func AnalyzeWithOptions(ctx context.Context, patterns []string, cfg config.Config, options Options) (Result, error) {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	if err := cfg.Validate(); err != nil {
		return Result{}, err
	}
	fileSet := token.NewFileSet()
	buildFlags := []string{}
	if len(options.BuildTags) > 0 {
		buildFlags = append(buildFlags, "-tags="+strings.Join(options.BuildTags, ","))
	}
	loaded, err := packages.Load(&packages.Config{Context: ctx, Dir: options.Dir, BuildFlags: buildFlags, Fset: fileSet, Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedModule}, patterns...)
	if err != nil {
		return Result{}, fmt.Errorf("load packages: %w", err)
	}
	var packageErrors []string
	packages.Visit(loaded, nil, func(pkg *packages.Package) {
		for _, packageError := range pkg.Errors {
			packageErrors = append(packageErrors, packageError.Error())
		}
	})
	if len(packageErrors) > 0 {
		sort.Strings(packageErrors)
		return Result{}, fmt.Errorf("package analysis could not complete:\n%s", strings.Join(packageErrors, "\n"))
	}
	registry := namespace.NewRegistry(genericOptions(cfg))
	result := Result{FileSet: fileSet, GoVersion: runtime.Version()}
	for _, pkg := range loaded {
		if pkg.Types == nil || pkg.TypesInfo == nil {
			return Result{}, fmt.Errorf("package %s loaded without type information", pkg.ID)
		}
		contracts, issues := contract.BuildPackage(pkg.Syntax, pkg.TypesInfo, pkg.Types, cfg, registry)
		result.Contracts = append(result.Contracts, contracts...)
		if pkg.Module != nil && result.ModulePath == "" {
			result.ModulePath = pkg.Module.Path
			result.ModuleDir = pkg.Module.Dir
			result.ModuleVersion = pkg.Module.Version
		}
		for index, item := range contracts {
			filename := fileSet.Position(item.Position).Filename
			effective := cfg.Effective(filename, pkg.PkgPath)
			itemIssues := []contract.Issue(nil)
			if index == 0 {
				itemIssues = issues
			}
			result.Diagnostics = append(result.Diagnostics, rules.Evaluate(item, effective, itemIssues)...)
		}
	}
	sort.SliceStable(result.Diagnostics, func(i, j int) bool {
		left, right := fileSet.Position(result.Diagnostics[i].Pos), fileSet.Position(result.Diagnostics[j].Pos)
		if left.Filename != right.Filename {
			return left.Filename < right.Filename
		}
		if left.Line != right.Line {
			return left.Line < right.Line
		}
		if left.Column != right.Column {
			return left.Column < right.Column
		}
		if result.Diagnostics[i].Rule.ID != result.Diagnostics[j].Rule.ID {
			return result.Diagnostics[i].Rule.ID < result.Diagnostics[j].Rule.ID
		}
		return result.Diagnostics[i].Message < result.Diagnostics[j].Message
	})
	return result, nil
}

func genericOptions(cfg config.Config) map[string][]string {
	result := map[string][]string{}
	for name, policy := range cfg.NamespacePolicies {
		result[name] = append([]string(nil), policy.KnownOptions...)
	}
	return result
}
