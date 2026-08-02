package migration_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/theworker02/taglock/config"
	"github.com/theworker02/taglock/contract"
	"github.com/theworker02/taglock/migration"
	"github.com/theworker02/taglock/namespace"
	"github.com/theworker02/taglock/semantics"
)

func TestMigrationRuleCatalogComplete(t *testing.T) {
	rules := migration.RuleDefinitions()
	for index := 1; index <= 10; index++ {
		id := fmt.Sprintf("JSONMIG%03d", index)
		if _, ok := rules[id]; !ok {
			t.Errorf("missing %s", id)
		}
	}
}

func BenchmarkCrossProfileComparison(b *testing.B) {
	const source = `package sample
		type Child struct { ID string ` + "`json:\"id\"`" + ` }
		type Payload struct { Child Child ` + "`json:\",inline\"`" + `; Values []string ` + "`json:\"values,omitempty\"`" + ` }
	`
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, "sample.go", source, parser.ParseComments)
	if err != nil {
		b.Fatal(err)
	}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
	pkg, err := (&types.Config{}).Check("sample", set, []*ast.File{file}, info)
	if err != nil {
		b.Fatal(err)
	}
	contracts, issues := contract.BuildPackage([]*ast.File{file}, info, pkg, config.Default(), namespace.NewRegistry(nil))
	if len(issues) != 0 {
		b.Fatalf("contract issues: %v", issues)
	}
	registry := semantics.NewRegistry()
	b.ResetTimer()
	for range b.N {
		_ = migration.Analyze(contracts, registry)
	}
}
