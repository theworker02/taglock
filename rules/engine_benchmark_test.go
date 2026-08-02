package rules_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/theworker02/taglock/config"
	"github.com/theworker02/taglock/contract"
	"github.com/theworker02/taglock/namespace"
	"github.com/theworker02/taglock/rules"
)

func BenchmarkCompleteRuleRegistry(b *testing.B) {
	const source = `package sample
		type Credentials struct {
			ID string ` + "`json:\"id\" yaml:\"id\" db:\"id\"`" + `
			LegacyID string ` + "`json:\"id\" yaml:\"legacy_id\"`" + `
			DisplayName string ` + "`json:\"displayName,omitempty,omitempty\" yaml:\"display_name\"`" + `
			PasswordHash string ` + "`json:\"password\" yaml:\"password_hash\"`" + `
			Count int ` + "`json:\"count,string\" yaml:\"count\"`" + `
			Labels []string ` + "`json:\"labels,omitempty\" yaml:\"labels,omitempty\"`" + `
			Metadata map[string]string ` + "`json:\"metadata,omitempty\" yaml:\",inline\"`" + `
		}
	`
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, "contracts.go", source, parser.ParseComments)
	if err != nil {
		b.Fatal(err)
	}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
	pkg, err := (&types.Config{}).Check("sample", set, []*ast.File{file}, info)
	if err != nil {
		b.Fatal(err)
	}
	cfg := config.Default()
	contracts, issues := contract.BuildPackage([]*ast.File{file}, info, pkg, cfg, namespace.NewRegistry(nil))
	if len(contracts) != 1 {
		b.Fatalf("got %d contracts", len(contracts))
	}
	b.ResetTimer()
	for range b.N {
		_ = rules.Evaluate(contracts[0], cfg, issues)
	}
}
