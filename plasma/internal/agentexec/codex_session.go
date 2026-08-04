package agentexec

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CheckForkSession은 Codex session을 실제로 복제하기 전에 fork 가능성을 점검한다.
func (executor CodexExecutor) CheckForkSession(_ context.Context, sourceSessionID string) error {
	sessionFile, _, err := executor.codexSessionFileContent(sourceSessionID)
	if err != nil {
		return err
	}
	checkFile, err := os.CreateTemp(filepath.Dir(sessionFile), ".plasma-fork-check-*")
	if err != nil {
		return fmt.Errorf("Codex session directory is not writable: %w", err)
	}
	checkName := checkFile.Name()
	closeErr := checkFile.Close()
	removeErr := os.Remove(checkName)
	if closeErr != nil {
		return closeErr
	}
	if removeErr != nil {
		return removeErr
	}
	return nil
}

func ensureAgentWorkDir(workDir string) error {
	workDir = strings.TrimSpace(workDir)
	if workDir == "" {
		return nil
	}
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		return fmt.Errorf("ensure agent workdir %q: %w", workDir, err)
	}
	return nil
}

// ForkSession은 기존 Codex session을 report patch 전용 session으로 복제한다.
func (executor CodexExecutor) ForkSession(_ context.Context, sourceSessionID string) (AgentSessionForkResult, error) {
	sessionFile, content, err := executor.codexSessionFileContent(sourceSessionID)
	if err != nil {
		return AgentSessionForkResult{}, err
	}
	sourceSessionID = strings.TrimSpace(sourceSessionID)
	cloneID, err := newUUID()
	if err != nil {
		return AgentSessionForkResult{}, err
	}
	cloneContent := bytes.ReplaceAll(content, []byte(sourceSessionID), []byte(cloneID))
	cloneFile := filepath.Join(filepath.Dir(sessionFile), fmt.Sprintf("rollout-%s-%s.jsonl", time.Now().UTC().Format("2006-01-02T15-04-05"), cloneID))
	if err := os.WriteFile(cloneFile, cloneContent, 0o600); err != nil {
		return AgentSessionForkResult{}, err
	}
	sourceInfo, err := os.Stat(sessionFile)
	if err != nil {
		return AgentSessionForkResult{}, err
	}
	cloneInfo, err := os.Stat(cloneFile)
	if err != nil {
		return AgentSessionForkResult{}, err
	}
	return AgentSessionForkResult{
		SessionID:       cloneID,
		SourceSessionID: sourceSessionID,
		SourceHash:      sha256Hex(content),
		CloneHash:       sha256Hex(cloneContent),
		SourceSizeBytes: sourceInfo.Size(),
		CloneSizeBytes:  cloneInfo.Size(),
	}, nil
}

func (executor CodexExecutor) codexSessionFileContent(sourceSessionID string) (string, []byte, error) {
	sourceSessionID = strings.TrimSpace(sourceSessionID)
	if sourceSessionID == "" {
		return "", nil, fmt.Errorf("source session id is required")
	}
	codexHome := codexHomeFromEnv(executor.Env)
	if codexHome == "" {
		return "", nil, fmt.Errorf("Codex home is required for Codex session fork")
	}
	sessionFile, err := findCodexSessionFile(codexHome, sourceSessionID)
	if err != nil {
		return "", nil, err
	}
	content, err := os.ReadFile(sessionFile)
	if err != nil {
		return "", nil, err
	}
	if !bytes.Contains(content, []byte(sourceSessionID)) {
		return "", nil, fmt.Errorf("source session id %q was not present in Codex session file", sourceSessionID)
	}
	return sessionFile, content, nil
}

func codexHomeFromEnv(explicit []string) string {
	home := ""
	for _, item := range explicit {
		if strings.HasPrefix(item, "CODEX_HOME=") {
			if codexHome := strings.TrimSpace(strings.TrimPrefix(item, "CODEX_HOME=")); codexHome != "" {
				return codexHome
			}
		}
		if strings.HasPrefix(item, "HOME=") {
			home = strings.TrimSpace(strings.TrimPrefix(item, "HOME="))
		}
	}
	if codexHome := strings.TrimSpace(os.Getenv("CODEX_HOME")); codexHome != "" {
		return codexHome
	}
	if home == "" {
		home = strings.TrimSpace(os.Getenv("HOME"))
	}
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".codex")
}

func findCodexSessionFile(codexHome string, sessionID string) (string, error) {
	sessionsRoot := filepath.Join(codexHome, "sessions")
	var matches []string
	err := filepath.WalkDir(sessionsRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			return nil
		}
		if strings.Contains(entry.Name(), sessionID) {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("Codex session file not found for session %q under %s", sessionID, sessionsRoot)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("Codex session file is ambiguous for session %q", sessionID)
	}
	return matches[0], nil
}

func newUUID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		bytes[0:4],
		bytes[4:6],
		bytes[6:8],
		bytes[8:10],
		bytes[10:16],
	), nil
}
