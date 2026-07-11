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
