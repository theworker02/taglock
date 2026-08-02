package analyzer

import "fmt"

// ExportedContractFact is a compact versioned cross-package contract summary.
type ExportedContractFact struct {
	Version     int
	TypeName    string
	Fingerprint string
	Profiles    []string
	Certainty   string
}

func (*ExportedContractFact) AFact() {}
func (f *ExportedContractFact) String() string {
	return fmt.Sprintf("v%d:%s:%s:%s", f.Version, f.TypeName, f.Fingerprint, f.Certainty)
}
