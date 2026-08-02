package sourcediagnostics

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"unicode/utf8"

	htmltoken "golang.org/x/net/html"
)

type browserRenderHTMLStats struct {
	VisibleTextLength int
	ScriptCount       int
	ExternalScripts   int
	HasAppMount       bool
	HasHydration      bool
	HasJSRequiredText bool
	HasBotWall        bool
}

func inspectBrowserRenderHTML(content []byte) browserRenderHTMLStats {
	stats := browserRenderHTMLStats{}
	lowerRaw := strings.ToLower(string(content))
	stats.HasBotWall = browserRenderCandidateLooksLikeBotWall(lowerRaw)
	stats.HasJSRequiredText = strings.Contains(lowerRaw, "please enable javascript") ||
		strings.Contains(lowerRaw, "enable javascript") ||
		strings.Contains(lowerRaw, "requires javascript")
	stats.HasHydration = strings.Contains(lowerRaw, "__next_data__") ||
		strings.Contains(lowerRaw, "data-reactroot") ||
		strings.Contains(lowerRaw, "window.__initial_state__") ||
		strings.Contains(lowerRaw, "ng-version") ||
		strings.Contains(lowerRaw, "data-v-app")

	tokenizer := htmltoken.NewTokenizer(bytes.NewReader(content))
	hiddenDepth := 0
	textParts := make([]string, 0, 16)
	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case htmltoken.ErrorToken:
			stats.VisibleTextLength = browserRenderVisibleTextLength(textParts)
			if errors.Is(tokenizer.Err(), io.EOF) {
				return stats
			}
			return stats
		case htmltoken.StartTagToken, htmltoken.SelfClosingTagToken:
			token := tokenizer.Token()
			tag := strings.ToLower(token.Data)
			if tag == "script" {
				stats.ScriptCount++
				if browserRenderTokenAttr(token, "src") != "" {
					stats.ExternalScripts++
				}
			}
			if browserRenderTagSuggestsAppMount(token) {
				stats.HasAppMount = true
			}
			if browserRenderTagSuggestsHydration(token) {
				stats.HasHydration = true
			}
			if tokenType == htmltoken.StartTagToken && browserRenderHiddenTextTag(tag) {
				hiddenDepth++
			}
		case htmltoken.EndTagToken:
			token := tokenizer.Token()
			if browserRenderHiddenTextTag(strings.ToLower(token.Data)) && hiddenDepth > 0 {
				hiddenDepth--
			}
		case htmltoken.TextToken:
			if hiddenDepth > 0 {
				continue
			}
			text := strings.Join(strings.Fields(string(tokenizer.Text())), " ")
			if text != "" {
				textParts = append(textParts, text)
			}
		}
	}
}

func browserRenderVisibleTextLength(textParts []string) int {
	return utf8.RuneCountInString(strings.Join(textParts, " "))
}

func browserRenderHiddenTextTag(tag string) bool {
	switch tag {
	case "head", "script", "style", "template", "svg", "canvas":
		return true
	default:
		return false
	}
}

func browserRenderTagSuggestsAppMount(token htmltoken.Token) bool {
	id := strings.ToLower(browserRenderTokenAttr(token, "id"))
	switch id {
	case "root", "app", "__next", "___gatsby", "gatsby-focus-wrapper":
		return true
	}
	return strings.Contains(strings.ToLower(browserRenderTokenAttr(token, "class")), "app-root")
}

func browserRenderTagSuggestsHydration(token htmltoken.Token) bool {
	for _, attr := range token.Attr {
		key := strings.ToLower(strings.TrimSpace(attr.Key))
		value := strings.ToLower(strings.TrimSpace(attr.Val))
		if key == "data-reactroot" || key == "ng-version" || key == "data-v-app" || key == "data-server-rendered" {
			return true
		}
		if strings.Contains(value, "__next_data__") || strings.Contains(value, "hydrate") {
			return true
		}
	}
	return false
}

func browserRenderTokenAttr(token htmltoken.Token, key string) string {
	for _, attr := range token.Attr {
		if strings.EqualFold(strings.TrimSpace(attr.Key), key) {
			return strings.TrimSpace(attr.Val)
		}
	}
	return ""
}

func browserRenderCandidateLooksLikeBotWall(lowerRaw string) bool {
	return strings.Contains(lowerRaw, "cf-chl") ||
		strings.Contains(lowerRaw, "turnstile") ||
		strings.Contains(lowerRaw, "captcha") ||
		strings.Contains(lowerRaw, "checking your browser") ||
		strings.Contains(lowerRaw, "verify you are human") ||
		strings.Contains(lowerRaw, "cloudflare")
}
