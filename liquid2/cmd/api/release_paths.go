package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func defaultReleasePaths(home, xdgDataHome string) (string, string, string) {
	return releasePathsForPlatform(runtime.GOOS, kernelRelease(), home, xdgDataHome)
}

func releasePathsForPlatform(goos, release, home, xdgDataHome string) (string, string, string) {
	dataDir := releaseDataDirForPlatform(goos, release, home, xdgDataHome)
	return filepath.Join(dataDir, "liquid2.db"), filepath.Join(dataDir, "exports"), filepath.Join(dataDir, "backups")
}

func releaseDataDirForPlatform(goos, release, home, xdgDataHome string) string {
	if goos == "linux" && isWSL2KernelRelease(release) {
		if strings.TrimSpace(xdgDataHome) != "" {
			return filepath.Join(xdgDataHome, "liquid2")
		}
		return filepath.Join(home, ".local", "share", "liquid2")
	}
	return filepath.Join(home, "Library", "Application Support", "Liquid2")
}

func kernelRelease() string {
	value, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return ""
	}
	return string(value)
}

func isWSL2KernelRelease(release string) bool {
	value := strings.ToLower(release)
	return strings.Contains(value, "microsoft") && strings.Contains(value, "standard") && strings.Contains(value, "wsl2")
}
