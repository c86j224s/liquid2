package source

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

// ErrLocalPathInput identifies invalid local path input.
var ErrLocalPathInput = errors.New("invalid local path input")

// NormalizeLocalRelativePath normalizes and validates a local path source input.
func NormalizeLocalRelativePath(value string) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" {
		value = "."
	}
	if strings.ContainsRune(value, 0) || containsLocalPathControl(value) {
		return "", fmt.Errorf("%w: relative path contains control characters", ErrLocalPathInput)
	}
	if filepath.IsAbs(value) || strings.HasPrefix(value, "/") || strings.HasPrefix(value, `\\`) || looksLikeLocalWindowsAbs(value) {
		return "", fmt.Errorf("%w: absolute paths are not accepted", ErrLocalPathInput)
	}
	for _, part := range strings.Split(value, "/") {
		if part == ".." {
			return "", fmt.Errorf("%w: path traversal is not accepted", ErrLocalPathInput)
		}
	}
	cleaned := path.Clean(value)
	if cleaned == "/" {
		cleaned = "."
	}
	for _, part := range strings.Split(cleaned, "/") {
		if part == ".." {
			return "", fmt.Errorf("%w: path traversal is not accepted", ErrLocalPathInput)
		}
	}
	return cleaned, nil
}

// LocalPathTargetRelativePath combines a base path and optional subpath without traversal.
func LocalPathTargetRelativePath(relativePath string, subpath string) (string, string, error) {
	relative, err := NormalizeLocalRelativePath(relativePath)
	if err != nil {
		return "", "", err
	}
	cleanSubpath, err := NormalizeLocalRelativePath(subpath)
	if err != nil {
		return "", "", err
	}
	if cleanSubpath == "." {
		return relative, "", nil
	}
	return joinLocalRelative(relative, cleanSubpath), cleanSubpath, nil
}

func joinLocalRelative(base string, name string) string {
	if base == "." || base == "" {
		return name
	}
	return path.Clean(base + "/" + name)
}

func containsLocalPathControl(value string) bool {
	for _, r := range value {
		if r >= 0 && r < 0x20 {
			return true
		}
	}
	return false
}

func looksLikeLocalWindowsAbs(value string) bool {
	if len(value) >= 3 && ((value[0] >= 'a' && value[0] <= 'z') || (value[0] >= 'A' && value[0] <= 'Z')) && value[1] == ':' && (value[2] == '/' || value[2] == '\\') {
		return true
	}
	return strings.HasPrefix(value, "//")
}
