package config

import (
	"net"
	"sort"
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

func stringValue(raw any) string {
	value, _ := configValue(raw)
	return value
}

func listValue(raw any) (string, bool) {
	values := configList(raw)
	if len(values) == 0 {
		return "", false
	}
	return strings.Join(values, ","), true
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
		return listValue(value)
	default:
		return "", false
	}
}

func configList(raw any) []string {
	switch value := raw.(type) {
	case string:
		return splitList(value)
	case []any:
		values := make([]string, 0, len(value))
		for _, item := range value {
			if text, ok := configValue(item); ok {
				values = append(values, text)
			}
		}
		return splitList(strings.Join(values, ","))
	case map[string]any:
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		values := make([]string, 0, len(keys))
		for _, key := range keys {
			rootID := strings.TrimSpace(key)
			rootPath, ok := configValue(value[key])
			if rootID == "" || !ok || strings.TrimSpace(rootPath) == "" {
				continue
			}
			values = append(values, rootID+"="+strings.TrimSpace(rootPath))
		}
		return values
	default:
		return nil
	}
}

func splitList(value string) []string {
	fields := strings.Split(value, ",")
	values := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			values = append(values, field)
		}
	}
	return values
}
