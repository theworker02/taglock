package cli_test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/magnexis/taglock/internal/cli"
)

func TestExitCodes(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want int
	}{{"success", []string{"explain", "TAG104"}, cli.ExitOK}, {"usage", []string{"explain", "NOPE"}, cli.ExitUsage}, {"violations", []string{"check", "./testdata/violation"}, cli.ExitViolations}, {"analysis", []string{"check", "./testdata/missing"}, cli.ExitAnalysis}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var out, err bytes.Buffer
			if got := cli.Run(test.args, &out, &err); got != test.want {
				t.Fatalf("Run=%d want %d\nout=%s\nerr=%s", got, test.want, out.String(), err.String())
			}
		})
	}
}

func TestSnapshotCompareAndSchemaCommands(t *testing.T) {
	directory := t.TempDir()
	snapshotPath := filepath.Join(directory, "contracts.json")
	var out, errOut bytes.Buffer
	code := cli.Run([]string{"snapshot", "--output", snapshotPath, "--semantics", "v1", "./testdata/violation"}, &out, &errOut)
	if code != cli.ExitOK {
		t.Fatalf("snapshot code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	code = cli.Run([]string{"compare", "--format", "json", snapshotPath, snapshotPath}, &out, &errOut)
	if code != cli.ExitOK || !strings.Contains(out.String(), `"changes": []`) {
		t.Fatalf("compare code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	code = cli.Run([]string{"schema", "--format", "json-schema", "./testdata/violation"}, &out, &errOut)
	if code != cli.ExitOK || !strings.Contains(out.String(), `"$defs"`) {
		t.Fatalf("schema code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
}
func TestRulesAndExplainShareCatalog(t *testing.T) {
	var rulesOut, rulesErr bytes.Buffer
	if code := cli.Run([]string{"rules"}, &rulesOut, &rulesErr); code != 0 {
		t.Fatal(rulesErr.String())
	}
	var explainOut, explainErr bytes.Buffer
	if code := cli.Run([]string{"explain", "TAG401"}, &explainOut, &explainErr); code != 0 {
		t.Fatal(explainErr.String())
	}
	if !strings.Contains(rulesOut.String(), "TAG401") || !strings.Contains(explainOut.String(), "Sensitive field exposed") {
		t.Fatal("catalog output drifted")
	}
}
