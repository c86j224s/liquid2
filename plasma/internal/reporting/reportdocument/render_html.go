package reportdocument

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	htmlpkg "html"
	"strings"
)

func renderReportHTML(blocks []ReportBlock) ([]byte, error) {
	var out bytes.Buffer
	footnotes := newReportFootnotes()
	out.WriteString("<!doctype html>\n<html lang=\"ko\">\n<head>\n<meta charset=\"utf-8\">\n")
	out.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	out.WriteString("<title>Plasma Report</title>\n")
	out.WriteString("<style>body{margin:0;font:16px/1.65 -apple-system,BlinkMacSystemFont,\"Segoe UI\",sans-serif;color:#1f2937;background:#f8fafc}main{max-width:880px;margin:0 auto;padding:48px 24px 72px;background:#fff;min-height:100vh}h1{font-size:2rem;line-height:1.2;margin:0 0 24px}h2{font-size:1.35rem;margin:36px 0 12px}h3{font-size:1.1rem;margin:28px 0 10px}p{margin:0 0 16px}.lead{font-size:1.08rem;color:#475569;border-left:4px solid #0f766e;padding-left:14px}blockquote{margin:20px 0;padding:12px 16px;background:#f1f5f9;border-left:4px solid #64748b}ul{padding-left:1.4rem}.footnote-refs{font-size:.78em;vertical-align:super;margin-left:2px}.footnote-refs a{color:#0f766e;text-decoration:none}.footnotes{margin-top:44px;padding-top:18px;border-top:1px solid #e2e8f0;color:#475569;font-size:.9rem}.footnotes h2{font-size:1rem;margin:0 0 10px}.footnotes li{margin:4px 0;word-break:break-all}</style>\n")
	out.WriteString("</head>\n<body>\n<main>\n")
	for _, block := range blocks {
		switch block.BlockType {
		case "document":
			continue
		case "title":
			text, err := blockText(block)
			if err != nil {
				return nil, err
			}
			out.WriteString("<h1>" + htmlpkg.EscapeString(text) + "</h1>\n")
		case "abstract":
			text, err := blockText(block)
			if err != nil {
				return nil, err
			}
			out.WriteString("<p class=\"lead\">" + htmlpkg.EscapeString(text) + footnotes.htmlMarker(block.SourceRefs) + "</p>\n")
		case "paragraph":
			text, err := blockText(block)
			if err != nil {
				return nil, err
			}
			out.WriteString("<p>" + htmlpkg.EscapeString(text) + footnotes.htmlMarker(block.SourceRefs) + "</p>\n")
		case "heading":
			heading, err := blockHeading(block)
			if err != nil {
				return nil, err
			}
			level := heading.Level
			if level < 2 {
				level = 2
			}
			if level > 4 {
				level = 4
			}
			tag := fmt.Sprintf("h%d", level)
			out.WriteString("<" + tag + ">" + htmlpkg.EscapeString(heading.Text) + footnotes.htmlMarker(block.SourceRefs) + "</" + tag + ">\n")
		case "bullet_list":
			items, err := blockListItems(block)
			if err != nil {
				return nil, err
			}
			out.WriteString("<ul>\n")
			for index, item := range items {
				marker := ""
				if index == len(items)-1 {
					marker = footnotes.htmlMarker(block.SourceRefs)
				}
				out.WriteString("<li>" + htmlpkg.EscapeString(item) + marker + "</li>\n")
			}
			out.WriteString("</ul>\n")
		case "quote":
			text, err := blockText(block)
			if err != nil {
				return nil, err
			}
			out.WriteString("<blockquote>" + htmlpkg.EscapeString(text) + footnotes.htmlMarker(block.SourceRefs) + "</blockquote>\n")
		case "claim":
			var content struct {
				RenderedText string `json:"rendered_text"`
			}
			if err := json.Unmarshal(block.Content, &content); err != nil {
				return nil, err
			}
			out.WriteString("<p>" + htmlpkg.EscapeString(strings.TrimSpace(content.RenderedText)) + footnotes.htmlMarker(block.SourceRefs) + "</p>\n")
		case "evidence_summary":
			text, err := blockText(block)
			if err != nil {
				return nil, err
			}
			out.WriteString("<p>" + htmlpkg.EscapeString(text) + footnotes.htmlMarker(block.SourceRefs) + "</p>\n")
		case "unresolved_question", "option":
			text, err := blockText(block)
			if err != nil {
				return nil, err
			}
			out.WriteString("<p>" + htmlpkg.EscapeString(text) + footnotes.htmlMarker(block.SourceRefs) + "</p>\n")
		default:
			return nil, fmt.Errorf("%w: unsupported report block type", producterror.ErrInvalidInput)
		}
	}
	footnotes.writeHTMLDefinitions(&out)
	out.WriteString("</main>\n</body>\n</html>\n")
	return out.Bytes(), nil
}
