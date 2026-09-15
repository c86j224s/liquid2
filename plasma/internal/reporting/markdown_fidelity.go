package reporting

import (
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"regexp"
	"strings"
)

var markdownHeadingPattern = regexp.MustCompile(`^#{1,6}\s+`)

var markdownCoreMeaningMarkerPattern = regexp.MustCompile(`원문|인상|근거|추측|사실|의견|출처|인용|검증|확인|승인|기각|허용|금지|성공|실패|위험|안전`)

var markdownNegativeMeaningMarkerPattern = regexp.MustCompile(`불가능|불필요|부정확|불확실|불충분|불일치|않|아니|못|없`)

var markdownStandaloneNegativePattern = regexp.MustCompile(`(^|[^가-힣A-Za-z0-9])(안된다[가-힣]*|안된[가-힣]*|안돼[가-힣]*|안됨[가-힣]*|안한다[가-힣]*|안한[가-힣]*|안하[가-힣]*|안함[가-힣]*|안해[가-힣]*|안\s*되[가-힣]*|안\s+[가-힣]+)`)

// ValidateHumanizedMarkdown는 보고서 생성 파이프라인 계약을 검사한다. 제품 상태를 변경하지 않는 순수 검증 경계다.
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
		return fmt.Errorf("%w: humanized Markdown failed fidelity guard: %s", producterror.ErrInvalidInput, strings.Join(failures, ", "))
	}
	return nil
}

// ValidateFinalEditStyleMarkdown는 보고서 생성 파이프라인 계약을 검사한다. 제품 상태를 변경하지 않는 순수 검증 경계다.
func ValidateFinalEditStyleMarkdown(reader string, style string) error {
	var failures []string
	checkEqual := func(label string, a []string, b []string) {
		if !sameStringSlice(a, b) {
			failures = append(failures, label)
		}
	}
	checkEqual("heading_order", markdownHeadings(reader), markdownHeadings(style))
	checkEqual("table_lines", markdownTableLines(reader), markdownTableLines(style))
	checkEqual("code_fences", markdownCodeFenceBlocks(reader), markdownCodeFenceBlocks(style))
	checkEqual("blockquote_lines", markdownBlockquoteLines(reader), markdownBlockquoteLines(style))
	checkEqual("source_bearing_lines", markdownSourceBearingLines(reader), markdownSourceBearingLines(style))
	checkEqual("list_markers", markdownListMarkers(reader), markdownListMarkers(style))
	checkEqual("links", regexpFindAll(`!?\[[^\]]*\]\([^)]+\)|https?://[^\s)<]+`, reader), regexpFindAll(`!?\[[^\]]*\]\([^)]+\)|https?://[^\s)<]+`, style))
	checkEqual("footnotes", regexpFindAll(`\[\^[^\]]+\](?::[^\n]*)?`, reader), regexpFindAll(`\[\^[^\]]+\](?::[^\n]*)?`, style))
	checkEqual("inline_code", regexpFindAll("`[^`\n]+`", reader), regexpFindAll("`[^`\n]+`", style))
	checkEqual("quoted_text", regexpFindAll(`"[^"\n]+"|'[^'\n]+'|“[^”\n]+”|‘[^’\n]+’`, reader), regexpFindAll(`"[^"\n]+"|'[^'\n]+'|“[^”\n]+”|‘[^’\n]+’`, style))
	checkEqual("numbers", regexpFindAll(`[-+]?\d+(?:[.,:/-]\d+)*(?:%|[A-Za-z가-힣]+)?`, reader), regexpFindAll(`[-+]?\d+(?:[.,:/-]\d+)*(?:%|[A-Za-z가-힣]+)?`, style))
	checkEqual("latin_technical_tokens", regexpFindAll(`[A-Za-z][A-Za-z0-9._:+/#-]*[A-Za-z0-9]`, reader), regexpFindAll(`[A-Za-z][A-Za-z0-9._:+/#-]*[A-Za-z0-9]`, style))
	if len(markdownNonEmptyBlocks(reader)) != len(markdownNonEmptyBlocks(style)) {
		failures = append(failures, "nonempty_block_count")
	}
	if len(failures) > 0 {
		return fmt.Errorf("%w: final edit style Markdown failed fidelity guard: %s", producterror.ErrInvalidInput, strings.Join(failures, ", "))
	}
	return nil
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
