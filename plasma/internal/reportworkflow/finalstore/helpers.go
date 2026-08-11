package finalstore

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const markdownMediaType = "text/markdown; charset=utf-8"

func (runner Runner) id(prefix string) string {
	if runner.NewID != nil {
		return runner.NewID(prefix)
	}
	return prefix + "_missing"
}

func fallbackSessionID(primary string, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return strings.TrimSpace(primary)
	}
	return strings.TrimSpace(fallback)
}

func safeFilename(title string, ext string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "source"
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
		name = "source"
	}
	if len(name) > 80 {
		name = name[:80]
	}
	return name + ext
}

func sha256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
