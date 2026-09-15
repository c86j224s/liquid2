package reportdocument

import (
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"testing"
)

func TestNormalizeScopePreservesApprovalRestrictions(t *testing.T) {
	for _, scope := range []ReportEvidenceScope{{IncludeProposed: true}, {QuestionIDs: []string{"qst_one"}}, {ClaimIDs: []string{"clm_one", "clm_one"}}} {
		if _, err := NormalizeScope(scope); err == nil {
			t.Fatalf("accepted invalid scope: %+v", scope)
		}
	}
	scope, err := NormalizeScope(ReportEvidenceScope{ClaimIDs: []string{" clm_one "}})
	if err != nil || !scope.AcceptedOnly || scope.IncludeProposed || len(scope.ClaimIDs) != 1 || scope.ClaimIDs[0] != "clm_one" {
		t.Fatalf("scope=%+v err=%v", scope, err)
	}
}

func TestPromotionRequiresExactVersionAndProducer(t *testing.T) {
	event := ledger.Event{EventType: "report.promoted", Producer: ledger.Producer{Type: "user"}, Payload: []byte(`{"report_version_id":"rvn_one"}`)}
	if err := RequirePromotionEvent(event, "rvn_one"); err != nil {
		t.Fatal(err)
	}
	if err := RequirePromotionEvent(event, "rvn_other"); err == nil {
		t.Fatal("accepted wrong version")
	}
	event.Producer.Type = "agent"
	if err := RequirePromotionEvent(event, "rvn_one"); err == nil {
		t.Fatal("accepted agent approval")
	}
}
