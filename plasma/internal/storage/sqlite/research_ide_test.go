package sqlite

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestResearchIDEReaderListsChunksGrepsAndReferences(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newResearchIDEFixture(t, store)

	if _, err := svc.ListMissionObjects(ctx, "mis_1", app.ResearchIDEObjectEvidenceRecord, 1, ""); !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("default ListMissionObjects should reject legacy evidence records, got %v", err)
	}

	page, err := svc.ListMissionObjects(ctx, "mis_1", app.ResearchIDEObjectRawArtifact, 1, "")
	if err != nil {
		t.Fatalf("ListMissionObjects returned error: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].ObjectID != "art_1" {
		t.Fatalf("unexpected raw artifact page: %#v", page)
	}

	read, err := svc.ReadMissionObject(ctx, app.ResearchIDEReadRequest{
		MissionID:  "mis_1",
		ObjectKind: app.ResearchIDEObjectRawArtifact,
		ObjectID:   "art_1",
		MaxBytes:   12,
	})
	if err != nil {
		t.Fatalf("ReadMissionObject returned error: %v", err)
	}
	if string(read.Data) != "alpha beta g" || !read.Truncated || read.NextOffset != 12 {
		t.Fatalf("unexpected chunked read: %#v", read)
	}

	grep, err := svc.GrepMissionObjects(ctx, "mis_1", "gamma", 10, "")
	if err != nil {
		t.Fatalf("GrepMissionObjects returned error: %v", err)
	}
	if len(grep.Matches) == 0 {
		t.Fatalf("expected grep match")
	}

	legacyPage, err := svc.ListMissionObjectsLegacy(ctx, "mis_1", app.ResearchIDEObjectEvidenceRecord, 1, "")
	if err != nil {
		t.Fatalf("ListMissionObjectsLegacy returned error: %v", err)
	}
	if len(legacyPage.Items) != 1 || legacyPage.Items[0].ObjectID != "evd_2" || legacyPage.NextCursor == "" || !legacyPage.Truncated {
		t.Fatalf("unexpected first legacy evidence page: %#v", legacyPage)
	}
	legacyNext, err := svc.ListMissionObjectsLegacy(ctx, "mis_1", app.ResearchIDEObjectEvidenceRecord, 1, legacyPage.NextCursor)
	if err != nil {
		t.Fatalf("ListMissionObjectsLegacy second page returned error: %v", err)
	}
	if len(legacyNext.Items) != 1 || legacyNext.Items[0].ObjectID != "evd_1" {
		t.Fatalf("unexpected second legacy evidence page: %#v", legacyNext)
	}

	refs, err := svc.ListObjectReferencesLegacy(ctx, "mis_1", app.ResearchIDEObjectEvidenceRecord, "evd_1", 10, "")
	if err != nil {
		t.Fatalf("ListObjectReferences returned error: %v", err)
	}
	if !hasResearchIDERef(refs.Forward, app.ResearchIDEObjectSourceSnapshot, "src_1") {
		t.Fatalf("expected evidence forward source ref: %#v", refs)
	}
	if !hasResearchIDERef(refs.Backward, app.ResearchIDEObjectClaimRecord, "clm_1") ||
		!hasAnyResearchIDERefKind(refs.Backward, app.ResearchIDEObjectReportBlock) {
		t.Fatalf("expected claim and report block backward refs: %#v", refs)
	}
}

func TestResearchIDEHidesStagedSourceCandidateArtifacts(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newResearchIDEFixture(t, store)
	if _, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_candidate",
		MissionID:  "mis_1",
		MediaType:  "text/plain; charset=utf-8",
		Filename:   "candidate.txt",
		Producer:   app.Producer{Type: "agent", ID: "codex"},
		Content:    []byte("unapproved candidate body"),
	}); err != nil {
		t.Fatalf("CreateRawArtifact candidate returned error: %v", err)
	}
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   "evt_candidate_staged",
		MissionID: "mis_1",
		EventType: "source.candidate.staged",
		Producer:  app.Producer{Type: "agent", ID: "codex"},
		Payload: []byte(`{
			"url":"https://example.com/source",
			"proposal_event_id":"evt_candidate_proposed",
			"artifact_id":"art_candidate",
			"approval_state":"unapproved_candidate",
			"not_report_default":true
		}`),
	}); err != nil {
		t.Fatalf("AppendEvent source candidate staged returned error: %v", err)
	}

	page, err := svc.ListMissionObjects(ctx, "mis_1", app.ResearchIDEObjectRawArtifact, 10, "")
	if err != nil {
		t.Fatalf("ListMissionObjects returned error: %v", err)
	}
	for _, item := range page.Items {
		if item.ObjectID == "art_candidate" {
			t.Fatalf("staged source candidate artifact must not be listed as a normal raw artifact: %#v", page.Items)
		}
	}
	_, err = svc.ReadMissionObject(ctx, app.ResearchIDEReadRequest{
		MissionID:  "mis_1",
		ObjectKind: app.ResearchIDEObjectRawArtifact,
		ObjectID:   "art_candidate",
		MaxBytes:   64,
	})
	if !errors.Is(err, app.ErrInvalidInput) || !strings.Contains(err.Error(), "unapproved source candidate") {
		t.Fatalf("expected staged candidate raw artifact read to be blocked, got %v", err)
	}
}

func TestResearchIDEReferencesArePaged(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newResearchIDEFixture(t, store)

	first, err := svc.ListObjectReferencesLegacy(ctx, "mis_1", app.ResearchIDEObjectEvidenceRecord, "evd_1", 1, "")
	if err != nil {
		t.Fatalf("ListObjectReferences first page returned error: %v", err)
	}
	if len(first.Forward)+len(first.Backward) != 1 || first.NextCursor == "" || !first.Truncated {
		t.Fatalf("expected one paged reference and next cursor, got %#v", first)
	}
	second, err := svc.ListObjectReferencesLegacy(ctx, "mis_1", app.ResearchIDEObjectEvidenceRecord, "evd_1", 10, first.NextCursor)
	if err != nil {
		t.Fatalf("ListObjectReferences second page returned error: %v", err)
	}
	if len(second.Forward)+len(second.Backward) == 0 {
		t.Fatalf("expected remaining references on second page, got %#v", second)
	}
}

func TestResearchIDEReadReportVersionChildrenAreFilteredBeforePaging(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newResearchIDEFixture(t, store)
	if _, err := svc.CreateReportDraft(ctx, app.CreateReportDraftRequest{
		ReportID:        "rpt_2",
		ReportVersionID: "rvn_2",
		MissionID:       "mis_1",
		Title:           "Second Report",
		FormatIntent:    "briefing",
		Scope:           app.ReportEvidenceScope{AcceptedOnly: true, ClaimIDs: []string{"clm_1"}, EvidenceIDs: []string{"evd_1"}},
		Producer:        app.Producer{Type: "agent_session", ID: "ses_report"},
		CreatedEventID:  "evt_report_drafted_2",
		Blocks: []app.ReportBlockDraftInput{{
			BlockType: "paragraph",
			Content:   []byte(`{"text":"Second report block."}`),
		}},
	}); err != nil {
		t.Fatalf("CreateReportDraft second returned error: %v", err)
	}

	read, err := svc.ReadMissionObject(ctx, app.ResearchIDEReadRequest{
		MissionID:  "mis_1",
		ObjectKind: app.ResearchIDEObjectReportVersion,
		ObjectID:   "rvn_1",
		Limit:      1,
		Legacy:     true,
	})
	if err != nil {
		t.Fatalf("ReadMissionObject report version returned error: %v", err)
	}
	if read.Children == nil || len(read.Children.Items) != 1 {
		t.Fatalf("expected one child block for rvn_1, got %#v", read.Children)
	}
	if got := read.Children.Items[0].Metadata["report_version_id"]; got != "rvn_1" {
		t.Fatalf("expected child block from rvn_1, got %#v", read.Children.Items[0])
	}
}

func TestResearchIDEReadKeepsUTF8Boundaries(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newResearchIDEFixture(t, store)
	content := []byte("가나다🙂xyz")
	if _, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_utf8",
		MissionID:  "mis_1",
		MediaType:  "text/plain; charset=utf-8",
		Producer:   app.Producer{Type: "connector", ID: "test"},
		Content:    content,
	}); err != nil {
		t.Fatalf("CreateRawArtifact utf8 returned error: %v", err)
	}

	first, err := svc.ReadMissionObject(ctx, app.ResearchIDEReadRequest{
		MissionID:  "mis_1",
		ObjectKind: app.ResearchIDEObjectRawArtifact,
		ObjectID:   "art_utf8",
		MaxBytes:   4,
	})
	if err != nil {
		t.Fatalf("ReadMissionObject UTF-8 first chunk returned error: %v", err)
	}
	if first.Data != "가" || !first.Truncated || first.NextOffset != len([]byte("가")) {
		t.Fatalf("unexpected UTF-8 chunk: %#v", first)
	}

	_, err = svc.ReadMissionObject(ctx, app.ResearchIDEReadRequest{
		MissionID:  "mis_1",
		ObjectKind: app.ResearchIDEObjectRawArtifact,
		ObjectID:   "art_utf8",
		Offset:     1,
		MaxBytes:   4,
	})
	if !errors.Is(err, app.ErrInvalidInput) || !strings.Contains(err.Error(), "UTF-8 boundary") {
		t.Fatalf("expected UTF-8 boundary error, got %v", err)
	}
}

func TestResearchIDEReaderRejectsCrossMissionReads(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newResearchIDEFixture(t, store)
	createSecondMissionArtifact(t, ctx, svc)

	_, err := svc.ReadMissionObject(ctx, app.ResearchIDEReadRequest{
		MissionID:  "mis_1",
		ObjectKind: app.ResearchIDEObjectRawArtifact,
		ObjectID:   "art_other",
	})
	if !errors.Is(err, app.ErrInvalidInput) || !strings.Contains(err.Error(), "art_other") {
		t.Fatalf("expected cross-mission invalid input with object id, got %v", err)
	}
}

func TestResearchIDEOutlineKeepsSmallMissionOverview(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newResearchIDEFixture(t, store)

	outline, err := svc.OutlineMission(ctx, "mis_1")
	if err != nil {
		t.Fatalf("OutlineMission returned error: %v", err)
	}
	if outline.Title != "Research Mission" ||
		outline.Counts[app.ResearchIDEObjectSourceSnapshot] != 1 ||
		outline.Counts[app.ResearchIDEObjectRawArtifact] != 1 ||
		outline.Counts["evidence_record.proposed"] != 0 ||
		outline.ActiveReportVersionID != "" {
		t.Fatalf("unexpected outline: %#v", outline)
	}
	if len(outline.NextSuggestedObjectRefs) > 6 {
		t.Fatalf("outline suggestions should stay capped, got %#v", outline.NextSuggestedObjectRefs)
	}
	for _, event := range outline.RecentLedgerEvents {
		if strings.Contains(event.Summary, "alpha beta gamma") {
			t.Fatalf("outline leaked source body into ledger summary: %#v", outline)
		}
	}

	legacyOutline, err := svc.OutlineMissionLegacy(ctx, "mis_1")
	if err != nil {
		t.Fatalf("OutlineMissionLegacy returned error: %v", err)
	}
	if legacyOutline.Counts["evidence_record.proposed"] != 2 || legacyOutline.ActiveReportVersionID != "rvn_1" {
		t.Fatalf("unexpected legacy outline: %#v", legacyOutline)
	}
}

func newResearchIDEFixture(t *testing.T, store *Store) *app.Service {
	t.Helper()
	ctx := context.Background()
	svc := newResearchTestService(t, store)
	if err := store.SaveMissionProjection(ctx, app.MissionProjection{
		MissionID:             "mis_1",
		LastEventID:           "evt_report_drafted",
		LastSequence:          8,
		Title:                 "Research Mission",
		Objective:             "Explain the research ledger.",
		Scope:                 app.MissionScope{Included: []string{"sources"}, Excluded: []string{"prompt stuffing"}},
		AcceptedClaimIDs:      []string{"clm_1"},
		OpenQuestionIDs:       []string{"qst_1", "qst_extra_1", "qst_extra_2", "qst_extra_3", "qst_extra_4", "qst_extra_5", "qst_extra_6"},
		ActiveReportVersionID: "rvn_1",
		LifecycleState:        "active",
	}); err != nil {
		t.Fatalf("SaveMissionProjection returned error: %v", err)
	}
	artifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_1",
		MissionID:  "mis_1",
		MediaType:  "text/plain",
		Filename:   "source.txt",
		Producer:   app.Producer{Type: "connector", ID: "liquid2"},
		Content:    []byte("alpha beta gamma delta epsilon zeta"),
	})
	if err != nil {
		t.Fatalf("CreateRawArtifact returned error: %v", err)
	}
	if _, err := svc.CreateSourceSnapshot(ctx, app.CreateSourceSnapshotRequest{
		SnapshotID:  "src_1",
		MissionID:   "mis_1",
		Connector:   app.ConnectorRef{ConnectorID: "liquid2", ConnectorType: "liquid2", ExternalSourceID: "doc_1"},
		Title:       "Pinned source",
		ArtifactIDs: []string{"art_1"},
		ContentHash: app.ContentHash{Algorithm: "sha256", Value: artifact.SHA256},
	}); err != nil {
		t.Fatalf("CreateSourceSnapshot returned error: %v", err)
	}
	for _, evidence := range []app.CreateEvidenceRecordRequest{
		{
			EvidenceID:     "evd_1",
			MissionID:      "mis_1",
			Summary:        "Gamma appears in the pinned source.",
			EvidenceType:   "quote",
			SnapshotRefs:   []app.SnapshotRef{{SnapshotID: "src_1", ArtifactID: "art_1", Locator: []byte(`{"locator_type":"text_quote","exact":"gamma"}`)}},
			Confidence:     app.Confidence{Level: "medium"},
			Producer:       app.Producer{Type: "autopilot", ID: "ses_auto"},
			CreatedEventID: "evt_evidence",
		},
		{
			EvidenceID:     "evd_2",
			MissionID:      "mis_1",
			Summary:        "Delta is nearby.",
			EvidenceType:   "quote",
			SnapshotRefs:   []app.SnapshotRef{{SnapshotID: "src_1", ArtifactID: "art_1", Locator: []byte(`{"locator_type":"text_quote","exact":"delta"}`)}},
			Confidence:     app.Confidence{Level: "low"},
			Producer:       app.Producer{Type: "autopilot", ID: "ses_auto"},
			CreatedEventID: "evt_evidence",
		},
	} {
		if _, err := svc.CreateEvidenceRecord(ctx, evidence); err != nil {
			t.Fatalf("CreateEvidenceRecord %s returned error: %v", evidence.EvidenceID, err)
		}
	}
	if _, err := svc.CreateClaimRecord(ctx, app.CreateClaimRecordRequest{
		ClaimID:               "clm_1",
		MissionID:             "mis_1",
		State:                 "approved",
		Text:                  "Gamma is a useful research signal.",
		ClaimType:             "descriptive",
		SupportingEvidenceIDs: []string{"evd_1"},
		Confidence:            app.Confidence{Level: "high"},
		Approval:              app.Approval{State: "approved", ApprovalEventID: "evt_approval"},
		CreatedEventID:        "evt_claim",
	}); err != nil {
		t.Fatalf("CreateClaimRecord returned error: %v", err)
	}
	if _, err := svc.CreateQuestionRecord(ctx, app.CreateQuestionRecordRequest{
		QuestionID:         "qst_1",
		MissionID:          "mis_1",
		Text:               "How should gamma be interpreted?",
		Priority:           "medium",
		RelatedEvidenceIDs: []string{"evd_1"},
		RelatedClaimIDs:    []string{"clm_1"},
		CreatedEventID:     "evt_question",
	}); err != nil {
		t.Fatalf("CreateQuestionRecord returned error: %v", err)
	}
	if _, err := svc.CreateReportDraft(ctx, app.CreateReportDraftRequest{
		ReportID:        "rpt_1",
		ReportVersionID: "rvn_1",
		MissionID:       "mis_1",
		Title:           "Research Report",
		FormatIntent:    "briefing",
		Scope:           app.ReportEvidenceScope{AcceptedOnly: true, ClaimIDs: []string{"clm_1"}, EvidenceIDs: []string{"evd_1"}},
		Producer:        app.Producer{Type: "agent_session", ID: "ses_report"},
		CreatedEventID:  "evt_report_drafted",
		Blocks: []app.ReportBlockDraftInput{{
			BlockType: "paragraph",
			Content:   []byte(`{"text":"Gamma is a useful research signal."}`),
			SourceRefs: app.ReportBlockSourceRefs{
				ClaimIDs:    []string{"clm_1"},
				EvidenceIDs: []string{"evd_1"},
				SnapshotIDs: []string{"src_1"},
			},
		}},
	}); err != nil {
		t.Fatalf("CreateReportDraft returned error: %v", err)
	}
	return svc
}

func createSecondMissionArtifact(t *testing.T, ctx context.Context, svc *app.Service) {
	t.Helper()
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: "mis_2", Title: "Other"}); err != nil {
		t.Fatalf("CreateMission second returned error: %v", err)
	}
	if _, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_other",
		MissionID:  "mis_2",
		MediaType:  "text/plain",
		Producer:   app.Producer{Type: "connector", ID: "liquid2"},
		Content:    []byte("other mission body"),
	}); err != nil {
		t.Fatalf("CreateRawArtifact second returned error: %v", err)
	}
}

func hasResearchIDERef(refs []app.ResearchIDEObjectRef, kind, id string) bool {
	for _, ref := range refs {
		if ref.ObjectKind == kind && ref.ObjectID == id {
			return true
		}
	}
	return false
}

func hasAnyResearchIDERefKind(refs []app.ResearchIDEObjectRef, kind string) bool {
	for _, ref := range refs {
		if ref.ObjectKind == kind {
			return true
		}
	}
	return false
}
