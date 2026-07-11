package exporter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type DirectoryWriter struct {
	root string
}

func NewDirectoryWriter(root string) DirectoryWriter {
	return DirectoryWriter{root: root}
}

func (writer DirectoryWriter) WriteFile(ctx context.Context, relativePath string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	root, clean, err := writer.cleanPath(relativePath)
	if err != nil {
		return err
	}
	parent, name := filepath.Split(clean)
	parentDir, err := ensureDirectory(root, strings.TrimSuffix(parent, string(filepath.Separator)))
	if err != nil {
		return err
	}
	target := filepath.Join(parentDir, name)
	if err = rejectExistingTarget(target); err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("write export file: %w", err)
	}
	if _, err = file.Write(data); err != nil {
		_ = file.Close()
		_ = os.Remove(target)
		return fmt.Errorf("write export file: %w", err)
	}
	if err = file.Close(); err != nil {
		_ = os.Remove(target)
		return fmt.Errorf("close export file: %w", err)
	}
	return nil
}

func (writer DirectoryWriter) cleanPath(relativePath string) (string, string, error) {
	root := filepath.Clean(strings.TrimSpace(writer.root))
	if root == "" || root == "." {
		return "", "", fmt.Errorf("export root is required")
	}
	if relativePath == "" || filepath.IsAbs(relativePath) {
		return "", "", fmt.Errorf("export path must be relative")
	}
	clean := filepath.Clean(filepath.FromSlash(relativePath))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("export path escapes root")
	}
	return root, clean, nil
}

func ensureDirectory(root string, relativeDir string) (string, error) {
	current := root
	if err := ensureRootDirectory(current); err != nil {
		return "", err
	}
	if relativeDir == "" || relativeDir == "." {
		return current, nil
	}
	for _, part := range strings.Split(relativeDir, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		if err := ensureSafeDirectory(current); err != nil {
			return "", err
		}
	}
	return current, nil
}

func ensureRootDirectory(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("create export directory: %w", err)
	}
	return rejectUnsafeDirectory(path)
}

func ensureSafeDirectory(path string) error {
	_, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		if err = os.Mkdir(path, 0o700); err != nil {
			return fmt.Errorf("create export directory: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("inspect export directory: %w", err)
	}
	return rejectUnsafeDirectory(path)
}

func rejectUnsafeDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect export directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("export directory must not be a symlink")
	}
	if !info.IsDir() {
		return fmt.Errorf("export path parent must be a directory")
	}
	return nil
}

func rejectExistingTarget(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect export file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("export file must not be a symlink")
	}
	return fmt.Errorf("export file already exists")
}
