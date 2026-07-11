package translation

import (
	"errors"
	"testing"
)

func TestEncodeDecodeTranslateDocumentPayload(t *testing.T) {
	raw, err := EncodeTranslateDocumentPayload(" doc_1 ", " content_1 ", " KO ")
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	payload, err := DecodeTranslateDocumentPayload(raw)
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.DocumentID != "doc_1" {
		t.Fatalf("unexpected document id %q", payload.DocumentID)
	}
	if payload.SourceContentID != "content_1" {
		t.Fatalf("unexpected source content id %q", payload.SourceContentID)
	}
	if payload.TargetLanguage != "ko" {
		t.Fatalf("expected normalized language, got %q", payload.TargetLanguage)
	}
}

func TestDecodeTranslateDocumentPayloadRejectsInvalidPayload(t *testing.T) {
	tests := []string{
		`not-json`,
		`{"sourceContentId":"content_1","targetLanguage":"ko"}`,
		`{"documentId":"doc_1","targetLanguage":"ko"}`,
		`{"documentId":"doc_1","sourceContentId":"content_1"}`,
		`{"documentId":"doc_1","sourceContentId":"content_1","targetLanguage":"??"}`,
	}
	for _, raw := range tests {
		_, err := DecodeTranslateDocumentPayload(raw)
		if !errors.Is(err, ErrInvalidJobPayload) {
			t.Fatalf("expected invalid payload for %s, got %v", raw, err)
		}
	}
}
