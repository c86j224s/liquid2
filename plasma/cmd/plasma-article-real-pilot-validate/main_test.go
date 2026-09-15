package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRequiresAllArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if got := run([]string{"--archive-root", t.TempDir()}, &stdout, &stderr); got != 2 {
		t.Fatalf("run()=%d want=2", got)
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "provide archive-root") {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestRunRejectsWrongProtocolHashWithoutProviderExecution(t *testing.T) {
	archive := t.TempDir()
	if err := os.Chmod(archive, 0o700); err != nil {
		t.Fatal(err)
	}
	protocolPath := filepath.Join(archive, "protocol.json")
	if err := os.WriteFile(protocolPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	got := run([]string{
		"--archive-root", archive,
		"--repository-root", t.TempDir(),
		"--protocol", protocolPath,
		"--protocol-sha256", strings.Repeat("0", 64),
	}, &stdout, &stderr)
	if got != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "SHA-256 mismatch") {
		t.Fatalf("run()=%d stdout=%q stderr=%q", got, stdout.String(), stderr.String())
	}
}
