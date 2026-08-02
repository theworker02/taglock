package evolution

type RuleDefinition struct{ ID, Name, Summary string }

var ruleDefinitions = []RuleDefinition{{"EVOL001", "Public contract removed", "A tracked serialized type no longer exists."}, {"EVOL002", "Serialized field removed", "A serialized field is absent from the head contract."}, {"EVOL003", "Serialized field renamed", "A Go field has a different external name."}, {"EVOL004", "Serialized field type changed", "The wire representation type changed."}, {"EVOL005", "Optional field became required", "Legacy inputs may no longer decode."}, {"EVOL006", "Required field became optional", "Consumers may receive omitted data."}, {"EVOL007", "Ignored field became exposed", "Previously internal data is now serialized."}, {"EVOL008", "Exposed field became ignored", "Consumers stop receiving a field."}, {"EVOL009", "Embedded contract changed", "Promotion path changes affect the parent surface."}, {"EVOL010", "Custom marshaler introduced", "Static certainty is reduced by custom code."}, {"EVOL011", "Custom marshaler removed", "Reflection-based behavior replaces custom code."}, {"EVOL012", "Contract certainty reduced", "TagLock can no longer prove the exact wire shape."}, {"EVOL013", "Semantic profile changed", "Snapshots use different implementation profiles."}, {"EVOL014", "Deprecation policy violated", "A removal violates or cannot prove the configured deprecation window."}, {"EVOL015", "Storage migration required", "Persistent serialized data changed incompatibly."}, {"EVOL016", "Undocumented breaking change", "No narrow approval or migration note acknowledges the break."}, {"EVOL100", "Serialized contract added", "A new serialized type appears."}, {"EVOL101", "Serialized field added", "A new field appears in the serialized contract."}, {"EVOL901", "Invalid change approval", "An approval entry is malformed or too broad."}, {"EVOL902", "Expired change approval", "An approval is past its expiry date."}, {"EVOL903", "Unused change approval", "An approval matched no current change."}}

func RuleDefinitions() []RuleDefinition { return append([]RuleDefinition(nil), ruleDefinitions...) }

func knownRule(id string) bool {
	for _, definition := range ruleDefinitions {
		if definition.ID == id {
			return true
		}
	}
	return false
}
