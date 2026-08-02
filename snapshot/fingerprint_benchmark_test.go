package snapshot

import (
	"fmt"
	"testing"

	"github.com/theworker02/taglock/semantics"
)

func BenchmarkContractFingerprint(b *testing.B) {
	item := ContractSnapshot{TypeName: "Payload", Level: "public", Exported: true}
	surface := SurfaceSnapshot{Profile: "json/v1", Namespace: "json", Certainty: semantics.CertaintyExact}
	for index := 0; index < 32; index++ {
		surface.Fields = append(surface.Fields, FieldSnapshot{GoName: fmt.Sprintf("Field%d", index), ExternalName: fmt.Sprintf("field_%d", index), GoType: "string", WireType: "string", Required: index%2 == 0})
	}
	item.Profiles = []SurfaceSnapshot{surface}
	b.ResetTimer()
	for range b.N {
		_ = fingerprint(contractSemantic(item))
	}
}
