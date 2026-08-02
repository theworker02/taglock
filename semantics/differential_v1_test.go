package semantics_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/theworker02/taglock/semantics"
)

type DifferentialA struct {
	ID string `json:"id"`
}
type DifferentialB struct {
	Legacy string `json:"id"`
}

func TestJSONV1PredictionMatchesDuplicatePromotion(t *testing.T) {
	payloadType := reflect.StructOf([]reflect.StructField{
		{Name: "DifferentialA", Type: reflect.TypeFor[DifferentialA](), Anonymous: true},
		{Name: "DifferentialB", Type: reflect.TypeFor[DifferentialB](), Anonymous: true},
	})
	payload := reflect.New(payloadType).Elem()
	payload.Field(0).Set(reflect.ValueOf(DifferentialA{ID: "a"}))
	payload.Field(1).Set(reflect.ValueOf(DifferentialB{Legacy: "b"}))
	data, err := json.Marshal(payload.Interface())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"id"`) {
		t.Fatalf("stdlib unexpectedly serialized ambiguous id: %s", data)
	}
	item := build(t, `package sample; type A struct { ID string `+"`json:\"id\"`"+` }; type B struct { Legacy string `+"`json:\"id\"`"+` }; type Payload struct { A; B }`)
	surface, err := semantics.NewJSONV1("test").ResolveStruct(item)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range surface.Fields {
		if field.Name == "id" && !field.Ignored {
			t.Fatalf("TagLock predicted ambiguous id: %#v", surface.Fields)
		}
	}
}
