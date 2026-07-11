package config

import (
	"net"
	"strconv"
	"strings"
)

func productTable(values map[string]any, product string) map[string]any {
	table := namedTable(values, product)
	if len(table) == 0 {
		return values
	}
	return table
}

func namedTable(values map[string]any, name string) map[string]any {
	if table, ok := values[name].(map[string]any); ok {
		return table
	}
	return map[string]any{}
}

func combinedAddrValue(values map[string]any) (string, bool) {
	host, hasHost := configValue(values["addr"])
	port, hasPort := configValue(values["port"])
	if !hasHost && !hasPort {
		return "", false
	}
	host = strings.TrimSpace(host)
	port = strings.TrimSpace(port)
	if port == "" {
		return host, true
	}
	if host == "" {
		return ":" + port, true
	}
	if _, _, err := net.SplitHostPort(host); err == nil {
		return host, true
	}
	if strings.Contains(host, ":") {
		return net.JoinHostPort(host, port), true
	}
	return host + ":" + port, true
}

func configValue(raw any) (string, bool) {
	switch value := raw.(type) {
	case string:
		return strings.TrimSpace(value), strings.TrimSpace(value) != ""
	case bool:
		if value {
			return "1", true
		}
		return "0", true
	case int64:
		return strconv.FormatInt(value, 10), true
	case []any:
		parts := make([]string, 0, len(value))
		for _, item := range value {
			part, ok := configValue(item)
			if !ok {
				continue
			}
			parts = append(parts, part)
		}
		return strings.Join(parts, ","), len(parts) > 0
	default:
		return "", false
	}
}
