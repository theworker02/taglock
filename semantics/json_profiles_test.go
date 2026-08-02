package semantics_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/magnexis/taglock/config"
	"github.com/magnexis/taglock/contract"
	"github.com/magnexis/taglock/namespace"
	"github.com/magnexis/taglock/semantics"
)

func build(t testing.TB, source string) *contract.StructContract {
	t.Helper()
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
	contracts, _ := contract.BuildPackage([]*ast.File{file}, info, pkg, config.Default(), namespace.NewRegistry(nil))
	for _, item := range contracts {
		if item.TypeName == "Payload" {
			return item
		}
	}
	t.Fatal("contract not found")
	return nil
}
func BenchmarkJSONProfiles(b *testing.B) {
	item := build(b, `package sample; type Child struct { ID string `+"`json:\"id\"`"+` }; type Payload struct { Child; Values []string `+"`json:\"values,omitempty\"`"+` }`)
	profiles := []semantics.Profile{semantics.NewJSONV1("bench"), semantics.NewJSONV2("bench", true)}
	b.ResetTimer()
	for range b.N {
		for _, profile := range profiles {
			_, _ = profile.ResolveStruct(item)
		}
	}
}
func TestJSONV1AndV2SeparateResolution(t *testing.T) {
	item := build(t, `package sample; type Child struct { ID string `+"`json:\"id\"`"+` }; type Payload struct { Child Child `+"`json:\",inline\"`"+` }`)
	v1, _ := semantics.NewJSONV1("go1.26").ResolveStruct(item)
	v2, _ := semantics.NewJSONV2("go1.26", true).ResolveStruct(item)
	if len(v1.Fields) != 1 || v1.Fields[0].Name != "Child" {
		t.Fatalf("v1 surface: %#v", v1.Fields)
	}
	if len(v2.Fields) != 1 || v2.Fields[0].Name != "id" {
		t.Fatalf("v2 surface: %#v", v2.Fields)
	}
}
func TestCustomMarshalerMakesContractOpaque(t *testing.T) {
	item := build(t, `package sample; type Payload struct { ID string }; func (Payload) MarshalJSON() ([]byte,error) { return nil,nil }`)
	surface, _ := semantics.NewJSONV1("go1.26").ResolveStruct(item)
	if surface.Certainty != semantics.CertaintyOpaque {
		t.Fatalf("certainty=%s", surface.Certainty)
	}
}
