package evolution

import (
	"fmt"
	"io"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type ApprovalFile struct {
	Version int        `yaml:"version"`
	Changes []Approval `yaml:"changes"`
}
type Approval struct {
	ID          string `yaml:"id"`
	Rule        string `yaml:"rule"`
	Contract    string `yaml:"contract"`
	Field       string `yaml:"field"`
	Fingerprint string `yaml:"fingerprint,omitempty"`
	ApprovedFor string `yaml:"approved_for"`
	Reason      string `yaml:"reason"`
	Migration   string `yaml:"migration"`
	Expires     string `yaml:"expires,omitempty"`
}
type ApprovalIssue struct {
	ID         string `json:"id"`
	ApprovalID string `json:"approval_id"`
	Message    string `json:"message"`
}

func ReadApprovals(reader io.Reader) (ApprovalFile, error) {
	var file ApprovalFile
	decoder := yaml.NewDecoder(reader)
	decoder.KnownFields(true)
	if err := decoder.Decode(&file); err != nil {
		return ApprovalFile{}, fmt.Errorf("decode change approvals: %w", err)
	}
	if file.Version != 1 {
		return ApprovalFile{}, fmt.Errorf("unsupported approval version %d", file.Version)
	}
	return file, nil
}
func ApplyApprovals(report *Report, file ApprovalFile, now time.Time) []ApprovalIssue {
	var issues []ApprovalIssue
	used := map[string]bool{}
	for _, approval := range file.Changes {
		if err := validateApproval(approval); err != nil {
			issues = append(issues, ApprovalIssue{ID: "EVOL901", ApprovalID: approval.ID, Message: err.Error()})
			continue
		}
		if approval.Expires != "" {
			expires, err := time.Parse("2006-01-02", approval.Expires)
			if err != nil {
				issues = append(issues, ApprovalIssue{ID: "EVOL901", ApprovalID: approval.ID, Message: "invalid expiry date"})
				continue
			}
			if now.After(expires.Add(24*time.Hour - time.Nanosecond)) {
				issues = append(issues, ApprovalIssue{ID: "EVOL902", ApprovalID: approval.ID, Message: "change approval expired on " + approval.Expires})
				continue
			}
		}
		for index := range report.Changes {
			change := &report.Changes[index]
			if approval.Rule != change.ID || !contractMatches(approval.Contract, change) {
				continue
			}
			if approval.Field != "" && !strings.HasSuffix(change.FieldPath, "."+approval.Field) && change.FieldPath != approval.Field {
				continue
			}
			if approval.Fingerprint != "" && approval.Fingerprint != change.ChangeFingerprint {
				continue
			}
			change.Acknowledged = true
			change.Approval = &ApprovalMatch{ID: approval.ID, Reason: approval.Reason, Migration: approval.Migration}
			used[approval.ID] = true
		}
	}
	for _, approval := range file.Changes {
		if approval.ID != "" && !used[approval.ID] {
			found := false
			for _, issue := range issues {
				if issue.ApprovalID == approval.ID {
					found = true
				}
			}
			if !found {
				issues = append(issues, ApprovalIssue{ID: "EVOL903", ApprovalID: approval.ID, Message: "change approval did not match any finding"})
			}
		}
	}
	var undocumented []Change
	for _, change := range report.Changes {
		if change.Severity == Breaking && !change.Acknowledged && change.ID != "EVOL016" {
			undocumented = append(undocumented, newChange("EVOL016", "undocumented-breaking-change", change.Package, change.TypeName, change.FieldPath, change.Profile, change.Direction, Breaking, change.Before, change.After, "breaking change has no matching approval or migration note", "add a narrow approval with a migration note, or restore compatibility"))
		}
	}
	report.Changes = append(report.Changes, undocumented...)
	sortChanges(report.Changes)
	for index := range report.Changes {
		report.Changes[index].ChangeFingerprint = fingerprint(report.Changes[index])
	}
	report.Summary = summarize(report.Changes)
	return issues
}
func validateApproval(value Approval) error {
	if value.ID == "" || value.Rule == "" || value.Contract == "" || value.ApprovedFor == "" || value.Reason == "" || value.Migration == "" {
		return fmt.Errorf("approval %q must provide id, rule, contract, approved_for, reason, and migration", value.ID)
	}
	if strings.ContainsAny(value.Contract, "*?[") || strings.ContainsAny(value.Field, "*?[") {
		return fmt.Errorf("approval %q uses a broad wildcard", value.ID)
	}
	if !knownRule(value.Rule) || strings.HasPrefix(value.Rule, "EVOL9") {
		return fmt.Errorf("approval %q references unknown or non-breaking rule %q", value.ID, value.Rule)
	}
	return nil
}
func contractMatches(value string, change *Change) bool {
	identity := change.Package + "." + change.TypeName
	return value == identity || strings.HasSuffix(identity, "/"+value) || strings.HasSuffix(identity, "."+value)
}
