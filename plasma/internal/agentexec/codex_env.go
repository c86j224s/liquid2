package agentexec

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
)

func resolveAgentCommand(command string) string {
	if strings.ContainsRune(command, os.PathSeparator) {
		return command
	}
	if resolved, err := exec.LookPath(command); err == nil {
		return resolved
	}
	for _, dir := range []string{"/opt/homebrew/bin", "/usr/local/bin"} {
		candidate := filepath.Join(dir, command)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return candidate
		}
	}
	return command
}

// ResolveAgentCommand resolves common provider binary locations for compatibility tests.
func ResolveAgentCommand(command string) string {
	return resolveAgentCommand(command)
}

func codexEnvironment(explicit []string) []string {
	if len(explicit) > 0 {
		return append([]string(nil), explicit...)
	}
	allowedKeys := []string{
		"HOME",
		"PATH",
		"SHELL",
		"TERM",
		"USER",
		"LOGNAME",
		"TMPDIR",
		"LANG",
		"LC_ALL",
		"LC_CTYPE",
		"CODEX_HOME",
		"PLASMA_RUNTIME_MODE",
		"XDG_CONFIG_HOME",
		"XDG_DATA_HOME",
		"XDG_CACHE_HOME",
	}
	env := make([]string, 0, len(allowedKeys)+1)
	seen := map[string]struct{}{}
	for _, key := range allowedKeys {
		value, ok := os.LookupEnv(key)
		if !ok {
			continue
		}
		env = append(env, key+"="+value)
		seen[key] = struct{}{}
	}
	if _, ok := seen["PATH"]; !ok {
		env = append(env, "PATH="+agentPATH(""))
	} else {
		for i, value := range env {
			if strings.HasPrefix(value, "PATH=") {
				env[i] = "PATH=" + agentPATH(strings.TrimPrefix(value, "PATH="))
				break
			}
		}
	}
	env = append(env, "PLASMA_AGENT=1")
	return env
}

// CodexEnvironment returns the sanitized Codex process environment.
func CodexEnvironment(explicit []string) []string {
	return codexEnvironment(explicit)
}

func agentPATH(current string) string {
	values := []string{}
	addPath := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		for _, existing := range values {
			if existing == value {
				return
			}
		}
		values = append(values, value)
	}
	for _, value := range []string{"/opt/homebrew/bin", "/usr/local/bin"} {
		addPath(value)
	}
	for _, value := range filepath.SplitList(current) {
		addPath(value)
	}
	for _, value := range []string{"/usr/bin", "/bin", "/usr/sbin", "/sbin"} {
		addPath(value)
	}
	return strings.Join(values, string(os.PathListSeparator))
}

// AgentPATH returns the provider PATH with common Homebrew locations first and duplicates removed.
func AgentPATH(current string) string {
	return agentPATH(current)
}

func codexSessionID(log string) string {
	return agentusage.ParseCodexSessionID(log)
}

func codexUsageFromLog(usage agentusage.AgentUsage, log string, previousSessionID string, sessionID string, resumed bool, compaction bool) agentusage.AgentUsage {
	if providerUsage, ok := agentusage.ParseCodexProviderUsage(log); ok {
		usage = usage.WithProviderUsage(providerUsage, "codex_jsonl_turn_completed")
	} else {
		usage = usage.WithUnavailable("codex JSONL did not include turn.completed usage")
	}
	return usage.WithSession(previousSessionID, sessionID, resumed, compaction)
}

func codexUsageExecutorName(executorName string) string {
	executorName = strings.TrimSpace(executorName)
	if executorName == "" {
		return "codex"
	}
	return executorName
}
