package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type memoryExportWriter struct {
	files map[string][]byte
}

func newMemoryExportWriter() *memoryExportWriter {
	return &memoryExportWriter{files: map[string][]byte{}}
}

func (writer *memoryExportWriter) WriteFile(_ context.Context, path string, data []byte) error {
	writer.files[path] = append([]byte(nil), data...)
	return nil
}

func TestExportMarkdownWritesManifestDocumentsAndBlobs(t *testing.T) {
	ctx := context.Background()
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	folder := createIngestFolder(t, ctx, service)
	tag := createIngestTag(t, ctx, service)
	detail, err := service.CreateUploadedDocument(ctx, UploadedDocumentInput{
		Filename: "Report Final.pdf", MimeType: "application/pdf",
		Data: []byte("PDF bytes"), Content: "Extracted text", Format: ContentFormatText,
		FolderID: folder.ID, TagIDs: []string{tag.ID},
	})
	if err != nil {
		t.Fatalf("create uploaded document: %v", err)
	}
	writer := newMemoryExportWriter()
	version := int64(5)

	result, err := service.ExportMarkdown(ctx, MarkdownExportInput{
		ExportID: "export_1", CreatedAt: 1760000000000, SchemaVersion: &version,
	}, writer)
	if err != nil {
		t.Fatalf("export markdown: %v", err)
	}
	if result.DocumentCount != 1 || result.BlobCount != 1 {
		t.Fatalf("unexpected counts %#v", result)
	}
	docPath := "documents/" + detail.Document.ID + ".md"
	if !bytes.Contains(writer.files[docPath], []byte("# Report Final.pdf")) {
		t.Fatalf("expected markdown document at %s, got %q", docPath, string(writer.files[docPath]))
	}
	if !bytes.Contains(writer.files[docPath], []byte("Extracted text")) {
		t.Fatalf("expected exported text content, got %q", string(writer.files[docPath]))
	}
	blobPath := result.Manifest.Documents[0].Blobs[0].Path
	if !bytes.Equal(writer.files[blobPath], []byte("PDF bytes")) {
		t.Fatalf("expected blob bytes at %s, got %q", blobPath, string(writer.files[blobPath]))
	}
	var manifest ExportManifest
	if err = json.Unmarshal(writer.files["manifest.json"], &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if manifest.ManifestVersion != 1 || manifest.ExportID != "export_1" {
		t.Fatalf("unexpected manifest identity %#v", manifest)
	}
	doc := manifest.Documents[0]
	if doc.FolderID == nil || *doc.FolderID != folder.ID || len(doc.Tags) != 1 || doc.Tags[0].Slug != tag.Slug {
		t.Fatalf("expected folder and tag in manifest: %#v", doc)
	}
	if strings.Contains(string(writer.files["manifest.json"]), "/tmp") ||
		strings.HasPrefix(blobPath, "/") || strings.Contains(blobPath, "..") {
		t.Fatalf("manifest contains unsafe path data: %s", string(writer.files["manifest.json"]))
	}
	if doc.Blobs[0].SizeBytes != int64(len("PDF bytes")) || len(doc.Blobs[0].SHA256) != 64 {
		t.Fatalf("unexpected blob manifest %#v", doc.Blobs[0])
	}
}

func TestExportMarkdownSkipsDeletedDocuments(t *testing.T) {
	ctx := context.Background()
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	detail, err := service.CreateScrapedDocument(ctx, ScrapedDocumentInput{
		URL: "https://example.com/a", Title: "Deleted", Content: "Gone",
	})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	if _, err = service.DeleteDocument(ctx, detail.Document.ID); err != nil {
		t.Fatalf("delete document: %v", err)
	}
	writer := newMemoryExportWriter()
	result, err := service.ExportMarkdown(ctx, MarkdownExportInput{ExportID: "export_1"}, writer)
	if err != nil {
		t.Fatalf("export markdown: %v", err)
	}
	if result.DocumentCount != 0 || len(result.Manifest.Documents) != 0 {
		t.Fatalf("expected deleted document skipped, got %#v", result.Manifest.Documents)
	}
}

func TestExportMarkdownUsesDeterministicOrderAndRenderingRules(t *testing.T) {
	ctx := context.Background()
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	markdownDoc, err := service.CreateScrapedDocument(ctx, ScrapedDocumentInput{
		URL: "https://example.com/markdown", Title: "Markdown",
		Content: "## Native Section", Format: ContentFormatMarkdown,
	})
	if err != nil {
		t.Fatalf("create markdown document: %v", err)
	}
	htmlDoc, err := service.CreateScrapedDocument(ctx, ScrapedDocumentInput{
		URL: "https://example.com/html", Title: "HTML",
		Content: "<p>Hello</p>", Format: ContentFormatHTML,
	})
	if err != nil {
		t.Fatalf("create html document: %v", err)
	}
	writer := newMemoryExportWriter()

	result, err := service.ExportMarkdown(ctx, MarkdownExportInput{
		ExportID: "export_1", DocumentIDs: []string{htmlDoc.Document.ID, markdownDoc.Document.ID},
	}, writer)
	if err != nil {
		t.Fatalf("export markdown: %v", err)
	}
	if got := result.Manifest.Documents[0].ID; got != markdownDoc.Document.ID {
		t.Fatalf("expected deterministic ID order, got first %q", got)
	}
	markdownPath := result.Manifest.Documents[0].MarkdownPath
	if !bytes.Contains(writer.files[markdownPath], []byte("## Native Section")) {
		t.Fatalf("expected markdown content to pass through, got %q", string(writer.files[markdownPath]))
	}
	htmlPath := result.Manifest.Documents[1].MarkdownPath
	if !bytes.Contains(writer.files[htmlPath], []byte("```html\n<p>Hello</p>\n```")) {
		t.Fatalf("expected HTML fenced block, got %q", string(writer.files[htmlPath]))
	}
}

func TestExportMarkdownRejectsMissingSelectedDocument(t *testing.T) {
	service := NewService()
	t.Cleanup(func() { _ = service.Close() })
	_, err := service.ExportMarkdown(context.Background(), MarkdownExportInput{
		ExportID: "export_1", DocumentIDs: []string{"missing"},
	}, newMemoryExportWriter())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestExportMarkdownRejectsInvalidInput(t *testing.T) {
	service := NewService()
	t.Cleanup(func() { _ = service.Close() })
	_, err := service.ExportMarkdown(context.Background(), MarkdownExportInput{}, newMemoryExportWriter())
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation for missing export id, got %v", err)
	}
	_, err = service.ExportMarkdown(context.Background(), MarkdownExportInput{ExportID: "export_1"}, nil)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation for nil writer, got %v", err)
	}
}
