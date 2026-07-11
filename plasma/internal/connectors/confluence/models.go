package confluence

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func (client *Client) candidate(
	item confluenceSearchResult,
	baseURL string,
) app.ConfluenceSourceCandidate {
	content := item.Content
	pageID := strings.TrimSpace(content.ID)
	space := content.Space
	if space.Key == "" && len(item.Space.ID) > 0 {
		space = item.Space
	}
	updatedAt := parseConfluenceTime(content.Version.When)
	if updatedAt.IsZero() {
		updatedAt = parseConfluenceTime(item.LastModified)
	}
	return app.ConfluenceSourceCandidate{
		Connector: app.ConnectorRef{
			ConnectorID:      app.ConfluenceConnectorID,
			ConnectorType:    app.ConfluenceConnectorType,
			ExternalSourceID: app.ConfluenceExternalSourceID(client.cloudID, pageID),
			ExternalURI:      app.ConfluenceExternalURI(client.cloudID, pageID),
			ExternalVersion:  confluenceExternalVersion(content.Version.Number, content.Version.When),
			ConnectorVersion: client.connectorVersion,
		},
		CloudID:     client.cloudID,
		SiteURL:     client.siteURLString(),
		SpaceID:     rawIDString(space.ID),
		SpaceKey:    space.Key,
		Title:       firstNonBlank(content.Title, item.Title),
		SourceURI:   client.absoluteURL(baseURL, item.URL, content.Links.WebUI),
		Version:     content.Version.Number,
		UpdatedAt:   updatedAt,
		CanSnapshot: pageID != "",
	}
}

func (client *Client) siteURLString() string {
	if client.siteURL != nil {
		return client.siteURL.String()
	}
	return ""
}

func (client *Client) absoluteURL(baseURL string, values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		parsed, err := url.Parse(value)
		if err == nil && parsed.Scheme != "" && parsed.Host != "" {
			return parsed.String()
		}
		if joined := joinBasePath(baseURL, value); joined != "" {
			return joined
		}
		if client.siteURL != nil {
			if joined := joinBasePath(client.siteURL.String(), value); joined != "" {
				return joined
			}
		}
	}
	return ""
}

func joinBasePath(base string, path string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	path = strings.TrimSpace(path)
	if base == "" || path == "" {
		return ""
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	if strings.HasPrefix(path, "/") {
		parsed.Path = strings.TrimRight(parsed.Path, "/") + path
	} else {
		parsed.Path = strings.TrimRight(parsed.Path, "/") + "/" + path
	}
	return parsed.String()
}

func confluenceCQL(query string, spaceKey string) string {
	parts := []string{"type=page"}
	if trimmed := strings.TrimSpace(spaceKey); trimmed != "" {
		parts = append(parts, "space = "+cqlQuoted(trimmed))
	}
	if trimmed := strings.TrimSpace(query); trimmed != "" {
		parts = append(parts, "text ~ "+cqlQuoted(trimmed))
	}
	return strings.Join(parts, " and ")
}

func cqlQuoted(value string) string {
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value)
	return `"` + escaped + `"`
}

func cursorFromNextLink(value string) string {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return ""
	}
	return parsed.Query().Get("cursor")
}

func parseConfluenceTime(value string) time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.000-0700"} {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}

func confluenceExternalVersion(number int, timestamp string) string {
	if number > 0 {
		return strconv.Itoa(number)
	}
	return strings.TrimSpace(timestamp)
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func rawIDString(raw json.RawMessage) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return ""
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return strings.TrimSpace(text)
	}
	return trimmed
}
