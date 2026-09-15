package reportilsource

import (
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"testing"
)

func TestVerifySourceReadValidatesBeforeEventAccess(t *testing.T) {
	read := func(context.Context, string) ([]ledger.Event, error) {
		t.Fatal("invalid binding/catalog triggered IO")
		return nil, nil
	}
	for _, mission := range []string{"invalid", "mis_one"} {
		_, err := VerifySourceRead(context.Background(), read, mission, "ses_one", "il_reader", reportilcontract.SourceCatalog{MissionID: "mis_one"})
		if err == nil {
			t.Fatal("accepted invalid binding/catalog")
		}
	}
}
