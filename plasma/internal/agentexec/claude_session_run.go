package agentexec

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
)

// CheckForkSession은 Claude session 파일과 대상 디렉터리가 fork 가능한지 점검한다.
func (executor ClaudeExecutor) CheckForkSession(_ context.Context, sourceSessionID string) error {
	if !claudeSessionIDPattern.MatchString(strings.TrimSpace(sourceSessionID)) {
		return fmt.Errorf("Claude session id must be a UUID")
	}
	sessionFile, err := executor.claudeSessionFile(sourceSessionID)
	if err != nil {
		return err
	}
	if _, err := os.Stat(sessionFile); err != nil {
		return err
	}
	if err := checkWritableClaudeSessionDir(filepath.Dir(sessionFile)); err != nil {
		return err
	}
	return nil
}

// ForkSession은 Claude CLI의 --fork-session으로 기존 session을 복제한다.
func (executor ClaudeExecutor) ForkSession(ctx context.Context, sourceSessionID string) (AgentSessionForkResult, error) {
	sourceSessionID = strings.TrimSpace(sourceSessionID)
	if err := executor.CheckForkSession(ctx, sourceSessionID); err != nil {
		return AgentSessionForkResult{}, err
	}
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
		return AgentSessionForkResult{}, err
	}
	if timeout := executor.Timeout; timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	args := executor.baseArgs("", false)
	args = append(args, "--resume", sourceSessionID, "--fork-session")
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = workDir
	cmd.Env = claudeEnvironment(executor.Env)
	cmd.Stdin = strings.NewReader("Reply exactly: FORK_READY\n")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	log := claudeLog(stdout.String(), stderr.String())
	usage := agentusage.New("claude", "claude", claudeModel("", executor.Model), "", "Reply exactly: FORK_READY\n")
	result, parseErr := parseClaudeJSONOutput(stdout.Bytes(), usage)
	if ctx.Err() != nil {
		return AgentSessionForkResult{}, ctx.Err()
	}
	if parseErr != nil {
		if err != nil {
			return AgentSessionForkResult{}, fmt.Errorf("Claude fork failed: %w; parse Claude output: %v", err, parseErr)
		}
		return AgentSessionForkResult{}, parseErr
	}
	cloneID := strings.TrimSpace(result.SessionID)
	if cloneID == "" || cloneID == sourceSessionID {
		if err != nil {
			return AgentSessionForkResult{}, fmt.Errorf("Claude fork failed: %w: %s", err, headTailExcerpt(log, 2000))
		}
		return AgentSessionForkResult{}, fmt.Errorf("Claude fork did not return a new session id")
	}
	return AgentSessionForkResult{SessionID: cloneID, SourceSessionID: sourceSessionID}, nil
}
