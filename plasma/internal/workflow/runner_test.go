package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

type fakeAgent struct {
	responses []AgentResult
	errs      []error
	err       error
	requests  []AgentRequest
	onRun     func()
}

func (agent *fakeAgent) Run(_ context.Context, req AgentRequest) (AgentResult, error) {
	agent.requests = append(agent.requests, req)
	if agent.onRun != nil {
		agent.onRun()
		agent.onRun = nil
	}
	var err error
	if agent.err != nil {
		err = agent.err
	}
	if len(agent.responses) == 0 {
		result := AgentResult{Text: "done\n" + controlMarker + ` {"decision":"stop","reason":"done"}`, SessionID: req.PreviousSessionID, Resumed: req.PreviousSessionID != ""}
		if len(agent.errs) > 0 {
			err = agent.errs[0]
			agent.errs = agent.errs[1:]
		}
		return result, err
	}
	response := agent.responses[0]
	agent.responses = agent.responses[1:]
	if response.SessionID == "" {
		response.SessionID = req.PreviousSessionID
	}
	response.Resumed = req.PreviousSessionID != ""
	if len(agent.errs) > 0 {
		err = agent.errs[0]
		agent.errs = agent.errs[1:]
	}
	return response, err
}

func TestRunnerDefersUntilCurrentTurnTerminalExists(t *testing.T) {
	ctx := context.Background()
	svc := newWorkflowTestService(t)
	mission := createWorkflowMission(t, svc)
	appendRawEvent(t, svc, mission.MissionID, "evt_user_active", "turn.user", map[string]any{"kind": "user_turn", "text": "start later"})
	appendRawEvent(t, svc, mission.MissionID, "evt_pending_active", "turn.agent.pending", map[string]any{"user_event_id": "evt_user_active", "agent_executor": "codex"})
	requestWorkflow(t, svc, mission.MissionID, app.RequestWorkflowRunRequest{
		WorkflowRunID:     "wfr_deferred",
		StartAfterEventID: "evt_user_active",
		MaxSteps:          1,
	})

	agent := &fakeAgent{responses: []AgentResult{{Text: "later result\n" + controlMarker + ` {"decision":"stop","reason":"complete"}`, SessionID: "agent-session-1"}}}
	runner := testRunner(svc, agent)
	view, err := runner.Run(ctx, mission.MissionID, "wfr_deferred")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if view.Status != app.WorkflowStatusQueued || len(agent.requests) != 0 {
		t.Fatalf("expected queued deferred run without agent call, got view=%#v requests=%d", view, len(agent.requests))
	}

	appendRawEvent(t, svc, mission.MissionID, "evt_response_active", "turn.agent.response", map[string]any{
		"kind":             "agent_response",
		"user_event_id":    "evt_user_active",
		"agent_executor":   "codex",
		"agent_session_id": "agent-session-1",
	})
	view, err = runner.Run(ctx, mission.MissionID, "wfr_deferred")
	if err != nil {
		t.Fatalf("Run after terminal returned error: %v", err)
	}
	if view.Status != app.WorkflowStatusCompleted || len(agent.requests) != 1 {
		t.Fatalf("expected completed drained run, got view=%#v requests=%d", view, len(agent.requests))
	}
}

func TestRunnerResumesLatestProviderSession(t *testing.T) {
	ctx := context.Background()
	svc := newWorkflowTestService(t)
	mission := createWorkflowMission(t, svc)
	appendRawEvent(t, svc, mission.MissionID, "evt_user_1", "turn.user", map[string]any{"kind": "user_turn", "text": "hello"})
	appendRawEvent(t, svc, mission.MissionID, "evt_agent_1", "turn.agent.response", map[string]any{
		"kind":             "agent_response",
		"user_event_id":    "evt_user_1",
		"agent_executor":   "codex",
		"agent_session_id": "agent-session-1",
	})
	requestWorkflow(t, svc, mission.MissionID, app.RequestWorkflowRunRequest{WorkflowRunID: "wfr_resume", MaxSteps: 1})

	agent := &fakeAgent{responses: []AgentResult{{Text: "resumed step\n" + controlMarker + ` {"decision":"stop","reason":"done"}`, SessionID: "agent-session-1"}}}
	view, err := testRunner(svc, agent).Run(ctx, mission.MissionID, "wfr_resume")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if view.Status != app.WorkflowStatusCompleted {
		t.Fatalf("expected completed run, got %#v", view)
	}
	if got := agent.requests[0].PreviousSessionID; got != "agent-session-1" {
		t.Fatalf("expected same provider session resume, got %q", got)
	}
}

func TestRunnerStopsBeforeNextStepAfterStopRequest(t *testing.T) {
	ctx := context.Background()
	svc := newWorkflowTestService(t)
	mission := createWorkflowMission(t, svc)
	requestWorkflow(t, svc, mission.MissionID, app.RequestWorkflowRunRequest{WorkflowRunID: "wfr_stop", MaxSteps: 3})

	agent := &fakeAgent{responses: []AgentResult{{Text: "first step\n" + controlMarker + ` {"decision":"continue","reason":"more"}`, SessionID: "agent-session-1"}}}
	agent.onRun = func() {
		if _, err := svc.RequestWorkflowStop(ctx, app.RequestWorkflowStopRequest{
			WorkflowRunID:      "wfr_stop",
			MissionID:          mission.MissionID,
			RequestedBySurface: app.WorkflowSurfaceWeb,
			Reason:             "user stop",
		}); err != nil {
			t.Fatalf("RequestWorkflowStop returned error: %v", err)
		}
	}
	view, err := testRunner(svc, agent).Run(ctx, mission.MissionID, "wfr_stop")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if view.Status != app.WorkflowStatusStopped || view.CompletedStepCount != 1 || len(agent.requests) != 1 {
		t.Fatalf("expected one step then stopped, got view=%#v requests=%d", view, len(agent.requests))
	}
}

func TestRunnerSkipsSourceRemovedDuringWorkflowOnNextStep(t *testing.T) {
	ctx := context.Background()
	svc := newWorkflowTestService(t)
	mission := createWorkflowMission(t, svc)
	artifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_workflow_source",
		MissionID:  mission.MissionID,
		MediaType:  "text/plain; charset=utf-8",
		Filename:   "source.txt",
		Producer:   app.Producer{Type: "user", ID: "test"},
		Content:    []byte("source body"),
	})
	if err != nil {
		t.Fatalf("CreateRawArtifact returned error: %v", err)
	}
	source, err := svc.CreateSourceSnapshot(ctx, app.CreateSourceSnapshotRequest{
		SnapshotID: "src_workflow_source",
		MissionID:  mission.MissionID,
		Connector: app.ConnectorRef{
			ConnectorID:      "manual",
			ConnectorType:    "manual",
			ExternalSourceID: "source.txt",
		},
		Title:       "Workflow source",
		ArtifactIDs: []string{artifact.ArtifactID},
		ContentHash: app.ContentHash{Algorithm: "sha256", Value: artifact.SHA256},
		Access:      app.SourceAccess{RetrievalPolicy: app.SourceRetrievalPolicySnapshotOnly},
	})
	if err != nil {
		t.Fatalf("CreateSourceSnapshot returned error: %v", err)
	}
	requestWorkflow(t, svc, mission.MissionID, app.RequestWorkflowRunRequest{WorkflowRunID: "wfr_removed_source", MaxSteps: 2})

	agent := &fakeAgent{responses: []AgentResult{
		{Text: "first step\n" + controlMarker + ` {"decision":"continue","reason":"source changed","next_instruction":"continue without removed source"}`, SessionID: "agent-session-1"},
		{Text: "second step\n" + controlMarker + ` {"decision":"stop","reason":"done"}`, SessionID: "agent-session-1"},
	}}
	agent.onRun = func() {
		if _, err := svc.RemoveSource(ctx, app.RemoveSourceRequest{
			MissionID:  mission.MissionID,
			SnapshotID: source.SnapshotID,
			Reason:     "removed during workflow",
			Producer:   app.Producer{Type: "user", ID: "test"},
		}); err != nil {
			t.Fatalf("RemoveSource returned error: %v", err)
		}
	}
	view, err := testRunner(svc, agent).Run(ctx, mission.MissionID, "wfr_removed_source")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if view.Status != app.WorkflowStatusCompleted || len(agent.requests) != 2 {
		t.Fatalf("expected two-step completed workflow, got view=%#v requests=%d", view, len(agent.requests))
	}
	events, err := svc.ListEvents(ctx, mission.MissionID)
	if err != nil {
		t.Fatalf("ListEvents returned error: %v", err)
	}
	var payload app.WorkflowSourceSkippedPayload
	for _, event := range events {
		if event.EventType != app.WorkflowSourceSkippedEvent {
			continue
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatalf("decode skip payload: %v", err)
		}
	}
	if payload.SnapshotID != source.SnapshotID || payload.Reason != "source_removed" || payload.WorkflowStepID != "wfs_2" || payload.RemovedEventID == "" {
		t.Fatalf("unexpected source skip payload: %#v", payload)
	}
	if countEvents(events, app.WorkflowSourceSkippedEvent) != 1 {
		t.Fatalf("expected one source skip event, got %#v", events)
	}
}

func TestRunnerPausesAtMaxStepsWhenAgentWantsToContinue(t *testing.T) {
	ctx := context.Background()
	svc := newWorkflowTestService(t)
	mission := createWorkflowMission(t, svc)
	requestWorkflow(t, svc, mission.MissionID, app.RequestWorkflowRunRequest{WorkflowRunID: "wfr_max", MaxSteps: 1})

	agent := &fakeAgent{responses: []AgentResult{{Text: "step result\n" + controlMarker + ` {"decision":"continue","reason":"could continue","next_instruction":"read primary source"}`, SessionID: "agent-session-1"}}}
	view, err := testRunner(svc, agent).Run(ctx, mission.MissionID, "wfr_max")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if view.Status != app.WorkflowStatusPaused || view.StopReason != "max_steps reached" || view.CompletedStepCount != 1 {
		t.Fatalf("expected max-step pause, got %#v", view)
	}
	if view.ContinuationInstruction != "read primary source" {
		t.Fatalf("expected continuation instruction, got %#v", view)
	}
}

func TestRunnerPausesAtMaxDurationWhenAgentWantsToContinue(t *testing.T) {
	ctx := context.Background()
	svc := newWorkflowTestService(t)
	mission := createWorkflowMission(t, svc)
	requestWorkflow(t, svc, mission.MissionID, app.RequestWorkflowRunRequest{
		WorkflowRunID: "wfr_duration",
		MaxSteps:      3,
		MaxDurationMS: 1000,
	})

	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	agent := &fakeAgent{responses: []AgentResult{{Text: "step result\n" + controlMarker + ` {"decision":"continue","reason":"could continue","next_instruction":"read primary source"}`, SessionID: "agent-session-1"}}}
	agent.onRun = func() {
		now = now.Add(2 * time.Second)
	}
	runner := testRunner(svc, agent)
	runner.Now = func() time.Time { return now }
	view, err := runner.Run(ctx, mission.MissionID, "wfr_duration")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if view.Status != app.WorkflowStatusPaused || view.StopReason != "max_duration reached" || view.CompletedStepCount != 1 {
		t.Fatalf("expected max-duration pause, got %#v", view)
	}
	if view.ContinuationInstruction != "read primary source" {
		t.Fatalf("expected continuation instruction, got %#v", view)
	}
}

func TestRunnerProposesExplicitSourceCandidatesFromWorkflowResponse(t *testing.T) {
	ctx := context.Background()
	svc := newWorkflowTestService(t)
	mission := createWorkflowMission(t, svc)
	requestWorkflow(t, svc, mission.MissionID, app.RequestWorkflowRunRequest{WorkflowRunID: "wfr_candidates", MaxSteps: 1})

	response := strings.Join([]string{
		"새 원자료 후보를 찾았습니다.",
		"소스 후보: https://Example.com/report#section",
		"채택 의견: 이 자료는 사건별 원문 대조에 필요한 원자료 후보입니다.",
		controlMarker + ` {"decision":"stop","reason":"done"}`,
	}, "\n")
	agent := &fakeAgent{responses: []AgentResult{{Text: response, SessionID: "agent-session-1"}}}
	view, err := testRunner(svc, agent).Run(ctx, mission.MissionID, "wfr_candidates")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if view.Status != app.WorkflowStatusCompleted {
		t.Fatalf("expected completed run, got %#v", view)
	}
	events, err := svc.ListEvents(ctx, mission.MissionID)
	if err != nil {
		t.Fatalf("ListEvents returned error: %v", err)
	}
	var payload struct {
		AgentEventID string            `json:"agent_event_id"`
		Candidates   []sourceCandidate `json:"candidates"`
	}
	for _, event := range events {
		if event.EventType != "source.candidate.proposed" {
			continue
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatalf("candidate payload is invalid: %v", err)
		}
	}
	if len(payload.Candidates) != 1 {
		t.Fatalf("expected one source candidate, got %#v", payload)
	}
	if payload.AgentEventID == "" || payload.Candidates[0].URL != "https://example.com/report" {
		t.Fatalf("unexpected source candidate payload: %#v", payload)
	}
	if payload.Candidates[0].Reason == "" {
		t.Fatalf("expected source candidate reason: %#v", payload)
	}
}

func TestWorkflowSourceCandidateExtractionRequiresExplicitLabelAndOpinion(t *testing.T) {
	for _, text := range []string{
		"참고: https://example.com/report",
		"참고: https://example.com/report\n채택 의견: 좋아 보입니다.",
		"소스 후보: https://example.com/report",
	} {
		if got := sourceCandidatesFromText(text); len(got) != 0 {
			t.Fatalf("expected no candidates from %q, got %#v", text, got)
		}
	}
	got := sourceCandidatesFromText("소스 후보: https://example.com/report\n채택 의견: 원문 대조에 필요합니다.")
	if len(got) != 1 || got[0].URL != "https://example.com/report" {
		t.Fatalf("expected explicit source candidate, got %#v", got)
	}
	got = sourceCandidatesFromText("소스 후보: https://example.com/a\n소스 후보: https://example.com/b\n채택 의견: 두 번째 후보의 이유입니다.")
	if len(got) != 1 || got[0].URL != "https://example.com/b" {
		t.Fatalf("expected reason not to cross source candidate boundary, got %#v", got)
	}
}

func TestRunnerFailsOnInvalidControlDecision(t *testing.T) {
	ctx := context.Background()
	svc := newWorkflowTestService(t)
	mission := createWorkflowMission(t, svc)
	requestWorkflow(t, svc, mission.MissionID, app.RequestWorkflowRunRequest{WorkflowRunID: "wfr_invalid", MaxSteps: 2})

	agent := &fakeAgent{responses: []AgentResult{{Text: "visible result without control", SessionID: "agent-session-1"}}}
	view, err := testRunner(svc, agent).Run(ctx, mission.MissionID, "wfr_invalid")
	if err != nil {
		t.Fatalf("Run should record failure instead of returning error, got %v", err)
	}
	if view.Status != app.WorkflowStatusFailed || !strings.Contains(view.StopReason, "workflow step failed") {
		t.Fatalf("expected failed run, got %#v", view)
	}
	events, err := svc.ListEvents(ctx, mission.MissionID)
	if err != nil {
		t.Fatalf("ListEvents returned error: %v", err)
	}
	if countEvents(events, "turn.agent.response") != 1 || countEvents(events, app.WorkflowRunFailedEvent) != 1 {
		t.Fatalf("expected one saved result and one failure event, got %#v", events)
	}
}

func TestRunnerAutoCompactsAndRetriesWhenContextWindowIsFull(t *testing.T) {
	ctx := context.Background()
	svc := newWorkflowTestService(t)
	mission := createWorkflowMission(t, svc)
	appendRawEvent(t, svc, mission.MissionID, "evt_user_1", "turn.user", map[string]any{"kind": "user_turn", "text": "hello"})
	appendRawEvent(t, svc, mission.MissionID, "evt_agent_1", "turn.agent.response", map[string]any{
		"kind":             "agent_response",
		"user_event_id":    "evt_user_1",
		"agent_executor":   "codex",
		"agent_session_id": "agent-session-1",
	})
	requestWorkflow(t, svc, mission.MissionID, app.RequestWorkflowRunRequest{WorkflowRunID: "wfr_compact", MaxSteps: 1})

	agent := &fakeAgent{
		responses: []AgentResult{
			{Log: "ERROR: Codex ran out of room in the model's context window. Start a new thread or clear earlier history before retrying.", SessionID: "agent-session-1"},
			{Text: "compact summary", SessionID: "agent-session-1"},
			{Text: "retry result\n" + controlMarker + ` {"decision":"stop","reason":"done"}`, SessionID: "agent-session-1"},
		},
		errs: []error{errors.New("agent command failed"), nil, nil},
	}
	view, err := testRunner(svc, agent).Run(ctx, mission.MissionID, "wfr_compact")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if view.Status != app.WorkflowStatusCompleted || len(agent.requests) != 3 {
		t.Fatalf("expected compact retry completed workflow, got view=%#v requests=%d", view, len(agent.requests))
	}
	if !agent.requests[1].Compaction || agent.requests[1].PreviousSessionID != "agent-session-1" {
		t.Fatalf("expected second request to compact same session, got %#v", agent.requests[1])
	}
	if agent.requests[2].Compaction || agent.requests[2].PreviousSessionID != "agent-session-1" {
		t.Fatalf("expected third request to retry same session, got %#v", agent.requests[2])
	}
	events, err := svc.ListEvents(ctx, mission.MissionID)
	if err != nil {
		t.Fatalf("ListEvents returned error: %v", err)
	}
	if countEvents(events, "turn.agent.compacted") != 1 {
		t.Fatalf("expected one compaction event, got %#v", events)
	}
	var responsePayload map[string]any
	for _, event := range events {
		if event.EventType != "turn.agent.response" {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload["kind"] == "agent_response" {
			responsePayload = payload
		}
	}
	if responsePayload["retry_after_compacted"] != true || responsePayload["compaction_attempted"] != true {
		t.Fatalf("expected retry metadata after compaction, got %#v", responsePayload)
	}
}

func TestRunnerFailsOnDifferentReturnedSession(t *testing.T) {
	ctx := context.Background()
	svc := newWorkflowTestService(t)
	mission := createWorkflowMission(t, svc)
	appendRawEvent(t, svc, mission.MissionID, "evt_user_1", "turn.user", map[string]any{"kind": "user_turn", "text": "hello"})
	appendRawEvent(t, svc, mission.MissionID, "evt_agent_1", "turn.agent.response", map[string]any{
		"kind":             "agent_response",
		"user_event_id":    "evt_user_1",
		"agent_executor":   "codex",
		"agent_session_id": "agent-session-1",
	})
	requestWorkflow(t, svc, mission.MissionID, app.RequestWorkflowRunRequest{WorkflowRunID: "wfr_session", MaxSteps: 1})

	agent := &fakeAgent{responses: []AgentResult{{Text: "bad session\n" + controlMarker + ` {"decision":"stop","reason":"done"}`, SessionID: "agent-session-2"}}}
	view, err := testRunner(svc, agent).Run(ctx, mission.MissionID, "wfr_session")
	if err != nil {
		t.Fatalf("Run should record same-session failure instead of returning error, got %v", err)
	}
	if view.Status != app.WorkflowStatusFailed {
		t.Fatalf("expected same-session failure, got %#v", view)
	}
}

func TestLatestAgentSessionIDIncludesReportArtifactCreated(t *testing.T) {
	events := []app.LedgerEvent{{
		EventType: "report.artifact.created",
		Payload:   json.RawMessage(`{"agent_executor":"codex","agent_session_id":"report-session-1"}`),
	}}
	if got := LatestAgentSessionID(events, "codex"); got != "report-session-1" {
		t.Fatalf("expected report artifact session, got %q", got)
	}
}

func TestLatestAgentSessionIDKeepsPreReportResearchSessionForIsolatedReport(t *testing.T) {
	events := []app.LedgerEvent{{
		EventType: "turn.agent.response",
		Payload:   json.RawMessage(`{"kind":"agent_response","agent_executor":"codex","agent_session_id":"research-session-1"}`),
	}, {
		EventType: "report.artifact.created",
		Payload:   json.RawMessage(`{"agent_executor":"codex","agent_session_id":"report-session-1","report_session_policy":"isolated_fork","pre_report_research_session_id":"research-session-1"}`),
	}}
	if got := LatestAgentSessionID(events, "codex"); got != "research-session-1" {
		t.Fatalf("expected isolated report to preserve research session, got %q", got)
	}
}

func TestStepPromptUsesLayeredShapeForLegacyCurrentMode(t *testing.T) {
	prompt := StepPrompt(app.WorkflowRunView{
		MissionID:          "mis_1",
		UserInstructionRaw: "다각도로 조사",
		RunGoal:            "여러 가능성을 열어둔 조사",
		Instruction:        "첫 자료를 확인",
	}, "Investigate one thing", "ses_1", true)
	for _, expected := range []string{
		"Continue the existing Plasma research agent session",
		"one bounded workflow step",
		"User's original autonomous-run request",
		"다각도로 조사",
		"Derived autonomous-run goal",
		"여러 가능성을 열어둔 조사",
		"Instruction for this step",
		"Investigate one thing",
		"outranks the derived goal",
		"Do not let the derived goal close off possibilities",
		"Your answer is a result, not a source",
		"Use decision \"continue\" when the current step is complete",
		"Do not use stop merely because the current step instruction is complete",
		"소스 후보:",
		"채택 의견:",
		"PLASMA_WORKFLOW_CONTROL",
		"plasma.research.outline",
		"plasma.research.read",
		"plasma.sources.read",
		"live_reference local_path",
		"source.observed",
		"observation_event_id",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected prompt to contain %q:\n%s", expected, prompt)
		}
	}
	for _, forbidden := range []string{
		"plasma.evidence.propose",
		"plasma.claims.propose",
		"claim confidence",
		"report AST",
		"full source bodies",
		"full transcripts",
		`Use decision "stop" when the mission has no useful next workflow step or this instruction is complete`,
	} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("workflow prompt contains forbidden text %q:\n%s", forbidden, prompt)
		}
	}
}

func TestStepPromptLayeredModeKeepsRawGoalAndStepBoundary(t *testing.T) {
	prompt := StepPrompt(app.WorkflowRunView{
		MissionID:           "mis_1",
		StepInstructionMode: app.WorkflowStepInstructionModeLayered,
		UserInstructionRaw:  "다각도로 조사",
		RunGoal:             "여러 가능성을 열어둔 조사",
		Instruction:         "첫 자료를 확인",
	}, "Investigate one thing", "ses_1", true)
	for _, expected := range []string{
		"Continue the existing Plasma research agent session",
		"User's original autonomous-run request",
		"다각도로 조사",
		"Derived autonomous-run goal",
		"여러 가능성을 열어둔 조사",
		"Instruction for this step",
		"Investigate one thing",
		"outranks the derived goal",
		"Do not let the derived goal close off possibilities",
		"Use decision \"continue\" when the current step is complete",
		"Do not use stop merely because the current step instruction is complete",
		"PLASMA_WORKFLOW_CONTROL",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected layered prompt to contain %q:\n%s", expected, prompt)
		}
	}
}

func newWorkflowTestService(t *testing.T) *app.Service {
	t.Helper()
	store, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("Close returned error: %v", err)
		}
	})
	return app.NewService(store)
}

func createWorkflowMission(t *testing.T, svc *app.Service) app.Mission {
	t.Helper()
	mission, err := svc.CreateMission(context.Background(), app.CreateMissionRequest{MissionID: "mis_workflow", Title: "Workflow mission"})
	if err != nil {
		t.Fatalf("CreateMission returned error: %v", err)
	}
	return mission
}

func requestWorkflow(t *testing.T, svc *app.Service, missionID string, req app.RequestWorkflowRunRequest) app.WorkflowRunView {
	t.Helper()
	req.MissionID = missionID
	req.RequestedBySurface = app.WorkflowSurfaceWeb
	req.AgentExecutor = "codex"
	req.MCPMode = "auto"
	if req.Instruction == "" {
		req.Instruction = "Make bounded progress."
	}
	if req.MaxDurationMS == 0 {
		req.MaxDurationMS = 60000
	}
	view, err := svc.RequestWorkflowRun(context.Background(), req)
	if err != nil {
		t.Fatalf("RequestWorkflowRun returned error: %v", err)
	}
	return view
}

func testRunner(svc *app.Service, agent *fakeAgent) Runner {
	counters := map[string]int{}
	return Runner{
		Service: svc,
		Agent:   agent,
		NewID: func(prefix string) string {
			counters[prefix]++
			return prefix + "_" + string(rune('0'+counters[prefix]))
		},
	}
}

func appendRawEvent(t *testing.T, svc *app.Service, missionID string, eventID string, eventType string, payload map[string]any) {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	if _, err := svc.AppendEvent(context.Background(), app.AppendEventRequest{
		EventID:   eventID,
		MissionID: missionID,
		EventType: eventType,
		Producer:  app.Producer{Type: "test", ID: "test"},
		Payload:   encoded,
	}); err != nil {
		t.Fatalf("AppendEvent %s returned error: %v", eventID, err)
	}
}

func countEvents(events []app.LedgerEvent, eventType string) int {
	count := 0
	for _, event := range events {
		if event.EventType == eventType {
			count++
		}
	}
	return count
}
