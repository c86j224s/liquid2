package ingest

import (
	"bytes"
	"net/url"
	"strings"

	"codeberg.org/readeck/go-readability/v2"
	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
)

type DefaultExtractor struct{}

func (extractor DefaultExtractor) Extract(pageURL string, contentType string, data []byte) (ExtractedContent, error) {
	mediaType := mediaType(contentType)
	body := string(data)
	switch mediaType {
	case "text/html", "application/xhtml+xml":
		return extractHTML(pageURL, body, data)
	case "text/markdown":
		content := strings.TrimSpace(body)
		if content == "" {
			return ExtractedContent{}, fetchFailed("markdown response was empty")
		}
		return ExtractedContent{Content: content, Format: FormatMarkdown}, nil
	case "text/plain", "":
		content := strings.TrimSpace(body)
		if content == "" {
			return ExtractedContent{}, fetchFailed("text response was empty")
		}
		return ExtractedContent{Content: content, Format: FormatText}, nil
	default:
		return ExtractedContent{}, unsupportedMedia("content type is not supported")
	}
}

func extractHTML(pageURL string, body string, data []byte) (ExtractedContent, error) {
	if content, ok := extractReadableMarkdown(pageURL, body, data); ok {
		return content, nil
	}
	if content, ok := extractSemanticMarkdown(pageURL, body); ok {
		return content, nil
	}
	title := extractTitle(body)
	content := stripHTML(body)
	if content == "" {
		return ExtractedContent{}, fetchFailed("html response had no readable text")
	}
	return ExtractedContent{Title: title, Content: content, Format: FormatText}, nil
}

func extractReadableMarkdown(pageURL string, body string, data []byte) (ExtractedContent, bool) {
	article, err := readability.FromReader(bytes.NewReader(data), readabilityURL(pageURL))
	if err != nil {
		return ExtractedContent{}, false
	}
	var articleHTML bytes.Buffer
	if err := article.RenderHTML(&articleHTML); err != nil {
		return ExtractedContent{}, false
	}
	markdown, err := htmltomarkdown.ConvertString(articleHTML.String())
	if err != nil {
		return ExtractedContent{}, false
	}
	markdown = strings.TrimSpace(markdown)
	if markdown == "" {
		return ExtractedContent{}, false
	}
	title := collapseText(article.Title())
	if title == "" {
		title = extractTitle(body)
	}
	return ExtractedContent{Title: title, Content: markdown, Format: FormatMarkdown}, true
}

func readabilityURL(pageURL string) *url.URL {
	if strings.TrimSpace(pageURL) == "" {
		return nil
	}
	parsed, err := url.Parse(pageURL)
	if err != nil {
		return nil
	}
	return parsed
}
