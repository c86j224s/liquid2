package partassembly

import (
	"fmt"
	"strings"
)

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func partHeadingLine(value string, partNumber int) string {
	heading := fmt.Sprintf("# Part %d.", partNumber)
	if title := displayPartHeadingText(value, partNumber); title != "" {
		return heading + " " + title
	}
	return heading
}

// displayPartHeadingText removes only a matching Part label; other leading numbers remain title content.
func displayPartHeadingText(value string, partNumber int) string {
	value = strings.TrimSpace(value)
	for _, prefix := range []string{
		fmt.Sprintf("Part %d.", partNumber),
		fmt.Sprintf("%d부.", partNumber),
	} {
		if value == prefix {
			return ""
		}
		if strings.HasPrefix(value, prefix+" ") {
			return strings.TrimSpace(strings.TrimPrefix(value, prefix))
		}
	}
	return value
}

func displayHeadingText(value string, partNumber int, sectionNumber int) string {
	value = strings.TrimSpace(value)
	for _, prefix := range []string{
		fmt.Sprintf("%d.%d. ", partNumber, sectionNumber),
		fmt.Sprintf("%d.%d ", partNumber, sectionNumber),
	} {
		value = strings.TrimPrefix(value, prefix)
	}
	withoutNumbering := strings.TrimSpace(markdownLeadingGlobalSectionNumberRE.ReplaceAllString(value, ""))
	if withoutNumbering == "" {
		return value
	}
	return withoutNumbering
}

func stripBoundaryHorizontalRules(lines []string) []string {
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	if start < len(lines) && isHorizontalRule(strings.TrimSpace(lines[start])) {
		lines = append(lines[:start], lines[start+1:]...)
	}
	end := len(lines) - 1
	for end >= 0 && strings.TrimSpace(lines[end]) == "" {
		end--
	}
	if end >= 0 && isHorizontalRule(strings.TrimSpace(lines[end])) {
		lines = append(lines[:end], lines[end+1:]...)
	}
	return lines
}

func isHorizontalRule(value string) bool {
	if len(value) < 3 {
		return false
	}
	for _, r := range value {
		if r != '-' && r != '*' && r != '_' {
			return false
		}
	}
	return true
}

func markdownFenceMarker(value string) (string, bool) {
	if strings.HasPrefix(value, "```") {
		return "```", true
	}
	if strings.HasPrefix(value, "~~~") {
		return "~~~", true
	}
	return "", false
}

func canonicalHeadingText(text string) string {
	text = markdownLeadingNumberingRE.ReplaceAllString(strings.TrimSpace(text), "")
	return strings.ToLower(strings.Join(strings.Fields(text), " "))
}

func canonicalHeadingTextSet(values []string) map[string]bool {
	set := map[string]bool{}
	for _, value := range values {
		canonical := canonicalHeadingText(value)
		if canonical != "" {
			set[canonical] = true
		}
	}
	return set
}
