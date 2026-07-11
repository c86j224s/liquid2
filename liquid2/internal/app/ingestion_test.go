package app

import (
	"bytes"
	"context"
	"fmt"
	"testing"
)

func TestCreateBookmarkDocumentAssignsFolderTagsAndURL(t *testing.T) {
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()
	folder := createIngestFolder(t, ctx, service)
	tag := createIngestTag(t, ctx, service)

	detail, err := service.CreateBookmarkDocument(ctx, BookmarkDocumentInput{
		URL: "https://example.com/a", SourceURL: "https://example.com/a#x",
		Title: "Example", FolderID: folder.ID, TagIDs: []string{tag.ID},
	})
	if err != nil {
		t.Fatalf("create bookmark: %v", err)
	}
	if detail.Document.Kind != DocumentKindBookmark || *detail.Document.FolderID != folder.ID {
		t.Fatalf("unexpected document metadata %#v", detail.Document)
	}
	if *detail.Document.CanonicalURL != "https://example.com/a" || *detail.Document.SourceURL != "https://example.com/a#x" {
		t.Fatalf("unexpected URLs %#v", detail.Document)
	}
	if len(detail.Tags) != 1 || detail.Tags[0].ID != tag.ID {
		t.Fatalf("unexpected tags %#v", detail.Tags)
	}
	if detail.Contents != nil {
		t.Fatalf("expected nil bookmark contents, got %#v", detail.Contents)
	}
}

func TestCreateScrapedDocumentStoresContent(t *testing.T) {
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })

	detail, err := service.CreateScrapedDocument(context.Background(), ScrapedDocumentInput{
		URL: "https://example.com/a", SourceURL: "https://example.com/a",
		Title: "Example", Content: "Readable body", Format: ContentFormatText,
	})
	if err != nil {
		t.Fatalf("create scraped document: %v", err)
	}
	if detail.Document.Kind != DocumentKindScrapedArticle {
		t.Fatalf("expected scraped kind, got %#v", detail.Document)
	}
	if len(detail.Contents) != 1 || detail.Contents[0].Content != "Readable body" {
		t.Fatalf("unexpected contents %#v", detail.Contents)
	}
}

func TestCreateUploadedDocumentStoresBlobAndExtractedContent(t *testing.T) {
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })

	detail, err := service.CreateUploadedDocument(context.Background(), UploadedDocumentInput{
		Filename: "note.txt", MimeType: "text/plain",
		Data: []byte("Stored body"), Content: "Stored body", Format: ContentFormatText,
	})
	if err != nil {
		t.Fatalf("create uploaded document: %v", err)
	}
	if detail.Document.Kind != DocumentKindUploadedFile || detail.Document.Title != "note.txt" {
		t.Fatalf("unexpected document metadata %#v", detail.Document)
	}
	if len(detail.Blobs) != 1 || detail.Blobs[0].Size != int64(len("Stored body")) {
		t.Fatalf("unexpected blobs %#v", detail.Blobs)
	}
	if len(detail.Contents) != 1 || detail.Contents[0].Content != "Stored body" {
		t.Fatalf("unexpected contents %#v", detail.Contents)
	}
	blobID := detail.Blobs[0].ID
	if err := service.repo.View(context.Background(), func(tx RepositoryReader) error {
		record, ok := tx.Document(detail.Document.ID)
		if !ok {
			return fmt.Errorf("document missing")
		}
		if !bytes.Equal(record.blobData[blobID], []byte("Stored body")) {
			return fmt.Errorf("unexpected blob data %q", string(record.blobData[blobID]))
		}
		record.blobData[blobID][0] = 'X'
		return nil
	}); err != nil {
		t.Fatalf("view stored blob data: %v", err)
	}
	if err := service.repo.View(context.Background(), func(tx RepositoryReader) error {
		record, ok := tx.Document(detail.Document.ID)
		if !ok {
			return fmt.Errorf("document missing")
		}
		if !bytes.Equal(record.blobData[blobID], []byte("Stored body")) {
			return fmt.Errorf("blob data was not isolated")
		}
		return nil
	}); err != nil {
		t.Fatalf("view isolated blob data: %v", err)
	}
}

func createIngestFolder(t *testing.T, ctx context.Context, service *Service) Folder {
	t.Helper()
	folder, err := service.CreateFolder(ctx, FolderInput{Name: "Inbox"})
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	return folder
}

func createIngestTag(t *testing.T, ctx context.Context, service *Service) Tag {
	t.Helper()
	tag, err := service.CreateTag(ctx, "go")
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}
	return tag
}
