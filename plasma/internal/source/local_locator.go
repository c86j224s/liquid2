package source

import (
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

// ParseLocalPathLocator accepts a local path locator object or one-item array and normalizes it.
func ParseLocalPathLocator(raw json.RawMessage) (LocalPathLocator, error) {
	if len(raw) == 0 {
		return LocalPathLocator{}, fmt.Errorf("%w: local_path locator is required", producterror.ErrInvalidInput)
	}
	var locator LocalPathLocator
	if err := json.Unmarshal(raw, &locator); err == nil && locatorDiscriminator(locator.LocatorType, locator.Kind) != "" {
		return normalizeLocalPathLocator(locator)
	}
	var locators []LocalPathLocator
	if err := json.Unmarshal(raw, &locators); err != nil {
		return LocalPathLocator{}, fmt.Errorf("%w: local_path locator must be an object or one-item array", producterror.ErrInvalidInput)
	}
	if len(locators) != 1 {
		return LocalPathLocator{}, fmt.Errorf("%w: local_path locator must contain exactly one locator", producterror.ErrInvalidInput)
	}
	return normalizeLocalPathLocator(locators[0])
}

func normalizeLocalPathLocator(locator LocalPathLocator) (LocalPathLocator, error) {
	discriminator := locatorDiscriminator(locator.LocatorType, locator.Kind)
	locator.RootID = strings.TrimSpace(locator.RootID)
	locator.RelativePath = normalizePublicRelativePath(locator.RelativePath)
	locator.PathKind = strings.TrimSpace(locator.PathKind)
	if discriminator != LocatorTypeLocalPath {
		return LocalPathLocator{}, fmt.Errorf("%w: local_path locator kind is required", producterror.ErrInvalidInput)
	}
	locator.LocatorType = LocatorTypeLocalPath
	locator.Kind = ""
	if !validLocalPathRootID(locator.RootID) {
		return LocalPathLocator{}, fmt.Errorf("%w: invalid local path root id", producterror.ErrInvalidInput)
	}
	if err := validatePublicRelativePath(locator.RelativePath); err != nil {
		return LocalPathLocator{}, err
	}
	switch locator.PathKind {
	case "file", "directory":
	default:
		return LocalPathLocator{}, fmt.Errorf("%w: local_path path_kind must be file or directory", producterror.ErrInvalidInput)
	}
	return locator, nil
}

func locatorDiscriminator(locatorType string, legacyKind string) string {
	if trimmed := strings.TrimSpace(locatorType); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(legacyKind)
}

func normalizePublicRelativePath(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" {
		return "."
	}
	cleaned := path.Clean(value)
	if cleaned == "/" {
		return "."
	}
	return cleaned
}

func validatePublicRelativePath(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%w: relative path is required", producterror.ErrInvalidInput)
	}
	if strings.ContainsRune(value, 0) || containsControl(value) {
		return fmt.Errorf("%w: relative path contains control characters", producterror.ErrInvalidInput)
	}
	if filepath.IsAbs(value) || strings.HasPrefix(value, "/") || strings.HasPrefix(value, `\\`) || looksLikeWindowsAbs(value) {
		return fmt.Errorf("%w: relative path must not be absolute", producterror.ErrInvalidInput)
	}
	for _, part := range strings.Split(value, "/") {
		if part == ".." {
			return fmt.Errorf("%w: relative path must not contain traversal", producterror.ErrInvalidInput)
		}
	}
	return nil
}

func validLocalPathRootID(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func looksLikeWindowsAbs(value string) bool {
	if len(value) >= 3 && ((value[0] >= 'a' && value[0] <= 'z') || (value[0] >= 'A' && value[0] <= 'Z')) && value[1] == ':' && (value[2] == '/' || value[2] == '\\') {
		return true
	}
	return strings.HasPrefix(value, "//")
}

func containsControl(value string) bool {
	for _, r := range value {
		if r >= 0 && r < 0x20 {
			return true
		}
	}
	return false
}
