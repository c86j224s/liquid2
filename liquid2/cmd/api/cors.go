package main

import "strings"

func corsOriginsFromEnv() []string {
	return splitCommaList(getenv("LIQUID2_CORS_ORIGINS", ""))
}

func splitCommaList(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}
