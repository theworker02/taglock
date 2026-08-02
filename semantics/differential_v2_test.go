//go:build goexperiment.jsonv2

package semantics_test

import (
	jsonv2 "encoding/json/v2"
	"strings"
	"testing"

	"github.com/theworker02/taglock/semantics"
)

func TestJSONV2ExperimentAvailable(t *testing.T) {
	if _, err := jsonv2.Marshal(struct {
		ID string `json:"id"`
	}{ID: "x"}); err != nil {
		t.Fatal(err)
	}
}

func TestJSONV2PredictionMatchesExplicitInline(t *testing.T) {
	type Child struct {
		ID string `json:"id"`
	}
	type Payload struct {
		Child Child `json:",inline"`
	}
	data, err := jsonv2.Marshal(Payload{Child: Child{ID: "x"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"id":"x"`) || strings.Contains(string(data), `"Child"`) {
		t.Fatalf("unexpected json/v2 inline output: %s", data)
	}
	item := build(t, `package sample; type Child struct { ID string `+"`json:\"id\"`"+` }; type Payload struct { Child Child `+"`json:\",inline\"`"+` }`)
	surface, err := semantics.NewJSONV2("test", true).ResolveStruct(item)
	if err != nil {
		t.Fatal(err)
	}
	if len(surface.Fields) != 1 || surface.Fields[0].Name != "id" {
		t.Fatalf("TagLock predicted the wrong v2 surface: %#v", surface.Fields)
	}
}
