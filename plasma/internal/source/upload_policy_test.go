package source

import (
	artifact "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"testing"
)

func TestUploadTextAndBinaryClassification(t *testing.T) {
	media, kind, err := ClassifyUploadedFile("notes.md", []byte("한글 notes"))
	if err != nil || kind != UploadedContentKindText || media != "text/markdown; charset=utf-8" {
		t.Fatalf("media=%q kind=%q err=%v", media, kind, err)
	}
	for _, tc := range []struct {
		name    string
		content []byte
	}{{"bad.pdf", []byte("not a PDF")}, {"bad.png", []byte("not an image")}, {"data.bin", []byte{0, 1, 2, 255}}} {
		if _, _, err := ClassifyUploadedFile(tc.name, tc.content); err == nil {
			t.Fatalf("accepted invalid upload %s", tc.name)
		}
	}
}

func TestUploadImageMetadataOnly(t *testing.T) {
	raw := artifact.Raw{ArtifactID: "art_one", MediaType: "image/png", Content: []byte("not returned")}
	if kind := UploadedArtifactReadKind(raw); kind != "metadata" {
		t.Fatalf("kind=%q", kind)
	}
	metadata := UploadedArtifactMetadata(raw)
	if metadata["read_kind"] != "metadata" || metadata["artifact_id"] != "art_one" {
		t.Fatalf("metadata=%v", metadata)
	}
	if _, ok := metadata["content"]; ok {
		t.Fatal("metadata contains content")
	}
	if name := SanitizeUploadedFilename("../", "application/pdf"); name != "uploaded-source.pdf" {
		t.Fatalf("name=%q", name)
	}
}
