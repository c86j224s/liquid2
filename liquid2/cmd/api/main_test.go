package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	liquidconfig "github.com/c86j224s/liquid2/internal/config"
)

func TestRunStatusReportsDevelopmentDefaults(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(liquidconfig.RuntimeModeEnv, liquidconfig.RuntimeModeDev)
	t.Chdir(t.TempDir())

	var stdout, stderr bytes.Buffer
	code := runWithArgs([]string{"status"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("status returned %d stderr=%q", code, stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, "Liquid2 dev") ||
		!strings.Contains(output, "API     http://127.0.0.1:6011") ||
		!strings.Contains(output, "Web     http://127.0.0.1:6001") ||
		!strings.Contains(output, filepath.Join("research-artifacts", "liquid2", "liquid2", "runtime", "dev-6011", "liquid2-dev.db")) {
		t.Fatalf("unexpected status output %q", output)
	}
}

func TestRunStatusFieldReportsResolvedWebPort(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(liquidconfig.RuntimeModeEnv, liquidconfig.RuntimeModeRelease)
	t.Chdir(t.TempDir())

	var stdout, stderr bytes.Buffer
	code := runWithArgs([]string{"status", "-field", "web_port"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("status returned %d stderr=%q", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "3001" {
		t.Fatalf("unexpected web port %q", stdout.String())
	}
}

func TestReleasePathsUseWSL2FallbackOnly(t *testing.T) {
	home := t.TempDir()
	dbPath, exportDir, backupDir := releasePathsForPlatform("darwin", "", home, "/xdg/data")
	assertReleasePaths(t, filepath.Join(home, "Library", "Application Support", "Liquid2"), dbPath, exportDir, backupDir)
	dbPath, exportDir, backupDir = releasePathsForPlatform("linux", "6.6.87.2-microsoft-standard-WSL2", home, "")
	assertReleasePaths(t, filepath.Join(home, ".local", "share", "liquid2"), dbPath, exportDir, backupDir)
	dbPath, exportDir, backupDir = releasePathsForPlatform("linux", "6.6.87.2-microsoft-standard-WSL2", home, "/xdg/data")
	assertReleasePaths(t, filepath.Join("/xdg/data", "liquid2"), dbPath, exportDir, backupDir)
	dbPath, exportDir, backupDir = releasePathsForPlatform("linux", "4.4.0-19041-Microsoft", home, "/xdg/data")
	assertReleasePaths(t, filepath.Join(home, "Library", "Application Support", "Liquid2"), dbPath, exportDir, backupDir)
}

func assertReleasePaths(t *testing.T, wantDir, dbPath, exportDir, backupDir string) {
	t.Helper()
	if dbPath != filepath.Join(wantDir, "liquid2.db") ||
		exportDir != filepath.Join(wantDir, "exports") ||
		backupDir != filepath.Join(wantDir, "backups") {
		t.Fatalf("unexpected release paths db=%q export=%q backup=%q want dir=%q", dbPath, exportDir, backupDir, wantDir)
	}
}

func TestRuntimeDefaultsDoNotOverrideConfiguredReleaseDBPath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(liquidconfig.RuntimeModeEnv, liquidconfig.RuntimeModeRelease)
	t.Setenv("LIQUID2_DB_PATH", "/configured/liquid2.db")
	t.Chdir(t.TempDir())

	cfg, err := loadRuntimeConfig(nil)
	if err != nil {
		t.Fatalf("load runtime config: %v", err)
	}
	if got := cfg.Value(liquidconfig.KeyDBPath, ""); got != "/configured/liquid2.db" {
		t.Fatalf("configured database path was replaced: %q", got)
	}
}
