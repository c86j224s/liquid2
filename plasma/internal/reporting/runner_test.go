package reporting

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestLogTerminalWriteFailureUsesSafeStructuredFields(t *testing.T) {
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	t.Cleanup(func() { log.SetOutput(previous) })
	logTerminalWriteFailure("mis_1", "evt_pending", "draft", "report.draft.failed", errors.New("sqlite busy"))
	line := output.String()
	for _, want := range []string{"report_terminal_write_failed", `mission_id="mis_1"`, `pending_event_id="evt_pending"`, `report_type="draft"`, `intended_event_type="report.draft.failed"`, `err="sqlite busy"`} {
		if !strings.Contains(line, want) {
			t.Fatalf("missing safe structured log field %q: %s", want, line)
		}
	}
}

func TestRunnerStartDraftUsesSharedPendingAndFailurePolicy(t *testing.T) {
	ctx := context.Background()
	svc := &fakeRunnerService{}
	inFlight := &InFlight{}
	inFlight.SetNewID(testRunnerID)
	done := make(chan struct{})
	runner := Runner{
		Service:  svc,
		InFlight: inFlight,
		NewID:    testRunnerID,
		GenerateDraft: func(context.Context, string, DraftRequest, string) error {
			close(done)
			return errors.New("agent failed")
		},
	}

	pending, err := runner.StartDraft(ctx, "mis_1", DraftRequest{Title: "Report", AgentExecutor: "codex", MCPMode: "auto"}, app.Producer{Type: "user", ID: "test"})
	if err != nil {
		t.Fatalf("StartDraft returned error: %v", err)
	}
	if pending.EventType != "report.draft.pending" {
		t.Fatalf("expected report.draft.pending, got %#v", pending)
	}
	if mode := runnerPayloadString(t, pending, "report_mode"); mode != ModePlanned {
		t.Fatalf("expected planned default mode, got %q", mode)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runner did not call shared draft generator")
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if countRunnerEvents(svc.snapshot(), "report.draft.failed") == 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected shared runner failure event, got %#v", svc.snapshot())
}

func TestRunnerStartDraftFreezesConfluenceSourceContextOutsideDraftRequest(t *testing.T) {
	checkedAt := time.Date(2026, 7, 14, 1, 2, 3, 0, time.UTC)
	svc := &fakeRunnerService{sources: []app.SourceSnapshot{
		{
			SnapshotID: "src_2", MissionID: "mis_1", Title: "Unchecked page",
			Connector:  app.ConnectorRef{ConnectorType: app.ConfluenceConnectorType, ExternalVersion: "4", ExternalURI: "https://private.example/wiki/2"},
			CapturedAt: checkedAt.Add(-2 * time.Hour), ExternalUpdatedAt: checkedAt.Add(-3 * time.Hour),
		},
		{
			SnapshotID: "src_1", MissionID: "mis_1", Title: "Current page",
			Connector:  app.ConnectorRef{ConnectorType: app.ConfluenceConnectorType, ExternalVersion: "7"},
			CapturedAt: checkedAt.Add(-time.Hour), ExternalUpdatedAt: checkedAt.Add(-90 * time.Minute),
			State: app.SourceState{ConfluenceUpdate: &app.ConfluenceUpdateState{
				Status: app.ConfluenceUpdateStatusAvailable, CheckedAt: checkedAt, CurrentVersion: 7, LatestVersion: 8,
			}},
		},
		{SnapshotID: "src_local", MissionID: "mis_1", Connector: app.ConnectorRef{ConnectorType: app.SourceConnectorTypeLocalPath}},
		{SnapshotID: "src_removed", MissionID: "mis_1", Connector: app.ConnectorRef{ConnectorType: app.ConfluenceConnectorType}, State: app.SourceState{Removed: true}},
	}}
	generated := make(chan DraftRequest, 1)
	runner := Runner{
		Service: svc, InFlight: &InFlight{}, NewID: testRunnerID,
		GenerateDraft: func(_ context.Context, _ string, req DraftRequest, _ string) error {
			generated <- req
			return nil
		},
	}
	runner.InFlight.SetNewID(testRunnerID)
	pending, err := runner.StartDraft(context.Background(), "mis_1", DraftRequest{Title: "Report"}, app.Producer{Type: "user", ID: "test"})
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		SourceContext reportSourceContext `json:"source_context"`
	}
	if err := json.Unmarshal(pending.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	context := payload.SourceContext
	if context.SchemaVersion != reportSourceContextSchemaVersion || context.CapturedAt == "" || len(context.ConfluenceSources) != 2 {
		t.Fatalf("unexpected report source context: %#v", context)
	}
	if context.ConfluenceSources[0].SnapshotID != "src_1" || context.ConfluenceSources[0].LastCheck.Status != app.ConfluenceUpdateStatusAvailable || context.ConfluenceSources[0].LastCheck.LatestVersion != 8 {
		t.Fatalf("checked source context changed: %#v", context.ConfluenceSources[0])
	}
	if context.ConfluenceSources[1].SnapshotID != "src_2" || context.ConfluenceSources[1].LastCheck.Status != "not_checked" {
		t.Fatalf("unchecked source context changed: %#v", context.ConfluenceSources[1])
	}
	for _, forbidden := range []string{"private.example", "external_uri", "artifact_ids", "locators", "content_hash"} {
		if strings.Contains(strings.ToLower(string(pending.Payload)), forbidden) {
			t.Fatalf("report source context leaked %q: %s", forbidden, pending.Payload)
		}
	}
	select {
	case req := <-generated:
		if req.Title != "Report" {
			t.Fatalf("source capture changed draft request: %#v", req)
		}
	case <-time.After(time.Second):
		t.Fatal("draft generator was not called")
	}
}

func TestReportSourceContextDropsUnknownConfluenceErrorDetails(t *testing.T) {
	check := buildReportConfluenceCheckContext(&app.ConfluenceUpdateState{
		Status: app.ConfluenceUpdateStatusFailed, CheckedAt: time.Now().UTC(),
		ErrorCategory: app.ConfluenceErrorCategoryAuth, ErrorCode: "raw_provider_detail",
	})
	if check.Status != app.ConfluenceUpdateStatusFailed || check.ErrorCategory != "" || check.ErrorCode != "" {
		t.Fatalf("unsafe error detail entered report context: %#v", check)
	}
}

func TestRunnerGenerationCallbacksDoNotInheritWorkflowStepDeadline(t *testing.T) {
	tests := []struct {
		name  string
		start func(context.Context, Runner) (app.LedgerEvent, error)
	}{
		{name: "draft", start: func(ctx context.Context, r Runner) (app.LedgerEvent, error) {
			return r.StartDraft(ctx, "mis_draft", DraftRequest{}, app.Producer{Type: "user", ID: "test"})
		}},
		{name: "design", start: func(ctx context.Context, r Runner) (app.LedgerEvent, error) {
			return r.StartDesign(ctx, "mis_design", DesignRequest{}, app.Producer{Type: "user", ID: "test"})
		}},
		{name: "humanize", start: func(ctx context.Context, r Runner) (app.LedgerEvent, error) {
			return r.StartHumanize(ctx, "mis_humanize", HumanizeRequest{}, app.Producer{Type: "user", ID: "test"})
		}},
		{name: "patch", start: func(ctx context.Context, r Runner) (app.LedgerEvent, error) {
			return r.StartPatch(ctx, "mis_patch", PatchRequest{}, app.Producer{Type: "user", ID: "test"})
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			called := make(chan bool, 1)
			observe := func(ctx context.Context) error {
				_, hasDeadline := ctx.Deadline()
				called <- hasDeadline
				return nil
			}
			runner := Runner{
				Service:  &fakeRunnerService{},
				InFlight: &InFlight{},
				NewID:    testRunnerID,
				GenerateDraft: func(ctx context.Context, _ string, _ DraftRequest, _ string) error {
					return observe(ctx)
				},
				GenerateDesign: func(ctx context.Context, _ string, _ DesignRequest, _ string) error {
					return observe(ctx)
				},
				GenerateHumanize: func(ctx context.Context, _ string, _ HumanizeRequest, _ string) error {
					return observe(ctx)
				},
				GeneratePatch: func(ctx context.Context, _ string, _ PatchRequest, _ string) error {
					return observe(ctx)
				},
			}
			runner.InFlight.SetNewID(testRunnerID)
			workflowStepCtx, cancel := context.WithTimeout(context.Background(), time.Hour)
			defer cancel()
			if _, err := tc.start(workflowStepCtx, runner); err != nil {
				t.Fatal(err)
			}
			select {
			case hasDeadline := <-called:
				if hasDeadline {
					t.Fatal("report generation callback unexpectedly received a deadline")
				}
			case <-time.After(time.Second):
				t.Fatal("report generation callback was not called")
			}
		})
	}
}

func TestDraftDirectionPendingIsOptionalAndRecoverable(t *testing.T) {
	legacy, err := DraftRequestFromPendingEvent(app.LedgerEvent{Payload: json.RawMessage(`{"title":"Old"}`)})
	if err != nil || legacy.DirectionHint != "" {
		t.Fatalf("legacy recovery = %#v, %v", legacy, err)
	}
	recovered, err := DraftRequestFromPendingEvent(app.LedgerEvent{Payload: json.RawMessage(`{"title":"New","direction_hint":"  focus here  "}`)})
	if err != nil || recovered.DirectionHint != "focus here" {
		t.Fatalf("hint recovery = %#v, %v", recovered, err)
	}

	svc := &fakeRunnerService{}
	runner := Runner{Service: svc, InFlight: &InFlight{}, NewID: testRunnerID, GenerateDraft: func(context.Context, string, DraftRequest, string) error { return nil }}
	runner.InFlight.SetNewID(testRunnerID)
	pending, err := runner.StartDraft(context.Background(), "mis_1", DraftRequest{Title: "Report", DirectionHint: "  focus here  "}, app.Producer{Type: "user", ID: "u"})
	if err != nil {
		t.Fatal(err)
	}
	if runnerPayloadString(t, pending, "direction_hint") != "focus here" {
		t.Fatalf("pending payload: %s", pending.Payload)
	}
	without, err := runner.StartDraft(context.Background(), "mis_2", DraftRequest{Title: "Other"}, app.Producer{Type: "user", ID: "u"})
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(without.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if _, exists := payload["direction_hint"]; exists {
		t.Fatalf("empty direction serialized: %s", without.Payload)
	}
}

func TestModeNormalizationKeepsOneTakeExplicit(t *testing.T) {
	mode, err := NormalizeMode("one-take")
	if err != nil {
		t.Fatalf("NormalizeMode returned error: %v", err)
	}
	if mode != ModeOneTake {
		t.Fatalf("expected one_take compatibility mode, got %q", mode)
	}
}

func TestSelectSessionPolicyAutoIsolatedForkForReadyPlannedReport(t *testing.T) {
	policy, selection, err := SelectSessionPolicy(SessionPolicySelectionInput{
		ReportMode:                  ModePlanned,
		CanForkSession:              true,
		HasPreReportResearchSession: true,
		ForkReady:                   true,
	})
	if err != nil {
		t.Fatalf("SelectSessionPolicy returned error: %v", err)
	}
	if policy != SessionPolicyIsolatedFork || selection != SessionPolicySelectionAutoIsolatedFork {
		t.Fatalf("expected automatic isolated fork, got policy=%q selection=%q", policy, selection)
	}
}

func TestSelectSessionPolicyAutoIsolatedForkForReadyLongFormReport(t *testing.T) {
	policy, selection, err := SelectSessionPolicy(SessionPolicySelectionInput{
		ReportMode:                  ModeLongForm,
		CanForkSession:              true,
		HasPreReportResearchSession: true,
		ForkReady:                   true,
	})
	if err != nil {
		t.Fatalf("SelectSessionPolicy returned error: %v", err)
	}
	if policy != SessionPolicyIsolatedFork || selection != SessionPolicySelectionAutoIsolatedFork {
		t.Fatalf("expected automatic isolated fork for long-form reports, got policy=%q selection=%q", policy, selection)
	}
}

func TestSelectSessionPolicyAutoFallsBackWhenForkIsUnavailable(t *testing.T) {
	policy, selection, err := SelectSessionPolicy(SessionPolicySelectionInput{
		ReportMode:                  ModeLongForm,
		CanForkSession:              true,
		HasPreReportResearchSession: true,
		ForkReady:                   false,
	})
	if err != nil {
		t.Fatalf("SelectSessionPolicy returned error: %v", err)
	}
	if policy != SessionPolicySameSession || selection != SessionPolicySelectionAutoSameSessionForkFailed {
		t.Fatalf("expected same-session fork-unavailable fallback, got policy=%q selection=%q", policy, selection)
	}
}

func TestSelectSessionPolicyExplicitIsolatedForkRequiresReadyFork(t *testing.T) {
	_, _, err := SelectSessionPolicy(SessionPolicySelectionInput{
		RequestedPolicy:             SessionPolicyIsolatedFork,
		ReportMode:                  ModePlanned,
		CanForkSession:              true,
		HasPreReportResearchSession: true,
		ForkReady:                   false,
	})
	if !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("expected invalid explicit isolated fork without ready session, got %v", err)
	}
}

func TestSelectSessionPolicyExplicitIsolatedForkRejectsOneTake(t *testing.T) {
	_, _, err := SelectSessionPolicy(SessionPolicySelectionInput{
		RequestedPolicy:             SessionPolicyIsolatedFork,
		ReportMode:                  ModeOneTake,
		CanForkSession:              true,
		HasPreReportResearchSession: true,
		ForkReady:                   true,
	})
	if !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("expected invalid explicit isolated fork for one-take report, got %v", err)
	}
}

func TestSelectSessionPolicyOneTakeUsesSameSession(t *testing.T) {
	policy, selection, err := SelectSessionPolicy(SessionPolicySelectionInput{
		ReportMode:                  ModeOneTake,
		CanForkSession:              true,
		HasPreReportResearchSession: true,
		ForkReady:                   true,
	})
	if err != nil {
		t.Fatalf("SelectSessionPolicy returned error: %v", err)
	}
	if policy != SessionPolicySameSession || selection != SessionPolicySelectionAutoSameSessionOneTake {
		t.Fatalf("expected one-take report to use same session, got policy=%q selection=%q", policy, selection)
	}
}

func TestAppendDraftFailedMergesFailurePayload(t *testing.T) {
	ctx := context.Background()
	svc := &fakeRunnerService{}
	runner := Runner{Service: svc, NewID: testRunnerID}
	_, err := runner.AppendDraftFailed(ctx, "mis_1", "evt_pending_1", "codex", ModeLongForm, failurePayloadErr{
		err: errors.New("agent failed after usage"),
		payload: map[string]any{
			"kind":             "wrong_kind",
			"pending_event_id": "evt_wrong",
			"error":            "wrong error",
			"failed_surface":   "report_section",
			"agent_usage":      map[string]any{"surface": "report_section"},
		},
	})
	if err != nil {
		t.Fatalf("AppendDraftFailed returned error: %v", err)
	}
	if countRunnerEvents(svc.events, "report.draft.failed") != 1 {
		t.Fatalf("expected one failure event, got %#v", svc.events)
	}
	var payload map[string]any
	if err := json.Unmarshal(svc.events[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["failed_surface"] != "report_section" {
		t.Fatalf("expected merged failure surface, got %#v", payload)
	}
	if payload["kind"] != "report_draft_failed" || payload["pending_event_id"] != "evt_pending_1" || payload["error"] != "agent failed after usage" {
		t.Fatalf("structural failure fields must not be overwritten, got %#v", payload)
	}
	agentUsage, ok := payload["agent_usage"].(map[string]any)
	if !ok || agentUsage["surface"] != "report_section" {
		t.Fatalf("expected merged agent_usage, got %#v", payload)
	}
}

func TestAppendCanceledPreservesDraftCancelPayload(t *testing.T) {
	ctx := context.Background()
	svc := &fakeRunnerService{}
	runner := Runner{Service: svc, NewID: testRunnerID}
	pending := app.LedgerEvent{
		EventID:   "evt_pending_draft",
		MissionID: "mis_1",
		EventType: "report.draft.pending",
		Payload: mustRunnerJSON(map[string]any{
			"agent_executor": " codex ",
			"report_mode":    "long-form",
		}),
	}

	event, err := runner.AppendCanceled(ctx, "mis_1", pending, true, app.Producer{Type: "user", ID: "plasma-ui"})
	if err != nil {
		t.Fatalf("AppendCanceled returned error: %v", err)
	}
	if event.EventType != "report.draft.failed" || event.Producer.Type != "user" || event.Producer.ID != "plasma-ui" {
		t.Fatalf("unexpected cancel event shell: %#v", event)
	}
	payload := runnerPayload(t, event)
	if payload["kind"] != "report_draft_canceled" ||
		payload["pending_event_id"] != "evt_pending_draft" ||
		payload["agent_executor"] != "codex" ||
		payload["report_mode"] != ModeLongForm ||
		payload["report_mode_label"] != ModeLabel(ModeLongForm) ||
		payload["text"] != "리포트 초안 생성이 취소되었습니다." ||
		payload["error"] != "report draft canceled by user" ||
		payload["canceled"] != true ||
		payload["in_flight"] != true {
		t.Fatalf("draft cancel payload changed: %#v", payload)
	}
	if payload["canceled_at"] == "" {
		t.Fatalf("expected canceled_at timestamp, got %#v", payload)
	}
}

func TestAppendCanceledPreservesPatchCancelPayload(t *testing.T) {
	ctx := context.Background()
	svc := &fakeRunnerService{}
	runner := Runner{Service: svc, NewID: testRunnerID}
	pending := app.LedgerEvent{
		EventID:   "evt_pending_patch",
		MissionID: "mis_1",
		EventType: "report.patch.pending",
		Payload: mustRunnerJSON(map[string]any{
			"base_artifact_id": "art_base",
			"agent_executor":   "claude",
		}),
	}

	event, err := runner.AppendCanceled(ctx, "mis_1", pending, false, app.Producer{Type: "user", ID: "plasma-ui"})
	if err != nil {
		t.Fatalf("AppendCanceled returned error: %v", err)
	}
	if event.EventType != "report.patch.failed" {
		t.Fatalf("expected report.patch.failed, got %#v", event)
	}
	payload := runnerPayload(t, event)
	if payload["kind"] != "report_patch_canceled" ||
		payload["pending_event_id"] != "evt_pending_patch" ||
		payload["base_artifact_id"] != "art_base" ||
		payload["agent_executor"] != "claude" ||
		payload["text"] != "MCP 리포트 패치가 취소되었습니다." ||
		payload["error"] != "report patch canceled by user" ||
		payload["canceled"] != true ||
		payload["in_flight"] != false {
		t.Fatalf("patch cancel payload changed: %#v", payload)
	}
}

func TestBuildPatchFinalizedAppendRequestPreservesPayloadContract(t *testing.T) {
	req := BuildPatchFinalizedAppendRequest(PatchFinalizedEventRequest{
		EventID:                      "evt_patch",
		MissionID:                    "mis_1",
		CorrelationID:                "ses_tool",
		PendingEventID:               "evt_pending",
		Title:                        "Patched report",
		Artifact:                     app.RawArtifact{ArtifactID: "art_patch", MediaType: "text/markdown; charset=utf-8", ByteSize: 123, SHA256: strings.Repeat("a", 64), Filename: "patched.md"},
		BaseArtifactID:               "art_base",
		PatchID:                      "rptp_1",
		PatchInstruction:             "말투를 다듬어라",
		PatchSummary:                 "두 문단 수정",
		OperationCount:               1,
		Operations:                   []map[string]any{{"operation": "replace", "bytes": 12}},
		AgentExecutor:                "codex",
		AgentModel:                   "gpt-5.5",
		AgentReasoningEffort:         "medium",
		AgentSessionID:               "ses_report",
		PreviousAgentSessionID:       "ses_prev",
		ReturnedAgentSessionID:       "ses_returned",
		ReportSessionID:              "ses_report",
		ForkSourceAgentSessionID:     "ses_source",
		ReportSessionPolicy:          SessionPolicyIsolatedFork,
		ReportSessionPolicySelection: SessionPolicySelectionExplicitIsolatedFork,
		ToolSessionID:                "ses_tool",
		MCPMode:                      "auto",
		ProducerToolName:             "report.patch.finalize",
		SessionChainKind:             "forked",
		Producer:                     app.Producer{Type: "mcp_tool", ID: "report.patch.finalize"},
	})
	if req.EventID != "evt_patch" || req.MissionID != "mis_1" || req.EventType != "report.patch.finalized" ||
		req.CorrelationID != "ses_tool" || req.Producer.Type != "mcp_tool" || req.Producer.ID != "report.patch.finalize" {
		t.Fatalf("unexpected patch finalized event shell: %#v", req)
	}
	payload := runnerPayload(t, app.LedgerEvent{Payload: req.Payload})
	expected := map[string]any{
		"kind":                            "markdown_report_patch_finalized",
		"pending_event_id":                "evt_pending",
		"title":                           "Patched report",
		"artifact_id":                     "art_patch",
		"media_type":                      "text/markdown; charset=utf-8",
		"byte_size":                       float64(123),
		"sha256":                          strings.Repeat("a", 64),
		"filename":                        "patched.md",
		"base_artifact_id":                "art_base",
		"base_report_artifact_id":         "art_base",
		"patch_id":                        "rptp_1",
		"patch_instruction":               "말투를 다듬어라",
		"patch_summary":                   "두 문단 수정",
		"operation_count":                 float64(1),
		"agent_executor":                  "codex",
		"agent_model":                     "gpt-5.5",
		"agent_reasoning_effort":          "medium",
		"agent_session_id":                "ses_report",
		"previous_agent_session_id":       "ses_prev",
		"returned_agent_session_id":       "ses_returned",
		"report_session_id":               "ses_report",
		"fork_source_agent_session_id":    "ses_source",
		"report_session_policy":           SessionPolicyIsolatedFork,
		"report_session_policy_selection": SessionPolicySelectionExplicitIsolatedFork,
		"tool_session_id":                 "ses_tool",
		"mcp_mode":                        "auto",
		"producer_tool_name":              "report.patch.finalize",
		"composition_strategy":            "mcp_patch_markdown",
		"session_chain_kind":              "forked",
		"post_report_research_session_id": "",
		"text":                            "MCP 패치 방식으로 Markdown 리포트 artifact 새 버전을 생성했습니다.",
	}
	for key, want := range expected {
		if got := payload[key]; got != want {
			t.Fatalf("payload key %q mismatch: got %#v want %#v in %#v", key, got, want, payload)
		}
	}
	operations, ok := payload["operations"].([]any)
	if !ok || len(operations) != 1 {
		t.Fatalf("expected one operation, got %#v", payload["operations"])
	}
}

func TestBuildPromotedMarkdownReportArtifactAppendRequestPreservesFinalizedPayload(t *testing.T) {
	sourcePayload := map[string]any{
		"kind":                            "markdown_report_patch_finalized",
		"pending_event_id":                "evt_pending",
		"title":                           "Patched report",
		"artifact_id":                     "art_patch",
		"media_type":                      "text/markdown; charset=utf-8",
		"report_session_id":               "ses_report",
		"report_session_policy":           SessionPolicyIsolatedFork,
		"report_session_policy_selection": SessionPolicySelectionExplicitIsolatedFork,
		"operation_count":                 2,
		"operations":                      []any{map[string]any{"operation": "replace"}},
	}
	req := BuildPromotedMarkdownReportArtifactAppendRequest(PromotedMarkdownReportArtifactEventRequest{
		EventID:             "evt_artifact",
		MissionID:           "mis_1",
		PromotedFromEventID: "evt_patch",
		Payload:             sourcePayload,
		Producer:            app.Producer{Type: "agent_session", ID: "ses_report"},
	})
	if req.EventID != "evt_artifact" || req.MissionID != "mis_1" || req.EventType != "report.artifact.created" ||
		req.Producer.Type != "agent_session" || req.Producer.ID != "ses_report" {
		t.Fatalf("unexpected promoted artifact event shell: %#v", req)
	}
	payload := runnerPayload(t, app.LedgerEvent{Payload: req.Payload})
	expected := map[string]any{
		"kind":                            "markdown_report_artifact",
		"pending_event_id":                "evt_pending",
		"title":                           "Patched report",
		"artifact_id":                     "art_patch",
		"media_type":                      "text/markdown; charset=utf-8",
		"report_session_id":               "ses_report",
		"report_session_policy":           SessionPolicyIsolatedFork,
		"report_session_policy_selection": SessionPolicySelectionExplicitIsolatedFork,
		"operation_count":                 float64(2),
		"promoted_from_event_id":          "evt_patch",
	}
	for key, want := range expected {
		if got := payload[key]; got != want {
			t.Fatalf("payload key %q mismatch: got %#v want %#v in %#v", key, got, want, payload)
		}
	}
	operations, ok := payload["operations"].([]any)
	if !ok || len(operations) != 1 {
		t.Fatalf("expected operations to be preserved, got %#v", payload["operations"])
	}
	if sourcePayload["kind"] != "markdown_report_patch_finalized" {
		t.Fatalf("builder mutated source payload: %#v", sourcePayload)
	}
	if _, ok := sourcePayload["promoted_from_event_id"]; ok {
		t.Fatalf("builder added promoted_from_event_id to source payload: %#v", sourcePayload)
	}
}

func TestBuildMarkdownReportPlanCreatedAppendRequestPreservesPayloadContract(t *testing.T) {
	usage := agentusage.New("codex", "codex", "gpt-5.5", "medium", "plan prompt")
	req := BuildMarkdownReportPlanCreatedAppendRequest(MarkdownReportPlanCreatedEventRequest{
		MarkdownReportEventBase: MarkdownReportEventBase{
			EventID:                      "evt_plan",
			MissionID:                    "mis_1",
			PendingEventID:               "evt_pending",
			Title:                        "Report",
			AgentExecutor:                "codex",
			AgentModel:                   "gpt-5.5",
			AgentReasoningEffort:         "medium",
			AgentSessionID:               "ses_plan",
			PreviousAgentSessionID:       "ses_prev",
			ReturnedAgentSessionID:       "ses_returned",
			ToolSessionID:                "ses_tool",
			MCPMode:                      "auto",
			RigorLevel:                   "standard",
			RigorLabel:                   "표준",
			ReportMode:                   ModePlanned,
			ReportModeLabel:              ModeLabel(ModePlanned),
			ReportSessionPolicy:          SessionPolicySameSession,
			ReportSessionPolicySelection: SessionPolicySelectionExplicitSameSession,
			PostReportHumanize:           "h5",
			HumanizeEnabled:              true,
			GenerationGuidanceProfile:    "g2",
			GenerationGuidanceSHA256:     "sha",
			SessionChainKind:             "same_session_report",
			PreReportResearchSessionID:   "ses_prev",
			ReportPlanSessionID:          "ses_plan",
			ReportSessionID:              "",
			ForkSourceAgentSessionID:     "",
			PostReportResearchSessionID:  "",
			CompositionStrategy:          "planned_markdown",
			DurationMS:                   123,
			Text:                         "Markdown 리포트 생성 계획을 만들었습니다.",
			AgentUsage:                   usage,
			AgentUsageSurface:            "report_plan",
			AgentUsageDurationMS:         123,
			AgentResumed:                 true,
			Producer:                     app.Producer{Type: "agent_session", ID: "ses_plan"},
		},
		ArtifactID:         "art_report",
		Plan:               map[string]any{"sections": []string{"A"}},
		PlanReviewRequired: false,
		PlanReviewState:    "auto_accepted",
	})
	if req.EventID != "evt_plan" || req.MissionID != "mis_1" || req.EventType != "report.plan.created" ||
		req.Producer.Type != "agent_session" || req.Producer.ID != "ses_plan" {
		t.Fatalf("unexpected plan event shell: %#v", req)
	}
	payload := runnerPayload(t, app.LedgerEvent{Payload: req.Payload})
	expected := map[string]any{
		"kind":                            "markdown_report_plan",
		"pending_event_id":                "evt_pending",
		"title":                           "Report",
		"artifact_id":                     "art_report",
		"agent_executor":                  "codex",
		"agent_model":                     "gpt-5.5",
		"agent_reasoning_effort":          "medium",
		"agent_session_id":                "ses_plan",
		"previous_agent_session_id":       "ses_prev",
		"returned_agent_session_id":       "ses_returned",
		"tool_session_id":                 "ses_tool",
		"mcp_mode":                        "auto",
		"rigor_level":                     "standard",
		"rigor_label":                     "표준",
		"report_mode":                     ModePlanned,
		"report_mode_label":               ModeLabel(ModePlanned),
		"report_session_policy":           SessionPolicySameSession,
		"report_session_policy_selection": SessionPolicySelectionExplicitSameSession,
		"post_report_humanize":            "h5",
		"humanize_enabled":                true,
		"generation_guidance_profile":     "g2",
		"generation_guidance_sha256":      "sha",
		"session_chain_kind":              "same_session_report",
		"pre_report_research_session_id":  "ses_prev",
		"report_plan_session_id":          "ses_plan",
		"report_session_id":               "",
		"fork_source_agent_session_id":    "",
		"post_report_research_session_id": "",
		"composition_strategy":            "planned_markdown",
		"plan_review_required":            false,
		"plan_review_state":               "auto_accepted",
		"duration_ms":                     float64(123),
		"text":                            "Markdown 리포트 생성 계획을 만들었습니다.",
	}
	for key, want := range expected {
		if got := payload[key]; got != want {
			t.Fatalf("payload key %q mismatch: got %#v want %#v in %#v", key, got, want, payload)
		}
	}
	if _, ok := payload["plan"].(map[string]any); !ok {
		t.Fatalf("expected plan object, got %#v", payload["plan"])
	}
	assertRunnerAgentUsage(t, payload, "report_plan", 123, "ses_prev", "ses_plan", true)
}

func TestBuildMarkdownReportPlanCreatedAppendRequestUsesLongFormKindAndAssembly(t *testing.T) {
	usage := agentusage.New("codex", "codex", "gpt-5.5", "medium", "long form plan prompt")
	req := BuildMarkdownReportPlanCreatedAppendRequest(MarkdownReportPlanCreatedEventRequest{
		MarkdownReportEventBase: MarkdownReportEventBase{
			EventID:                      "evt_plan",
			MissionID:                    "mis_1",
			PendingEventID:               "evt_pending",
			Title:                        "Report",
			AgentSessionID:               "ses_plan",
			PreviousAgentSessionID:       "ses_prev",
			ReportMode:                   ModeLongForm,
			ReportModeLabel:              ModeLabel(ModeLongForm),
			ReportSessionPolicy:          SessionPolicyIsolatedFork,
			ReportSessionPolicySelection: SessionPolicySelectionAutoIsolatedFork,
			HumanizeEnabled:              true,
			CompositionStrategy:          "sectional_preserve_markdown",
			DurationMS:                   456,
			AgentUsage:                   usage,
			AgentUsageSurface:            "report_plan",
			AgentUsageDurationMS:         456,
			AgentResumed:                 true,
			Producer:                     app.Producer{Type: "agent_session", ID: "ses_plan"},
		},
		ArtifactID:         "art_report",
		Plan:               map[string]any{"parts": []string{"A"}},
		AssemblyStrategy:   "c4_normalized_section_headings",
		PlanReviewRequired: false,
		PlanReviewState:    "auto_accepted",
	})
	payload := runnerPayload(t, app.LedgerEvent{Payload: req.Payload})
	if got := payload["kind"]; got != "sectional_markdown_report_plan" {
		t.Fatalf("unexpected plan kind: got %#v in %#v", got, payload)
	}
	if got := payload["assembly_strategy"]; got != "c4_normalized_section_headings" {
		t.Fatalf("unexpected assembly_strategy: got %#v in %#v", got, payload)
	}
	assertRunnerAgentUsage(t, payload, "report_plan", 456, "ses_prev", "ses_plan", true)
}

func TestBuildMarkdownReportArtifactCreatedAppendRequestPreservesPayloadContract(t *testing.T) {
	usage := agentusage.New("codex", "codex", "gpt-5.5", "medium", "report prompt")
	req := BuildMarkdownReportArtifactCreatedAppendRequest(MarkdownReportArtifactCreatedEventRequest{
		MarkdownReportEventBase: MarkdownReportEventBase{
			EventID:                      "evt_report",
			MissionID:                    "mis_1",
			PendingEventID:               "evt_pending",
			Title:                        "Report",
			AgentExecutor:                "codex",
			AgentModel:                   "gpt-5.5",
			AgentReasoningEffort:         "medium",
			AgentSessionID:               "ses_report",
			PreviousAgentSessionID:       "ses_plan",
			ReturnedAgentSessionID:       "ses_returned",
			ToolSessionID:                "ses_tool",
			MCPMode:                      "auto",
			RigorLevel:                   "standard",
			RigorLabel:                   "표준",
			ReportMode:                   ModePlanned,
			ReportModeLabel:              ModeLabel(ModePlanned),
			ReportSessionPolicy:          SessionPolicySameSession,
			ReportSessionPolicySelection: SessionPolicySelectionExplicitSameSession,
			PostReportHumanize:           "h5",
			HumanizeEnabled:              true,
			GenerationGuidanceProfile:    "g2",
			GenerationGuidanceSHA256:     "sha",
			SessionChainKind:             "same_session_report",
			PreReportResearchSessionID:   "ses_research",
			ReportPlanSessionID:          "ses_plan",
			ReportSessionID:              "ses_report",
			ForkSourceAgentSessionID:     "",
			PostReportResearchSessionID:  "",
			CompositionStrategy:          "planned_markdown",
			DurationMS:                   456,
			Text:                         "계획 기반 Markdown 리포트 artifact를 생성했습니다.",
			AgentUsage:                   usage,
			AgentUsageSurface:            "report_markdown",
			AgentUsageDurationMS:         234,
			AgentResumed:                 true,
			Producer:                     app.Producer{Type: "agent_session", ID: "ses_report"},
		},
		Artifact:           app.RawArtifact{ArtifactID: "art_report", MediaType: "text/markdown; charset=utf-8"},
		PlanEventID:        "evt_plan",
		PlanToolSessionID:  "ses_plan_tool",
		IncludePlanReview:  true,
		PlanReviewRequired: false,
		PlanReviewState:    "auto_accepted",
	})
	if req.EventID != "evt_report" || req.MissionID != "mis_1" || req.EventType != "report.artifact.created" ||
		req.Producer.Type != "agent_session" || req.Producer.ID != "ses_report" {
		t.Fatalf("unexpected artifact event shell: %#v", req)
	}
	payload := runnerPayload(t, app.LedgerEvent{Payload: req.Payload})
	expected := map[string]any{
		"kind":                            "markdown_report_artifact",
		"pending_event_id":                "evt_pending",
		"title":                           "Report",
		"artifact_id":                     "art_report",
		"media_type":                      "text/markdown; charset=utf-8",
		"agent_executor":                  "codex",
		"agent_model":                     "gpt-5.5",
		"agent_reasoning_effort":          "medium",
		"agent_session_id":                "ses_report",
		"previous_agent_session_id":       "ses_plan",
		"returned_agent_session_id":       "ses_returned",
		"tool_session_id":                 "ses_tool",
		"plan_event_id":                   "evt_plan",
		"plan_tool_session_id":            "ses_plan_tool",
		"mcp_mode":                        "auto",
		"rigor_level":                     "standard",
		"rigor_label":                     "표준",
		"report_mode":                     ModePlanned,
		"report_mode_label":               ModeLabel(ModePlanned),
		"report_session_policy":           SessionPolicySameSession,
		"report_session_policy_selection": SessionPolicySelectionExplicitSameSession,
		"post_report_humanize":            "h5",
		"humanize_enabled":                true,
		"generation_guidance_profile":     "g2",
		"generation_guidance_sha256":      "sha",
		"session_chain_kind":              "same_session_report",
		"pre_report_research_session_id":  "ses_research",
		"report_plan_session_id":          "ses_plan",
		"report_session_id":               "ses_report",
		"fork_source_agent_session_id":    "",
		"post_report_research_session_id": "",
		"composition_strategy":            "planned_markdown",
		"plan_review_required":            false,
		"plan_review_state":               "auto_accepted",
		"duration_ms":                     float64(456),
		"text":                            "계획 기반 Markdown 리포트 artifact를 생성했습니다.",
	}
	for key, want := range expected {
		if got := payload[key]; got != want {
			t.Fatalf("payload key %q mismatch: got %#v want %#v in %#v", key, got, want, payload)
		}
	}
	assertRunnerAgentUsage(t, payload, "report_markdown", 234, "ses_plan", "ses_report", true)
}

func TestBuildMarkdownReportArtifactCreatedAppendRequestPreservesLongFormPayloadContract(t *testing.T) {
	usage := agentusage.New("codex", "codex", "gpt-5.5", "medium", "frame prompt")
	req := BuildMarkdownReportArtifactCreatedAppendRequest(MarkdownReportArtifactCreatedEventRequest{
		MarkdownReportEventBase: MarkdownReportEventBase{
			EventID:                      "evt_report",
			MissionID:                    "mis_1",
			PendingEventID:               "evt_pending",
			Title:                        "Long Report",
			AgentExecutor:                "codex",
			AgentModel:                   "gpt-5.5",
			AgentReasoningEffort:         "medium",
			AgentSessionID:               "ses_frame",
			PreviousAgentSessionID:       "ses_part",
			ReturnedAgentSessionID:       "ses_returned",
			ToolSessionID:                "ses_tool",
			MCPMode:                      "auto",
			RigorLevel:                   "standard",
			RigorLabel:                   "표준",
			ReportMode:                   ModeLongForm,
			ReportModeLabel:              ModeLabel(ModeLongForm),
			ReportSessionPolicy:          SessionPolicyIsolatedFork,
			ReportSessionPolicySelection: SessionPolicySelectionAutoIsolatedFork,
			PostReportHumanize:           "h5",
			HumanizeEnabled:              true,
			GenerationGuidanceProfile:    "g2",
			GenerationGuidanceSHA256:     "sha",
			SessionChainKind:             "forked_report",
			PreReportResearchSessionID:   "ses_research",
			ReportPlanSessionID:          "ses_plan",
			ReportSessionID:              "ses_frame",
			ForkSourceAgentSessionID:     "ses_research",
			PostReportResearchSessionID:  "",
			CompositionStrategy:          "sectional_preserve_markdown",
			DurationMS:                   999,
			Text:                         "섹션별 보존 조립 방식으로 장문 Markdown 리포트 artifact를 생성했습니다.",
			AgentUsage:                   usage,
			AgentUsageSurface:            "report_frame",
			AgentUsageDurationMS:         333,
			AgentResumed:                 true,
			Producer:                     app.Producer{Type: "agent_session", ID: "ses_frame"},
		},
		Artifact:              app.RawArtifact{ArtifactID: "art_report", MediaType: "text/markdown; charset=utf-8"},
		PlanEventID:           "evt_plan",
		IncludePlanReview:     true,
		PlanReviewRequired:    false,
		PlanReviewState:       "auto_accepted",
		AssemblyStrategy:      "c4_normalized_section_headings",
		SectionCount:          3,
		PartCount:             2,
		SectionArtifactIDs:    []string{"art_s1", "art_s2", "art_s3"},
		PartArtifactIDs:       []string{"art_p1", "art_p2"},
		SectionWordCount:      2400,
		FinalWordCount:        2600,
		PreservationRatio:     1.0833333333333333,
		IncludeLongFormFields: true,
	})
	payload := runnerPayload(t, app.LedgerEvent{Payload: req.Payload})
	expected := map[string]any{
		"kind":                            "markdown_report_artifact",
		"pending_event_id":                "evt_pending",
		"title":                           "Long Report",
		"artifact_id":                     "art_report",
		"media_type":                      "text/markdown; charset=utf-8",
		"agent_executor":                  "codex",
		"agent_model":                     "gpt-5.5",
		"agent_reasoning_effort":          "medium",
		"agent_session_id":                "ses_frame",
		"previous_agent_session_id":       "ses_part",
		"returned_agent_session_id":       "ses_returned",
		"tool_session_id":                 "ses_tool",
		"plan_event_id":                   "evt_plan",
		"mcp_mode":                        "auto",
		"rigor_level":                     "standard",
		"rigor_label":                     "표준",
		"report_mode":                     ModeLongForm,
		"report_mode_label":               ModeLabel(ModeLongForm),
		"report_session_policy":           SessionPolicyIsolatedFork,
		"report_session_policy_selection": SessionPolicySelectionAutoIsolatedFork,
		"post_report_humanize":            "h5",
		"humanize_enabled":                true,
		"generation_guidance_profile":     "g2",
		"generation_guidance_sha256":      "sha",
		"session_chain_kind":              "forked_report",
		"pre_report_research_session_id":  "ses_research",
		"report_plan_session_id":          "ses_plan",
		"report_session_id":               "ses_frame",
		"fork_source_agent_session_id":    "ses_research",
		"post_report_research_session_id": "",
		"composition_strategy":            "sectional_preserve_markdown",
		"assembly_strategy":               "c4_normalized_section_headings",
		"plan_review_required":            false,
		"plan_review_state":               "auto_accepted",
		"section_count":                   float64(3),
		"part_count":                      float64(2),
		"section_word_count":              float64(2400),
		"final_word_count":                float64(2600),
		"preservation_ratio":              1.0833333333333333,
		"duration_ms":                     float64(999),
		"text":                            "섹션별 보존 조립 방식으로 장문 Markdown 리포트 artifact를 생성했습니다.",
	}
	for key, want := range expected {
		if got := payload[key]; got != want {
			t.Fatalf("payload key %q mismatch: got %#v want %#v in %#v", key, got, want, payload)
		}
	}
	assertStringSlicePayload(t, payload, "section_artifact_ids", []string{"art_s1", "art_s2", "art_s3"})
	assertStringSlicePayload(t, payload, "part_artifact_ids", []string{"art_p1", "art_p2"})
	assertRunnerAgentUsage(t, payload, "report_frame", 333, "ses_part", "ses_frame", true)
}

func TestBuildMarkdownReportSectionCreatedAppendRequestPreservesPayloadContract(t *testing.T) {
	usage := agentusage.New("codex", "codex", "gpt-5.5", "medium", "section prompt")
	req := BuildMarkdownReportSectionCreatedAppendRequest(MarkdownReportSectionCreatedEventRequest{
		MarkdownReportStageEventBase: MarkdownReportStageEventBase{
			EventID:                      "evt_section",
			MissionID:                    "mis_1",
			PendingEventID:               "evt_pending",
			PlanEventID:                  "evt_plan",
			Title:                        "Section",
			Artifact:                     app.RawArtifact{ArtifactID: "art_section", MediaType: "text/markdown; charset=utf-8"},
			AgentExecutor:                "codex",
			AgentModel:                   "gpt-5.5",
			AgentReasoningEffort:         "medium",
			AgentSessionID:               "ses_section",
			PreviousAgentSessionID:       "ses_plan",
			ReturnedAgentSessionID:       "ses_returned",
			ToolSessionID:                "ses_tool",
			ReportMode:                   ModeLongForm,
			ReportModeLabel:              ModeLabel(ModeLongForm),
			ReportSessionPolicy:          SessionPolicyIsolatedFork,
			ReportSessionPolicySelection: SessionPolicySelectionAutoIsolatedFork,
			PostReportHumanize:           "h5",
			HumanizeEnabled:              true,
			GenerationGuidanceProfile:    "g2",
			GenerationGuidanceSHA256:     "sha",
			SessionChainKind:             "forked_report",
			PreReportResearchSessionID:   "ses_research",
			ReportPlanSessionID:          "ses_plan",
			ReportSessionID:              "ses_section",
			ForkSourceAgentSessionID:     "ses_research",
			PostReportResearchSessionID:  "",
			CompositionStrategy:          "sectional_preserve_markdown",
			AssemblyStrategy:             "c4_normalized_section_headings",
			DurationMS:                   111,
			Text:                         "장문 리포트 섹션 Markdown을 생성했습니다.",
			AgentUsage:                   usage,
			AgentUsageSurface:            "report_section",
			AgentUsageDurationMS:         111,
			AgentResumed:                 true,
			Producer:                     app.Producer{Type: "agent_session", ID: "ses_section"},
		},
		PartIndex:    1,
		SectionIndex: 2,
		WordCount:    345,
	})
	if req.EventID != "evt_section" || req.MissionID != "mis_1" || req.EventType != "report.section.created" ||
		req.Producer.Type != "agent_session" || req.Producer.ID != "ses_section" {
		t.Fatalf("unexpected section event shell: %#v", req)
	}
	payload := runnerPayload(t, app.LedgerEvent{Payload: req.Payload})
	expected := map[string]any{
		"kind":                            "sectional_markdown_report_section",
		"pending_event_id":                "evt_pending",
		"plan_event_id":                   "evt_plan",
		"title":                           "Section",
		"artifact_id":                     "art_section",
		"media_type":                      "text/markdown; charset=utf-8",
		"agent_executor":                  "codex",
		"agent_model":                     "gpt-5.5",
		"agent_reasoning_effort":          "medium",
		"agent_session_id":                "ses_section",
		"previous_agent_session_id":       "ses_plan",
		"returned_agent_session_id":       "ses_returned",
		"tool_session_id":                 "ses_tool",
		"report_mode":                     ModeLongForm,
		"report_mode_label":               ModeLabel(ModeLongForm),
		"report_session_policy":           SessionPolicyIsolatedFork,
		"report_session_policy_selection": SessionPolicySelectionAutoIsolatedFork,
		"post_report_humanize":            "h5",
		"humanize_enabled":                true,
		"generation_guidance_profile":     "g2",
		"generation_guidance_sha256":      "sha",
		"session_chain_kind":              "forked_report",
		"pre_report_research_session_id":  "ses_research",
		"report_plan_session_id":          "ses_plan",
		"report_session_id":               "ses_section",
		"fork_source_agent_session_id":    "ses_research",
		"post_report_research_session_id": "",
		"part_index":                      float64(1),
		"section_index":                   float64(2),
		"word_count":                      float64(345),
		"duration_ms":                     float64(111),
		"composition_strategy":            "sectional_preserve_markdown",
		"assembly_strategy":               "c4_normalized_section_headings",
		"text":                            "장문 리포트 섹션 Markdown을 생성했습니다.",
	}
	for key, want := range expected {
		if got := payload[key]; got != want {
			t.Fatalf("payload key %q mismatch: got %#v want %#v in %#v", key, got, want, payload)
		}
	}
	for _, unexpected := range []string{"mcp_mode", "rigor_level", "rigor_label", "plan_review_required", "plan_review_state"} {
		if _, ok := payload[unexpected]; ok {
			t.Fatalf("unexpected section payload key %q in %#v", unexpected, payload)
		}
	}
	assertRunnerAgentUsage(t, payload, "report_section", 111, "ses_plan", "ses_section", true)
}

func TestBuildMarkdownReportSectionStartedAppendRequestPreservesPayloadContract(t *testing.T) {
	req := BuildMarkdownReportSectionStartedAppendRequest(MarkdownReportSectionStartedEventRequest{
		MarkdownReportStageEventBase: MarkdownReportStageEventBase{
			EventID:                      "evt_section_start",
			MissionID:                    "mis_1",
			PendingEventID:               "evt_pending",
			PlanEventID:                  "evt_plan",
			Title:                        "Section",
			AgentExecutor:                "codex",
			AgentModel:                   "gpt-5.5",
			AgentReasoningEffort:         "medium",
			AgentSessionID:               "ses_section",
			PreviousAgentSessionID:       "ses_plan",
			ToolSessionID:                "ses_tool",
			ReportMode:                   ModeLongForm,
			ReportModeLabel:              ModeLabel(ModeLongForm),
			ReportSessionPolicy:          SessionPolicyIsolatedFork,
			ReportSessionPolicySelection: SessionPolicySelectionAutoIsolatedFork,
			PostReportHumanize:           "h5",
			HumanizeEnabled:              true,
			GenerationGuidanceProfile:    "g2",
			GenerationGuidanceSHA256:     "sha",
			SessionChainKind:             "forked_report",
			PreReportResearchSessionID:   "ses_research",
			ReportPlanSessionID:          "ses_plan",
			ReportSessionID:              "ses_section",
			ForkSourceAgentSessionID:     "ses_research",
			CompositionStrategy:          "sectional_preserve_markdown",
			AssemblyStrategy:             "c4_normalized_section_headings",
			Text:                         "장문 리포트 섹션 Markdown 생성을 시작했습니다.",
			Producer:                     app.Producer{Type: "agent_session", ID: "ses_section"},
		},
		PartIndex:    1,
		SectionIndex: 2,
	})
	if req.EventID != "evt_section_start" || req.MissionID != "mis_1" || req.EventType != "report.section.started" ||
		req.Producer.Type != "agent_session" || req.Producer.ID != "ses_section" {
		t.Fatalf("unexpected section started event shell: %#v", req)
	}
	payload := runnerPayload(t, app.LedgerEvent{Payload: req.Payload})
	expected := map[string]any{
		"kind":                            "sectional_markdown_report_section_started",
		"pending_event_id":                "evt_pending",
		"plan_event_id":                   "evt_plan",
		"title":                           "Section",
		"agent_executor":                  "codex",
		"agent_model":                     "gpt-5.5",
		"agent_reasoning_effort":          "medium",
		"agent_session_id":                "ses_section",
		"previous_agent_session_id":       "ses_plan",
		"tool_session_id":                 "ses_tool",
		"report_mode":                     ModeLongForm,
		"report_mode_label":               ModeLabel(ModeLongForm),
		"report_session_policy":           SessionPolicyIsolatedFork,
		"report_session_policy_selection": SessionPolicySelectionAutoIsolatedFork,
		"post_report_humanize":            "h5",
		"humanize_enabled":                true,
		"generation_guidance_profile":     "g2",
		"generation_guidance_sha256":      "sha",
		"session_chain_kind":              "forked_report",
		"pre_report_research_session_id":  "ses_research",
		"report_plan_session_id":          "ses_plan",
		"report_session_id":               "ses_section",
		"fork_source_agent_session_id":    "ses_research",
		"part_index":                      float64(1),
		"section_index":                   float64(2),
		"composition_strategy":            "sectional_preserve_markdown",
		"assembly_strategy":               "c4_normalized_section_headings",
		"text":                            "장문 리포트 섹션 Markdown 생성을 시작했습니다.",
	}
	for key, want := range expected {
		if got := payload[key]; got != want {
			t.Fatalf("payload key %q mismatch: got %#v want %#v in %#v", key, got, want, payload)
		}
	}
	for _, unexpected := range []string{"artifact_id", "media_type", "duration_ms", "agent_usage"} {
		if _, ok := payload[unexpected]; ok {
			t.Fatalf("unexpected section started payload key %q in %#v", unexpected, payload)
		}
	}
}

func TestBuildMarkdownReportPartCreatedAppendRequestPreservesPayloadContract(t *testing.T) {
	usage := agentusage.New("codex", "codex", "gpt-5.5", "medium", "part prompt")
	req := BuildMarkdownReportPartCreatedAppendRequest(MarkdownReportPartCreatedEventRequest{
		MarkdownReportStageEventBase: MarkdownReportStageEventBase{
			EventID:                      "evt_part",
			MissionID:                    "mis_1",
			PendingEventID:               "evt_pending",
			PlanEventID:                  "evt_plan",
			Title:                        "Part",
			Artifact:                     app.RawArtifact{ArtifactID: "art_part", MediaType: "text/markdown; charset=utf-8"},
			AgentExecutor:                "codex",
			AgentModel:                   "gpt-5.5",
			AgentReasoningEffort:         "medium",
			AgentSessionID:               "ses_part",
			PreviousAgentSessionID:       "ses_section",
			ReturnedAgentSessionID:       "ses_returned",
			ToolSessionID:                "ses_tool",
			ReportMode:                   ModeLongForm,
			ReportModeLabel:              ModeLabel(ModeLongForm),
			ReportSessionPolicy:          SessionPolicyIsolatedFork,
			ReportSessionPolicySelection: SessionPolicySelectionAutoIsolatedFork,
			PostReportHumanize:           "h5",
			HumanizeEnabled:              true,
			GenerationGuidanceProfile:    "g2",
			GenerationGuidanceSHA256:     "sha",
			SessionChainKind:             "forked_report",
			PreReportResearchSessionID:   "ses_research",
			ReportPlanSessionID:          "ses_plan",
			ReportSessionID:              "ses_part",
			ForkSourceAgentSessionID:     "ses_research",
			PostReportResearchSessionID:  "",
			CompositionStrategy:          "sectional_preserve_markdown",
			AssemblyStrategy:             "c4_normalized_section_headings",
			DurationMS:                   222,
			Text:                         "장문 리포트 파트 Markdown을 보존 조립했습니다.",
			AgentUsage:                   usage,
			AgentUsageSurface:            "report_part",
			AgentUsageDurationMS:         222,
			AgentResumed:                 true,
			Producer:                     app.Producer{Type: "agent_session", ID: "ses_part"},
		},
		PartIndex:    3,
		SectionCount: 4,
		WordCount:    1200,
	})
	if req.EventID != "evt_part" || req.MissionID != "mis_1" || req.EventType != "report.part.created" ||
		req.Producer.Type != "agent_session" || req.Producer.ID != "ses_part" {
		t.Fatalf("unexpected part event shell: %#v", req)
	}
	payload := runnerPayload(t, app.LedgerEvent{Payload: req.Payload})
	expected := map[string]any{
		"kind":                            "sectional_markdown_report_part",
		"pending_event_id":                "evt_pending",
		"plan_event_id":                   "evt_plan",
		"title":                           "Part",
		"artifact_id":                     "art_part",
		"media_type":                      "text/markdown; charset=utf-8",
		"agent_executor":                  "codex",
		"agent_model":                     "gpt-5.5",
		"agent_reasoning_effort":          "medium",
		"agent_session_id":                "ses_part",
		"previous_agent_session_id":       "ses_section",
		"returned_agent_session_id":       "ses_returned",
		"tool_session_id":                 "ses_tool",
		"report_mode":                     ModeLongForm,
		"report_mode_label":               ModeLabel(ModeLongForm),
		"report_session_policy":           SessionPolicyIsolatedFork,
		"report_session_policy_selection": SessionPolicySelectionAutoIsolatedFork,
		"post_report_humanize":            "h5",
		"humanize_enabled":                true,
		"generation_guidance_profile":     "g2",
		"generation_guidance_sha256":      "sha",
		"session_chain_kind":              "forked_report",
		"pre_report_research_session_id":  "ses_research",
		"report_plan_session_id":          "ses_plan",
		"report_session_id":               "ses_part",
		"fork_source_agent_session_id":    "ses_research",
		"post_report_research_session_id": "",
		"part_index":                      float64(3),
		"section_count":                   float64(4),
		"word_count":                      float64(1200),
		"duration_ms":                     float64(222),
		"composition_strategy":            "sectional_preserve_markdown",
		"assembly_strategy":               "c4_normalized_section_headings",
		"text":                            "장문 리포트 파트 Markdown을 보존 조립했습니다.",
	}
	for key, want := range expected {
		if got := payload[key]; got != want {
			t.Fatalf("payload key %q mismatch: got %#v want %#v in %#v", key, got, want, payload)
		}
	}
	for _, unexpected := range []string{"mcp_mode", "rigor_level", "rigor_label", "plan_review_required", "plan_review_state"} {
		if _, ok := payload[unexpected]; ok {
			t.Fatalf("unexpected part payload key %q in %#v", unexpected, payload)
		}
	}
	assertRunnerAgentUsage(t, payload, "report_part", 222, "ses_section", "ses_part", true)
}

func TestBuildSelfContainedHTMLExportAppendRequestPreservesPayloadContract(t *testing.T) {
	req := BuildSelfContainedHTMLExportAppendRequest(SelfContainedHTMLExportEventRequest{
		EventID:          "evt_export",
		MissionID:        "mis_1",
		SourceArtifactID: "art_md",
		Artifact:         app.RawArtifact{ArtifactID: "art_html", MediaType: "text/html; charset=utf-8"},
		RendererVersion:  "html-test",
		Producer:         app.Producer{Type: "plasma", ID: "html-export"},
	})
	if req.EventID != "evt_export" || req.MissionID != "mis_1" || req.EventType != "report.artifact.exported" ||
		req.Producer.Type != "plasma" || req.Producer.ID != "html-export" {
		t.Fatalf("unexpected export event shell: %#v", req)
	}
	payload := runnerPayload(t, app.LedgerEvent{Payload: req.Payload})
	expected := map[string]any{
		"kind":               ExportKindSelfContainedHTML,
		"source_artifact_id": "art_md",
		"artifact_id":        "art_html",
		"media_type":         "text/html; charset=utf-8",
		"target":             ExportTargetSelfContainedHTML,
		"renderer_version":   "html-test",
		"text":               "Self-contained HTML 리포트 artifact를 생성했습니다.",
	}
	for key, want := range expected {
		if got := payload[key]; got != want {
			t.Fatalf("payload key %q mismatch: got %#v want %#v in %#v", key, got, want, payload)
		}
	}
}

func TestBuildDesignedHTMLExportAppendRequestPreservesPayloadContract(t *testing.T) {
	usage := agentusage.New("codex", "codex", "gpt-5.5", "medium", "design prompt").
		WithProviderUsage(agentusage.ProviderUsage{InputTokens: 120, CachedInputTokens: 80, OutputTokens: 9}, "test")
	req := BuildDesignedHTMLExportAppendRequest(DesignedHTMLExportEventRequest{
		EventID:                "evt_design_export",
		MissionID:              "mis_1",
		PendingEventID:         "evt_pending",
		SourceArtifactID:       "art_md",
		ContentModelArtifactID: "art_model",
		Artifact:               app.RawArtifact{ArtifactID: "art_html", MediaType: "text/html; charset=utf-8"},
		RendererVersion:        "dh-test",
		ImageSetFingerprint:    "image-fp",
		AgentExecutor:          "codex",
		AgentModel:             "gpt-5.5",
		AgentReasoningEffort:   "medium",
		AgentSessionID:         "ses_agent",
		ToolSessionID:          "ses_tool",
		DurationMS:             2345,
		AgentDurationMS:        1234,
		AgentUsage:             usage,
		AgentResumed:           true,
		Producer:               app.Producer{Type: "agent_session", ID: "ses_agent"},
	})
	if req.EventID != "evt_design_export" || req.MissionID != "mis_1" || req.EventType != "report.artifact.exported" ||
		req.Producer.Type != "agent_session" || req.Producer.ID != "ses_agent" {
		t.Fatalf("unexpected designed export event shell: %#v", req)
	}
	payload := runnerPayload(t, app.LedgerEvent{Payload: req.Payload})
	expected := map[string]any{
		"kind":                      ExportKindDesignedHTML,
		"pending_event_id":          "evt_pending",
		"source_artifact_id":        "art_md",
		"content_model_artifact_id": "art_model",
		"artifact_id":               "art_html",
		"media_type":                "text/html; charset=utf-8",
		"target":                    ExportTargetDesignedHTML,
		"renderer_version":          "dh-test",
		"content_model_contract":    DesignedContentModelContract,
		"image_set_fingerprint":     "image-fp",
		"agent_executor":            "codex",
		"agent_model":               "gpt-5.5",
		"agent_reasoning_effort":    "medium",
		"agent_session_id":          "ses_agent",
		"tool_session_id":           "ses_tool",
		"duration_ms":               float64(2345),
		"text":                      "Designed HTML 리포트 artifact를 생성했습니다.",
	}
	for key, want := range expected {
		if got := payload[key]; got != want {
			t.Fatalf("payload key %q mismatch: got %#v want %#v in %#v", key, got, want, payload)
		}
	}
	eventUsage, ok := payload["agent_usage"].(map[string]any)
	if !ok {
		t.Fatalf("expected agent_usage payload, got %#v", payload["agent_usage"])
	}
	if eventUsage["surface"] != "report_design" || eventUsage["duration_ms"] != float64(1234) {
		t.Fatalf("unexpected agent usage envelope: %#v", eventUsage)
	}
	session, ok := eventUsage["session"].(map[string]any)
	if !ok || session["agent_session_id"] != "ses_agent" || session["resumed"] != true {
		t.Fatalf("unexpected agent usage session: %#v", eventUsage["session"])
	}
}

func TestBuildHumanizeAppendRequestsPreservePayloadContracts(t *testing.T) {
	base := HumanizeEventBase{
		EventID:                "evt_h5",
		MissionID:              "mis_1",
		PendingEventID:         "evt_pending_h5",
		ReportPendingEventID:   "evt_report",
		Title:                  "H5 report",
		SourceArtifactID:       "art_md",
		SourceArtifactSHA256:   strings.Repeat("b", 64),
		AgentExecutor:          "codex",
		AgentModel:             "gpt-5.5",
		AgentReasoningEffort:   "medium",
		PreviousAgentSessionID: "ses_prev",
		ToolSessionID:          "ses_tool",
		MCPMode:                "auto",
		ReportMode:             ModeLongForm,
		Producer:               app.Producer{Type: "agent", ID: "codex"},
	}

	pending := BuildHumanizePendingAppendRequest(HumanizePendingEventRequest{HumanizeEventBase: base})
	if pending.EventType != "report.humanize.pending" || pending.EventID != "evt_h5" {
		t.Fatalf("unexpected pending event shell: %#v", pending)
	}
	pendingPayload := runnerPayload(t, app.LedgerEvent{Payload: pending.Payload})
	if pendingPayload["kind"] != "humanized_markdown_report_pending" ||
		pendingPayload["target"] != ExportTargetHumanizedMarkdown ||
		pendingPayload["profile"] != HumanizeProfileH5 ||
		pendingPayload["pending_event_id"] != "evt_h5" ||
		pendingPayload["relationship"] != "pending_post_report_tone_pass_of_source_artifact" ||
		pendingPayload["text"] != "H5 말투 보정 Markdown artifact를 생성하는 중입니다." {
		t.Fatalf("humanize pending payload changed: %#v", pendingPayload)
	}

	skipped := BuildHumanizeSkippedAppendRequest(HumanizeSkippedEventRequest{HumanizeEventBase: base, DurationMS: 17})
	if skipped.EventType != "report.humanize.skipped" {
		t.Fatalf("unexpected skipped event shell: %#v", skipped)
	}
	skippedPayload := runnerPayload(t, app.LedgerEvent{Payload: skipped.Payload})
	if skippedPayload["kind"] != "humanized_markdown_report_skipped" ||
		skippedPayload["duration_ms"] != float64(17) ||
		skippedPayload["relationship"] != "no_change_post_report_tone_pass_of_source_artifact" ||
		skippedPayload["preserved_original_markdown"] != true ||
		skippedPayload["text"] != "H5 말투 보정 결과가 원본과 같아 별도 artifact를 만들지 않았습니다." {
		t.Fatalf("humanize skipped payload changed: %#v", skippedPayload)
	}

	failed := BuildHumanizeFailedAppendRequest(HumanizeFailedEventRequest{HumanizeEventBase: base, DurationMS: 23, Error: "agent failed"})
	if failed.EventType != "report.humanize.failed" {
		t.Fatalf("unexpected failed event shell: %#v", failed)
	}
	failedPayload := runnerPayload(t, app.LedgerEvent{Payload: failed.Payload})
	if failedPayload["kind"] != "humanized_markdown_report_failed" ||
		failedPayload["duration_ms"] != float64(23) ||
		failedPayload["error"] != "agent failed" ||
		failedPayload["relationship"] != "failed_post_report_tone_pass_of_source_artifact" ||
		failedPayload["preserved_original_markdown"] != true ||
		failedPayload["text"] != "H5 말투 보정이 실패해 원본 Markdown artifact를 유지했습니다." {
		t.Fatalf("humanize failed payload changed: %#v", failedPayload)
	}

	staleFailed := BuildHumanizeFailedAppendRequest(HumanizeFailedEventRequest{
		HumanizeEventBase: base,
		Kind:              "humanized_markdown_report_stale_failed",
		Error:             "stale humanized Markdown report generation was not running after restart",
		Text:              "H5 말투 보정 작업이 중단된 상태로 남아 원본 Markdown artifact를 유지했습니다.",
		Relationship:      "stale_post_report_tone_pass_of_source_artifact",
		OmitDuration:      true,
		FailedAt:          "2026-07-09T01:02:03Z",
	})
	stalePayload := runnerPayload(t, app.LedgerEvent{Payload: staleFailed.Payload})
	if stalePayload["kind"] != "humanized_markdown_report_stale_failed" ||
		stalePayload["relationship"] != "stale_post_report_tone_pass_of_source_artifact" ||
		stalePayload["failed_at"] != "2026-07-09T01:02:03Z" ||
		stalePayload["duration_ms"] != nil {
		t.Fatalf("humanize stale failure payload changed: %#v", stalePayload)
	}

	rejected := BuildHumanizePatchRejectedAppendRequest(HumanizePatchRejectedEventRequest{
		HumanizeEventBase: base,
		PatchEventID:      "evt_patch",
		Artifact:          app.RawArtifact{ArtifactID: "art_patch", MediaType: "text/markdown"},
		Reason:            "validation_failed",
	})
	if rejected.EventType != "report.patch.rejected" {
		t.Fatalf("unexpected rejected event shell: %#v", rejected)
	}
	rejectedPayload := runnerPayload(t, app.LedgerEvent{Payload: rejected.Payload})
	if rejectedPayload["kind"] != "markdown_report_patch_rejected" ||
		rejectedPayload["patch_event_id"] != "evt_patch" ||
		rejectedPayload["artifact_id"] != "art_patch" ||
		rejectedPayload["media_type"] != "text/markdown" ||
		rejectedPayload["reason"] != "validation_failed" ||
		rejectedPayload["relationship"] != "rejected_post_report_tone_pass_patch_artifact" {
		t.Fatalf("humanize rejected payload changed: %#v", rejectedPayload)
	}
}

func TestBuildHumanizedMarkdownExportAppendRequestPreservesPayloadContract(t *testing.T) {
	usage := agentusage.New("codex", "codex", "gpt-5.5", "medium", "humanize prompt").
		WithProviderUsage(agentusage.ProviderUsage{InputTokens: 88, CachedInputTokens: 40, OutputTokens: 11}, "test")
	req := BuildHumanizedMarkdownExportAppendRequest(HumanizedMarkdownExportEventRequest{
		HumanizeEventBase: HumanizeEventBase{
			EventID:                "evt_h5_export",
			MissionID:              "mis_1",
			PendingEventID:         "evt_pending_h5",
			ReportPendingEventID:   "evt_report",
			Title:                  "H5 report",
			SourceArtifactID:       "art_md",
			SourceArtifactSHA256:   strings.Repeat("c", 64),
			AgentExecutor:          "codex",
			AgentModel:             "gpt-5.5",
			AgentReasoningEffort:   "medium",
			PreviousAgentSessionID: "ses_prev",
			ToolSessionID:          "ses_tool",
			MCPMode:                "auto",
			ReportMode:             ModeLongForm,
			Producer:               app.Producer{Type: "agent_session", ID: "ses_agent"},
		},
		PatchEventID:           "evt_patch",
		Artifact:               app.RawArtifact{ArtifactID: "art_h5", MediaType: "text/markdown; charset=utf-8"},
		AgentSessionID:         "ses_agent",
		ReturnedAgentSessionID: "ses_agent_returned",
		SourceWordCount:        100,
		HumanizedWordCount:     102,
		DurationMS:             345,
		AgentUsage:             usage,
		AgentResumed:           true,
	})
	if req.EventType != "report.artifact.exported" || req.EventID != "evt_h5_export" ||
		req.Producer.Type != "agent_session" || req.Producer.ID != "ses_agent" {
		t.Fatalf("unexpected humanized export event shell: %#v", req)
	}
	payload := runnerPayload(t, app.LedgerEvent{Payload: req.Payload})
	expected := map[string]any{
		"kind":                        ExportKindHumanizedMarkdown,
		"target":                      ExportTargetHumanizedMarkdown,
		"profile":                     HumanizeProfileH5,
		"humanize_transport":          HumanizeTransportPatch,
		"pending_event_id":            "evt_pending_h5",
		"report_pending_event_id":     "evt_report",
		"title":                       "H5 report",
		"source_artifact_id":          "art_md",
		"source_artifact_sha256":      strings.Repeat("c", 64),
		"artifact_id":                 "art_h5",
		"media_type":                  "text/markdown; charset=utf-8",
		"agent_executor":              "codex",
		"agent_model":                 "gpt-5.5",
		"agent_reasoning_effort":      "medium",
		"agent_session_id":            "ses_agent",
		"previous_agent_session_id":   "ses_prev",
		"returned_agent_session_id":   "ses_agent_returned",
		"tool_session_id":             "ses_tool",
		"mcp_mode":                    "auto",
		"report_mode":                 ModeLongForm,
		"report_mode_label":           ModeLabel(ModeLongForm),
		"patch_event_id":              "evt_patch",
		"source_word_count":           float64(100),
		"humanized_word_count":        float64(102),
		"duration_ms":                 float64(345),
		"text":                        "H5 말투 보정 Markdown artifact를 생성했습니다.",
		"relationship":                "post_report_tone_pass_of_source_artifact",
		"preserved_original_markdown": true,
	}
	for key, want := range expected {
		if got := payload[key]; got != want {
			t.Fatalf("payload key %q mismatch: got %#v want %#v in %#v", key, got, want, payload)
		}
	}
	eventUsage, ok := payload["agent_usage"].(map[string]any)
	if !ok {
		t.Fatalf("expected agent_usage payload, got %#v", payload["agent_usage"])
	}
	if eventUsage["surface"] != "report_humanize_h5" || eventUsage["duration_ms"] != float64(345) {
		t.Fatalf("unexpected agent usage envelope: %#v", eventUsage)
	}
}

func TestAppendCanceledPreservesDesignCancelPayload(t *testing.T) {
	ctx := context.Background()
	svc := &fakeRunnerService{}
	runner := Runner{Service: svc, NewID: testRunnerID}
	pending := app.LedgerEvent{
		EventID:   "evt_pending_design",
		MissionID: "mis_1",
		EventType: "report.design.pending",
		Payload: mustRunnerJSON(map[string]any{
			"source_artifact_id": " art_source ",
			"agent_executor":     " codex ",
			"renderer_version":   " renderer-test ",
		}),
	}

	event, err := runner.AppendCanceled(ctx, "mis_1", pending, true, app.Producer{Type: "user", ID: "plasma-ui"})
	if err != nil {
		t.Fatalf("AppendCanceled returned error: %v", err)
	}
	if event.EventType != "report.design.failed" {
		t.Fatalf("expected report.design.failed, got %#v", event)
	}
	payload := runnerPayload(t, event)
	if payload["kind"] != "designed_html_report_canceled" ||
		payload["pending_event_id"] != "evt_pending_design" ||
		payload["source_artifact_id"] != " art_source " ||
		payload["agent_executor"] != "codex" ||
		payload["target"] != DesignTargetDesigned ||
		payload["renderer_version"] != " renderer-test " ||
		payload["text"] != "Designed HTML 리포트 artifact 생성이 취소되었습니다." ||
		payload["error"] != "designed HTML report generation canceled by user" ||
		payload["canceled"] != true ||
		payload["in_flight"] != true {
		t.Fatalf("design cancel payload changed: %#v", payload)
	}
}

func TestAppendCanceledPreservesHumanizeCancelPayload(t *testing.T) {
	ctx := context.Background()
	svc := &fakeRunnerService{}
	runner := Runner{Service: svc, NewID: testRunnerID}
	pending := app.LedgerEvent{
		EventID:   "evt_pending_humanize",
		MissionID: "mis_1",
		EventType: "report.humanize.pending",
		Payload: mustRunnerJSON(map[string]any{
			"report_pending_event_id":   " evt_report ",
			"title":                     " Report title ",
			"source_artifact_id":        " art_md ",
			"source_artifact_sha256":    " sha ",
			"agent_executor":            " codex ",
			"agent_model":               " model ",
			"agent_reasoning_effort":    " high ",
			"previous_agent_session_id": " ses_prev ",
			"tool_session_id":           " ses_tool ",
			"mcp_mode":                  " auto ",
			"report_mode":               " long_form ",
			"report_mode_label":         " 장문 보고서 ",
		}),
	}

	event, err := runner.AppendCanceled(ctx, "mis_1", pending, false, app.Producer{Type: "user", ID: "plasma-ui"})
	if err != nil {
		t.Fatalf("AppendCanceled returned error: %v", err)
	}
	if event.EventType != "report.humanize.failed" {
		t.Fatalf("expected report.humanize.failed, got %#v", event)
	}
	payload := runnerPayload(t, event)
	if payload["kind"] != "humanized_markdown_report_canceled" ||
		payload["target"] != ExportTargetHumanizedMarkdown ||
		payload["profile"] != HumanizeProfileH5 ||
		payload["pending_event_id"] != "evt_pending_humanize" ||
		payload["report_pending_event_id"] != "evt_report" ||
		payload["title"] != "Report title" ||
		payload["source_artifact_id"] != "art_md" ||
		payload["source_artifact_sha256"] != "sha" ||
		payload["agent_executor"] != "codex" ||
		payload["agent_model"] != "model" ||
		payload["agent_reasoning_effort"] != "high" ||
		payload["previous_agent_session_id"] != "ses_prev" ||
		payload["tool_session_id"] != "ses_tool" ||
		payload["mcp_mode"] != "auto" ||
		payload["report_mode"] != "long_form" ||
		payload["report_mode_label"] != "장문 보고서" ||
		payload["humanize_transport"] != HumanizeTransportPatch ||
		payload["text"] != "H5 말투 보정이 취소되어 원본 Markdown artifact를 유지했습니다." ||
		payload["error"] != "humanized Markdown report generation canceled by user" ||
		payload["relationship"] != "canceled_post_report_tone_pass_of_source_artifact" ||
		payload["preserved_original_markdown"] != true ||
		payload["canceled"] != true ||
		payload["in_flight"] != false {
		t.Fatalf("humanize cancel payload changed: %#v", payload)
	}
}

func TestAppendCanceledNoOpsWhenPendingAlreadyClosed(t *testing.T) {
	ctx := context.Background()
	svc := &fakeRunnerService{}
	runner := Runner{Service: svc, NewID: testRunnerID}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   "evt_existing_failed",
		MissionID: "mis_1",
		EventType: "report.draft.failed",
		Producer:  app.Producer{Type: "agent", ID: "codex"},
		Payload: mustRunnerJSON(map[string]any{
			"pending_event_id": "evt_pending_closed",
		}),
	}); err != nil {
		t.Fatal(err)
	}
	pending := app.LedgerEvent{
		EventID:   "evt_pending_closed",
		MissionID: "mis_1",
		EventType: "report.draft.pending",
		Payload:   mustRunnerJSON(map[string]any{}),
	}

	event, err := runner.AppendCanceled(ctx, "mis_1", pending, true, app.Producer{Type: "user", ID: "plasma-ui"})
	if err != nil {
		t.Fatalf("AppendCanceled returned error: %v", err)
	}
	if event.EventID != "" {
		t.Fatalf("expected no-op zero event for closed pending, got %#v", event)
	}
	if countRunnerEvents(svc.events, "report.draft.failed") != 1 {
		t.Fatalf("expected no duplicate terminal event, got %#v", svc.events)
	}
}

func TestCompletedPendingEventIDsTreatsHumanizeTerminalAsClosed(t *testing.T) {
	events := []app.LedgerEvent{
		{
			EventType: "report.humanize.failed",
			Payload: mustRunnerJSON(map[string]any{
				"pending_event_id": "evt_humanize_failed",
			}),
		},
		{
			EventType: "report.humanize.skipped",
			Payload: mustRunnerJSON(map[string]any{
				"pending_event_id": "evt_humanize_skipped",
			}),
		},
	}
	completed := CompletedPendingEventIDs(events)
	for _, pendingEventID := range []string{"evt_humanize_failed", "evt_humanize_skipped"} {
		if _, ok := completed[pendingEventID]; !ok {
			t.Fatalf("expected %s to be completed by humanize terminal event, got %#v", pendingEventID, completed)
		}
	}
}

func TestInFlightStartIsAtomic(t *testing.T) {
	var inFlight InFlight
	inFlight.SetNewID(testRunnerID)
	runID, ok := inFlight.Start("mis_1", "evt_pending_1", func() {})
	if !ok || runID == "" {
		t.Fatalf("expected first report start to succeed, got runID=%q ok=%v", runID, ok)
	}
	if secondRunID, ok := inFlight.Start("mis_1", "evt_pending_1", func() {}); ok || secondRunID != "" {
		t.Fatalf("expected duplicate report start to be rejected, got runID=%q ok=%v", secondRunID, ok)
	}
	if !inFlight.Owns("mis_1", "evt_pending_1") {
		t.Fatalf("expected first report to own pending event")
	}
	if pendingEventID, ok := inFlight.PendingEventID("mis_1"); !ok || pendingEventID != "evt_pending_1" {
		t.Fatalf("expected pending event lookup for active run, got pending=%q ok=%v", pendingEventID, ok)
	}
	inFlight.Finish("mis_1", runID)
	nextRunID, ok := inFlight.Start("mis_1", "evt_pending_2", func() {})
	if !ok || nextRunID == "" {
		t.Fatalf("expected report start after finish to succeed, got runID=%q ok=%v", nextRunID, ok)
	}
}

func TestInFlightCancelInvokesRegisteredCancel(t *testing.T) {
	var inFlight InFlight
	canceled := false
	if _, ok := inFlight.Start("mis_1", "evt_pending_1", func() { canceled = true }); !ok {
		t.Fatal("expected in-flight start to succeed")
	}
	if !inFlight.Cancel("mis_1", "evt_pending_1") {
		t.Fatal("expected cancel to find in-flight report")
	}
	if !canceled {
		t.Fatal("expected registered cancel function to be invoked")
	}
	if inFlight.Owns("mis_1", "evt_pending_1") {
		t.Fatal("expected cancel to clear in-flight ownership")
	}
}

func TestRunnerDuplicateResumeSameDraftPendingNoOps(t *testing.T) {
	ctx := context.Background()
	svc := &fakeRunnerService{}
	inFlight := &InFlight{}
	inFlight.SetNewID(testRunnerID)
	started := make(chan struct{})
	release := make(chan struct{})
	var generateCount int
	runner := Runner{
		Service:  svc,
		InFlight: inFlight,
		NewID:    testRunnerID,
		GenerateDraft: func(context.Context, string, DraftRequest, string) error {
			generateCount++
			close(started)
			<-release
			return nil
		},
	}

	if err := runner.RunDraft(ctx, "mis_1", DraftRequest{Title: "Report", AgentExecutor: "codex", ReportMode: ModePlanned}, "evt_pending_1"); err != nil {
		t.Fatalf("first RunDraft returned error: %v", err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("runner did not start first draft")
	}
	if err := runner.RunDraft(ctx, "mis_1", DraftRequest{Title: "Report", AgentExecutor: "codex", ReportMode: ModePlanned}, "evt_pending_1"); err != nil {
		t.Fatalf("duplicate same-pending RunDraft should no-op, got %v", err)
	}
	close(release)
	time.Sleep(10 * time.Millisecond)
	if generateCount != 1 {
		t.Fatalf("duplicate same-pending RunDraft should not generate twice, got %d", generateCount)
	}
	if countRunnerEvents(svc.events, "report.draft.failed") != 0 {
		t.Fatalf("duplicate same-pending RunDraft should not append failure, got %#v", svc.events)
	}
}

func TestRunnerDuplicateDifferentDraftPendingClosesNewPending(t *testing.T) {
	ctx := context.Background()
	svc := &fakeRunnerService{}
	inFlight := &InFlight{}
	inFlight.SetNewID(testRunnerID)
	started := make(chan struct{})
	release := make(chan struct{})
	runner := Runner{
		Service:  svc,
		InFlight: inFlight,
		NewID:    testRunnerID,
		GenerateDraft: func(context.Context, string, DraftRequest, string) error {
			close(started)
			<-release
			return nil
		},
	}

	if err := runner.RunDraft(ctx, "mis_1", DraftRequest{Title: "Report", AgentExecutor: "codex", ReportMode: ModePlanned}, "evt_pending_1"); err != nil {
		t.Fatalf("first RunDraft returned error: %v", err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("runner did not start first draft")
	}
	err := runner.RunDraft(ctx, "mis_1", DraftRequest{Title: "Report", AgentExecutor: "codex", ReportMode: ModePlanned}, "evt_pending_2")
	if !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("expected different pending conflict error, got %v", err)
	}
	close(release)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if countRunnerEvents(svc.events, "report.draft.failed") == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if countRunnerEvents(svc.events, "report.draft.failed") != 1 {
		t.Fatalf("different pending conflict should append one failure, got %#v", svc.events)
	}
	failed := latestRunnerEventOfType(svc.events, "report.draft.failed")
	if pending := runnerPayloadString(t, failed, "pending_event_id"); pending != "evt_pending_2" {
		t.Fatalf("expected failed event to close new pending, got %q", pending)
	}
}

func TestRunnerDuplicateResumeSameDesignPendingNoOps(t *testing.T) {
	ctx := context.Background()
	svc := &fakeRunnerService{}
	inFlight := &InFlight{}
	inFlight.SetNewID(testRunnerID)
	started := make(chan struct{})
	release := make(chan struct{})
	var generateCount int
	runner := Runner{
		Service:  svc,
		InFlight: inFlight,
		NewID:    testRunnerID,
		GenerateDesign: func(context.Context, string, DesignRequest, string) error {
			generateCount++
			close(started)
			<-release
			return nil
		},
	}

	req := DesignRequest{SourceArtifactID: "art_1", Title: "Report", AgentExecutor: "codex", RendererVersion: "test"}
	if err := runner.RunDesign(ctx, "mis_1", req, "evt_design_pending_1"); err != nil {
		t.Fatalf("first RunDesign returned error: %v", err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("runner did not start first design export")
	}
	if err := runner.RunDesign(ctx, "mis_1", req, "evt_design_pending_1"); err != nil {
		t.Fatalf("duplicate same-pending RunDesign should no-op, got %v", err)
	}
	close(release)
	time.Sleep(10 * time.Millisecond)
	if generateCount != 1 {
		t.Fatalf("duplicate same-pending RunDesign should not generate twice, got %d", generateCount)
	}
	if countRunnerEvents(svc.events, "report.design.failed") != 0 {
		t.Fatalf("duplicate same-pending RunDesign should not append failure, got %#v", svc.events)
	}
}

type failurePayloadErr struct {
	err     error
	payload map[string]any
}

func (err failurePayloadErr) Error() string {
	return err.err.Error()
}

func (err failurePayloadErr) Unwrap() error {
	return err.err
}

func (err failurePayloadErr) FailurePayload() map[string]any {
	return err.payload
}

type fakeRunnerService struct {
	mu      sync.Mutex
	events  []app.LedgerEvent
	sources []app.SourceSnapshot
}

func (svc *fakeRunnerService) snapshot() []app.LedgerEvent {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	return append([]app.LedgerEvent(nil), svc.events...)
}

func (svc *fakeRunnerService) AppendEvent(_ context.Context, req app.AppendEventRequest) (app.LedgerEvent, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	event := app.LedgerEvent{
		EventID:   req.EventID,
		MissionID: req.MissionID,
		EventType: req.EventType,
		Producer:  req.Producer,
		Payload:   append(json.RawMessage(nil), req.Payload...),
		CreatedAt: time.Now().UTC(),
	}
	svc.events = append(svc.events, event)
	return event, nil
}

func (svc *fakeRunnerService) AppendEvents(_ context.Context, _ string, reqs []app.AppendEventRequest) ([]app.LedgerEvent, error) {
	appended := make([]app.LedgerEvent, 0, len(reqs))
	for _, req := range reqs {
		event, err := svc.AppendEvent(context.Background(), req)
		if err != nil {
			return nil, err
		}
		appended = append(appended, event)
	}
	return appended, nil
}

func (svc *fakeRunnerService) AppendReportTerminalIfOpen(ctx context.Context, missionID, pendingID string, reqs []app.AppendEventRequest) ([]app.LedgerEvent, bool, error) {
	svc.mu.Lock()
	for _, event := range svc.events {
		if event.MissionID != missionID {
			continue
		}
		var payload struct {
			PendingID string `json:"pending_event_id"`
		}
		_ = json.Unmarshal(event.Payload, &payload)
		if payload.PendingID == pendingID && strings.Contains(event.EventType, "failed") {
			svc.mu.Unlock()
			return nil, false, nil
		}
	}
	svc.mu.Unlock()
	appended, err := svc.AppendEvents(ctx, missionID, reqs)
	return appended, err == nil, err
}

func (svc *fakeRunnerService) AppendEventsIfNoActiveAgentWork(_ context.Context, _ string, reqs []app.AppendEventRequest) ([]app.LedgerEvent, error) {
	appended := make([]app.LedgerEvent, 0, len(reqs))
	for _, req := range reqs {
		event, err := svc.AppendEvent(context.Background(), req)
		if err != nil {
			return nil, err
		}
		appended = append(appended, event)
	}
	return appended, nil
}

func (svc *fakeRunnerService) ListEvents(_ context.Context, missionID string) ([]app.LedgerEvent, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	events := []app.LedgerEvent{}
	for _, event := range svc.events {
		if event.MissionID == strings.TrimSpace(missionID) {
			events = append(events, event)
		}
	}
	return events, nil
}

func (svc *fakeRunnerService) ListSourceSnapshotsWithState(_ context.Context, req app.ListSourceSnapshotsRequest) ([]app.SourceSnapshot, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	sources := make([]app.SourceSnapshot, 0, len(svc.sources))
	for _, source := range svc.sources {
		if source.MissionID == strings.TrimSpace(req.MissionID) {
			sources = append(sources, source)
		}
	}
	return sources, nil
}

func runnerPayloadString(t *testing.T, event app.LedgerEvent, key string) string {
	t.Helper()
	payload := runnerPayload(t, event)
	value, _ := payload[key].(string)
	return value
}

func runnerPayload(t *testing.T, event app.LedgerEvent) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func assertRunnerAgentUsage(t *testing.T, payload map[string]any, surface string, durationMS float64, previousSessionID string, sessionID string, resumed bool) {
	t.Helper()
	eventUsage, ok := payload["agent_usage"].(map[string]any)
	if !ok {
		t.Fatalf("expected agent_usage payload, got %#v in %#v", payload["agent_usage"], payload)
	}
	if eventUsage["surface"] != surface || eventUsage["duration_ms"] != durationMS {
		t.Fatalf("unexpected agent usage envelope: %#v", eventUsage)
	}
	session, ok := eventUsage["session"].(map[string]any)
	if !ok {
		t.Fatalf("expected agent usage session, got %#v", eventUsage["session"])
	}
	if session["previous_agent_session_id"] != previousSessionID || session["agent_session_id"] != sessionID || session["resumed"] != resumed {
		t.Fatalf("unexpected agent usage session: %#v", session)
	}
}

func assertStringSlicePayload(t *testing.T, payload map[string]any, key string, want []string) {
	t.Helper()
	values, ok := payload[key].([]any)
	if !ok {
		t.Fatalf("expected %s payload slice, got %#v in %#v", key, payload[key], payload)
	}
	if len(values) != len(want) {
		t.Fatalf("payload key %q length mismatch: got %#v want %#v", key, values, want)
	}
	for index, expected := range want {
		if values[index] != expected {
			t.Fatalf("payload key %q[%d] mismatch: got %#v want %#v in %#v", key, index, values[index], expected, values)
		}
	}
}

func countRunnerEvents(events []app.LedgerEvent, eventType string) int {
	count := 0
	for _, event := range events {
		if event.EventType == eventType {
			count++
		}
	}
	return count
}

func latestRunnerEventOfType(events []app.LedgerEvent, eventType string) app.LedgerEvent {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].EventType == eventType {
			return events[i]
		}
	}
	return app.LedgerEvent{}
}

func testRunnerID(prefix string) string {
	return prefix + "_test"
}

func mustRunnerJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}
