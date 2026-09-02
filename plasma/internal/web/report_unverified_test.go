package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/agentcapability"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportpipeline"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestUnverifiedReportUsesOneSourceOnlyCallAndStoresExactProviderBytes(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	const (
		missionID    = "mis_unverified_exact"
		pendingID    = "evt_unverified_exact_pending"
		sourceBody   = "PRIVATE SOURCE BODY MUST NOT ENTER THE PROMPT"
		providerText = "\n\n# 그대로 저장\n\n본문 끝의 공백도 보존한다.  \n\n"
	)
	svc := app.NewService(store)
	createUnverifiedMissionFixture(t, ctx, svc, missionID, pendingID, sourceBody)
	executor := &fakeAgentExecutor{responses: []AgentResult{{
		Text: providerText, SessionID: "ses_unverified_provider",
	}}}
	handler := NewServer(svc, Options{AgentExecutor: executor})
	server := handler.(*Server)

	err = server.createUnverifiedReportDraft(ctx, missionID, reportexecution.DraftRequest{
		Title: "자료 기반 자유 보고서", DirectionHint: "연대기보다 쟁점을 중심으로",
		PipelineFamily: reportpipeline.Unverified,
	}, pendingID)
	if err != nil {
		t.Fatal(err)
	}
	if len(executor.requests) != 1 {
		t.Fatalf("provider call count = %d, want 1", len(executor.requests))
	}
	request := executor.requests[0]
	if request.UserText != "write unverified markdown report" ||
		request.Model != "gpt-5.6-luna" || request.ReasoningEffort != "xhigh" ||
		request.AgentExecutor != "codex" || request.MCPMode != "source_read_only" ||
		request.CapabilityProfile != agentcapability.ProfileReportUnverifiedV1 ||
		request.ProfileRevision != agentcapability.RevisionV1 || !request.ReplaceMCPTools ||
		request.DisableTools || request.OutputJSONSchema != nil || request.ReportILSources != nil ||
		request.ReportPlan != nil || request.ReportRequirements != nil || request.PartAssembly != nil ||
		request.PartEdit != nil || request.LongFormFinalize != nil || request.FinalEditStage != nil {
		t.Fatalf("unexpected unverified provider request: %#v", request)
	}
	wantTools := []string{mcptools.ToolSourcesList, mcptools.ToolSourcesRead}
	if !slices.Equal(request.ExtraMCPTools, wantTools) {
		t.Fatalf("unverified tools = %#v, want %#v", request.ExtraMCPTools, wantTools)
	}
	for _, expected := range []string{
		"plasma.sources.list", "plasma.sources.read", "자료 기반 자유 보고서",
		"연대기보다 쟁점을 중심으로", "제품 목표 원문",
	} {
		if !strings.Contains(request.Prompt, expected) {
			t.Fatalf("unverified prompt is missing %q: %s", expected, request.Prompt)
		}
	}
	if strings.Contains(request.Prompt, sourceBody) {
		t.Fatal("unverified prompt embedded a source body")
	}

	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	if countLedgerEvents(events, "report.plan.created") != 0 ||
		countLedgerEvents(events, "report.requirements.mapped") != 0 ||
		countLedgerEvents(events, "report.final_edit.started") != 0 ||
		countLedgerEvents(events, "report.humanize.pending") != 0 ||
		countLedgerEvents(events, "report.artifact.created") != 1 ||
		countLedgerEvents(events, "report.run.completed") != 1 {
		t.Fatalf("unexpected unverified report event sequence: %#v", events)
	}
	var terminal app.LedgerEvent
	for _, event := range events {
		if event.EventType == "report.artifact.created" {
			terminal = event
		}
	}
	if terminal.EventID == "" {
		t.Fatal("unverified terminal event is missing")
	}
	var payload map[string]any
	if err := json.Unmarshal(terminal.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	artifactID := unverifiedReportArtifactID(missionID, pendingID)
	if payload["pipeline_family"] != reportpipeline.Unverified ||
		payload["composition_strategy"] != "unverified_single_call" ||
		payload["artifact_id"] != artifactID || payload["pending_event_id"] != pendingID ||
		payload["post_report_humanize"] != "disabled" {
		t.Fatalf("unexpected unverified terminal payload: %#v", payload)
	}
	artifact, err := svc.GetRawArtifact(ctx, artifactID)
	if err != nil {
		t.Fatal(err)
	}
	if string(artifact.Content) != providerText {
		t.Fatalf("stored bytes changed provider output: got %q want %q", string(artifact.Content), providerText)
	}

	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()
	response, err := http.Get(httpServer.URL + "/api/missions/" + missionID + "/artifacts/" + artifactID + "/download")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	downloaded, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || string(downloaded) != providerText {
		t.Fatalf("download changed provider bytes: status=%d got=%q want=%q", response.StatusCode, string(downloaded), providerText)
	}

	if err := server.createUnverifiedReportDraft(ctx, missionID, reportexecution.DraftRequest{
		Title: "자료 기반 자유 보고서", PipelineFamily: reportpipeline.Unverified,
	}, pendingID); err != nil {
		t.Fatalf("closed-pending replay failed: %v", err)
	}
	events, err = svc.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(executor.requests) != 1 || countLedgerEvents(events, "report.artifact.created") != 1 ||
		countLedgerEvents(events, "report.run.completed") != 1 {
		t.Fatalf("closed-pending replay duplicated output: calls=%d events=%#v", len(executor.requests), events)
	}
}

func TestUnverifiedReportReplayCompletesStoredTerminalWithoutProviderCall(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	const (
		missionID = "mis_unverified_completion_replay"
		pendingID = "evt_unverified_completion_replay_pending"
	)
	svc := app.NewService(store)
	createUnverifiedMissionFixture(t, ctx, svc, missionID, pendingID, "source body")
	artifactID := unverifiedReportArtifactID(missionID, pendingID)
	if _, _, created, err := svc.CreateMarkdownReportArtifactIfOpen(
		ctx,
		missionID,
		pendingID,
		app.CreateRawArtifactRequest{
			ArtifactID: artifactID, MissionID: missionID,
			MediaType: "text/markdown; charset=utf-8", Filename: "report.md",
			Producer: app.Producer{Type: "agent_session", ID: "ses_unverified_replay"},
			Content:  []byte("# stored before completion receipt\n"),
		},
		func(artifact app.RawArtifact) app.AppendEventRequest {
			return app.AppendEventRequest{
				EventID: "evt_unverified_completion_replay_terminal", MissionID: missionID,
				EventType: "report.artifact.created",
				Producer:  app.Producer{Type: "agent_session", ID: "ses_unverified_replay"},
				Payload: mustJSON(map[string]any{
					"kind": "markdown_report_artifact", "pending_event_id": pendingID,
					"pipeline_family": reportpipeline.Unverified, "artifact_id": artifact.ArtifactID,
				}),
			}
		},
	); err != nil || !created {
		t.Fatalf("precondition terminal commit failed: created=%t err=%v", created, err)
	}
	executor := &fakeAgentExecutor{}
	server := NewServer(svc, Options{AgentExecutor: executor}).(*Server)
	if err := server.createUnverifiedReportDraft(ctx, missionID, reportexecution.DraftRequest{
		Title: "completion replay", PipelineFamily: reportpipeline.Unverified,
	}, pendingID); err != nil {
		t.Fatalf("completion replay failed: %v", err)
	}
	if len(executor.requests) != 0 {
		t.Fatalf("completion replay called provider: %#v", executor.requests)
	}
	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	if countLedgerEvents(events, "report.artifact.created") != 1 ||
		countLedgerEvents(events, "report.run.completed") != 1 {
		t.Fatalf("completion replay did not close receipt exactly once: %#v", events)
	}
}

func TestUnverifiedReportAcceptsCancellationThatWinsWhileProviderRuns(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	const (
		missionID = "mis_unverified_cancel_race"
		pendingID = "evt_unverified_cancel_race_pending"
	)
	svc := app.NewService(store)
	createUnverifiedMissionFixture(t, ctx, svc, missionID, pendingID, "source body")
	executor := &fakeAgentExecutor{
		responses: []AgentResult{{Text: "# provider finished\n", SessionID: "ses_unverified_cancel"}},
		onRun: func(_ context.Context, _ AgentRequest) {
			_, _, err := svc.AppendReportTerminalIfOpen(ctx, missionID, pendingID, []app.AppendEventRequest{{
				EventID: "evt_unverified_cancel_race_terminal", MissionID: missionID,
				EventType: "report.draft.failed", Producer: app.Producer{Type: "user", ID: "test"},
				Payload: mustJSON(map[string]any{
					"kind": "report_draft_canceled", "pending_event_id": pendingID,
					"canceled": true,
				}),
			}})
			if err != nil {
				t.Errorf("cancel terminal failed: %v", err)
			}
		},
	}
	server := NewServer(svc, Options{AgentExecutor: executor}).(*Server)
	if err := server.createUnverifiedReportDraft(ctx, missionID, reportexecution.DraftRequest{
		Title: "cancel race", PipelineFamily: reportpipeline.Unverified,
	}, pendingID); err != nil {
		t.Fatalf("cancel winner surfaced as generation failure: %v", err)
	}
	if len(executor.requests) != 1 {
		t.Fatalf("provider call count = %d, want 1", len(executor.requests))
	}
	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	if countLedgerEvents(events, "report.draft.failed") != 1 ||
		countLedgerEvents(events, "report.artifact.created") != 0 ||
		countLedgerEvents(events, "report.run.completed") != 0 {
		t.Fatalf("cancel race stored provider output after cancellation: %#v", events)
	}
}

func TestUnverifiedReportRejectsOnlyEmptyOrOversizedProviderContent(t *testing.T) {
	for _, test := range []struct {
		name string
		text string
		want string
	}{
		{name: "empty", text: " \n\t ", want: "empty Markdown"},
		{name: "oversized", text: strings.Repeat("x", unverifiedReportMaxBytes+1), want: "size limit"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			svc := app.NewService(store)
			missionID := "mis_unverified_boundary_" + test.name
			pendingID := "evt_unverified_boundary_" + test.name
			createUnverifiedMissionFixture(t, ctx, svc, missionID, pendingID, "source body")
			executor := &fakeAgentExecutor{responses: []AgentResult{{
				Text: test.text, SessionID: "ses_unverified_boundary",
			}}}
			server := NewServer(svc, Options{AgentExecutor: executor}).(*Server)
			err = server.createUnverifiedReportDraft(ctx, missionID, reportexecution.DraftRequest{
				Title: "boundary", PipelineFamily: reportpipeline.Unverified,
			}, pendingID)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q rejection, got %v", test.want, err)
			}
			if len(executor.requests) != 1 {
				t.Fatalf("provider call count = %d, want 1", len(executor.requests))
			}
			events, err := svc.ListEvents(ctx, missionID)
			if err != nil {
				t.Fatal(err)
			}
			if countLedgerEvents(events, "report.artifact.created") != 0 ||
				countLedgerEvents(events, "report.run.completed") != 0 {
				t.Fatalf("invalid provider content created an artifact: %#v", events)
			}
		})
	}
}

func createUnverifiedMissionFixture(
	t *testing.T,
	ctx context.Context,
	svc *app.Service,
	missionID string,
	pendingID string,
	sourceBody string,
) {
	t.Helper()
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{
		MissionID: missionID, Title: "무검증형 fixture",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, app.BuildMissionCreatedAppendRequest(app.MissionCreatedEventRequest{
		EventID: "evt_created_" + strings.TrimPrefix(missionID, "mis_"), MissionID: missionID,
		Title: "무검증형 fixture", Objective: "제품 목표 원문", Producer: app.Producer{Type: "user", ID: "test"},
	})); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RebuildProjection(ctx, missionID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateSourceSnapshotWithEvent(ctx, app.CreateSourceSnapshotWithEventRequest{
		Artifact: app.CreateRawArtifactRequest{
			ArtifactID: "art_source_" + strings.TrimPrefix(missionID, "mis_"), MissionID: missionID,
			MediaType: "text/plain; charset=utf-8", Filename: "source.txt",
			Producer: app.Producer{Type: "user", ID: "test"}, Content: []byte(sourceBody),
		},
		Snapshot: app.CreateSourceSnapshotRequest{
			SnapshotID: "src_" + strings.TrimPrefix(missionID, "mis_"), MissionID: missionID,
			Connector: app.ConnectorRef{
				ConnectorID: "fixture", ConnectorType: app.SourceConnectorTypeFileUpload,
				ExternalSourceID: "source.txt",
			},
			Title: "fixture source", Locators: json.RawMessage(`[{"locator_type":"full_text"}]`),
			Access: app.SourceAccess{Visibility: "private", RetrievalPolicy: app.SourceRetrievalPolicySnapshotOnly},
		},
		Event: app.AppendEventRequest{
			EventID: "evt_source_" + strings.TrimPrefix(missionID, "mis_"), MissionID: missionID,
			EventType: "source.snapshotted", Producer: app.Producer{Type: "user", ID: "test"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID: pendingID, MissionID: missionID, EventType: "report.draft.pending",
		Producer: app.Producer{Type: "user", ID: "test"}, Payload: mustJSON(map[string]any{
			"title": "자료 기반 자유 보고서", "pipeline_family": reportpipeline.Unverified,
			"agent_executor": "codex", "agent_model": "gpt-5.6-luna",
			"agent_reasoning_effort": "xhigh", "mcp_mode": "source_read_only",
		}),
	}); err != nil {
		t.Fatal(err)
	}
}
