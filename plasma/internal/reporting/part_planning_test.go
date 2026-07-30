package reporting_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestFinalizePartPlanPersistsAndReplaysCanonicalBrief(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newPartPlanFixture(t, ctx, true, 1)
	defer closeStore()
	req := partPlanRequest(1)

	created, err := reporting.FinalizePartPlan(ctx, svc, req)
	if err != nil {
		t.Fatal(err)
	}
	if created.Event.EventID != "evt_part_plan_1" || created.Brief != "독자가 따라갈 Part의 흐름" || created.ProviderSessionID != "part-owner-session-1" {
		t.Fatalf("unexpected Part plan: %#v", created)
	}

	req.EventID = "evt_ignored"
	req.Brief = "다시 실행하며 달라진 메모"
	replayed, err := reporting.FinalizePartPlan(ctx, svc, req)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Event.EventID != created.Event.EventID || replayed.Brief != created.Brief || replayed.ProviderSessionID != created.ProviderSessionID {
		t.Fatalf("Part plan replay did not return canonical state: %#v", replayed)
	}
	events, err := svc.ListEvents(ctx, "mis_part_plan")
	if err != nil {
		t.Fatal(err)
	}
	if countEventType(events, reporting.PartPlanCreatedEventType) != 1 {
		t.Fatalf("Part plan event count differs: %#v", events)
	}
}

func TestFinalizePartPlanRejectsDisabledMissingParentWrongPartAndProviderSession(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name      string
		enabled   bool
		partCount int
		mutate    func(*reporting.PartPlanCreatedEventRequest)
		want      error
	}{
		{name: "disabled parent", enabled: false, partCount: 1, want: app.ErrConflict},
		{name: "missing parent", enabled: true, partCount: 1, mutate: func(req *reporting.PartPlanCreatedEventRequest) {
			req.PlanEventID = "evt_missing_plan"
		}, want: app.ErrConflict},
		{name: "wrong part", enabled: true, partCount: 1, mutate: func(req *reporting.PartPlanCreatedEventRequest) {
			req.PartIndex = 2
		}, want: app.ErrConflict},
		{name: "missing provider session", enabled: true, partCount: 1, mutate: func(req *reporting.PartPlanCreatedEventRequest) {
			req.AgentSessionID = ""
		}, want: app.ErrInvalidInput},
		{name: "report plan session reuse", enabled: true, partCount: 1, mutate: func(req *reporting.PartPlanCreatedEventRequest) {
			req.AgentSessionID = req.ReportPlanSessionID
			req.Producer = app.Producer{Type: "agent_session", ID: req.AgentSessionID}
		}, want: app.ErrInvalidInput},
		{name: "wrong fork source", enabled: true, partCount: 1, mutate: func(req *reporting.PartPlanCreatedEventRequest) {
			req.ForkSourceAgentSessionID = "wrong-source"
		}, want: app.ErrInvalidInput},
		{name: "oversized brief", enabled: true, partCount: 1, mutate: func(req *reporting.PartPlanCreatedEventRequest) {
			req.Brief = strings.Repeat("x", 16*1024+1)
		}, want: app.ErrInvalidInput},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, closeStore := newPartPlanFixture(t, ctx, tc.enabled, tc.partCount)
			defer closeStore()
			req := partPlanRequest(1)
			if tc.mutate != nil {
				tc.mutate(&req)
			}
			if _, err := reporting.FinalizePartPlan(ctx, svc, req); !errors.Is(err, tc.want) {
				t.Fatalf("error=%v, want %v", err, tc.want)
			}
		})
	}
}

func TestFinalizePartPlanRejectsInvalidParentAuthorization(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
	}{
		{name: "wrong parent kind", mutate: func(payload map[string]any) {
			payload["kind"] = "markdown_report_plan"
		}},
		{name: "missing parent kind", mutate: func(payload map[string]any) {
			delete(payload, "kind")
		}},
		{name: "missing parent report plan session", mutate: func(payload map[string]any) {
			payload["report_plan_session_id"] = ""
		}},
		{name: "wrong parent report plan session", mutate: func(payload map[string]any) {
			payload["report_plan_session_id"] = "other-report-plan-session"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, closeStore := newPartPlanFixtureWithParentMutation(t, ctx, true, 1, tc.mutate)
			defer closeStore()
			if _, err := reporting.FinalizePartPlan(ctx, svc, partPlanRequest(1)); !errors.Is(err, app.ErrConflict) {
				t.Fatalf("error=%v, want conflict", err)
			}
		})
	}
}

func TestFinalizePartPlanRejectsInitialCandidateProvenanceBeforePersist(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newPartPlanFixture(t, ctx, true, 1)
	defer closeStore()
	req := partPlanRequest(1)
	req.AgentExecutor = "Codex"

	if _, err := reporting.FinalizePartPlan(ctx, svc, req); !errors.Is(err, app.ErrConflict) {
		t.Fatalf("error=%v, want conflict", err)
	}
	events, err := svc.ListEvents(ctx, "mis_part_plan")
	if err != nil {
		t.Fatal(err)
	}
	if countEventType(events, reporting.PartPlanCreatedEventType) != 0 {
		t.Fatalf("malformed initial candidate was persisted: %#v", events)
	}
}

func TestFinalizePartPlanRejectsRequestProvenanceDriftFromParentBeforePersist(t *testing.T) {
	ctx := context.Background()
	svc, closeStore := newPartPlanFixture(t, ctx, true, 1)
	defer closeStore()
	req := partPlanRequest(1)
	req.AgentExecutor = "claude"

	if _, err := reporting.FinalizePartPlan(ctx, svc, req); !errors.Is(err, app.ErrConflict) {
		t.Fatalf("error=%v, want conflict", err)
	}
	events, err := svc.ListEvents(ctx, "mis_part_plan")
	if err != nil {
		t.Fatal(err)
	}
	if countEventType(events, reporting.PartPlanCreatedEventType) != 0 {
		t.Fatalf("parent/request provenance drift was persisted: %#v", events)
	}
}

func TestFinalizePartPlanRejectsMalformedAndDuplicateStoredPlans(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name   string
		events []app.AppendEventRequest
	}{
		{name: "malformed canonical", events: []app.AppendEventRequest{partPlanStoredEvent("evt_existing", 1, "", "part-owner-session-1")}},
		{name: "duplicate canonical", events: []app.AppendEventRequest{
			partPlanStoredEvent("evt_existing_1", 1, "brief one", "part-owner-session-1"),
			partPlanStoredEvent("evt_existing_2", 1, "brief two", "part-owner-session-2"),
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, closeStore := newPartPlanFixture(t, ctx, true, 1)
			defer closeStore()
			if _, err := svc.AppendEvents(ctx, "mis_part_plan", tc.events); err != nil {
				t.Fatal(err)
			}
			if _, err := reporting.FinalizePartPlan(ctx, svc, partPlanRequest(1)); !errors.Is(err, app.ErrConflict) {
				t.Fatalf("error=%v, want conflict", err)
			}
		})
	}
}

func newPartPlanFixture(t *testing.T, ctx context.Context, partPlanningEnabled bool, partCount int) (*app.Service, func()) {
	t.Helper()
	return newPartPlanFixtureWithParentMutation(t, ctx, partPlanningEnabled, partCount, nil)
}

func newPartPlanFixtureWithParentMutation(t *testing.T, ctx context.Context, partPlanningEnabled bool, partCount int, mutateParent func(map[string]any)) (*app.Service, func()) {
	t.Helper()
	store, err := sqlite.Open(ctx, t.TempDir()+"/plasma.db")
	if err != nil {
		t.Fatal(err)
	}
	svc := app.NewService(store)
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: "mis_part_plan", Title: "Part plan"}); err != nil {
		t.Fatal(err)
	}
	parts := make([]map[string]any, partCount)
	for i := range parts {
		parts[i] = map[string]any{"title": fmt.Sprintf("Part %d", i+1), "sections": []any{map[string]any{"title": "Section"}}}
	}
	planPayload := map[string]any{
		"kind":                            "sectional_markdown_report_plan",
		"pending_event_id":                "evt_pending",
		"report_mode":                     "long_form",
		"agent_executor":                  "codex",
		"agent_model":                     "model",
		"agent_reasoning_effort":          "high",
		"agent_selection_source":          "request",
		"agent_session_id":                "report-plan-session",
		"report_plan_session_id":          "report-plan-session",
		"report_session_policy":           reporting.SessionPolicySameSession,
		"report_session_policy_selection": reporting.SessionPolicySelectionExplicitSameSession,
		"generation_guidance_profile":     "narrative-contract",
		"generation_guidance_sha256":      "guidance-sha",
		"session_chain_kind":              "section_fanout_report",
		"part_edit_enabled":               true,
		"part_planning_enabled":           partPlanningEnabled,
		"plan":                            map[string]any{"parts": parts},
	}
	if mutateParent != nil {
		mutateParent(planPayload)
	}
	if _, err := svc.AppendEvents(ctx, "mis_part_plan", []app.AppendEventRequest{
		{EventID: "evt_pending", MissionID: "mis_part_plan", EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: testJSON(map[string]any{"report_mode": "long_form", "agent_executor": "codex"})},
		{EventID: "evt_plan", MissionID: "mis_part_plan", EventType: "report.plan.created", Producer: app.Producer{Type: "agent_session", ID: "report-plan-session"}, Payload: testJSON(planPayload)},
	}); err != nil {
		t.Fatal(err)
	}
	return svc, func() { _ = store.Close() }
}

func partPlanRequest(partIndex int) reporting.PartPlanCreatedEventRequest {
	sessionID := fmt.Sprintf("part-owner-session-%d", partIndex)
	return reporting.PartPlanCreatedEventRequest{
		MarkdownReportStageEventBase: reporting.MarkdownReportStageEventBase{
			EventID: "evt_part_plan_1", MissionID: "mis_part_plan", PendingEventID: "evt_pending", PlanEventID: "evt_plan",
			Title: "Part 1", AgentExecutor: "codex", AgentModel: "model",
			AgentReasoningEffort: "high", AgentSelectionSource: "request",
			AgentSessionID: sessionID, PreviousAgentSessionID: sessionID,
			ReturnedAgentSessionID: sessionID, ToolSessionID: "ses_part_plan", ReportMode: reporting.ModeLongForm,
			ReportSessionPolicy: reporting.SessionPolicySameSession, ReportSessionPolicySelection: reporting.SessionPolicySelectionExplicitSameSession,
			GenerationGuidanceProfile: "narrative-contract", GenerationGuidanceSHA256: "guidance-sha",
			SessionChainKind: "section_fanout_report", ReportPlanSessionID: "report-plan-session",
			ReportSessionID: sessionID, ForkSourceAgentSessionID: "report-plan-session",
			Producer: app.Producer{Type: "agent_session", ID: sessionID},
		},
		PartIndex: partIndex,
		Brief:     "독자가 따라갈 Part의 흐름",
	}
}

func partPlanStoredEvent(eventID string, partIndex int, brief string, sessionID string) app.AppendEventRequest {
	req := partPlanRequest(partIndex)
	req.EventID = eventID
	req.Brief = brief
	req.AgentSessionID = sessionID
	req.PreviousAgentSessionID = sessionID
	req.ReturnedAgentSessionID = sessionID
	req.Producer = app.Producer{Type: "agent_session", ID: sessionID}
	return reporting.BuildPartPlanCreatedAppendRequest(req)
}
