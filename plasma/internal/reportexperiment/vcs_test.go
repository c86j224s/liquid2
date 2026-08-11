package reportexperiment

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestReadExecutableMetadataCanonicalizesAndRequiresExecutable(t *testing.T) {
	dir := t.TempDir()
	executable := filepath.Join(dir, "tool")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "tool-link")
	if err := os.Symlink(executable, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	metadata, err := ReadExecutableMetadata(link)
	if err != nil {
		t.Fatal(err)
	}
	expectedPath, err := filepath.EvalSymlinks(executable)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Path != expectedPath || metadata.SHA256 != fileSHA256ForTest(t, executable) {
		t.Fatalf("metadata = %#v, want canonical executable receipt", metadata)
	}

	notExecutable := filepath.Join(dir, "not-executable")
	if err := os.WriteFile(notExecutable, []byte("no"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadExecutableMetadata(notExecutable); !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("ReadExecutableMetadata error = %v, want ErrInvalidInput", err)
	}
}
