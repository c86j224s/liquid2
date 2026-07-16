package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func defaultReleaseDBDataDir(home, xdgDataHome string) string {
	return releaseDBDataDirForPlatform(runtime.GOOS, kernelRelease(), home, xdgDataHome)
}

func releaseDBDataDirForPlatform(goos, release, home, xdgDataHome string) string {
	if goos == "linux" && isWSL2KernelRelease(release) {
		if strings.TrimSpace(xdgDataHome) != "" {
			return filepath.Join(xdgDataHome, "plasma")
		}
		return filepath.Join(home, ".local", "share", "plasma")
	}
	return filepath.Join(home, "Library", "Application Support", "Plasma")
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
