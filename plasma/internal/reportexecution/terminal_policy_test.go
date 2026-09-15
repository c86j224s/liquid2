package reportexecution

import (
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"testing"
)

func TestTerminalPolicyPreservesPendingKindCompatibility(t *testing.T) {
	for _, tc := range []struct {
		pending, terminal string
		valid             bool
	}{
		{"report.draft.pending", "report.draft.failed", true},
		{"report.design.pending", "report.artifact.exported", true},
		{"report.humanize.pending", "report.humanize.skipped", true},
		{"report.patch.pending", "report.artifact.created", true},
		{"report.patch.pending", "report.drafted", false},
	} {
		t.Run(tc.pending+"/"+tc.terminal, func(t *testing.T) {
			pending := ledger.Event{EventID: "evt_pending", EventType: tc.pending, Payload: []byte(`{}`)}
			terminal := ledger.Event{EventType: tc.terminal, Payload: []byte(`{"pending_event_id":"evt_pending"}`)}
			isTerminal, err := ValidateReportTerminalAppend(pending, terminal)
			if tc.valid && (err != nil || !isTerminal) {
				t.Fatalf("terminal=%v err=%v", isTerminal, err)
			}
			if !tc.valid && err == nil {
				t.Fatal("accepted incompatible terminal")
			}
		})
	}
}

func TestILStoreReceiptStrictDecoding(t *testing.T) {
	valid := `{"kind":"report_il_stage_progress","pending_event_id":"evt_pending","pipeline_family":"report_il_experimental","stage":"il_store","status":"completed"}`
	if err := ValidateReportILStoreCompletedPayload([]byte(valid), "evt_pending"); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{valid + ` {}`, valid[:len(valid)-1] + `,"extra":true}`, `null`, valid} {
		pending := "evt_pending"
		if raw == valid {
			pending = "evt_other"
		}
		if err := ValidateReportILStoreCompletedPayload([]byte(raw), pending); err == nil {
			t.Fatalf("accepted invalid receipt: %s", raw)
		}
	}
}
