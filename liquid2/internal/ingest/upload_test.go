package ingest

import (
	"bytes"
	"errors"
	"testing"
)

func TestPrepareUploadExtractsText(t *testing.T) {
	upload, err := PrepareUpload(UploadInput{
		Filename:    "note.txt",
		ContentType: "text/plain",
		Data:        []byte("stored body"),
	})
	if err != nil {
		t.Fatalf("prepare upload: %v", err)
	}
	if upload.MimeType != "text/plain" || upload.Content != "stored body" {
		t.Fatalf("unexpected upload %#v", upload)
	}
	if !bytes.Equal(upload.Data, []byte("stored body")) {
		t.Fatalf("unexpected upload bytes %q", upload.Data)
	}
}

func TestPrepareUploadInfersMarkdownFromExtension(t *testing.T) {
	upload, err := PrepareUpload(UploadInput{
		Filename:    "note.md",
		ContentType: "application/octet-stream",
		Data:        []byte("# Stored body"),
	})
	if err != nil {
		t.Fatalf("prepare upload: %v", err)
	}
	if upload.MimeType != "text/markdown" || upload.Format != FormatMarkdown {
		t.Fatalf("unexpected upload %#v", upload)
	}
}

func TestPrepareUploadRejectsMismatchedDeclaredContent(t *testing.T) {
	_, err := PrepareUpload(UploadInput{
		Filename:    "note.txt",
		ContentType: "text/plain",
		Data:        []byte{0xff, 0x00, 0xfe},
	})
	if !errors.Is(err, ErrUnsupportedMedia) {
		t.Fatalf("expected unsupported media, got %v", err)
	}

	_, err = PrepareUpload(UploadInput{
		Filename:    "document.pdf",
		ContentType: "application/pdf",
		Data:        []byte("not a pdf"),
	})
	if !errors.Is(err, ErrUnsupportedMedia) {
		t.Fatalf("expected unsupported media, got %v", err)
	}
}

func TestPrepareUploadRejectsUnsupportedAndOversizedFiles(t *testing.T) {
	_, err := PrepareUpload(UploadInput{
		Filename:    "image.png",
		ContentType: "image/png",
		Data:        []byte("not really png"),
	})
	if !errors.Is(err, ErrUnsupportedMedia) {
		t.Fatalf("expected unsupported media, got %v", err)
	}

	_, err = PrepareUpload(UploadInput{
		Filename:    "large.txt",
		ContentType: "text/plain",
		Data:        bytes.Repeat([]byte("x"), MaxUploadBytes+1),
	})
	if !errors.Is(err, ErrPayloadTooLarge) {
		t.Fatalf("expected payload too large, got %v", err)
	}
}
