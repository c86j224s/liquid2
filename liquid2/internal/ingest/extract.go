package ingest

import (
	"html"
	"mime"
	"regexp"
	"strings"
)

const (
	FormatHTML     = "html"
	FormatMarkdown = "markdown"
	FormatText     = "text"
)

var (
	scriptStylePattern = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	tagPattern         = regexp.MustCompile(`(?is)<[^>]+>`)
	titlePattern       = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	spacePattern       = regexp.MustCompile(`\s+`)
)

type ExtractedContent struct {
	Title   string
	Content string
	Format  string
}

func Extract(contentType string, data []byte) (ExtractedContent, error) {
	return ExtractWithURL("", contentType, data)
}

func ExtractWithURL(pageURL string, contentType string, data []byte) (ExtractedContent, error) {
	return DefaultExtractor{}.Extract(pageURL, contentType, data)
}

func mediaType(contentType string) string {
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		return ""
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return strings.ToLower(contentType)
	}
	return strings.ToLower(mediaType)
}

func extractTitle(body string) string {
	match := titlePattern.FindStringSubmatch(body)
	if len(match) < 2 {
		return ""
	}
	return collapseText(match[1])
}

func stripHTML(body string) string {
	body = scriptStylePattern.ReplaceAllString(body, " ")
	body = tagPattern.ReplaceAllString(body, " ")
	return collapseText(body)
}

func collapseText(value string) string {
	value = html.UnescapeString(value)
	value = spacePattern.ReplaceAllString(value, " ")
	return strings.TrimSpace(value)
}
