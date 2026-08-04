package agentexec

import (
	"regexp"
	"strconv"
	"strings"
)

func effectiveMCPEnabledTools(base []string, extra []string, replace bool) []string {
	seen := map[string]struct{}{}
	out := []string{}
	tools := append([]string(nil), extra...)
	if !replace {
		tools = append(append([]string{}, base...), extra...)
	}
	for _, tool := range tools {
		tool = strings.TrimSpace(tool)
		if tool == "" {
			continue
		}
		if _, ok := seen[tool]; ok {
			continue
		}
		seen[tool] = struct{}{}
		out = append(out, tool)
	}
	return out
}

func hasReportPatchTool(tools []string) bool {
	for _, tool := range tools {
		if strings.HasPrefix(strings.TrimSpace(tool), "plasma.report.patch.") {
			return true
		}
	}
	return false
}

func sanitizeMCPServerName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "plasma"
	}
	if regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(name) {
		return name
	}
	return "plasma"
}

func tomlString(value string) string {
	return strconv.Quote(value)
}

func tomlStringArray(values []string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, tomlString(value))
	}
	return "[" + strings.Join(parts, ",") + "]"
}
