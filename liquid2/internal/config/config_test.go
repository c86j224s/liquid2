package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppliesPriority(t *testing.T) {
	clearEnv(t)
	home := t.TempDir()
	work := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(RuntimeModeEnv, RuntimeModeDev)
	t.Setenv("LIQUID2_DB_PATH", "/env/liquid2.db")
	t.Chdir(work)

	writeConfig(t, filepath.Join(work, "config.toml"), `db_path = "/local/unified.db"`)
	writeConfig(t, filepath.Join(work, "config.api.toml"), `db_path = "/local/api.db"`)
	writeConfig(t, filepath.Join(home, ".config", "liquid2-dev", "config.toml"), `db_path = "/user/unified.db"`)
	writeConfig(t, filepath.Join(home, ".config", "liquid2-api-dev", "config.toml"), `db_path = "/user/api.db"`)

	cfg, err := Load(Args{KeyDBPath: "/arg/liquid2.db"})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Value(KeyDBPath, ""); got != "/arg/liquid2.db" {
		t.Fatalf("expected args to win, got %q", got)
	}
}

func TestLoadReleaseModeIgnoresLocalConfig(t *testing.T) {
	clearEnv(t)
	home := t.TempDir()
	work := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(RuntimeModeEnv, RuntimeModeRelease)
	t.Chdir(work)

	writeConfig(t, filepath.Join(work, "config.toml"), `db_path = "/local/unified.db"`)
	writeConfig(t, filepath.Join(work, "config.api.toml"), `db_path = "/local/api.db"`)
	writeConfig(t, filepath.Join(home, ".config", "liquid2", "config.toml"), `db_path = "/user/unified.db"`)
	writeConfig(t, filepath.Join(home, ".config", "liquid2-api", "config.toml"), `db_path = "/user/api.db"`)

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Value(KeyDBPath, ""); got != "/user/api.db" {
		t.Fatalf("expected user API config to win while local config is skipped, got %q", got)
	}
}

func TestLoadSupportsProductTableAndArrayValues(t *testing.T) {
	clearEnv(t)
	home := t.TempDir()
	work := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(RuntimeModeEnv, RuntimeModeDev)
	t.Chdir(work)
	writeConfig(t, filepath.Join(work, "config.toml"), `
[liquid2]
cors_origins = ["http://127.0.0.1:6001", "http://localhost:6001"]
jobs_enabled = true
`)

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Value(KeyCORSOrigins, ""); got != "http://127.0.0.1:6001,http://localhost:6001" {
		t.Fatalf("unexpected CORS origins %q", got)
	}
	if got := cfg.Value(KeyJobsEnabled, ""); got != "1" {
		t.Fatalf("expected bool true to map to 1, got %q", got)
	}
}

func TestLoadSupportsStructuredAPITable(t *testing.T) {
	clearEnv(t)
	home := t.TempDir()
	work := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(RuntimeModeEnv, RuntimeModeDev)
	t.Chdir(work)
	writeConfig(t, filepath.Join(work, "config.toml"), `
[liquid2]
addr = "127.0.0.1:6000"
db_path = "/legacy/liquid2.db"

[liquid2-api]
addr = "127.0.0.1"
port = 6011
db_path = "/structured/liquid2.db"
`)

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Value(KeyAddr, ""); got != "127.0.0.1:6011" {
		t.Fatalf("expected structured API addr, got %q", got)
	}
	if got := cfg.Value(KeyDBPath, ""); got != "/structured/liquid2.db" {
		t.Fatalf("expected structured DB path, got %q", got)
	}
}

func TestLoadSupportsGroupedAPIConfigTables(t *testing.T) {
	clearEnv(t)
	home := t.TempDir()
	work := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(RuntimeModeEnv, RuntimeModeDev)
	t.Chdir(work)
	writeConfig(t, filepath.Join(work, "config.toml"), `
[liquid2-api]
addr = "127.0.0.1:6000"
db_path = "/legacy/liquid2.db"
`)
	writeConfig(t, filepath.Join(work, "config.api.toml"), `
[server]
addr = "127.0.0.1"
port = 6011
cors_origins = ["http://127.0.0.1:6001", "http://localhost:6001"]

[paths]
db_path = "/grouped/liquid2.db"
export_dir = "/grouped/exports"
backup_dir = "/grouped/backups"

[logging]
level = "debug"
format = "json"
source = true

[runtime]
jobs_enabled = true
seed_demo = false

[translation]
provider = "codex"

[translation.codex]
command = "/opt/bin/codex"
model = "gpt-5"
timeout_seconds = 120
`)

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	assertValue(t, cfg, KeyAddr, "127.0.0.1:6011")
	assertValue(t, cfg, KeyDBPath, "/grouped/liquid2.db")
	assertValue(t, cfg, KeyExportDir, "/grouped/exports")
	assertValue(t, cfg, KeyBackupDir, "/grouped/backups")
	assertValue(t, cfg, KeyLogLevel, "debug")
	assertValue(t, cfg, KeyLogFormat, "json")
	assertValue(t, cfg, KeyLogSource, "1")
	assertValue(t, cfg, KeyJobsEnabled, "1")
	assertValue(t, cfg, KeySeedDemo, "0")
	assertValue(t, cfg, KeyTranslationProvider, "codex")
	assertValue(t, cfg, KeyCodexCommand, "/opt/bin/codex")
	assertValue(t, cfg, KeyCodexModel, "gpt-5")
	assertValue(t, cfg, KeyCodexTimeoutSeconds, "120")
	assertValue(t, cfg, KeyCORSOrigins, "http://127.0.0.1:6001,http://localhost:6001")
}

func TestLoadSupportsPrefixedGroupedAPIConfigTables(t *testing.T) {
	clearEnv(t)
	home := t.TempDir()
	work := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(RuntimeModeEnv, RuntimeModeDev)
	t.Chdir(work)
	writeConfig(t, filepath.Join(work, "config.toml"), `
[liquid2-api.server]
addr = "127.0.0.1"
port = 6111

[liquid2-api.paths]
db_path = "/prefixed/liquid2.db"
`)

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	assertValue(t, cfg, KeyAddr, "127.0.0.1:6111")
	assertValue(t, cfg, KeyDBPath, "/prefixed/liquid2.db")
}

func TestLoadSupportsWebConfigTables(t *testing.T) {
	clearEnv(t)
	home := t.TempDir()
	work := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(RuntimeModeEnv, RuntimeModeDev)
	t.Chdir(work)
	writeConfig(t, filepath.Join(work, "config.web.toml"), `
[server]
addr = "127.0.0.1"
port = 6001

[runtime]
environment_label = "DEV"
`)

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	assertValue(t, cfg, KeyWebAddr, "127.0.0.1")
	assertValue(t, cfg, KeyWebPort, "6001")
	assertValue(t, cfg, KeyEnvironmentLabel, "DEV")
}

func TestLoadArgsOverrideEnv(t *testing.T) {
	clearEnv(t)
	home := t.TempDir()
	work := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("LIQUID2_LOG_FORMAT", "json")
	t.Chdir(work)

	cfg, err := Load(Args{KeyLogFormat: "text"})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Value(KeyLogFormat, ""); got != "text" {
		t.Fatalf("expected args to win over env config, got %q", got)
	}
}

func TestLoadRejectsInvalidRuntimeMode(t *testing.T) {
	clearEnv(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv(RuntimeModeEnv, "stage")
	if _, err := Load(nil); err == nil {
		t.Fatal("expected invalid runtime mode error")
	}
}

func clearEnv(t *testing.T) {
	t.Helper()
	for envKey := range envKeys {
		t.Setenv(envKey, "")
	}
	t.Setenv(RuntimeModeEnv, "")
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

func assertValue(t *testing.T, cfg Config, key string, want string) {
	t.Helper()
	if got := cfg.Value(key, ""); got != want {
		t.Fatalf("expected %s %q, got %q", key, want, got)
	}
}
