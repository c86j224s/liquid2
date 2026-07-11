package main

import (
	"io"
	"log/slog"
	"strings"
	"testing"
)

func TestReadJobRuntimeConfigDefaultsJobsOff(t *testing.T) {
	clearJobRuntimeEnv(t)
	config, err := readJobRuntimeConfig()
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if config.jobsEnabled {
		t.Fatalf("expected runtime disabled by default, got %#v", config)
	}
}

func TestReadJobRuntimeConfigEnablesJobs(t *testing.T) {
	clearJobRuntimeEnv(t)
	t.Setenv("LIQUID2_JOBS_ENABLED", "1")
	config, err := readJobRuntimeConfig()
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !config.jobsEnabled {
		t.Fatalf("expected runtime enabled, got %#v", config)
	}
}

func TestReadJobRuntimeConfigNormalizesTranslationProvider(t *testing.T) {
	clearJobRuntimeEnv(t)
	t.Setenv("LIQUID2_TRANSLATION_PROVIDER", " Passthrough ")
	config, err := readJobRuntimeConfig()
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if config.translationProvider != "passthrough" {
		t.Fatalf("unexpected translation provider %q", config.translationProvider)
	}
}

func clearJobRuntimeEnv(t *testing.T) {
	t.Helper()
	t.Setenv("LIQUID2_JOBS_ENABLED", "")
	t.Setenv("LIQUID2_TRANSLATION_PROVIDER", "")
	t.Setenv("LIQUID2_CODEX_COMMAND", "")
	t.Setenv("LIQUID2_CODEX_MODEL", "")
	t.Setenv("LIQUID2_CODEX_TIMEOUT_SECONDS", "")
}

func TestRunRuntimeReturnsPanicsAsErrors(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	err := runRuntime(logger, "test runtime", func() error {
		panic("boom")
	})
	if err == nil || !strings.Contains(err.Error(), "test runtime panicked") {
		t.Fatalf("expected panic error, got %v", err)
	}
}
