package web

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

type resolverFileInfo struct {
	mode os.FileMode
}

func (i resolverFileInfo) Name() string       { return "chrome" }
func (i resolverFileInfo) Size() int64        { return 1 }
func (i resolverFileInfo) Mode() os.FileMode  { return i.mode }
func (i resolverFileInfo) ModTime() time.Time { return time.Time{} }
func (i resolverFileInfo) IsDir() bool        { return i.mode.IsDir() }
func (i resolverFileInfo) Sys() any           { return nil }

func TestResolveChromePathConfiguredFailuresDoNotFallback(t *testing.T) {
	for _, tc := range []struct {
		name     string
		explicit string
		env      string
		info     os.FileInfo
		statErr  error
	}{
		{name: "missing explicit", explicit: "/private/explicit-missing", statErr: errors.New("missing")},
		{name: "directory explicit", explicit: "/private/explicit-directory", info: resolverFileInfo{mode: os.ModeDir | 0o755}},
		{name: "non-executable explicit", explicit: "/private/explicit-nonexec", info: resolverFileInfo{mode: 0o644}},
		{name: "missing environment", env: "/private/env-missing", statErr: errors.New("missing")},
		{name: "directory environment", env: "/private/env-directory", info: resolverFileInfo{mode: os.ModeDir | 0o755}},
		{name: "non-executable environment", env: "/private/env-nonexec", info: resolverFileInfo{mode: 0o644}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lookCalls := 0
			look := func(string) (string, error) {
				lookCalls++
				return "/discovered/chrome", nil
			}
			stat := func(string) (os.FileInfo, error) { return tc.info, tc.statErr }
			_, err := resolveChromePath(tc.explicit, tc.env, look, stat, "linux")
			if err == nil {
				t.Fatal("invalid configured Chrome path unexpectedly succeeded")
			}
			if lookCalls != 0 {
				t.Fatalf("invalid configured Chrome path fell back to discovery: calls=%d", lookCalls)
			}
			for _, privatePath := range []string{tc.explicit, tc.env} {
				if privatePath != "" && strings.Contains(err.Error(), privatePath) {
					t.Fatalf("resolver error exposed configured path: %v", err)
				}
			}
		})
	}
}

func TestResolveChromePathPrecedenceAndValidation(t *testing.T) {
	valid := func(path string) (os.FileInfo, error) {
		return resolverFileInfo{mode: 0o755}, nil
	}
	invalid := func(path string) (os.FileInfo, error) {
		return nil, errors.New("missing")
	}
	look := func(name string) (string, error) {
		return "/discovered/" + name, nil
	}
	if got, err := resolveChromePath("/explicit/chrome", "/env/chrome", look, valid, "linux"); err != nil || got != "/explicit/chrome" {
		t.Fatalf("valid explicit = %q, %v", got, err)
	}
	if _, err := resolveChromePath("/explicit/missing", "/env/chrome", look, invalid, "linux"); err == nil || strings.Contains(err.Error(), "/explicit/missing") {
		t.Fatalf("invalid explicit fallback/error = %v", err)
	}
	if got, err := resolveChromePath("", "/env/chrome", look, valid, "linux"); err != nil || got != "/env/chrome" {
		t.Fatalf("valid env = %q, %v", got, err)
	}
	if _, err := resolveChromePath("", "/env/missing", look, invalid, "linux"); err == nil || strings.Contains(err.Error(), "/env/missing") {
		t.Fatalf("invalid env fallback/error = %v", err)
	}
	if got, err := resolveChromePath("", "", look, valid, "linux"); err != nil || got != "/discovered/google-chrome" {
		t.Fatalf("discovery = %q, %v", got, err)
	}
	if _, err := resolveChromePath("", "", func(string) (string, error) { return "", errors.New("not found") }, invalid, "linux"); err == nil {
		t.Fatal("missing discovery unexpectedly succeeded")
	}
}
