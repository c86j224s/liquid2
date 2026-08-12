package agentexec

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/c86j224s/liquid2/plasma/internal/agentmodels"
	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
)

// Run은 Codex CLI를 실행하고 stdout/log에서 session과 usage를 추출한다.
//
// 이 메서드는 prompt를 파일이나 장부에 저장하지 않으며, model/effort는
// agentmodels.ResolveForSession 계약을 따른다.
func (executor CodexExecutor) Run(ctx context.Context, req AgentRequest) (AgentResult, error) {
	return executor.run(ctx, req, nil)
}

// RunWithObserver는 Codex CLI JSONL을 실행 중 읽어 안전한 관찰 이벤트만 내보낸다.
func (executor CodexExecutor) RunWithObserver(ctx context.Context, req AgentRequest, observer AgentObserver) (AgentResult, error) {
	return executor.run(ctx, req, observer)
}

func (executor CodexExecutor) run(ctx context.Context, req AgentRequest, observer AgentObserver) (AgentResult, error) {
	model, effort, err := agentmodels.ResolveForSession(req.Model, req.ReasoningEffort, req.PreviousSessionID)
	if err != nil {
		return AgentResult{}, fmt.Errorf("invalid Codex model settings: %w", err)
	}
	req.Model = model
	req.ReasoningEffort = effort
	command := strings.TrimSpace(executor.Command)
	if command == "" {
		command = "codex"
	}
	command = resolveAgentCommand(command)
	workDir := strings.TrimSpace(executor.WorkDir)
	if workDir == "" {
		workDir = "."
	}
	if err := ensureAgentWorkDir(workDir); err != nil {
		return AgentResult{}, err
	}
	tmp, err := os.CreateTemp("", "plasma-codex-last-*.txt")
	if err != nil {
		return AgentResult{}, err
	}
	lastPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(lastPath)

	timeout := executor.Timeout
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	if req.Compaction {
		if strings.TrimSpace(req.PreviousSessionID) == "" {
			return AgentResult{}, fmt.Errorf("Codex context compaction requires an existing session")
		}
		return executor.runCodexCompaction(ctx, command, workDir, req)
	}

	resumed := strings.TrimSpace(req.PreviousSessionID) != ""
	args := codexCommandArgs(executor.MCPServer, req, workDir, lastPath)

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = workDir
	cmd.Env = codexEnvironment(executor.Env)
	prompt := req.Prompt
	usage := agentusage.New("codex", codexUsageExecutorName(req.AgentExecutor), req.Model, req.ReasoningEffort, prompt).
		WithSession(req.PreviousSessionID, "", resumed, req.Compaction)
	cmd.Stdin = strings.NewReader(prompt)
	var combined bytes.Buffer
	if observer == nil {
		cmd.Stdout = &combined
		cmd.Stderr = &combined
	}

	var scanErr error
	if observer == nil {
		err = cmd.Run()
	} else {
		observePhase(observer, AgentPhaseThinking)
		var mu sync.Mutex
		var scanMu sync.Mutex
		recordScanErr := func(err error) {
			if err == nil {
				return
			}
			scanMu.Lock()
			defer scanMu.Unlock()
			if scanErr == nil {
				scanErr = err
			}
		}
		stdout, pipeErr := cmd.StdoutPipe()
		if pipeErr != nil {
			return AgentResult{Usage: usage}, pipeErr
		}
		stderr, pipeErr := cmd.StderrPipe()
		if pipeErr != nil {
			return AgentResult{Usage: usage}, pipeErr
		}
		if err = cmd.Start(); err == nil {
			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()
				recordScanErr(scanCodexOutput(stdout, &combined, &mu, observer))
			}()
			go func() {
				defer wg.Done()
				recordScanErr(scanCodexOutput(stderr, &combined, &mu, nil))
			}()
			wg.Wait()
			err = cmd.Wait()
		}
	}
	log := combined.String()
	sessionID := codexSessionID(log)
	usage = codexUsageFromLog(usage, log, req.PreviousSessionID, sessionID, resumed, req.Compaction)
	telemetrySessionID := codexTelemetrySessionID(sessionID, req.PreviousSessionID)
	if contextWindow, ok := executor.codexContextWindowMetrics(telemetrySessionID); ok {
		usage = usage.WithContextWindow(contextWindow)
	}
	if ctx.Err() == context.Canceled {
		return AgentResult{Log: log, Resumed: resumed, SessionID: sessionID, Usage: usage}, context.Canceled
	}
	if ctx.Err() == context.DeadlineExceeded {
		if timeout <= 0 {
			return AgentResult{Log: log, Resumed: resumed, SessionID: sessionID, Usage: usage}, context.DeadlineExceeded
		}
		return AgentResult{Log: log, Resumed: resumed, SessionID: sessionID, Usage: usage}, fmt.Errorf("agent timed out after %s", timeout)
	}
	if err != nil {
		return AgentResult{Log: log, Resumed: resumed, SessionID: sessionID, Usage: usage}, fmt.Errorf("agent command failed: %w", err)
	}
	if scanErr != nil {
		return AgentResult{Log: log, Resumed: resumed, SessionID: sessionID, Usage: usage}, scanErr
	}
	content, err := os.ReadFile(lastPath)
	if err != nil {
		return AgentResult{Log: log, Resumed: resumed, SessionID: sessionID, Usage: usage}, err
	}
	text := strings.TrimSpace(string(content))
	if text == "" {
		return AgentResult{Log: log, Resumed: resumed, SessionID: sessionID, Usage: usage}, fmt.Errorf("agent returned an empty response")
	}
	return AgentResult{
		Text:      text,
		SessionID: sessionID,
		Resumed:   resumed,
		Log:       log,
		Usage:     usage,
	}, nil
}

func scanCodexOutput(reader io.Reader, combined *bytes.Buffer, mu *sync.Mutex, observer AgentObserver) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		mu.Lock()
		combined.WriteString(line)
		combined.WriteByte('\n')
		mu.Unlock()
		if observer != nil {
			observeCodexJSONLine(line, observer)
		}
	}
	return scanner.Err()
}

type codexJSONLine struct {
	Type string        `json:"type"`
	Item codexJSONItem `json:"item"`
}

type codexJSONItem struct {
	Type     string             `json:"type"`
	Tool     string             `json:"tool"`
	Text     string             `json:"text"`
	Content  []codexJSONContent `json:"content"`
	Name     string             `json:"name"`
	ToolName string             `json:"tool_name"`
	CallType string             `json:"call_type"`
}

type codexJSONContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func observeCodexJSONLine(line string, observer AgentObserver) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	var event codexJSONLine
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return
	}
	switch event.Type {
	case "thread.started", "turn.started":
		observePhase(observer, AgentPhaseThinking)
	case "item.started":
		observeCodexItemLifecycle(event.Item, observer)
	case "item.completed":
		if event.Item.Type == "agent_message" {
			observeAnswer(observer, codexAgentMessageText(event.Item))
			return
		}
		observeCodexItemLifecycle(event.Item, observer)
	}
}

func observeCodexItemLifecycle(item codexJSONItem, observer AgentObserver) {
	if item.Type == "" || item.Type == "agent_message" || item.Type == "reasoning" {
		if item.Type == "reasoning" {
			observePhase(observer, AgentPhaseThinking)
		}
		return
	}
	name := firstNonEmpty(item.Tool, item.Name, item.ToolName, item.CallType, item.Type)
	observeTool(observer, toolCategoryFromName(name))
}

func codexAgentMessageText(item codexJSONItem) string {
	if strings.TrimSpace(item.Text) != "" {
		return item.Text
	}
	var builder strings.Builder
	for _, content := range item.Content {
		if content.Type == "text" && strings.TrimSpace(content.Text) != "" {
			if builder.Len() > 0 {
				builder.WriteString("\n")
			}
			builder.WriteString(content.Text)
		}
	}
	return builder.String()
}
