package evolution_test

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/theworker02/taglock/config"
	"github.com/theworker02/taglock/evolution"
	"github.com/theworker02/taglock/semantics"
	"github.com/theworker02/taglock/snapshot"
)

func document(field snapshot.FieldSnapshot) snapshot.Snapshot {
	return snapshot.Snapshot{FormatVersion: 1, ModulePath: "example", SemanticProfiles: []string{"json/v1"}, Packages: []snapshot.PackageSnapshot{{ImportPath: "example/api", Contracts: []snapshot.ContractSnapshot{{TypeName: "User", Level: "public", Profiles: []snapshot.SurfaceSnapshot{{Profile: "json/v1", Certainty: semantics.CertaintyExact, Fields: []snapshot.FieldSnapshot{field}}}}}}}}
}

func TestApprovalsAndMachineReports(t *testing.T) {
	report := evolution.Compare(
		document(snapshot.FieldSnapshot{GoName: "Name", ExternalName: "name", GoType: "string", Required: true}),
		document(snapshot.FieldSnapshot{GoName: "Name", ExternalName: "display_name", GoType: "string", Required: true}),
		config.Default(),
	)
	if len(report.Changes) == 0 {
		t.Fatal("expected rename change")
	}
	change := report.Changes[0]
	issues := evolution.ApplyApprovals(&report, evolution.ApprovalFile{Version: 1, Changes: []evolution.Approval{{
		ID: "legacy-name", Rule: change.ID, Contract: "example/api.User", Field: "Name",
		ApprovedFor: "2.0.0", Reason: "legacy client", Migration: "accept both names during rollout",
	}}}, time.Date(2026, time.August, 2, 0, 0, 0, 0, time.UTC))
	if len(issues) != 0 || !report.Changes[0].Acknowledged {
		t.Fatalf("approval not applied: issues=%v report=%#v", issues, report.Changes)
	}
	for name, write := range map[string]func(io.Writer, evolution.Report) error{
		"text": evolution.WriteText, "json": evolution.WriteJSON,
		"markdown": evolution.WriteMarkdown, "sarif": evolution.WriteSARIF,
	} {
		var output bytes.Buffer
		if err := write(&output, report); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !strings.Contains(output.String(), change.ID) {
			t.Fatalf("%s report omits %s: %s", name, change.ID, output.String())
		}
		if (name == "json" || name == "sarif") && !strings.Contains(output.String(), change.ChangeFingerprint) {
			t.Fatalf("%s report omits stable fingerprint", name)
		}
	}
}
func TestCompareClassifiesRenameTypeAndRequiredness(t *testing.T) {
	base := document(snapshot.FieldSnapshot{GoName: "Name", ExternalName: "name", GoType: "string", Required: false})
	head := document(snapshot.FieldSnapshot{GoName: "Name", ExternalName: "display_name", GoType: "int", Required: true})
	report := evolution.Compare(base, head, config.Default())
	seen := map[string]bool{}
	for _, change := range report.Changes {
		seen[change.ID] = true
	}
	for _, id := range []string{"EVOL003", "EVOL004", "EVOL005"} {
		if !seen[id] {
			t.Errorf("missing %s", id)
		}
	}
}

func TestDeprecationReleaseHistoryAndReplacementRename(t *testing.T) {
	base := document(snapshot.FieldSnapshot{GoName: "LegacyName", ExternalName: "legacy_name", GoType: "string", Required: true, Deprecated: &snapshot.DeprecationSnapshot{Since: "1.4.0", RemoveAfter: "2.0.0", Replacement: "DisplayName"}})
	head := document(snapshot.FieldSnapshot{GoName: "DisplayName", ExternalName: "display_name", GoType: "string", Required: true})
	base.ModuleVersion, head.ModuleVersion = "v1.5.0", "v2.0.0"
	cfg := config.Default()
	cfg.Evolution.RequireDeprecationBeforeRemoval = true
	cfg.Evolution.MinimumDeprecationReleases = 2
	cfg.Evolution.ReleaseHistory = []string{"1.4.0", "1.5.0", "2.0.0"}
	report := evolution.Compare(base, head, cfg)
	seenRename := false
	for _, change := range report.Changes {
		if change.ID == "EVOL014" {
			t.Fatalf("valid deprecation window rejected: %#v", change)
		}
		if change.ID == "EVOL003" {
			seenRename = change.RenameConfidence == "confirmed"
		}
		if change.ID == "EVOL002" || change.ID == "EVOL101" {
			t.Fatalf("explicit replacement should be classified as rename: %#v", change)
		}
	}
	if !seenRename {
		t.Fatal("confirmed replacement rename not reported")
	}
}

func TestDeprecationReleaseCountIsEnforced(t *testing.T) {
	base := document(snapshot.FieldSnapshot{GoName: "Email", ExternalName: "email", GoType: "string", Required: false, Deprecated: &snapshot.DeprecationSnapshot{Since: "1.4.0", RemoveAfter: "2.0.0"}})
	head := document(snapshot.FieldSnapshot{GoName: "Placeholder", ExternalName: "placeholder", GoType: "string"})
	head.Packages[0].Contracts[0].Profiles[0].Fields = nil
	base.ModuleVersion, head.ModuleVersion = "v1.5.0", "v2.0.0"
	cfg := config.Default()
	cfg.Evolution.RequireDeprecationBeforeRemoval = true
	cfg.Evolution.MinimumDeprecationReleases = 2
	cfg.Evolution.ReleaseHistory = []string{"1.4.0", "1.5.0", "2.0.0"}
	report := evolution.Compare(base, head, cfg)
	for _, change := range report.Changes {
		if change.ID == "EVOL014" {
			t.Fatalf("valid two-release window rejected: %#v", change)
		}
	}
	cfg.Evolution.MinimumDeprecationReleases = 3
	report = evolution.Compare(base, head, cfg)
	found := false
	for _, change := range report.Changes {
		found = found || change.ID == "EVOL014" && change.Severity == evolution.Breaking
	}
	if !found {
		t.Fatal("short deprecation window was not rejected")
	}
}
func BenchmarkCompare100Contracts(b *testing.B) {
	base := snapshot.Snapshot{FormatVersion: 1, ModulePath: "example", SemanticProfiles: []string{"json/v1"}}
	pkg := snapshot.PackageSnapshot{ImportPath: "example/api"}
	for index := 0; index < 100; index++ {
		pkg.Contracts = append(pkg.Contracts, snapshot.ContractSnapshot{TypeName: fmt.Sprintf("Type%03d", index), Level: "public", Profiles: []snapshot.SurfaceSnapshot{{Profile: "json/v1", Certainty: semantics.CertaintyExact}}})
	}
	base.Packages = []snapshot.PackageSnapshot{pkg}
	head := base
	for range b.N {
		_ = evolution.Compare(base, head, config.Default())
	}
}

func BenchmarkCompare10000Contracts(b *testing.B) {
	base := snapshot.Snapshot{FormatVersion: 1, ModulePath: "example", SemanticProfiles: []string{"json/v1"}}
	pkg := snapshot.PackageSnapshot{ImportPath: "example/api"}
	for index := 0; index < 10000; index++ {
		pkg.Contracts = append(pkg.Contracts, snapshot.ContractSnapshot{TypeName: fmt.Sprintf("Type%05d", index), Level: "public", Profiles: []snapshot.SurfaceSnapshot{{Profile: "json/v1", Certainty: semantics.CertaintyExact}}})
	}
	base.Packages = []snapshot.PackageSnapshot{pkg}
	b.ResetTimer()
	for range b.N {
		_ = evolution.Compare(base, base, config.Default())
	}
}
