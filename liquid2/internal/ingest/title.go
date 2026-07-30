package ingest

import (
	"net/url"
	"path"
	"strconv"
	"strings"
)

func titleFromURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" {
		return "Untitled document"
	}
	host := displayHost(parsed.Hostname())
	title := titleFromPath(parsed.EscapedPath())
	if title == "" {
		return host
	}
	return title
}

func titleFromPath(rawPath string) string {
	rawPath = strings.Trim(strings.TrimSpace(rawPath), "/")
	if rawPath == "" {
		return ""
	}
	parts := strings.Split(rawPath, "/")
	title, err := url.PathUnescape(parts[len(parts)-1])
	if err != nil {
		title = parts[len(parts)-1]
	}
	title = strings.TrimSuffix(title, path.Ext(title))
	title = strings.NewReplacer("-", " ", "_", " ", "+", " ").Replace(title)
	title = strings.Join(strings.Fields(title), " ")
	if title == "" || len([]rune(title)) < 3 {
		return ""
	}
	switch strings.ToLower(title) {
	case "index", "home", "default":
		return ""
	}
	if _, err := strconv.Atoi(title); err == nil {
		return ""
	}
	return title
}

func displayHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.TrimPrefix(host, "www.")
	if host == "" {
		return "Untitled document"
	}
	return host
}
