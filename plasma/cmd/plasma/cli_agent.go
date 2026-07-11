package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/mcp"
	"github.com/c86j224s/liquid2/plasma/internal/web"
)

func codexEnabledTools() []string {
	return agentMCPEnabledTools()
}

func agentMCPEnabledTools() []string {
	return []string{
		mcp.ToolResearchOutline,
		mcp.ToolResearchList,
		mcp.ToolResearchGrep,
		mcp.ToolResearchRead,
		mcp.ToolResearchRefs,
		mcp.ToolSourcesList,
		mcp.ToolSourcesRead,
		mcp.ToolSourcesTree,
		mcp.ToolSourcesGrep,
		mcp.ToolSourcesSearch,
		mcp.ToolSourceCandidatesPropose,
		mcp.ToolSourceCandidatesRead,
		mcp.ToolWorkflowStart,
		mcp.ToolWorkflowStatus,
		mcp.ToolWorkflowStop,
	}
}

func appendAgentMCPEnabledToolArgs(args []string) []string {
	for _, tool := range agentMCPEnabledTools() {
		args = append(args, "-enabled-tool", tool)
	}
	return args
}

func codexSharedDBPath(dbPath string) (string, error) {
	return agentSharedDBPath(dbPath)
}

func agentSharedDBPath(dbPath string) (string, error) {
	trimmed := strings.TrimSpace(dbPath)
	if trimmed == "" || trimmed == ":memory:" {
		return "", fmt.Errorf("agent requires a file-backed Plasma database")
	}
	if filepath.IsAbs(trimmed) {
		return trimmed, nil
	}
	return filepath.Abs(trimmed)
}

func plasmaExecutablePath() string {
	if executable, err := os.Executable(); err == nil && strings.TrimSpace(executable) != "" {
		return executable
	}
	return os.Args[0]
}

func codexWorkDir(value string) (string, error) {
	workDir := strings.TrimSpace(value)
	if workDir == "" {
		workDir = filepath.Join(os.TempDir(), "plasma-agent-workdir")
	}
	absolute, err := filepath.Abs(workDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(absolute, 0o700); err != nil {
		return "", err
	}
	return absolute, nil
}

type cliAgentConfig struct {
	AgentName          string
	DBPath             string
	Liquid2URL         string
	CodexCommand       string
	ClaudeCommand      string
	ClaudeModel        string
	ClaudeMaxBudgetUSD string
	AgentWorkDir       string
	AgentTimeout       time.Duration
	LocalRoots         []string
}

var newCLIAgentExecutor = buildCLIAgentExecutor

func buildAgentExecutorMap(ctx context.Context, cfg cliAgentConfig) (map[string]web.AgentExecutor, error) {
	names, err := parseAgentExecutorList(cfg.AgentName)
	if err != nil {
		return nil, err
	}
	agents := map[string]web.AgentExecutor{}
	for _, name := range names {
		itemCfg := cfg
		itemCfg.AgentName = name
		executor, err := newCLIAgentExecutor(ctx, itemCfg)
		if err != nil {
			return nil, err
		}
		agents[name] = executor
	}
	return agents, nil
}

func parseAgentExecutorList(value string) ([]string, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" || value == "none" {
		return nil, nil
	}
	if value == "all" || value == "both" {
		value = "codex,claude"
	}
	seen := map[string]struct{}{}
	var names []string
	for _, part := range strings.Split(value, ",") {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		switch name {
		case "codex", "claude":
		default:
			return nil, fmt.Errorf("unsupported agent %q", name)
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return names, nil
}

func buildCLIAgentExecutor(_ context.Context, cfg cliAgentConfig) (web.AgentExecutor, error) {
	agentName := strings.TrimSpace(cfg.AgentName)
	if agentName == "" {
		agentName = "codex"
	}
	switch agentName {
	case "codex":
		sharedDBPath, err := agentSharedDBPath(cfg.DBPath)
		if err != nil {
			return nil, err
		}
		workDir, err := codexWorkDir(cfg.AgentWorkDir)
		if err != nil {
			return nil, err
		}
		mcpArgs := []string{"mcp", "-db", sharedDBPath}
		if strings.TrimSpace(cfg.Liquid2URL) != "" {
			mcpArgs = append(mcpArgs, "-liquid2-url", strings.TrimSpace(cfg.Liquid2URL))
		}
		for _, rootSpec := range cfg.LocalRoots {
			rootSpec = strings.TrimSpace(rootSpec)
			if rootSpec != "" {
				mcpArgs = append(mcpArgs, "-local-source-root", rootSpec)
			}
		}
		mcpArgs = appendAgentMCPEnabledToolArgs(mcpArgs)
		return web.CodexExecutor{
			Command: strings.TrimSpace(cfg.CodexCommand),
			WorkDir: workDir,
			Timeout: cfg.AgentTimeout,
			MCPServer: web.CodexMCPServer{
				Name:              "plasma",
				Command:           plasmaExecutablePath(),
				Args:              mcpArgs,
				Required:          true,
				StartupTimeoutSec: 10,
				ToolTimeoutSec:    60,
				EnabledTools:      codexEnabledTools(),
			},
		}, nil
	case "claude":
		sharedDBPath, err := agentSharedDBPath(cfg.DBPath)
		if err != nil {
			return nil, err
		}
		workDir, err := codexWorkDir(cfg.AgentWorkDir)
		if err != nil {
			return nil, err
		}
		return web.ClaudeExecutor{
			Command:      strings.TrimSpace(cfg.ClaudeCommand),
			WorkDir:      workDir,
			Model:        firstNonEmptyString(strings.TrimSpace(cfg.ClaudeModel), "haiku"),
			Timeout:      cfg.AgentTimeout,
			Permission:   "dontAsk",
			MaxBudgetUSD: strings.TrimSpace(cfg.ClaudeMaxBudgetUSD),
			MCPServer: web.ClaudeMCPServer{
				Name:    "plasma",
				Command: plasmaExecutablePath(),
				Args:    agentMCPArgs(sharedDBPath, cfg.Liquid2URL, cfg.LocalRoots),
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported agent %q", agentName)
	}
}

func agentMCPArgs(sharedDBPath string, liquid2URL string, localRoots []string) []string {
	mcpArgs := []string{"mcp", "-db", sharedDBPath}
	if strings.TrimSpace(liquid2URL) != "" {
		mcpArgs = append(mcpArgs, "-liquid2-url", strings.TrimSpace(liquid2URL))
	}
	for _, rootSpec := range localRoots {
		rootSpec = strings.TrimSpace(rootSpec)
		if rootSpec != "" {
			mcpArgs = append(mcpArgs, "-local-source-root", rootSpec)
		}
	}
	return appendAgentMCPEnabledToolArgs(mcpArgs)
}
