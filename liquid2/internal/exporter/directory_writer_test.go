package exporter

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDirectoryWriterWritesRelativeFiles(t *testing.T) {
	root := t.TempDir()
	writer := NewDirectoryWriter(root)

	if err := writer.WriteFile(context.Background(), "documents/doc_1.md", []byte("body")); err != nil {
		t.Fatalf("write file: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "documents", "doc_1.md"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if !bytes.Equal(data, []byte("body")) {
		t.Fatalf("unexpected data %q", string(data))
	}
	info, err := os.Stat(filepath.Join(root, "documents", "doc_1.md"))
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("expected file mode 0600, got %o", mode)
	}
}

func TestDirectoryWriterRejectsUnsafePaths(t *testing.T) {
	writer := NewDirectoryWriter(t.TempDir())
	for _, path := range []string{"", "/tmp/export.md", "../escape.md", "documents/../../escape.md"} {
		t.Run(path, func(t *testing.T) {
			if err := writer.WriteFile(context.Background(), path, []byte("x")); err == nil {
				t.Fatal("expected unsafe path error")
			}
		})
	}
}

func TestDirectoryWriterRejectsSymlinkedParent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on Windows")
	}
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "documents")); err != nil {
		t.Skipf("create symlink: %v", err)
	}
	writer := NewDirectoryWriter(root)

	if err := writer.WriteFile(context.Background(), "documents/doc_1.md", []byte("body")); err == nil {
		t.Fatal("expected symlinked parent rejection")
	}
	if _, err := os.Stat(filepath.Join(outside, "doc_1.md")); !os.IsNotExist(err) {
		t.Fatalf("expected no write outside export root, stat err=%v", err)
	}
}

func TestDirectoryWriterRejectsSymlinkedTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on Windows")
	}
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "documents"), 0o700); err != nil {
		t.Fatalf("create documents dir: %v", err)
	}
	if err := os.Symlink(filepath.Join(outside, "doc_1.md"), filepath.Join(root, "documents", "doc_1.md")); err != nil {
		t.Skipf("create symlink: %v", err)
	}
	writer := NewDirectoryWriter(root)

	if err := writer.WriteFile(context.Background(), "documents/doc_1.md", []byte("body")); err == nil {
		t.Fatal("expected symlinked target rejection")
	}
	if _, err := os.Stat(filepath.Join(outside, "doc_1.md")); !os.IsNotExist(err) {
		t.Fatalf("expected no write outside export root, stat err=%v", err)
	}
}

func TestDirectoryWriterRequiresRoot(t *testing.T) {
	writer := NewDirectoryWriter("")
	if err := writer.WriteFile(context.Background(), "manifest.json", []byte("{}")); err == nil {
		t.Fatal("expected root error")
	}
}
