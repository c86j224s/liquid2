package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestNewJSONLoggerUsesConfiguredLevel(t *testing.T) {
	var output bytes.Buffer
	logger, err := New(&output, Config{Level: "trace", Format: "json"})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}

	logger.Debug("debug event", slog.String("component", "test"))
	logger.LogAttrs(context.Background(), LevelTrace, "trace event", slog.String("component", "test"))

	logs := output.String()
	if !strings.Contains(logs, `"level":"debug"`) {
		t.Fatalf("expected debug log, got %q", logs)
	}
	if !strings.Contains(logs, `"level":"trace"`) {
		t.Fatalf("expected trace log, got %q", logs)
	}
}

func TestNewLoggerRejectsInvalidConfig(t *testing.T) {
	if _, err := New(&bytes.Buffer{}, Config{Level: "verbose"}); err == nil {
		t.Fatal("expected invalid level error")
	}
	if _, err := New(&bytes.Buffer{}, Config{Format: "yaml"}); err == nil {
		t.Fatal("expected invalid format error")
	}
}

func TestDefaultLoggerConfig(t *testing.T) {
	var output bytes.Buffer
	logger, err := New(&output, Config{})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}

	logger.Debug("hidden")
	logger.Info("visible")

	logs := output.String()
	if strings.Contains(logs, "hidden") {
		t.Fatalf("expected debug log to be filtered, got %q", logs)
	}
	if !strings.Contains(logs, `"level":"info"`) {
		t.Fatalf("expected info JSON log, got %q", logs)
	}
}
