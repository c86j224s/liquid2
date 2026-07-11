package sourcecandidates

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

type Candidate struct {
	URL    string `json:"url"`
	Title  string `json:"title"`
	Reason string `json:"reason"`
	State  string `json:"state"`
}

var (
	urlPattern              = regexp.MustCompile(`https?://[^\s<>"']+`)
	labelLinePattern        = regexp.MustCompile(`(?i)^\s*[-*•]?\s*(소스 후보|후보 소스|원자료 후보|source candidate|candidate source)\s*[:：]?\s*$`)
	candidateURLLinePattern = regexp.MustCompile(`(?i)^\s*[-*•]?\s*(소스 후보|후보 소스|원자료 후보|source candidate|candidate source)\s*[:：]?\s*https?://`)
	reasonLabelPatterns     = []*regexp.Regexp{
		regexp.MustCompile(`(?i)채택\s*의견\s*[:：]\s*(.+)$`),
		regexp.MustCompile(`(?i)수락\s*의견\s*[:：]\s*(.+)$`),
		regexp.MustCompile(`(?i)추천\s*이유\s*[:：]\s*(.+)$`),
		regexp.MustCompile(`(?i)검토\s*이유\s*[:：]\s*(.+)$`),
		regexp.MustCompile(`(?i)acceptance opinion\s*[:：]\s*(.+)$`),
		regexp.MustCompile(`(?i)why accept\s*[:：]\s*(.+)$`),
		regexp.MustCompile(`(?i)why this source\s*[:：]\s*(.+)$`),
	}
)

func Parse(text string) []Candidate {
	matches := urlPattern.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	candidates := make([]Candidate, 0, len(matches))
	for _, match := range matches {
		raw := strings.TrimRight(text[match[0]:match[1]], ".,;:!?)]}")
		normalized, err := normalizeHTTPURL(raw)
		if err != nil {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		if !hasCandidateLabel(text, match[0]) {
			continue
		}
		reason := acceptanceOpinion(text, match[0], match[0]+len(raw), raw, normalized)
		if reason == "" {
			continue
		}
		parsed, err := url.Parse(normalized)
		if err != nil {
			continue
		}
		seen[normalized] = struct{}{}
		candidates = append(candidates, Candidate{
			URL:    normalized,
			Title:  parsed.Hostname(),
			Reason: trimReason(reason),
			State:  "proposed",
		})
		if len(candidates) >= 8 {
			break
		}
	}
	return candidates
}

func normalizeHTTPURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("source URL is required")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("source URL must be absolute")
	}
	if parsed.User != nil {
		return "", fmt.Errorf("source URL must not include credentials")
	}
	parsed.Fragment = ""
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		parsed.Scheme = strings.ToLower(parsed.Scheme)
	default:
		return "", fmt.Errorf("source URL must use http or https")
	}
	parsed.Host = strings.ToLower(parsed.Host)
	return parsed.String(), nil
}

func hasCandidateLabel(text string, urlStart int) bool {
	if urlStart < 0 || urlStart > len(text) {
		return false
	}
	left := strings.LastIndex(text[:urlStart], "\n")
	if left < 0 {
		left = 0
	} else {
		left++
	}
	prefix := strings.TrimSpace(text[left:urlStart])
	return labelLinePattern.MatchString(prefix)
}

func acceptanceOpinion(text string, start int, end int, rawURL string, normalizedURL string) string {
	if start < 0 || end > len(text) || start >= end {
		return ""
	}
	lineEnd := strings.Index(text[end:], "\n")
	if lineEnd < 0 {
		lineEnd = len(text)
	} else {
		lineEnd += end
	}
	if reason := reasonFromLine(text[end:lineEnd], rawURL, normalizedURL); reason != "" {
		return reason
	}
	for _, line := range followingLines(text, lineEnd, 6) {
		cleaned := strings.TrimSpace(line)
		if cleaned == "" || labelLinePattern.MatchString(cleaned) || candidateURLLinePattern.MatchString(cleaned) {
			return ""
		}
		if reason := reasonFromLine(cleaned, rawURL, normalizedURL); reason != "" {
			return reason
		}
	}
	return ""
}

func reasonFromLine(line string, rawURL string, normalizedURL string) string {
	cleaned := strings.TrimSpace(line)
	cleaned = strings.ReplaceAll(cleaned, rawURL, "")
	cleaned = strings.ReplaceAll(cleaned, normalizedURL, "")
	for _, pattern := range reasonLabelPatterns {
		matches := pattern.FindStringSubmatch(cleaned)
		if len(matches) == 2 {
			return strings.TrimSpace(matches[1])
		}
	}
	return ""
}

func followingLines(text string, offset int, limit int) []string {
	if offset < 0 || offset > len(text) || limit <= 0 {
		return nil
	}
	rest := text[offset:]
	lines := strings.Split(rest, "\n")
	if len(lines) > 0 {
		lines = lines[1:]
	}
	if len(lines) > limit {
		lines = lines[:limit]
	}
	return lines
}

func trimReason(value string) string {
	const maxReasonLength = 360
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= maxReasonLength {
		return value
	}
	return strings.TrimSpace(string(runes[:maxReasonLength])) + "..."
}
