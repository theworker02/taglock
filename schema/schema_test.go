package schema_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/theworker02/taglock/schema"
	"github.com/theworker02/taglock/semantics"
	"github.com/theworker02/taglock/snapshot"
)

func sample() snapshot.Snapshot {
	return snapshot.Snapshot{ModulePath: "example", SemanticProfiles: []string{"json/v1"}, Packages: []snapshot.PackageSnapshot{{ImportPath: "example/api", Contracts: []snapshot.ContractSnapshot{{TypeName: "User", Profiles: []snapshot.SurfaceSnapshot{{Profile: "json/v1", Certainty: semantics.CertaintyExact, Fields: []snapshot.FieldSnapshot{{GoName: "ID", ExternalName: "id", GoType: "string", Required: true}, {GoName: "Age", ExternalName: "age", GoType: "int"}}}}}}}}}
}
func TestJSONSchemaAndOpenAPI(t *testing.T) {
	for _, format := range []string{"json-schema", "openapi"} {
		value, err := schema.Generate(sample(), schema.Options{Format: format})
		if err != nil {
			t.Fatal(err)
		}
		var buffer bytes.Buffer
		if err := schema.Write(&buffer, value); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buffer.String(), "User") || !strings.Contains(buffer.String(), "id") {
			t.Fatalf("%s output: %s", format, buffer.String())
		}
	}
}
func BenchmarkGenerate(b *testing.B) {
	for range b.N {
		_, _ = schema.Generate(sample(), schema.Options{})
	}
}

func TestSchemaDriftComparisonIgnoresFormatting(t *testing.T) {
	equal, err := schema.EqualJSON([]byte(`{"type":"object","required":["id"]}`), []byte("{\n  \"required\": [\"id\"], \"type\": \"object\"\n}"))
	if err != nil {
		t.Fatal(err)
	}
	if !equal {
		t.Fatal("semantically identical schemas reported as drift")
	}
	equal, err = schema.EqualJSON([]byte(`{"type":"string"}`), []byte(`{"type":"integer"}`))
	if err != nil || equal {
		t.Fatalf("schema drift not detected: equal=%v err=%v", equal, err)
	}
}
