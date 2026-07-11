package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFromEnvReadsPlasmaDBPath(t *testing.T) {
	clearEnv(t)
	t.Setenv(DBPathEnv, " /tmp/plasma.db ")
	cfg := FromEnv()
	if cfg.DBPath != "/tmp/plasma.db" {
		t.Fatalf("expected trimmed DB path, got %q", cfg.DBPath)
	}
}

func TestEffectiveDBPathDefaultsToMemory(t *testing.T) {
	cfg := Config{}
	if got := cfg.EffectiveDBPath(); got != ":memory:" {
		t.Fatalf("expected memory DB, got %q", got)
	}
	if got := cfg.DisplayDBPath(); got != "memory" {
		t.Fatalf("expected memory display, got %q", got)
	}
}

func TestLoadAppliesPriority(t *testing.T) {
	clearEnv(t)
	home := t.TempDir()
	work := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(RuntimeModeEnv, RuntimeModeDev)
	t.Setenv(DBPathEnv, "/env/plasma.db")
	t.Chdir(work)

	writeConfig(t, filepath.Join(work, "config.toml"), `db_path = "/local/plasma.db"`)
	writeConfig(t, filepath.Join(home, ".config", "plasma-dev", "config.toml"), `db_path = "/user/plasma.db"`)

	cfg, err := Load(Args{DBPath: "/arg/plasma.db"})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.DBPath != "/arg/plasma.db" {
		t.Fatalf("expected args to win, got %q", cfg.DBPath)
	}
}

func TestLoadReleaseModeIgnoresLocalConfig(t *testing.T) {
	clearEnv(t)
	home := t.TempDir()
	work := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(RuntimeModeEnv, RuntimeModeRelease)
	t.Chdir(work)

	writeConfig(t, filepath.Join(work, "config.toml"), `
[plasma-paths]
static_dir = "auto"
[plasma-local-sources]
roots = ["dev=/tmp/dev"]
`)
	writeConfig(t, filepath.Join(home, ".config", "plasma", "config.toml"), `
[plasma-paths]
db_path = "/release/plasma.db"
`)

	cfg, err := Load(Args{})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.DBPath != "/release/plasma.db" {
		t.Fatalf("expected user config to load, got %q", cfg.DBPath)
	}
	if cfg.StaticDir != "" || len(cfg.LocalSourceRoots) != 0 {
		t.Fatalf("expected local config to be skipped, got static_dir=%q roots=%#v", cfg.StaticDir, cfg.LocalSourceRoots)
	}
}

func TestLoadSupportsProductTableAndListValues(t *testing.T) {
	clearEnv(t)
	home := t.TempDir()
	work := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(RuntimeModeEnv, RuntimeModeDev)
	t.Chdir(work)
	writeConfig(t, filepath.Join(work, "config.toml"), `
[plasma]
addr = "127.0.0.1:6002"
local_source_roots = ["workspace=/tmp/workspace", "docs=/tmp/docs"]
`)

	cfg, err := Load(Args{})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Addr != "127.0.0.1:6002" {
		t.Fatalf("expected local config addr, got %q", cfg.Addr)
	}
	if len(cfg.LocalSourceRoots) != 2 || cfg.LocalSourceRoots[0] != "workspace=/tmp/workspace" {
		t.Fatalf("unexpected local roots %#v", cfg.LocalSourceRoots)
	}
}

func TestLoadSupportsStructuredTables(t *testing.T) {
	clearEnv(t)
	home := t.TempDir()
	work := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(RuntimeModeEnv, RuntimeModeDev)
	t.Chdir(work)

	writeConfig(t, filepath.Join(work, "config.toml"), `
[plasma]
addr = "127.0.0.1:6000"
agent = "codex"
local_source_roots = ["legacy=/legacy"]

[plasma-server]
addr = "127.0.0.1"
port = 6002
liquid2_url = "http://127.0.0.1:6011"
environment_label = "DEV"

[plasma-paths]
db_path = "/structured/plasma.db"
static_dir = "auto"

[plasma-agents]
enabled = ["codex", "claude"]
workdir = "/tmp/plasma-agent-workdir"
timeout = "45s"

[plasma-agents.codex]
command = "/opt/bin/codex"
workflow_goal_model = "gpt-5"
workflow_goal_reasoning_effort = "medium"

[plasma-agents.claude]
command = "/opt/bin/claude"
model = "sonnet"
max_budget_usd = "1.25"

[plasma-local-sources.roots]
workspace = "/workspace"
docs = "/docs"
`)

	cfg, err := Load(Args{})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Addr != "127.0.0.1:6002" {
		t.Fatalf("expected structured server addr, got %q", cfg.Addr)
	}
	if cfg.DBPath != "/structured/plasma.db" ||
		cfg.Liquid2URL != "http://127.0.0.1:6011" ||
		cfg.EnvironmentLabel != "DEV" {
		t.Fatalf("unexpected server config: %#v", cfg)
	}
	if cfg.Agent != "codex,claude" ||
		cfg.CodexCommand != "/opt/bin/codex" ||
		cfg.ClaudeCommand != "/opt/bin/claude" ||
		cfg.ClaudeModel != "sonnet" ||
		cfg.ClaudeMaxBudgetUSD != "1.25" ||
		cfg.AgentWorkDir != "/tmp/plasma-agent-workdir" ||
		cfg.AgentTimeout != "45s" ||
		cfg.WorkflowGoalModel != "gpt-5" ||
		cfg.WorkflowGoalReasoningEffort != "medium" {
		t.Fatalf("unexpected agent config: %#v", cfg)
	}
	if len(cfg.LocalSourceRoots) != 2 ||
		cfg.LocalSourceRoots[0] != "docs=/docs" ||
		cfg.LocalSourceRoots[1] != "workspace=/workspace" {
		t.Fatalf("unexpected local source root map: %#v", cfg.LocalSourceRoots)
	}
}

func TestLoadLocalSourceRootsReplaceLowerPriorityValues(t *testing.T) {
	for _, tc := range []struct {
		name        string
		userConfig  string
		localConfig string
		envValue    string
		argValue    string
		want        []string
	}{
		{
			name:     "args replace user local and env",
			envValue: "env=/env/root",
			argValue: "arg=/arg/root",
			localConfig: `
[plasma]
local_source_roots = ["local=/local/root"]
`,
			userConfig: `
[plasma]
local_source_roots = ["user=/user/root", "user=/user/root"]
`,
			want: []string{"arg=/arg/root"},
		},
		{
			name:     "env replaces user and local",
			envValue: "env=/env/root",
			localConfig: `
[plasma]
local_source_roots = ["local=/local/root"]
`,
			userConfig: `
[plasma]
local_source_roots = ["user=/user/root"]
`,
			want: []string{"env=/env/root"},
		},
		{
			name: "local replaces user",
			localConfig: `
[plasma]
local_source_roots = ["local=/local/root"]
`,
			userConfig: `
[plasma]
local_source_roots = ["user=/user/root"]
`,
			want: []string{"local=/local/root"},
		},
		{
			name: "user is base config",
			userConfig: `
[plasma]
local_source_roots = ["user=/user/root", "user=/user/root"]
`,
			want: []string{"user=/user/root"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clearEnv(t)
			home := t.TempDir()
			work := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv(RuntimeModeEnv, RuntimeModeDev)
			t.Setenv(LocalSourceRootsEnv, tc.envValue)
			t.Chdir(work)

			if tc.localConfig != "" {
				writeConfig(t, filepath.Join(work, "config.toml"), tc.localConfig)
			}
			if tc.userConfig != "" {
				writeConfig(t, filepath.Join(home, ".config", "plasma-dev", "config.toml"), tc.userConfig)
			}

			cfg, err := Load(Args{LocalSourceRoots: []string{tc.argValue}})
			if err != nil {
				t.Fatalf("load config: %v", err)
			}
			if len(cfg.LocalSourceRoots) != len(tc.want) {
				t.Fatalf("expected roots %#v, got %#v", tc.want, cfg.LocalSourceRoots)
			}
			for i, want := range tc.want {
				if cfg.LocalSourceRoots[i] != want {
					t.Fatalf("expected roots %#v, got %#v", tc.want, cfg.LocalSourceRoots)
				}
			}
		})
	}
}

func TestLoadArgsOverrideEnv(t *testing.T) {
	clearEnv(t)
	home := t.TempDir()
	work := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PLASMA_AGENT", "codex")
	t.Chdir(work)

	cfg, err := Load(Args{Agent: "none"})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Agent != "none" {
		t.Fatalf("expected args to win over env config, got %q", cfg.Agent)
	}
}

func TestLoadAppliesClaudeConfig(t *testing.T) {
	clearEnv(t)
	home := t.TempDir()
	work := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(RuntimeModeEnv, RuntimeModeDev)
	t.Chdir(work)

	writeConfig(t, filepath.Join(work, "config.toml"), `
[plasma]
claude_command = "/opt/claude"
claude_model = "sonnet"
claude_max_budget_usd = "1.25"
`)

	cfg, err := Load(Args{
		ClaudeCommand:      "claude",
		ClaudeModel:        "haiku",
		ClaudeMaxBudgetUSD: "0.10",
	})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.ClaudeCommand != "claude" {
		t.Fatalf("expected Claude command from args, got %q", cfg.ClaudeCommand)
	}
	if cfg.ClaudeModel != "haiku" {
		t.Fatalf("expected Claude model from args, got %q", cfg.ClaudeModel)
	}
	if cfg.ClaudeMaxBudgetUSD != "0.10" {
		t.Fatalf("expected Claude budget from args, got %q", cfg.ClaudeMaxBudgetUSD)
	}
}

func TestLoadAppliesConfluenceOAuthConfig(t *testing.T) {
	clearEnv(t)
	home := t.TempDir()
	work := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(ConfluenceOAuthClientIDEnv, "env-client")
	t.Setenv(RuntimeModeEnv, RuntimeModeDev)
	t.Chdir(work)

	writeConfig(t, filepath.Join(work, "config.toml"), `
[plasma]
confluence_oauth_client_id = "local-client"
confluence_oauth_client_secret = "local-secret"
confluence_oauth_redirect_uri = "http://127.0.0.1:6002/api/missions/mis_1/sources/confluence/oauth/callback"
confluence_oauth_scopes = ["read:confluence-content.all", "offline_access", "offline_access"]
confluence_oauth_authorize_url = "http://127.0.0.1:7000/authorize"
confluence_oauth_token_url = "http://127.0.0.1:7000/oauth/token"
confluence_oauth_discovery_url = "http://127.0.0.1:7000"
`)

	cfg, err := Load(Args{})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.ConfluenceOAuthClientID != "env-client" ||
		cfg.ConfluenceOAuthClientSecret != "local-secret" ||
		cfg.ConfluenceOAuthRedirectURI == "" ||
		cfg.ConfluenceOAuthAuthorizeURL != "http://127.0.0.1:7000/authorize" ||
		cfg.ConfluenceOAuthTokenURL != "http://127.0.0.1:7000/oauth/token" ||
		cfg.ConfluenceOAuthDiscoveryURL != "http://127.0.0.1:7000" {
		t.Fatalf("unexpected Confluence OAuth config: %#v", cfg)
	}
	if len(cfg.ConfluenceOAuthScopes) != 2 || cfg.ConfluenceOAuthScopes[0] != "read:confluence-content.all" {
		t.Fatalf("unexpected Confluence OAuth scopes: %#v", cfg.ConfluenceOAuthScopes)
	}
}

func TestLoadAppliesStructuredConfluenceOAuthConfig(t *testing.T) {
	clearEnv(t)
	home := t.TempDir()
	work := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(RuntimeModeEnv, RuntimeModeDev)
	t.Chdir(work)

	writeConfig(t, filepath.Join(work, "config.toml"), `
[plasma]
confluence_oauth_client_id = "legacy-client"

[plasma-confluence-oauth]
client_id = "structured-client"
client_secret = "structured-secret"
redirect_uri = "http://127.0.0.1:6002/callback"
scopes = ["read:confluence-content.all", "offline_access"]
authorize_url = "http://127.0.0.1:7000/authorize"
token_url = "http://127.0.0.1:7000/oauth/token"
discovery_url = "http://127.0.0.1:7000"
`)

	cfg, err := Load(Args{})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.ConfluenceOAuthClientID != "structured-client" ||
		cfg.ConfluenceOAuthClientSecret != "structured-secret" ||
		cfg.ConfluenceOAuthRedirectURI != "http://127.0.0.1:6002/callback" ||
		cfg.ConfluenceOAuthAuthorizeURL != "http://127.0.0.1:7000/authorize" ||
		cfg.ConfluenceOAuthTokenURL != "http://127.0.0.1:7000/oauth/token" ||
		cfg.ConfluenceOAuthDiscoveryURL != "http://127.0.0.1:7000" {
		t.Fatalf("unexpected structured Confluence OAuth config: %#v", cfg)
	}
	if len(cfg.ConfluenceOAuthScopes) != 2 || cfg.ConfluenceOAuthScopes[1] != "offline_access" {
		t.Fatalf("unexpected structured Confluence OAuth scopes: %#v", cfg.ConfluenceOAuthScopes)
	}
}

func TestLoadRejectsInvalidRuntimeMode(t *testing.T) {
	clearEnv(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv(RuntimeModeEnv, "staging")
	if _, err := Load(Args{}); err == nil {
		t.Fatal("expected invalid runtime mode error")
	}
}

func clearEnv(t *testing.T) {
	t.Helper()
	for _, envKey := range []string{
		DBPathEnv,
		LocalSourceRootsEnv,
		"PLASMA_ADDR",
		"PLASMA_LIQUID2_URL",
		"PLASMA_AGENT",
		"PLASMA_CODEX_COMMAND",
		"PLASMA_CLAUDE_COMMAND",
		"PLASMA_CLAUDE_MODEL",
		"PLASMA_CLAUDE_MAX_BUDGET_USD",
		"PLASMA_AGENT_WORKDIR",
		"PLASMA_AGENT_TIMEOUT",
		"PLASMA_WORKFLOW_GOAL_MODEL",
		"PLASMA_WORKFLOW_GOAL_REASONING_EFFORT",
		"PLASMA_STATIC_DIR",
		ConfluenceOAuthClientIDEnv,
		ConfluenceOAuthClientSecretEnv,
		ConfluenceOAuthRedirectURIEnv,
		ConfluenceOAuthScopesEnv,
		ConfluenceOAuthAuthorizeURLEnv,
		ConfluenceOAuthTokenURLEnv,
		ConfluenceOAuthDiscoveryURLEnv,
		RuntimeModeEnv,
	} {
		t.Setenv(envKey, "")
	}
}

func writeConfig(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
