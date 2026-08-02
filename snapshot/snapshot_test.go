package snapshot_test

import (
	"bytes"
	"fmt"
	"go/token"
	"go/types"
	"testing"

	"github.com/theworker02/taglock/config"
	"github.com/theworker02/taglock/contract"
	"github.com/theworker02/taglock/engine"
	"github.com/theworker02/taglock/snapshot"
)

func TestCanonicalIgnoresGeneratedTimestamp(t *testing.T) {
	first := snapshot.Snapshot{FormatVersion: 1, ModulePath: "example", GeneratedAt: "2026-01-01T00:00:00Z"}
	second := first
	second.GeneratedAt = "2027-01-01T00:00:00Z"
	a, err := snapshot.Canonical(first)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := snapshot.Canonical(second)
	if !bytes.Equal(a, b) {
		t.Fatalf("canonical output changed with timestamp:\n%s\n%s", a, b)
	}
}
func BenchmarkBuildSnapshot(b *testing.B) {
	result := representativeResult()
	cfg := config.Default()
	for range b.N {
		_, _ = snapshot.Build(result, cfg, snapshot.BuildOptions{Semantics: "v1", Reproducible: true})
	}
}

func representativeResult() engine.Result {
	item := &contract.StructContract{TypeName: "Payload", Package: "example/api", Exported: true, Namespaces: map[string]*contract.NamespaceContract{"json": {}}}
	var variables []*types.Var
	for index := 0; index < 32; index++ {
		goName := fmt.Sprintf("Field%d", index)
		externalName := fmt.Sprintf("field_%d", index)
		value := &contract.TagContract{Namespace: "json", Raw: externalName, Name: externalName, Explicit: true}
		field := &contract.FieldContract{GoName: goName, GoType: types.Typ[types.String], Exported: true, Tags: map[string]*contract.TagContract{"json": value}}
		item.Fields = append(item.Fields, field)
		item.Namespaces["json"].Fields = append(item.Namespaces["json"].Fields, &contract.EffectiveField{Field: field, Name: externalName, Path: []string{goName}, Explicit: true, Tag: value})
		variables = append(variables, types.NewVar(token.NoPos, nil, goName, types.Typ[types.String]))
	}
	underlying := types.NewStruct(variables, make([]string, len(variables)))
	item.DeclaredType = types.NewNamed(types.NewTypeName(token.NoPos, nil, "Payload", nil), underlying, nil)
	item.GoType = underlying
	return engine.Result{FileSet: token.NewFileSet(), ModulePath: "example", GoVersion: "go1.24", Contracts: []*contract.StructContract{item}}
}
func TestReadRejectsUnknownFormat(t *testing.T) {
	_, err := snapshot.Read(bytes.NewBufferString(`{"format_version":99}`))
	if err == nil {
		t.Fatal("expected format error")
	}
}
func BenchmarkCanonical(b *testing.B) {
	document := snapshot.Snapshot{FormatVersion: 1, ModulePath: "example", Packages: []snapshot.PackageSnapshot{{ImportPath: "example/api", Contracts: []snapshot.ContractSnapshot{{TypeName: "User"}}}}}
	for range b.N {
		_, _ = snapshot.Canonical(document)
	}
}
