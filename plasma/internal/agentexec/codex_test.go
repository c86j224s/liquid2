package agentexec

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCodexEnvironmentUsesAllowlist(t *testing.T) {
	t.Setenv("PATH", "/bin")
	t.Setenv("PLASMA_RUNTIME_MODE", "dev")
	t.Setenv("OPENAI_API_KEY", "should-not-be-inherited")

	env := codexEnvironment(nil)
	if !containsEnv(env, "PATH=/opt/homebrew/bin:/usr/local/bin:/bin:/usr/bin:/usr/sbin:/sbin") {
		t.Fatalf("expected PATH to be retained in %#v", env)
	}
	if !containsEnv(env, "PLASMA_RUNTIME_MODE=dev") {
		t.Fatalf("expected PLASMA_RUNTIME_MODE to be retained in %#v", env)
	}
	for _, value := range env {
		if strings.HasPrefix(value, "OPENAI_API_KEY=") {
			t.Fatalf("expected OPENAI_API_KEY to be scrubbed from %#v", env)
		}
	}
}

func TestCodexExecutorCreatesMissingWorkDir(t *testing.T) {
	dir := t.TempDir()
	workDir := filepath.Join(dir, "missing-workdir")
	command := filepath.Join(dir, "fake-codex")
	script := `#!/bin/sh
out=""
want_out=0
for arg in "$@"; do
  if [ "$want_out" = "1" ]; then
    out="$arg"
    want_out=0
  elif [ "$arg" = "--output-last-message" ]; then
    want_out=1
  fi
done
cat >/dev/null
printf 'session id: created-workdir-session\n'
printf 'done' > "$out"
`
	if err := os.WriteFile(command, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	result, err := (CodexExecutor{
		Command: command,
		WorkDir: workDir,
		Timeout: 10 * time.Second,
		Env:     []string{"PATH=/usr/bin:/bin"},
	}).Run(context.Background(), AgentRequest{
		Prompt:        "test prompt",
		MissionID:     "mis_1",
		ToolSessionID: "ses_1",
		AgentExecutor: "codex",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.SessionID != "created-workdir-session" {
		t.Fatalf("unexpected session id %q", result.SessionID)
	}
	if info, err := os.Stat(workDir); err != nil || !info.IsDir() {
		t.Fatalf("expected workdir to be created, info=%#v err=%v", info, err)
	}
}

func containsEnv(env []string, value string) bool {
	for _, item := range env {
		if item == value {
			return true
		}
	}
	return false
}
