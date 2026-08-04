package agentexec

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentmodels"
	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
)

// Run은 Codex CLI를 실행하고 stdout/log에서 session과 usage를 추출한다.
//
// 이 메서드는 prompt를 파일이나 장부에 저장하지 않으며, model/effort는
// agentmodels.ResolveForSession 계약을 따른다.
func (executor CodexExecutor) Run(ctx context.Context, req AgentRequest) (AgentResult, error) {
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

	args := []string{"exec"}
	mcpArgs := codexMCPConfigArgs(executor.MCPServer, req)
	if model := strings.TrimSpace(req.Model); model != "" {
		args = append(args, "--model", model)
	}
	if effort := strings.TrimSpace(req.ReasoningEffort); effort != "" {
		args = append(args, "-c", "model_reasoning_effort="+strconv.Quote(effort))
	}
	args = append(args, "--json")
	resumed := strings.TrimSpace(req.PreviousSessionID) != ""
	if resumed {
		args = append(args, "resume")
		args = append(args, mcpArgs...)
		args = append(args,
			"-c", `sandbox_mode="read-only"`,
			"--skip-git-repo-check",
			"--ignore-rules",
			"--output-last-message", lastPath,
			strings.TrimSpace(req.PreviousSessionID),
			"-",
		)
	} else {
		args = append(args, mcpArgs...)
		args = append(args,
			"--sandbox", "read-only",
			"--skip-git-repo-check",
			"--ignore-rules",
			"-C", workDir,
			"--output-last-message", lastPath,
			"-",
		)
	}

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = workDir
	cmd.Env = codexEnvironment(executor.Env)
	prompt := req.Prompt
	if req.Compaction && resumed {
		prompt = "/compact"
	}
	usage := agentusage.New("codex", codexUsageExecutorName(req.AgentExecutor), req.Model, req.ReasoningEffort, prompt).
		WithSession(req.PreviousSessionID, "", resumed, req.Compaction)
	cmd.Stdin = strings.NewReader(prompt)
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined

	err = cmd.Run()
	log := combined.String()
	sessionID := codexSessionID(log)
	usage = codexUsageFromLog(usage, log, req.PreviousSessionID, sessionID, resumed, req.Compaction)
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
