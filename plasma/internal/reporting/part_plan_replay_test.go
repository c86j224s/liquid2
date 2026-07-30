package reporting_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestFinalizePartPlanRejectsStoredProvenanceDrift(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name   string
		mutate func(*app.AppendEventRequest)
	}{
		{name: "producer drift", mutate: func(req *app.AppendEventRequest) {
			req.Producer = app.Producer{Type: "agent_session", ID: "wrong-session"}
		}},
		{name: "correlation drift", mutate: func(req *app.AppendEventRequest) {
			req.CorrelationID = "wrong-correlation"
		}},
		{name: "fork source drift", mutate: func(req *app.AppendEventRequest) {
			payload := partPlanRequestPayload(t, *req)
			payload["fork_source_agent_session_id"] = "wrong-source"
			req.Payload = testJSON(payload)
		}},
		{name: "tool session missing", mutate: func(req *app.AppendEventRequest) {
			payload := partPlanRequestPayload(t, *req)
			payload["tool_session_id"] = ""
			req.Payload = testJSON(payload)
		}},
		{name: "returned session drift", mutate: func(req *app.AppendEventRequest) {
			payload := partPlanRequestPayload(t, *req)
			payload["returned_agent_session_id"] = "wrong-owner"
			req.Payload = testJSON(payload)
		}},
		{name: "report session drift", mutate: func(req *app.AppendEventRequest) {
			payload := partPlanRequestPayload(t, *req)
			payload["report_session_id"] = "wrong-owner"
			req.Payload = testJSON(payload)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, closeStore := newPartPlanFixture(t, ctx, true, 1)
			defer closeStore()
			req := reporting.BuildPartPlanCreatedAppendRequest(partPlanRequest(1))
			req.EventID = "evt_existing_part_plan"
			tc.mutate(&req)
			if _, err := svc.AppendEvent(ctx, req); err != nil {
				t.Fatal(err)
			}
			retry := partPlanRequest(1)
			retry.EventID = "evt_retry_part_plan"
			retry.Brief = "retry supplied a different brief that must not be compared"
			if _, err := reporting.FinalizePartPlan(ctx, svc, retry); !errors.Is(err, app.ErrConflict) {
				t.Fatalf("error=%v, want conflict", err)
			}
		})
	}
}

func partPlanRequestPayload(t *testing.T, req app.AppendEventRequest) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}
