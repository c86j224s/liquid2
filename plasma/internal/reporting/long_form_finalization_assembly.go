package reporting

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
)

var longFormHeadingLineRE = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*$`)

func AssembleLongFormFinalMarkdown(title, opening, closing string, parts []string) string {
	var out strings.Builder
	if opening = normalizeLongFormBoundary(opening, false); opening != "" {
		out.WriteString(opening)
	} else {
		out.WriteString("# " + firstReportingNonEmpty(title, "Mission report"))
	}
	out.WriteString("\n\n---\n\n")
	for _, part := range parts {
		out.WriteString(strings.TrimSpace(part))
		out.WriteString("\n\n")
	}
	if closing = normalizeLongFormBoundary(closing, true); closing != "" {
		out.WriteString("---\n\n" + closing + "\n")
	}
	return strings.TrimSpace(out.String()) + "\n"
}

func normalizeLongFormBoundary(markdown string, closing bool) string {
	lines := strings.Split(strings.TrimSpace(markdown), "\n")
	lines = stripLongFormBoundaryRules(lines)
	inFence := ""
	out := make([]string, 0, len(lines))
	lastHeading := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if marker := longFormFenceMarker(trimmed); marker != "" {
			if inFence == "" {
				inFence = marker
			} else if inFence == marker {
				inFence = ""
			}
			out = append(out, line)
			lastHeading = ""
			continue
		}
		if inFence != "" {
			out = append(out, line)
			lastHeading = ""
			continue
		}
		match := longFormHeadingLineRE.FindStringSubmatch(trimmed)
		if len(match) == 3 {
			canonical := strings.ToLower(strings.Join(strings.Fields(match[2]), " "))
			if canonical != "" && canonical == lastHeading {
				continue
			}
			level := len(match[1])
			if closing {
				level = 2
			} else if level > 2 {
				level = 2
			}
			out = append(out, strings.Repeat("#", level)+" "+strings.TrimSpace(match[2]))
			lastHeading = canonical
			continue
		}
		out = append(out, line)
		if trimmed != "" {
			lastHeading = ""
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func stripLongFormBoundaryRules(lines []string) []string {
	start, end := 0, len(lines)
	for start < end && (strings.TrimSpace(lines[start]) == "" || longFormHorizontalRule(lines[start])) {
		start++
	}
	for end > start && (strings.TrimSpace(lines[end-1]) == "" || longFormHorizontalRule(lines[end-1])) {
		end--
	}
	return lines[start:end]
}

func longFormHorizontalRule(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 {
		return false
	}
	for _, marker := range []rune{'-', '*', '_'} {
		count, valid := 0, true
		for _, value := range trimmed {
			if value == marker {
				count++
			} else if value != ' ' && value != '\t' {
				valid = false
				break
			}
		}
		if valid && count >= 3 {
			return true
		}
	}
	return false
}

func longFormFenceMarker(line string) string {
	for _, marker := range []string{"```", "~~~"} {
		if strings.HasPrefix(line, marker) {
			return marker
		}
	}
	return ""
}
func firstReportingNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
func contentSHA256(content []byte) string { return fmt.Sprintf("%x", sha256.Sum256(content)) }
