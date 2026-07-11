package sourcecandidates

import (
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func NormalizeSourceCandidateProposals(input []SourceCandidateProposalInput) ([]SourceCandidateProposal, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("%w: at least one source candidate is required", app.ErrInvalidInput)
	}
	if len(input) > 8 {
		return nil, fmt.Errorf("%w: at most 8 source candidates can be proposed at once", app.ErrInvalidInput)
	}
	seen := map[string]struct{}{}
	candidates := make([]SourceCandidateProposal, 0, len(input))
	for _, candidate := range input {
		normalizedURL, host, err := NormalizeSourceCandidateURL(candidate.URL)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[normalizedURL]; ok {
			continue
		}
		reason := truncateRunes(strings.TrimSpace(candidate.Reason), 720)
		if reason == "" {
			return nil, fmt.Errorf("%w: source candidate reason is required", app.ErrInvalidInput)
		}
		title := truncateRunes(strings.TrimSpace(candidate.Title), 180)
		if title == "" {
			title = sourceCandidateTitleFromURL(normalizedURL)
		}
		if title == "" {
			title = host
		}
		seen[normalizedURL] = struct{}{}
		candidates = append(candidates, SourceCandidateProposal{
			URL:    normalizedURL,
			Title:  title,
			Reason: reason,
			State:  "proposed",
		})
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("%w: at least one unique source candidate is required", app.ErrInvalidInput)
	}
	return candidates, nil
}

func NormalizeSourceCandidateURL(raw string) (string, string, error) {
	return normalizeSourceCandidateURL(raw, "source candidate URL")
}

func sourceCandidateTitleFromURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "atlassian.net" && !strings.HasSuffix(host, ".atlassian.net") {
		return ""
	}
	segments := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	for i := 0; i+2 < len(segments); i++ {
		if segments[i] != "pages" {
			continue
		}
		if _, err := url.PathUnescape(segments[i+1]); err != nil {
			return ""
		}
		title, err := url.PathUnescape(segments[i+2])
		if err != nil {
			return ""
		}
		title = strings.ReplaceAll(title, "+", " ")
		title = strings.Join(strings.Fields(title), " ")
		if title == "" || !utf8.ValidString(title) {
			return ""
		}
		return truncateRunes(title, 180)
	}
	return ""
}

func normalizeSourceCandidateDecisionURL(raw string) (string, error) {
	normalizedURL, _, err := normalizeSourceCandidateURL(raw, "source URL")
	return normalizedURL, err
}

func normalizeSourceCandidateURL(raw string, errorLabel string) (string, string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", "", fmt.Errorf("%w: %s is required", app.ErrInvalidInput, errorLabel)
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", "", fmt.Errorf("%w: %s must be absolute", app.ErrInvalidInput, errorLabel)
	}
	if parsed.User != nil {
		return "", "", fmt.Errorf("%w: %s must not include credentials", app.ErrInvalidInput, errorLabel)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		parsed.Scheme = strings.ToLower(parsed.Scheme)
	default:
		return "", "", fmt.Errorf("%w: %s must use http or https", app.ErrInvalidInput, errorLabel)
	}
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Fragment = ""
	return parsed.String(), parsed.Hostname(), nil
}

func truncateRunes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
