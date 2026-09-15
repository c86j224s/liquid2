package reporting

import (
	"regexp"
	"strings"
)

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
