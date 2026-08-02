package baseline_test

import (
	"testing"

	"github.com/magnexis/taglock/baseline"
	"github.com/magnexis/taglock/rule"
	"github.com/magnexis/taglock/rules"
)

func TestFingerprintIgnoresPositionsAndIsStable(t *testing.T) {
	definition, _ := rule.Lookup("TAG104")
	first := rules.Diagnostic{Rule: definition, Package: "example/pkg", TypeName: "User", FieldPath: "User.ID", Namespace: "json", Message: "duplicate JSON name \"id\""}
	second := first
	second.Pos = 999
	if baseline.Fingerprint(first) != baseline.Fingerprint(second) {
		t.Fatal("position changed fingerprint")
	}
	if len(baseline.Fingerprint(first)) != 32 {
		t.Fatal("fingerprint is not compact SHA-256 prefix")
	}
}

func BenchmarkFingerprint(b *testing.B) {
	definition, _ := rule.Lookup("TAG401")
	diagnostic := rules.Diagnostic{Rule: definition, Package: "example", TypeName: "User", FieldPath: "User.Token", Namespace: "json", Message: "sensitive field exposed"}
	for range b.N {
		_ = baseline.Fingerprint(diagnostic)
	}
}
