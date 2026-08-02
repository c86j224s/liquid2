package browserrender

import (
	"fmt"
	htmlpkg "html"
	"regexp"
	"strings"
	"time"
)

const minReadableTextLength = 400

var (
	titlePattern    = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	headingPattern  = regexp.MustCompile(`(?is)<h1[^>]*>(.*?)</h1>`)
	mainPattern     = regexp.MustCompile(`(?is)<main\b[^>]*>.*?</main>`)
	articlePattern  = regexp.MustCompile(`(?is)<article\b[^>]*>.*?</article>`)
	bodyPattern     = regexp.MustCompile(`(?is)<body\b[^>]*>(.*?)</body>`)
	commentPattern  = regexp.MustCompile(`(?is)<!--.*?-->`)
	tagPattern      = regexp.MustCompile(`(?is)<[^>]+>`)
	spacePattern    = regexp.MustCompile(`\s+`)
	dropTagPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script>`),
		regexp.MustCompile(`(?is)<style\b[^>]*>.*?</style>`),
		regexp.MustCompile(`(?is)<svg\b[^>]*>.*?</svg>`),
		regexp.MustCompile(`(?is)<template\b[^>]*>.*?</template>`),
		regexp.MustCompile(`(?is)<noscript\b[^>]*>.*?</noscript>`),
	}
)

func readableDocument(dom string, finalURL string, renderedAt time.Time) (Result, error) {
	title := firstReadableText(headingPattern.FindStringSubmatch(dom), titlePattern.FindStringSubmatch(dom))
	fragment := readableFragment(dom)
	cleaned := cleanHTMLFragment(fragment)
	textLength := len([]rune(visibleText(cleaned)))
	if textLength < minReadableTextLength {
		return Result{}, ErrNoReadableBody
	}
	content := []byte(fmt.Sprintf(`<!doctype html>
<html>
<head>
<meta charset="utf-8">
<title>%s</title>
</head>
<body data-plasma-retrieval-method="browser_render">
%s
</body>
</html>
`, htmlpkg.EscapeString(title), cleaned))
	return Result{
		Content:    content,
		MediaType:  MediaTypeHTML,
		Title:      title,
		FinalURL:   strings.TrimSpace(finalURL),
		RenderedAt: renderedAt,
		TextLength: textLength,
	}, nil
}

func readableFragment(dom string) string {
	if main := strings.TrimSpace(mainPattern.FindString(dom)); main != "" {
		return main
	}
	if article := strings.TrimSpace(articlePattern.FindString(dom)); article != "" {
		return article
	}
	if body := bodyPattern.FindStringSubmatch(dom); len(body) > 1 {
		return body[1]
	}
	return dom
}

func cleanHTMLFragment(fragment string) string {
	fragment = commentPattern.ReplaceAllString(fragment, " ")
	for _, pattern := range dropTagPatterns {
		fragment = pattern.ReplaceAllString(fragment, " ")
	}
	return strings.TrimSpace(fragment)
}

func visibleText(fragment string) string {
	text := tagPattern.ReplaceAllString(fragment, " ")
	text = htmlpkg.UnescapeString(text)
	text = spacePattern.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func firstReadableText(matches ...[]string) string {
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		text := visibleText(match[1])
		if text != "" {
			return text
		}
	}
	return "Browser rendered source"
}
