package ingest

import (
	"strings"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/PuerkitoBio/goquery"
)

var semanticContentSelectors = []string{
	"article",
	"main article",
	"[role=main] article",
	"main",
	"[role=main]",
	".post-content",
	".entry-content",
	".article-content",
	".content",
}

func extractSemanticMarkdown(pageURL string, body string) (ExtractedContent, bool) {
	document, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return ExtractedContent{}, false
	}
	title := extractTitle(body)
	best := ExtractedContent{}
	bestScore := 0
	for _, selector := range semanticContentSelectors {
		document.Find(selector).Each(func(_ int, selection *goquery.Selection) {
			candidate, ok := markdownCandidate(pageURL, title, selection)
			if !ok {
				return
			}
			score := markdownScore(candidate.Content)
			if score > bestScore {
				best = candidate
				bestScore = score
			}
		})
		if bestScore > 0 && selector == "article" {
			break
		}
	}
	return best, bestScore > 0
}

func markdownCandidate(
	pageURL string,
	title string,
	selection *goquery.Selection,
) (ExtractedContent, bool) {
	html, ok := cleanedSelectionHTML(selection)
	if !ok {
		return ExtractedContent{}, false
	}
	markdown, err := htmltomarkdown.ConvertString(
		html,
		converter.WithDomain(pageURL),
	)
	if err != nil {
		return ExtractedContent{}, false
	}
	markdown = normalizeMarkdown(markdown)
	if !looksReadableMarkdown(markdown) {
		return ExtractedContent{}, false
	}
	return ExtractedContent{
		Title: title, Content: markdown, Format: FormatMarkdown,
	}, true
}

func cleanedSelectionHTML(selection *goquery.Selection) (string, bool) {
	clone := selection.Clone()
	clone.Find("script,style,noscript,svg,form,button,dialog,nav,aside,footer").Remove()
	if strings.TrimSpace(clone.Text()) == "" {
		return "", false
	}
	html, err := goquery.OuterHtml(clone)
	if err != nil {
		return "", false
	}
	return html, true
}

func normalizeMarkdown(markdown string) string {
	markdown = strings.ReplaceAll(markdown, "\r\n", "\n")
	markdown = strings.ReplaceAll(markdown, "\r", "\n")
	lines := strings.Split(markdown, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func looksReadableMarkdown(markdown string) bool {
	text := collapseText(markdown)
	if len([]rune(text)) < 80 {
		return false
	}
	return strings.Count(markdown, "\n") >= 2 ||
		strings.Contains(markdown, "# ") ||
		strings.Contains(markdown, "## ") ||
		strings.Contains(markdown, "[")
}

func markdownScore(markdown string) int {
	textLength := len([]rune(collapseText(markdown)))
	return textLength + strings.Count(markdown, "\n")*4
}
