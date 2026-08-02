package engine_test

import (
	"context"
	"testing"

	"github.com/theworker02/taglock/config"
	"github.com/theworker02/taglock/engine"
)

func TestCollisionHasRelatedLocations(t *testing.T) {
	result, err := engine.Analyze(context.Background(), []string{"./testdata/collision"}, config.Default())
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Rule.ID == "TAG105" {
			if len(diagnostic.Related) < 2 {
				t.Fatalf("related locations=%d", len(diagnostic.Related))
			}
			return
		}
	}
	t.Fatal("TAG105 not reported")
}
