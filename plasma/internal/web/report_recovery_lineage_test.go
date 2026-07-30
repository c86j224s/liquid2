package web

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func retryPending(id, origin, parent, strategy string) app.LedgerEvent {
	payload, _ := json.Marshal(map[string]any{"origin_pending_event_id": origin, "retry_of_pending_event_id": parent, "retry_strategy": strategy})
	return app.LedgerEvent{EventID: id, MissionID: "mis_1", EventType: "report.draft.pending", Payload: payload}
}

func TestReportRecoveryLineageIncludesAllAncestors(t *testing.T) {
	events := []app.LedgerEvent{retryPending("evt_root", "evt_root", "", "initial"), retryPending("evt_one", "evt_root", "evt_root", "resume_failed"), retryPending("evt_two", "evt_root", "evt_one", "resume_failed")}
	lineage, err := reportRecoveryLineage(events, "evt_two")
	if err != nil {
		t.Fatal(err)
	}
	if len(lineage) != 3 || lineage[0] != "evt_root" || lineage[2] != "evt_two" {
		t.Fatalf("unexpected lineage: %#v", lineage)
	}
}

func TestReportRecoveryLineageRejectsCycle(t *testing.T) {
	events := []app.LedgerEvent{retryPending("evt_one", "evt_one", "evt_two", "resume_failed"), retryPending("evt_two", "evt_one", "evt_one", "resume_failed")}
	if _, err := reportRecoveryLineage(events, "evt_one"); err == nil {
		t.Fatal("expected cycle rejection")
	}
}

func TestReportRecoveryLineageRejectsMissingAncestorAndOriginMismatch(t *testing.T) {
	if _, err := reportRecoveryLineage([]app.LedgerEvent{retryPending("evt_retry", "evt_root", "evt_missing", "resume_failed")}, "evt_retry"); err == nil {
		t.Fatal("expected missing ancestor")
	}
	events := []app.LedgerEvent{retryPending("evt_root", "evt_root", "", "initial"), retryPending("evt_retry", "evt_other", "evt_root", "resume_failed")}
	if _, err := reportRecoveryLineage(events, "evt_retry"); err == nil {
		t.Fatal("expected origin mismatch")
	}
}

func TestReportRecoveryLineageRestartIsIsolated(t *testing.T) {
	events := []app.LedgerEvent{retryPending("evt_root", "evt_root", "", "initial"), retryPending("evt_restart", "evt_root", "evt_root", "restart")}
	lineage, err := reportRecoveryLineage(events, "evt_restart")
	if err != nil {
		t.Fatal(err)
	}
	if len(lineage) != 1 || lineage[0] != "evt_restart" {
		t.Fatalf("restart reused ancestor: %#v", lineage)
	}
}

func TestReportRecoveryLineageRestartBoundsDescendantResume(t *testing.T) {
	events := []app.LedgerEvent{retryPending("evt_a", "evt_a", "", "initial"), retryPending("evt_b", "evt_a", "evt_a", "restart"), retryPending("evt_c", "evt_a", "evt_b", "resume_failed")}
	lineage, err := reportRecoveryLineage(events, "evt_c")
	if err != nil {
		t.Fatal(err)
	}
	if len(lineage) != 2 || lineage[0] != "evt_b" || lineage[1] != "evt_c" {
		t.Fatalf("restart boundary failed: %#v", lineage)
	}
}

func TestApplyPartPlanProgressUsesReportingReplayValidation(t *testing.T) {
	validProgress := func() sectionalReportProgress {
		return sectionalReportProgress{
			agentExecutor:                "codex",
			agentModel:                   "model",
			agentReasoningEffort:         "high",
			agentSelectionSource:         "request",
			reportSessionPolicy:          reportSessionPolicySameSession,
			reportSessionPolicySelection: reporting.SessionPolicySelectionExplicitSameSession,
			generationGuidanceProfile:    reportGenerationGuidanceProfilePartConnectiveEconomyVoice,
			generationGuidanceSHA256:     "guidance-sha",
			sessionChainKind:             "section_fanout_report",
			reportPlanSessionID:          "provider-plan",
			partPlans:                    map[int]sectionFanoutPartPlan{},
		}
	}
	t.Run("accepts canonical stored plan", func(t *testing.T) {
		progress := validProgress()
		if err := applyPartPlanProgress("mis_recovery", "evt_pending", "evt_plan", 1, partPlanRecoveryEvent(t, nil), &progress); err != nil {
			t.Fatal(err)
		}
		plan, ok := progress.partPlans[0]
		if !ok || plan.brief != "canonical Part brief" || plan.providerSessionID != "provider-part-owner" {
			t.Fatalf("unexpected recovered plan: %#v", progress.partPlans)
		}
	})
	for _, tc := range []struct {
		name   string
		mutate func(*app.AppendEventRequest)
	}{
		{name: "producer drift", mutate: func(req *app.AppendEventRequest) {
			req.Producer = app.Producer{Type: "agent_session", ID: "wrong-owner"}
		}},
		{name: "causation drift", mutate: func(req *app.AppendEventRequest) {
			req.CausationEventID = "evt_wrong_plan"
		}},
		{name: "correlation drift", mutate: func(req *app.AppendEventRequest) {
			req.CorrelationID = "wrong-correlation"
		}},
		{name: "executor drift", mutate: func(req *app.AppendEventRequest) {
			payload := recoveryPayload(t, *req)
			payload["agent_executor"] = "claude"
			req.Payload = mustJSON(payload)
		}},
		{name: "session policy drift", mutate: func(req *app.AppendEventRequest) {
			payload := recoveryPayload(t, *req)
			payload["report_session_policy"] = "isolated_fork"
			req.Payload = mustJSON(payload)
		}},
		{name: "returned session drift", mutate: func(req *app.AppendEventRequest) {
			payload := recoveryPayload(t, *req)
			payload["returned_agent_session_id"] = "wrong-owner"
			req.Payload = mustJSON(payload)
		}},
		{name: "fork source drift", mutate: func(req *app.AppendEventRequest) {
			payload := recoveryPayload(t, *req)
			payload["fork_source_agent_session_id"] = "wrong-source"
			req.Payload = mustJSON(payload)
		}},
		{name: "malformed payload", mutate: func(req *app.AppendEventRequest) {
			req.Payload = json.RawMessage(`{`)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			progress := validProgress()
			err := applyPartPlanProgress("mis_recovery", "evt_pending", "evt_plan", 1, partPlanRecoveryEvent(t, tc.mutate), &progress)
			if !errors.Is(err, app.ErrConflict) {
				t.Fatalf("error=%v, want conflict", err)
			}
		})
	}
	t.Run("duplicate canonical plan", func(t *testing.T) {
		progress := validProgress()
		event := partPlanRecoveryEvent(t, nil)
		if err := applyPartPlanProgress("mis_recovery", "evt_pending", "evt_plan", 1, event, &progress); err != nil {
			t.Fatal(err)
		}
		if err := applyPartPlanProgress("mis_recovery", "evt_pending", "evt_plan", 1, event, &progress); !errors.Is(err, app.ErrConflict) {
			t.Fatalf("duplicate error=%v, want conflict", err)
		}
	})
}

func TestSectionFanoutPartPlanningParentUsesCanonicalEventProvenance(t *testing.T) {
	parent, err := sectionFanoutPartPlanningParent(sectionFanoutParentPlanEvent(t, nil), "evt_pending")
	if err != nil {
		t.Fatal(err)
	}
	if parent.AgentExecutor != "codex" ||
		parent.AgentModel != "stored-model" ||
		parent.AgentReasoningEffort != "stored-reasoning" ||
		parent.AgentSelectionSource != "stored-selection" ||
		parent.ReportSessionPolicy != reportSessionPolicySameSession ||
		parent.ReportSessionPolicySelection != reporting.SessionPolicySelectionExplicitSameSession ||
		parent.GenerationGuidanceProfile != reportGenerationGuidanceProfilePartConnectiveEconomyVoice ||
		parent.GenerationGuidanceSHA256 != "stored-guidance-sha" ||
		parent.SessionChainKind != "section_fanout_report" ||
		parent.ReportPlanSessionID != "stored-report-plan-session" {
		t.Fatalf("parent provenance came from a noncanonical source: %#v", parent)
	}
}

func TestEnsureSectionFanoutPlanUsesReplayedLifecycleEventProvenance(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	const missionID = "mis_fresh_plan_replay"
	const pendingID = "evt_fresh_plan_replay_pending"
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "Fresh replay"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID: pendingID, MissionID: missionID, EventType: "report.draft.pending",
		Producer: app.Producer{Type: "user", ID: "test"},
		Payload: mustJSON(map[string]any{
			"kind":                        "markdown_report_artifact_pending",
			"title":                       "Reader Report",
			"report_mode":                 reportModeLongForm,
			"execution_strategy":          reportExecutionStrategySectionFanout,
			"agent_executor":              "claude",
			"mcp_mode":                    "auto",
			"generation_guidance_profile": reportGenerationGuidanceProfilePartConnectiveEconomyVoice,
		}),
	}); err != nil {
		t.Fatal(err)
	}

	executor := &sectionFanoutPlanReplayExecutor{service: svc, plan: narrativeContractTestPlan()}
	server := NewServer(svc, Options{AgentExecutor: executor}).(*Server)
	req := sectionFanoutLongFormRequest{
		missionID: missionID, pendingEventID: pendingID, title: "Reader Report",
		executorName: "claude", agentModel: "request-model", agentReasoningEffort: "request-reasoning",
		agentSelectionSource: "request-selection", mcpMode: "auto", rigor: reportRigorProfiles["balanced"],
		reportSessionPolicy: reportSessionPolicySameSession, reportSessionPolicySelection: "request-policy-selection",
		generationGuidanceProfile: reportGenerationGuidanceProfilePartConnectiveEconomyVoice, generationGuidanceSHA256: "request-guidance-sha",
	}
	state, err := server.ensureSectionFanoutPlan(ctx, req, sectionalReportProgress{}, executor)
	if err != nil {
		t.Fatalf("%v: %v", err, errors.Unwrap(err))
	}
	if state.reportPlanSessionID != "stored-report-plan-session" ||
		state.agentExecutor != "claude" ||
		state.agentModel != "stored-model" ||
		state.agentReasoningEffort != "stored-reasoning" ||
		state.agentSelectionSource != "stored-selection" ||
		state.reportSessionPolicySelection != reporting.SessionPolicySelectionExplicitSameSession ||
		state.generationGuidanceSHA256 != "stored-guidance-sha" ||
		!state.partPlanningEnabled {
		t.Fatalf("fresh state did not use replayed lifecycle event provenance: %#v", state)
	}

	outcome := server.runSectionFanoutPartPlan(ctx, req, state, sectionFanoutPartPlanTask{
		partIndex: 0, part: state.plan.Parts[0], providerSession: "stored-part-owner-session", forkSourceSession: state.reportPlanSessionID,
	}, executor)
	if outcome.err != nil {
		t.Fatal(outcome.err)
	}
	if len(executor.partPlanRequests) != 1 {
		t.Fatalf("expected one downstream Part-plan request, got %#v", executor.partPlanRequests)
	}
	partReq := executor.partPlanRequests[0]
	if partReq.AgentExecutor != "claude" ||
		partReq.Model != "stored-model" ||
		partReq.ReasoningEffort != "stored-reasoning" ||
		partReq.PreviousSessionID != "stored-part-owner-session" {
		t.Fatalf("downstream Part-plan invocation did not use stored parent provenance: %#v", partReq)
	}
	payload := map[string]any{}
	if err := json.Unmarshal(outcome.plan.event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["agent_executor"] != "claude" ||
		payload["agent_model"] != "stored-model" ||
		payload["agent_reasoning_effort"] != "stored-reasoning" ||
		payload["agent_selection_source"] != "stored-selection" ||
		payload["report_plan_session_id"] != "stored-report-plan-session" ||
		payload["generation_guidance_sha256"] != "stored-guidance-sha" {
		t.Fatalf("downstream Part-plan event did not use stored parent provenance: %#v", payload)
	}
}

func TestApplySectionalPlanProgressRejectsInvalidPartPlanningParent(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
	}{
		{name: "missing kind", mutate: func(payload map[string]any) {
			delete(payload, "kind")
		}},
		{name: "wrong kind", mutate: func(payload map[string]any) {
			payload["kind"] = "markdown_report_plan"
		}},
		{name: "missing report plan session", mutate: func(payload map[string]any) {
			delete(payload, "report_plan_session_id")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			progress := sectionalReportProgress{partPlans: map[int]sectionFanoutPartPlan{}}
			err := (&Server{}).applySectionalPlanProgress(context.Background(), "evt_pending", sectionFanoutParentPlanEvent(t, tc.mutate), &progress)
			if !errors.Is(err, app.ErrConflict) {
				t.Fatalf("error=%v, want conflict", err)
			}
			if progress.hasPlan {
				t.Fatalf("invalid parent should not recover progress: %#v", progress)
			}
		})
	}
}

func TestReportDraftPendingRecoveryContractGate(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload json.RawMessage
		want    bool
	}{
		{name: "markdown pending kind", payload: mustJSON(map[string]any{"kind": "markdown_report_artifact_pending"}), want: true},
		{name: "legacy pending kind", payload: mustJSON(map[string]any{"kind": "report_draft_pending"}), want: true},
		{name: "executor and mode", payload: mustJSON(map[string]any{"agent_executor": "codex", "report_mode": reportModeLongForm}), want: true},
		{name: "executor only", payload: mustJSON(map[string]any{"agent_executor": "codex"}), want: false},
		{name: "mode only", payload: mustJSON(map[string]any{"report_mode": reportModeLongForm}), want: false},
		{name: "mcp mode only", payload: mustJSON(map[string]any{"mcp_mode": "auto"}), want: false},
		{name: "execution strategy only", payload: mustJSON(map[string]any{"execution_strategy": reportExecutionStrategySectionFanout}), want: false},
		{name: "title only", payload: mustJSON(map[string]any{"title": "Visible active work"}), want: false},
		{name: "malformed", payload: json.RawMessage(`{`), want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			event := app.LedgerEvent{EventType: "report.draft.pending", Payload: tc.payload}
			if got := reportDraftPendingHasRecoveryContract(event); got != tc.want {
				t.Fatalf("recoverable=%t, want %t", got, tc.want)
			}
		})
	}
}

func TestReportDraftRecoveryLeavesNonRecoverablePendingActive(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	const missionID = "mis_nonrecoverable_pending"
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "Nonrecoverable pending"}); err != nil {
		t.Fatal(err)
	}
	event, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID: "evt_nonrecoverable_report", MissionID: missionID, EventType: "report.draft.pending",
		Producer: app.Producer{Type: "user", ID: "test"}, Payload: mustJSON(map[string]any{"title": "Visible active work"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(svc, Options{}).(*Server)
	if err := server.resumeReportDraftWorker(ctx, missionID, event); err != nil {
		t.Fatal(err)
	}
	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	if countLedgerEvents(events, "report.draft.failed") != 0 || countLedgerEvents(events, "report.artifact.created") != 0 {
		t.Fatalf("nonrecoverable pending should not be terminalized: %#v", events)
	}
	active := app.ActiveWorkFromMissionState(events, nil)
	if len(active.Blocks) != 1 || active.Blocks[0].ReasonCode != app.BlockingReasonReport || active.Blocks[0].PendingEventID != event.EventID {
		t.Fatalf("nonrecoverable pending must remain visible as active work: %#v", active)
	}
}

func partPlanRecoveryEvent(t *testing.T, mutate func(*app.AppendEventRequest)) app.LedgerEvent {
	t.Helper()
	req := reporting.BuildPartPlanCreatedAppendRequest(reporting.PartPlanCreatedEventRequest{
		MarkdownReportStageEventBase: reporting.MarkdownReportStageEventBase{
			EventID: "evt_part_plan", MissionID: "mis_recovery", PendingEventID: "evt_pending", PlanEventID: "evt_plan",
			Title: "Part 1", AgentExecutor: "codex", AgentModel: "model", AgentReasoningEffort: "high",
			AgentSelectionSource: "request", AgentSessionID: "provider-part-owner",
			PreviousAgentSessionID: "provider-part-owner", ReturnedAgentSessionID: "provider-part-owner",
			ToolSessionID: "ses_part_plan", ReportMode: reportModeLongForm,
			ReportSessionPolicy: reportSessionPolicySameSession, ReportSessionPolicySelection: reporting.SessionPolicySelectionExplicitSameSession,
			GenerationGuidanceProfile: reportGenerationGuidanceProfilePartConnectiveEconomyVoice,
			GenerationGuidanceSHA256:  "guidance-sha",
			SessionChainKind:          "section_fanout_report", ReportPlanSessionID: "provider-plan",
			ReportSessionID: "provider-part-owner", ForkSourceAgentSessionID: "provider-plan",
			Producer: app.Producer{Type: "agent_session", ID: "provider-part-owner"},
		},
		PartIndex: 1,
		Brief:     "canonical Part brief",
	})
	if mutate != nil {
		mutate(&req)
	}
	return app.LedgerEvent{
		EventID: req.EventID, MissionID: req.MissionID, EventType: req.EventType, Producer: req.Producer,
		CausationEventID: req.CausationEventID, CorrelationID: req.CorrelationID, Payload: req.Payload,
	}
}

func recoveryPayload(t *testing.T, req app.AppendEventRequest) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func sectionFanoutParentPlanEvent(t *testing.T, mutate func(map[string]any)) app.LedgerEvent {
	t.Helper()
	req := reporting.BuildMarkdownReportPlanCreatedAppendRequest(reporting.MarkdownReportPlanCreatedEventRequest{
		MarkdownReportEventBase: reporting.MarkdownReportEventBase{
			EventID: "evt_plan", MissionID: "mis_recovery", PendingEventID: "evt_pending",
			Title: "Reader Report", AgentExecutor: "codex", AgentModel: "stored-model",
			AgentReasoningEffort: "stored-reasoning", AgentSelectionSource: "stored-selection",
			AgentSessionID: "stored-report-plan-session", ReturnedAgentSessionID: "stored-report-plan-session",
			ToolSessionID: "ses_plan", MCPMode: "auto", ReportMode: reportModeLongForm,
			ReportSessionPolicy: reportSessionPolicySameSession, ReportSessionPolicySelection: reporting.SessionPolicySelectionExplicitSameSession,
			GenerationGuidanceProfile: reportGenerationGuidanceProfilePartConnectiveEconomyVoice,
			GenerationGuidanceSHA256:  "stored-guidance-sha",
			SessionChainKind:          "section_fanout_report", ReportPlanSessionID: "stored-report-plan-session",
			CompositionStrategy: "sectional_preserve_markdown", Producer: app.Producer{Type: "agent_session", ID: "stored-report-plan-session"},
		},
		ArtifactID: "art_plan", Plan: narrativeContractTestPlan(), AssemblyStrategy: "c4_normalized_section_headings",
		PartEditEnabled: true, PartPlanningEnabled: true, PlanReviewState: "auto_accepted",
	})
	payload := recoveryPayload(t, req)
	if mutate != nil {
		mutate(payload)
	}
	req.Payload = mustJSON(payload)
	return app.LedgerEvent{
		EventID: req.EventID, MissionID: req.MissionID, EventType: req.EventType, Producer: req.Producer,
		CausationEventID: req.CausationEventID, CorrelationID: req.CorrelationID, Payload: req.Payload,
	}
}

type sectionFanoutPlanReplayExecutor struct {
	service          *app.Service
	plan             reporting.SectionalReportPlan
	partPlanRequests []AgentRequest
}

func (executor *sectionFanoutPlanReplayExecutor) Run(ctx context.Context, req AgentRequest) (AgentResult, error) {
	if req.ReportPlan != nil {
		plan, err := reporting.NormalizeSectionalReportPlan(executor.plan)
		if err != nil {
			return AgentResult{}, err
		}
		planHash, encoded, err := reporting.ReportPlanHash(plan)
		if err != nil {
			return AgentResult{}, err
		}
		submission, err := executor.service.SubmitReportPlan(ctx, app.ReportPlanSubmissionRequest{
			EventID: "evt_fresh_plan_replay_submission", MissionID: req.MissionID, PendingEventID: req.ReportPlan.PendingEventID,
			ReportMode: req.ReportPlan.ReportMode, ToolSessionID: req.ToolSessionID, PreviousProviderSessionID: req.ReportPlan.PreviousProviderSessionID,
			AgentExecutor: req.AgentExecutor, AgentModel: req.ReportPlan.AgentModel, AgentReasoningEffort: req.ReportPlan.AgentReasoningEffort,
			IdempotencyKey: req.ReportPlan.IdempotencyKey, ArgumentsHash: "fixture-arguments", PlanHash: planHash, Plan: encoded, Attempt: 1,
			ToolProducer: app.Producer{Type: "agent_session", ID: req.ToolSessionID},
		})
		if err != nil {
			return AgentResult{}, err
		}
		canonical := reporting.BuildMarkdownReportPlanCreatedAppendRequest(reporting.MarkdownReportPlanCreatedEventRequest{
			MarkdownReportEventBase: reporting.MarkdownReportEventBase{
				EventID: "evt_fresh_plan_replay_canonical", MissionID: req.MissionID, PendingEventID: req.ReportPlan.PendingEventID,
				Title: "Reader Report", AgentExecutor: "claude", AgentModel: "stored-model", AgentReasoningEffort: "stored-reasoning",
				AgentSelectionSource: "stored-selection", AgentSessionID: "stored-report-plan-session",
				ReturnedAgentSessionID: "stored-report-plan-session", ToolSessionID: req.ToolSessionID, MCPMode: req.MCPMode,
				ReportMode: reportModeLongForm, ReportSessionPolicy: reportSessionPolicySameSession,
				ReportSessionPolicySelection: reporting.SessionPolicySelectionExplicitSameSession,
				GenerationGuidanceProfile:    reportGenerationGuidanceProfilePartConnectiveEconomyVoice,
				GenerationGuidanceSHA256:     "stored-guidance-sha",
				SessionChainKind:             "section_fanout_report", ReportPlanSessionID: "stored-report-plan-session",
				CompositionStrategy: "sectional_preserve_markdown", Producer: app.Producer{Type: "agent_session", ID: "stored-report-plan-session"},
			},
			ArtifactID: "art_fresh_plan_replay", Plan: plan, AssemblyStrategy: "c4_normalized_section_headings",
			PartEditEnabled: true, PartPlanningEnabled: true, PlanReviewState: "auto_accepted",
		})
		payload := map[string]any{}
		if err := json.Unmarshal(canonical.Payload, &payload); err != nil {
			return AgentResult{}, err
		}
		payload["plan_submission"] = map[string]any{
			"submission_event_id": submission.Event.EventID,
			"plan_hash":           planHash,
			"arguments_hash":      "fixture-arguments",
			"tool_session_id":     req.ToolSessionID,
			"idempotency_key":     req.ReportPlan.IdempotencyKey,
		}
		canonical.Payload = mustJSON(payload)
		if _, err := executor.service.AppendEvent(ctx, canonical); err != nil {
			return AgentResult{}, err
		}
		return AgentResult{Text: reporting.ReportPlanSubmittedSentinel, SessionID: "request-returned-session"}, nil
	}
	if strings.HasPrefix(req.UserText, "plan the reading flow for Part") {
		executor.partPlanRequests = append(executor.partPlanRequests, req)
		return AgentResult{Text: "Stored parent Part brief.", SessionID: req.PreviousSessionID, Resumed: true}, nil
	}
	return AgentResult{Text: "unused", SessionID: req.PreviousSessionID}, nil
}
