package agentexec

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCodexExecutorUsesAppServerForCompaction(t *testing.T) {
	dir := t.TempDir()
	requestsPath := filepath.Join(dir, "requests.jsonl")
	command := writeFakeCodexAppServer(t, dir, requestsPath, true)
	result, err := (CodexExecutor{
		Command: command,
		WorkDir: dir,
		Timeout: 10 * time.Second,
		Env:     []string{"PATH=/usr/bin:/bin"},
	}).Run(context.Background(), AgentRequest{
		PreviousSessionID: "agent-session-1",
		AgentExecutor:     "codex",
		Compaction:        true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.SessionID != "agent-session-1" || !result.Resumed || !result.Usage.Session.CompactionAttempted {
		t.Fatalf("unexpected compaction result: %#v", result)
	}
	raw, err := os.ReadFile(requestsPath)
	if err != nil {
		t.Fatal(err)
	}
	var methods []string
	for _, line := range splitNonEmptyLines(string(raw)) {
		var message struct {
			Method string `json:"method"`
		}
		if err := json.Unmarshal([]byte(line), &message); err != nil {
			t.Fatal(err)
		}
		methods = append(methods, message.Method)
	}
	want := []string{"initialize", "initialized", "thread/resume", "thread/compact/start"}
	if len(methods) != len(want) {
		t.Fatalf("methods = %#v, want %#v", methods, want)
	}
	for index := range want {
		if methods[index] != want[index] {
			t.Fatalf("methods = %#v, want %#v", methods, want)
		}
	}
}

func TestCodexExecutorRejectsCompactionWithoutCompletedItem(t *testing.T) {
	dir := t.TempDir()
	command := writeFakeCodexAppServer(t, dir, filepath.Join(dir, "requests.jsonl"), false)
	_, err := (CodexExecutor{
		Command: command,
		WorkDir: dir,
		Timeout: 10 * time.Second,
		Env:     []string{"PATH=/usr/bin:/bin"},
	}).Run(context.Background(), AgentRequest{
		PreviousSessionID: "agent-session-1",
		AgentExecutor:     "codex",
		Compaction:        true,
	})
	if err == nil {
		t.Fatal("expected incomplete compaction to fail")
	}
}

func writeFakeCodexAppServer(t *testing.T, dir string, requestsPath string, complete bool) string {
	t.Helper()
	completion := ""
	if complete {
		completion = `
printf '%s\n' '{"method":"item/completed","params":{"threadId":"agent-session-1","turnId":"turn-1","item":{"id":"item-1","type":"contextCompaction"},"completedAtMs":1}}'
printf '%s\n' '{"method":"turn/completed","params":{"threadId":"agent-session-1","turn":{"id":"turn-1","items":[],"status":"completed"}}}'
`
	}
	command := filepath.Join(dir, "fake-codex")
	script := `#!/bin/sh
if [ "$1" != "app-server" ]; then
  exit 2
fi
: > "` + requestsPath + `"
IFS= read -r line
printf '%s\n' "$line" >> "` + requestsPath + `"
printf '%s\n' '{"id":0,"result":{}}'
IFS= read -r line
printf '%s\n' "$line" >> "` + requestsPath + `"
IFS= read -r line
printf '%s\n' "$line" >> "` + requestsPath + `"
printf '%s\n' '{"id":1,"result":{"thread":{"id":"agent-session-1"}}}'
IFS= read -r line
printf '%s\n' "$line" >> "` + requestsPath + `"
printf '%s\n' '{"id":2,"result":{}}'
` + completion
	if err := os.WriteFile(command, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return command
}

func splitNonEmptyLines(value string) []string {
	lines := []string{}
	start := 0
	for index := 0; index <= len(value); index++ {
		if index < len(value) && value[index] != '\n' {
			continue
		}
		if line := value[start:index]; line != "" {
			lines = append(lines, line)
		}
		start = index + 1
	}
	return lines
}
