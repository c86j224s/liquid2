// Package reportilsource canonicalizes accepted report sources for bounded
// provider reads without exposing stored markup, locators, or binary bytes.
package reportilsource

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"net/netip"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/pdfdocument"
	"golang.org/x/net/html"
)

const MaxReadableBytes = 8 * 1024 * 1024

var ErrUnusableReadableText = errors.New("report IL source readable text is unusable")

type Readable struct {
	Text           string
	SHA256         string
	ByteSize       int
	Extraction     string
	DocumentTitle  string `json:"-"`
	PrimaryHeading string `json:"-"`
}

func Extract(content []byte, mediaType string) (Readable, error) {
	base, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		base = mediaType
	}
	base = strings.ToLower(strings.TrimSpace(base))
	var text, extraction, documentTitle, primaryHeading string
	switch {
	case base == "text/html" || base == "application/xhtml+xml":
		if !utf8.Valid(content) {
			return Readable{}, fmt.Errorf("%w: HTML is not UTF-8", ErrUnusableReadableText)
		}
		text, documentTitle, primaryHeading, err = extractHTML(content)
		extraction = "html_visible_text"
	case pdfdocument.IsPDFMediaType(base) || pdfdocument.IsPDFBytes(content):
		extracted, extractErr := pdfdocument.Extract(content)
		if extractErr != nil || extracted.Truncated {
			return Readable{}, fmt.Errorf("%w: PDF extraction failed or was truncated", ErrUnusableReadableText)
		}
		text = extracted.Text
		extraction = "pdf_text"
	case strings.HasPrefix(base, "text/") || base == "application/json" || base == "application/ld+json" || base == "application/xml" || strings.HasSuffix(base, "+json") || strings.HasSuffix(base, "+xml"):
		if !utf8.Valid(content) {
			return Readable{}, fmt.Errorf("%w: text is not UTF-8", ErrUnusableReadableText)
		}
		text = string(content)
		extraction = "stored_text"
	default:
		return Readable{}, fmt.Errorf("report IL source media type is unsupported")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return Readable{}, fmt.Errorf("%w: empty", ErrUnusableReadableText)
	}
	if !utf8.ValidString(text) {
		return Readable{}, fmt.Errorf("%w: invalid UTF-8", ErrUnusableReadableText)
	}
	text, err = normalizeExtractedText(text)
	if err != nil {
		return Readable{}, err
	}
	if len(text) > MaxReadableBytes {
		return Readable{}, fmt.Errorf("%w: size ceiling exceeded", ErrUnusableReadableText)
	}
	text, err = RedactProviderContent(text)
	if err != nil {
		return Readable{}, err
	}
	return Readable{
		Text: text, SHA256: sha256Hex([]byte(text)), ByteSize: len(text), Extraction: extraction,
		DocumentTitle: documentTitle, PrimaryHeading: primaryHeading,
	}, nil
}

func extractHTML(content []byte) (string, string, string, error) {
	document, err := html.Parse(strings.NewReader(string(content)))
	if err != nil {
		return "", "", "", fmt.Errorf("report IL source HTML parsing failed")
	}
	var output strings.Builder
	appendHTMLText(&output, document, false)
	lines := strings.Split(output.String(), "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.Join(strings.Fields(line), " ")
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}
	return strings.Join(cleaned, "\n"), firstHTMLElementText(document, "title"), firstHTMLElementText(document, "h1"), nil
}

func firstHTMLElementText(node *html.Node, element string) string {
	if node.Type == html.ElementNode && strings.EqualFold(node.Data, element) {
		var text strings.Builder
		appendHTMLElementText(&text, node)
		return strings.Join(strings.Fields(text.String()), " ")
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if value := firstHTMLElementText(child, element); value != "" {
			return value
		}
	}
	return ""
}

func appendHTMLElementText(output *strings.Builder, node *html.Node) {
	if node.Type == html.ElementNode {
		switch strings.ToLower(node.Data) {
		case "script", "style", "template", "svg", "canvas", "noscript":
			return
		}
	}
	if node.Type == html.TextNode {
		output.WriteString(node.Data)
		output.WriteByte(' ')
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		appendHTMLElementText(output, child)
	}
}

func appendHTMLText(output *strings.Builder, node *html.Node, hidden bool) {
	if node.Type == html.ElementNode {
		switch strings.ToLower(node.Data) {
		case "head", "script", "style", "template", "svg", "canvas", "noscript":
			hidden = true
		case "br", "p", "div", "article", "section", "header", "footer", "main", "aside", "nav", "blockquote", "pre", "tr", "h1", "h2", "h3", "h4", "h5", "h6":
			appendBoundary(output)
		case "li":
			appendBoundary(output)
			output.WriteString("- ")
		case "td", "th":
			if output.Len() > 0 {
				output.WriteString(" | ")
			}
		}
	}
	if node.Type == html.TextNode && !hidden {
		text := strings.Join(strings.Fields(node.Data), " ")
		if text != "" {
			if output.Len() > 0 {
				last := output.String()[output.Len()-1]
				if last != '\n' && last != ' ' {
					output.WriteByte(' ')
				}
			}
			output.WriteString(text)
		}
	}
	if !hidden {
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			appendHTMLText(output, child, false)
		}
	}
	if node.Type == html.ElementNode {
		switch strings.ToLower(node.Data) {
		case "p", "div", "article", "section", "header", "footer", "main", "aside", "nav", "blockquote", "pre", "li", "tr", "h1", "h2", "h3", "h4", "h5", "h6":
			appendBoundary(output)
		}
	}
}

func appendBoundary(output *strings.Builder) {
	if output.Len() == 0 {
		return
	}
	value := output.String()
	if value[len(value)-1] != '\n' {
		output.WriteByte('\n')
	}
}

var (
	credentialPattern = regexp.MustCompile(`(?i)(["']?(?:authorization|api[_-]?key|access[_-]?token|refresh[_-]?token|client[_-]?secret|cookie|session[_-]?token)["']?\s*[:=]|bearer\s+[A-Za-z0-9._~+/=-]{8,})`)
	urlPattern        = regexp.MustCompile(`(?i)[a-z][a-z0-9+.-]*://[^\s<>\"']+`)
	escapedURLPattern = regexp.MustCompile(`(?i)[a-z][a-z0-9+.-]*:(?:\\/){2}[^\s<>\"']+`)
)

func normalizeExtractedText(content string) (string, error) {
	var normalized strings.Builder
	normalized.Grow(len(content))
	changed := false
	for _, value := range content {
		if value == unicode.ReplacementChar {
			return "", fmt.Errorf("%w: replacement characters", ErrUnusableReadableText)
		}
		if unicode.IsControl(value) && value != '\n' && value != '\r' && value != '\t' {
			normalized.WriteByte(' ')
			changed = true
			continue
		}
		normalized.WriteRune(value)
	}
	if !changed {
		return content, nil
	}
	content = strings.TrimSpace(normalized.String())
	if content == "" {
		return "", fmt.Errorf("%w: empty after normalization", ErrUnusableReadableText)
	}
	return content, nil
}

const (
	redactedCredential = "[redacted credential]"
	redactedURL        = "[redacted private URL]"
)

func RedactProviderContent(content string) (string, error) {
	content = redactEscapedURLs(content)
	content = redactCredentialSpans(content)
	matches := urlPattern.FindAllStringIndex(content, -1)
	if len(matches) > 0 {
		var redacted strings.Builder
		start := 0
		for _, match := range matches {
			value := content[match[0]:match[1]]
			trimmed := strings.TrimRight(value, `.,;:!?)]}`)
			if !restrictedProviderURL(trimmed) {
				continue
			}
			redacted.WriteString(content[start:match[0]])
			redacted.WriteString(redactedURL)
			redacted.WriteString(value[len(trimmed):])
			start = match[1]
		}
		if start > 0 {
			redacted.WriteString(content[start:])
			content = redacted.String()
		}
	}
	if err := ValidateProviderContent(content); err != nil {
		return "", err
	}
	return content, nil
}

func redactEscapedURLs(content string) string {
	return escapedURLPattern.ReplaceAllStringFunc(content, func(value string) string {
		plain := strings.ReplaceAll(value, `\/`, "/")
		if restrictedProviderURL(strings.TrimRight(plain, `.,;:!?)]}`)) {
			return redactedURL
		}
		return value
	})
}

func redactCredentialSpans(content string) string {
	matches := credentialPattern.FindAllStringIndex(content, -1)
	if len(matches) == 0 {
		return content
	}
	var redacted strings.Builder
	start := 0
	for _, match := range matches {
		if match[0] < start {
			continue
		}
		end := match[1]
		if end < len(content) && content[end] != '\n' && content[end] != '\r' {
			if relative := strings.IndexAny(content[end:], "\r\n"); relative >= 0 {
				end += relative
			} else {
				end = len(content)
			}
		}
		redacted.WriteString(content[start:match[0]])
		redacted.WriteString(redactedCredential)
		start = end
	}
	redacted.WriteString(content[start:])
	return redacted.String()
}

func ValidateProviderContent(content string) error {
	if credentialPattern.MatchString(content) {
		return fmt.Errorf("report IL source contains restricted credential material")
	}
	for _, value := range urlPattern.FindAllString(strings.ReplaceAll(content, `\/`, "/"), -1) {
		if restrictedProviderURL(strings.TrimRight(value, `.,;:!?)]}`)) {
			return fmt.Errorf("report IL source contains restricted URL material")
		}
	}
	for _, value := range escapedURLPattern.FindAllString(content, -1) {
		plain := strings.ReplaceAll(value, `\/`, "/")
		if restrictedProviderURL(strings.TrimRight(plain, `.,;:!?)]}`)) {
			return fmt.Errorf("report IL source contains restricted URL material")
		}
	}
	return nil
}

func restrictedProviderURL(value string) bool {
	u, err := url.Parse(value)
	if err != nil || u.Scheme == "" {
		return false
	}
	if strings.EqualFold(u.Scheme, "file") || u.User != nil {
		return true
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".ts.net") {
		return true
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		ip = ip.Unmap()
		return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || netip.MustParsePrefix("100.64.0.0/10").Contains(ip)
	}
	return false
}

func sha256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
