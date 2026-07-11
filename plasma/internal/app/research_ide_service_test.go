package app

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
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
