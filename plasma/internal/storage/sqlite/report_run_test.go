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

func TestReportRunNativeMembershipAndDeletePurgesOwnedArtifacts(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	final := seedCompletedReportRun(t, ctx, store, "mis_rr_delete", "evt_rr_pending", "evt_rr_final", "art_rr_final")

	facts, err := store.LoadReportRunDeleteFacts(ctx, final.MissionID, final.ArtifactID)
	if err != nil {
		t.Fatalf("LoadReportRunDeleteFacts returned error: %v", err)
	}
	preview := reportrun.PreviewDelete(facts, "")
	if !preview.Eligible || preview.DeletableEventCount != 2 ||
		preview.DeletableArtifactCount != 1 || preview.DeletableArtifactBytes != final.ByteSize {
		t.Fatalf("unexpected preview: %#v", preview)
	}
	if preview.Usage.UsageRecordCount != 1 || preview.Usage.TotalTokens != 9 {
		t.Fatalf("unexpected usage aggregate: %#v", preview.Usage)
	}

	deleted, err := store.DeleteReportRun(ctx, final.MissionID, final.ArtifactID, preview.Revision, preview.DeleteFactsHash, func(facts reportrun.DeleteFacts) (reportrun.DeleteDecision, error) {
		return reportrun.BuildDeleteDecision(facts, "", "user", "test", time.Now().UTC()), nil
	})
	if err != nil {
		t.Fatalf("DeleteReportRun returned error: %v", err)
	}
	if !deleted.Eligible || deleted.RunID != "evt_rr_pending" {
		t.Fatalf("unexpected delete preview: %#v", deleted)
	}
	if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_ledger_events WHERE mission_id = ?`, final.MissionID) != 0 {
		t.Fatal("report ledger events remained")
	}
	if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_raw_artifacts WHERE artifact_id = ?`, final.ArtifactID) != 0 {
		t.Fatal("owned final artifact remained")
	}
	var state, title, finalArtifactID, aggregationVersion string
	var usageRecordCount, totalTokens int64
	if err := store.db.QueryRowContext(ctx, `
SELECT lifecycle_state, title, final_artifact_id, usage_record_count, total_tokens, aggregation_version
FROM plasma_report_runs WHERE run_id = ?`, "evt_rr_pending").Scan(
		&state, &title, &finalArtifactID, &usageRecordCount, &totalTokens, &aggregationVersion); err != nil {
		t.Fatalf("load tombstone: %v", err)
	}
	if state != reportrun.LifecyclePurged || title != "" || finalArtifactID != "" ||
		usageRecordCount != 1 || totalTokens != 9 || aggregationVersion != reportrun.UsageAggregationVersion {
		t.Fatalf("unexpected tombstone: state=%q title=%q final=%q count=%d total=%d version=%q",
			state, title, finalArtifactID, usageRecordCount, totalTokens, aggregationVersion)
	}
}

func TestReportRunArtifactMembershipPromotesIntermediateToFinal(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	missionID := "mis_rr_promote"
	pendingID := "evt_rr_promote_pending"
	artifactID := "art_rr_promoted"
	if err := store.CreateMission(ctx, app.Mission{MissionID: missionID, Title: missionID}); err != nil {
		t.Fatalf("CreateMission returned error: %v", err)
	}
	createStoredArtifact(t, ctx, store, missionID, artifactID, []byte("promoted final"))
	appendLedgerEvent(t, ctx, store, app.LedgerEvent{
		EventID:   pendingID,
		MissionID: missionID,
		EventType: "report.draft.pending",
		Producer:  app.Producer{Type: "user", ID: "test"},
		Payload:   []byte(`{"title":"Promoted report"}`),
	})
	appendLedgerEvent(t, ctx, store, app.LedgerEvent{
		EventID:   "evt_rr_patch_finalized",
		MissionID: missionID,
		EventType: "report.patch.finalized",
		Producer:  app.Producer{Type: "agent", ID: "test"},
		Payload:   []byte(`{"pending_event_id":"` + pendingID + `","artifact_id":"` + artifactID + `"}`),
	})
	assertArtifactMembership(t, ctx, store, pendingID, artifactID, reportrun.ArtifactRoleIntermediate, reportrun.OwnershipCreated)

	appendLedgerEvent(t, ctx, store, app.LedgerEvent{
		EventID:   "evt_rr_promoted_final",
		MissionID: missionID,
		EventType: "report.artifact.created",
		Producer:  app.Producer{Type: "agent", ID: "test"},
		Payload:   []byte(`{"pending_event_id":"` + pendingID + `","artifact_id":"` + artifactID + `"}`),
	})
	assertArtifactMembership(t, ctx, store, pendingID, artifactID, reportrun.ArtifactRoleFinal, reportrun.OwnershipCreated)

	facts, err := store.LoadReportRunDeleteFacts(ctx, missionID, artifactID)
	if err != nil {
		t.Fatalf("LoadReportRunDeleteFacts on promoted final returned error: %v", err)
	}
	preview := reportrun.PreviewDelete(facts, "")
	if !preview.Eligible || preview.DeletableEventCount != 3 || preview.DeletableArtifactCount != 1 {
		t.Fatalf("unexpected promoted preview: %#v", preview)
	}
	if _, err := store.DeleteReportRun(ctx, missionID, artifactID, preview.Revision, preview.DeleteFactsHash, func(facts reportrun.DeleteFacts) (reportrun.DeleteDecision, error) {
		return reportrun.BuildDeleteDecision(facts, "", "user", "test", time.Now().UTC()), nil
	}); err != nil {
		t.Fatalf("DeleteReportRun promoted final returned error: %v", err)
	}
	if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_ledger_events WHERE mission_id = ?`, missionID) != 0 {
		t.Fatal("promoted lineage ledger events remained")
	}
	if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_raw_artifacts WHERE artifact_id = ?`, artifactID) != 0 {
		t.Fatal("promoted final artifact remained")
	}
}

func TestReportRunBackfillSkipsPurgedRunMemberships(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	final := seedCompletedReportRun(t, ctx, store, "mis_rr_purged_backfill", "evt_rr_purged_backfill_pending", "evt_rr_purged_backfill_final", "art_rr_purged_backfill_final")
	if err := store.CreateSourceSnapshot(ctx, app.SourceSnapshot{
		SnapshotID:  "src_rr_purged_backfill",
		MissionID:   final.MissionID,
		Connector:   app.ConnectorRef{ConnectorID: "test", ConnectorType: app.SourceConnectorTypeFileUpload},
		Title:       "Preserved final",
		CapturedAt:  time.Now().UTC(),
		ArtifactIDs: []string{final.ArtifactID},
		ContentHash: app.ContentHash{Algorithm: "sha256", Value: final.SHA256},
		Locators:    []byte(`[]`),
		Access:      app.SourceAccess{Visibility: "private", License: "unknown", RetrievalPolicy: app.SourceRetrievalPolicySnapshotOnly},
	}); err != nil {
		t.Fatalf("CreateSourceSnapshot returned error: %v", err)
	}
	facts, err := store.LoadReportRunDeleteFacts(ctx, final.MissionID, final.ArtifactID)
	if err != nil {
		t.Fatalf("LoadReportRunDeleteFacts returned error: %v", err)
	}
	preview := reportrun.PreviewDelete(facts, "")
	if preview.SharedArtifactCount != 1 || preview.DeletableArtifactCount != 0 {
		t.Fatalf("source snapshot should preserve final artifact: %#v", preview)
	}
	if _, err := store.DeleteReportRun(ctx, final.MissionID, final.ArtifactID, preview.Revision, preview.DeleteFactsHash, func(facts reportrun.DeleteFacts) (reportrun.DeleteDecision, error) {
		return reportrun.BuildDeleteDecision(facts, "", "user", "test", time.Now().UTC()), nil
	}); err != nil {
		t.Fatalf("DeleteReportRun returned error: %v", err)
	}
	var tombstoneRevision int64
	var tombstoneState, tombstoneTitle, tombstoneFinal string
	if err := store.db.QueryRowContext(ctx, `
SELECT lifecycle_state, revision, title, final_artifact_id
FROM plasma_report_runs
WHERE run_id = ?`, "evt_rr_purged_backfill_pending").Scan(
		&tombstoneState, &tombstoneRevision, &tombstoneTitle, &tombstoneFinal,
	); err != nil {
		t.Fatalf("load purged tombstone: %v", err)
	}
	now := time.Now().UTC()
	memberEventID := "evt_rr_purged_backfill_member"
	insertLedgerPayloadDirect(
		t, ctx, store, final.MissionID, memberEventID, "report.patch.finalized",
		`{"pending_event_id":"evt_rr_purged_backfill_pending","artifact_id":"`+final.ArtifactID+`"}`,
	)
	if err := store.BackfillReportRuns(ctx, reportrun.Registration{
		Runs: []reportrun.Run{{
			RunID: "evt_rr_purged_backfill_pending", MissionID: final.MissionID, RootPendingEventID: "evt_rr_purged_backfill_pending",
			LifecycleState: reportrun.LifecycleCompleted, FinalArtifactID: final.ArtifactID, RegistrationStatus: reportrun.RegistrationBackfilled,
			Usage: reportrun.UsageAggregate{AggregationVersion: reportrun.UsageAggregationVersion}, CreatedAt: now, UpdatedAt: now,
		}},
		Events: []reportrun.EventMembership{{
			RunID: "evt_rr_purged_backfill_pending", EventID: memberEventID, MissionID: final.MissionID,
			EventRole: "stage", AttemptEventID: "evt_rr_purged_backfill_pending", CreatedAt: now,
		}},
		Artifacts: []reportrun.ArtifactMembership{{
			RunID: "evt_rr_purged_backfill_pending", ArtifactID: final.ArtifactID, MissionID: final.MissionID,
			ArtifactRole: reportrun.ArtifactRoleFinal, Ownership: reportrun.OwnershipCreated, CreatedAt: now,
		}},
	}); err != nil {
		t.Fatalf("BackfillReportRuns returned error: %v", err)
	}
	if got := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_report_run_events WHERE run_id = ?`, "evt_rr_purged_backfill_pending"); got != 0 {
		t.Fatalf("purged run event membership reappeared: %d", got)
	}
	if got := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_report_run_artifacts WHERE run_id = ?`, "evt_rr_purged_backfill_pending"); got != 0 {
		t.Fatalf("purged run artifact membership reappeared: %d", got)
	}
	var afterRevision int64
	var afterState, afterTitle, afterFinal string
	if err := store.db.QueryRowContext(ctx, `
SELECT lifecycle_state, revision, title, final_artifact_id
FROM plasma_report_runs
WHERE run_id = ?`, "evt_rr_purged_backfill_pending").Scan(
		&afterState, &afterRevision, &afterTitle, &afterFinal,
	); err != nil {
		t.Fatalf("reload purged tombstone: %v", err)
	}
	if afterState != tombstoneState || afterRevision != tombstoneRevision || afterTitle != tombstoneTitle || afterFinal != tombstoneFinal {
		t.Fatalf("purged tombstone changed: state %q->%q revision %d->%d title %q->%q final %q->%q",
			tombstoneState, afterState, tombstoneRevision, afterRevision, tombstoneTitle, afterTitle, tombstoneFinal, afterFinal)
	}
}

func TestReportRunNativeRegistrationRejectsPurgedRootReuse(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	final := seedCompletedReportRun(t, ctx, store, "mis_rr_purged_native", "evt_rr_purged_native_pending", "evt_rr_purged_native_final", "art_rr_purged_native_final")
	facts, err := store.LoadReportRunDeleteFacts(ctx, final.MissionID, final.ArtifactID)
	if err != nil {
		t.Fatalf("LoadReportRunDeleteFacts returned error: %v", err)
	}
	preview := reportrun.PreviewDelete(facts, "")
	if _, err := store.DeleteReportRun(ctx, final.MissionID, final.ArtifactID, preview.Revision, preview.DeleteFactsHash, func(facts reportrun.DeleteFacts) (reportrun.DeleteDecision, error) {
		return reportrun.BuildDeleteDecision(facts, "", "user", "test", time.Now().UTC()), nil
	}); err != nil {
		t.Fatalf("DeleteReportRun returned error: %v", err)
	}
	svc := app.NewService(store)
	_, _, err = svc.CreateRawArtifactWithEvent(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_rr_purged_native_reuse",
		MissionID:  final.MissionID,
		MediaType:  "text/markdown",
		Filename:   "reuse.md",
		Producer:   app.Producer{Type: "agent", ID: "test"},
		Content:    []byte("reuse"),
	}, func(artifact app.RawArtifact) app.AppendEventRequest {
		return app.AppendEventRequest{
			EventID:   "evt_rr_purged_native_reuse",
			MissionID: final.MissionID,
			EventType: "report.artifact.created",
			Producer:  app.Producer{Type: "agent", ID: "test"},
			Payload:   []byte(`{"pending_event_id":"evt_rr_purged_native_pending","artifact_id":"` + artifact.ArtifactID + `"}`),
		}
	})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("expected purged root reuse conflict, got %v", err)
	}
	if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_ledger_events WHERE event_id = ?`, "evt_rr_purged_native_reuse") != 0 {
		t.Fatal("native reuse event was not rolled back")
	}
	if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_raw_artifacts WHERE artifact_id = ?`, "art_rr_purged_native_reuse") != 0 {
		t.Fatal("native reuse artifact was not rolled back")
	}
}

func TestReportRunDeleteRevisionConflictAndSharedArtifactPreservation(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	final := seedCompletedReportRun(t, ctx, store, "mis_rr_shared", "evt_rr_shared_pending", "evt_rr_shared_final", "art_rr_shared_final")
	appendLedgerEvent(t, ctx, store, app.LedgerEvent{
		EventID:   "evt_rr_shared_reference",
		MissionID: final.MissionID,
		EventType: "mission.note",
		Producer:  app.Producer{Type: "user", ID: "test"},
		Payload:   []byte(`{"artifact_id":"art_rr_shared_final"}`),
	})

	facts, err := store.LoadReportRunDeleteFacts(ctx, final.MissionID, final.ArtifactID)
	if err != nil {
		t.Fatalf("LoadReportRunDeleteFacts returned error: %v", err)
	}
	preview := reportrun.PreviewDelete(facts, "")
	if preview.SharedArtifactCount != 1 || preview.DeletableArtifactCount != 0 {
		t.Fatalf("expected shared artifact preservation, got %#v", preview)
	}
	_, err = store.DeleteReportRun(ctx, final.MissionID, final.ArtifactID, preview.Revision+1, preview.DeleteFactsHash, func(facts reportrun.DeleteFacts) (reportrun.DeleteDecision, error) {
		return reportrun.BuildDeleteDecision(facts, "", "user", "test", time.Now().UTC()), nil
	})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("expected revision conflict, got %v", err)
	}
	if _, err := store.DeleteReportRun(ctx, final.MissionID, final.ArtifactID, preview.Revision, preview.DeleteFactsHash, func(facts reportrun.DeleteFacts) (reportrun.DeleteDecision, error) {
		return reportrun.BuildDeleteDecision(facts, "", "user", "test", time.Now().UTC()), nil
	}); err != nil {
		t.Fatalf("DeleteReportRun returned error: %v", err)
	}
	if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_raw_artifacts WHERE artifact_id = ?`, final.ArtifactID) != 1 {
		t.Fatal("shared artifact was removed")
	}
	if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_ledger_events WHERE event_id = ?`, "evt_rr_shared_reference") != 1 {
		t.Fatal("out-of-run ledger reference was removed")
	}
}

func TestReportRunCrossMissionLedgerReferencePreservesArtifact(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	final := seedCompletedReportRun(t, ctx, store, "mis_rr_cross_ref", "evt_rr_cross_pending", "evt_rr_cross_final", "art_rr_cross_final")
	if err := store.CreateMission(ctx, app.Mission{MissionID: "mis_rr_cross_ref_other", Title: "Other"}); err != nil {
		t.Fatalf("CreateMission other returned error: %v", err)
	}
	appendLedgerEvent(t, ctx, store, app.LedgerEvent{
		EventID:   "evt_rr_cross_reference",
		MissionID: "mis_rr_cross_ref_other",
		EventType: "mission.note",
		Producer:  app.Producer{Type: "user", ID: "test"},
		Payload:   []byte(`{"artifact_id":"` + final.ArtifactID + `"}`),
	})

	facts, err := store.LoadReportRunDeleteFacts(ctx, final.MissionID, final.ArtifactID)
	if err != nil {
		t.Fatalf("LoadReportRunDeleteFacts returned error: %v", err)
	}
	preview := reportrun.PreviewDelete(facts, "")
	if preview.SharedArtifactCount != 1 || preview.DeletableArtifactCount != 0 {
		t.Fatalf("expected cross-mission shared artifact preservation, got %#v", preview)
	}
	if _, err := store.DeleteReportRun(ctx, final.MissionID, final.ArtifactID, preview.Revision, preview.DeleteFactsHash, func(facts reportrun.DeleteFacts) (reportrun.DeleteDecision, error) {
		return reportrun.BuildDeleteDecision(facts, "", "user", "test", time.Now().UTC()), nil
	}); err != nil {
		t.Fatalf("DeleteReportRun returned error: %v", err)
	}
	if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_raw_artifacts WHERE artifact_id = ?`, final.ArtifactID) != 1 {
		t.Fatal("cross-mission referenced artifact was removed")
	}
	if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_ledger_events WHERE event_id = ?`, "evt_rr_cross_reference") != 1 {
		t.Fatal("cross-mission reference event was removed")
	}
}

func TestReportRunDeleteFactsHashConflictsWhenExternalFactsChange(t *testing.T) {
	t.Run("source snapshot link added", func(t *testing.T) {
		ctx := context.Background()
		store := newTestStore(t)
		final := seedCompletedReportRun(t, ctx, store, "mis_rr_hash_source", "evt_rr_hash_source_pending", "evt_rr_hash_source_final", "art_rr_hash_source_final")
		before, err := store.LoadReportRunDeleteFacts(ctx, final.MissionID, final.ArtifactID)
		if err != nil {
			t.Fatalf("LoadReportRunDeleteFacts returned error: %v", err)
		}
		if err := store.CreateSourceSnapshot(ctx, app.SourceSnapshot{
			SnapshotID:  "src_rr_hash_source",
			MissionID:   final.MissionID,
			Connector:   app.ConnectorRef{ConnectorID: "test", ConnectorType: app.SourceConnectorTypeFileUpload},
			Title:       "Shared final",
			CapturedAt:  time.Now().UTC(),
			ArtifactIDs: []string{final.ArtifactID},
			ContentHash: app.ContentHash{Algorithm: "sha256", Value: final.SHA256},
			Locators:    []byte(`[]`),
			Access:      app.SourceAccess{Visibility: "private", License: "unknown", RetrievalPolicy: app.SourceRetrievalPolicySnapshotOnly},
		}); err != nil {
			t.Fatalf("CreateSourceSnapshot returned error: %v", err)
		}
		assertRevisionUnchanged(t, ctx, store, before)
		_, err = store.DeleteReportRun(ctx, final.MissionID, final.ArtifactID, before.Run.Revision, reportrun.DeleteFactsHash(before), func(facts reportrun.DeleteFacts) (reportrun.DeleteDecision, error) {
			return reportrun.BuildDeleteDecision(facts, "", "user", "test", time.Now().UTC()), nil
		})
		if !errors.Is(err, app.ErrConflict) {
			t.Fatalf("expected facts hash conflict, got %v", err)
		}
	})
	t.Run("other live report run membership added", func(t *testing.T) {
		ctx := context.Background()
		store := newTestStore(t)
		final := seedCompletedReportRun(t, ctx, store, "mis_rr_hash_other_run", "evt_rr_hash_other_pending", "evt_rr_hash_other_final", "art_rr_hash_other_final")
		svc := app.NewService(store)
		workcopy, err := svc.SaveReportRedpenWorkcopy(ctx, app.SaveReportRedpenRequest{
			EventID:          "evt_rr_hash_other_redpen",
			ArtifactID:       "art_rr_hash_other_redpen",
			NewWorkcopyID:    "rwc_rr_hash_other",
			MissionID:        final.MissionID,
			SourceArtifactID: final.ArtifactID,
			Producer:         app.Producer{Type: "user", ID: "test"},
			Content:          []byte("redpen shared later"),
		})
		if err != nil {
			t.Fatalf("SaveReportRedpenWorkcopy returned error: %v", err)
		}
		before, err := store.LoadReportRunDeleteFacts(ctx, final.MissionID, final.ArtifactID)
		if err != nil {
			t.Fatalf("LoadReportRunDeleteFacts returned error: %v", err)
		}
		now := time.Now().UTC()
		if err := store.BackfillReportRuns(ctx, reportrun.Registration{
			Runs: []reportrun.Run{{
				RunID: "evt_rr_hash_other_owner", MissionID: final.MissionID, RootPendingEventID: "evt_rr_hash_other_owner",
				LifecycleState: reportrun.LifecycleCompleted, FinalArtifactID: workcopy.Artifact.ArtifactID, RegistrationStatus: reportrun.RegistrationBackfilled,
				Usage: reportrun.UsageAggregate{AggregationVersion: reportrun.UsageAggregationVersion}, CreatedAt: now, UpdatedAt: now,
			}},
			Artifacts: []reportrun.ArtifactMembership{{
				RunID: "evt_rr_hash_other_owner", ArtifactID: workcopy.Artifact.ArtifactID, MissionID: final.MissionID,
				ArtifactRole: reportrun.ArtifactRoleIntermediate, Ownership: reportrun.OwnershipCreated, CreatedAt: now,
			}},
		}); err != nil {
			t.Fatalf("BackfillReportRuns returned error: %v", err)
		}
		assertRevisionUnchanged(t, ctx, store, before)
		_, err = store.DeleteReportRun(ctx, final.MissionID, final.ArtifactID, before.Run.Revision, reportrun.DeleteFactsHash(before), func(facts reportrun.DeleteFacts) (reportrun.DeleteDecision, error) {
			return reportrun.BuildDeleteDecision(facts, "", "user", "test", time.Now().UTC()), nil
		})
		if !errors.Is(err, app.ErrConflict) {
			t.Fatalf("expected facts hash conflict, got %v", err)
		}
	})
	t.Run("external ledger reference removed", func(t *testing.T) {
		ctx := context.Background()
		store := newTestStore(t)
		final := seedCompletedReportRun(t, ctx, store, "mis_rr_hash_ledger", "evt_rr_hash_ledger_pending", "evt_rr_hash_ledger_final", "art_rr_hash_ledger_final")
		appendLedgerEvent(t, ctx, store, app.LedgerEvent{
			EventID:   "evt_rr_hash_ledger_reference",
			MissionID: final.MissionID,
			EventType: "mission.note",
			Producer:  app.Producer{Type: "user", ID: "test"},
			Payload:   []byte(`{"artifact_id":"` + final.ArtifactID + `"}`),
		})
		before, err := store.LoadReportRunDeleteFacts(ctx, final.MissionID, final.ArtifactID)
		if err != nil {
			t.Fatalf("LoadReportRunDeleteFacts returned error: %v", err)
		}
		if reportrun.PreviewDelete(before, "").SharedArtifactCount != 1 {
			t.Fatalf("ledger reference should make artifact shared before mutation")
		}
		if _, err := store.db.ExecContext(ctx, `DELETE FROM plasma_ledger_events WHERE event_id = ?`, "evt_rr_hash_ledger_reference"); err != nil {
			t.Fatalf("delete external ledger reference: %v", err)
		}
		assertRevisionUnchanged(t, ctx, store, before)
		_, err = store.DeleteReportRun(ctx, final.MissionID, final.ArtifactID, before.Run.Revision, reportrun.DeleteFactsHash(before), func(facts reportrun.DeleteFacts) (reportrun.DeleteDecision, error) {
			return reportrun.BuildDeleteDecision(facts, "", "user", "test", time.Now().UTC()), nil
		})
		if !errors.Is(err, app.ErrConflict) {
			t.Fatalf("expected facts hash conflict, got %v", err)
		}
	})
}

func TestReportRunMalformedOutOfRunPayloadRelevance(t *testing.T) {
	t.Run("unrelated malformed payload blocks", func(t *testing.T) {
		ctx := context.Background()
		store := newTestStore(t)
		final := seedCompletedReportRun(t, ctx, store, "mis_rr_unrelated_malformed", "evt_rr_unrelated_pending", "evt_rr_unrelated_final", "art_rr_unrelated_final")
		insertLedgerPayloadDirect(t, ctx, store, final.MissionID, "evt_rr_unrelated_malformed", "mission.note", `{"artifact_id":`)

		facts, err := store.LoadReportRunDeleteFacts(ctx, final.MissionID, final.ArtifactID)
		if err != nil {
			t.Fatalf("LoadReportRunDeleteFacts returned error: %v", err)
		}
		preview := reportrun.PreviewDelete(facts, "")
		if preview.Eligible || !hasDeleteBlocker(preview, reportrun.BlockerMalformedReference) {
			t.Fatalf("unrelated malformed payload should block: %#v", preview)
		}
	})
	t.Run("candidate-looking malformed payload blocks", func(t *testing.T) {
		ctx := context.Background()
		store := newTestStore(t)
		final := seedCompletedReportRun(t, ctx, store, "mis_rr_relevant_malformed", "evt_rr_relevant_pending", "evt_rr_relevant_final", "art_rr_relevant_final")
		insertLedgerPayloadDirect(t, ctx, store, final.MissionID, "evt_rr_relevant_malformed", "mission.note", `{"artifact_id":"`+final.ArtifactID+`"`)

		facts, err := store.LoadReportRunDeleteFacts(ctx, final.MissionID, final.ArtifactID)
		if err != nil {
			t.Fatalf("LoadReportRunDeleteFacts returned error: %v", err)
		}
		preview := reportrun.PreviewDelete(facts, "")
		if preview.Eligible || !hasDeleteBlocker(preview, reportrun.BlockerMalformedReference) {
			t.Fatalf("candidate-looking malformed payload should block: %#v", preview)
		}
	})
	t.Run("unicode escaped malformed payload blocks", func(t *testing.T) {
		ctx := context.Background()
		store := newTestStore(t)
		final := seedCompletedReportRun(t, ctx, store, "mis_rr_escaped_malformed", "evt_rr_escaped_pending", "evt_rr_escaped_final", "art_rr_escaped_final")
		insertLedgerPayloadDirect(t, ctx, store, final.MissionID, "evt_rr_escaped_malformed", "mission.note", `{"artifact_id":"\u0061rt_rr_escaped_final"`)

		facts, err := store.LoadReportRunDeleteFacts(ctx, final.MissionID, final.ArtifactID)
		if err != nil {
			t.Fatalf("LoadReportRunDeleteFacts returned error: %v", err)
		}
		preview := reportrun.PreviewDelete(facts, "")
		if preview.Eligible || !hasDeleteBlocker(preview, reportrun.BlockerMalformedReference) {
			t.Fatalf("unicode escaped malformed payload should block: %#v", preview)
		}
	})
	t.Run("cross-mission malformed payload blocks", func(t *testing.T) {
		ctx := context.Background()
		store := newTestStore(t)
		final := seedCompletedReportRun(t, ctx, store, "mis_rr_cross_malformed", "evt_rr_cross_malformed_pending", "evt_rr_cross_malformed_final", "art_rr_cross_malformed_final")
		if err := store.CreateMission(ctx, app.Mission{MissionID: "mis_rr_cross_malformed_other", Title: "Other"}); err != nil {
			t.Fatalf("CreateMission other returned error: %v", err)
		}
		insertLedgerPayloadDirect(t, ctx, store, "mis_rr_cross_malformed_other", "evt_rr_cross_malformed_other", "mission.note", `{"note":`)

		facts, err := store.LoadReportRunDeleteFacts(ctx, final.MissionID, final.ArtifactID)
		if err != nil {
			t.Fatalf("LoadReportRunDeleteFacts returned error: %v", err)
		}
		preview := reportrun.PreviewDelete(facts, "")
		if preview.Eligible || !hasDeleteBlocker(preview, reportrun.BlockerMalformedReference) {
			t.Fatalf("cross-mission malformed payload should block: %#v", preview)
		}
	})
}

func TestReportRunDeleteRequiresFinalMarkdownTarget(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	final := seedCompletedReportRun(t, ctx, store, "mis_rr_target", "evt_rr_target_pending", "evt_rr_target_final", "art_rr_target_final")
	svc := app.NewService(store)
	derivative, _, err := svc.CreateRawArtifactWithEvent(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_rr_target_html",
		MissionID:  final.MissionID,
		MediaType:  "text/html; charset=utf-8",
		Filename:   "target.html",
		Producer:   app.Producer{Type: "plasma", ID: "test"},
		Content:    []byte("<html></html>"),
	}, func(artifact app.RawArtifact) app.AppendEventRequest {
		return app.AppendEventRequest{
			EventID:   "evt_rr_target_html",
			MissionID: final.MissionID,
			EventType: "report.artifact.exported",
			Producer:  app.Producer{Type: "plasma", ID: "test"},
			Payload:   []byte(`{"pending_event_id":"evt_rr_target_pending","source_artifact_id":"` + final.ArtifactID + `","artifact_id":"` + artifact.ArtifactID + `","kind":"self_contained_html","target":"self_contained_html"}`),
		}
	})
	if err != nil {
		t.Fatalf("CreateRawArtifactWithEvent derivative returned error: %v", err)
	}
	if _, err := store.LoadReportRunDeleteFacts(ctx, final.MissionID, derivative.ArtifactID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("derivative target should not load delete facts, got %v", err)
	}
	facts, err := store.LoadReportRunDeleteFacts(ctx, final.MissionID, final.ArtifactID)
	if err != nil {
		t.Fatalf("final target should load delete facts: %v", err)
	}
	preview := reportrun.PreviewDelete(facts, "")
	if !preview.Eligible || preview.DeletableArtifactCount != 2 {
		t.Fatalf("final Markdown target should delete derivative as impact: %#v", preview)
	}
}

func TestReportRunUnknownArtifactEventCannotMakeArtifactDeletable(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	missionID := "mis_rr_unknown_owner"
	artifactID := "art_rr_unknown_existing"
	if err := store.CreateMission(ctx, app.Mission{MissionID: missionID, Title: missionID}); err != nil {
		t.Fatalf("CreateMission returned error: %v", err)
	}
	createStoredArtifact(t, ctx, store, missionID, artifactID, []byte("existing"))
	appendLedgerEvent(t, ctx, store, app.LedgerEvent{
		EventID:   "evt_rr_unknown_pending",
		MissionID: missionID,
		EventType: "report.draft.pending",
		Producer:  app.Producer{Type: "user", ID: "test"},
		Payload:   []byte(`{"title":"Unknown owner"}`),
	})
	appendLedgerEvent(t, ctx, store, app.LedgerEvent{
		EventID:   "evt_rr_unknown_artifact",
		MissionID: missionID,
		EventType: "report.future.created",
		Producer:  app.Producer{Type: "agent", ID: "test"},
		Payload:   []byte(`{"pending_event_id":"evt_rr_unknown_pending","artifact_id":"` + artifactID + `"}`),
	})
	assertArtifactMembership(t, ctx, store, "evt_rr_unknown_pending", artifactID, reportrun.ArtifactRoleIntermediate, reportrun.OwnershipReferenced)
	if _, err := store.LoadReportRunDeleteFacts(ctx, missionID, artifactID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("unknown artifact event should not make artifact deletable, got %v", err)
	}
	var state string
	if err := store.db.QueryRowContext(ctx, `SELECT lifecycle_state FROM plasma_report_runs WHERE run_id = ?`, "evt_rr_unknown_pending").Scan(&state); err != nil {
		t.Fatalf("load unknown owner run: %v", err)
	}
	if state != reportrun.LifecycleAmbiguous {
		t.Fatalf("unknown owner run state = %q, want ambiguous", state)
	}
}

func TestReportRunRedpenCommitUpdatesRevisionAndDeletesWithRun(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	final := seedCompletedReportRun(t, ctx, store, "mis_rr_redpen", "evt_rr_redpen_pending", "evt_rr_redpen_final", "art_rr_redpen_final")
	before, err := store.LoadReportRunDeleteFacts(ctx, final.MissionID, final.ArtifactID)
	if err != nil {
		t.Fatalf("LoadReportRunDeleteFacts before redpen returned error: %v", err)
	}
	svc := app.NewService(store)
	workcopy, err := svc.SaveReportRedpenWorkcopy(ctx, app.SaveReportRedpenRequest{
		EventID:          "evt_rr_redpen_saved",
		ArtifactID:       "art_rr_redpen_saved",
		NewWorkcopyID:    "rwc_rr_redpen",
		MissionID:        final.MissionID,
		SourceArtifactID: final.ArtifactID,
		Producer:         app.Producer{Type: "user", ID: "test"},
		Content:          []byte("redpen final"),
	})
	if err != nil {
		t.Fatalf("SaveReportRedpenWorkcopy returned error: %v", err)
	}
	after, err := store.LoadReportRunDeleteFacts(ctx, final.MissionID, final.ArtifactID)
	if err != nil {
		t.Fatalf("LoadReportRunDeleteFacts after redpen returned error: %v", err)
	}
	if after.Run.Revision <= before.Run.Revision {
		t.Fatalf("redpen commit did not advance run revision: before=%d after=%d", before.Run.Revision, after.Run.Revision)
	}
	_, err = store.DeleteReportRun(ctx, final.MissionID, final.ArtifactID, before.Run.Revision, reportrun.DeleteFactsHash(before), func(facts reportrun.DeleteFacts) (reportrun.DeleteDecision, error) {
		return reportrun.BuildDeleteDecision(facts, "", "user", "test", time.Now().UTC()), nil
	})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("expected stale delete revision conflict, got %v", err)
	}
	preview := reportrun.PreviewDelete(after, "")
	if !preview.Eligible || preview.DeletableArtifactCount != 2 {
		t.Fatalf("redpen artifact should be deleted as run-owned impact: %#v", preview)
	}
	if _, err := store.DeleteReportRun(ctx, final.MissionID, final.ArtifactID, after.Run.Revision, reportrun.DeleteFactsHash(after), func(facts reportrun.DeleteFacts) (reportrun.DeleteDecision, error) {
		return reportrun.BuildDeleteDecision(facts, "", "user", "test", time.Now().UTC()), nil
	}); err != nil {
		t.Fatalf("DeleteReportRun after redpen returned error: %v", err)
	}
	if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_raw_artifacts WHERE artifact_id = ?`, workcopy.Artifact.ArtifactID) != 0 {
		t.Fatal("redpen artifact remained after report delete")
	}
	if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_ledger_events WHERE event_id = ?`, workcopy.Event.EventID) != 0 {
		t.Fatal("redpen event remained after report delete")
	}
}

func TestReportRunDeletePreservesReusedRedpenArtifact(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	final := seedCompletedReportRun(t, ctx, store, "mis_rr_redpen_reuse", "evt_rr_redpen_reuse_pending", "evt_rr_redpen_reuse_final", "art_rr_redpen_reuse_final")
	svc := app.NewService(store)
	reused, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_rr_redpen_generic",
		MissionID:  final.MissionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   "generic.md",
		Producer:   app.Producer{Type: "user", ID: "test"},
		Content:    []byte("# Generic\n\nReusable redpen content.\n"),
	})
	if err != nil {
		t.Fatalf("CreateRawArtifact reused target returned error: %v", err)
	}
	workcopy, err := svc.SaveReportRedpenWorkcopy(ctx, app.SaveReportRedpenRequest{
		EventID:          "evt_rr_redpen_reused_saved",
		ArtifactID:       "art_rr_redpen_should_not_insert",
		NewWorkcopyID:    "rwc_rr_redpen_reuse",
		MissionID:        final.MissionID,
		SourceArtifactID: final.ArtifactID,
		Producer:         app.Producer{Type: "user", ID: "test"},
		Content:          append([]byte(nil), reused.Content...),
	})
	if err != nil {
		t.Fatalf("SaveReportRedpenWorkcopy returned error: %v", err)
	}
	if workcopy.Artifact.ArtifactID != reused.ArtifactID {
		t.Fatalf("redpen should reuse generic artifact, got %#v want %s", workcopy.Artifact, reused.ArtifactID)
	}
	assertArtifactMembership(t, ctx, store, "evt_rr_redpen_reuse_pending", reused.ArtifactID, reportrun.ArtifactRoleIntermediate, reportrun.OwnershipReferenced)
	facts, err := store.LoadReportRunDeleteFacts(ctx, final.MissionID, final.ArtifactID)
	if err != nil {
		t.Fatalf("LoadReportRunDeleteFacts returned error: %v", err)
	}
	preview := reportrun.PreviewDelete(facts, "")
	if !preview.Eligible || preview.DeletableArtifactCount != 1 {
		t.Fatalf("reused redpen artifact must not be deletable: %#v", preview)
	}
	if _, err := store.DeleteReportRun(ctx, final.MissionID, final.ArtifactID, preview.Revision, preview.DeleteFactsHash, func(facts reportrun.DeleteFacts) (reportrun.DeleteDecision, error) {
		return reportrun.BuildDeleteDecision(facts, "", "user", "test", time.Now().UTC()), nil
	}); err != nil {
		t.Fatalf("DeleteReportRun returned error: %v", err)
	}
	if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_raw_artifacts WHERE artifact_id = ?`, reused.ArtifactID) != 1 {
		t.Fatal("reused redpen artifact was deleted")
	}
	if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_ledger_events WHERE event_id = ?`, workcopy.Event.EventID) != 0 {
		t.Fatal("redpen event remained after report delete")
	}
}

func TestReportRunInvalidRetryDoesNotAllowCrossRunDeletion(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	missionID := "mis_rr_invalid_retry_delete"
	if err := store.CreateMission(ctx, app.Mission{MissionID: missionID, Title: missionID}); err != nil {
		t.Fatalf("CreateMission returned error: %v", err)
	}
	rootA := seedCompletedReportAttempt(t, ctx, store, missionID, "evt_rr_root_a", "evt_rr_root_a_final", "art_rr_root_a")
	rootB := seedCompletedReportAttempt(t, ctx, store, missionID, "evt_rr_root_b", "evt_rr_root_b_final", "art_rr_root_b")
	appendLedgerEvent(t, ctx, store, app.LedgerEvent{
		EventID:   "evt_rr_invalid_retry",
		MissionID: missionID,
		EventType: "report.draft.pending",
		Producer:  app.Producer{Type: "user", ID: "test"},
		Payload:   []byte(`{"origin_pending_event_id":"evt_rr_root_a","retry_of_pending_event_id":"evt_rr_root_b","retry_strategy":"resume_failed"}`),
	})
	svc := app.NewService(store)
	retryArtifact, _, err := svc.CreateRawArtifactWithEvent(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_rr_invalid_retry",
		MissionID:  missionID,
		MediaType:  "text/markdown",
		Filename:   "invalid-retry.md",
		Producer:   app.Producer{Type: "agent", ID: "test"},
		Content:    []byte("invalid retry"),
	}, func(artifact app.RawArtifact) app.AppendEventRequest {
		return app.AppendEventRequest{
			EventID:   "evt_rr_invalid_retry_final",
			MissionID: missionID,
			EventType: "report.artifact.created",
			Producer:  app.Producer{Type: "agent", ID: "test"},
			Payload:   []byte(`{"pending_event_id":"evt_rr_invalid_retry","artifact_id":"` + artifact.ArtifactID + `"}`),
		}
	})
	if err != nil {
		t.Fatalf("CreateRawArtifactWithEvent invalid retry final returned error: %v", err)
	}

	facts, err := store.LoadReportRunDeleteFacts(ctx, missionID, rootA.ArtifactID)
	if err != nil {
		t.Fatalf("LoadReportRunDeleteFacts root A returned error: %v", err)
	}
	preview := reportrun.PreviewDelete(facts, "")
	if preview.Eligible || !hasDeleteBlocker(preview, reportrun.BlockerAmbiguousLineage) {
		t.Fatalf("invalid retry should make claimed root deletion ineligible: %#v", preview)
	}
	_, err = store.DeleteReportRun(ctx, missionID, rootA.ArtifactID, preview.Revision, preview.DeleteFactsHash, func(facts reportrun.DeleteFacts) (reportrun.DeleteDecision, error) {
		return reportrun.BuildDeleteDecision(facts, "", "user", "test", time.Now().UTC()), nil
	})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("expected ambiguous delete conflict, got %v", err)
	}
	for _, artifactID := range []string{rootA.ArtifactID, rootB.ArtifactID, retryArtifact.ArtifactID} {
		if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_raw_artifacts WHERE artifact_id = ?`, artifactID) != 1 {
			t.Fatalf("artifact %s should not be deleted by invalid retry lineage", artifactID)
		}
	}
}

func TestReportRunRegistrationSkipsGenericWrites(t *testing.T) {
	t.Run("plain ledger append", func(t *testing.T) {
		ctx := context.Background()
		store := newTestStore(t)
		missionID := "mis_rr_guard_ledger"
		if err := store.CreateMission(ctx, app.Mission{MissionID: missionID, Title: missionID}); err != nil {
			t.Fatalf("CreateMission returned error: %v", err)
		}
		insertLedgerPayloadDirect(t, ctx, store, missionID, "evt_rr_guard_pending", "report.draft.pending", `{"title":"guard"}`)
		appendLedgerEvent(t, ctx, store, app.LedgerEvent{
			EventID:   "evt_rr_guard_note",
			MissionID: missionID,
			EventType: "mission.note",
			Producer:  app.Producer{Type: "user", ID: "test"},
			Payload:   []byte(`{"note":"generic"}`),
		})
		if got := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_report_runs WHERE mission_id = ?`, missionID); got != 0 {
			t.Fatalf("generic ledger append rebuilt report-run rows: %d", got)
		}
	})
	t.Run("atomic write without report event", func(t *testing.T) {
		ctx := context.Background()
		store := newTestStore(t)
		missionID := "mis_rr_guard_atomic"
		if err := store.CreateMission(ctx, app.Mission{MissionID: missionID, Title: missionID}); err != nil {
			t.Fatalf("CreateMission returned error: %v", err)
		}
		insertLedgerPayloadDirect(t, ctx, store, missionID, "evt_rr_guard_atomic_pending", "report.draft.pending", `{"title":"guard"}`)
		svc := app.NewService(store)
		if _, _, err := svc.CreateRawArtifactWithEvent(ctx, app.CreateRawArtifactRequest{
			ArtifactID: "art_rr_guard_atomic",
			MissionID:  missionID,
			MediaType:  "text/plain",
			Filename:   "generic.txt",
			Producer:   app.Producer{Type: "user", ID: "test"},
			Content:    []byte("generic"),
		}, func(artifact app.RawArtifact) app.AppendEventRequest {
			return app.AppendEventRequest{
				EventID:   "evt_rr_guard_atomic_note",
				MissionID: missionID,
				EventType: "mission.note",
				Producer:  app.Producer{Type: "user", ID: "test"},
				Payload:   []byte(`{"artifact_id":"` + artifact.ArtifactID + `"}`),
			}
		}); err != nil {
			t.Fatalf("CreateRawArtifactWithEvent returned error: %v", err)
		}
		if got := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_report_runs WHERE mission_id = ?`, missionID); got != 0 {
			t.Fatalf("generic atomic write rebuilt report-run rows: %d", got)
		}
	})
}

func TestReportCanvasEventsDoNotCreateReportRuns(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	missionID := "mis_rr_canvas_events"
	if err := store.CreateMission(ctx, app.Mission{MissionID: missionID, Title: missionID}); err != nil {
		t.Fatalf("CreateMission returned error: %v", err)
	}

	appendLedgerEvent(t, ctx, store, app.LedgerEvent{
		EventID:   "evt_rr_canvas_promoted",
		MissionID: missionID,
		EventType: "report.promoted",
		Producer:  app.Producer{Type: "user", ID: "test"},
		Payload:   []byte(`{"report_id":"rpt_canvas","report_version_id":"rvn_canvas"}`),
	})
	svc := app.NewService(store)
	if _, _, err := svc.CreateRawArtifactWithEvent(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_rr_canvas_export",
		MissionID:  missionID,
		MediaType:  "text/markdown",
		Filename:   "canvas-export.md",
		Producer:   app.Producer{Type: "user", ID: "test"},
		Content:    []byte("canvas export"),
	}, func(artifact app.RawArtifact) app.AppendEventRequest {
		return app.AppendEventRequest{
			EventID:   "evt_rr_canvas_exported",
			MissionID: missionID,
			EventType: "report.exported",
			Producer:  app.Producer{Type: "user", ID: "test"},
			Payload:   []byte(`{"report_id":"rpt_canvas","report_version_id":"rvn_canvas","artifact_id":"` + artifact.ArtifactID + `"}`),
		}
	}); err != nil {
		t.Fatalf("CreateRawArtifactWithEvent returned error: %v", err)
	}
	if got := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_report_runs WHERE mission_id = ?`, missionID); got != 0 {
		t.Fatalf("report canvas events created report-run rows: %d", got)
	}
}

func TestReportRunDeleteRollsBackWhenArtifactDeleteFails(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	final := seedCompletedReportRun(t, ctx, store, "mis_rr_rollback", "evt_rr_rollback_pending", "evt_rr_rollback_final", "art_rr_rollback_final")
	if err := store.CreateSourceSnapshot(ctx, app.SourceSnapshot{
		SnapshotID:  "src_rr_rollback",
		MissionID:   final.MissionID,
		Connector:   app.ConnectorRef{ConnectorID: "test", ConnectorType: app.SourceConnectorTypeFileUpload},
		Title:       "Linked report",
		CapturedAt:  time.Now().UTC(),
		ArtifactIDs: []string{final.ArtifactID},
		ContentHash: app.ContentHash{Algorithm: "sha256", Value: final.SHA256},
		Locators:    []byte(`[]`),
		Access:      app.SourceAccess{Visibility: "private", License: "unknown", RetrievalPolicy: app.SourceRetrievalPolicySnapshotOnly},
	}); err != nil {
		t.Fatalf("CreateSourceSnapshot returned error: %v", err)
	}

	facts, err := store.LoadReportRunDeleteFacts(ctx, final.MissionID, final.ArtifactID)
	if err != nil {
		t.Fatalf("LoadReportRunDeleteFacts returned error: %v", err)
	}
	_, err = store.DeleteReportRun(ctx, final.MissionID, final.ArtifactID, facts.Run.Revision, reportrun.DeleteFactsHash(facts), func(facts reportrun.DeleteFacts) (reportrun.DeleteDecision, error) {
		decision := reportrun.BuildDeleteDecision(facts, "", "user", "test", time.Now().UTC())
		decision.DeleteArtifactIDs = []string{final.ArtifactID}
		return decision, nil
	})
	if err == nil {
		t.Fatal("expected delete to fail on source snapshot foreign key")
	}
	if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_report_run_events WHERE run_id = ?`, "evt_rr_rollback_pending") != 2 {
		t.Fatal("event memberships were not rolled back")
	}
	if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_ledger_events WHERE event_id = ?`, "evt_rr_rollback_final") != 1 {
		t.Fatal("ledger event delete was not rolled back")
	}
	if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_raw_artifacts WHERE artifact_id = ?`, final.ArtifactID) != 1 {
		t.Fatal("artifact delete was not rolled back")
	}
}

func TestMissionHardDeleteRemovesReportRunRows(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	final := seedCompletedReportRun(t, ctx, store, "mis_rr_hard_delete", "evt_rr_hard_pending", "evt_rr_hard_final", "art_rr_hard_final")
	if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_report_runs WHERE mission_id = ?`, final.MissionID) == 0 {
		t.Fatal("report run fixture did not register")
	}

	if _, err := store.HardDeleteMission(ctx, final.MissionID, nil); err != nil {
		t.Fatalf("HardDeleteMission returned error: %v", err)
	}
	for _, query := range []string{
		`SELECT COUNT(*) FROM plasma_report_runs WHERE mission_id = ?`,
		`SELECT COUNT(*) FROM plasma_report_run_events WHERE mission_id = ?`,
		`SELECT COUNT(*) FROM plasma_report_run_artifacts WHERE mission_id = ?`,
	} {
		if got := countRows(t, ctx, store, query, final.MissionID); got != 0 {
			t.Fatalf("report run row remained for %q: %d", query, got)
		}
	}
}

func seedCompletedReportRun(t *testing.T, ctx context.Context, store *Store, missionID, pendingID, finalEventID, artifactID string) app.RawArtifact {
	t.Helper()
	if err := store.CreateMission(ctx, app.Mission{MissionID: missionID, Title: missionID}); err != nil {
		t.Fatalf("CreateMission returned error: %v", err)
	}
	return seedCompletedReportAttempt(t, ctx, store, missionID, pendingID, finalEventID, artifactID)
}

func seedCompletedReportAttempt(t *testing.T, ctx context.Context, store *Store, missionID, pendingID, finalEventID, artifactID string) app.RawArtifact {
	t.Helper()
	appendLedgerEvent(t, ctx, store, app.LedgerEvent{
		EventID:   pendingID,
		MissionID: missionID,
		EventType: "report.draft.pending",
		Producer:  app.Producer{Type: "user", ID: "test"},
		Payload:   []byte(`{"title":"Report"}`),
	})
	svc := app.NewService(store)
	artifact, _, err := svc.CreateRawArtifactWithEvent(ctx, app.CreateRawArtifactRequest{
		ArtifactID: artifactID,
		MissionID:  missionID,
		MediaType:  "text/markdown",
		Filename:   artifactID + ".md",
		Producer:   app.Producer{Type: "agent", ID: "test"},
		Content:    []byte("final md " + artifactID),
	}, func(artifact app.RawArtifact) app.AppendEventRequest {
		return app.AppendEventRequest{
			EventID:   finalEventID,
			MissionID: missionID,
			EventType: "report.artifact.created",
			Producer:  app.Producer{Type: "agent", ID: "test"},
			Payload:   []byte(`{"kind":"markdown_report_artifact","pending_event_id":"` + pendingID + `","artifact_id":"` + artifact.ArtifactID + `","agent_usage":{"provider_usage":{"input_tokens":4,"output_tokens":5,"total_tokens":9}}}`),
		}
	})
	if err != nil {
		t.Fatalf("CreateRawArtifactWithEvent returned error: %v", err)
	}
	return artifact
}

func appendLedgerEvent(t *testing.T, ctx context.Context, store *Store, event app.LedgerEvent) {
	t.Helper()
	if _, err := store.AppendLedgerEvent(ctx, event); err != nil {
		t.Fatalf("AppendLedgerEvent %s returned error: %v", event.EventID, err)
	}
}

func createStoredArtifact(t *testing.T, ctx context.Context, store *Store, missionID string, artifactID string, content []byte) {
	t.Helper()
	if err := store.CreateRawArtifact(ctx, app.RawArtifact{
		ArtifactID: artifactID,
		MissionID:  missionID,
		MediaType:  "text/markdown",
		ByteSize:   int64(len(content)),
		SHA256:     "sha_" + artifactID,
		Producer:   app.Producer{Type: "agent", ID: "test"},
		CreatedAt:  time.Now().UTC(),
		Content:    content,
		Filename:   artifactID + ".md",
	}); err != nil {
		t.Fatalf("CreateRawArtifact returned error: %v", err)
	}
}

func assertArtifactMembership(t *testing.T, ctx context.Context, store *Store, runID string, artifactID string, role string, ownership string) {
	t.Helper()
	var gotRole string
	var gotOwnership string
	if err := store.db.QueryRowContext(ctx, `
SELECT artifact_role, ownership
FROM plasma_report_run_artifacts
WHERE run_id = ? AND artifact_id = ?`, runID, artifactID).Scan(&gotRole, &gotOwnership); err != nil {
		t.Fatalf("load artifact membership: %v", err)
	}
	if gotRole != role || gotOwnership != ownership {
		t.Fatalf("artifact membership = role %q ownership %q, want role %q ownership %q", gotRole, gotOwnership, role, ownership)
	}
}

func insertLedgerPayloadDirect(t *testing.T, ctx context.Context, store *Store, missionID string, eventID string, eventType string, payload string) {
	t.Helper()
	var sequence int64
	if err := store.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence), 0) + 1 FROM plasma_ledger_events WHERE mission_id = ?`, missionID).Scan(&sequence); err != nil {
		t.Fatalf("next sequence: %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `
INSERT INTO plasma_ledger_events (
  event_id, mission_id, sequence, event_type, producer_type, producer_id,
  payload_json, created_at
) VALUES (?, ?, ?, ?, 'user', 'test', ?, ?)`,
		eventID, missionID, sequence, eventType, payload, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		t.Fatalf("insert direct ledger payload: %v", err)
	}
}

func hasDeleteBlocker(preview reportrun.DeletePreview, code string) bool {
	for _, blocker := range preview.Blockers {
		if blocker.ReasonCode == code {
			return true
		}
	}
	return false
}

func assertRevisionUnchanged(t *testing.T, ctx context.Context, store *Store, before reportrun.DeleteFacts) {
	t.Helper()
	var revision int64
	if err := store.db.QueryRowContext(ctx, `SELECT revision FROM plasma_report_runs WHERE run_id = ?`, before.Run.RunID).Scan(&revision); err != nil {
		t.Fatalf("load run revision: %v", err)
	}
	if revision != before.Run.Revision {
		t.Fatalf("run revision changed: before=%d after=%d", before.Run.Revision, revision)
	}
}
