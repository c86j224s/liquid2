package sqlite

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"github.com/c86j224s/liquid2/plasma/internal/reportrun"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/artifactrepo"
)

func TestReportILAtomicBundleCommitsSevenArtifactsAndMemberships(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	missionID := "mis_il_atomic_success"
	pendingID := "evt_il_atomic_pending"
	if err := store.CreateMission(ctx, app.Mission{MissionID: missionID, Title: "IL atomic"}); err != nil {
		t.Fatal(err)
	}
	appendReportILPending(t, ctx, store, missionID, pendingID)

	req, expected := reportILBundleRequest(t, missionID, pendingID, "evt_il_atomic_store", "evt_il_atomic_terminal")
	artifacts, terminal, created, err := app.NewService(store).CreateReportILBundleIfOpen(ctx, req)
	if err != nil {
		t.Fatalf("CreateReportILBundleIfOpen returned error: %v", err)
	}
	if !created || terminal.EventID != "evt_il_atomic_terminal" || len(artifacts) != 7 {
		t.Fatalf("unexpected bundle result: created=%v terminal=%#v artifacts=%d", created, terminal, len(artifacts))
	}

	var eventTypes []string
	for _, event := range mustListEvents(t, store, missionID) {
		eventTypes = append(eventTypes, event.EventType)
	}
	if got := strings.Join(eventTypes, ","); got != "report.draft.pending,report.il_store.completed,report.artifact.created" {
		t.Fatalf("event order/types = %q", got)
	}
	if got := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_raw_artifacts WHERE mission_id = ?`, missionID); got != 7 {
		t.Fatalf("raw artifact count = %d, want 7", got)
	}
	if got := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_report_run_artifacts WHERE run_id = ?`, pendingID); got != 7 {
		t.Fatalf("artifact membership count = %d, want 7", got)
	}
	for _, item := range expected {
		var role, ownership, sourceEventID string
		if err := store.db.QueryRowContext(ctx, `
SELECT artifact_role, ownership, source_event_id
FROM plasma_report_run_artifacts
WHERE run_id = ? AND artifact_id = ?`, pendingID, item.ArtifactID).Scan(&role, &ownership, &sourceEventID); err != nil {
			t.Fatalf("load membership %s: %v", item.ArtifactID, err)
		}
		if role != item.Role || ownership != "created" || sourceEventID != "evt_il_atomic_terminal" {
			t.Fatalf("membership %s = role %q ownership %q source %q", item.ArtifactID, role, ownership, sourceEventID)
		}
	}
	var lifecycle, finalArtifactID, registrationStatus string
	if err := store.db.QueryRowContext(ctx, `
SELECT lifecycle_state, final_artifact_id, registration_status
FROM plasma_report_runs WHERE run_id = ?`, pendingID).Scan(&lifecycle, &finalArtifactID, &registrationStatus); err != nil {
		t.Fatal(err)
	}
	if lifecycle != "completed" || finalArtifactID != "art_il_atomic_markdown" || registrationStatus != "native" {
		t.Fatalf("report run = lifecycle %q final %q registration %q", lifecycle, finalArtifactID, registrationStatus)
	}
}

func TestReportILAtomicBundleStoresRepeatedImageBytesWithFreshIdentity(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	missionID := "mis_il_repeated_image"
	pendingID := "evt_il_repeated_image_pending"
	if err := store.CreateMission(ctx, app.Mission{MissionID: missionID, Title: "Repeated image"}); err != nil {
		t.Fatal(err)
	}
	imageContent := []byte("same source-backed image")
	existingCreatedAt := time.Date(2026, time.August, 28, 1, 2, 3, 0, time.UTC)
	if _, err := app.NewService(store).CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_existing_image", MissionID: missionID, MediaType: "image/jpeg",
		Filename: "source-image.jpg", Producer: app.Producer{Type: "agent", ID: "codex"},
		Content: imageContent,
	}); err != nil {
		t.Fatal(err)
	}
	appendReportILPending(t, ctx, store, missionID, pendingID)

	req, _ := reportILBundleRequest(t, missionID, pendingID, "evt_il_repeated_image_store", "evt_il_repeated_image_terminal")
	imageSHA := sha256.Sum256(imageContent)
	imageHash := hex.EncodeToString(imageSHA[:])
	req.Artifacts = append(req.Artifacts, app.CreateRawArtifactRequest{
		ArtifactID: "art_report_image", MissionID: missionID, MediaType: "image/jpeg",
		Filename: "report-image-1.jpg", Producer: app.Producer{Type: "agent", ID: "codex"},
		Content: imageContent, ExpectedSHA256: imageHash,
	})
	var payload map[string]any
	if err := json.Unmarshal(req.Terminal.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	bundle := payload["artifact_bundle"].(map[string]any)
	entries := bundle["artifacts"].([]any)
	bundle["artifacts"] = append(entries, map[string]any{
		"artifact_id": "art_report_image", "kind": "image", "media_type": "image/jpeg",
		"sha256": imageHash, "byte_size": len(imageContent), "role": "asset",
		"filename": "report-image-1.jpg", "asset_id": "asset_report_image",
	})
	req.Terminal.Payload, _ = json.Marshal(payload)

	artifacts, _, created, err := app.NewService(store).CreateReportILBundleIfOpen(ctx, req)
	if err != nil || !created {
		t.Fatalf("repeated image bundle failed: created=%v err=%v", created, err)
	}
	if len(artifacts) != 8 {
		t.Fatalf("bundle artifacts = %d, want 8", len(artifacts))
	}
	if got := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_raw_artifacts WHERE mission_id = ? AND sha256 = ?`, missionID, imageHash); got != 2 {
		t.Fatalf("same-content image identities = %d, want 2", got)
	}
	var storedSHA, filename string
	if err := store.db.QueryRowContext(ctx, `SELECT sha256, filename FROM plasma_raw_artifacts WHERE artifact_id = ?`, "art_report_image").Scan(&storedSHA, &filename); err != nil {
		t.Fatal(err)
	}
	if storedSHA != imageHash || filename != "report-image-1.jpg" {
		t.Fatalf("report image identity lost: sha=%q filename=%q", storedSHA, filename)
	}
	if _, err := store.db.ExecContext(ctx, `
UPDATE plasma_raw_artifacts
SET created_at = ?
WHERE artifact_id IN (?, ?)`, existingCreatedAt.Format(time.RFC3339Nano), "art_existing_image", "art_report_image"); err != nil {
		t.Fatal(err)
	}
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	original, found, err := artifactrepo.GetRawArtifactByMissionSHA(ctx, tx, missionID, imageHash)
	if err != nil {
		t.Fatal(err)
	}
	if !found || original.ArtifactID != "art_existing_image" {
		t.Fatalf("hash dedup lookup = found %v artifact %q", found, original.ArtifactID)
	}
	var role, ownership, sourceEventID string
	if err := store.db.QueryRowContext(ctx, `
SELECT artifact_role, ownership, source_event_id
FROM plasma_report_run_artifacts
WHERE run_id = ? AND artifact_id = ?`, pendingID, "art_report_image").Scan(&role, &ownership, &sourceEventID); err != nil {
		t.Fatal(err)
	}
	if role != "intermediate" || ownership != "created" || sourceEventID != "evt_il_repeated_image_terminal" {
		t.Fatalf("report image membership = role %q ownership %q source %q", role, ownership, sourceEventID)
	}
}

func TestReportILAtomicBundleRollsBackOnTerminalEventIDConflict(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	missionID := "mis_il_atomic_conflict"
	pendingID := "evt_il_conflict_pending"
	if err := store.CreateMission(ctx, app.Mission{MissionID: missionID, Title: "IL conflict"}); err != nil {
		t.Fatal(err)
	}
	appendReportILPending(t, ctx, store, missionID, pendingID)
	if _, err := store.AppendLedgerEvent(ctx, app.LedgerEvent{EventID: "evt_il_conflict_terminal", MissionID: missionID, EventType: "mission.note", Producer: app.Producer{Type: "test", ID: "conflict"}, Payload: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	req, _ := reportILBundleRequest(t, missionID, pendingID, "evt_il_conflict_store", "evt_il_conflict_terminal")
	_, _, created, err := app.NewService(store).CreateReportILBundleIfOpen(ctx, req)
	if err == nil || created {
		t.Fatalf("expected terminal conflict rollback, created=%v err=%v", created, err)
	}
	if got := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_ledger_events WHERE event_id = ?`, "evt_il_conflict_store"); got != 0 {
		t.Fatalf("store-completed event count = %d", got)
	}
	if got := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_raw_artifacts WHERE mission_id = ?`, missionID); got != 0 {
		t.Fatalf("new raw artifacts remained: %d", got)
	}
	if got := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_report_run_artifacts WHERE run_id = ? AND source_event_id = ?`, pendingID, "evt_il_conflict_terminal"); got != 0 {
		t.Fatalf("new memberships remained: %d", got)
	}
}

func TestReportILAtomicBundleRegistrationFailureRollsBack(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	missionID := "mis_il_atomic_purged"
	pendingID := "evt_il_purged_pending"
	if err := store.CreateMission(ctx, app.Mission{MissionID: missionID, Title: "IL purged"}); err != nil {
		t.Fatal(err)
	}
	appendReportILPending(t, ctx, store, missionID, pendingID)
	first, _ := reportILBundleRequest(t, missionID, pendingID, "evt_il_purged_store", "evt_il_purged_terminal")
	if _, _, created, err := app.NewService(store).CreateReportILBundleIfOpen(ctx, first); err != nil || !created {
		t.Fatalf("initial bundle failed: created=%v err=%v", created, err)
	}
	if _, err := store.AppendLedgerEvent(ctx, ledger.Event{EventID: "evt_report_run_completed_il_purged_pending", MissionID: missionID, EventType: "report.run.completed", Producer: ledger.Producer{Type: "system", ID: "report-completion"}, CausationEventID: "evt_il_purged_terminal", CorrelationID: pendingID, Payload: []byte(fmt.Sprintf(`{"kind":"report_run_completed","schema_version":"plasma.report_run_completion.v1","run_id":%q,"pending_event_id":%q,"canonical_event_id":"evt_il_purged_terminal","artifact_id":"art_il_atomic_markdown","delayed_usage_target_count":0,"usage_recorded_count":0,"usage_unavailable_count":0}`, pendingID, pendingID))}); err != nil {
		t.Fatal(err)
	}
	// Purge the native run through the real report-run deletion path.
	finalID := "art_il_atomic_markdown"
	facts, err := store.LoadReportRunDeleteFacts(ctx, missionID, finalID)
	if err != nil {
		t.Fatal(err)
	}
	preview := reportrun.PreviewDelete(facts, "")
	if !preview.Eligible {
		t.Fatalf("purge fixture is not eligible: %#v", preview)
	}
	if _, err := store.DeleteReportRun(ctx, missionID, finalID, preview.Revision, preview.DeleteFactsHash, func(facts reportrun.DeleteFacts) (reportrun.DeleteDecision, error) {
		return reportrun.BuildDeleteDecision(facts, "", "user", "test", time.Now().UTC()), nil
	}); err != nil {
		t.Fatal(err)
	}
	// Reinsert an experimental pending event for the purged root without
	// registration, then force a new bundle through native registration.
	insertLedgerPayloadDirect(t, ctx, store, missionID, pendingID, "report.draft.pending", `{"title":"IL","pipeline_family":"report_il_experimental"}`)
	req, _ := reportILBundleRequest(t, missionID, pendingID, "evt_il_purged_store_retry", "evt_il_purged_terminal_retry")
	if _, _, created, err := app.NewService(store).CreateReportILBundleIfOpen(ctx, req); err == nil || created {
		t.Fatalf("expected purged registration failure, created=%v err=%v", created, err)
	}
	if got := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_ledger_events WHERE event_id IN (?, ?)`, "evt_il_purged_store_retry", "evt_il_purged_terminal_retry"); got != 0 {
		t.Fatalf("new events remained: %d", got)
	}
	if got := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_raw_artifacts WHERE mission_id = ?`, missionID); got != 0 {
		t.Fatalf("new artifacts remained after registration failure: %d", got)
	}
	if got := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_report_run_artifacts WHERE run_id = ?`, pendingID); got != 0 {
		t.Fatalf("new memberships remained after registration failure: %d", got)
	}
}

func TestReportILAtomicBundleAlreadyClosedDoesNotWrite(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	missionID := "mis_il_atomic_closed"
	pendingID := "evt_il_closed_pending"
	if err := store.CreateMission(ctx, app.Mission{MissionID: missionID, Title: "IL closed"}); err != nil {
		t.Fatal(err)
	}
	appendReportILPending(t, ctx, store, missionID, pendingID)
	first, _ := reportILBundleRequest(t, missionID, pendingID, "evt_il_closed_store", "evt_il_closed_terminal")
	if _, _, created, err := app.NewService(store).CreateReportILBundleIfOpen(ctx, first); err != nil || !created {
		t.Fatalf("initial bundle failed: created=%v err=%v", created, err)
	}
	second, _ := reportILBundleRequest(t, missionID, pendingID, "evt_il_closed_store_again", "evt_il_closed_terminal_again")
	_, _, created, err := app.NewService(store).CreateReportILBundleIfOpen(ctx, second)
	if err != nil || created {
		t.Fatalf("closed bundle result = created=%v err=%v", created, err)
	}
	if got := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_ledger_events WHERE mission_id = ?`, missionID); got != 3 {
		t.Fatalf("closed mission event count = %d, want 3", got)
	}
	if got := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_raw_artifacts WHERE mission_id = ?`, missionID); got != 7 {
		t.Fatalf("closed retry changed raw artifact count: %d", got)
	}
	if got := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_report_run_artifacts WHERE run_id = ?`, pendingID); got != 7 {
		t.Fatalf("closed retry changed membership count: %d", got)
	}
	if got := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_ledger_events WHERE event_id IN (?, ?)`, "evt_il_closed_store_again", "evt_il_closed_terminal_again"); got != 0 {
		t.Fatalf("closed retry wrote terminal events: %d", got)
	}
}

func TestReportILAtomicBundleRejectsInvalidBindingsBeforeMutation(t *testing.T) {
	tests := []struct {
		name       string
		pendingFam string
		mutate     func(*app.ReportILBundleRequest)
	}{
		{name: "store mission", mutate: func(req *app.ReportILBundleRequest) { req.StoreCompleted.MissionID = "mis_other" }},
		{name: "terminal causation", mutate: func(req *app.ReportILBundleRequest) { req.Terminal.CausationEventID = "evt_other" }},
		{name: "terminal correlation", mutate: func(req *app.ReportILBundleRequest) { req.Terminal.CorrelationID = "evt_other" }},
		{name: "terminal pending payload", mutate: func(req *app.ReportILBundleRequest) {
			req.Terminal.Payload = []byte(strings.ReplaceAll(string(req.Terminal.Payload), `"pending_event_id":"evt_il_invalid_pending"`, `"pending_event_id":"evt_other"`))
		}},
		{name: "classic pending", pendingFam: "classic", mutate: func(*app.ReportILBundleRequest) {}},
		{name: "malformed receipt", mutate: func(req *app.ReportILBundleRequest) {
			req.StoreCompleted.Payload = []byte(`{"kind":"report_il_stage_progress" trailing`)
		}},
		{name: "extra receipt field", mutate: func(req *app.ReportILBundleRequest) {
			req.StoreCompleted.Payload = []byte(`{"kind":"report_il_stage_progress","pending_event_id":"evt_il_invalid_pending","pipeline_family":"report_il_experimental","stage":"il_store","status":"completed","extra":true}`)
		}},
		{name: "event id collision", mutate: func(req *app.ReportILBundleRequest) { req.Terminal.EventID = req.PendingID }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			store := newTestStore(t)
			missionID := "mis_il_invalid_" + strings.ReplaceAll(test.name, " ", "_")
			pendingID := "evt_il_invalid_pending"
			if err := store.CreateMission(ctx, app.Mission{MissionID: missionID, Title: "IL invalid"}); err != nil {
				t.Fatal(err)
			}
			family := "report_il_experimental"
			if test.pendingFam == "classic" {
				family = "classic"
			}
			appendReportILPendingFamily(t, ctx, store, missionID, pendingID, family)
			req, _ := reportILBundleRequest(t, missionID, pendingID, "evt_il_invalid_store", "evt_il_invalid_terminal")
			test.mutate(&req)
			if _, _, created, err := app.NewService(store).CreateReportILBundleIfOpen(ctx, req); err == nil || created {
				t.Fatalf("invalid bundle result = created=%v err=%v", created, err)
			}
			if got := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_ledger_events WHERE mission_id = ?`, missionID); got != 1 {
				t.Fatalf("ledger event count = %d, want 1", got)
			}
			if got := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_raw_artifacts WHERE mission_id = ?`, missionID); got != 0 {
				t.Fatalf("raw artifact count = %d, want 0", got)
			}
			if got := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_report_run_artifacts WHERE mission_id = ?`, missionID); got != 0 {
				t.Fatalf("artifact membership count = %d, want 0", got)
			}
		})
	}
}

func TestReportILAtomicBundleRollsBackDuplicateRawArtifact(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	missionID := "mis_il_atomic_duplicate_artifact"
	pendingID := "evt_il_duplicate_pending"
	if err := store.CreateMission(ctx, app.Mission{MissionID: missionID, Title: "IL duplicate"}); err != nil {
		t.Fatal(err)
	}
	if _, err := app.NewService(store).CreateRawArtifact(ctx, app.CreateRawArtifactRequest{ArtifactID: "art_il_atomic_narrative", MissionID: missionID, MediaType: "application/json", Filename: "existing.json", Producer: app.Producer{Type: "agent", ID: "codex"}, Content: []byte("existing")}); err != nil {
		t.Fatal(err)
	}
	appendReportILPending(t, ctx, store, missionID, pendingID)
	req, _ := reportILBundleRequest(t, missionID, pendingID, "evt_il_duplicate_store", "evt_il_duplicate_terminal")
	if _, _, created, err := app.NewService(store).CreateReportILBundleIfOpen(ctx, req); err == nil || created {
		t.Fatalf("expected duplicate artifact rollback, created=%v err=%v", created, err)
	}
	if got := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_ledger_events WHERE event_id IN (?, ?)`, "evt_il_duplicate_store", "evt_il_duplicate_terminal"); got != 0 {
		t.Fatalf("new events remained: %d", got)
	}
	if got := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_raw_artifacts WHERE mission_id = ?`, missionID); got != 1 {
		t.Fatalf("raw artifact count = %d, want existing artifact only", got)
	}
	if got := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_report_run_artifacts WHERE mission_id = ?`, missionID); got != 0 {
		t.Fatalf("artifact memberships remained: %d", got)
	}
}

func appendReportILPending(t *testing.T, ctx context.Context, store *Store, missionID, pendingID string) {
	appendReportILPendingFamily(t, ctx, store, missionID, pendingID, "report_il_experimental")
}

func appendReportILPendingFamily(t *testing.T, ctx context.Context, store *Store, missionID, pendingID, family string) {
	t.Helper()
	payload := fmt.Sprintf(`{"title":"IL","pipeline_family":%q}`, family)
	if _, err := store.AppendLedgerEvent(ctx, app.LedgerEvent{EventID: pendingID, MissionID: missionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "agent", ID: "codex"}, Payload: []byte(payload)}); err != nil {
		t.Fatal(err)
	}
}

type reportILExpectedArtifact struct {
	ArtifactID string
	Role       string
}

func reportILBundleRequest(t *testing.T, missionID, pendingID, storeEventID, terminalEventID string) (app.ReportILBundleRequest, []reportILExpectedArtifact) {
	t.Helper()
	types := []struct {
		kind, id, media, filename, role string
	}{
		{"narrative", "art_il_atomic_narrative", "application/json", "narrative.json", "intermediate"},
		{"semantic_il", "art_il_atomic_semantic", "application/json", "semantic-il.json", "intermediate"},
		{"flow_attestation", "art_il_atomic_flow", "application/json", "flow-attestation.json", "intermediate"},
		{"markdown", "art_il_atomic_markdown", "text/markdown; charset=utf-8", "report.md", "final"},
		{"html", "art_il_atomic_html", "text/html; charset=utf-8", "report.html", "derivative"},
		{"pdf", "art_il_atomic_pdf", "application/pdf", "report.pdf", "derivative"},
		{"manifest", "art_il_atomic_manifest", "application/json", "manifest.json", "intermediate"},
	}
	artifacts := make([]app.CreateRawArtifactRequest, 0, len(types))
	entries := make([]map[string]any, 0, len(types))
	expected := make([]reportILExpectedArtifact, 0, len(types))
	for _, item := range types {
		content := []byte("content-" + item.kind)
		sum := sha256.Sum256(content)
		sha := hex.EncodeToString(sum[:])
		artifacts = append(artifacts, app.CreateRawArtifactRequest{ArtifactID: item.id, MissionID: missionID, MediaType: item.media, Filename: item.filename, Producer: app.Producer{Type: "agent", ID: "codex"}, Content: content, ExpectedSHA256: sha})
		entries = append(entries, map[string]any{"artifact_id": item.id, "kind": item.kind, "media_type": item.media, "sha256": sha, "byte_size": len(content), "role": item.role, "filename": item.filename})
		expected = append(expected, reportILExpectedArtifact{ArtifactID: item.id, Role: item.role})
	}
	terminalPayload, err := json.Marshal(map[string]any{"kind": "markdown_report_artifact", "pending_event_id": pendingID, "pipeline_family": reportilcontract.PipelineFamily, "artifact_id": "art_il_atomic_markdown", "artifact_bundle": map[string]any{"pipeline_family": reportilcontract.PipelineFamily, "markdown_artifact_id": "art_il_atomic_markdown", "artifacts": entries}})
	if err != nil {
		t.Fatal(err)
	}
	storePayload := []byte(fmt.Sprintf(`{"kind":"report_il_stage_progress","pending_event_id":%q,"pipeline_family":%q,"stage":"il_store","status":"completed"}`, pendingID, reportilcontract.PipelineFamily))
	return app.ReportILBundleRequest{MissionID: missionID, PendingID: pendingID, Artifacts: artifacts, StoreCompleted: app.AppendEventRequest{EventID: storeEventID, MissionID: missionID, EventType: "report.il_store.completed", CausationEventID: pendingID, CorrelationID: pendingID, Producer: app.Producer{Type: "system", ID: "report-il"}, Payload: storePayload}, Terminal: app.AppendEventRequest{EventID: terminalEventID, MissionID: missionID, EventType: "report.artifact.created", CausationEventID: pendingID, CorrelationID: pendingID, Producer: app.Producer{Type: "agent", ID: "codex"}, Payload: terminalPayload}}, expected
}

func mustListEvents(t *testing.T, store *Store, missionID string) []app.LedgerEvent {
	t.Helper()
	events, err := store.ListLedgerEvents(context.Background(), missionID)
	if err != nil {
		t.Fatal(err)
	}
	return events
}
