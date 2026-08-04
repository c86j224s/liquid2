package agentexec

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func (executor ClaudeExecutor) claudeSessionFile(sourceSessionID string) (string, error) {
	sourceSessionID = strings.TrimSpace(sourceSessionID)
	if sourceSessionID == "" {
		return "", fmt.Errorf("source session id is required")
	}
	claudeHome := claudeHomeFromEnv(executor.Env)
	if claudeHome == "" {
		return "", fmt.Errorf("Claude home is required for Claude session fork")
	}
	return findClaudeSessionFile(claudeHome, sourceSessionID)
}

func claudeHomeFromEnv(explicit []string) string {
	home := ""
	for _, item := range explicit {
		if strings.HasPrefix(item, "CLAUDE_CONFIG_DIR=") {
			if claudeHome := strings.TrimSpace(strings.TrimPrefix(item, "CLAUDE_CONFIG_DIR=")); claudeHome != "" {
				return claudeHome
			}
		}
		if strings.HasPrefix(item, "HOME=") {
			home = strings.TrimSpace(strings.TrimPrefix(item, "HOME="))
		}
	}
	if claudeHome := strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_DIR")); claudeHome != "" {
		return claudeHome
	}
	if home == "" {
		home = strings.TrimSpace(os.Getenv("HOME"))
	}
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".claude")
}

func findClaudeSessionFile(claudeHome string, sessionID string) (string, error) {
	projectsRoot := filepath.Join(claudeHome, "projects")
	var matches []string
	err := filepath.WalkDir(projectsRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Name() != sessionID+".jsonl" {
			return nil
		}
		matches = append(matches, path)
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("Claude session file not found for session %q under %s", sessionID, projectsRoot)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("Claude session file is ambiguous for session %q", sessionID)
	}
	return matches[0], nil
}

func checkWritableClaudeSessionDir(dir string) error {
	file, err := os.CreateTemp(dir, ".plasma-claude-fork-ready-*")
	if err != nil {
		return fmt.Errorf("Claude session directory is not writable: %w", err)
	}
	path := file.Name()
	closeErr := file.Close()
	removeErr := os.Remove(path)
	if closeErr != nil {
		return fmt.Errorf("Claude session readiness temp file close failed: %w", closeErr)
	}
	if removeErr != nil {
		return fmt.Errorf("Claude session readiness temp file cleanup failed: %w", removeErr)
	}
	return nil
}

func claudeDisallowedBuiltinTools() []string {
	return []string{
		"Bash",
		"Edit",
		"MultiEdit",
		"Write",
		"NotebookEdit",
		"Task",
		"TodoWrite",
	}
}
