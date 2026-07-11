package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

type exportFile struct {
	path string
	rel  string
}

func exportDirectoryStats(root string) (int64, string, error) {
	files := []exportFile{}
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || path == root {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("inspect export artifact: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("export artifact contains symlink")
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("export artifact contains non-regular file")
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, exportFile{path: path, rel: filepath.ToSlash(rel)})
		return nil
	}); err != nil {
		return 0, "", err
	}
	sort.Slice(files, func(i int, j int) bool { return files[i].rel < files[j].rel })
	return hashExportFiles(files)
}

func hashExportFiles(files []exportFile) (int64, string, error) {
	hash := sha256.New()
	var size int64
	for _, file := range files {
		hash.Write([]byte(file.rel))
		hash.Write([]byte{0})
		source, err := os.Open(file.path)
		if err != nil {
			return 0, "", err
		}
		n, copyErr := io.Copy(hash, source)
		closeErr := source.Close()
		if copyErr != nil {
			return 0, "", copyErr
		}
		if closeErr != nil {
			return 0, "", closeErr
		}
		size += n
		hash.Write([]byte{0})
	}
	return size, fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func ensureExportRoot(root string) error {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return fmt.Errorf("create export directory: %w", err)
	}
	info, err := os.Lstat(root)
	if err != nil {
		return fmt.Errorf("inspect export directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("export directory is invalid")
	}
	if info.Mode().Perm()&0o077 != 0 {
		if err := os.Chmod(root, 0o700); err != nil {
			return fmt.Errorf("secure export directory: %w", err)
		}
	}
	return nil
}
