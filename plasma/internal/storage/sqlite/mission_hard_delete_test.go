package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestMissionHardDeleteDeletesMissionScopedRowsAndKeepsOtherMissions(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	seedMissionHardDeleteFixture(t, ctx, store, "mis_delete", "art_delete", "src_delete", "rpt_delete")
	seedMissionHardDeleteFixture(t, ctx, store, "mis_keep", "art_keep", "src_keep", "rpt_keep")

	preview, err := store.PreviewMissionHardDelete(ctx, "mis_delete")
	if err != nil {
		t.Fatal(err)
	}
	if preview.LedgerEvents != 2 || preview.RawArtifacts != 1 || preview.RawArtifactBytes != 14 ||
		preview.SourceSnapshots != 1 || preview.SourceSnapshotArtifactLinks != 1 ||
		preview.EvidenceRecords != 1 || preview.ClaimRecords != 1 || preview.QuestionRecords != 1 ||
		preview.OptionRecords != 1 || preview.ProposalBundles != 1 ||
		preview.Reports != 1 || preview.ReportVersions != 1 || preview.ReportBlocks != 1 {
		t.Fatalf("preview impact = %#v", preview)
	}

	deleted, err := store.HardDeleteMission(ctx, "mis_delete", nil)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != preview {
		t.Fatalf("deleted impact = %#v, preview = %#v", deleted, preview)
	}
	if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_missions WHERE mission_id = ?`, "mis_delete") != 0 {
		t.Fatal("deleted mission row remained")
	}
	if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_source_snapshot_artifacts WHERE snapshot_id = ? OR artifact_id = ?`, "src_delete", "art_delete") != 0 {
		t.Fatal("deleted mission source-artifact link remained")
	}
	for _, query := range []string{
		`SELECT COUNT(*) FROM plasma_ledger_events WHERE mission_id = ?`,
		`SELECT COUNT(*) FROM plasma_raw_artifacts WHERE mission_id = ?`,
		`SELECT COUNT(*) FROM plasma_source_snapshots WHERE mission_id = ?`,
		`SELECT COUNT(*) FROM plasma_evidence_records WHERE mission_id = ?`,
		`SELECT COUNT(*) FROM plasma_claim_records WHERE mission_id = ?`,
		`SELECT COUNT(*) FROM plasma_question_records WHERE mission_id = ?`,
		`SELECT COUNT(*) FROM plasma_option_records WHERE mission_id = ?`,
		`SELECT COUNT(*) FROM plasma_proposal_bundles WHERE mission_id = ?`,
		`SELECT COUNT(*) FROM plasma_reports WHERE mission_id = ?`,
		`SELECT COUNT(*) FROM plasma_report_versions WHERE mission_id = ?`,
		`SELECT COUNT(*) FROM plasma_report_blocks WHERE mission_id = ?`,
	} {
		if got := countRows(t, ctx, store, query, "mis_delete"); got != 0 {
			t.Fatalf("deleted mission rows remained for %q: %d", query, got)
		}
	}
	if _, err := store.GetRawArtifact(ctx, "art_delete"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected deleted artifact to be gone, got %v", err)
	}
	if _, err := store.GetReport(ctx, "rpt_delete"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected deleted report to be gone, got %v", err)
	}
	if _, err := store.GetRawArtifact(ctx, "art_keep"); err != nil {
		t.Fatalf("kept artifact was removed: %v", err)
	}
	if _, err := store.GetReport(ctx, "rpt_keep"); err != nil {
		t.Fatalf("kept report was removed: %v", err)
	}
	if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_missions WHERE mission_id = ?`, "mis_keep") != 1 {
		t.Fatal("kept mission row was removed")
	}
}

func TestMissionHardDeleteRollsBackWhenValidatorRejects(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	seedMissionHardDeleteFixture(t, ctx, store, "mis_delete", "art_delete", "src_delete", "rpt_delete")

	_, err = store.HardDeleteMission(ctx, "mis_delete", func([]app.LedgerEvent) error {
		return app.ErrConflict
	})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("expected validator conflict, got %v", err)
	}
	if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_missions WHERE mission_id = ?`, "mis_delete") != 1 {
		t.Fatal("mission row changed after validator rejection")
	}
	if _, err := store.GetRawArtifact(ctx, "art_delete"); err != nil {
		t.Fatalf("artifact changed after validator rejection: %v", err)
	}
}

func seedMissionHardDeleteFixture(t *testing.T, ctx context.Context, store *Store, missionID, artifactID, snapshotID, reportID string) {
	t.Helper()
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	if err := store.CreateMission(ctx, app.Mission{
		MissionID: missionID, Title: missionID, CreatedAt: now, UpdatedAt: now, LifecycleState: app.MissionLifecycleArchived,
	}); err != nil {
		t.Fatal(err)
	}
	for i, eventType := range []string{"mission.created", app.MissionArchivedEvent} {
		if _, err := store.AppendLedgerEvent(ctx, app.LedgerEvent{
			EventID:   "evt_" + missionID + "_" + eventType,
			MissionID: missionID,
			EventType: eventType,
			Producer:  app.Producer{Type: "user", ID: "test"},
			Payload:   []byte(`{"title":"fixture"}`),
			CreatedAt: now.Add(time.Duration(i) * time.Second),
		}); err != nil {
			t.Fatal(err)
		}
	}
	content := []byte("fixture source")
	if err := store.CreateRawArtifact(ctx, app.RawArtifact{
		ArtifactID: artifactID,
		MissionID:  missionID,
		MediaType:  "text/plain",
		ByteSize:   int64(len(content)),
		SHA256:     "sha_" + artifactID,
		Producer:   app.Producer{Type: "user", ID: "test"},
		CreatedAt:  now,
		Content:    content,
		StorageURI: "",
		Filename:   artifactID + ".txt",
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSourceSnapshot(ctx, app.SourceSnapshot{
		SnapshotID:  snapshotID,
		MissionID:   missionID,
		Connector:   app.ConnectorRef{ConnectorID: "test", ConnectorType: app.SourceConnectorTypeFileUpload},
		Title:       "Fixture source",
		CapturedAt:  now,
		ArtifactIDs: []string{artifactID},
		ContentHash: app.ContentHash{Algorithm: "sha256", Value: "sha_" + artifactID},
		Locators:    []byte(`[]`),
		Access:      app.SourceAccess{Visibility: "private", License: "unknown", RetrievalPolicy: app.SourceRetrievalPolicySnapshotOnly},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateEvidenceRecord(ctx, app.EvidenceRecord{
		SchemaVersion:  app.EvidenceRecordSchemaVersion,
		ObjectKind:     app.EvidenceRecordObjectKind,
		EvidenceID:     "evd_" + missionID,
		MissionID:      missionID,
		State:          "approved",
		Summary:        "Evidence",
		EvidenceType:   "quote",
		SnapshotRefs:   []app.SnapshotRef{{SnapshotID: snapshotID, ArtifactID: artifactID, Locator: []byte(`{}`)}},
		Confidence:     app.Confidence{Level: "medium"},
		Producer:       app.Producer{Type: "agent", ID: "test"},
		CreatedEventID: "evt_" + missionID + "_evidence",
		CreatedAt:      now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateClaimRecord(ctx, app.ClaimRecord{
		SchemaVersion:         app.ClaimRecordSchemaVersion,
		ObjectKind:            app.ClaimRecordObjectKind,
		ClaimID:               "clm_" + missionID,
		MissionID:             missionID,
		State:                 "approved",
		Text:                  "Claim",
		ClaimType:             "finding",
		SupportingEvidenceIDs: []string{"evd_" + missionID},
		Confidence:            app.Confidence{Level: "medium"},
		Approval:              app.Approval{State: "approved"},
		CreatedEventID:        "evt_" + missionID + "_claim",
		CreatedAt:             now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateQuestionRecord(ctx, app.QuestionRecord{
		SchemaVersion:  app.QuestionRecordSchemaVersion,
		ObjectKind:     app.QuestionRecordObjectKind,
		QuestionID:     "qst_" + missionID,
		MissionID:      missionID,
		State:          "open",
		Text:           "Question",
		Priority:       "medium",
		CreatedEventID: "evt_" + missionID + "_question",
		CreatedAt:      now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateOptionRecord(ctx, app.OptionRecord{
		SchemaVersion:  app.OptionRecordSchemaVersion,
		ObjectKind:     app.OptionRecordObjectKind,
		OptionID:       "opt_" + missionID,
		MissionID:      missionID,
		State:          "open",
		Title:          "Option",
		RiskLevel:      "medium",
		CreatedEventID: "evt_" + missionID + "_option",
		CreatedAt:      now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateProposalBundle(ctx, app.ProposalBundle{
		SchemaVersion:     app.ProposalBundleSchemaVersion,
		ObjectKind:        app.ProposalBundleObjectKind,
		ProposalID:        "prp_" + missionID,
		MissionID:         missionID,
		State:             "pending_review",
		Title:             "Proposal",
		ObjectRefs:        []app.ObjectRef{{ObjectKind: app.EvidenceRecordObjectKind, ObjectID: "evd_" + missionID}},
		RequestedDecision: "approve",
		CreatedEventID:    "evt_" + missionID + "_proposal",
		CreatedAt:         now,
		UpdatedAt:         now,
	}); err != nil {
		t.Fatal(err)
	}
	versionID := "rvn_" + missionID
	blockID := "blk_" + missionID
	if err := store.CreateReport(ctx, app.Report{
		SchemaVersion:   app.ReportSchemaVersion,
		ObjectKind:      app.ReportObjectKind,
		ReportID:        reportID,
		MissionID:       missionID,
		Title:           "Report",
		ActiveVersionID: versionID,
		State:           "draft",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateReportVersion(ctx, app.ReportVersion{
		SchemaVersion:   app.ReportVersionSchemaVersion,
		ObjectKind:      app.ReportVersionObjectKind,
		ReportVersionID: versionID,
		ReportID:        reportID,
		MissionID:       missionID,
		State:           "draft",
		RootBlockID:     blockID,
		BlockIDs:        []string{blockID},
		CreatedEventID:  "evt_" + missionID + "_report",
		CreatedAt:       now,
	}, []app.ReportBlock{{
		SchemaVersion:   app.ReportBlockSchemaVersion,
		ObjectKind:      app.ReportBlockObjectKind,
		BlockID:         blockID,
		ReportVersionID: versionID,
		MissionID:       missionID,
		BlockType:       "paragraph",
		Order:           1,
		Content:         []byte(`{"text":"Report block"}`),
		SourceRefs:      app.ReportBlockSourceRefs{EvidenceIDs: []string{"evd_" + missionID}},
		Authorship:      app.ReportBlockAuthorship{Mode: "agent", Producer: app.Producer{Type: "agent", ID: "test"}},
		Approval:        app.Approval{State: "pending"},
	}}); err != nil {
		t.Fatal(err)
	}
}

func countRows(t *testing.T, ctx context.Context, store *Store, query string, args ...any) int64 {
	t.Helper()
	var count int64
	if err := store.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
