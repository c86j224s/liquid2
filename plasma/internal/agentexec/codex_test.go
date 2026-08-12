package agentexec

import (
	"context"
	"fmt"
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

func TestCodexExecutorIgnoreUserConfigIsOptInForExec(t *testing.T) {
	args := runCodexArgsRecorder(t, AgentRequest{
		Prompt:           "test prompt",
		AgentExecutor:    "codex",
		IgnoreUserConfig: true,
	})
	if len(args) < 2 || args[0] != "exec" || args[1] != "--ignore-user-config" {
		t.Fatalf("ignore-user-config args = %#v, want exec subcommand option", args)
	}

	defaultArgs := runCodexArgsRecorder(t, AgentRequest{
		Prompt:        "test prompt",
		AgentExecutor: "codex",
	})
	for _, arg := range defaultArgs {
		if arg == "--ignore-user-config" {
			t.Fatalf("default args unexpectedly ignored user config: %#v", defaultArgs)
		}
	}
}

func TestCodexExecutorEphemeralSessionIsOptInForExec(t *testing.T) {
	args := runCodexArgsRecorder(t, AgentRequest{
		Prompt:           "test prompt",
		AgentExecutor:    "codex",
		EphemeralSession: true,
	})
	if !containsEnv(args, "--ephemeral") {
		t.Fatalf("ephemeral args missing --ephemeral: %#v", args)
	}

	defaultArgs := runCodexArgsRecorder(t, AgentRequest{
		Prompt:        "test prompt",
		AgentExecutor: "codex",
	})
	if containsEnv(defaultArgs, "--ephemeral") {
		t.Fatalf("default args unexpectedly used ephemeral session: %#v", defaultArgs)
	}
}

func TestCodexExecutorBindsModelAndEffortToResumeSubcommand(t *testing.T) {
	args := runCodexArgsRecorder(t, AgentRequest{
		Prompt:            "test prompt",
		AgentExecutor:     "codex",
		Model:             "gpt-5.6-luna",
		ReasoningEffort:   "high",
		PreviousSessionID: "existing-session",
		IgnoreUserConfig:  true,
	})
	wantPrefix := []string{
		"exec", "resume", "--ignore-user-config", "--model", "gpt-5.6-luna",
		"-c", `model_reasoning_effort="high"`, "--json",
	}
	if len(args) < len(wantPrefix) {
		t.Fatalf("resume args too short: %#v", args)
	}
	for i, want := range wantPrefix {
		if args[i] != want {
			t.Fatalf("resume args[%d] = %q, want %q; all args = %#v", i, args[i], want, args)
		}
	}
}

func runCodexArgsRecorder(t *testing.T, req AgentRequest) []string {
	t.Helper()
	dir := t.TempDir()
	argsPath := filepath.Join(dir, "args.txt")
	command := filepath.Join(dir, "fake-codex")
	script := fmt.Sprintf(`#!/bin/sh
out=""
want_out=0
: > %q
for arg in "$@"; do
  printf '%%s\n' "$arg" >> %q
  if [ "$want_out" = "1" ]; then
    out="$arg"
    want_out=0
  elif [ "$arg" = "--output-last-message" ]; then
    want_out=1
  fi
done
cat >/dev/null
printf 'session id: args-session\n'
printf 'done' > "$out"
`, argsPath, argsPath)
	if err := os.WriteFile(command, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := (CodexExecutor{
		Command: command,
		WorkDir: dir,
		Timeout: 10 * time.Second,
		Env:     []string{"PATH=/usr/bin:/bin"},
	}).Run(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Fields(string(raw))
}

func containsEnv(env []string, value string) bool {
	for _, item := range env {
		if item == value {
			return true
		}
	}
	return false
}
