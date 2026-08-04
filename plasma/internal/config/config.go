package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// RuntimeModeEnv는 release/dev config 경로 선택에 쓰는 환경 변수다.
const RuntimeModeEnv = "PLASMA_RUNTIME_MODE"

// RuntimeModeRelease와 RuntimeModeDev는 config 파일 계층을 고르는 닫힌 runtime mode다.
const RuntimeModeRelease = "release"
const RuntimeModeDev = "dev"

// DBPathEnv는 Plasma SQLite DB 위치를 지정한다.
const DBPathEnv = "PLASMA_DB_PATH"

// LocalSourceRootsEnv는 local path source allowlist root들을 지정한다.
const LocalSourceRootsEnv = "PLASMA_LOCAL_SOURCE_ROOTS"

const ConfluenceOAuthClientIDEnv = "PLASMA_CONFLUENCE_OAUTH_CLIENT_ID"
const ConfluenceOAuthClientSecretEnv = "PLASMA_CONFLUENCE_OAUTH_CLIENT_SECRET"
const ConfluenceOAuthRedirectURIEnv = "PLASMA_CONFLUENCE_OAUTH_REDIRECT_URI"
const ConfluenceOAuthScopesEnv = "PLASMA_CONFLUENCE_OAUTH_SCOPES"
const ConfluenceOAuthAuthorizeURLEnv = "PLASMA_CONFLUENCE_OAUTH_AUTHORIZE_URL"
const ConfluenceOAuthTokenURLEnv = "PLASMA_CONFLUENCE_OAUTH_TOKEN_URL"
const ConfluenceOAuthDiscoveryURLEnv = "PLASMA_CONFLUENCE_OAUTH_DISCOVERY_URL"

// Args는 CLI flag와 embedding code가 Config에 덮어쓸 수 있는 입력값이다.
//
// 빈 값은 “지정하지 않음”으로 처리되며 파일/환경 변수에서 온 값을 지우지 않는다.
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

// Config는 Plasma server를 시작할 때 필요한 정규화된 실행 설정이다.
//
// Config 자체는 immutable이 아니지만 Load 이후에는 adapter와 service에 주입하는
// 값으로 취급한다. credential 성격의 필드는 로그나 사용자 응답에 그대로 노출하면
// 안 된다.
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

// Load는 runtime mode에 맞는 설정 파일을 읽고 환경 변수와 명령행 인자를 순서대로
// 적용한다.
//
// 우선순위는 파일 < 환경 변수 < Args다. 이 함수는 입력을 병합하며, StaticDir 같은
// 경로 값의 존재 여부와 제공 가능성 검증은 서버 구성 경계가 맡는다.
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

// RuntimeMode는 현재 실행이 dev/release 중 어느 config 계층을 쓸지 판정한다.
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
