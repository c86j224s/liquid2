package sourceretrieval

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

// Normalize returns the stable absolute HTTP(S) identity used for URL source
// deduplication. Fragments and host casing do not participate in identity.
func Normalize(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%w: source URL is required", producterror.ErrInvalidInput)
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("%w: source URL must be absolute", producterror.ErrInvalidInput)
	}
	if parsed.User != nil {
		return "", fmt.Errorf("%w: source URL must not include credentials", producterror.ErrInvalidInput)
	}
	parsed.Fragment = ""
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		parsed.Scheme = strings.ToLower(parsed.Scheme)
	default:
		return "", fmt.Errorf("%w: source URL must use http or https", producterror.ErrInvalidInput)
	}
	parsed.Host = strings.ToLower(parsed.Host)
	return parsed.String(), nil
}
