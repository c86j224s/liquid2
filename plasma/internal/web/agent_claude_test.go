package web

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
)

func TestParseClaudeJSONOutputUsesAssistantTextWhenResultIsBudgetError(t *testing.T) {
	raw := []byte(`[
		{"type":"system","subtype":"init","session_id":"ses-ignored"},
		{"type":"assistant","session_id":"11111111-1111-4111-8111-111111111111","message":{"content":[{"type":"text","text":"final answer"}]}},
		{"type":"result","subtype":"error_max_budget_usd","session_id":"11111111-1111-4111-8111-111111111111","result":null,"modelUsage":{"claude-haiku-4-5-20251001":{"inputTokens":10,"cacheReadInputTokens":20,"cacheCreationInputTokens":30,"outputTokens":4,"costUSD":0.01}}}
	]`)
	usage := agentusage.New("claude", "claude", "haiku", "", "prompt").WithSession("previous", "", true, false)

	result, err := parseClaudeJSONOutput(raw, usage)
	if err != nil {
		t.Fatalf("parseClaudeJSONOutput returned error: %v", err)
	}
	if result.Text != "final answer" {
		t.Fatalf("expected assistant text to be preserved, got %q", result.Text)
	}
	if result.SessionID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("expected session id from Claude output, got %q", result.SessionID)
	}
	if result.Usage.ProviderUsage == nil {
		t.Fatal("expected provider usage")
	}
	if result.Usage.ProviderUsage.InputTokens != 60 || result.Usage.ProviderUsage.OutputTokens != 4 {
		t.Fatalf("unexpected provider usage: %#v", result.Usage.ProviderUsage)
	}
}

func TestParseClaudeJSONOutputAcceptsSingleResultObject(t *testing.T) {
	raw := []byte(`{"type":"result","session_id":"22222222-2222-4222-8222-222222222222","result":"single result answer","usage":{"input_tokens":7,"cache_read_input_tokens":11,"cache_creation_input_tokens":13,"output_tokens":5}}`)
	usage := agentusage.New("claude", "claude", "haiku", "", "prompt")

	result, err := parseClaudeJSONOutput(raw, usage)
	if err != nil {
		t.Fatalf("parseClaudeJSONOutput returned error: %v", err)
	}
	if result.Text != "single result answer" {
		t.Fatalf("expected single result text, got %q", result.Text)
	}
	if result.SessionID != "22222222-2222-4222-8222-222222222222" {
		t.Fatalf("expected session id from single result, got %q", result.SessionID)
	}
	if result.Usage.ProviderUsage == nil {
		t.Fatal("expected provider usage")
	}
	if result.Usage.ProviderUsage.InputTokens != 31 || result.Usage.ProviderUsage.CachedInputTokens != 11 || result.Usage.ProviderUsage.OutputTokens != 5 {
		t.Fatalf("unexpected provider usage: %#v", result.Usage.ProviderUsage)
	}
}

func TestClaudeExecutorRunReturnsErrorOnNonZeroExitEvenWithText(t *testing.T) {
	dir := t.TempDir()
	command := filepath.Join(dir, "fake-claude")
	script := `#!/bin/sh
cat >/dev/null
printf '%s\n' '{"type":"result","session_id":"33333333-3333-4333-8333-333333333333","result":"partial answer","usage":{"input_tokens":7,"output_tokens":5}}'
exit 1
`
	if err := os.WriteFile(command, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	result, err := (ClaudeExecutor{Command: command, WorkDir: dir}).Run(context.Background(), AgentRequest{
		Prompt:        "test prompt",
		AgentExecutor: "claude",
	})
	if err == nil || !strings.Contains(err.Error(), "agent command failed") {
		t.Fatalf("expected nonzero exit to be returned as an error, got result=%#v err=%v", result, err)
	}
	if result.Text != "partial answer" {
		t.Fatalf("expected parsed text to be preserved for failure reporting, got %q", result.Text)
	}
	if result.SessionID != "33333333-3333-4333-8333-333333333333" {
		t.Fatalf("expected parsed session id to be preserved, got %q", result.SessionID)
	}
	if !strings.Contains(result.Log, "partial answer") {
		t.Fatalf("expected raw log to be preserved, got %q", result.Log)
	}
	if result.Usage.ProviderUsage == nil || result.Usage.ProviderUsage.InputTokens != 7 || result.Usage.ProviderUsage.OutputTokens != 5 {
		t.Fatalf("expected parsed usage to be preserved, got %#v", result.Usage.ProviderUsage)
	}
}

func TestClaudeExecutorCreatesMissingWorkDir(t *testing.T) {
	dir := t.TempDir()
	workDir := filepath.Join(dir, "missing-workdir")
	command := filepath.Join(dir, "fake-claude")
	script := `#!/bin/sh
cat >/dev/null
printf '%s\n' '{"type":"result","session_id":"44444444-4444-4444-8444-444444444444","result":"created workdir","usage":{"input_tokens":3,"output_tokens":2}}'
`
	if err := os.WriteFile(command, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	result, err := (ClaudeExecutor{Command: command, WorkDir: workDir}).Run(context.Background(), AgentRequest{
		Prompt:        "test prompt",
		AgentExecutor: "claude",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "created workdir" {
		t.Fatalf("unexpected result text %q", result.Text)
	}
	if info, err := os.Stat(workDir); err != nil || !info.IsDir() {
		t.Fatalf("expected workdir to be created, info=%#v err=%v", info, err)
	}
}

func TestClaudeExecutorBuildsMissionBoundMCPConfig(t *testing.T) {
	executor := ClaudeExecutor{
		MCPServer: ClaudeMCPServer{
			Name:    "plasma",
			Command: "/tmp/plasma",
			Args:    []string{"mcp", "-db", "/tmp/plasma.db"},
		},
	}
	path, cleanup, err := executor.writeMCPConfig(AgentRequest{
		MissionID:     "mis_1",
		ToolSessionID: "ses_1",
		UserEventID:   "evt_1",
		AgentExecutor: "claude",
	})
	if err != nil {
		t.Fatalf("writeMCPConfig returned error: %v", err)
	}
	defer cleanup()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read mcp config: %v", err)
	}
	var config struct {
		MCPServers map[string]struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatalf("unmarshal mcp config: %v; raw=%s", err, raw)
	}
	plasma, ok := config.MCPServers["plasma"]
	if !ok {
		t.Fatalf("expected plasma MCP server, got %#v", config.MCPServers)
	}
	if plasma.Command != "/tmp/plasma" {
		t.Fatalf("expected command /tmp/plasma, got %q", plasma.Command)
	}
	for _, expected := range []string{"-mission-id", "mis_1", "-agent-session-id", "ses_1", "-current-user-event-id", "evt_1", "-agent-executor", "claude"} {
		if !hasString(plasma.Args, expected) {
			t.Fatalf("expected MCP config args to include %q, got %#v", expected, plasma.Args)
		}
	}
}

func TestClaudeExecutorDisallowsBuiltinTools(t *testing.T) {
	args := (ClaudeExecutor{}).baseArgs("", false)
	start := indexOfArg(args, "--disallowedTools")
	if start < 0 {
		t.Fatalf("expected --disallowedTools in args, got %#v", args)
	}
	for _, tool := range []string{"Bash", "Write", "Edit", "MultiEdit", "Task", "TodoWrite"} {
		index := indexOfArg(args, tool)
		if index <= start {
			t.Fatalf("expected %s to be listed after --disallowedTools, got %#v", tool, args)
		}
	}
	for _, tool := range []string{"Read", "Glob", "Grep", "LS", "WebFetch", "WebSearch"} {
		if index := indexOfArg(args, tool); index >= 0 {
			t.Fatalf("expected %s to stay available, got %#v", tool, args)
		}
	}
}

func TestClaudeExecutorAllowsPlasmaMCPTools(t *testing.T) {
	executor := ClaudeExecutor{
		MCPServer: ClaudeMCPServer{Name: "plasma"},
	}
	args := executor.baseArgs("", false)
	value := argValueAfter(args, "--allowedTools")
	for _, expected := range []string{"mcp__plasma__*", "Read", "Glob", "Grep", "LS", "WebFetch", "WebSearch"} {
		if !strings.Contains(value, expected) {
			t.Fatalf("expected allowlist to include %q, got %q in %#v", expected, value, args)
		}
	}
}

func TestClaudeExecutorCanRestrictRequestToMCPOnly(t *testing.T) {
	executor := ClaudeExecutor{
		MCPServer: ClaudeMCPServer{Name: "plasma"},
	}
	args := executor.baseArgsForRequest(AgentRequest{ReplaceMCPTools: true})
	value := argValueAfter(args, "--allowedTools")
	if value != "mcp__plasma__*" {
		t.Fatalf("expected MCP-only allowlist, got %q in %#v", value, args)
	}
	start := indexOfArg(args, "--disallowedTools")
	if start < 0 {
		t.Fatalf("expected --disallowedTools in args, got %#v", args)
	}
	for _, tool := range []string{"Read", "Glob", "Grep", "LS", "WebFetch", "WebSearch"} {
		index := indexOfArg(args, tool)
		if index <= start {
			t.Fatalf("expected MCP-only request to disallow %s, got %#v", tool, args)
		}
	}
}

func TestClaudeExecutorWritesSameBoundReportPlanMCPContext(t *testing.T) {
	executor := ClaudeExecutor{MCPServer: ClaudeMCPServer{Name: "plasma", Command: "/tmp/plasma", Args: []string{"mcp", "-db", "/tmp/plasma.db", "-enabled-tool", "plasma.sources.read"}}}
	path, cleanup, err := executor.writeMCPConfig(AgentRequest{MissionID: "mis_1", ToolSessionID: "ses_tool", AgentExecutor: "claude", ExtraMCPTools: []string{"plasma.report.plan.submit"}, ReportPlan: &AgentReportPlanContext{PendingEventID: "evt_pending", ReportMode: "long_form", IdempotencyKey: "key_1", PreviousProviderSessionID: "ses_previous", AgentModel: "claude-test", AgentReasoningEffort: "high"}})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, expected := range []string{"plasma.sources.read", "plasma.report.plan.submit", "-report-plan-pending-event-id", "evt_pending", "-report-plan-mode", "long_form", "-report-plan-idempotency-key", "key_1", "-report-plan-tool-session-id", "ses_tool", "-report-plan-previous-provider-session-id", "ses_previous", "-report-plan-agent-model", "claude-test"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("missing %q in %s", expected, text)
		}
	}
}

func TestClaudeExecutorCanDisableTools(t *testing.T) {
	executor := ClaudeExecutor{
		MCPServer: ClaudeMCPServer{
			Name:    "plasma",
			Command: "/tmp/plasma",
			Args:    []string{"mcp", "-db", "/tmp/plasma.db"},
		},
	}
	args := executor.baseArgs("", true)
	if index := indexOfArg(args, "--allowedTools"); index >= 0 {
		t.Fatalf("expected disabled tools to omit --allowedTools, got %#v", args)
	}
	start := indexOfArg(args, "--disallowedTools")
	if start < 0 {
		t.Fatalf("expected disabled tools to keep --disallowedTools, got %#v", args)
	}
	for _, tool := range append(claudeDisallowedBuiltinTools(), claudeAllowedBuiltinTools()...) {
		index := indexOfArg(args, tool)
		if index <= start {
			t.Fatalf("expected disabled tools to disallow %s, got %#v", tool, args)
		}
	}
	path, cleanup, err := executor.writeMCPConfig(AgentRequest{
		MissionID:     "mis_1",
		ToolSessionID: "ses_1",
		DisableTools:  true,
	})
	if err != nil {
		t.Fatalf("writeMCPConfig returned error: %v", err)
	}
	defer cleanup()
	if path != "" {
		t.Fatalf("expected disabled tools to omit MCP config file, got %q", path)
	}
}

func TestClaudeEnvironmentPassesClaudeConfigDir(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), "claude-config")
	t.Setenv("CLAUDE_CONFIG_DIR", configDir)
	t.Setenv("PLASMA_RUNTIME_MODE", "dev")
	if got := claudeHomeFromEnv(nil); got != configDir {
		t.Fatalf("expected fork readiness to use CLAUDE_CONFIG_DIR, got %q", got)
	}
	if !hasString(claudeEnvironment(nil), "CLAUDE_CONFIG_DIR="+configDir) {
		t.Fatalf("expected child environment to include CLAUDE_CONFIG_DIR, got %#v", claudeEnvironment(nil))
	}
	if !hasString(claudeEnvironment(nil), "PLASMA_RUNTIME_MODE=dev") {
		t.Fatalf("expected child environment to include PLASMA_RUNTIME_MODE, got %#v", claudeEnvironment(nil))
	}
}

func TestClaudeExecutorCheckForkSessionRequiresExistingSessionFile(t *testing.T) {
	home := t.TempDir()
	sessionID := "33333333-3333-4333-8333-333333333333"
	executor := ClaudeExecutor{Env: []string{"HOME=" + home}}
	if err := executor.CheckForkSession(nil, sessionID); err == nil {
		t.Fatal("expected missing session file to fail readiness")
	}
	projectDir := filepath.Join(home, ".claude", "projects", "-tmp-project")
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, sessionID+".jsonl"), []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("write session file: %v", err)
	}
	if err := executor.CheckForkSession(nil, sessionID); err != nil {
		t.Fatalf("expected existing session file to pass readiness: %v", err)
	}
}

func TestClaudeExecutorCheckForkSessionRequiresWritableSessionDirectory(t *testing.T) {
	home := t.TempDir()
	sessionID := "44444444-4444-4444-8444-444444444444"
	executor := ClaudeExecutor{Env: []string{"HOME=" + home}}
	projectDir := filepath.Join(home, ".claude", "projects", "-tmp-project")
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, sessionID+".jsonl"), []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("write session file: %v", err)
	}
	if err := os.Chmod(projectDir, 0o500); err != nil {
		t.Fatalf("chmod project dir read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(projectDir, 0o700) })
	if err := executor.CheckForkSession(nil, sessionID); err == nil {
		t.Fatal("expected read-only session directory to fail readiness")
	}
}

func argValueAfter(args []string, key string) string {
	for i, value := range args {
		if value == key && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func indexOfArg(args []string, key string) int {
	for i, value := range args {
		if value == key {
			return i
		}
	}
	return -1
}

func hasString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
