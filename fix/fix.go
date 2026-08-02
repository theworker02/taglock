// Package fix models source edits and their safety classification.
package fix

import (
	"go/token"

	"github.com/magnexis/taglock/rule"
)

// Edit replaces one precise source range.
type Edit struct {
	Pos     token.Pos
	End     token.Pos
	NewText []byte
}

// Suggestion is a deterministic group of non-overlapping edits.
type Suggestion struct {
	Message string
	Safety  rule.FixSafety
	Edits   []Edit
}
