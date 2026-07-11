package app

import (
	"encoding/json"
	"testing"
)

func TestBuildReportPromotionAppendRequestPreservesPayloadContract(t *testing.T) {
	req := BuildReportPromotionAppendRequest(ReportPromotionAppendRequest{
		EventID: "evt_promoted",
		Version: ReportVersion{
			ReportVersionID: "rvn_1",
			MissionID:       "mis_1",
		},
		Producer: Producer{Type: "user", ID: "plasma-ui"},
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
