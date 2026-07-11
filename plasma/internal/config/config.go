package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const RuntimeModeEnv = "PLASMA_RUNTIME_MODE"

const RuntimeModeRelease = "release"
const RuntimeModeDev = "dev"

const DBPathEnv = "PLASMA_DB_PATH"
const LocalSourceRootsEnv = "PLASMA_LOCAL_SOURCE_ROOTS"

const ConfluenceOAuthClientIDEnv = "PLASMA_CONFLUENCE_OAUTH_CLIENT_ID"
const ConfluenceOAuthClientSecretEnv = "PLASMA_CONFLUENCE_OAUTH_CLIENT_SECRET"
const ConfluenceOAuthRedirectURIEnv = "PLASMA_CONFLUENCE_OAUTH_REDIRECT_URI"
const ConfluenceOAuthScopesEnv = "PLASMA_CONFLUENCE_OAUTH_SCOPES"
const ConfluenceOAuthAuthorizeURLEnv = "PLASMA_CONFLUENCE_OAUTH_AUTHORIZE_URL"
const ConfluenceOAuthTokenURLEnv = "PLASMA_CONFLUENCE_OAUTH_TOKEN_URL"
const ConfluenceOAuthDiscoveryURLEnv = "PLASMA_CONFLUENCE_OAUTH_DISCOVERY_URL"

type Args struct {
	DBPath                      string
	Addr                        string
	Liquid2URL                  string
	Agent                       string
	CodexCommand                string
	ClaudeCommand               string
	ClaudeModel                 string
	ClaudeMaxBudgetUSD          string
	AgentWorkDir                string
	AgentTimeout                string
	WorkflowGoalModel           string
	WorkflowGoalReasoningEffort string
	StaticDir                   string
	EnvironmentLabel            string
	LocalSourceRoots            []string
	ConfluenceOAuthClientID     string
	ConfluenceOAuthClientSecret string
	ConfluenceOAuthRedirectURI  string
	ConfluenceOAuthScopes       []string
	ConfluenceOAuthAuthorizeURL string
	ConfluenceOAuthTokenURL     string
	ConfluenceOAuthDiscoveryURL string
}

type Config struct {
	DBPath                      string
	Addr                        string
	Liquid2URL                  string
	Agent                       string
	CodexCommand                string
	ClaudeCommand               string
	ClaudeModel                 string
	ClaudeMaxBudgetUSD          string
	AgentWorkDir                string
	AgentTimeout                string
	WorkflowGoalModel           string
	WorkflowGoalReasoningEffort string
	StaticDir                   string
	EnvironmentLabel            string
	LocalSourceRoots            []string
	ConfluenceOAuthClientID     string
	ConfluenceOAuthClientSecret string
	ConfluenceOAuthRedirectURI  string
	ConfluenceOAuthScopes       []string
	ConfluenceOAuthAuthorizeURL string
	ConfluenceOAuthTokenURL     string
	ConfluenceOAuthDiscoveryURL string
}

func Load(args Args) (Config, error) {
	var cfg Config
	mode, err := RuntimeMode()
	if err != nil {
		return Config{}, err
	}
	paths, err := configPaths(mode)
	if err != nil {
		return Config{}, err
	}
	for _, path := range paths {
		if err := cfg.applyFile(path); err != nil {
			return Config{}, err
		}
	}
	cfg.applyEnv()
	cfg.applyArgs(args)
	return cfg, nil
}

func (c *Config) applyFile(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read config %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("read config %s: is a directory", path)
	}
	raw := map[string]any{}
	if _, err := toml.DecodeFile(path, &raw); err != nil {
		return fmt.Errorf("parse config %s: %w", path, err)
	}
	c.applyTable(raw)
	c.applyTable(productTable(raw, "plasma"))
	c.applyServerTable(namedTable(raw, "plasma-server"))
	c.applyPathsTable(namedTable(raw, "plasma-paths"))
	c.applyAgentTables(namedTable(raw, "plasma-agents"))
	c.applyLocalSourcesTable(namedTable(raw, "plasma-local-sources"))
	c.applyConfluenceOAuthTable(namedTable(raw, "plasma-confluence-oauth"))
	return nil
}

func (c *Config) applyTable(values map[string]any) {
	for key, raw := range values {
		switch key {
		case "db_path", "addr", "liquid2_url", "agent", "codex_command",
			"claude_command", "claude_model", "claude_max_budget_usd",
			"agent_workdir", "agent_timeout", "workflow_goal_model",
			"workflow_goal_reasoning_effort", "static_dir", "environment_label",
			"confluence_oauth_client_id", "confluence_oauth_client_secret",
			"confluence_oauth_redirect_uri", "confluence_oauth_authorize_url",
			"confluence_oauth_token_url", "confluence_oauth_discovery_url":
			if value, ok := configValue(raw); ok {
				c.setString(key, value)
			}
		case "local_source_roots":
			c.setList(configList(raw))
		case "confluence_oauth_scopes":
			c.setConfluenceOAuthScopes(configList(raw))
		}
	}
}

func RuntimeMode() (string, error) {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(RuntimeModeEnv)))
	switch value {
	case "", RuntimeModeRelease:
		return RuntimeModeRelease, nil
	case RuntimeModeDev, "development":
		return RuntimeModeDev, nil
	default:
		return "", fmt.Errorf("%s must be release or dev", RuntimeModeEnv)
	}
}

func configPaths(mode string) ([]string, error) {
	switch mode {
	case RuntimeModeRelease:
		path, err := userConfigPath("plasma")
		if err != nil {
			return nil, err
		}
		return []string{path}, nil
	case RuntimeModeDev:
		path, err := userConfigPath("plasma-dev")
		if err != nil {
			return nil, err
		}
		return []string{path, "config.toml"}, nil
	default:
		return nil, fmt.Errorf("unknown Plasma runtime mode %q", mode)
	}
}

func userConfigPath(product string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", product, "config.toml"), nil
}
