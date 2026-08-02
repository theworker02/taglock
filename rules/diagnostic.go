package rules

import (
	"go/token"

	"github.com/magnexis/taglock/contract"
	"github.com/magnexis/taglock/fix"
	"github.com/magnexis/taglock/rule"
)

// Location points to another declaration participating in a finding.
type Location struct {
	Pos     token.Pos
	End     token.Pos
	Message string
}

// Diagnostic is TagLock's analyzer- and output-independent finding.
type Diagnostic struct {
	Rule      rule.Definition
	Severity  rule.Severity
	Message   string
	Namespace string
	Profile   string
	Pos       token.Pos
	End       token.Pos
	Package   string
	TypeName  string
	FieldPath string
	Related   []Location
	Fixes     []fix.Suggestion
	Contract  *contract.StructContract
	Field     *contract.FieldContract
}

func (d Diagnostic) FullMessage() string { return d.Rule.ID + " " + d.Message }
