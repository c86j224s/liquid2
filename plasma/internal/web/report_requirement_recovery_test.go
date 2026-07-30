package web

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestReportRequirementRecoverySkipsLegacyStartedOnlyFanoutWithoutExecutor(t *testing.T) {
	ctx := context.Background()
	server, closeStore := newReportRequirementRecoveryServer(t, ctx)
	defer closeStore()
	missionID := "mis_requirement_recovery_started"
	createRequirementRecoveryMission(t, ctx, server.service, missionID)
	pendingID := "evt_pending_requirement_recovery"
	planID := "evt_plan_requirement_recovery"
	plan := recoveryRequirementPlan()
	appendRequirementRecoveryEvent(t, ctx, server.service, app.AppendEventRequest{
		EventID: "evt_unrelated_pending", MissionID: missionID, EventType: "report.draft.pending",
		Producer: app.Producer{Type: "user", ID: "test"}, Payload: mustJSON(map[string]any{}),
	})
	appendRequirementRecoveryEvent(t, ctx, server.service, app.AppendEventRequest{
		EventID: pendingID, MissionID: missionID, EventType: "report.draft.pending",
		Producer: app.Producer{Type: "user", ID: "test"}, Payload: mustJSON(map[string]any{}),
	})
	appendRequirementRecoveryEvent(t, ctx, server.service, app.AppendEventRequest{
		EventID: planID, MissionID: missionID, EventType: "report.plan.created",
		Producer: app.Producer{Type: "agent_session", ID: "report-session"}, Payload: mustJSON(map[string]any{
			"pending_event_id": pendingID, "report_mode": reportModeLongForm, "artifact_id": "art_plan",
			"agent_session_id": "report-session", "report_plan_session_id": "report-session", "plan": plan,
		}),
	})
	wrongPendingStarted := reporting.BuildMarkdownReportSectionStartedAppendRequest(reporting.MarkdownReportSectionStartedEventRequest{
		MarkdownReportStageEventBase: reporting.MarkdownReportStageEventBase{
			EventID: "evt_started_wrong_pending", MissionID: missionID, PendingEventID: "evt_unrelated_pending", PlanEventID: planID,
			AgentSessionID: "wrong-pending-session", ReportMode: reportModeLongForm, Producer: app.Producer{Type: "agent_session", ID: "wrong-pending-session"},
		},
		PartIndex: 1, SectionIndex: 1,
	})
	if reportSectionStartedMatches(pendingID, planID, app.LedgerEvent{Payload: wrongPendingStarted.Payload}) {
		t.Fatal("section started event with unrelated pending matched recovery lineage")
	}
	appendRequirementRecoveryEvent(t, ctx, server.service, wrongPendingStarted)
	wrongPlanStarted := reporting.BuildMarkdownReportSectionStartedAppendRequest(reporting.MarkdownReportSectionStartedEventRequest{
		MarkdownReportStageEventBase: reporting.MarkdownReportStageEventBase{
			EventID: "evt_started_wrong_plan", MissionID: missionID, PendingEventID: pendingID, PlanEventID: "evt_other_plan",
			AgentSessionID: "wrong-plan-session", ReportMode: reportModeLongForm, Producer: app.Producer{Type: "agent_session", ID: "wrong-plan-session"},
		},
		PartIndex: 1, SectionIndex: 1,
	})
	if reportSectionStartedMatches(pendingID, planID, app.LedgerEvent{Payload: wrongPlanStarted.Payload}) {
		t.Fatal("section started event with unrelated plan matched recovery lineage")
	}
	appendRequirementRecoveryEvent(t, ctx, server.service, wrongPlanStarted)
	matchingStarted := reporting.BuildMarkdownReportSectionStartedAppendRequest(reporting.MarkdownReportSectionStartedEventRequest{
		MarkdownReportStageEventBase: reporting.MarkdownReportStageEventBase{
			EventID: "evt_started_matching", MissionID: missionID, PendingEventID: pendingID, PlanEventID: planID,
			AgentSessionID: "matching-section-session", ReportMode: reportModeLongForm, Producer: app.Producer{Type: "agent_session", ID: "matching-section-session"},
		},
		PartIndex: 1, SectionIndex: 1,
	})
	if !reportSectionStartedMatches(pendingID, planID, app.LedgerEvent{Payload: matchingStarted.Payload}) {
		t.Fatal("matching section started event did not match recovery lineage")
	}
	appendRequirementRecoveryEvent(t, ctx, server.service, matchingStarted)
	progress, err := server.loadSectionalReportProgress(ctx, missionID, pendingID)
	if err != nil {
		t.Fatal(err)
	}
	if !progress.hasPlan || !progress.hasPostPlanSectionStarted || progress.hasRequirementStage || len(progress.sections) != 0 || len(progress.parts) != 0 {
		t.Fatalf("unexpected recovered progress: %#v", progress)
	}
	requirementMap, requirementEvent, err := server.ensureReportRequirementMap(ctx, reportRequirementStageRequest{
		missionID: missionID, title: "Recovered fanout", pendingEventID: pendingID,
		planEventID: planID, planSessionID: "report-session", plan: plan,
	}, progress, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(requirementMap.Requirements) != 0 || requirementEvent.EventID != "" {
		t.Fatalf("legacy started-only recovery must not synthesize a requirement map: map=%#v event=%#v", requirementMap, requirementEvent)
	}
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	if countRequirementRecoveryEvents(events, reporting.ReportRequirementsStartedEventType) != 0 || countRequirementRecoveryEvents(events, reporting.ReportRequirementsMappedEventType) != 0 {
		t.Fatalf("legacy skip synthesized requirement events: %#v", events)
	}
}

func TestReportRequirementRecoveryDoesNotSkipWhenRequirementStageStarted(t *testing.T) {
	ctx := context.Background()
	server, closeStore := newReportRequirementRecoveryServer(t, ctx)
	defer closeStore()
	missionID := "mis_requirement_recovery_stage_started"
	createRequirementRecoveryMission(t, ctx, server.service, missionID)
	pendingID := "evt_pending_requirement_stage_started"
	planID := "evt_plan_requirement_stage_started"
	plan := recoveryRequirementPlan()
	appendRequirementRecoveryEvent(t, ctx, server.service, app.AppendEventRequest{
		EventID: pendingID, MissionID: missionID, EventType: "report.draft.pending",
		Producer: app.Producer{Type: "user", ID: "test"}, Payload: mustJSON(map[string]any{}),
	})
	appendRequirementRecoveryEvent(t, ctx, server.service, app.AppendEventRequest{
		EventID: planID, MissionID: missionID, EventType: "report.plan.created",
		Producer: app.Producer{Type: "agent_session", ID: "report-session"}, Payload: mustJSON(map[string]any{
			"pending_event_id": pendingID, "report_mode": reportModeLongForm,
			"agent_session_id": "report-session", "report_plan_session_id": "report-session", "plan": plan,
		}),
	})
	appendRequirementRecoveryEvent(t, ctx, server.service, app.AppendEventRequest{
		EventID: "evt_requirement_started", MissionID: missionID, EventType: reporting.ReportRequirementsStartedEventType,
		Producer: app.Producer{Type: "agent_session", ID: "requirement-session"}, Payload: mustJSON(map[string]any{
			"pending_event_id": pendingID, "plan_event_id": planID,
		}),
	})
	appendRequirementRecoveryEvent(t, ctx, server.service, reporting.BuildMarkdownReportSectionStartedAppendRequest(reporting.MarkdownReportSectionStartedEventRequest{
		MarkdownReportStageEventBase: reporting.MarkdownReportStageEventBase{
			EventID: "evt_started_matching", MissionID: missionID, PendingEventID: pendingID, PlanEventID: planID,
			AgentSessionID: "matching-section-session", ReportMode: reportModeLongForm, Producer: app.Producer{Type: "agent_session", ID: "matching-section-session"},
		},
		PartIndex: 1, SectionIndex: 1,
	}))
	progress, err := server.loadSectionalReportProgress(ctx, missionID, pendingID)
	if err != nil {
		t.Fatal(err)
	}
	if !progress.hasRequirementStage || !progress.hasPostPlanSectionStarted {
		t.Fatalf("expected both requirement and section started provenance, got %#v", progress)
	}
	_, _, err = server.ensureReportRequirementMap(ctx, reportRequirementStageRequest{
		missionID: missionID, title: "Recovered fanout", executorName: "codex", pendingEventID: pendingID,
		planEventID: planID, planSessionID: "report-session", plan: plan,
	}, progress, failingRequirementExecutor{})
	if err == nil || !errors.Is(err, errRequirementExecutorCalled) {
		t.Fatalf("requirement stage was skipped instead of invoking the mapper: %v", err)
	}
}

var errRequirementExecutorCalled = errors.New("requirement executor called")

type failingRequirementExecutor struct{}

func (failingRequirementExecutor) Run(context.Context, AgentRequest) (AgentResult, error) {
	return AgentResult{}, errRequirementExecutorCalled
}

func newReportRequirementRecoveryServer(t *testing.T, ctx context.Context) (*Server, func()) {
	t.Helper()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	return &Server{service: app.NewService(store)}, func() { _ = store.Close() }
}

func appendRequirementRecoveryEvent(t *testing.T, ctx context.Context, service *app.Service, req app.AppendEventRequest) {
	t.Helper()
	if _, err := service.AppendEvent(ctx, req); err != nil {
		t.Fatal(err)
	}
}

func createRequirementRecoveryMission(t *testing.T, ctx context.Context, service *app.Service, missionID string) {
	t.Helper()
	if _, err := service.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "Requirement recovery"}); err != nil {
		t.Fatal(err)
	}
}

func countRequirementRecoveryEvents(events []app.LedgerEvent, eventType string) int {
	count := 0
	for _, event := range events {
		if event.EventType == eventType {
			count++
		}
	}
	return count
}

func recoveryRequirementPlan() agentSectionalReportPlan {
	return agentSectionalReportPlan{
		Summary: "Recovery plan",
		Parts: []agentReportPart{{
			Title:    "Part",
			Sections: []agentReportSection{{Title: "Section", Purpose: "Recover the section."}},
		}},
	}
}
