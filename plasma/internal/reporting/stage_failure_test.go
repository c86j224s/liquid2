package reporting

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestAppendStageFailureUsesSafePayload(t *testing.T) {
	for _, tc := range []struct {
		name, kind, eventType, stageID string
		part, section                  int
	}{
		{name: "requirements", kind: "requirements", eventType: "report.requirements.failed", stageID: "requirements"},
		{name: "part_plan", kind: "part_plan", eventType: "report.part_plan.failed", stageID: "part-plan-2", part: 2},
		{name: "section", kind: "section", eventType: "report.section.failed", stageID: "section-2-3", part: 2, section: 3},
		{name: "part_edit", kind: "part_edit", eventType: "report.part_edit.failed", stageID: "part-edit-2", part: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakeRunnerService{}
			runner := Runner{Service: svc, NewID: testRunnerID}
			_, err := runner.AppendStageFailure(context.Background(), StageFailureRequest{MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan", StageKind: tc.kind, PartIndex: tc.part, SectionIndex: tc.section, ErrorClass: "agent_failed", Message: "safe failure", Retryable: true, Producer: app.Producer{Type: "agent", ID: "codex"}})
			if err != nil {
				t.Fatal(err)
			}
			events := svc.snapshot()
			if len(events) != 1 || events[0].EventType != tc.eventType {
				t.Fatalf("unexpected events: %#v", events)
			}
			var payload map[string]any
			_ = json.Unmarshal(events[0].Payload, &payload)
			if payload["stage_id"] != tc.stageID || payload["terminal_pending_event_id"] != "evt_pending" {
				t.Fatalf("unexpected payload: %#v", payload)
			}
		})
	}
}

func TestAppendDraftFailedAssignsStageEventID(t *testing.T) {
	for _, tc := range []struct {
		name, kind, eventType, stageID string
		part, section                  int
	}{
		{name: "requirements", kind: "requirements", eventType: "report.requirements.failed", stageID: "requirements"},
		{name: "part_plan", kind: "part_plan", eventType: "report.part_plan.failed", stageID: "part-plan-1", part: 1},
		{name: "section", kind: "section", eventType: "report.section.failed", stageID: "section-1-2", part: 1, section: 2},
		{name: "part_edit", kind: "part_edit", eventType: "report.part_edit.failed", stageID: "part-edit-1", part: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakeRunnerService{}
			runner := Runner{Service: svc, NewID: testRunnerID}
			_, err := runner.AppendDraftFailed(context.Background(), "mis_1", "evt_pending", "codex", ModeLongForm, NewStageFailure(tc.kind, "evt_plan", tc.part, tc.section, context.Canceled))
			if err != nil {
				t.Fatal(err)
			}
			events := svc.snapshot()
			if len(events) != 2 || events[0].EventID == "" || events[0].EventType != tc.eventType {
				t.Fatalf("invalid atomic stage events: %#v", events)
			}
			var terminal map[string]any
			_ = json.Unmarshal(events[1].Payload, &terminal)
			if terminal["failed_stage_id"] != tc.stageID || terminal["stage_failure_event_id"] != events[0].EventID {
				t.Fatalf("terminal lost stage identity: %#v", terminal)
			}
		})
	}
}
