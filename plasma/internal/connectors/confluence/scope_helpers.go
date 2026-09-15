package confluence

import "strings"

func normalizeScopes(scopes []string) []string {
	seen := map[string]struct{}{}
	var normalized []string
	for _, scope := range scopes {
		for _, field := range strings.Fields(strings.ReplaceAll(scope, ",", " ")) {
			field = strings.TrimSpace(field)
			if field == "" {
				continue
			}
			if _, ok := seen[field]; ok {
				continue
			}
			seen[field] = struct{}{}
			normalized = append(normalized, field)
		}
	}
	return normalized
}

func normalizeScopeString(scope string) []string {
	return normalizeScopes([]string{scope})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
