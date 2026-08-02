// Package semantics defines implementation-specific serialization profiles.
package semantics

import (
	"go/types"

	"github.com/magnexis/taglock/contract"
)

type ContractCertainty string

const (
	CertaintyExact   ContractCertainty = "exact"
	CertaintyPartial ContractCertainty = "partial"
	CertaintyOpaque  ContractCertainty = "opaque"
)

type ResolvedSurface struct {
	Profile             string
	Namespace           string
	Certainty           ContractCertainty
	Reason              string
	Fields              []ResolvedField
	CaseSensitiveDecode bool
	Available           bool
	Toolchain           string
	CustomMethods       CustomMethods
}
type ResolvedField struct {
	GoName     string
	Name       string
	GoType     types.Type
	TypeString string
	Path       []string
	Required   bool
	OmitEmpty  bool
	OmitZero   bool
	Nullable   bool
	Embedded   bool
	Ignored    bool
	Deprecated *contract.Deprecation
	Schema     map[string]string
}
type CustomMethods struct {
	MarshalJSON   bool `json:"marshal_json"`
	UnmarshalJSON bool `json:"unmarshal_json"`
	MarshalText   bool `json:"marshal_text"`
	UnmarshalText bool `json:"unmarshal_text"`
}
type Finding struct {
	ID        string
	Message   string
	FieldPath string
}
type SemanticDifference struct {
	Kind      string
	FieldPath string
	Before    any
	After     any
	Message   string
}

type Profile interface {
	ID() string
	Namespace() string
	Version() string
	Available() bool
	ResolveStruct(*contract.StructContract) (*ResolvedSurface, error)
	ValidateField(*contract.FieldContract) []Finding
	Compare(Profile) []SemanticDifference
}
