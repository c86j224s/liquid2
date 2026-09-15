package reportdocument

import (
	"bytes"
	"fmt"
	htmlpkg "html"
	"strings"
)

func reportRefValues(refs ReportBlockSourceRefs) []string {
	values := []string{}
	seen := map[string]struct{}{}
	appendValue := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	for _, value := range refs.ClaimIDs {
		appendValue(value)
	}
	for _, value := range refs.EvidenceIDs {
		appendValue(value)
	}
	for _, value := range refs.SnapshotIDs {
		appendValue(value)
	}
	for _, value := range refs.QuestionIDs {
		appendValue(value)
	}
	for _, value := range refs.OptionIDs {
		appendValue(value)
	}
	return values
}

type reportFootnotes struct {
	index  map[string]int
	values []string
}

func newReportFootnotes() *reportFootnotes {
	return &reportFootnotes{index: map[string]int{}}
}

func (footnotes *reportFootnotes) numbers(refs ReportBlockSourceRefs) []int {
	values := reportRefValues(refs)
	numbers := make([]int, 0, len(values))
	for _, value := range values {
		number, ok := footnotes.index[value]
		if !ok {
			footnotes.values = append(footnotes.values, value)
			number = len(footnotes.values)
			footnotes.index[value] = number
		}
		numbers = append(numbers, number)
	}
	return numbers
}

func (footnotes *reportFootnotes) markdownMarker(refs ReportBlockSourceRefs) string {
	numbers := footnotes.numbers(refs)
	if len(numbers) == 0 {
		return ""
	}
	parts := make([]string, 0, len(numbers))
	for _, number := range numbers {
		parts = append(parts, fmt.Sprintf("[^%d]", number))
	}
	return " " + strings.Join(parts, " ")
}

func (footnotes *reportFootnotes) htmlMarker(refs ReportBlockSourceRefs) string {
	numbers := footnotes.numbers(refs)
	if len(numbers) == 0 {
		return ""
	}
	var out bytes.Buffer
	out.WriteString("<sup class=\"footnote-refs\">")
	for _, number := range numbers {
		out.WriteString(fmt.Sprintf("<a href=\"#fn-%d\" id=\"fnref-%d\">[%d]</a>", number, number, number))
	}
	out.WriteString("</sup>")
	return out.String()
}

func (footnotes *reportFootnotes) writeMarkdownDefinitions(out *bytes.Buffer) {
	if len(footnotes.values) == 0 {
		return
	}
	out.WriteString("\n## 각주\n\n")
	for index, value := range footnotes.values {
		out.WriteString(fmt.Sprintf("[^%d]: `%s`\n", index+1, value))
	}
}

func (footnotes *reportFootnotes) writeHTMLDefinitions(out *bytes.Buffer) {
	if len(footnotes.values) == 0 {
		return
	}
	out.WriteString("<section class=\"footnotes\"><h2>각주</h2><ol>\n")
	for index, value := range footnotes.values {
		number := index + 1
		out.WriteString(fmt.Sprintf("<li id=\"fn-%d\"><code>%s</code> <a href=\"#fnref-%d\" aria-label=\"본문으로 돌아가기\">back</a></li>\n", number, htmlpkg.EscapeString(value), number))
	}
	out.WriteString("</ol></section>\n")
}
