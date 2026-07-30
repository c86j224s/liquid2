package reporting

import (
	"strings"
	"testing"
)

func TestValidateFinalEditStyleMarkdownAllowsKoreanSentencePolishLegacyRejects(t *testing.T) {
	reader := "# Reader\n\n독자는 문서 내용만이 아니라 발행 위치에서 먼저 맥락을 확인해야 하며, 이후 증거와 한계를 차례로 검토합니다.\n\n그 결과를 바탕으로 팀은 결론을 더 분명하게 정리합니다.\n"
	style := "# Reader\n\n독자는 문서 내용만큼이나 발행 위치에서도 맥락을 먼저 확인하고, 이어서 증거와 한계를 차분히 검토합니다.\n\n그 판단을 바탕으로 팀은 결론을 한층 명확하게 정리합니다.\n"
	if err := ValidateFinalEditStyleMarkdown(reader, style); err != nil {
		t.Fatalf("style structural preflight rejected valid polish: %v", err)
	}
	err := ValidateHumanizedMarkdown(reader, style)
	if err == nil {
		t.Fatal("legacy humanized guard unexpectedly accepted the observed Korean polish")
	}
	for _, label := range []string{"negative_meaning_markers", "line_locality"} {
		if !strings.Contains(err.Error(), label) {
			t.Fatalf("legacy guard error missing %s: %v", label, err)
		}
	}
}

func TestValidateFinalEditStyleMarkdownRejectsRetainedInvariants(t *testing.T) {
	reader := strings.Join([]string{
		"# Title",
		"",
		"문단은 `codeValue`와 \"quoted value\"를 포함하며 ISO/IEC 40500:2012 및 WCAG 2.0을 유지합니다 [link](https://example.com/a).",
		"",
		"> 인용 블록은 그대로 둡니다.",
		"",
		"- 첫 항목 [^1]",
		"- 둘째 항목",
		"",
		"| A | B |",
		"| - | - |",
		"| 1 | 2 |",
		"",
		"```go",
		"func main() {}",
		"```",
		"",
		"## Sources",
		"",
		"[^1]: Evidence 2026-07-28.",
	}, "\n")
	cases := map[string]string{
		"heading_order":          strings.Replace(reader, "# Title", "# Different", 1),
		"table_lines":            strings.Replace(reader, "| 1 | 2 |", "| 1 | 3 |", 1),
		"code_fences":            strings.Replace(reader, "func main() {}", "func run() {}", 1),
		"blockquote_lines":       strings.Replace(reader, "> 인용 블록은 그대로 둡니다.", "> 인용 블록을 바꿉니다.", 1),
		"source_bearing_lines":   strings.Replace(reader, "[^1]: Evidence 2026-07-28.", "[^1]: Evidence 2026-07-29.", 1),
		"list_markers":           strings.Replace(reader, "- 둘째 항목", "* 둘째 항목", 1),
		"links":                  strings.Replace(reader, "https://example.com/a", "https://example.com/b", 1),
		"footnotes":              strings.Replace(reader, "[^1]", "[^2]", 1),
		"inline_code":            strings.Replace(reader, "`codeValue`", "`codeOther`", 1),
		"quoted_text":            strings.Replace(reader, "\"quoted value\"", "\"quoted other\"", 1),
		"numbers":                strings.Replace(reader, "2026-07-28", "2026-07-29", 1),
		"latin_technical_tokens": strings.Replace(reader, "WCAG", "ARIA", 1),
		"nonempty_block_count":   strings.Replace(reader, "\n\n> 인용 블록은 그대로 둡니다.", "\n> 인용 블록은 그대로 둡니다.", 1),
	}
	for label, style := range cases {
		t.Run(label, func(t *testing.T) {
			err := ValidateFinalEditStyleMarkdown(reader, style)
			if err == nil || !strings.Contains(err.Error(), label) {
				t.Fatalf("error=%v, want retained invariant label %s", err, label)
			}
		})
	}
}
