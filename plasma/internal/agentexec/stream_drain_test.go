package agentexec

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestCodexStreamingDrainsPipesBeforeWait(t *testing.T) {
	const lines = 1500
	command := writeAgentDrainScript(t, fmt.Sprintf(`#!/bin/sh
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
i=0
while [ "$i" -lt %d ]; do
  i=$((i + 1))
  printf '{"type":"item.completed","item":{"type":"agent_message","text":"codex-%%04d"}}\n' "$i"
  printf 'stderr-%%04d\n' "$i" >&2
done
printf 'session id: codex-drain-session\n'
printf 'codex final' > "$out"
`, lines))

	var mu sync.Mutex
	answerEvents := 0
	result, err := (CodexExecutor{
		Command: command,
		WorkDir: filepath.Dir(command),
		Timeout: 30 * time.Second,
		Env:     []string{"PATH=/usr/bin:/bin"},
	}).RunWithObserver(context.Background(), AgentRequest{
		Prompt:        "drain test",
		AgentExecutor: "codex",
	}, func(event AgentObservation) {
		if event.Type != AgentObservationAnswer {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		answerEvents++
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "codex final" || result.SessionID != "codex-drain-session" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if answerEvents != lines {
		t.Fatalf("expected %d answer observations, got %d", lines, answerEvents)
	}
}

func TestClaudeStreamingDrainsPipesBeforeWait(t *testing.T) {
	const lines = 1500
	command := writeAgentDrainScript(t, fmt.Sprintf(`#!/bin/sh
cat >/dev/null
i=0
while [ "$i" -lt %d ]; do
  i=$((i + 1))
  printf '{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"x "}}}\n'
  printf 'stderr-%%04d\n' "$i" >&2
done
printf '{"type":"result","session_id":"claude-drain-session","result":"claude final","usage":{"input_tokens":1,"output_tokens":2}}\n'
`, lines))

	var mu sync.Mutex
	answerEvents := 0
	result, err := (ClaudeExecutor{
		Command: command,
		WorkDir: filepath.Dir(command),
		Timeout: 30 * time.Second,
		Env:     []string{"PATH=/usr/bin:/bin"},
	}).RunWithObserver(context.Background(), AgentRequest{
		Prompt:        "drain test",
		AgentExecutor: "claude",
	}, func(event AgentObservation) {
		if event.Type != AgentObservationAnswer {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		answerEvents++
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "claude final" || result.SessionID != "claude-drain-session" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if answerEvents != lines {
		t.Fatalf("expected %d answer observations, got %d", lines, answerEvents)
	}
}

func writeAgentDrainScript(t *testing.T, script string) string {
	t.Helper()
	dir := t.TempDir()
	command := filepath.Join(dir, "fake-agent")
	if err := os.WriteFile(command, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return command
}

func assertObservationOrder(t *testing.T, events []AgentObservation, want []AgentObservationType) {
	t.Helper()
	if len(events) != len(want) {
		t.Fatalf("events = %#v, want %d events", events, len(want))
	}
	for i, event := range events {
		if event.Type != want[i] {
			t.Fatalf("event[%d] = %#v, want type %q", i, event, want[i])
		}
	}
}
