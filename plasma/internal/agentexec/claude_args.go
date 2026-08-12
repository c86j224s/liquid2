package agentexec

import (
	"encoding/json"
	"os"
	"strings"
)

func (executor ClaudeExecutor) baseArgs(requestModel string, disableTools bool) []string {
	return executor.baseArgsWithToolMode(requestModel, disableTools, false)
}

func (executor ClaudeExecutor) baseArgsForRequest(req AgentRequest) []string {
	args := executor.baseArgsWithToolMode(req.Model, req.DisableTools, req.ReplaceMCPTools)
	if req.IgnoreUserConfig {
		args = append(args, "--safe-mode")
	}
	if req.EphemeralSession {
		args = append(args, "--no-session-persistence")
	}
	return args
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
		// local source root는 Claude --add-dir가 아니라 Plasma MCP 인자로만 노출한다.
		// 내장 파일 도구는 제품 정책이 더 넓은 디렉터리를 열기 전까지 설정된 workdir에 묶인다.
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

// WriteMCPConfig writes the Claude MCP config file for compatibility tests and transport wiring checks.
func (executor ClaudeExecutor) WriteMCPConfig(req AgentRequest) (string, func(), error) {
	return executor.writeMCPConfig(req)
}
