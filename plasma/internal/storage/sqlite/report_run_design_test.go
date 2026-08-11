package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reportrun"
)

func TestDesignedReportHTMLExportConditionalCommitRegistersTwoArtifactsAtomically(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	final := seedCompletedReportRun(t, ctx, store, "mis_rr_design_atomic", "evt_rr_design_root", "evt_rr_design_final", "art_rr_design_final")
	pendingID := appendDesignPending(t, ctx, store, final.MissionID, final.ArtifactID, "evt_rr_design_pending")
	before, err := store.LoadReportRunDeleteFacts(ctx, final.MissionID, final.ArtifactID)
	if err != nil {
		t.Fatalf("LoadReportRunDeleteFacts before design returned error: %v", err)
	}

	model, html, event, closed, err := commitDesignedExport(t, ctx, store, final.MissionID, pendingID, final.ArtifactID, "art_rr_design_model", "art_rr_design_html")
	if err != nil || !closed {
		t.Fatalf("designed export commit failed: closed=%t err=%v", closed, err)
	}
	after, err := store.LoadReportRunDeleteFacts(ctx, final.MissionID, final.ArtifactID)
	if err != nil {
		t.Fatalf("LoadReportRunDeleteFacts after design returned error: %v", err)
	}
	if after.Run.Revision <= before.Run.Revision {
		t.Fatalf("designed export did not advance run revision: before=%d after=%d", before.Run.Revision, after.Run.Revision)
	}
	preview := reportrun.PreviewDelete(after, "")
	if !preview.Eligible || preview.DeletableArtifactCount != 3 || preview.DeletableEventCount != 4 {
		t.Fatalf("designed export should delete final, model, html, and lineage: %#v", preview)
	}
	if event.EventType != "report.artifact.exported" {
		t.Fatalf("unexpected terminal event: %#v", event)
	}
	for _, artifact := range []app.RawArtifact{model, html} {
		if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_raw_artifacts WHERE artifact_id = ?`, artifact.ArtifactID) != 1 {
			t.Fatalf("artifact %s was not committed", artifact.ArtifactID)
		}
	}
}

func TestDesignedReportHTMLExportConditionalCommitLeavesNoArtifactsWhenPendingClosed(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	final := seedCompletedReportRun(t, ctx, store, "mis_rr_design_closed", "evt_rr_design_closed_root", "evt_rr_design_closed_final", "art_rr_design_closed_final")
	pendingID := appendDesignPending(t, ctx, store, final.MissionID, final.ArtifactID, "evt_rr_design_closed_pending")
	svc := app.NewService(store)
	if _, ok, err := svc.AppendReportTerminalIfOpen(ctx, final.MissionID, pendingID, []app.AppendEventRequest{{
		EventID:   "evt_rr_design_closed_failed",
		MissionID: final.MissionID,
		EventType: "report.design.failed",
		Producer:  app.Producer{Type: "agent", ID: "test"},
		Payload:   []byte(`{"pending_event_id":"` + pendingID + `","kind":"report_design_canceled","canceled":true}`),
	}}); err != nil || !ok {
		t.Fatalf("closing design pending failed: ok=%t err=%v", ok, err)
	}

	_, _, _, closed, err := commitDesignedExport(t, ctx, store, final.MissionID, pendingID, final.ArtifactID, "art_rr_design_closed_model", "art_rr_design_closed_html")
	if err != nil || closed {
		t.Fatalf("closed pending should lose condition without error: closed=%t err=%v", closed, err)
	}
	assertNoDesignedCompletionWrites(t, ctx, store, "evt_rr_design_export_"+pendingID, "art_rr_design_closed_model", "art_rr_design_closed_html")
}

func TestDesignedReportHTMLExportConditionalCommitRollsBackArtifactInsertError(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	final := seedCompletedReportRun(t, ctx, store, "mis_rr_design_insert_fail", "evt_rr_design_insert_root", "evt_rr_design_insert_final", "art_rr_design_insert_final")
	pendingID := appendDesignPending(t, ctx, store, final.MissionID, final.ArtifactID, "evt_rr_design_insert_pending")
	createStoredArtifact(t, ctx, store, final.MissionID, "art_rr_design_insert_model", []byte("duplicate"))

	_, _, _, closed, err := commitDesignedExport(t, ctx, store, final.MissionID, pendingID, final.ArtifactID, "art_rr_design_insert_model", "art_rr_design_insert_html")
	if err == nil || closed {
		t.Fatalf("expected duplicate artifact insert to fail before commit, closed=%t err=%v", closed, err)
	}
	assertNoDesignedCompletionWrites(t, ctx, store, "evt_rr_design_export_"+pendingID, "art_rr_design_insert_html")
}

func TestDesignedReportHTMLExportConditionalCommitRollsBackRegistrationError(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	final := seedCompletedReportRun(t, ctx, store, "mis_rr_design_reg_fail", "evt_rr_design_reg_root", "evt_rr_design_reg_final", "art_rr_design_reg_final")
	pendingID := appendDesignPending(t, ctx, store, final.MissionID, final.ArtifactID, "evt_rr_design_reg_pending")
	if err := store.CreateMission(ctx, app.Mission{MissionID: "mis_rr_design_reg_other", Title: "Other"}); err != nil {
		t.Fatalf("CreateMission other returned error: %v", err)
	}
	createStoredArtifact(t, ctx, store, "mis_rr_design_reg_other", "art_rr_design_reg_foreign", []byte("foreign"))

	_, _, _, closed, err := commitDesignedExport(t, ctx, store, final.MissionID, pendingID, "art_rr_design_reg_foreign", "art_rr_design_reg_model", "art_rr_design_reg_html")
	if err == nil || closed {
		t.Fatalf("expected cross-mission registration to fail before commit, closed=%t err=%v", closed, err)
	}
	assertNoDesignedCompletionWrites(t, ctx, store, "evt_rr_design_export_"+pendingID, "art_rr_design_reg_model", "art_rr_design_reg_html")
}

func TestReportRunNativeRejectsCrossMissionArtifactReference(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	final := seedCompletedReportRun(t, ctx, store, "mis_rr_native_cross", "evt_rr_native_cross_root", "evt_rr_native_cross_final", "art_rr_native_cross_final")
	if err := store.CreateMission(ctx, app.Mission{MissionID: "mis_rr_native_cross_other", Title: "Other"}); err != nil {
		t.Fatalf("CreateMission other returned error: %v", err)
	}
	createStoredArtifact(t, ctx, store, "mis_rr_native_cross_other", "art_rr_native_cross_foreign", []byte("foreign"))
	svc := app.NewService(store)

	_, _, err := svc.CreateRawArtifactWithEvent(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_rr_native_cross_html",
		MissionID:  final.MissionID,
		MediaType:  "text/html",
		Filename:   "cross.html",
		Producer:   app.Producer{Type: "agent", ID: "test"},
		Content:    []byte("<html></html>"),
	}, func(artifact app.RawArtifact) app.AppendEventRequest {
		return app.AppendEventRequest{
			EventID:   "evt_rr_native_cross_export",
			MissionID: final.MissionID,
			EventType: "report.artifact.exported",
			Producer:  app.Producer{Type: "agent", ID: "test"},
			Payload:   []byte(`{"pending_event_id":"evt_rr_native_cross_root","source_artifact_id":"art_rr_native_cross_foreign","artifact_id":"` + artifact.ArtifactID + `"}`),
		}
	})
	if err == nil {
		t.Fatal("expected native cross-mission report artifact reference to be rejected")
	}
	assertNoDesignedCompletionWrites(t, ctx, store, "evt_rr_native_cross_export", "art_rr_native_cross_html")
}

func TestReportRunBackfillCrossMissionArtifactReferenceMarksRunAmbiguous(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	missionID := "mis_rr_backfill_cross"
	if err := store.CreateMission(ctx, app.Mission{MissionID: missionID, Title: missionID}); err != nil {
		t.Fatalf("CreateMission returned error: %v", err)
	}
	if err := store.CreateMission(ctx, app.Mission{MissionID: "mis_rr_backfill_cross_other", Title: "Other"}); err != nil {
		t.Fatalf("CreateMission other returned error: %v", err)
	}
	createStoredArtifact(t, ctx, store, "mis_rr_backfill_cross_other", "art_rr_backfill_cross_foreign", []byte("foreign"))
	insertLedgerPayloadDirect(t, ctx, store, missionID, "evt_rr_backfill_cross_pending", "report.draft.pending", `{"title":"Legacy"}`)
	insertLedgerPayloadDirect(t, ctx, store, missionID, "evt_rr_backfill_cross_final", "report.artifact.created", `{"pending_event_id":"evt_rr_backfill_cross_pending","artifact_id":"art_rr_backfill_cross_foreign"}`)

	events, err := store.ListLedgerEvents(ctx, missionID)
	if err != nil {
		t.Fatalf("ListLedgerEvents returned error: %v", err)
	}
	registration, err := reportrun.BuildRegistration(reportrunEvents(events), reportrun.RegistrationBackfilled, time.Now().UTC())
	if err != nil {
		t.Fatalf("BuildRegistration returned error: %v", err)
	}
	if err := store.BackfillReportRuns(ctx, registration); err != nil {
		t.Fatalf("BackfillReportRuns returned error: %v", err)
	}
	var state string
	if err := store.db.QueryRowContext(ctx, `SELECT lifecycle_state FROM plasma_report_runs WHERE run_id = ?`, "evt_rr_backfill_cross_pending").Scan(&state); err != nil {
		t.Fatalf("load backfilled run: %v", err)
	}
	if state != reportrun.LifecycleAmbiguous {
		t.Fatalf("cross-mission backfill run state = %q, want ambiguous", state)
	}
	if got := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_report_run_artifacts WHERE run_id = ?`, "evt_rr_backfill_cross_pending"); got != 0 {
		t.Fatalf("cross-mission artifact membership should be skipped, got %d", got)
	}
	if _, err := store.LoadReportRunDeleteFacts(ctx, missionID, "art_rr_backfill_cross_foreign"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("foreign artifact should not become delete target, got %v", err)
	}
}

func TestReportRunDeleteFactsBlocksExistingMissionInconsistentProjection(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	final := seedCompletedReportRun(t, ctx, store, "mis_rr_projection_mismatch", "evt_rr_projection_root", "evt_rr_projection_final", "art_rr_projection_final")
	if err := store.CreateMission(ctx, app.Mission{MissionID: "mis_rr_projection_other", Title: "Other"}); err != nil {
		t.Fatalf("CreateMission other returned error: %v", err)
	}
	createStoredArtifact(t, ctx, store, "mis_rr_projection_other", "art_rr_projection_foreign", []byte("foreign"))
	if _, err := store.db.ExecContext(ctx, `
INSERT INTO plasma_report_run_artifacts (
  run_id, artifact_id, mission_id, artifact_role, ownership,
  attempt_event_id, source_event_id, created_at
) VALUES (?, ?, ?, 'intermediate', 'referenced', '', '', ?)`,
		"evt_rr_projection_root", "art_rr_projection_foreign", final.MissionID, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		t.Fatalf("insert mismatched projection: %v", err)
	}

	facts, err := store.LoadReportRunDeleteFacts(ctx, final.MissionID, final.ArtifactID)
	if err != nil {
		t.Fatalf("LoadReportRunDeleteFacts returned error: %v", err)
	}
	preview := reportrun.PreviewDelete(facts, "")
	if preview.Eligible || !hasDeleteBlocker(preview, reportrun.BlockerAmbiguousLineage) {
		t.Fatalf("mission-inconsistent projection should block as ambiguous lineage: %#v", preview)
	}
}

func appendDesignPending(t *testing.T, ctx context.Context, store *Store, missionID string, sourceArtifactID string, pendingID string) string {
	t.Helper()
	appendLedgerEvent(t, ctx, store, app.LedgerEvent{
		EventID:   pendingID,
		MissionID: missionID,
		EventType: "report.design.pending",
		Producer:  app.Producer{Type: "user", ID: "test"},
		Payload:   []byte(`{"kind":"designed_html_report_pending","source_artifact_id":"` + sourceArtifactID + `","target":"designed_html","renderer_version":"test"}`),
	})
	return pendingID
}

func commitDesignedExport(t *testing.T, ctx context.Context, store *Store, missionID string, pendingID string, sourceArtifactID string, modelID string, htmlID string) (app.RawArtifact, app.RawArtifact, app.LedgerEvent, bool, error) {
	t.Helper()
	svc := app.NewService(store)
	return svc.CreateDesignedReportHTMLExportIfOpen(ctx, missionID, pendingID, app.CreateRawArtifactRequest{
		ArtifactID: modelID,
		MissionID:  missionID,
		MediaType:  "application/json",
		Filename:   modelID + ".json",
		Producer:   app.Producer{Type: "agent", ID: "test"},
		Content:    []byte(`{"title":"Designed"}`),
	}, app.CreateRawArtifactRequest{
		ArtifactID: htmlID,
		MissionID:  missionID,
		MediaType:  "text/html",
		Filename:   htmlID + ".html",
		Producer:   app.Producer{Type: "agent", ID: "test"},
		Content:    []byte("<!doctype html><title>Designed</title>"),
	}, func(model app.RawArtifact, html app.RawArtifact) app.AppendEventRequest {
		return app.AppendEventRequest{
			EventID:   "evt_rr_design_export_" + pendingID,
			MissionID: missionID,
			EventType: "report.artifact.exported",
			Producer:  app.Producer{Type: "agent", ID: "test"},
			Payload: []byte(`{"kind":"designed_html_report_artifact","pending_event_id":"` + pendingID +
				`","source_artifact_id":"` + sourceArtifactID +
				`","content_model_artifact_id":"` + model.ArtifactID +
				`","artifact_id":"` + html.ArtifactID +
				`","target":"designed_html","renderer_version":"test"}`),
		}
	})
}

func assertNoDesignedCompletionWrites(t *testing.T, ctx context.Context, store *Store, eventID string, artifactIDs ...string) {
	t.Helper()
	if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_ledger_events WHERE event_id = ?`, eventID) != 0 {
		t.Fatalf("terminal event %s remained after failed conditional commit", eventID)
	}
	for _, artifactID := range artifactIDs {
		if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_raw_artifacts WHERE artifact_id = ?`, artifactID) != 0 {
			t.Fatalf("artifact %s remained after failed conditional commit", artifactID)
		}
	}
}
