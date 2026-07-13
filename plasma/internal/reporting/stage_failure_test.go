package reporting

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestAppendStageFailureUsesSafePayload(t *testing.T) {
	svc := &fakeRunnerService{}
	runner := Runner{Service: svc, NewID: testRunnerID}
	_, err := runner.AppendStageFailure(context.Background(), StageFailureRequest{MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan", StageKind: "section", PartIndex: 2, SectionIndex: 3, ErrorClass: "agent_failed", Message: "safe failure", Retryable: true, Producer: app.Producer{Type: "agent", ID: "codex"}})
	if err != nil {
		t.Fatal(err)
	}
	events := svc.snapshot()
	if len(events) != 1 || events[0].EventType != "report.section.failed" {
		t.Fatalf("unexpected events: %#v", events)
	}
	var payload map[string]any
	_ = json.Unmarshal(events[0].Payload, &payload)
	if payload["stage_id"] != "section-2-3" || payload["terminal_pending_event_id"] != "evt_pending" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestAppendDraftFailedAssignsStageEventID(t *testing.T) {
	svc := &fakeRunnerService{}
	runner := Runner{Service: svc, NewID: testRunnerID}
	_, err := runner.AppendDraftFailed(context.Background(), "mis_1", "evt_pending", "codex", ModeLongForm, NewStageFailure("section", "evt_plan", 1, 2, context.Canceled))
	if err != nil {
		t.Fatal(err)
	}
	events := svc.snapshot()
	if len(events) != 2 || events[0].EventID == "" || events[0].EventType != "report.section.failed" {
		t.Fatalf("invalid atomic stage events: %#v", events)
	}
	var terminal map[string]any
	_ = json.Unmarshal(events[1].Payload, &terminal)
	if terminal["failed_stage_id"] != "section-1-2" || terminal["stage_failure_event_id"] != events[0].EventID {
		t.Fatalf("terminal lost stage identity: %#v", terminal)
	}
}
