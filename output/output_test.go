package output_test

import (
	"bytes"
	"encoding/json"
	"go/token"
	"strings"
	"testing"

	"github.com/magnexis/taglock/output"
	"github.com/magnexis/taglock/rule"
	"github.com/magnexis/taglock/rules"
)

func testDiagnostic() (*token.FileSet, []rules.Diagnostic) {
	set := token.NewFileSet()
	file := set.AddFile("model.go", -1, 100)
	file.SetLines([]int{0, 10})
	definition, _ := rule.Lookup("TAG104")
	return set, []rules.Diagnostic{{Rule: definition, Severity: rule.SeverityError, Message: "duplicate json name \"id\"", Pos: file.Pos(12), Package: "example", TypeName: "User"}}
}
func TestJSONStableSchema(t *testing.T) {
	set, diagnostics := testDiagnostic()
	var buffer bytes.Buffer
	if err := output.JSON(&buffer, set, diagnostics); err != nil {
		t.Fatal(err)
	}
	var document output.JSONDocument
	if err := json.Unmarshal(buffer.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if document.SchemaVersion != 1 || document.Findings[0].RuleID != "TAG104" {
		t.Fatalf("unexpected document: %#v", document)
	}
}
func TestSARIFContainsRuleAndLocation(t *testing.T) {
	set, diagnostics := testDiagnostic()
	var buffer bytes.Buffer
	if err := output.SARIF(&buffer, set, diagnostics); err != nil {
		t.Fatal(err)
	}
	value := buffer.String()
	for _, expected := range []string{`"version": "2.1.0"`, `"ruleId": "TAG104"`, `"startLine": 2`} {
		if !strings.Contains(value, expected) {
			t.Fatalf("missing %s in %s", expected, value)
		}
	}
}
