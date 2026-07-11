package sourcecandidates

import (
	"crypto/sha256"
	"encoding/hex"
	"mime"
	"strings"
)

func sourceCandidateFilename(title string, mediaType string, fallback string) string {
	title = strings.TrimSpace(title)
	fallback = strings.TrimSpace(fallback)
	if fallback == "" {
		fallback = "source-candidate"
	}
	if title == "" {
		title = fallback
	}
	var b strings.Builder
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		case r == ' ' || r == '.':
			b.WriteRune('-')
		}
	}
	name := strings.Trim(b.String(), "-_")
	if name == "" {
		name = fallback
	}
	if len(name) > 80 {
		name = name[:80]
	}
	return name + sourceCandidateFileExtension(mediaType)
}

func sourceCandidateFileExtension(mediaType string) string {
	base, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		base = mediaType
	}
	switch strings.ToLower(strings.TrimSpace(base)) {
	case "text/html", "application/xhtml+xml":
		return ".html"
	case "application/pdf":
		return ".pdf"
	case "application/json", "application/ld+json":
		return ".json"
	case "application/xml", "application/rss+xml", "application/atom+xml":
		return ".xml"
	default:
		return ".txt"
	}
}

func sha256HexBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
