package tag_test

import (
	"testing"

	"github.com/theworker02/taglock/tag"
)

func TestParseRetainsDuplicateNamespaces(t *testing.T) {
	parsed := tag.Parse(`json:"id" json:"legacy"`)
	if len(parsed.Values) != 2 || len(parsed.Problems) != 1 || parsed.Problems[0].Kind != tag.ProblemDuplicateNamespace {
		t.Fatalf("unexpected parse result: %#v", parsed)
	}
}

func TestRemoveDuplicateOptions(t *testing.T) {
	parsed := tag.Parse(`json:"id,omitempty,omitempty,string"`)
	value, _ := parsed.First("json")
	got, changed := tag.RemoveDuplicateOptions(value)
	if !changed || got != "id,omitempty,string" {
		t.Fatalf("got %q, %v", got, changed)
	}
}

func FuzzParse(f *testing.F) {
	for _, seed := range []string{"", `json:"id"`, `json:"x,omitempty,omitempty"`, `bad`, `xml:"name,attr"`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) { _ = tag.Parse(raw) })
}

func BenchmarkParse(b *testing.B) {
	const raw = `json:"display_name,omitempty" yaml:"display_name,omitempty" db:"display_name" validate:"required"`
	for range b.N {
		_ = tag.Parse(raw)
	}
}
