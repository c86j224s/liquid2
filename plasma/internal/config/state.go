package config

import (
	"os"
	"strings"
)

func FromEnv() Config {
	var cfg Config
	cfg.applyEnv()
	return cfg
}

func (c Config) EffectiveDBPath() string {
	if strings.TrimSpace(c.DBPath) == "" {
		return ":memory:"
	}
	return c.DBPath
}

func (c Config) DisplayDBPath() string {
	if strings.TrimSpace(c.DBPath) == "" {
		return "memory"
	}
	return c.DBPath
}

func (c *Config) applyArgs(args Args) {
	c.setString("db_path", args.DBPath)
	c.setString("addr", args.Addr)
	c.setString("liquid2_url", args.Liquid2URL)
	c.setString("agent", args.Agent)
	c.setString("codex_command", args.CodexCommand)
	c.setString("claude_command", args.ClaudeCommand)
	c.setString("claude_model", args.ClaudeModel)
	c.setString("claude_max_budget_usd", args.ClaudeMaxBudgetUSD)
	c.setString("agent_workdir", args.AgentWorkDir)
	c.setString("agent_timeout", args.AgentTimeout)
	c.setString("workflow_goal_model", args.WorkflowGoalModel)
	c.setString("workflow_goal_reasoning_effort", args.WorkflowGoalReasoningEffort)
	c.setString("static_dir", args.StaticDir)
	c.setString("environment_label", args.EnvironmentLabel)
	c.setList(args.LocalSourceRoots)
	c.setString("confluence_oauth_client_id", args.ConfluenceOAuthClientID)
	c.setString("confluence_oauth_client_secret", args.ConfluenceOAuthClientSecret)
	c.setString("confluence_oauth_redirect_uri", args.ConfluenceOAuthRedirectURI)
	c.setConfluenceOAuthScopes(args.ConfluenceOAuthScopes)
	c.setString("confluence_oauth_authorize_url", args.ConfluenceOAuthAuthorizeURL)
	c.setString("confluence_oauth_token_url", args.ConfluenceOAuthTokenURL)
	c.setString("confluence_oauth_discovery_url", args.ConfluenceOAuthDiscoveryURL)
}

func (c *Config) applyEnv() {
	c.setString("db_path", os.Getenv(DBPathEnv))
	c.setString("addr", os.Getenv("PLASMA_ADDR"))
	c.setString("liquid2_url", os.Getenv("PLASMA_LIQUID2_URL"))
	c.setString("agent", os.Getenv("PLASMA_AGENT"))
	c.setString("codex_command", os.Getenv("PLASMA_CODEX_COMMAND"))
	c.setString("claude_command", os.Getenv("PLASMA_CLAUDE_COMMAND"))
	c.setString("claude_model", os.Getenv("PLASMA_CLAUDE_MODEL"))
	c.setString("claude_max_budget_usd", os.Getenv("PLASMA_CLAUDE_MAX_BUDGET_USD"))
	c.setString("agent_workdir", os.Getenv("PLASMA_AGENT_WORKDIR"))
	c.setString("agent_timeout", os.Getenv("PLASMA_AGENT_TIMEOUT"))
	c.setString("workflow_goal_model", os.Getenv("PLASMA_WORKFLOW_GOAL_MODEL"))
	c.setString("workflow_goal_reasoning_effort", os.Getenv("PLASMA_WORKFLOW_GOAL_REASONING_EFFORT"))
	c.setString("static_dir", os.Getenv("PLASMA_STATIC_DIR"))
	c.setString("environment_label", os.Getenv("PLASMA_ENVIRONMENT_LABEL"))
	c.setList(splitList(os.Getenv(LocalSourceRootsEnv)))
	c.setString("confluence_oauth_client_id", os.Getenv(ConfluenceOAuthClientIDEnv))
	c.setString("confluence_oauth_client_secret", os.Getenv(ConfluenceOAuthClientSecretEnv))
	c.setString("confluence_oauth_redirect_uri", os.Getenv(ConfluenceOAuthRedirectURIEnv))
	c.setConfluenceOAuthScopes(splitList(os.Getenv(ConfluenceOAuthScopesEnv)))
	c.setString("confluence_oauth_authorize_url", os.Getenv(ConfluenceOAuthAuthorizeURLEnv))
	c.setString("confluence_oauth_token_url", os.Getenv(ConfluenceOAuthTokenURLEnv))
	c.setString("confluence_oauth_discovery_url", os.Getenv(ConfluenceOAuthDiscoveryURLEnv))
}

func (c *Config) setString(key string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	switch key {
	case "db_path":
		c.DBPath = value
	case "addr":
		c.Addr = value
	case "liquid2_url":
		c.Liquid2URL = value
	case "agent":
		c.Agent = value
	case "codex_command":
		c.CodexCommand = value
	case "claude_command":
		c.ClaudeCommand = value
	case "claude_model":
		c.ClaudeModel = value
	case "claude_max_budget_usd":
		c.ClaudeMaxBudgetUSD = value
	case "agent_workdir":
		c.AgentWorkDir = value
	case "agent_timeout":
		c.AgentTimeout = value
	case "workflow_goal_model":
		c.WorkflowGoalModel = value
	case "workflow_goal_reasoning_effort":
		c.WorkflowGoalReasoningEffort = value
	case "static_dir":
		c.StaticDir = value
	case "environment_label":
		c.EnvironmentLabel = value
	case "confluence_oauth_client_id":
		c.ConfluenceOAuthClientID = value
	case "confluence_oauth_client_secret":
		c.ConfluenceOAuthClientSecret = value
	case "confluence_oauth_redirect_uri":
		c.ConfluenceOAuthRedirectURI = value
	case "confluence_oauth_authorize_url":
		c.ConfluenceOAuthAuthorizeURL = value
	case "confluence_oauth_token_url":
		c.ConfluenceOAuthTokenURL = value
	case "confluence_oauth_discovery_url":
		c.ConfluenceOAuthDiscoveryURL = value
	}
}

func (c *Config) setList(values []string) {
	values = normalizeList(values)
	if len(values) > 0 {
		c.LocalSourceRoots = values
	}
}

func (c *Config) setConfluenceOAuthScopes(values []string) {
	values = normalizeList(values)
	if len(values) > 0 {
		c.ConfluenceOAuthScopes = values
	}
}

func normalizeList(values []string) []string {
	values = splitList(strings.Join(values, ","))
	if len(values) == 0 {
		return nil
	}
	merged := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		merged = append(merged, value)
	}
	return merged
}
