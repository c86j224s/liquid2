package agentexec

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
)

// Run은 웹 및 에이전트 어댑터의 실행 진입점이다. 호출자는 취소, 실패, 외부 부작용 범위를 해당 패키지 계약에 맞게 보존해야 한다.
func (executor ClaudeExecutor) Run(ctx context.Context, req AgentRequest) (AgentResult, error) {
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
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
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
	parsed.Log = log
	if runErr != nil {
		return parsed, fmt.Errorf("agent command failed: %w", runErr)
	}
	if strings.TrimSpace(parsed.Text) == "" {
		return parsed, fmt.Errorf("agent returned an empty response")
	}
	return parsed, nil
}
