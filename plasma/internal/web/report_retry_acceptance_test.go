package web

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/legacyfinalize"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partedit"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

type retryBlockingExecutor struct {
	started  chan struct{}
	release  chan struct{}
	canceled chan struct{}
	calls    atomic.Int32
}

func (e *retryBlockingExecutor) Run(ctx context.Context, _ AgentRequest) (AgentResult, error) {
	e.calls.Add(1)
	select {
	case e.started <- struct{}{}:
	default:
	}
	select {
	case <-e.release:
		return AgentResult{Text: `{"summary":"s","parts":[{"title":"p","sections":[{"title":"s"}]}]}`, SessionID: "ses"}, nil
	case <-ctx.Done():
		close(e.canceled)
		return AgentResult{}, ctx.Err()
	}
}

type partEditRecoveryExecutor struct {
	mu               sync.Mutex
	failPartEditOnce bool
	requests         []AgentRequest
	forkSources      []string
	forkSessions     []string
	forkSequence     int
	planSequence     int
}

func (executor *partEditRecoveryExecutor) Run(_ context.Context, req AgentRequest) (AgentResult, error) {
	executor.mu.Lock()
	executor.requests = append(executor.requests, req)
	sessionID := strings.TrimSpace(req.PreviousSessionID)
	if req.ReportPlan != nil && sessionID == "" {
		executor.planSequence++
		sessionID = fmt.Sprintf("report-plan-session-%d", executor.planSequence)
	}
	failPartEdit := req.PartEdit != nil && executor.failPartEditOnce
	if failPartEdit {
		executor.failPartEditOnce = false
	}
	executor.mu.Unlock()

	var result AgentResult
	var err error
	switch {
	case req.ReportPlan != nil:
		result = AgentResult{Text: agentReportAnyJSON(narrativeContractTestPlan()), SessionID: sessionID}
	case req.ReportRequirements != nil:
		result = AgentResult{Text: "requirements mapped", SessionID: sessionID, Resumed: sessionID != ""}
	case strings.HasPrefix(req.UserText, "plan the reading flow for Part"):
		result = AgentResult{Text: "Part owner brief: setup to reader decision.", SessionID: sessionID, Resumed: true}
	case req.PartAssembly != nil:
		result = AgentResult{Text: `{"intro":"Intro","transitions":[],"closing":"Close"}`, SessionID: sessionID, Resumed: true}
	case req.PartEdit != nil:
		if failPartEdit {
			result = AgentResult{SessionID: sessionID, Log: "part edit failed"}
			err = errors.New("part edit failed")
			break
		}
		result = AgentResult{Text: "# Part 1. Core Part\n\nEdited Part body.\n", SessionID: sessionID, Resumed: true}
	case req.FinalEditStage != nil:
		result = AgentResult{Text: "# Recovered report\n\nEdited Part body.\n", SessionID: sessionID, Resumed: true}
	case req.LongFormFinalize != nil:
		result = AgentResult{Text: "# Recovered report\n\nEdited Part body.\n", SessionID: sessionID, Resumed: true}
	default:
		result = AgentResult{Text: "Section body.", SessionID: sessionID, Resumed: sessionID != ""}
	}
	return result, err
}

func (executor *partEditRecoveryExecutor) ForkSession(_ context.Context, sourceSessionID string) (AgentSessionForkResult, error) {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	executor.forkSequence++
	sessionID := fmt.Sprintf("forked-report-session-%d", executor.forkSequence)
	executor.forkSources = append(executor.forkSources, sourceSessionID)
	executor.forkSessions = append(executor.forkSessions, sessionID)
	return AgentSessionForkResult{
		SessionID: sessionID, SourceSessionID: sourceSessionID,
		SourceHash: "source-hash", CloneHash: "clone-hash", SourceSizeBytes: 100, CloneSizeBytes: 100,
	}, nil
}

func (executor *partEditRecoveryExecutor) CheckForkSession(_ context.Context, sourceSessionID string) error {
	if strings.TrimSpace(sourceSessionID) == "" {
		return fmt.Errorf("source session id is required")
	}
	return nil
}

func (executor *partEditRecoveryExecutor) snapshotRequests() []AgentRequest {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	return append([]AgentRequest(nil), executor.requests...)
}

func TestReportRetryHTTPIdempotencyStartsOneDetachedWorker(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	exec := &retryBlockingExecutor{started: make(chan struct{}, 2), release: make(chan struct{}), canceled: make(chan struct{})}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: exec}))
	defer server.Close()
	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "retry"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	if _, err := svc.AppendEvents(ctx, missionID, []app.AppendEventRequest{{EventID: "evt_failed", MissionID: missionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: mustJSON(map[string]any{"title": "retry", "agent_executor": "codex", "mcp_mode": "auto", "report_mode": "long_form", "source_context": map[string]any{"schema_version": "plasma.report_source_context.v1", "captured_at": "2026-07-14T01:02:03Z", "confluence_sources": []any{}}})}, {EventID: "evt_terminal", MissionID: missionID, EventType: "report.draft.failed", Producer: app.Producer{Type: "agent", ID: "codex"}, Payload: mustJSON(map[string]any{"pending_event_id": "evt_failed", "kind": "report_draft_failed", "failed_stage_id": "plan"})}}); err != nil {
		t.Fatal(err)
	}
	body := map[string]any{"failed_pending_event_id": "evt_failed", "strategy": "resume_failed", "retry_request_id": "request_1"}
	first := postJSON(t, server.URL+"/api/missions/"+missionID+"/reports/retry", body)
	second := postJSON(t, server.URL+"/api/missions/"+missionID+"/reports/retry", body)
	firstID := nestedString(t, first, "pending_event", "EventID")
	if firstID == "" || firstID != nestedString(t, second, "pending_event", "EventID") {
		t.Fatalf("idempotency mismatch: %#v %#v", first, second)
	}
	if capturedAt := nestedString(t, first, "pending_event", "Payload", "source_context", "captured_at"); capturedAt != "2026-07-14T01:02:03Z" {
		t.Fatalf("retry recaptured report source context: %#v", first)
	}
	select {
	case <-exec.started:
	case <-time.After(time.Second):
		t.Fatal("retry worker did not start")
	}
	time.Sleep(20 * time.Millisecond)
	if exec.calls.Load() != 1 {
		t.Fatalf("expected one worker, got %d", exec.calls.Load())
	}
	select {
	case <-exec.canceled:
		t.Fatal("accepted retry worker was canceled with the completed HTTP request")
	default:
	}
	close(exec.release)
}

func TestSectionFanoutPartPlanningContinuityAndFinalPartAuthor(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	delegate := &partEditRecoveryExecutor{}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, delegate)}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "part planning continuity"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title": "Reader Report", "report_mode": "long_form", "execution_strategy": reportExecutionStrategySectionFanout,
		"post_report_humanize": "disabled", "generation_guidance_profile": reportprompt.ProfilePartConnectiveEconomyVoice,
	})
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	for eventType, want := range map[string]int{
		reporting.PartPlanCreatedEventType: 1,
		"report.section.created":           1,
		"report.part.created":              1,
		reporting.PartEditStartedEventType: 1,
		reporting.PartEditedEventType:      1,
		"report.artifact.created":          1,
	} {
		if got := countEvents(detail, eventType); got != want {
			t.Fatalf("%s count=%d, want %d: %#v", eventType, got, want, detail["events"])
		}
	}
	planPayload := lastEventPayload(t, detail, "report.plan.created")
	if planPayload["part_planning_enabled"] != true || planPayload["part_edit_enabled"] != true {
		t.Fatalf("plan payload did not persist Part planning/editing atomically: %#v", planPayload)
	}
	partPlan := lastEventPayload(t, detail, reporting.PartPlanCreatedEventType)
	partPlanSession := partPlan["agent_session_id"].(string)
	reportPlanSession := partPlan["report_plan_session_id"].(string)
	if partPlanSession == "" || reportPlanSession == "" || partPlanSession == reportPlanSession || partPlan["fork_source_agent_session_id"] != reportPlanSession {
		t.Fatalf("Part plan session provenance mismatch: %#v", partPlan)
	}
	requests := delegate.snapshotRequests()
	partPlanRequests := partPlanAgentRequests(requests)
	if len(partPlanRequests) != 1 || partPlanRequests[0].PreviousSessionID != partPlanSession {
		t.Fatalf("Part planner did not resume the forked Part session: session=%q requests=%#v", partPlanSession, partPlanRequests)
	}
	sectionRequests := sectionAgentRequests(requests)
	if len(sectionRequests) != 1 || sectionRequests[0].PreviousSessionID == partPlanSession {
		t.Fatalf("Section writer must use a fork, not the owner session directly: part=%q requests=%#v", partPlanSession, sectionRequests)
	}
	partRequests := partAssemblyAgentRequests(requests)
	if len(partRequests) != 1 || partRequests[0].PartAssembly == nil || partRequests[0].PreviousSessionID == partPlanSession {
		t.Fatalf("Part assembler must use a fork from the owner session: part=%q requests=%#v", partPlanSession, partRequests)
	}
	partAuthorRequests := partEditAgentRequests(requests)
	if len(partAuthorRequests) != 1 || partAuthorRequests[0].PreviousSessionID != partPlanSession {
		t.Fatalf("final Part author did not resume the Part owner session: part=%q requests=%#v", partPlanSession, partAuthorRequests)
	}
	if binding := partAuthorRequests[0].PartEdit; binding == nil ||
		binding.ProviderSessionID != partPlanSession ||
		binding.ForkSourceAgentSessionID != reportPlanSession ||
		!strings.Contains(partAuthorRequests[0].Prompt, "Part owner brief: setup to reader decision.") ||
		!strings.Contains(partAuthorRequests[0].Prompt, "final author") {
		t.Fatalf("final Part author binding/prompt mismatch: %#v", partAuthorRequests[0])
	}
	finalRequests := finalAgentRequests(requests)
	editedPart := lastEventPayload(t, detail, reporting.PartEditedEventType)
	if len(finalRequests) != 1 || finalRequests[0].LongFormFinalize == nil ||
		len(finalRequests[0].LongFormFinalize.PartArtifactIDs) != 1 ||
		finalRequests[0].LongFormFinalize.PartArtifactIDs[0] != editedPart["artifact_id"] ||
		finalRequests[0].LongFormFinalize.ForkSourceAgentSessionID != reportPlanSession {
		t.Fatalf("final editor did not remain an independent report-plan fork using authored Part: final=%#v edited=%#v", finalRequests, editedPart)
	}
	if len(delegate.forkSources) < 4 ||
		delegate.forkSources[0] != reportPlanSession ||
		delegate.forkSources[1] != partPlanSession ||
		delegate.forkSources[2] != partPlanSession ||
		delegate.forkSources[len(delegate.forkSources)-1] != reportPlanSession {
		t.Fatalf("fanout fork sources did not preserve Part owner continuity: sources=%#v part=%q plan=%q", delegate.forkSources, partPlanSession, reportPlanSession)
	}
}

func TestSectionFanoutPartPlanningRecoversAfterPlanCreatedBeforePartPlan(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	delegate := &partEditRecoveryExecutor{}
	webServer := NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, delegate)}).(*Server)
	server := httptest.NewServer(webServer)
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "plan atomic recovery"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	pendingID := "evt_plan_atomic_pending"
	planID := "evt_plan_atomic_plan"
	plan := narrativeContractTestPlan()
	appended, err := svc.AppendEvents(ctx, missionID, []app.AppendEventRequest{
		{
			EventID:   pendingID,
			MissionID: missionID,
			EventType: "report.draft.pending",
			Producer:  app.Producer{Type: "user", ID: "test"},
			Payload: mustJSON(map[string]any{
				"kind":                        "markdown_report_artifact_pending",
				"title":                       "Reader Report",
				"report_mode":                 reportModeLongForm,
				"execution_strategy":          reportExecutionStrategySectionFanout,
				"agent_executor":              "codex",
				"mcp_mode":                    "auto",
				"rigor_level":                 "balanced",
				"post_report_humanize":        "disabled",
				"generation_guidance_profile": reportprompt.ProfilePartConnectiveEconomyVoice,
			}),
		},
		reporting.BuildMarkdownReportPlanCreatedAppendRequest(reporting.MarkdownReportPlanCreatedEventRequest{
			MarkdownReportEventBase: reporting.MarkdownReportEventBase{
				EventID: planID, MissionID: missionID, PendingEventID: pendingID, Title: "Reader Report",
				AgentExecutor: "codex", AgentSessionID: "plan-atomic-report-session", ReturnedAgentSessionID: "plan-atomic-report-session",
				ToolSessionID: "ses_plan_atomic_plan", MCPMode: "auto", ReportMode: reportModeLongForm,
				ReportSessionPolicy: reportSessionPolicySameSession, ReportSessionPolicySelection: reportexecution.SessionPolicySelectionExplicitSameSession,
				GenerationGuidanceProfile: reportprompt.ProfilePartConnectiveEconomyVoice,
				SessionChainKind:          "section_fanout_report", ReportPlanSessionID: "plan-atomic-report-session",
				CompositionStrategy: "sectional_preserve_markdown", Text: "섹션 병렬 장문 Markdown 리포트 생성 계획을 만들었습니다.",
				Producer: app.Producer{Type: "agent_session", ID: "plan-atomic-report-session"},
			},
			ArtifactID: "art_plan_atomic_final", Plan: plan, AssemblyStrategy: "c4_normalized_section_headings",
			PartEditEnabled: true, PartPlanningEnabled: true, PlanReviewState: "auto_accepted",
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := webServer.resumeReportDraftWorker(ctx, missionID, appended[0]); err != nil {
		t.Fatal(err)
	}
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	if countEvents(detail, "report.plan.created") != 1 ||
		countEvents(detail, reporting.PartPlanCreatedEventType) != 1 ||
		countEvents(detail, "report.artifact.created") != 1 {
		t.Fatalf("plan-atomic recovery did not continue from canonical plan payload: %#v", detail["events"])
	}
	requests := delegate.snapshotRequests()
	if len(partPlanAgentRequests(requests)) != 1 || len(sectionAgentRequests(requests)) != 1 ||
		len(partAssemblyAgentRequests(requests)) != 1 || len(partEditAgentRequests(requests)) != 1 ||
		len(finalAgentRequests(requests)) != 1 {
		t.Fatalf("recovery regenerated the wrong stages or skipped W4 continuity: %#v", requests)
	}
	for _, req := range requests {
		if req.ReportPlan != nil {
			t.Fatalf("plan-atomic recovery must not run the planner again: %#v", requests)
		}
	}
}

func TestReportRetryResumeFailedReusesAcceptedAncestorPartPlan(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	delegate := &partEditRecoveryExecutor{failPartEditOnce: true}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, delegate)}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "resume failed part plan"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title": "Reader Report", "report_mode": "long_form", "execution_strategy": reportExecutionStrategySectionFanout,
		"post_report_humanize": "disabled", "generation_guidance_profile": reportprompt.ProfilePartConnectiveEconomyVoice,
	})
	failed := waitForEventType(t, server.URL, missionID, "report.draft.failed")
	if countEvents(failed, reporting.PartPlanCreatedEventType) != 1 ||
		countEvents(failed, "report.part_edit.failed") != 1 ||
		countEvents(failed, reporting.PartEditedEventType) != 0 {
		t.Fatalf("initial W4 run must fail after one durable Part plan and before final Part author submit: %#v", failed["events"])
	}
	failure := lastEventPayload(t, failed, "report.draft.failed")
	originalPendingID := failure["pending_event_id"].(string)
	partPlan := lastEventPayload(t, failed, reporting.PartPlanCreatedEventType)
	partPlanSession := partPlan["agent_session_id"].(string)

	retry := postJSON(t, server.URL+"/api/missions/"+missionID+"/reports/retry", map[string]any{
		"failed_pending_event_id": originalPendingID, "strategy": "resume_failed", "retry_request_id": "retry-part-plan-resume",
	})
	resumePendingID := nestedString(t, retry, "pending_event", "EventID")
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	if countEvents(detail, reporting.PartPlanCreatedEventType) != 1 ||
		countEvents(detail, reporting.PartEditStartedEventType) != 2 ||
		countEvents(detail, "report.part_edit.failed") != 1 ||
		countEvents(detail, reporting.PartEditedEventType) != 1 {
		t.Fatalf("resume_failed did not reuse accepted Part plan lineage: %#v", detail["events"])
	}
	requests := delegate.snapshotRequests()
	if len(partPlanAgentRequests(requests)) != 1 {
		t.Fatalf("resume_failed reran the Part planner: %#v", requests)
	}
	partAuthors := partEditAgentRequests(requests)
	if len(partAuthors) != 2 || partAuthors[1].PreviousSessionID != partPlanSession ||
		partAuthors[1].PartEdit == nil ||
		partAuthors[1].PartEdit.PendingEventID != resumePendingID ||
		partAuthors[1].PartEdit.ProviderSessionID != partPlanSession {
		t.Fatalf("resume_failed did not resume the accepted Part owner: part=%q authors=%#v", partPlanSession, partAuthors)
	}
}

func TestReportRetryRestartDoesNotReuseAcceptedAncestorPartPlan(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	server := NewServer(svc, Options{}).(*Server)

	const (
		missionID      = "mis_restart_part_plan"
		rootPendingID  = "evt_restart_part_plan_root_pending"
		rootPlanID     = "evt_restart_part_plan_root_plan"
		restartPlanID  = "evt_restart_part_plan_plan"
		restartPending = "evt_restart_part_plan_pending"
	)
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "restart part plan"}); err != nil {
		t.Fatal(err)
	}
	plan := narrativeContractTestPlan()
	rootPartPlan := appendRetryPartPlanningAttempt(t, ctx, svc, missionID, rootPendingID, rootPlanID, "root-plan", plan, true)
	failIDs := 0
	runner := reportexecution.Runner{Service: svc, NewID: func(prefix string) string {
		failIDs++
		return fmt.Sprintf("%s_restart_part_plan_failed_%d", prefix, failIDs)
	}}
	if _, err := runner.AppendDraftFailed(ctx, missionID, rootPendingID, "codex", reportexecution.ModeLongForm, reportexecution.NewStageFailure("part_edit", rootPlanID, 1, 0, errors.New("part edit failed"))); err != nil {
		t.Fatal(err)
	}

	retry, err := svc.RequestReportRetry(ctx, app.ReportRetryRequest{
		EventID: restartPending, MissionID: missionID, FailedPendingEventID: rootPendingID,
		Strategy: "restart", RetryRequestID: "retry-part-plan-restart", Producer: app.Producer{Type: "user", ID: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	progress, err := server.loadSectionalReportProgress(ctx, missionID, retry.EventID)
	if err != nil {
		t.Fatal(err)
	}
	if progress.hasPlan || len(progress.partPlans) != 0 {
		t.Fatalf("restart lineage reused ancestor Part planning state before the new attempt ran: progress=%#v ancestor=%#v", progress, rootPartPlan)
	}

	restartPartPlan := appendRetryPartPlanningAttempt(t, ctx, svc, missionID, retry.EventID, restartPlanID, "restart-plan", plan, false)
	progress, err = server.loadSectionalReportProgress(ctx, missionID, retry.EventID)
	if err != nil {
		t.Fatal(err)
	}
	recovered, ok := progress.partPlans[0]
	if !progress.hasPlan || !progress.partPlanningEnabled || !ok ||
		recovered.providerSessionID != restartPartPlan ||
		recovered.providerSessionID == rootPartPlan {
		t.Fatalf("restart progress did not isolate the new Part plan: progress=%#v ancestor=%q restart=%q", progress, rootPartPlan, restartPartPlan)
	}
}

func TestReportRetryResumeFailedReusesAcceptedAncestorPartEdit(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	delegate := &partEditRecoveryExecutor{failPartEditOnce: true}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, delegate)}))
	defer server.Close()

	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "resume failed part edit"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title": "Reader Report", "report_mode": "long_form", "post_report_humanize": "disabled",
		"generation_guidance_profile": reportprompt.ProfilePartConnectiveEconomyVoice,
	})
	failed := waitForEventType(t, server.URL, missionID, "report.draft.failed")
	failure := lastEventPayload(t, failed, "report.draft.failed")
	if failure["failed_stage_id"] != "part-edit-1" ||
		countEvents(failed, "report.part.created") != 1 ||
		countEvents(failed, reporting.PartEditStartedEventType) != 1 ||
		countEvents(failed, reporting.PartEditedEventType) != 0 {
		t.Fatalf("initial run must fail after Part assembly before edit completion: failure=%#v events=%#v", failure, failed["events"])
	}
	originalPendingID := failure["pending_event_id"].(string)
	ancestorPart := eventRecord(t, failed, "report.part.created", 0)
	ancestorPartPayload := nestedMap(t, ancestorPart, "Payload")

	retry := postJSON(t, server.URL+"/api/missions/"+missionID+"/reports/retry", map[string]any{
		"failed_pending_event_id": originalPendingID, "strategy": "resume_failed", "retry_request_id": "retry-part-edit-resume",
	})
	resumePendingID := nestedString(t, retry, "pending_event", "EventID")
	detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
	for eventType, want := range map[string]int{
		"report.draft.pending":             2,
		"report.plan.created":              1,
		"report.section.created":           1,
		"report.part.created":              1,
		reporting.PartEditStartedEventType: 2,
		"report.part_edit.failed":          1,
		reporting.PartEditedEventType:      1,
		"report.draft.failed":              1,
		"report.artifact.created":          1,
	} {
		if got := countEvents(detail, eventType); got != want {
			t.Fatalf("%s count=%d, want %d: %#v", eventType, got, want, detail["events"])
		}
	}
	partEditRequests := partEditAgentRequests(delegate.snapshotRequests())
	if len(partEditRequests) != 2 {
		t.Fatalf("expected failed Part edit plus resumed Part edit, got %#v", partEditRequests)
	}
	resumed := partEditRequests[1].PartEdit
	if resumed.PendingEventID != resumePendingID ||
		resumed.SourcePartEventID != ancestorPart["EventID"] ||
		resumed.SourceArtifactID != ancestorPartPayload["artifact_id"] {
		t.Fatalf("resume_failed did not bind the accepted ancestor Part: binding=%#v ancestor=%#v", resumed, ancestorPart)
	}
	finalRequests := finalAgentRequests(delegate.snapshotRequests())
	editedPart := lastEventPayload(t, detail, reporting.PartEditedEventType)
	if len(finalRequests) != 1 || finalRequests[0].LongFormFinalize == nil ||
		len(finalRequests[0].LongFormFinalize.PartArtifactIDs) != 1 ||
		finalRequests[0].LongFormFinalize.PartArtifactIDs[0] != editedPart["artifact_id"] {
		t.Fatalf("resume_failed did not complete through the final editor using edited Part artifact: final=%#v edited=%#v", finalRequests, editedPart)
	}
}

func TestReportRetryRestartDoesNotReuseAcceptedAncestorPartEdit(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	server := NewServer(svc, Options{}).(*Server)

	const (
		missionID      = "mis_restart_part_edit"
		rootPendingID  = "evt_restart_root_pending"
		rootPlanID     = "evt_restart_root_plan"
		restartPlanID  = "evt_restart_plan"
		restartPending = "evt_restart_pending"
	)
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "restart part edit"}); err != nil {
		t.Fatal(err)
	}

	plan := narrativeContractTestPlan()
	rootPartArtifact := appendRetryPartEditAttempt(t, ctx, svc, missionID, rootPendingID, rootPlanID, "root", plan, true)
	failIDs := 0
	runner := reportexecution.Runner{Service: svc, NewID: func(prefix string) string {
		failIDs++
		return fmt.Sprintf("%s_restart_failed_%d", prefix, failIDs)
	}}
	if _, err := runner.AppendDraftFailed(ctx, missionID, rootPendingID, "codex", reportexecution.ModeLongForm, reportexecution.NewStageFailure("part_edit", rootPlanID, 1, 0, errors.New("part edit failed"))); err != nil {
		t.Fatal(err)
	}

	retry, err := svc.RequestReportRetry(ctx, app.ReportRetryRequest{
		EventID: restartPending, MissionID: missionID, FailedPendingEventID: rootPendingID,
		Strategy: "restart", RetryRequestID: "retry-part-edit-restart", Producer: app.Producer{Type: "user", ID: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	progress, err := server.loadSectionalReportProgress(ctx, missionID, retry.EventID)
	if err != nil {
		t.Fatal(err)
	}
	if progress.hasPlan || len(progress.parts) != 0 || len(progress.editedParts) != 0 {
		t.Fatalf("restart lineage reused ancestor state before the new attempt ran: progress=%#v ancestor=%#v", progress, rootPartArtifact)
	}

	restartPartArtifact := appendRetryPartEditAttempt(t, ctx, svc, missionID, retry.EventID, restartPlanID, "restart", plan, false)
	progress, err = server.loadSectionalReportProgress(ctx, missionID, retry.EventID)
	if err != nil {
		t.Fatal(err)
	}
	if !progress.hasPlan || len(progress.parts) != 1 || progress.parts[0].ArtifactID != restartPartArtifact.ArtifactID || len(progress.editedParts) != 0 {
		t.Fatalf("restart progress did not isolate the new attempt Part: progress=%#v ancestor=%#v restart=%#v", progress, rootPartArtifact, restartPartArtifact)
	}
	binding, err := server.partEditBinding(ctx, reportPartEditorRequest{
		title: "Reader Report", missionID: missionID, pendingEventID: retry.EventID, planEventID: restartPlanID,
		toolSessionID: "ses_restart_part_edit", previousSessionID: "restart-part-editor-session", executorName: "codex", mcpMode: "auto",
		editedArtifactID: "art_restart_part_edit", filename: "restart-part-edited.md",
		rigor: reportRigorProfiles["balanced"], plan: plan, part: plan.Parts[0], partIndex: 0,
		source:              sectionalReportPartDraft{Title: plan.Parts[0].Title, Markdown: string(restartPartArtifact.Content), ArtifactID: restartPartArtifact.ArtifactID, WordCount: reportWordCount(string(restartPartArtifact.Content))},
		reportSessionPolicy: reportSessionPolicySameSession, generationGuidanceProfile: reportprompt.ProfilePartConnectiveEconomyVoice,
		sessionChainKind: "same_session_report", reportPlanSessionID: "restart-plan-session",
	})
	if err != nil {
		t.Fatal(err)
	}
	if binding.SourceArtifactID != restartPartArtifact.ArtifactID || binding.SourcePartEventID != "evt_restart_part" ||
		binding.SourceArtifactID == rootPartArtifact.ArtifactID || binding.SourcePartEventID == "evt_root_part" {
		t.Fatalf("restart Part edit did not bind only the new Part: binding=%#v ancestor=%#v restart=%#v", binding, rootPartArtifact, restartPartArtifact)
	}
}

func TestReportRecoveryIgnoresMalformedPartEditOutcomeAndRerunsEditor(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	delegate := &partEditRecoveryExecutor{}
	executor := withReportPlanSubmissionFixture(svc, delegate)
	forker, ok := executor.(AgentSessionForker)
	if !ok {
		t.Fatal("fixture executor must fork Part editor sessions")
	}
	server := NewServer(svc, Options{}).(*Server)

	const (
		missionID = "mis_malformed_part_edit_recovery"
		pendingID = "evt_malformed_pending"
		planID    = "evt_malformed_plan"
	)
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "malformed part edit recovery"}); err != nil {
		t.Fatal(err)
	}
	plan := narrativeContractTestPlan()
	partArtifact := appendRetryPartEditAttempt(t, ctx, svc, missionID, pendingID, planID, "malformed", plan, true)
	fakeArtifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_malformed_part_edit", MissionID: missionID,
		MediaType: "text/markdown; charset=utf-8", Filename: "malformed-edited.md",
		Producer: app.Producer{Type: "agent_session", ID: "provider-fake"}, Content: []byte("# Core Part\n\nFake edited body.\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID: "evt_malformed_part_edit", MissionID: missionID, EventType: reporting.PartEditedEventType,
		Producer:         app.Producer{Type: "agent_session", ID: "provider-fake"},
		CausationEventID: "evt_malformed_part", CorrelationID: "wrong-part-edit-key",
		Payload: mustJSON(map[string]any{
			"kind":                         reporting.PartEditedKind,
			"pending_event_id":             pendingID,
			"plan_event_id":                planID,
			"source_part_event_id":         "evt_malformed_part",
			"source_artifact_id":           partArtifact.ArtifactID,
			"artifact_id":                  fakeArtifact.ArtifactID,
			"tool_session_id":              "ses_wrong_part_edit",
			"provider_session_id":          "provider-fake",
			"previous_provider_session_id": "provider-fake",
			"idempotency_key":              "wrong-part-edit-key",
			"part_index":                   1,
			"agent_executor":               "codex",
			"mcp_mode":                     "auto",
			"report_session_policy":        reportSessionPolicySameSession,
			"generation_guidance_profile":  reportprompt.ProfilePartConnectiveEconomyVoice,
			"session_chain_kind":           "same_session_report",
			"report_plan_session_id":       "malformed-plan-session",
			"fork_source_agent_session_id": "malformed-plan-session",
			"changed":                      true,
			"edited_word_count":            4,
		}),
	}); err != nil {
		t.Fatal(err)
	}

	progress, err := server.loadSectionalReportProgress(ctx, missionID, pendingID)
	if err != nil {
		t.Fatal(err)
	}
	if len(progress.editedParts) != 0 {
		t.Fatalf("malformed Part edit was recovered as completed: %#v", progress.editedParts)
	}
	parts := []sectionalReportPartDraft{{
		Title: plan.Parts[0].Title, Markdown: string(partArtifact.Content),
		ArtifactID: partArtifact.ArtifactID, WordCount: reportWordCount(string(partArtifact.Content)),
	}}
	previousSessionID, forkSourceID, err := legacyfinalize.ForkSession(ctx, forker, progress.reportPlanSessionID)
	if err != nil {
		t.Fatal(err)
	}
	if forkSourceID == "" {
		forkSourceID = progress.reportPlanSessionID
	}
	editInput := partEditInput(reportPartEditorRequest{
		title: "Reader Report", missionID: missionID, pendingEventID: pendingID, planEventID: progress.planEvent.EventID,
		toolSessionID: newID("ses"), previousSessionID: previousSessionID,
		editedArtifactID: newID("art"), filename: safeFilename("Reader Report part 01 edited", ".md"),
		executorName: "codex", mcpMode: "auto", rigor: reportRigorProfiles["balanced"],
		plan: plan, part: plan.Parts[0], partIndex: 0, source: parts[0],
		reportSessionPolicy: progress.reportSessionPolicy, reportSessionPolicySelection: progress.reportSessionPolicySelection,
		generationGuidanceProfile: reportprompt.ProfilePartConnectiveEconomyVoice,
		sessionChainKind:          progress.sessionChainKind, reportPlanSessionID: progress.reportPlanSessionID,
		forkSourceAgentSessionID: forkSourceID,
	}, false, "")
	out, err := (partedit.Runner{Service: svc, Executor: executor, NewID: newID}).Run(ctx, editInput)
	if err != nil {
		t.Fatal(err)
	}
	editedParts := []sectionalReportPartDraft{partEditDraft(out)}
	editedArtifactIDs := []string{out.Draft.ArtifactID}
	if len(editedParts) != 1 || len(editedArtifactIDs) != 1 || editedArtifactIDs[0] == fakeArtifact.ArtifactID {
		t.Fatalf("Part editor was skipped or reused malformed artifact: parts=%#v ids=%#v fake=%s", editedParts, editedArtifactIDs, fakeArtifact.ArtifactID)
	}
	requests := partEditAgentRequests(delegate.snapshotRequests())
	if len(requests) != 1 || requests[0].PartEdit == nil || requests[0].PartEdit.SourceArtifactID != partArtifact.ArtifactID {
		t.Fatalf("malformed recovery did not rerun the bound Part editor: %#v", requests)
	}
}

func TestReportRecoveryAcceptsIdempotentPartEditStartReplay(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	server := NewServer(svc, Options{}).(*Server)

	const (
		missionID = "mis_replayed_start_recovery"
		pendingID = "evt_replayed_start_pending"
		planID    = "evt_replayed_start_plan"
	)
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "replayed start recovery"}); err != nil {
		t.Fatal(err)
	}
	plan := narrativeContractTestPlan()
	partArtifact := appendRetryPartEditAttempt(t, ctx, svc, missionID, pendingID, planID, "replayed_start", plan, true)
	binding, err := server.partEditBinding(ctx, reportPartEditorRequest{
		title: "Reader Report", missionID: missionID, pendingEventID: pendingID, planEventID: planID,
		toolSessionID: "ses_replayed_start_part_edit", previousSessionID: "replayed-start-editor-session", executorName: "codex", mcpMode: "auto",
		editedArtifactID: "art_replayed_start_part_edit", filename: "replayed-start-part-edited.md",
		rigor: reportRigorProfiles["balanced"], plan: plan, part: plan.Parts[0], partIndex: 0,
		source: sectionalReportPartDraft{
			Title: plan.Parts[0].Title, Markdown: string(partArtifact.Content),
			ArtifactID: partArtifact.ArtifactID, WordCount: reportWordCount(string(partArtifact.Content)),
		},
		reportSessionPolicy: reportSessionPolicySameSession, generationGuidanceProfile: reportprompt.ProfilePartConnectiveEconomyVoice,
		sessionChainKind: "same_session_report", reportPlanSessionID: "replayed_start-plan-session",
		forkSourceAgentSessionID: "replayed_start-plan-session",
	})
	if err != nil {
		t.Fatal(err)
	}
	started, created, err := reporting.StartPartEdit(ctx, svc, "evt_replayed_start_part_edit_started", binding)
	if err != nil || !created {
		t.Fatalf("Part edit start created=%t event=%#v err=%v", created, started, err)
	}
	replayed, created, err := reporting.StartPartEdit(ctx, svc, "evt_replayed_start_part_edit_started_again", binding)
	if err != nil || created || replayed.EventID != started.EventID {
		t.Fatalf("Part edit start replay created=%t event=%#v err=%v", created, replayed, err)
	}
	edit, err := reporting.FinalizePartEdit(ctx, svc, binding, "evt_replayed_start_part_edit", "# Core Part\n\nRecovered edited body.\n", 1)
	if err != nil {
		t.Fatal(err)
	}
	progress, err := server.loadSectionalReportProgress(ctx, missionID, pendingID)
	if err != nil {
		t.Fatal(err)
	}
	recovered, ok := progress.editedParts[0]
	if !ok || recovered.ArtifactID != edit.Artifact.ArtifactID {
		t.Fatalf("replayed start was not recoverable: recovered=%#v edit=%#v progress=%#v", recovered, edit, progress)
	}
	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	if countLedgerEvents(events, reporting.PartEditStartedEventType) != 1 {
		t.Fatalf("exact start replay appended duplicate starts: %#v", events)
	}
}

func appendRetryPartEditAttempt(t *testing.T, ctx context.Context, svc *app.Service, missionID, pendingID, planID, label string, plan agentSectionalReportPlan, appendPending bool) app.RawArtifact {
	t.Helper()
	producer := app.Producer{Type: "agent_session", ID: label + "-provider-session"}
	if appendPending {
		if _, err := svc.AppendEvents(ctx, missionID, []app.AppendEventRequest{{
			EventID:   pendingID,
			MissionID: missionID,
			EventType: "report.draft.pending",
			Producer:  app.Producer{Type: "user", ID: "test"},
			Payload: mustJSON(map[string]any{
				"title":                       "Reader Report",
				"report_mode":                 reportModeLongForm,
				"agent_executor":              "codex",
				"mcp_mode":                    "auto",
				"generation_guidance_profile": reportprompt.ProfilePartConnectiveEconomyVoice,
			}),
		}}); err != nil {
			t.Fatal(err)
		}
	}
	planArtifact := createRetryMarkdownArtifact(t, ctx, svc, missionID, "art_"+label+"_plan", label+"-plan.md", fmt.Sprintf("# Plan\n\n%s attempt plan.\n", label))
	sectionArtifact := createRetryMarkdownArtifact(t, ctx, svc, missionID, "art_"+label+"_section", label+"-section.md", fmt.Sprintf("# Core Section\n\n%s Section body.\n", label))
	partArtifact := createRetryMarkdownArtifact(t, ctx, svc, missionID, "art_"+label+"_part", label+"-part.md", fmt.Sprintf("# Core Part\n\n%s assembled Part body.\n", label))
	if _, err := svc.AppendEvents(ctx, missionID, []app.AppendEventRequest{
		reporting.BuildMarkdownReportPlanCreatedAppendRequest(reporting.MarkdownReportPlanCreatedEventRequest{
			MarkdownReportEventBase: reporting.MarkdownReportEventBase{
				EventID: planID, MissionID: missionID, PendingEventID: pendingID, Title: "Reader Report",
				AgentExecutor: "codex", AgentSessionID: label + "-plan-session", MCPMode: "auto",
				ReportMode: reportModeLongForm, ReportSessionPolicy: reportSessionPolicySameSession,
				GenerationGuidanceProfile: reportprompt.ProfilePartConnectiveEconomyVoice,
				SessionChainKind:          "same_session_report", ReportPlanSessionID: label + "-plan-session",
				Text: agentReportAnyJSON(plan), Producer: producer,
			},
			ArtifactID: planArtifact.ArtifactID, Plan: plan, AssemblyStrategy: "narrative_contract_final_edit", PartEditEnabled: true,
		}),
		reporting.BuildMarkdownReportSectionCreatedAppendRequest(reporting.MarkdownReportSectionCreatedEventRequest{
			MarkdownReportStageEventBase: reporting.MarkdownReportStageEventBase{
				EventID: "evt_" + label + "_section", MissionID: missionID, PendingEventID: pendingID, PlanEventID: planID,
				Title: plan.Parts[0].Sections[0].Title, Artifact: sectionArtifact, AgentExecutor: "codex",
				AgentSessionID: label + "-plan-session", ReportMode: reportModeLongForm,
				ReportSessionPolicy: reportSessionPolicySameSession, GenerationGuidanceProfile: reportprompt.ProfilePartConnectiveEconomyVoice,
				SessionChainKind: "same_session_report", ReportPlanSessionID: label + "-plan-session", Producer: producer,
			},
			PartIndex: 1, SectionIndex: 1, WordCount: reportWordCount(string(sectionArtifact.Content)),
		}),
		reporting.BuildMarkdownReportPartCreatedAppendRequest(reporting.MarkdownReportPartCreatedEventRequest{
			MarkdownReportStageEventBase: reporting.MarkdownReportStageEventBase{
				EventID: "evt_" + label + "_part", MissionID: missionID, PendingEventID: pendingID, PlanEventID: planID,
				Title: plan.Parts[0].Title, Artifact: partArtifact, AgentExecutor: "codex",
				AgentSessionID: label + "-plan-session", ReportMode: reportModeLongForm,
				ReportSessionPolicy: reportSessionPolicySameSession, GenerationGuidanceProfile: reportprompt.ProfilePartConnectiveEconomyVoice,
				SessionChainKind: "same_session_report", ReportPlanSessionID: label + "-plan-session", Producer: producer,
			},
			PartIndex: 1, SectionCount: 1, WordCount: reportWordCount(string(partArtifact.Content)),
		}),
	}); err != nil {
		t.Fatal(err)
	}
	return partArtifact
}

func appendRetryPartPlanningAttempt(t *testing.T, ctx context.Context, svc *app.Service, missionID, pendingID, planID, label string, plan agentSectionalReportPlan, appendPending bool) string {
	t.Helper()
	reportPlanSessionID := label + "-report-plan-session"
	partOwnerSessionID := label + "-part-owner-session"
	requests := []app.AppendEventRequest{}
	if appendPending {
		requests = append(requests, app.AppendEventRequest{
			EventID:   pendingID,
			MissionID: missionID,
			EventType: "report.draft.pending",
			Producer:  app.Producer{Type: "user", ID: "test"},
			Payload: mustJSON(map[string]any{
				"title":                       "Reader Report",
				"report_mode":                 reportModeLongForm,
				"execution_strategy":          reportExecutionStrategySectionFanout,
				"agent_executor":              "codex",
				"mcp_mode":                    "auto",
				"generation_guidance_profile": reportprompt.ProfilePartConnectiveEconomyVoice,
			}),
		})
	}
	requests = append(requests,
		reporting.BuildMarkdownReportPlanCreatedAppendRequest(reporting.MarkdownReportPlanCreatedEventRequest{
			MarkdownReportEventBase: reporting.MarkdownReportEventBase{
				EventID: planID, MissionID: missionID, PendingEventID: pendingID, Title: "Reader Report",
				AgentExecutor: "codex", AgentSessionID: reportPlanSessionID, ReturnedAgentSessionID: reportPlanSessionID,
				ToolSessionID: "ses_" + label + "_plan", MCPMode: "auto", ReportMode: reportModeLongForm,
				ReportSessionPolicy: reportSessionPolicySameSession, ReportSessionPolicySelection: reportexecution.SessionPolicySelectionExplicitSameSession,
				GenerationGuidanceProfile: reportprompt.ProfilePartConnectiveEconomyVoice,
				SessionChainKind:          "section_fanout_report", ReportPlanSessionID: reportPlanSessionID,
				CompositionStrategy: "sectional_preserve_markdown", Text: "섹션 병렬 장문 Markdown 리포트 생성 계획을 만들었습니다.",
				Producer: app.Producer{Type: "agent_session", ID: reportPlanSessionID},
			},
			ArtifactID: "art_" + label + "_final", Plan: plan, AssemblyStrategy: "c4_normalized_section_headings",
			PartEditEnabled: true, PartPlanningEnabled: true,
		}),
		reporting.BuildPartPlanCreatedAppendRequest(reporting.PartPlanCreatedEventRequest{
			MarkdownReportStageEventBase: reporting.MarkdownReportStageEventBase{
				EventID: "evt_" + label + "_part_plan", MissionID: missionID, PendingEventID: pendingID, PlanEventID: planID,
				Title: plan.Parts[0].Title, AgentExecutor: "codex", AgentSessionID: partOwnerSessionID,
				PreviousAgentSessionID: partOwnerSessionID, ReturnedAgentSessionID: partOwnerSessionID,
				ToolSessionID: "ses_" + label + "_part_plan", ReportMode: reportModeLongForm,
				ReportSessionPolicy: reportSessionPolicySameSession, ReportSessionPolicySelection: reportexecution.SessionPolicySelectionExplicitSameSession,
				GenerationGuidanceProfile: reportprompt.ProfilePartConnectiveEconomyVoice,
				SessionChainKind:          "section_fanout_report", ReportPlanSessionID: reportPlanSessionID,
				ReportSessionID: partOwnerSessionID, ForkSourceAgentSessionID: reportPlanSessionID, CompositionStrategy: "sectional_preserve_markdown",
				AssemblyStrategy: "c4_normalized_section_headings", Producer: app.Producer{Type: "agent_session", ID: partOwnerSessionID},
			},
			PartIndex: 1,
			Brief:     label + " Part owner brief.",
		}),
	)
	if _, err := svc.AppendEvents(ctx, missionID, requests); err != nil {
		t.Fatal(err)
	}
	return partOwnerSessionID
}

func createRetryMarkdownArtifact(t *testing.T, ctx context.Context, svc *app.Service, missionID, artifactID, filename, markdown string) app.RawArtifact {
	t.Helper()
	artifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: artifactID, MissionID: missionID, MediaType: "text/markdown; charset=utf-8", Filename: filename,
		Producer: app.Producer{Type: "agent_session", ID: "test"}, Content: []byte(markdown),
	})
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}

func TestReportRetryResumeFailedReusesLongFormStagesAndFinalizes(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	delegate := &fakeAgentExecutor{responses: []AgentResult{
		{Text: agentReportAnyJSON(agentSectionalReportPlan{Summary: "Plan", Parts: []agentReportPart{{Title: "Part", Sections: []agentReportSection{{Title: "Section"}}}}}), SessionID: "ses-report"},
		{Text: "Section body.", SessionID: "ses-report"},
		{Text: `{"intro":"Intro","transitions":[],"closing":"Close"}`, SessionID: "ses-report"},
		{Text: "invalid final frame one", SessionID: "ses-report"},
		{Text: "invalid final frame two", SessionID: "ses-report"},
		{Text: `{"front_matter":"# Recovered report","closing":"## Close"}`, SessionID: "ses-report"},
	}}
	server := httptest.NewServer(NewServer(svc, Options{AgentExecutor: withReportPlanSubmissionFixture(svc, delegate)}))
	defer server.Close()
	mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "resume failed finalization"})
	missionID := nestedString(t, mission, "projection", "mission_id")
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
		"title": "Report", "report_mode": "long_form", "post_report_humanize": "disabled",
		"generation_guidance_profile": reportprompt.ProfileVisualPlan,
	})
	failed := waitForEventType(t, server.URL, missionID, "report.draft.failed")
	var originalPendingID string
	for _, raw := range failed["events"].([]any) {
		event := raw.(map[string]any)
		if event["EventType"] == "report.draft.failed" {
			originalPendingID = nestedString(t, event, "Payload", "pending_event_id")
		}
	}
	if originalPendingID == "" {
		t.Fatalf("failed pending id missing: %#v", failed["events"])
	}
	postJSON(t, server.URL+"/api/missions/"+missionID+"/reports/retry", map[string]any{
		"failed_pending_event_id": originalPendingID, "strategy": "resume_failed", "retry_request_id": "retry-final-only",
	})
	var detail map[string]any
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		detail = getJSON(t, server.URL+"/api/missions/"+missionID)
		if countEvents(detail, "report.artifact.created") == 1 {
			break
		}
		if countEvents(detail, "report.draft.failed") > 1 {
			t.Fatalf("resume_failed finalization failed: events=%#v", detail["events"])
		}
		time.Sleep(20 * time.Millisecond)
	}
	if countEvents(detail, "report.artifact.created") != 1 {
		t.Fatalf("resume_failed finalization timed out: events=%#v", detail["events"])
	}
	for eventType, want := range map[string]int{
		"report.draft.pending":    2,
		"report.plan.created":     1,
		"report.section.created":  1,
		"report.part.created":     1,
		"report.draft.failed":     1,
		"report.artifact.created": 1,
	} {
		if got := countEvents(detail, eventType); got != want {
			t.Fatalf("%s count=%d, want %d: %#v", eventType, got, want, detail["events"])
		}
	}
	if len(delegate.requests) != 7 || delegate.requests[6].LongFormFinalize == nil {
		t.Fatalf("resume_failed regenerated stages instead of final-only recovery: %#v", delegate.requests)
	}
}

func eventRecord(t *testing.T, detail map[string]any, eventType string, index int) map[string]any {
	t.Helper()
	records := eventRecords(t, detail, eventType)
	if index < 0 || index >= len(records) {
		t.Fatalf("event %s index %d missing in %#v", eventType, index, detail["events"])
	}
	return records[index]
}

func eventRecords(t *testing.T, detail map[string]any, eventType string) []map[string]any {
	t.Helper()
	records := []map[string]any{}
	for _, raw := range detail["events"].([]any) {
		event := raw.(map[string]any)
		if event["EventType"] == eventType {
			records = append(records, event)
		}
	}
	return records
}

func partPlanAgentRequests(requests []AgentRequest) []AgentRequest {
	matches := []AgentRequest{}
	for _, req := range requests {
		if strings.HasPrefix(req.UserText, "plan the reading flow for Part") {
			matches = append(matches, req)
		}
	}
	return matches
}

func sectionAgentRequests(requests []AgentRequest) []AgentRequest {
	matches := []AgentRequest{}
	for _, req := range requests {
		if strings.HasPrefix(req.UserText, "draft section ") {
			matches = append(matches, req)
		}
	}
	return matches
}

func partAssemblyAgentRequests(requests []AgentRequest) []AgentRequest {
	matches := []AgentRequest{}
	for _, req := range requests {
		if req.PartAssembly != nil {
			matches = append(matches, req)
		}
	}
	return matches
}

func partEditAgentRequests(requests []AgentRequest) []AgentRequest {
	matches := []AgentRequest{}
	for _, req := range requests {
		if req.PartEdit != nil {
			matches = append(matches, req)
		}
	}
	return matches
}

func finalAgentRequests(requests []AgentRequest) []AgentRequest {
	matches := []AgentRequest{}
	for _, req := range requests {
		if req.LongFormFinalize != nil {
			matches = append(matches, req)
		}
	}
	return matches
}
