package confluence

import (
	htmlpkg "html"
	"strings"

	"golang.org/x/net/html"
)

func plainTextFromStorage(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	node, err := html.Parse(strings.NewReader(trimmed))
	if err != nil {
		return strings.Join(strings.Fields(htmlpkg.UnescapeString(trimmed)), " ")
	}
	var out strings.Builder
	appendText(&out, node)
	return strings.Join(strings.Fields(out.String()), " ")
}

func appendText(out *strings.Builder, node *html.Node) {
	if node.Type == html.TextNode {
		out.WriteString(node.Data)
		out.WriteByte(' ')
	}
	if node.Type == html.ElementNode {
		switch strings.ToLower(node.Data) {
		case "br", "p", "div", "li", "tr", "h1", "h2", "h3", "h4", "h5", "h6":
			out.WriteByte(' ')
		case "script", "style":
			return
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		appendText(out, child)
	}
}
