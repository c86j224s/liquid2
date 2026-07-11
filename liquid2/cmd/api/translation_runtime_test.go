package main

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/internal/app"
	"github.com/c86j224s/liquid2/internal/jobs"
	"github.com/c86j224s/liquid2/internal/translation"
)

func TestNewTranslationHandlerDisabledByDefault(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	handler, enabled, err := newTranslationHandler(nil, service, "")
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	if enabled || handler != nil {
		t.Fatalf("expected translation handler disabled, enabled=%v handler=%v", enabled, handler)
	}
}

func TestNewTranslationHandlerRejectsUnknownProvider(t *testing.T) {
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	_, _, err := newTranslationHandler(nil, service, "remote")
	if err == nil {
		t.Fatal("expected unknown provider error")
	}
}

func TestNewTranslationHandlerUsesPassthroughProvider(t *testing.T) {
	ctx := context.Background()
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	source, err := service.CreateScrapedDocument(ctx, app.ScrapedDocumentInput{
		URL: "https://example.com/a", Title: "Article", Content: "Original body",
	})
	if err != nil {
		t.Fatalf("create source document: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler, enabled, err := newTranslationHandler(logger, service, "passthrough")
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	if !enabled || handler == nil {
		t.Fatalf("expected translation handler enabled")
	}
	payload, err := translation.EncodeTranslateDocumentPayload(source.Document.ID, source.Contents[0].ID, "ko")
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	job := jobs.Job{ID: "job_1", Kind: jobs.KindTranslateDocument, PayloadJSON: payload}
	if err = handler(ctx, job); err != nil {
		t.Fatalf("handle translation job: %v", err)
	}
	detail, err := service.GetDocument(ctx, source.Document.ID)
	if err != nil {
		t.Fatalf("get document: %v", err)
	}
	if len(detail.Contents) != 2 || detail.Contents[1].Content != "Original body" {
		t.Fatalf("expected passthrough translation, got %#v", detail.Contents)
	}
}

func TestNewTranslationProviderSupportsCodex(t *testing.T) {
	clearJobRuntimeEnv(t)
	t.Setenv("LIQUID2_CODEX_COMMAND", "codex")
	provider, err := newTranslationProvider("codex")
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	if _, ok := provider.(translation.CodexProvider); !ok {
		t.Fatalf("expected codex provider, got %T", provider)
	}
}

func TestNewTranslationProviderRejectsInvalidCodexTimeout(t *testing.T) {
	clearJobRuntimeEnv(t)
	t.Setenv("LIQUID2_CODEX_TIMEOUT_SECONDS", "0")
	_, err := newTranslationProvider("codex")
	if err == nil || !strings.Contains(err.Error(), "LIQUID2_CODEX_TIMEOUT_SECONDS") {
		t.Fatalf("expected codex timeout error, got %v", err)
	}
}

func TestStartJobRuntimeRequiresQueueForTranslationProvider(t *testing.T) {
	clearJobRuntimeEnv(t)
	t.Setenv("LIQUID2_TRANSLATION_PROVIDER", "passthrough")
	service := app.NewService()
	t.Cleanup(func() { _ = service.Close() })
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	status, cleanup, err := startJobRuntime(context.Background(), logger, service, nil)
	if err == nil || !strings.Contains(err.Error(), "LIQUID2_DB_PATH") {
		t.Fatalf("expected queue requirement error, got status=%#v cleanup_nil=%v err=%v", status, cleanup == nil, err)
	}
}
