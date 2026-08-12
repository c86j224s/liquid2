package agentexec

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
)

const codexAppServerErrorBytes = 16 << 10

// runCodexCompaction invokes Codex's explicit app-server compaction operation.
// A JSON-RPC acknowledgement alone is insufficient: the adapter waits for both
// the contextCompaction item and its enclosing turn to complete.
func (executor CodexExecutor) runCodexCompaction(ctx context.Context, command string, workDir string, req AgentRequest) (AgentResult, error) {
	sessionID := strings.TrimSpace(req.PreviousSessionID)
	usage := agentusage.New("codex", codexUsageExecutorName(req.AgentExecutor), req.Model, req.ReasoningEffort, "").
		WithUnavailable("codex app-server compaction does not emit provider usage").
		WithSession(sessionID, sessionID, true, true)

	cmd := exec.CommandContext(ctx, command, "app-server", "--listen", "stdio://")
	cmd.Dir = workDir
	cmd.Env = codexEnvironment(executor.Env)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return AgentResult{Resumed: true, SessionID: sessionID, Usage: usage}, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return AgentResult{Resumed: true, SessionID: sessionID, Usage: usage}, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return AgentResult{Resumed: true, SessionID: sessionID, Usage: usage}, err
	}
	if err := cmd.Start(); err != nil {
		return AgentResult{Resumed: true, SessionID: sessionID, Usage: usage}, err
	}

	var errorLog boundedLogBuffer
	errorLog.limit = codexAppServerErrorBytes
	stderrDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(&errorLog, stderr)
		close(stderrDone)
	}()
	messages := make(chan codexRPCMessage, 16)
	readErrors := make(chan error, 1)
	go decodeCodexRPC(stdout, messages, readErrors)

	finished := false
	defer func() {
		if finished {
			return
		}
		_ = stdin.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
		<-stderrDone
	}()
	encoder := json.NewEncoder(stdin)
	if err := sendCodexRPC(encoder, "initialize", 0, map[string]any{
		"clientInfo": map[string]string{"name": "plasma", "title": "Plasma", "version": "1"},
	}); err != nil {
		return AgentResult{Resumed: true, SessionID: sessionID, Usage: usage}, err
	}
	if err := waitCodexRPCResponse(ctx, messages, readErrors, 0); err != nil {
		return AgentResult{Resumed: true, SessionID: sessionID, Log: errorLog.String(), Usage: usage}, err
	}
	if err := encoder.Encode(map[string]any{"method": "initialized", "params": map[string]any{}}); err != nil {
		return AgentResult{Resumed: true, SessionID: sessionID, Usage: usage}, err
	}
	if err := sendCodexRPC(encoder, "thread/resume", 1, map[string]string{"threadId": sessionID}); err != nil {
		return AgentResult{Resumed: true, SessionID: sessionID, Usage: usage}, err
	}
	if err := waitCodexRPCResponse(ctx, messages, readErrors, 1); err != nil {
		return AgentResult{Resumed: true, SessionID: sessionID, Log: errorLog.String(), Usage: usage}, err
	}
	if err := sendCodexRPC(encoder, "thread/compact/start", 2, map[string]string{"threadId": sessionID}); err != nil {
		return AgentResult{Resumed: true, SessionID: sessionID, Usage: usage}, err
	}
	if err := waitCodexCompaction(ctx, messages, readErrors, sessionID); err != nil {
		return AgentResult{Resumed: true, SessionID: sessionID, Log: errorLog.String(), Usage: usage}, err
	}

	_ = stdin.Close()
	if err := cmd.Wait(); err != nil && ctx.Err() == nil {
		return AgentResult{Resumed: true, SessionID: sessionID, Log: errorLog.String(), Usage: usage}, fmt.Errorf("Codex app-server failed after compaction: %w", err)
	}
	<-stderrDone
	finished = true
	if contextWindow, ok := executor.codexContextWindowMetrics(sessionID); ok {
		usage = usage.WithContextWindow(contextWindow)
	}
	return AgentResult{
		Text:      "Codex session context compacted.",
		SessionID: sessionID,
		Resumed:   true,
		Log:       errorLog.String(),
		Usage:     usage,
	}, nil
}

func sendCodexRPC(encoder *json.Encoder, method string, id int, params any) error {
	return encoder.Encode(map[string]any{"method": method, "id": id, "params": params})
}
