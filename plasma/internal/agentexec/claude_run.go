package agentexec

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
)

// Run은 웹 및 에이전트 어댑터의 실행 진입점이다. 호출자는 취소, 실패, 외부 부작용 범위를 해당 패키지 계약에 맞게 보존해야 한다.
func (executor ClaudeExecutor) Run(ctx context.Context, req AgentRequest) (AgentResult, error) {
	return executor.run(ctx, req, false, nil)
}

// RunWithObserver는 Claude stream-json 이벤트에서 안전한 관찰 이벤트만 내보낸다.
func (executor ClaudeExecutor) RunWithObserver(ctx context.Context, req AgentRequest, observer AgentObserver) (AgentResult, error) {
	return executor.run(ctx, req, true, observer)
}

func (executor ClaudeExecutor) run(ctx context.Context, req AgentRequest, stream bool, observer AgentObserver) (AgentResult, error) {
	command := strings.TrimSpace(executor.Command)
	if command == "" {
		command = "claude"
	}
	command = resolveAgentCommand(command)
	workDir := strings.TrimSpace(executor.WorkDir)
	if workDir == "" {
		workDir = "."
	}
	if err := ensureAgentWorkDir(workDir); err != nil {
		return AgentResult{}, err
	}
	if timeout := executor.Timeout; timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	configPath, cleanup, err := executor.writeMCPConfig(req)
	if err != nil {
		return AgentResult{}, err
	}
	defer cleanup()

	prompt := req.Prompt
	resumed := strings.TrimSpace(req.PreviousSessionID) != ""
	usage := agentusage.New("claude", claudeUsageExecutorName(req.AgentExecutor), claudeModel(req.Model, executor.Model), req.ReasoningEffort, prompt).
		WithSession(req.PreviousSessionID, "", resumed, req.Compaction)
	args := executor.baseArgsForRequest(req)
	if stream {
		args = claudeStreamArgs(args)
	}
	if resumed {
		args = append(args, "--resume", strings.TrimSpace(req.PreviousSessionID))
	}
	if configPath != "" {
		args = append(args, "--mcp-config", configPath, "--strict-mcp-config")
	}
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = workDir
	cmd.Env = claudeEnvironment(executor.Env)
	cmd.Stdin = strings.NewReader(prompt)
	var stdout, stderr bytes.Buffer
	if !stream {
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	}

	var runErr error
	var scanErr error
	if !stream {
		runErr = cmd.Run()
	} else {
		observePhase(observer, AgentPhaseThinking)
		var stdoutMu, stderrMu, scanMu sync.Mutex
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
		outPipe, pipeErr := cmd.StdoutPipe()
		if pipeErr != nil {
			return AgentResult{Usage: usage}, pipeErr
		}
		errPipe, pipeErr := cmd.StderrPipe()
		if pipeErr != nil {
			return AgentResult{Usage: usage}, pipeErr
		}
		if runErr = cmd.Start(); runErr == nil {
			var wg sync.WaitGroup
			streamObserver := &claudeStreamObserver{observer: observer}
			wg.Add(2)
			go func() {
				defer wg.Done()
				recordScanErr(scanClaudeOutput(outPipe, &stdout, &stdoutMu, streamObserver))
			}()
			go func() {
				defer wg.Done()
				recordScanErr(scanClaudeOutput(errPipe, &stderr, &stderrMu, nil))
			}()
			wg.Wait()
			runErr = cmd.Wait()
		}
	}
	log := claudeLog(stdout.String(), stderr.String())
	parsed, parseErr := parseClaudeJSONOutput(stdout.Bytes(), usage)
	if ctx.Err() == context.Canceled {
		return AgentResult{Log: log, Usage: usage}, context.Canceled
	}
	if ctx.Err() == context.DeadlineExceeded {
		return AgentResult{Log: log, Usage: usage}, fmt.Errorf("agent timed out after %s", executor.Timeout)
	}
	if parseErr != nil && runErr != nil {
		return AgentResult{Log: log, Usage: usage}, fmt.Errorf("agent command failed: %w; parse Claude output: %v", runErr, parseErr)
	}
	if parseErr != nil {
		return AgentResult{Log: log, Usage: usage}, parseErr
	}
	if scanErr != nil {
		return AgentResult{Log: log, Usage: usage}, scanErr
	}
	parsed.Log = log
	if runErr != nil {
		return parsed, fmt.Errorf("agent command failed: %w", runErr)
	}
	if strings.TrimSpace(parsed.Text) == "" {
		return parsed, fmt.Errorf("agent returned an empty response")
	}
	return parsed, nil
}

func claudeStreamArgs(args []string) []string {
	streamArgs := append([]string(nil), args...)
	for i := 0; i+1 < len(streamArgs); i++ {
		if streamArgs[i] == "--output-format" {
			streamArgs[i+1] = "stream-json"
			return append(streamArgs, "--include-partial-messages", "--verbose")
		}
	}
	return append(streamArgs, "--output-format", "stream-json", "--include-partial-messages", "--verbose")
}

func scanClaudeOutput(reader io.Reader, buffer *bytes.Buffer, mu *sync.Mutex, observer *claudeStreamObserver) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		mu.Lock()
		buffer.WriteString(line)
		buffer.WriteByte('\n')
		mu.Unlock()
		if observer != nil {
			observer.observe(line)
		}
	}
	return scanner.Err()
}

type claudeStreamObserver struct {
	observer      AgentObserver
	preview       strings.Builder
	emittedPrefix string
}

func (observer *claudeStreamObserver) observe(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	var event claudeEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return
	}
	if event.Type != "stream_event" {
		return
	}
	streamEvent := event.Event
	switch streamEvent.Type {
	case "content_block_delta":
		switch streamEvent.Delta.Type {
		case "text_delta":
			observer.preview.WriteString(streamEvent.Delta.Text)
			observer.emitAnswerPreview(stableWhitespacePrefix(observer.preview.String()))
		case "thinking_delta":
			observePhase(observer.observer, AgentPhaseThinking)
		}
	case "content_block_stop":
		observer.emitAnswerPreview(observer.preview.String())
	case "content_block_start":
		if streamEvent.ContentBlock.Type == "tool_use" {
			observeTool(observer.observer, toolCategoryFromName(streamEvent.ContentBlock.Name))
		}
	}
}

func (observer *claudeStreamObserver) emitAnswerPreview(text string) {
	if observer == nil || observer.observer == nil || text == "" || text == observer.emittedPrefix {
		return
	}
	safeText := safeAnswerPreview(text)
	if safeText == "" {
		return
	}
	observer.emittedPrefix = text
	observer.observer(AgentObservation{Type: AgentObservationAnswer, Text: safeText})
}
