package agentexec

import (
	"os"
	"regexp"
	"strings"
)

func claudeEnvironment(explicit []string) []string {
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
		"XDG_CONFIG_HOME",
		"XDG_DATA_HOME",
		"XDG_CACHE_HOME",
		"PLASMA_RUNTIME_MODE",
		"CLAUDE_CONFIG_DIR",
		"ANTHROPIC_API_KEY",
	}
	env := make([]string, 0, len(allowedKeys)+1)
	seenPath := false
	for _, key := range allowedKeys {
		value, ok := os.LookupEnv(key)
		if !ok {
			continue
		}
		if key == "PATH" {
			value = agentPATH(value)
			seenPath = true
		}
		env = append(env, key+"="+value)
	}
	if !seenPath {
		env = append(env, "PATH="+agentPATH(""))
	}
	env = append(env, "PLASMA_AGENT=1")
	return env
}

func claudeLog(stdout string, stderr string) string {
	if strings.TrimSpace(stderr) == "" {
		return stdout
	}
	if strings.TrimSpace(stdout) == "" {
		return stderr
	}
	return stdout + "\n" + stderr
}

func headTailExcerpt(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len(value) <= limit {
		return value
	}
	part := limit / 2
	if part < 1 {
		part = 1
	}
	return value[:part] + "\n...\n" + value[len(value)-part:]
}

var claudeSessionIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
