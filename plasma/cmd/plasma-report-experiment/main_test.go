package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/reportexperiment"
)

func TestRunRequiresExplicitFlags(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), nil, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "archive-root") {
		t.Fatalf("stderr did not explain required flags: %s", stderr.String())
	}
}

func TestRunBuildsCodexOnlyExecutorConfig(t *testing.T) {
	old := runExperiment
	defer func() { runExperiment = old }()
	archive := t.TempDir()
	fixture := filepath.Join(archive, "fixture.json")
	if err := os.WriteFile(fixture, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	mcpMetadata, err := reportexperiment.ReadExecutableMetadata(executable)
	if err != nil {
		t.Fatal(err)
	}
	codexCommand := filepath.Join(archive, "codex-dev")
	if err := os.WriteFile(codexCommand, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	codexMetadata, err := reportexperiment.ReadExecutableMetadata(codexCommand)
	if err != nil {
		t.Fatal(err)
	}
	runExperiment = func(ctx context.Context, config reportexperiment.Config) (reportexperiment.Result, error) {
		if config.ArchiveRoot != archive || config.FixturePath != fixture || config.RunID != "cmd-run" {
			t.Fatalf("unexpected run config identity: %#v", config)
		}
		if config.RepositoryRoot == "" {
			t.Fatalf("repository root was not required")
		}
		if config.AgentModel != "gpt-test" || config.AgentReasoningEffort != "high" {
			t.Fatalf("unexpected Codex model settings: %#v", config)
		}
		if config.BinaryPair.Codex.Path != codexMetadata.Path || config.BinaryPair.Codex.SHA256 != codexMetadata.SHA256 {
			t.Fatalf("Codex binary metadata mismatch: %#v", config.BinaryPair.Codex)
		}
		runDir := filepath.Join(archive, "runs", "cmd-run")
		executor, err := config.ExecutorFactory(ctx, reportexperiment.ExecutorContext{RunDir: runDir, DBPath: filepath.Join(runDir, "plasma.db")})
		if err != nil {
			t.Fatal(err)
		}
		codex, ok := executor.(agentexec.CodexExecutor)
		if !ok {
			t.Fatalf("executor type = %T, want CodexExecutor", executor)
		}
		if codex.Command != codexMetadata.Path || codex.Timeout != 5*time.Minute {
			t.Fatalf("unexpected Codex command settings: %#v", codex)
		}
		if codex.MCPServer.Command != mcpMetadata.Path || strings.Join(codex.MCPServer.Args, " ") != "mcp -db "+filepath.Join(runDir, "plasma.db") {
			t.Fatalf("unexpected MCP server wiring: %#v", codex.MCPServer)
		}
		if len(codex.Env) != 0 {
			t.Fatalf("Codex Env must use product sanitized defaults, got: %#v", codex.Env)
		}
		return reportexperiment.Result{RunDir: runDir, ReportPath: filepath.Join(runDir, "report.md"), LedgerPath: filepath.Join(runDir, "ledger.json"), ManifestPath: filepath.Join(runDir, "manifest.json")}, nil
	}
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{
		"-archive-root", archive,
		"-fixture", fixture,
		"-run-id", "cmd-run",
		"-plasma-mcp-binary", executable,
		"-codex-command", codexCommand,
		"-codex-model", "gpt-test",
		"-codex-effort", "high",
		"-codex-timeout", "5m",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "manifest=") {
		t.Fatalf("stdout did not include output paths: %s", stdout.String())
	}
}

func TestFindRepositoryRootFromFailsOutsideRepository(t *testing.T) {
	if _, err := findRepositoryRootFrom(t.TempDir()); err == nil {
		t.Fatal("expected repository root lookup to fail outside a Git worktree")
	}
}
