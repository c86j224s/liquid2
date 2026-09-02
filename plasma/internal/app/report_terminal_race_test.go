package app_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestAppendReportTerminalIfOpenClosesPendingOnceConcurrently(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	const missionID = "mis_terminal_race"
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "terminal race"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvents(ctx, missionID, []app.AppendEventRequest{{
		EventID:   "evt_pending",
		MissionID: missionID,
		EventType: "report.draft.pending",
		Producer:  app.Producer{Type: "agent", ID: "codex"},
		Payload:   jsonPayload(map[string]any{"report_mode": "long_form"}),
	}}); err != nil {
		t.Fatal(err)
	}

	terminal := func(id, kind string) []app.AppendEventRequest {
		return []app.AppendEventRequest{{
			EventID:   id,
			MissionID: missionID,
			EventType: "report.draft.failed",
			Producer:  app.Producer{Type: "agent", ID: kind},
			Payload:   jsonPayload(map[string]any{"kind": kind, "pending_event_id": "evt_pending"}),
		}}
	}
	type result struct {
		ok  bool
		err error
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for i, kind := range []string{"worker_failed", "user_canceled"} {
		wg.Add(1)
		go func(i int, kind string) {
			defer wg.Done()
			_, ok, err := svc.AppendReportTerminalIfOpen(ctx, missionID, "evt_pending", terminal("evt_terminal_"+string(rune('a'+i)), kind))
			results <- result{ok: ok, err: err}
		}(i, kind)
	}
	wg.Wait()
	close(results)

	var winners int
	for result := range results {
		if result.err != nil {
			t.Fatal(result.err)
		}
		if result.ok {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("expected exactly one terminal winner, got %d", winners)
	}
	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("expected pending plus one terminal event, got %d events", len(events))
	}
}

func TestCreateMarkdownReportArtifactIfOpenCommitsOneArtifactAndTerminalConcurrently(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	svc := app.NewService(store)
	const (
		missionID  = "mis_unverified_artifact_race"
		pendingID  = "evt_unverified_artifact_race_pending"
		artifactID = "art_unverified_race_fixture"
	)
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "unverified artifact race"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID: pendingID, MissionID: missionID, EventType: "report.draft.pending",
		Producer: app.Producer{Type: "user", ID: "test"},
		Payload:  jsonPayload(map[string]any{"pipeline_family": "report_unverified"}),
	}); err != nil {
		t.Fatal(err)
	}

	type result struct {
		created bool
		err     error
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, _, created, err := svc.CreateMarkdownReportArtifactIfOpen(
				ctx,
				missionID,
				pendingID,
				app.CreateRawArtifactRequest{
					ArtifactID: artifactID, MissionID: missionID,
					MediaType: "text/markdown; charset=utf-8", Filename: "report.md",
					Producer: app.Producer{Type: "agent_session", ID: "ses_unverified"},
					Content:  []byte("# exact provider output\n"),
				},
				func(artifact app.RawArtifact) app.AppendEventRequest {
					return app.AppendEventRequest{
						EventID:   "evt_unverified_artifact_race_terminal_" + string(rune('a'+index)),
						MissionID: missionID, EventType: "report.artifact.created",
						Producer: app.Producer{Type: "agent_session", ID: "ses_unverified"},
						Payload: jsonPayload(map[string]any{
							"kind": "markdown_report_artifact", "pending_event_id": pendingID,
							"pipeline_family": "report_unverified", "artifact_id": artifact.ArtifactID,
						}),
					}
				},
			)
			results <- result{created: created, err: err}
		}(i)
	}
	wg.Wait()
	close(results)

	winners := 0
	for result := range results {
		if result.err != nil {
			t.Fatal(result.err)
		}
		if result.created {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("expected exactly one artifact winner, got %d", winners)
	}
	artifact, err := svc.GetRawArtifact(ctx, artifactID)
	if err != nil {
		t.Fatal(err)
	}
	if string(artifact.Content) != "# exact provider output\n" {
		t.Fatalf("stored artifact changed: %q", string(artifact.Content))
	}
	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[1].EventType != "report.artifact.created" {
		t.Fatalf("race committed duplicate terminals: %#v", events)
	}
}

func TestAppendReportTerminalIfOpenRejectsWrongPendingTypeAndCorrelation(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	const missionID = "mis_terminal_validation"
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "terminal validation"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_1",
		MissionID:  missionID,
		MediaType:  "text/markdown",
		Filename:   "report.md",
		Producer:   app.Producer{Type: "agent", ID: "codex"},
		Content:    []byte("report"),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{EventID: "evt_design_pending", MissionID: missionID, EventType: "report.design.pending", Producer: app.Producer{Type: "agent", ID: "codex"}, Payload: jsonPayload(map[string]any{"source_artifact_id": "art_1"})}); err != nil {
		t.Fatal(err)
	}
	for _, req := range []app.AppendEventRequest{
		{EventID: "evt_wrong_type", MissionID: missionID, EventType: "report.patch.failed", Producer: app.Producer{Type: "agent", ID: "codex"}, Payload: jsonPayload(map[string]any{"pending_event_id": "evt_design_pending"})},
		{EventID: "evt_wrong_correlation", MissionID: missionID, EventType: "report.design.failed", Producer: app.Producer{Type: "agent", ID: "codex"}, Payload: jsonPayload(map[string]any{"pending_event_id": "evt_other"})},
	} {
		if _, _, err := svc.AppendReportTerminalIfOpen(ctx, missionID, "evt_design_pending", []app.AppendEventRequest{req}); err == nil {
			t.Fatalf("expected conditional terminal validation error for %s", req.EventID)
		}
	}
	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("invalid closures must not append, got %#v", events)
	}
}

func TestAppendReportTerminalIfOpenRejectsILCompanionForClassicPending(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	const missionID = "mis_classic_il_companion"
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "classic"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{EventID: "evt_pending", MissionID: missionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: jsonPayload(map[string]any{"report_mode": "long_form"})}); err != nil {
		t.Fatal(err)
	}
	stage := app.AppendEventRequest{EventID: "evt_stage", MissionID: missionID, EventType: "report.source_packet.failed", Producer: app.Producer{Type: "agent", ID: "codex"}, CorrelationID: "evt_terminal", Payload: jsonPayload(map[string]any{"pending_event_id": "evt_pending", "stage_kind": "source_packet", "stage_id": "source_packet", "terminal_event_id": "evt_terminal"})}
	terminal := app.AppendEventRequest{EventID: "evt_terminal", MissionID: missionID, EventType: "report.draft.failed", Producer: app.Producer{Type: "agent", ID: "codex"}, Payload: jsonPayload(map[string]any{"pending_event_id": "evt_pending", "kind": "report_draft_failed", "failed_stage_kind": "source_packet", "failed_stage_id": "source_packet", "stage_failure_event_id": "evt_stage"})}
	if _, _, err := svc.AppendReportTerminalIfOpen(ctx, missionID, "evt_pending", []app.AppendEventRequest{stage, terminal}); err == nil {
		t.Fatal("classic pending accepted experimental source companion")
	}
}

func TestAppendReportTerminalIfOpenRejectsExperimentalFamilyForClassicPending(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	const missionID = "mis_classic_il_terminal"
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "classic IL terminal"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID: "evt_classic_pending", MissionID: missionID, EventType: "report.draft.pending",
		Producer: app.Producer{Type: "user", ID: "test"}, Payload: jsonPayload(map[string]any{"report_mode": "planned"}),
	}); err != nil {
		t.Fatal(err)
	}
	terminal := app.AppendEventRequest{
		EventID: "evt_il_family_terminal", MissionID: missionID, EventType: "report.artifact.created",
		Producer: app.Producer{Type: "agent", ID: "codex"},
		Payload: jsonPayload(map[string]any{
			"kind": "markdown_report_artifact", "pending_event_id": "evt_classic_pending",
			"pipeline_family": "report_il_experimental", "artifact_id": "art_markdown",
		}),
	}
	if _, _, err := svc.AppendReportTerminalIfOpen(ctx, missionID, "evt_classic_pending", []app.AppendEventRequest{terminal}); err == nil || !strings.Contains(err.Error(), "family") {
		t.Fatalf("expected mismatched terminal family rejection, got %v", err)
	}
	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].EventID != "evt_classic_pending" {
		t.Fatalf("mismatched terminal family mutated ledger: %#v", events)
	}
}

func TestAppendReportTerminalIfOpenRejectsUnknownPendingFamilyAsClassic(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	const missionID = "mis_unknown_report_family"
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "unknown family"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID: "evt_unknown_pending", MissionID: missionID, EventType: "report.draft.pending",
		Producer: app.Producer{Type: "user", ID: "test"},
		Payload:  jsonPayload(map[string]any{"pipeline_family": "report_unknown"}),
	}); err != nil {
		t.Fatal(err)
	}
	terminal := app.AppendEventRequest{
		EventID: "evt_familyless_terminal", MissionID: missionID, EventType: "report.artifact.created",
		Producer: app.Producer{Type: "agent", ID: "codex"},
		Payload: jsonPayload(map[string]any{
			"kind": "markdown_report_artifact", "pending_event_id": "evt_unknown_pending",
			"artifact_id": "art_unknown_family",
		}),
	}
	if _, _, err := svc.AppendReportTerminalIfOpen(ctx, missionID, "evt_unknown_pending", []app.AppendEventRequest{terminal}); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("expected unknown pending family rejection, got %v", err)
	}
	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].EventID != "evt_unknown_pending" {
		t.Fatalf("unknown family terminal mutated ledger: %#v", events)
	}
}

func TestAppendReportTerminalIfOpenRejectsExperimentalSuccessOutsideAtomicBundle(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	const missionID = "mis_il_terminal_bypass"
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "IL terminal bypass"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   "evt_il_pending",
		MissionID: missionID,
		EventType: "report.draft.pending",
		Producer:  app.Producer{Type: "user", ID: "test"},
		Payload:   jsonPayload(map[string]any{"pipeline_family": "report_il_experimental"}),
	}); err != nil {
		t.Fatal(err)
	}
	terminal := app.AppendEventRequest{
		EventID:   "evt_il_terminal",
		MissionID: missionID,
		EventType: "report.artifact.created",
		Producer:  app.Producer{Type: "agent", ID: "codex"},
		Payload: jsonPayload(map[string]any{
			"kind": "markdown_report_artifact", "pending_event_id": "evt_il_pending",
			"pipeline_family": "report_il_experimental", "artifact_id": "art_markdown",
		}),
	}
	if _, _, err := svc.AppendReportTerminalIfOpen(ctx, missionID, "evt_il_pending", []app.AppendEventRequest{terminal}); err == nil || !strings.Contains(err.Error(), "atomic bundle") {
		t.Fatalf("expected non-atomic IL success rejection, got %v", err)
	}
	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].EventID != "evt_il_pending" {
		t.Fatalf("non-atomic IL success mutated ledger: %#v", events)
	}
}

func TestAppendReportTerminalIfOpenAcceptsLongFormILCompanions(t *testing.T) {
	for _, kind := range []string{"il_long_form_plan", "il_long_form_sections", "il_long_form_parts", "il_long_form_final"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			svc := app.NewService(store)
			const missionID = "mis_long_form_il_companion"
			if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "long-form IL companion"}); err != nil {
				t.Fatal(err)
			}
			if _, err := svc.AppendEvents(ctx, missionID, []app.AppendEventRequest{{
				EventID: "evt_pending", MissionID: missionID, EventType: "report.draft.pending",
				Producer: app.Producer{Type: "agent", ID: "codex"},
				Payload:  jsonPayload(map[string]any{"report_mode": "long_form", "pipeline_family": "report_il_experimental"}),
			}}); err != nil {
				t.Fatal(err)
			}
			stage := app.AppendEventRequest{
				EventID: "evt_stage", MissionID: missionID, EventType: "report." + kind + ".failed",
				Producer: app.Producer{Type: "agent", ID: "codex"}, CorrelationID: "evt_terminal",
				Payload: jsonPayload(map[string]any{
					"pending_event_id": "evt_pending", "stage_kind": kind, "stage_id": kind,
					"terminal_event_id": "evt_terminal",
				}),
			}
			terminal := app.AppendEventRequest{
				EventID: "evt_terminal", MissionID: missionID, EventType: "report.draft.failed",
				Producer: app.Producer{Type: "agent", ID: "codex"},
				Payload: jsonPayload(map[string]any{
					"pending_event_id": "evt_pending", "kind": "report_draft_failed",
					"failed_stage_kind": kind, "failed_stage_id": kind, "stage_failure_event_id": "evt_stage",
				}),
			}
			appended, ok, err := svc.AppendReportTerminalIfOpen(ctx, missionID, "evt_pending", []app.AppendEventRequest{stage, terminal})
			if err != nil || !ok || len(appended) != 2 {
				t.Fatalf("long-form IL companion append = %#v, ok=%t err=%v", appended, ok, err)
			}
		})
	}
}

func TestAppendReportTerminalIfOpenAcceptsRequirementsAndPartEditCompanions(t *testing.T) {
	for _, tc := range []struct {
		name, kind, stageID string
		partIndex           int
	}{
		{name: "requirements", kind: "requirements", stageID: "requirements"},
		{name: "part plan", kind: "part_plan", stageID: "part-plan-1", partIndex: 1},
		{name: "part edit", kind: "part_edit", stageID: "part-edit-1", partIndex: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			svc := app.NewService(store)
			const missionID = "mis_terminal_companion"
			if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "terminal companion"}); err != nil {
				t.Fatal(err)
			}
			if _, err := svc.AppendEvents(ctx, missionID, []app.AppendEventRequest{{
				EventID:   "evt_pending",
				MissionID: missionID,
				EventType: "report.draft.pending",
				Producer:  app.Producer{Type: "agent", ID: "codex"},
				Payload:   jsonPayload(map[string]any{"report_mode": "long_form"}),
			}}); err != nil {
				t.Fatal(err)
			}
			stageEventType := "report." + tc.kind + ".failed"
			stage := app.AppendEventRequest{
				EventID:       "evt_stage",
				MissionID:     missionID,
				EventType:     stageEventType,
				Producer:      app.Producer{Type: "agent", ID: "codex"},
				CorrelationID: "evt_terminal",
				Payload: jsonPayload(map[string]any{
					"pending_event_id": "evt_pending", "stage_kind": tc.kind, "stage_id": tc.stageID,
					"part_index": tc.partIndex, "terminal_event_id": "evt_terminal",
				}),
			}
			terminal := app.AppendEventRequest{
				EventID:   "evt_terminal",
				MissionID: missionID,
				EventType: "report.draft.failed",
				Producer:  app.Producer{Type: "agent", ID: "codex"},
				Payload: jsonPayload(map[string]any{
					"pending_event_id": "evt_pending", "kind": "report_draft_failed",
					"failed_stage_kind": tc.kind, "failed_stage_id": tc.stageID, "stage_failure_event_id": "evt_stage",
				}),
			}
			appended, ok, err := svc.AppendReportTerminalIfOpen(ctx, missionID, "evt_pending", []app.AppendEventRequest{stage, terminal})
			if err != nil || !ok {
				t.Fatalf("expected companion terminal append to succeed, ok=%t err=%v", ok, err)
			}
			if len(appended) != 2 || appended[0].EventType != stageEventType || appended[1].EventType != "report.draft.failed" {
				t.Fatalf("unexpected companion append result: %#v", appended)
			}
		})
	}
}

func jsonPayload(value map[string]any) json.RawMessage {
	data, _ := json.Marshal(value)
	return data
}

func TestRequestReportRetryAllowsExperimentalLongFormRestart(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	const missionID = "mis_retry_il"
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "IL retry"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvents(ctx, missionID, []app.AppendEventRequest{
		{EventID: "evt_il_pending", MissionID: missionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: jsonPayload(map[string]any{"report_mode": "long_form", "pipeline_family": "report_il_experimental"})},
		{EventID: "evt_il_failed", MissionID: missionID, EventType: "report.draft.failed", Producer: app.Producer{Type: "agent", ID: "codex"}, Payload: jsonPayload(map[string]any{"pending_event_id": "evt_il_pending", "kind": "report_draft_failed"})},
	}); err != nil {
		t.Fatal(err)
	}
	retry, err := svc.RequestReportRetry(ctx, app.ReportRetryRequest{EventID: "evt_il_retry", MissionID: missionID, FailedPendingEventID: "evt_il_pending", Strategy: "restart", RetryRequestID: "retry-il", Producer: app.Producer{Type: "user", ID: "test"}})
	if err != nil || retry.EventID != "evt_il_retry" {
		t.Fatalf("expected experimental restart, retry=%#v err=%v", retry, err)
	}
}

func TestRequestReportRetryAllowsRecoverableLegacyExperimentalResume(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	const missionID = "mis_retry_il_legacy"
	const pendingID = "evt_il_legacy_pending"
	finalizePayload := func(tool, stage, artifact string, accounts int) map[string]any {
		return map[string]any{
			"tool_name": tool, "success": true,
			"io_metrics": map[string]any{
				"report_il_stage": stage, "artifact_id": artifact,
				"sha256": strings.Repeat("a", 64), "byte_size": 100,
				"revision": 1, "accounts": accounts, "finalized": true,
			},
		}
	}
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "IL legacy retry"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvents(ctx, missionID, []app.AppendEventRequest{
		{EventID: pendingID, MissionID: missionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: jsonPayload(map[string]any{"report_mode": "long_form", "pipeline_family": "report_il_experimental"})},
		{EventID: "evt_memory", MissionID: missionID, EventType: "mcp.tool.called", Producer: app.Producer{Type: "agent_session", ID: "ses_memory"}, Payload: jsonPayload(finalizePayload(reportilcontract.EditorialMemoryFinalizeTool, "il_editorial_memory", "art_memory", 1))},
		{EventID: "evt_final_tool", MissionID: missionID, EventType: "mcp.tool.called", Producer: app.Producer{Type: "agent_session", ID: "ses_final"}, Payload: jsonPayload(finalizePayload(reportilcontract.LongFormDocumentFinalizeTool, "il_long_form_final", "art_final", 0))},
		{EventID: "evt_final_done", MissionID: missionID, EventType: "report.il_long_form_final.completed", Producer: app.Producer{Type: "system", ID: "report-il"}, Payload: jsonPayload(map[string]any{"pending_event_id": pendingID})},
		{EventID: "evt_il_failed", MissionID: missionID, EventType: "report.draft.failed", Producer: app.Producer{Type: "agent", ID: "codex"}, Payload: jsonPayload(map[string]any{"pending_event_id": pendingID, "kind": "report_draft_failed", "failed_stage_kind": "il_reader"})},
	}); err != nil {
		t.Fatal(err)
	}
	retry, err := svc.RequestReportRetry(ctx, app.ReportRetryRequest{EventID: "evt_il_retry", MissionID: missionID, FailedPendingEventID: pendingID, Strategy: "resume_failed", RetryRequestID: "retry-il-legacy", Producer: app.Producer{Type: "user", ID: "test"}})
	if err != nil || retry.EventID != "evt_il_retry" {
		t.Fatalf("expected recoverable legacy retry, retry=%#v err=%v", retry, err)
	}
}

func TestRequestReportRetryRejectsExperimentalResumeWithoutCheckpoint(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	const missionID = "mis_retry_il_checkpoint"
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: missionID, Title: "IL retry"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AppendEvents(ctx, missionID, []app.AppendEventRequest{
		{EventID: "evt_il_pending", MissionID: missionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: jsonPayload(map[string]any{"report_mode": "long_form", "pipeline_family": "report_il_experimental"})},
		{EventID: "evt_il_failed", MissionID: missionID, EventType: "report.draft.failed", Producer: app.Producer{Type: "agent", ID: "codex"}, Payload: jsonPayload(map[string]any{"pending_event_id": "evt_il_pending", "kind": "report_draft_failed"})},
	}); err != nil {
		t.Fatal(err)
	}
	_, err = svc.RequestReportRetry(ctx, app.ReportRetryRequest{EventID: "evt_il_retry", MissionID: missionID, FailedPendingEventID: "evt_il_pending", Strategy: "resume_failed", RetryRequestID: "retry-il", Producer: app.Producer{Type: "user", ID: "test"}})
	if err == nil || !strings.Contains(err.Error(), "checkpoint") {
		t.Fatalf("expected missing checkpoint rejection, got %v", err)
	}
}
