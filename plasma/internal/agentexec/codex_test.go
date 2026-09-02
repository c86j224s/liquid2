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

func TestCodexExecutorPreservesResponseWhitespaceOnlyWhenRequested(t *testing.T) {
	dir := t.TempDir()
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
printf 'session id: whitespace-session\n'
printf '\n\n# exact\n\ntrailing  \n\n' > "$out"
`
	if err := os.WriteFile(command, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	executor := CodexExecutor{
		Command: command,
		WorkDir: dir,
		Timeout: 10 * time.Second,
		Env:     []string{"PATH=/usr/bin:/bin"},
	}
	preserved, err := executor.Run(context.Background(), AgentRequest{
		Prompt: "test prompt", AgentExecutor: "codex",
		PreserveResponseWhitespace: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if preserved.Text != "\n\n# exact\n\ntrailing  \n\n" {
		t.Fatalf("preserved response bytes = %q", preserved.Text)
	}
	trimmed, err := executor.Run(context.Background(), AgentRequest{
		Prompt: "test prompt", AgentExecutor: "codex",
	})
	if err != nil {
		t.Fatal(err)
	}
	if trimmed.Text != "# exact\n\ntrailing" {
		t.Fatalf("default response changed compatibility behavior: %q", trimmed.Text)
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

func TestCodexExecutorAddsRequestScopedConfigOverrides(t *testing.T) {
	args := runCodexArgsRecorder(t, AgentRequest{
		Prompt:        "test prompt",
		AgentExecutor: "codex",
		CodexConfig: []string{
			"features.apps=false",
			" ",
			"agents.enabled=false",
		},
	})
	want := []string{
		"-c", "features.apps=false",
		"-c", "agents.enabled=false",
	}
	for index := 0; index+len(want) <= len(args); index++ {
		matched := true
		for offset, value := range want {
			if args[index+offset] != value {
				matched = false
				break
			}
		}
		if matched {
			return
		}
	}
	t.Fatalf("request-scoped config args missing: %#v", args)
}

func TestCodexExecutorTransportsAndCleansOutputSchema(t *testing.T) {
	dir := t.TempDir()
	argsPath := filepath.Join(dir, "args.txt")
	schemaCopyPath := filepath.Join(dir, "schema.json")
	schemaPathRecord := filepath.Join(dir, "schema-path.txt")
	command := filepath.Join(dir, "fake-codex")
	script := fmt.Sprintf(`#!/bin/sh
out=""
schema=""
want_out=0
want_schema=0
: > %q
for arg in "$@"; do
  printf '%%s\n' "$arg" >> %q
  if [ "$want_out" = "1" ]; then
    out="$arg"
    want_out=0
  elif [ "$want_schema" = "1" ]; then
    schema="$arg"
    want_schema=0
  elif [ "$arg" = "--output-last-message" ]; then
    want_out=1
  elif [ "$arg" = "--output-schema" ]; then
    want_schema=1
  fi
done
cat "$schema" > %q
printf '%%s' "$schema" > %q
cat >/dev/null
printf 'session id: schema-session\n'
printf 'done' > "$out"
`, argsPath, argsPath, schemaCopyPath, schemaPathRecord)
	if err := os.WriteFile(command, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	schema := []byte(`{"type":"object","additionalProperties":false,"required":[],"properties":{}}`)
	_, err := (CodexExecutor{Command: command, WorkDir: dir, Timeout: 10 * time.Second, Env: []string{"PATH=/usr/bin:/bin"}}).Run(context.Background(), AgentRequest{Prompt: "test prompt", AgentExecutor: "codex", OutputJSONSchema: schema})
	if err != nil {
		t.Fatal(err)
	}
	copied, err := os.ReadFile(schemaCopyPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(copied) != string(schema) {
		t.Fatalf("transported schema = %q, want %q", copied, schema)
	}
	recordedPath, err := os.ReadFile(schemaPathRecord)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(string(recordedPath)); !os.IsNotExist(err) {
		t.Fatalf("schema temp file was not removed after success: %v", err)
	}
	args, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(args), "--output-schema\n") != 1 {
		t.Fatalf("output schema flag count changed: %q", args)
	}
}

func TestCodexExecutorCleansOutputSchemaAfterCommandFailure(t *testing.T) {
	dir := t.TempDir()
	schemaPathRecord := filepath.Join(dir, "schema-path.txt")
	command := filepath.Join(dir, "fake-codex")
	script := fmt.Sprintf(`#!/bin/sh
schema=""
want_schema=0
for arg in "$@"; do
  if [ "$want_schema" = "1" ]; then
    schema="$arg"
    want_schema=0
  elif [ "$arg" = "--output-schema" ]; then
    want_schema=1
  fi
done
printf '%%s' "$schema" > %q
exit 17
`, schemaPathRecord)
	if err := os.WriteFile(command, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := (CodexExecutor{Command: command, WorkDir: dir, Timeout: 10 * time.Second, Env: []string{"PATH=/usr/bin:/bin"}}).Run(context.Background(), AgentRequest{Prompt: "test prompt", AgentExecutor: "codex", OutputJSONSchema: []byte(`{"type":"string"}`)})
	if err == nil {
		t.Fatal("failing Codex command unexpectedly succeeded")
	}
	recordedPath, readErr := os.ReadFile(schemaPathRecord)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if _, statErr := os.Stat(string(recordedPath)); !os.IsNotExist(statErr) {
		t.Fatalf("schema temp file was not removed after failure: %v", statErr)
	}
}

func TestCodexExecutorRejectsMalformedOutputSchemaBeforeProcess(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "process-ran")
	command := filepath.Join(t.TempDir(), "fake-codex")
	script := fmt.Sprintf("#!/bin/sh\ntouch %q\n", marker)
	if err := os.WriteFile(command, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := (CodexExecutor{Command: command, WorkDir: t.TempDir(), Timeout: 10 * time.Second, Env: []string{"PATH=/usr/bin:/bin"}}).Run(context.Background(), AgentRequest{Prompt: "test prompt", AgentExecutor: "codex", OutputJSONSchema: []byte(`{"type":`)})
	if err == nil || !strings.Contains(err.Error(), "malformed JSON") {
		t.Fatalf("malformed schema result = %v", err)
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("Codex process ran for malformed schema: %v", statErr)
	}
}

func TestCodexExecutorClassicRequestOmitsOutputSchema(t *testing.T) {
	args := runCodexArgsRecorder(t, AgentRequest{Prompt: "test prompt", AgentExecutor: "codex"})
	if containsEnv(args, "--output-schema") {
		t.Fatalf("classic request unexpectedly transported output schema: %#v", args)
	}
}

func TestCodexExecutorRejectsOutputSchemaForCompactionBeforeProcess(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "process-ran")
	command := filepath.Join(t.TempDir(), "fake-codex")
	if err := os.WriteFile(command, []byte(fmt.Sprintf("#!/bin/sh\ntouch %q\n", marker)), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := (CodexExecutor{Command: command, WorkDir: t.TempDir(), Timeout: 10 * time.Second}).Run(context.Background(), AgentRequest{
		AgentExecutor: "codex", PreviousSessionID: "ses_existing", Compaction: true,
		OutputJSONSchema: []byte(`{"type":"string"}`),
	})
	if err == nil || !strings.Contains(err.Error(), "unavailable for compaction") {
		t.Fatalf("schema compaction result = %v", err)
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("Codex process ran for schema compaction: %v", statErr)
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
