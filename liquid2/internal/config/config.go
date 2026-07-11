package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const RuntimeModeEnv = "LIQUID2_RUNTIME_MODE"

const RuntimeModeRelease = "release"
const RuntimeModeDev = "dev"

const (
	KeyAddr                = "addr"
	KeyDBPath              = "db_path"
	KeyLogLevel            = "log_level"
	KeyLogFormat           = "log_format"
	KeyLogSource           = "log_source"
	KeySeedDemo            = "seed_demo"
	KeyJobsEnabled         = "jobs_enabled"
	KeyTranslationProvider = "translation_provider"
	KeyCodexCommand        = "codex_command"
	KeyCodexModel          = "codex_model"
	KeyCodexTimeoutSeconds = "codex_timeout_seconds"
	KeyExportDir           = "export_dir"
	KeyBackupDir           = "backup_dir"
	KeyCORSOrigins         = "cors_origins"
	KeyWebAddr             = "web_addr"
	KeyWebPort             = "web_port"
	KeyEnvironmentLabel    = "environment_label"
)

type Args map[string]string

type Config struct {
	values map[string]string
}

type configFile struct {
	path  string
	scope string
}

const (
	configScopeUnified = "unified"
	configScopeAPI     = "api"
	configScopeWeb     = "web"
)

var envKeys = map[string]string{
	"LIQUID2_ADDR":                  KeyAddr,
	"LIQUID2_DB_PATH":               KeyDBPath,
	"LIQUID2_LOG_LEVEL":             KeyLogLevel,
	"LIQUID2_LOG_FORMAT":            KeyLogFormat,
	"LIQUID2_LOG_SOURCE":            KeyLogSource,
	"LIQUID2_SEED_DEMO":             KeySeedDemo,
	"LIQUID2_JOBS_ENABLED":          KeyJobsEnabled,
	"LIQUID2_TRANSLATION_PROVIDER":  KeyTranslationProvider,
	"LIQUID2_CODEX_COMMAND":         KeyCodexCommand,
	"LIQUID2_CODEX_MODEL":           KeyCodexModel,
	"LIQUID2_CODEX_TIMEOUT_SECONDS": KeyCodexTimeoutSeconds,
	"LIQUID2_EXPORT_DIR":            KeyExportDir,
	"LIQUID2_BACKUP_DIR":            KeyBackupDir,
	"LIQUID2_CORS_ORIGINS":          KeyCORSOrigins,
	"LIQUID2_WEB_ADDR":              KeyWebAddr,
	"LIQUID2_WEB_PORT":              KeyWebPort,
	"LIQUID2_ENVIRONMENT_LABEL":     KeyEnvironmentLabel,
}

var supportedKeys = map[string]struct{}{
	KeyAddr: {}, KeyDBPath: {}, KeyLogLevel: {}, KeyLogFormat: {},
	KeyLogSource: {}, KeySeedDemo: {}, KeyJobsEnabled: {},
	KeyTranslationProvider: {}, KeyCodexCommand: {}, KeyCodexModel: {},
	KeyCodexTimeoutSeconds: {}, KeyExportDir: {}, KeyBackupDir: {},
	KeyCORSOrigins: {}, KeyWebAddr: {}, KeyWebPort: {}, KeyEnvironmentLabel: {},
}

func Load(args Args) (Config, error) {
	cfg := Config{values: map[string]string{}}
	paths, err := configPaths()
	if err != nil {
		return Config{}, err
	}
	for _, file := range paths {
		if err := cfg.applyFile(file); err != nil {
			return Config{}, err
		}
	}
	cfg.applyEnv()
	cfg.apply(args)
	return cfg, nil
}

func (c Config) Value(key string, fallback string) string {
	if value := strings.TrimSpace(c.values[key]); value != "" {
		return value
	}
	return fallback
}

func (c Config) EnvValue(envKey string, fallback string) string {
	key, ok := envKeys[envKey]
	if !ok {
		if value := os.Getenv(envKey); value != "" {
			return value
		}
		return fallback
	}
	return c.Value(key, fallback)
}

func (c Config) apply(args Args) {
	for key, value := range args {
		if _, ok := supportedKeys[key]; !ok {
			continue
		}
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			c.values[key] = trimmed
		}
	}
}

func (c Config) ApplyDefaults(defaults Args) {
	for key, value := range defaults {
		if _, ok := supportedKeys[key]; !ok {
			continue
		}
		if strings.TrimSpace(c.values[key]) != "" {
			continue
		}
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			c.values[key] = trimmed
		}
	}
}

func (c Config) applyEnv() {
	for envKey, key := range envKeys {
		if value := strings.TrimSpace(os.Getenv(envKey)); value != "" {
			c.values[key] = value
		}
	}
}

func (c Config) applyFile(file configFile) error {
	path := file.path
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
	switch file.scope {
	case configScopeUnified:
		c.applyTable(raw)
		c.applyTable(productTable(raw, "liquid2"))
		c.applyAPITable(namedTable(raw, "liquid2-api"))
		c.applyAPIGroupTables(namedTable(raw, "liquid2-api"))
		c.applyWebTable(namedTable(raw, "liquid2-web"))
		c.applyWebGroupTables(namedTable(raw, "liquid2-web"))
	case configScopeAPI:
		c.applyTable(raw)
		c.applyTable(productTable(raw, "liquid2-api"))
		c.applyAPITable(namedTable(raw, "liquid2-api"))
		c.applyAPIGroupTables(raw)
		c.applyAPIGroupTables(namedTable(raw, "liquid2-api"))
	case configScopeWeb:
		c.applyWebTable(raw)
		c.applyWebTable(namedTable(raw, "liquid2-web"))
		c.applyWebGroupTables(raw)
		c.applyWebGroupTables(namedTable(raw, "liquid2-web"))
	default:
		return fmt.Errorf("unknown config scope %q for %s", file.scope, path)
	}
	return nil
}

func (c Config) applyTable(values map[string]any) {
	for key, raw := range values {
		if _, ok := supportedKeys[key]; !ok {
			continue
		}
		if value, ok := configValue(raw); ok {
			c.values[key] = value
		}
	}
}

func (c Config) applyAPITable(values map[string]any) {
	c.applyTable(values)
	if addr, ok := combinedAddrValue(values); ok {
		c.values[KeyAddr] = addr
	}
}

func configPaths() ([]configFile, error) {
	mode, err := RuntimeMode()
	if err != nil {
		return nil, err
	}
	switch mode {
	case RuntimeModeRelease:
		return userConfigFiles(
			configFile{path: "liquid2", scope: configScopeUnified},
			configFile{path: "liquid2-api", scope: configScopeAPI},
			configFile{path: "liquid2-web", scope: configScopeWeb},
		)
	case RuntimeModeDev:
		paths, err := userConfigFiles(
			configFile{path: "liquid2-dev", scope: configScopeUnified},
			configFile{path: "liquid2-api-dev", scope: configScopeAPI},
			configFile{path: "liquid2-web-dev", scope: configScopeWeb},
		)
		if err != nil {
			return nil, err
		}
		return append(paths,
			configFile{path: "config.toml", scope: configScopeUnified},
			configFile{path: "config.api.toml", scope: configScopeAPI},
			configFile{path: "config.web.toml", scope: configScopeWeb},
		), nil
	default:
		return nil, fmt.Errorf("unknown Liquid2 runtime mode %q", mode)
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

func userConfigFiles(files ...configFile) ([]configFile, error) {
	paths := make([]configFile, 0, len(files))
	for _, file := range files {
		path, err := userConfigPath(file.path)
		if err != nil {
			return nil, err
		}
		paths = append(paths, configFile{path: path, scope: file.scope})
	}
	return paths, nil
}

func userConfigPath(product string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", product, "config.toml"), nil
}
