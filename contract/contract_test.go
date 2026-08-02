package contract_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/magnexis/taglock/config"
	"github.com/magnexis/taglock/contract"
	"github.com/magnexis/taglock/namespace"
)

func TestBuildPackageResolvesEmbeddedPointersAndCycles(t *testing.T) {
	const source = `package sample
	type Meta struct { ID string ` + "`json:\"id\"`" + `; Next *Meta }
	type Other struct { Legacy string ` + "`json:\"id\"`" + ` }
	type Response[T any] struct { *Meta; Other; Value T ` + "`json:\"value\"`" + ` }
	`
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, "sample.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
	pkg, err := (&types.Config{}).Check("sample", set, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	contracts, issues := contract.BuildPackage([]*ast.File{file}, info, pkg, cfg, namespace.NewRegistry(nil))
	if len(issues) != 0 {
		t.Fatalf("issues: %v", issues)
	}
	var response *contract.StructContract
	for _, item := range contracts {
		if item.TypeName == "Response" {
			response = item
		}
	}
	if response == nil {
		t.Fatal("Response contract not found")
	}
	count := 0
	for _, field := range response.Namespaces["json"].Fields {
		if field.Name == "id" {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("expected two promoted id fields, got %d", count)
	}
}

func BenchmarkResolveMediumStruct(b *testing.B) {
	const source = `package sample; type S struct {
		A string ` + "`json:\"a\" yaml:\"a\"`" + `; B int ` + "`json:\"b\" yaml:\"b\"`" + `
		C []string ` + "`json:\"c,omitempty\" yaml:\"c,omitempty\"`" + `; D map[string]int ` + "`json:\"d,omitempty\"`" + `
		E bool ` + "`json:\"e\"`" + `; F float64 ` + "`json:\"f\"`" + `; G *string ` + "`json:\"g,omitempty\"`" + `
		H [4]byte ` + "`json:\"h\"`" + `; I uint64 ` + "`json:\"i,string\"`" + `; J []byte ` + "`json:\"j,omitempty\"`" + `
		K string ` + "`json:\"k\"`" + `; L string ` + "`json:\"l\"`" + `; M string ` + "`json:\"m\"`" + `
		N string ` + "`json:\"n\"`" + `; O string ` + "`json:\"o\"`" + `; P string ` + "`json:\"p\"`" + `
	}`
	set := token.NewFileSet()
	file, _ := parser.ParseFile(set, "sample.go", source, parser.ParseComments)
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
	pkg, _ := (&types.Config{}).Check("sample", set, []*ast.File{file}, info)
	cfg := config.Default()
	registry := namespace.NewRegistry(nil)
	b.ResetTimer()
	for range b.N {
		contract.BuildPackage([]*ast.File{file}, info, pkg, cfg, registry)
	}
}

func BenchmarkResolveHeavilyEmbeddedStruct(b *testing.B) {
	const source = `package sample
		type Leaf struct { ID string ` + "`json:\"id\"`" + `; Name string ` + "`json:\"name\"`" + ` }
		type A struct { Leaf }; type B struct { *Leaf }; type C struct { Leaf }; type D struct { *Leaf }
		type Left struct { A; B }; type Right struct { C; D }
		type Root struct { Left; *Right }
	`
	set := token.NewFileSet()
	file, _ := parser.ParseFile(set, "embedded.go", source, parser.ParseComments)
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
	pkg, _ := (&types.Config{}).Check("sample", set, []*ast.File{file}, info)
	cfg := config.Default()
	registry := namespace.NewRegistry(nil)
	b.ResetTimer()
	for range b.N {
		contract.BuildPackage([]*ast.File{file}, info, pkg, cfg, registry)
	}
}
