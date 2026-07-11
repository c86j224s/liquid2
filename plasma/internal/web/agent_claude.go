package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
)

type ClaudeExecutor struct {
	Command      string
	WorkDir      string
	Model        string
	Timeout      time.Duration
	Env          []string
	Permission   string
	MaxBudgetUSD string
	MCPServer    ClaudeMCPServer
}

type ClaudeMCPServer struct {
	Name    string
	Command string
	Args    []string
}

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

func (executor ClaudeExecutor) baseArgs(requestModel string, disableTools bool) []string {
	return executor.baseArgsWithToolMode(requestModel, disableTools, false)
}

func (executor ClaudeExecutor) baseArgsForRequest(req AgentRequest) []string {
	return executor.baseArgsWithToolMode(req.Model, req.DisableTools, req.ReplaceMCPTools)
}

func (executor ClaudeExecutor) baseArgsWithToolMode(requestModel string, disableTools bool, mcpOnly bool) []string {
	args := []string{
		"-p",
		"--model", claudeModel(requestModel, executor.Model),
		"--output-format", "json",
		"--permission-mode", firstNonEmpty(strings.TrimSpace(executor.Permission), "dontAsk"),
	}
	if !disableTools {
		args = append(args, "--allowedTools", executor.allowedTools(!mcpOnly))
	}
	args = append(args, "--disallowedTools")
	args = append(args, claudeDisallowedBuiltinTools()...)
	if disableTools || mcpOnly {
		args = append(args, claudeAllowedBuiltinTools()...)
	}
	args = append(args,
		"--disable-slash-commands",
	)
	if budget := strings.TrimSpace(executor.MaxBudgetUSD); budget != "" {
		args = append(args, "--max-budget-usd", budget)
	}
	return args
}

func (executor ClaudeExecutor) allowedTools(includeBuiltin bool) string {
	name := sanitizeMCPServerName(executor.MCPServer.Name)
	if name == "" {
		name = "plasma"
	}
	tools := []string{"mcp__" + name + "__*"}
	if includeBuiltin {
		// Local source roots are exposed through Plasma MCP args, not Claude --add-dir.
		// Built-in file tools stay scoped to the configured workdir until product policy opens more directories.
		tools = append(tools, claudeAllowedBuiltinTools()...)
	}
	return strings.Join(tools, ",")
}

func claudeAllowedBuiltinTools() []string {
	return []string{
		"Read",
		"Glob",
		"Grep",
		"LS",
		"WebFetch",
		"WebSearch",
	}
}

func (executor ClaudeExecutor) writeMCPConfig(req AgentRequest) (string, func(), error) {
	if req.DisableTools {
		return "", func() {}, nil
	}
	command := strings.TrimSpace(executor.MCPServer.Command)
	if command == "" {
		return "", func() {}, nil
	}
	name := sanitizeMCPServerName(executor.MCPServer.Name)
	args := codexMCPArgsForRequest(executor.MCPServer.Args, req)
	payload := map[string]any{
		"mcpServers": map[string]any{
			name: map[string]any{
				"command": command,
				"args":    args,
			},
		},
	}
	file, err := os.CreateTemp("", "plasma-claude-mcp-*.json")
	if err != nil {
		return "", func() {}, err
	}
	path := file.Name()
	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	err = encoder.Encode(payload)
	closeErr := file.Close()
	if err != nil {
		_ = os.Remove(path)
		return "", func() {}, err
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return "", func() {}, closeErr
	}
	return path, func() { _ = os.Remove(path) }, nil
}

func (executor ClaudeExecutor) claudeSessionFile(sourceSessionID string) (string, error) {
	sourceSessionID = strings.TrimSpace(sourceSessionID)
	if sourceSessionID == "" {
		return "", fmt.Errorf("source session id is required")
	}
	claudeHome := claudeHomeFromEnv(executor.Env)
	if claudeHome == "" {
		return "", fmt.Errorf("Claude home is required for Claude session fork")
	}
	return findClaudeSessionFile(claudeHome, sourceSessionID)
}

func claudeHomeFromEnv(explicit []string) string {
	home := ""
	for _, item := range explicit {
		if strings.HasPrefix(item, "CLAUDE_CONFIG_DIR=") {
			if claudeHome := strings.TrimSpace(strings.TrimPrefix(item, "CLAUDE_CONFIG_DIR=")); claudeHome != "" {
				return claudeHome
			}
		}
		if strings.HasPrefix(item, "HOME=") {
			home = strings.TrimSpace(strings.TrimPrefix(item, "HOME="))
		}
	}
	if claudeHome := strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_DIR")); claudeHome != "" {
		return claudeHome
	}
	if home == "" {
		home = strings.TrimSpace(os.Getenv("HOME"))
	}
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".claude")
}

func findClaudeSessionFile(claudeHome string, sessionID string) (string, error) {
	projectsRoot := filepath.Join(claudeHome, "projects")
	var matches []string
	err := filepath.WalkDir(projectsRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Name() != sessionID+".jsonl" {
			return nil
		}
		matches = append(matches, path)
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("Claude session file not found for session %q under %s", sessionID, projectsRoot)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("Claude session file is ambiguous for session %q", sessionID)
	}
	return matches[0], nil
}

func checkWritableClaudeSessionDir(dir string) error {
	file, err := os.CreateTemp(dir, ".plasma-claude-fork-ready-*")
	if err != nil {
		return fmt.Errorf("Claude session directory is not writable: %w", err)
	}
	path := file.Name()
	closeErr := file.Close()
	removeErr := os.Remove(path)
	if closeErr != nil {
		return fmt.Errorf("Claude session readiness temp file close failed: %w", closeErr)
	}
	if removeErr != nil {
		return fmt.Errorf("Claude session readiness temp file cleanup failed: %w", removeErr)
	}
	return nil
}

func claudeDisallowedBuiltinTools() []string {
	return []string{
		"Bash",
		"Edit",
		"MultiEdit",
		"Write",
		"NotebookEdit",
		"Task",
		"TodoWrite",
	}
}

type claudeEvent struct {
	Type         string                      `json:"type"`
	Subtype      string                      `json:"subtype"`
	SessionID    string                      `json:"session_id"`
	Result       *string                     `json:"result"`
	Message      claudeMessage               `json:"message"`
	Usage        claudeUsage                 `json:"usage"`
	ModelUsage   map[string]claudeModelUsage `json:"modelUsage"`
	TotalCostUSD float64                     `json:"total_cost_usd"`
	Errors       []string                    `json:"errors"`
}

type claudeMessage struct {
	Content []claudeContent `json:"content"`
	Usage   claudeUsage     `json:"usage"`
}

type claudeContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type claudeUsage struct {
	InputTokens              int `json:"input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	OutputTokens             int `json:"output_tokens"`
}

type claudeModelUsage struct {
	InputTokens              int     `json:"inputTokens"`
	OutputTokens             int     `json:"outputTokens"`
	CacheReadInputTokens     int     `json:"cacheReadInputTokens"`
	CacheCreationInputTokens int     `json:"cacheCreationInputTokens"`
	CostUSD                  float64 `json:"costUSD"`
}

func parseClaudeJSONOutput(raw []byte, usage agentusage.AgentUsage) (AgentResult, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return AgentResult{Usage: usage.WithUnavailable("claude emitted no JSON output")}, fmt.Errorf("agent emitted no JSON output")
	}
	events, err := parseClaudeEvents(raw)
	if err != nil {
		return AgentResult{Usage: usage.WithUnavailable("claude JSON output could not be parsed")}, err
	}
	result := AgentResult{Usage: usage}
	var lastText string
	var providerUsage agentusage.ProviderUsage
	var sawProviderUsage bool
	for _, event := range events {
		if strings.TrimSpace(event.SessionID) != "" {
			result.SessionID = strings.TrimSpace(event.SessionID)
		}
		if event.Result != nil && strings.TrimSpace(*event.Result) != "" {
			lastText = strings.TrimSpace(*event.Result)
		}
		for _, content := range event.Message.Content {
			if content.Type == "text" && strings.TrimSpace(content.Text) != "" {
				lastText = strings.TrimSpace(content.Text)
			}
		}
		if usage, ok := claudeProviderUsage(event); ok {
			providerUsage = usage
			sawProviderUsage = true
		}
	}
	result.Text = lastText
	result.Resumed = strings.TrimSpace(usage.Session.PreviousAgentSessionID) != ""
	if sawProviderUsage {
		result.Usage = result.Usage.WithProviderUsage(providerUsage, "claude_json")
	} else {
		result.Usage = result.Usage.WithUnavailable("claude JSON did not include provider usage")
	}
	result.Usage = result.Usage.WithSession(usage.Session.PreviousAgentSessionID, result.SessionID, result.Resumed, usage.Session.CompactionAttempted)
	return result, nil
}

func parseClaudeEvents(raw []byte) ([]claudeEvent, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, fmt.Errorf("agent emitted no JSON output")
	}
	if raw[0] == '[' {
		var events []claudeEvent
		if err := json.Unmarshal(raw, &events); err != nil {
			return nil, err
		}
		return events, nil
	}
	var event claudeEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		return nil, err
	}
	return []claudeEvent{event}, nil
}

func claudeProviderUsage(event claudeEvent) (agentusage.ProviderUsage, bool) {
	for _, modelUsage := range event.ModelUsage {
		return agentusage.ProviderUsage{
			InputTokens:         modelUsage.InputTokens + modelUsage.CacheReadInputTokens + modelUsage.CacheCreationInputTokens,
			CachedInputTokens:   modelUsage.CacheReadInputTokens,
			UncachedInputTokens: modelUsage.InputTokens + modelUsage.CacheCreationInputTokens,
			OutputTokens:        modelUsage.OutputTokens,
		}, true
	}
	usage := event.Usage
	totalInput := usage.InputTokens + usage.CacheReadInputTokens + usage.CacheCreationInputTokens
	if totalInput == 0 && usage.OutputTokens == 0 {
		return agentusage.ProviderUsage{}, false
	}
	return agentusage.ProviderUsage{
		InputTokens:         totalInput,
		CachedInputTokens:   usage.CacheReadInputTokens,
		UncachedInputTokens: usage.InputTokens + usage.CacheCreationInputTokens,
		OutputTokens:        usage.OutputTokens,
	}, true
}

func claudeModel(requestModel string, executorModel string) string {
	if model := strings.TrimSpace(requestModel); model != "" {
		return model
	}
	if model := strings.TrimSpace(executorModel); model != "" {
		return model
	}
	return "haiku"
}

func claudeUsageExecutorName(executorName string) string {
	executorName = strings.TrimSpace(executorName)
	if executorName == "" {
		return "claude"
	}
	return executorName
}

func claudeEnvironment(explicit []string) []string {
	if len(explicit) > 0 {
		return append([]string(nil), explicit...)
	}
	allowedKeys := []string{
		"HOME",
		"PATH",
		"SHELL",
		"TERM",
		"USER",
		"LOGNAME",
		"TMPDIR",
		"LANG",
		"LC_ALL",
		"LC_CTYPE",
		"XDG_CONFIG_HOME",
		"XDG_DATA_HOME",
		"XDG_CACHE_HOME",
		"PLASMA_RUNTIME_MODE",
		"CLAUDE_CONFIG_DIR",
		"ANTHROPIC_API_KEY",
	}
	env := make([]string, 0, len(allowedKeys)+1)
	seenPath := false
	for _, key := range allowedKeys {
		value, ok := os.LookupEnv(key)
		if !ok {
			continue
		}
		if key == "PATH" {
			value = agentPATH(value)
			seenPath = true
		}
		env = append(env, key+"="+value)
	}
	if !seenPath {
		env = append(env, "PATH="+agentPATH(""))
	}
	env = append(env, "PLASMA_AGENT=1")
	return env
}

func claudeLog(stdout string, stderr string) string {
	if strings.TrimSpace(stderr) == "" {
		return stdout
	}
	if strings.TrimSpace(stdout) == "" {
		return stderr
	}
	return stdout + "\n" + stderr
}

var claudeSessionIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
