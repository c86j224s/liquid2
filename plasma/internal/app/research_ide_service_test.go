package app

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestSummarizeSourceSnapshotPromotesUploadedFileLocatorMetadata(t *testing.T) {
	imageLocators, err := json.Marshal([]UploadedFileLocator{{
		LocatorType:       SourceLocatorTypeMedia,
		MediaKind:         MediaKindImage,
		OriginalFilename:  "Pixel Source.png",
		SanitizedFilename: "Pixel-Source.png",
		MIMEType:          "image/png",
		ByteSize:          512,
		SHA256:            "image-sha",
		ContentKind:       UploadedContentKindImage,
	}})
	if err != nil {
		t.Fatal(err)
	}
	image := summarizeSourceSnapshot(SourceSnapshot{
		SnapshotID: "src_image",
		MissionID:  "mis_1",
		Connector:  ConnectorRef{ConnectorType: SourceConnectorTypeFileUpload},
		Locators:   imageLocators,
	})
	if image.Metadata["locator_type"] != SourceLocatorTypeMedia ||
		image.Metadata["media_kind"] != MediaKindImage ||
		image.Metadata["filename"] != "Pixel-Source.png" ||
		image.Metadata["mime_type"] != "image/png" ||
		image.Metadata["content_kind"] != UploadedContentKindImage {
		t.Fatalf("expected uploaded image locator metadata, got %#v", image.Metadata)
	}

	legacyText := summarizeSourceSnapshot(SourceSnapshot{
		SnapshotID: "src_text",
		MissionID:  "mis_1",
		Connector:  ConnectorRef{ConnectorType: SourceConnectorTypeFileUpload},
		Locators:   json.RawMessage(`[{"kind":"file_upload","original_filename":"Legacy Notes.md","sanitized_filename":"Legacy-Notes.md","media_type":"text/markdown","byte_size":128,"sha256":"text-sha","content_kind":"text"}]`),
	})
	if legacyText.Metadata["locator_type"] != SourceLocatorTypeFullDocument ||
		legacyText.Metadata["filename"] != "Legacy-Notes.md" ||
		legacyText.Metadata["mime_type"] != "text/markdown" ||
		legacyText.Metadata["content_kind"] != UploadedContentKindText {
		t.Fatalf("expected legacy uploaded text locator metadata, got %#v", legacyText.Metadata)
	}
}

func TestReadMissionObjectSourceSnapshotPDFReturnsExtractedText(t *testing.T) {
	pdfBytes := testResearchIDEPDFBytes(t, []string{"MCP PDF Source", "Alpha code is 67."})
	svc := NewService(&researchIDEFakeStore{
		snapshot: SourceSnapshot{
			SnapshotID:  "src_pdf",
			MissionID:   "mis_1",
			Title:       "PDF source",
			ArtifactIDs: []string{"art_pdf"},
			Connector:   ConnectorRef{ConnectorType: SourceConnectorTypeFileUpload},
			Access:      SourceAccess{RetrievalPolicy: SourceRetrievalPolicySnapshotOnly},
		},
		artifact: RawArtifact{
			ArtifactID: "art_pdf",
			MissionID:  "mis_1",
			MediaType:  "application/pdf",
			ByteSize:   int64(len(pdfBytes)),
			SHA256:     "sha",
			Filename:   "paper.pdf",
			Content:    pdfBytes,
		},
	})

	read, err := svc.ReadMissionObject(context.Background(), ResearchIDEReadRequest{
		MissionID:  "mis_1",
		ObjectKind: ResearchIDEObjectSourceSnapshot,
		ObjectID:   "src_pdf",
		MaxBytes:   12,
	})
	if err != nil {
		t.Fatalf("ReadMissionObject returned error: %v", err)
	}
	if read.ObjectKind != ResearchIDEObjectSourceSnapshot || read.ObjectID != "src_pdf" {
		t.Fatalf("unexpected read identity: %#v", read)
	}
	if !strings.Contains(read.Data, "MCP PDF") || strings.Contains(read.Data, "%PDF-") {
		t.Fatalf("expected extracted PDF text without raw bytes, got %s", read.Data)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(read.Data), &payload); err != nil {
		t.Fatalf("read data is not JSON: %v\n%s", err, read.Data)
	}
	if _, ok := payload["snapshot"]; ok {
		t.Fatalf("expected minimal source metadata without full snapshot payload, got %s", read.Data)
	}
	if payload["extraction_type"] != "pdf_text" ||
		payload["read_kind"] != "source_pdf_text" ||
		payload["max_read_bytes"] != float64(researchIDEMaxBytes) {
		t.Fatalf("expected PDF source read metadata, got %#v", payload)
	}
	artifact, ok := payload["artifact"].(map[string]any)
	if !ok || artifact["artifact_id"] != "art_pdf" {
		t.Fatalf("expected artifact metadata, got %#v", payload["artifact"])
	}
	source, ok := payload["source"].(map[string]any)
	if !ok || source["snapshot_id"] != "src_pdf" {
		t.Fatalf("expected source metadata, got %#v", payload["source"])
	}
	if !read.Truncated || read.NextOffset == 0 {
		t.Fatalf("expected chunked PDF read metadata, got %#v", read)
	}

	next, err := svc.ReadMissionObject(context.Background(), ResearchIDEReadRequest{
		MissionID:  "mis_1",
		ObjectKind: ResearchIDEObjectSourceSnapshot,
		ObjectID:   "src_pdf",
		Offset:     read.NextOffset,
		MaxBytes:   64,
	})
	if err != nil {
		t.Fatalf("ReadMissionObject continuation returned error: %v", err)
	}
	if !strings.Contains(next.Data, "Alpha code is 67.") || strings.Contains(next.Data, "%PDF-") {
		t.Fatalf("expected continuation to return later extracted PDF text without raw bytes, got %s", next.Data)
	}
}

func TestGrepMissionObjectsFindsSourceSnapshotPDFText(t *testing.T) {
	pdfBytes := testResearchIDEPDFBytes(t, []string{"MCP PDF Source", "Alpha code is 67."})
	svc := NewService(&researchIDEFakeStore{
		snapshot: SourceSnapshot{
			SnapshotID:  "src_pdf",
			MissionID:   "mis_1",
			Title:       "PDF source",
			ArtifactIDs: []string{"art_pdf"},
			Connector:   ConnectorRef{ConnectorType: SourceConnectorTypeFileUpload},
			Access:      SourceAccess{RetrievalPolicy: SourceRetrievalPolicySnapshotOnly},
		},
		artifact: RawArtifact{
			ArtifactID: "art_pdf",
			MissionID:  "mis_1",
			MediaType:  "application/pdf",
			ByteSize:   int64(len(pdfBytes)),
			SHA256:     "sha",
			Filename:   "paper.pdf",
			Content:    pdfBytes,
		},
	})

	result, err := svc.GrepMissionObjects(context.Background(), "mis_1", "Alpha code", 10, "")
	if err != nil {
		t.Fatalf("GrepMissionObjects returned error: %v", err)
	}
	for _, match := range result.Matches {
		if match.ObjectKind == ResearchIDEObjectSourceSnapshot && match.ObjectID == "src_pdf" {
			return
		}
	}
	t.Fatalf("expected source_snapshot PDF grep match, got %#v", result.Matches)
}

func TestGrepMissionObjectsReturnsNonOverlappingLiteralMatchesWithPagination(t *testing.T) {
	event := LedgerEvent{
		EventID:   "evt_overlap",
		MissionID: "mis_1",
		Sequence:  1,
		EventType: "note.created",
		Payload:   json.RawMessage(`{"body":"AaAaA"}`),
	}
	svc := NewService(&researchIDEGrepFakeStore{events: []LedgerEvent{event}})
	candidateText := summarizeLedgerEvent(event).Summary + "\n" + string(mustJSON(event))
	base := strings.Index(candidateText, "AaAaA")
	if base < 0 {
		t.Fatalf("test fixture missing body in candidate text: %s", candidateText)
	}

	result, err := svc.GrepMissionObjects(context.Background(), "mis_1", "aa", 10, "")
	if err != nil {
		t.Fatalf("GrepMissionObjects returned error: %v", err)
	}
	if result.NextCursor != "" || result.Truncated {
		t.Fatalf("expected untruncated full page, got cursor %q truncated %v", result.NextCursor, result.Truncated)
	}
	if len(result.Matches) != 2 {
		t.Fatalf("match count = %d, want 2: %#v", len(result.Matches), result.Matches)
	}
	for i, match := range result.Matches {
		if match.ObjectKind != ResearchIDEObjectLedgerEvent || match.ObjectID != "evt_overlap" {
			t.Fatalf("match %d has unexpected identity: %#v", i, match)
		}
	}
	if result.Matches[0].Position != base || result.Matches[1].Position != base+2 {
		t.Fatalf("positions = [%d %d], want [%d %d]", result.Matches[0].Position, result.Matches[1].Position, base, base+2)
	}
	if result.Matches[0].Position >= result.Matches[1].Position {
		t.Fatalf("positions are not ascending: %#v", result.Matches)
	}
	if !strings.Contains(result.Matches[0].Snippet, "AaAaA") || !strings.Contains(result.Matches[1].Snippet, "AaAaA") {
		t.Fatalf("expected snippets to preserve matched candidate text, got %#v", result.Matches)
	}

	first, err := svc.GrepMissionObjects(context.Background(), "mis_1", "aa", 1, "")
	if err != nil {
		t.Fatalf("GrepMissionObjects first page returned error: %v", err)
	}
	if len(first.Matches) != 1 || first.Matches[0].Position != base || first.NextCursor != "1" || !first.Truncated {
		t.Fatalf("unexpected first page: %#v", first)
	}
	second, err := svc.GrepMissionObjects(context.Background(), "mis_1", "aa", 1, first.NextCursor)
	if err != nil {
		t.Fatalf("GrepMissionObjects second page returned error: %v", err)
	}
	if len(second.Matches) != 1 || second.Matches[0].Position != base+2 || second.NextCursor != "" || second.Truncated {
		t.Fatalf("unexpected terminal page: %#v", second)
	}
}

func TestResearchDiscoveryHidesReportLineageOutsideLegacy(t *testing.T) {
	ctx := context.Background()
	store := &researchIDEVisibilityStore{
		projection: MissionProjection{MissionID: "mis_1", Title: "Mission"},
		snapshots: []SourceSnapshot{{
			SnapshotID:  "src_visible",
			MissionID:   "mis_1",
			Title:       "Visible source",
			ArtifactIDs: []string{"art_visible"},
			Connector:   ConnectorRef{ConnectorType: SourceConnectorTypeFileUpload},
			Access:      SourceAccess{RetrievalPolicy: SourceRetrievalPolicySnapshotOnly},
		}},
		artifacts: []RawArtifact{
			{
				ArtifactID: "art_visible",
				MissionID:  "mis_1",
				MediaType:  "text/plain",
				ByteSize:   int64(len("visible raw artifact needle")),
				Filename:   "notes.txt",
				Content:    []byte("visible raw artifact needle"),
			},
			{
				ArtifactID: "art_report",
				MissionID:  "mis_1",
				MediaType:  "text/markdown",
				ByteSize:   int64(len("hidden report artifact needle")),
				Filename:   "final.md",
				Content:    []byte("hidden report artifact needle"),
			},
		},
		events: []LedgerEvent{
			{
				EventID:   "evt_ordinary",
				MissionID: "mis_1",
				Sequence:  1,
				EventType: "source.observed",
				Payload:   mustJSON(map[string]any{"artifact_id": "art_visible", "artifact_ids": []string{"art_report"}, "note": "ordinary ledger needle"}),
			},
			{
				EventID:   "evt_report",
				MissionID: "mis_1",
				Sequence:  2,
				EventType: "report.final.created",
				Payload:   mustJSON(map[string]any{"artifact_id": "art_report", "artifact_ids": []string{"art_visible"}, "note": "report ledger needle"}),
			},
		},
	}
	svc := NewService(store)

	outline, err := svc.OutlineMission(ctx, "mis_1")
	if err != nil {
		t.Fatalf("OutlineMission returned error: %v", err)
	}
	if outline.Counts[ResearchIDEObjectSourceSnapshot] != 1 ||
		outline.Counts[ResearchIDEObjectRawArtifact] != 1 ||
		outline.Counts[ResearchIDEObjectLedgerEvent] != 1 {
		t.Fatalf("unexpected non-legacy counts: %#v", outline.Counts)
	}
	if containsSummaryID(outline.RecentLedgerEvents, "evt_report") || !containsSummaryID(outline.RecentLedgerEvents, "evt_ordinary") {
		t.Fatalf("unexpected recent ledger events: %#v", outline.RecentLedgerEvents)
	}
	ordinaryRecent, ok := findSummaryByID(outline.RecentLedgerEvents, "evt_ordinary")
	if !ok {
		t.Fatalf("missing ordinary recent event: %#v", outline.RecentLedgerEvents)
	}
	if !containsRef(ordinaryRecent.Refs, ResearchIDEObjectRawArtifact, "art_visible") || containsRef(ordinaryRecent.Refs, ResearchIDEObjectRawArtifact, "art_report") {
		t.Fatalf("expected non-legacy outline recent refs to hide report artifact only, got %#v", ordinaryRecent.Refs)
	}

	rawPage, err := svc.ListMissionObjects(ctx, "mis_1", ResearchIDEObjectRawArtifact, 10, "")
	if err != nil {
		t.Fatalf("ListMissionObjects raw_artifact returned error: %v", err)
	}
	if !containsSummaryID(rawPage.Items, "art_visible") || containsSummaryID(rawPage.Items, "art_report") {
		t.Fatalf("unexpected non-legacy raw artifacts: %#v", rawPage.Items)
	}

	eventPage, err := svc.ListMissionObjects(ctx, "mis_1", ResearchIDEObjectLedgerEvent, 10, "")
	if err != nil {
		t.Fatalf("ListMissionObjects ledger_event returned error: %v", err)
	}
	if !containsSummaryID(eventPage.Items, "evt_ordinary") || containsSummaryID(eventPage.Items, "evt_report") {
		t.Fatalf("unexpected non-legacy ledger events: %#v", eventPage.Items)
	}
	if len(eventPage.Items) != 1 || containsResearchIDERef(eventPage.Items[0].Refs, ResearchIDEObjectRef{ObjectKind: ResearchIDEObjectRawArtifact, ObjectID: "art_report"}) {
		t.Fatalf("expected hidden report artifact ref to be removed from visible event summary: %#v", eventPage.Items)
	}

	assertGrepHasMatch(t, svc, "mis_1", "visible raw artifact needle", ResearchIDEObjectRawArtifact, "art_visible")
	assertGrepHasMatch(t, svc, "mis_1", "ordinary ledger needle", ResearchIDEObjectLedgerEvent, "evt_ordinary")
	assertGrepHasNoMatch(t, svc, "mis_1", "hidden report artifact needle")
	assertGrepHasNoMatch(t, svc, "mis_1", "report ledger needle")

	visibleRefs, err := svc.ListObjectReferences(ctx, "mis_1", ResearchIDEObjectRawArtifact, "art_visible", 10, "")
	if err != nil {
		t.Fatalf("ListObjectReferences visible raw artifact returned error: %v", err)
	}
	if !containsRef(visibleRefs.Backward, ResearchIDEObjectLedgerEvent, "evt_ordinary") || containsRef(visibleRefs.Backward, ResearchIDEObjectLedgerEvent, "evt_report") {
		t.Fatalf("unexpected visible raw artifact backward refs: %#v", visibleRefs.Backward)
	}

	ordinaryRefs, err := svc.ListObjectReferences(ctx, "mis_1", ResearchIDEObjectLedgerEvent, "evt_ordinary", 10, "")
	if err != nil {
		t.Fatalf("ListObjectReferences ordinary ledger event returned error: %v", err)
	}
	if !containsRef(ordinaryRefs.Forward, ResearchIDEObjectRawArtifact, "art_visible") || containsRef(ordinaryRefs.Forward, ResearchIDEObjectRawArtifact, "art_report") {
		t.Fatalf("unexpected ordinary ledger event forward refs: %#v", ordinaryRefs.Forward)
	}
	assertReferencesInvalidInput(t, svc, "mis_1", ResearchIDEObjectRawArtifact, "art_report")
	assertReferencesInvalidInput(t, svc, "mis_1", ResearchIDEObjectLedgerEvent, "evt_report")

	reportArtifact, err := svc.ReadMissionObject(ctx, ResearchIDEReadRequest{MissionID: "mis_1", ObjectKind: ResearchIDEObjectRawArtifact, ObjectID: "art_report"})
	if err != nil {
		t.Fatalf("ReadMissionObject report raw artifact returned error: %v", err)
	}
	if !strings.Contains(reportArtifact.Data, "hidden report artifact needle") {
		t.Fatalf("expected direct report artifact read, got %s", reportArtifact.Data)
	}
	reportEvent, err := svc.ReadMissionObject(ctx, ResearchIDEReadRequest{MissionID: "mis_1", ObjectKind: ResearchIDEObjectLedgerEvent, ObjectID: "evt_report"})
	if err != nil {
		t.Fatalf("ReadMissionObject report ledger event returned error: %v", err)
	}
	if !strings.Contains(reportEvent.Data, "report.final.created") {
		t.Fatalf("expected direct report event read, got %s", reportEvent.Data)
	}

	legacyRawPage, err := svc.ListMissionObjectsLegacy(ctx, "mis_1", ResearchIDEObjectRawArtifact, 10, "")
	if err != nil {
		t.Fatalf("ListMissionObjectsLegacy raw_artifact returned error: %v", err)
	}
	if !containsSummaryID(legacyRawPage.Items, "art_visible") || !containsSummaryID(legacyRawPage.Items, "art_report") {
		t.Fatalf("expected legacy raw artifact list to remain visible, got %#v", legacyRawPage.Items)
	}
	legacyEventPage, err := svc.ListMissionObjectsLegacy(ctx, "mis_1", ResearchIDEObjectLedgerEvent, 10, "")
	if err != nil {
		t.Fatalf("ListMissionObjectsLegacy ledger_event returned error: %v", err)
	}
	if !containsSummaryID(legacyEventPage.Items, "evt_ordinary") || !containsSummaryID(legacyEventPage.Items, "evt_report") {
		t.Fatalf("expected legacy ledger event list to remain visible, got %#v", legacyEventPage.Items)
	}
	legacyOutline, err := svc.OutlineMissionLegacy(ctx, "mis_1")
	if err != nil {
		t.Fatalf("OutlineMissionLegacy returned error: %v", err)
	}
	legacyOrdinaryRecent, ok := findSummaryByID(legacyOutline.RecentLedgerEvents, "evt_ordinary")
	if !ok {
		t.Fatalf("missing legacy ordinary recent event: %#v", legacyOutline.RecentLedgerEvents)
	}
	if !containsRef(legacyOrdinaryRecent.Refs, ResearchIDEObjectRawArtifact, "art_report") {
		t.Fatalf("expected legacy outline recent refs to retain report artifact, got %#v", legacyOrdinaryRecent.Refs)
	}
	legacyReportArtifactRefs, err := svc.ListObjectReferencesLegacy(ctx, "mis_1", ResearchIDEObjectRawArtifact, "art_report", 10, "")
	if err != nil {
		t.Fatalf("ListObjectReferencesLegacy report raw artifact returned error: %v", err)
	}
	if !containsRef(legacyReportArtifactRefs.Backward, ResearchIDEObjectLedgerEvent, "evt_report") {
		t.Fatalf("expected legacy references to include report event, got %#v", legacyReportArtifactRefs.Backward)
	}
	legacyReportEventRefs, err := svc.ListObjectReferencesLegacy(ctx, "mis_1", ResearchIDEObjectLedgerEvent, "evt_report", 10, "")
	if err != nil {
		t.Fatalf("ListObjectReferencesLegacy report ledger event returned error: %v", err)
	}
	if !containsRef(legacyReportEventRefs.Forward, ResearchIDEObjectRawArtifact, "art_report") {
		t.Fatalf("expected legacy report event refs to include report artifact, got %#v", legacyReportEventRefs.Forward)
	}
}

func TestListObjectReferencesInvalidKindDoesNotConsultLedgerStore(t *testing.T) {
	store := &researchIDEFailingLedgerStore{}
	svc := NewService(store)

	_, err := svc.ListObjectReferences(context.Background(), "mis_1", ResearchIDEObjectReport, "rpt_1", 10, "")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for unsupported object kind, got %v", err)
	}
	if store.ledgerRead {
		t.Fatalf("expected unsupported object kind to fail before ledger lookup")
	}
}

type researchIDEFakeStore struct {
	fakeStore
	snapshot SourceSnapshot
	artifact RawArtifact
}

func (store *researchIDEFakeStore) GetSourceSnapshot(_ context.Context, snapshotID string) (SourceSnapshot, error) {
	if store.snapshot.SnapshotID == snapshotID {
		return store.snapshot, nil
	}
	return SourceSnapshot{}, fmt.Errorf("missing source snapshot")
}

func (store *researchIDEFakeStore) ListSourceSnapshots(_ context.Context, missionID string) ([]SourceSnapshot, error) {
	if store.snapshot.MissionID == missionID {
		return []SourceSnapshot{store.snapshot}, nil
	}
	return nil, nil
}

func (store *researchIDEFakeStore) GetRawArtifact(_ context.Context, artifactID string) (RawArtifact, error) {
	if store.artifact.ArtifactID == artifactID {
		return store.artifact, nil
	}
	return RawArtifact{}, fmt.Errorf("missing raw artifact")
}

func (store *researchIDEFakeStore) ListRawArtifacts(_ context.Context, missionID string) ([]RawArtifact, error) {
	if store.artifact.MissionID == missionID {
		return []RawArtifact{store.artifact}, nil
	}
	return nil, nil
}

type researchIDEGrepFakeStore struct {
	researchIDEFakeStore
	events []LedgerEvent
}

func (store *researchIDEGrepFakeStore) ListLedgerEvents(_ context.Context, missionID string) ([]LedgerEvent, error) {
	var events []LedgerEvent
	for _, event := range store.events {
		if event.MissionID == missionID {
			events = append(events, event)
		}
	}
	return events, nil
}

type researchIDEVisibilityStore struct {
	fakeStore
	projection MissionProjection
	snapshots  []SourceSnapshot
	artifacts  []RawArtifact
	events     []LedgerEvent
}

func (store *researchIDEVisibilityStore) GetMissionProjection(_ context.Context, missionID string) (MissionProjection, error) {
	if store.projection.MissionID == missionID {
		return store.projection, nil
	}
	return MissionProjection{}, fmt.Errorf("missing mission projection")
}

func (store *researchIDEVisibilityStore) GetSourceSnapshot(_ context.Context, snapshotID string) (SourceSnapshot, error) {
	for _, snapshot := range store.snapshots {
		if snapshot.SnapshotID == snapshotID {
			return snapshot, nil
		}
	}
	return SourceSnapshot{}, fmt.Errorf("missing source snapshot")
}

func (store *researchIDEVisibilityStore) ListSourceSnapshots(_ context.Context, missionID string) ([]SourceSnapshot, error) {
	var snapshots []SourceSnapshot
	for _, snapshot := range store.snapshots {
		if snapshot.MissionID == missionID {
			snapshots = append(snapshots, snapshot)
		}
	}
	return snapshots, nil
}

func (store *researchIDEVisibilityStore) GetRawArtifact(_ context.Context, artifactID string) (RawArtifact, error) {
	for _, artifact := range store.artifacts {
		if artifact.ArtifactID == artifactID {
			return artifact, nil
		}
	}
	return RawArtifact{}, fmt.Errorf("missing raw artifact")
}

func (store *researchIDEVisibilityStore) ListRawArtifacts(_ context.Context, missionID string) ([]RawArtifact, error) {
	var artifacts []RawArtifact
	for _, artifact := range store.artifacts {
		if artifact.MissionID == missionID {
			artifacts = append(artifacts, artifact)
		}
	}
	return artifacts, nil
}

func (store *researchIDEVisibilityStore) ListLedgerEvents(_ context.Context, missionID string) ([]LedgerEvent, error) {
	var events []LedgerEvent
	for _, event := range store.events {
		if event.MissionID == missionID {
			events = append(events, event)
		}
	}
	return events, nil
}

func (store *researchIDEVisibilityStore) ListEvidenceRecords(context.Context, string) ([]EvidenceRecord, error) {
	return nil, nil
}

func (store *researchIDEVisibilityStore) ListClaimRecords(context.Context, string) ([]ClaimRecord, error) {
	return nil, nil
}

func (store *researchIDEVisibilityStore) ListQuestionRecords(context.Context, string) ([]QuestionRecord, error) {
	return nil, nil
}

func (store *researchIDEVisibilityStore) ListOptionRecords(context.Context, string) ([]OptionRecord, error) {
	return nil, nil
}

func (store *researchIDEVisibilityStore) ListProposalBundles(context.Context, string) ([]ProposalBundle, error) {
	return nil, nil
}

func (store *researchIDEVisibilityStore) ListReports(context.Context, string) ([]Report, error) {
	return nil, nil
}

func (store *researchIDEVisibilityStore) ListReportVersions(context.Context, string) ([]ReportVersion, error) {
	return nil, nil
}

type researchIDEFailingLedgerStore struct {
	fakeStore
	ledgerRead bool
}

func (store *researchIDEFailingLedgerStore) ListLedgerEvents(context.Context, string) ([]LedgerEvent, error) {
	store.ledgerRead = true
	return nil, errors.New("ledger should not be read")
}

func containsSummaryID(items []ResearchIDEObjectSummary, objectID string) bool {
	for _, item := range items {
		if item.ObjectID == objectID {
			return true
		}
	}
	return false
}

func findSummaryByID(items []ResearchIDEObjectSummary, objectID string) (ResearchIDEObjectSummary, bool) {
	for _, item := range items {
		if item.ObjectID == objectID {
			return item, true
		}
	}
	return ResearchIDEObjectSummary{}, false
}

func containsRef(refs []ResearchIDEObjectRef, objectKind string, objectID string) bool {
	return containsResearchIDERef(refs, ResearchIDEObjectRef{ObjectKind: objectKind, ObjectID: objectID})
}

func assertReferencesInvalidInput(t *testing.T, svc *Service, missionID string, objectKind string, objectID string) {
	t.Helper()
	_, err := svc.ListObjectReferences(context.Background(), missionID, objectKind, objectID, 10, "")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ListObjectReferences(%s/%s) to fail with ErrInvalidInput, got %v", objectKind, objectID, err)
	}
}

func assertGrepHasMatch(t *testing.T, svc *Service, missionID string, query string, objectKind string, objectID string) {
	t.Helper()
	result, err := svc.GrepMissionObjects(context.Background(), missionID, query, 10, "")
	if err != nil {
		t.Fatalf("GrepMissionObjects(%q) returned error: %v", query, err)
	}
	for _, match := range result.Matches {
		if match.ObjectKind == objectKind && match.ObjectID == objectID {
			return
		}
	}
	t.Fatalf("expected grep query %q to match %s/%s, got %#v", query, objectKind, objectID, result.Matches)
}

func assertGrepHasNoMatch(t *testing.T, svc *Service, missionID string, query string) {
	t.Helper()
	result, err := svc.GrepMissionObjects(context.Background(), missionID, query, 10, "")
	if err != nil {
		t.Fatalf("GrepMissionObjects(%q) returned error: %v", query, err)
	}
	if len(result.Matches) != 0 {
		t.Fatalf("expected grep query %q to have no matches, got %#v", query, result.Matches)
	}
}

func testResearchIDEPDFBytes(t *testing.T, lines []string) []byte {
	t.Helper()
	var stream bytes.Buffer
	stream.WriteString("BT\n/F1 12 Tf\n72 720 Td\n")
	for i, line := range lines {
		if i > 0 {
			stream.WriteString("0 -18 Td\n")
		}
		fmt.Fprintf(&stream, "(%s) Tj\n", escapeResearchIDEPDFString(line))
	}
	stream.WriteString("ET\n")
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	if _, err := zw.Write(stream.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d /Filter /FlateDecode >>\nstream\n%s\nendstream", compressed.Len(), compressed.String()),
	}
	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for i, obj := range objects {
		offsets = append(offsets, out.Len())
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return out.Bytes()
}

func escapeResearchIDEPDFString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `(`, `\(`)
	value = strings.ReplaceAll(value, `)`, `\)`)
	return value
}
