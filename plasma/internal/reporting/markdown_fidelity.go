package reporting

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

var markdownHeadingPattern = regexp.MustCompile(`^#{1,6}\s+`)
var markdownCoreMeaningMarkerPattern = regexp.MustCompile(`원문|인상|근거|추측|사실|의견|출처|인용|검증|확인|승인|기각|허용|금지|성공|실패|위험|안전`)
var markdownNegativeMeaningMarkerPattern = regexp.MustCompile(`불가능|불필요|부정확|불확실|불충분|불일치|않|아니|못|없`)
var markdownStandaloneNegativePattern = regexp.MustCompile(`(^|[^가-힣A-Za-z0-9])(안된다[가-힣]*|안된[가-힣]*|안돼[가-힣]*|안됨[가-힣]*|안한다[가-힣]*|안한[가-힣]*|안하[가-힣]*|안함[가-힣]*|안해[가-힣]*|안\s*되[가-힣]*|안\s+[가-힣]+)`)

func ValidateHumanizedMarkdown(original string, humanized string) error {
	var failures []string
	checkEqual := func(label string, a []string, b []string) {
		if !sameStringSlice(a, b) {
			failures = append(failures, label)
		}
	}
	checkEqual("heading_order", markdownHeadings(original), markdownHeadings(humanized))
	checkEqual("table_lines", markdownTableLines(original), markdownTableLines(humanized))
	checkEqual("code_fences", markdownCodeFenceBlocks(original), markdownCodeFenceBlocks(humanized))
	checkEqual("blockquote_lines", markdownBlockquoteLines(original), markdownBlockquoteLines(humanized))
	checkEqual("source_bearing_lines", markdownSourceBearingLines(original), markdownSourceBearingLines(humanized))
	checkEqual("list_markers", markdownListMarkers(original), markdownListMarkers(humanized))
	checkEqual("links", regexpFindAll(`!?\[[^\]]*\]\([^)]+\)|https?://[^\s)<]+`, original), regexpFindAll(`!?\[[^\]]*\]\([^)]+\)|https?://[^\s)<]+`, humanized))
	checkEqual("footnotes", regexpFindAll(`\[\^[^\]]+\](?::[^\n]*)?`, original), regexpFindAll(`\[\^[^\]]+\](?::[^\n]*)?`, humanized))
	checkEqual("inline_code", regexpFindAll("`[^`\n]+`", original), regexpFindAll("`[^`\n]+`", humanized))
	checkEqual("quoted_text", regexpFindAll(`"[^"\n]+"|'[^'\n]+'|“[^”\n]+”|‘[^’\n]+’`, original), regexpFindAll(`"[^"\n]+"|'[^'\n]+'|“[^”\n]+”|‘[^’\n]+’`, humanized))
	checkEqual("numbers", regexpFindAll(`[-+]?\d+(?:[.,:/-]\d+)*(?:%|[A-Za-z가-힣]+)?`, original), regexpFindAll(`[-+]?\d+(?:[.,:/-]\d+)*(?:%|[A-Za-z가-힣]+)?`, humanized))
	checkEqual("latin_technical_tokens", regexpFindAll(`[A-Za-z][A-Za-z0-9._:+/#-]*[A-Za-z0-9]`, original), regexpFindAll(`[A-Za-z][A-Za-z0-9._:+/#-]*[A-Za-z0-9]`, humanized))
	checkEqual("core_meaning_markers", markdownMeaningMarkers(markdownCoreMeaningMarkerPattern, original), markdownMeaningMarkers(markdownCoreMeaningMarkerPattern, humanized))
	checkEqual("negative_meaning_markers", markdownNegativeMeaningMarkers(original), markdownNegativeMeaningMarkers(humanized))
	if markdownSentenceTerminatorCount(original) != markdownSentenceTerminatorCount(humanized) {
		failures = append(failures, "sentence_terminator_count")
	}
	if len(markdownNonEmptyBlocks(original)) != len(markdownNonEmptyBlocks(humanized)) {
		failures = append(failures, "nonempty_block_count")
	}
	failures = append(failures, markdownHumanizeChangeBudgetFailures(original, humanized)...)
	if len(failures) > 0 {
		return fmt.Errorf("%w: humanized Markdown failed fidelity guard: %s", app.ErrInvalidInput, strings.Join(failures, ", "))
	}
	return nil
}

func markdownMeaningMarkers(pattern *regexp.Regexp, text string) []string {
	return pattern.FindAllString(markdownWithoutFenceBlocks(text), -1)
}

func markdownNegativeMeaningMarkers(text string) []string {
	withoutFences := markdownWithoutFenceBlocks(text)
	markers := markdownNegativeMeaningMarkerPattern.FindAllString(withoutFences, -1)
	for _, match := range markdownStandaloneNegativePattern.FindAllStringSubmatch(withoutFences, -1) {
		if len(match) >= 3 {
			markers = append(markers, strings.Join(strings.Fields(match[2]), " "))
		}
	}
	return markers
}

func markdownHeadings(text string) []string {
	lines := strings.Split(text, "\n")
	out := []string{}
	inFence := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isMarkdownFenceLine(trimmed) {
			inFence = !inFence
			continue
		}
		if !inFence && markdownHeadingPattern.MatchString(trimmed) {
			out = append(out, trimmed)
		}
	}
	return out
}

func markdownTableLines(text string) []string {
	return markdownLinesMatching(text, func(line string) bool {
		trimmed := strings.TrimSpace(line)
		return strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|")
	})
}

func markdownBlockquoteLines(text string) []string {
	return markdownLinesMatching(text, func(line string) bool {
		return strings.HasPrefix(strings.TrimSpace(line), ">")
	})
}

func markdownSourceBearingLines(text string) []string {
	lines := strings.Split(text, "\n")
	out := []string{}
	inFence := false
	inSourceSection := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isMarkdownFenceLine(trimmed) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if markdownHeadingPattern.MatchString(trimmed) {
			inSourceSection = isMarkdownSourceSectionHeading(trimmed)
			continue
		}
		if isMarkdownSourceBearingLine(line) || (inSourceSection && trimmed != "") {
			out = append(out, trimmed)
		}
	}
	return out
}

func isMarkdownSourceBearingLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "source:") ||
		strings.HasPrefix(lower, "sources:") ||
		strings.HasPrefix(lower, "reference:") ||
		strings.HasPrefix(lower, "references:") ||
		strings.HasPrefix(lower, "citation:") ||
		strings.HasPrefix(lower, "citations:") ||
		strings.HasPrefix(lower, "evidence:") ||
		strings.HasPrefix(lower, "출처:") ||
		strings.HasPrefix(lower, "출처：") ||
		strings.HasPrefix(lower, "참고:") ||
		strings.HasPrefix(lower, "참고：") ||
		strings.HasPrefix(lower, "근거:") ||
		strings.HasPrefix(lower, "근거：") ||
		strings.HasPrefix(lower, "인용:") ||
		strings.HasPrefix(lower, "인용：") {
		return true
	}
	if strings.HasPrefix(trimmed, "[^") && strings.Contains(trimmed, "]:") {
		return true
	}
	if regexp.MustCompile(`^\s*[-*+]\s+(source|reference|citation|evidence|출처|참고|근거|인용)\s*[:：]`).MatchString(lower) {
		return true
	}
	return false
}

func isMarkdownSourceSectionHeading(line string) bool {
	title := strings.TrimSpace(markdownHeadingPattern.ReplaceAllString(line, ""))
	title = strings.Trim(title, " #")
	lower := strings.ToLower(title)
	switch lower {
	case "source", "sources", "reference", "references", "citation", "citations", "evidence":
		return true
	case "출처", "참고", "참고자료", "참고 자료", "근거", "인용", "출처 및 참고자료", "출처와 참고자료":
		return true
	default:
		return false
	}
}

func markdownListMarkers(text string) []string {
	lines := strings.Split(text, "\n")
	out := []string{}
	inFence := false
	pattern := regexp.MustCompile(`^(\s*)([-*+]|\d+[.)])\s+(\[[ xX]\]\s+)?`)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isMarkdownFenceLine(trimmed) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if match := pattern.FindStringSubmatch(line); match != nil {
			out = append(out, match[1]+match[2]+match[3])
		}
	}
	return out
}

func markdownHumanizeChangeBudgetFailures(original string, humanized string) []string {
	originalLines := splitMarkdownLinesForComparison(original)
	humanizedLines := splitMarkdownLinesForComparison(humanized)
	if len(originalLines) != len(humanizedLines) {
		return []string{"line_count"}
	}
	totalNonEmpty := 0
	changedLines := 0
	changedSpan := 0
	lineLocalityFailure := false
	for i := range originalLines {
		a := strings.TrimSpace(originalLines[i])
		b := strings.TrimSpace(humanizedLines[i])
		if a != "" {
			totalNonEmpty++
		}
		if a == b {
			continue
		}
		changedLines++
		lineSpan, stablePrefix, stableSuffix := changedMiddleRuneMetrics(a, b)
		changedSpan += lineSpan
		if lineSpan > maxHumanizeLineChangedRunes(a, b) {
			lineLocalityFailure = true
		}
		if stablePrefix < 4 && lineSpan > 40 {
			lineLocalityFailure = true
		}
		if shortLineSemanticRewriteRisk(a, b, lineSpan, stablePrefix, stableSuffix) {
			lineLocalityFailure = true
		}
	}
	failures := []string{}
	if changedLines > maxHumanizeChangedLines(totalNonEmpty) {
		failures = append(failures, "changed_line_budget")
	}
	if changedSpan > maxHumanizeChangedRunes(original) {
		failures = append(failures, "changed_text_budget")
	}
	if lineLocalityFailure {
		failures = append(failures, "line_locality")
	}
	return failures
}

func splitMarkdownLinesForComparison(text string) []string {
	return strings.Split(strings.TrimRight(text, "\n"), "\n")
}

func maxHumanizeChangedLines(totalNonEmpty int) int {
	if totalNonEmpty <= 0 {
		return 0
	}
	limit := totalNonEmpty / 3
	if totalNonEmpty >= 24 && limit < 8 {
		limit = 8
	} else if limit < 1 {
		limit = 1
	}
	if limit > 48 {
		limit = 48
	}
	return limit
}

func maxHumanizeChangedRunes(original string) int {
	limit := len([]rune(original)) / 4
	if limit < 1200 {
		limit = 1200
	}
	if limit > 8000 {
		return 8000
	}
	return limit
}

func maxHumanizeLineChangedRunes(a string, b string) int {
	longer := len([]rune(a))
	if candidate := len([]rune(b)); candidate > longer {
		longer = candidate
	}
	limit := longer / 2
	switch {
	case longer <= 80:
		if limit < 18 {
			limit = 18
		}
	case longer <= 220:
		if limit < 72 {
			limit = 72
		}
	default:
		if limit < 220 {
			limit = 220
		}
	}
	if limit > 480 {
		return 480
	}
	return limit
}

func shortLineSemanticRewriteRisk(a string, b string, lineSpan int, stablePrefix int, stableSuffix int) bool {
	longer := len([]rune(a))
	if candidate := len([]rune(b)); candidate > longer {
		longer = candidate
	}
	if longer > 120 || lineSpan < 10 {
		return false
	}
	return stablePrefix < 8 && stableSuffix < 8
}

func changedMiddleRuneMetrics(a string, b string) (int, int, int) {
	ar := []rune(a)
	br := []rune(b)
	prefix := 0
	for prefix < len(ar) && prefix < len(br) && ar[prefix] == br[prefix] {
		prefix++
	}
	aSuffix := len(ar)
	bSuffix := len(br)
	for aSuffix > prefix && bSuffix > prefix && ar[aSuffix-1] == br[bSuffix-1] {
		aSuffix--
		bSuffix--
	}
	aChanged := aSuffix - prefix
	bChanged := bSuffix - prefix
	stableSuffix := len(ar) - aSuffix
	if candidate := len(br) - bSuffix; candidate < stableSuffix {
		stableSuffix = candidate
	}
	if bChanged > aChanged {
		return bChanged, prefix, stableSuffix
	}
	return aChanged, prefix, stableSuffix
}

func markdownSentenceTerminatorCount(text string) int {
	withoutFences := markdownWithoutFenceBlocks(text)
	return len(regexpFindAll(`[.!?。？！]`, withoutFences))
}

func markdownWithoutFenceBlocks(text string) string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	inFence := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isMarkdownFenceLine(trimmed) {
			inFence = !inFence
			continue
		}
		if !inFence {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

func markdownLinesMatching(text string, keep func(string) bool) []string {
	lines := strings.Split(text, "\n")
	out := []string{}
	inFence := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isMarkdownFenceLine(trimmed) {
			inFence = !inFence
			continue
		}
		if !inFence && keep(line) {
			out = append(out, trimmed)
		}
	}
	return out
}

func markdownCodeFenceBlocks(text string) []string {
	lines := strings.Split(text, "\n")
	out := []string{}
	var current []string
	inFence := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isMarkdownFenceLine(trimmed) {
			current = append(current, line)
			if inFence {
				out = append(out, strings.Join(current, "\n"))
				current = nil
			}
			inFence = !inFence
			continue
		}
		if inFence {
			current = append(current, line)
		}
	}
	if len(current) > 0 {
		out = append(out, strings.Join(current, "\n"))
	}
	return out
}

func markdownNonEmptyBlocks(text string) []string {
	parts := regexp.MustCompile(`\n\s*\n`).Split(strings.TrimSpace(text), -1)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			out = append(out, strings.TrimSpace(part))
		}
	}
	return out
}

func regexpFindAll(pattern string, text string) []string {
	return regexp.MustCompile(pattern).FindAllString(text, -1)
}

func sameStringSlice(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isMarkdownFenceLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")
}
