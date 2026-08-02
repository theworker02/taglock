package analyzer_test

import (
	"testing"

	"github.com/magnexis/taglock/analyzer"
	"github.com/magnexis/taglock/config"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	cfg := config.Default()
	cfg.Naming = map[string]string{"json": "snake_case", "yaml": "snake_case"}
	cfg.Rules.Disable = []string{"TAG303"}
	cfg.Security.OutputNamespaces = []string{"json"}
	analysistest.Run(t, analysistest.TestData(), analyzer.New(cfg), "contracts")
}

func TestRuleFixturesAndSuggestedFixes(t *testing.T) {
	cfg := config.Default()
	cfg.Rules.Disable = []string{"TAG303"}
	cfg.Security.OutputNamespaces = []string{"json"}
	checker := analyzer.New(cfg)
	analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), checker, "tag003")
	analysistest.Run(t, analysistest.TestData(), analyzer.New(cfg), "suppressions")
}
