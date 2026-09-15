package app

import "github.com/c86j224s/liquid2/plasma/internal/reporting/reportdocument"

import (
	"encoding/json"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"testing"
)

func TestBuildReportPromotionAppendRequestPreservesPayloadContract(t *testing.T) {
	req := BuildReportPromotionAppendRequest(reportdocument.ReportPromotionAppendRequest{
		EventID: "evt_promoted",
		Version: reportdocument.ReportVersion{
			ReportVersionID: "rvn_1",
			MissionID:       "mis_1",
		},
		Producer: ledger.Producer{Type: "user", ID: "plasma-ui"},
	})
	if req.EventID != "evt_promoted" || req.MissionID != "mis_1" ||
		req.EventType != "report.promoted" || req.Producer.Type != "user" || req.Producer.ID != "plasma-ui" {
		t.Fatalf("unexpected report promotion event request: %#v", req)
	}
	var payload struct {
		ReportVersionID string `json:"report_version_id"`
	}
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}
	if payload.ReportVersionID != "rvn_1" {
		t.Fatalf("unexpected report promotion payload: %#v", payload)
	}
}
