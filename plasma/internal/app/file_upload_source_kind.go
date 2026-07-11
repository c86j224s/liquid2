package app

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/sources/pdftext"
)

func classifyUploadedFile(filename string, content []byte) (string, string, error) {
	detectedMediaType := normalizeDetectedMediaType(http.DetectContentType(content))
	ext := strings.ToLower(filepath.Ext(filename))
	extMediaType := mediaTypeForUploadedExtension(ext)
	if pdftext.IsPDFMediaType(extMediaType) || pdftext.IsPDFMediaType(detectedMediaType) || pdftext.IsPDFBytes(content) {
		if err := validateUploadedPDF(content); err != nil {
			return "", "", err
		}
		return "application/pdf", UploadedContentKindPDF, nil
	}
	if strings.HasPrefix(extMediaType, "image/") || strings.HasPrefix(detectedMediaType, "image/") {
		mediaType, err := validateUploadedImage(content)
		if err != nil {
			return "", "", err
		}
		return mediaType, UploadedContentKindImage, nil
	}
	mediaType := detectedMediaType
	if extMediaType != "" && isUploadedTextMediaType(extMediaType) {
		mediaType = extMediaType
	}
	if isSupportedUploadedText(mediaType, ext, content) {
		if mediaType == "" || mediaType == "application/octet-stream" {
			mediaType = "text/plain; charset=utf-8"
		}
		return ensureUTF8TextMediaType(mediaType), UploadedContentKindText, nil
	}
	return "", "", fmt.Errorf("%w: unsupported uploaded file media type %q", ErrInvalidInput, detectedMediaType)
}

func validateUploadedPDF(content []byte) error {
	if !pdftext.IsPDFBytes(content) {
		return fmt.Errorf("%w: uploaded PDF does not contain valid PDF bytes", ErrInvalidInput)
	}
	if _, err := pdftext.Inspect(content); err != nil {
		return fmt.Errorf("%w: uploaded PDF inspection failed: %v", ErrInvalidInput, err)
	}
	return nil
}

func validateUploadedImage(content []byte) (string, error) {
	_, format, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("%w: uploaded image could not be decoded", ErrInvalidInput)
	}
	switch format {
	case "png":
		return "image/png", nil
	case "jpeg":
		return "image/jpeg", nil
	case "gif":
		return "image/gif", nil
	default:
		return "", fmt.Errorf("%w: unsupported uploaded image format %q", ErrInvalidInput, format)
	}
}

func isUploadedArtifactText(artifact RawArtifact) bool {
	mediaType, _, _ := mime.ParseMediaType(artifact.MediaType)
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	if strings.HasPrefix(mediaType, "text/") {
		return utf8.Valid(artifact.Content)
	}
	switch mediaType {
	case "application/json", "application/ld+json", "application/xml", "application/xhtml+xml", "application/rss+xml", "application/atom+xml":
		return utf8.Valid(artifact.Content)
	case "application/octet-stream":
		return utf8.Valid(artifact.Content)
	default:
		return false
	}
}

func isUploadedArtifactMetadataOnly(artifact RawArtifact) bool {
	mediaType, _, _ := mime.ParseMediaType(artifact.MediaType)
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	return strings.HasPrefix(mediaType, "image/")
}

func normalizeDetectedMediaType(mediaType string) string {
	base, params, err := mime.ParseMediaType(mediaType)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(mediaType))
	}
	if strings.EqualFold(base, "text/plain") {
		if charset := params["charset"]; charset != "" {
			return "text/plain; charset=" + strings.ToLower(charset)
		}
		return "text/plain"
	}
	return strings.ToLower(strings.TrimSpace(base))
}

func mediaTypeForUploadedExtension(ext string) string {
	switch ext {
	case ".md", ".markdown":
		return "text/markdown; charset=utf-8"
	case ".txt", ".log", ".csv", ".tsv":
		return "text/plain; charset=utf-8"
	case ".json", ".jsonl":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".xhtml":
		return "application/xhtml+xml"
	case ".pdf":
		return "application/pdf"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	default:
		return ""
	}
}

func isSupportedUploadedText(mediaType string, ext string, content []byte) bool {
	if !utf8.Valid(content) {
		return false
	}
	if !looksLikeText(content) {
		return false
	}
	base, _, err := mime.ParseMediaType(mediaType)
	if err == nil {
		mediaType = base
	}
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	if strings.HasPrefix(mediaType, "text/") {
		return true
	}
	switch mediaType {
	case "application/json", "application/ld+json", "application/xml", "application/xhtml+xml", "application/rss+xml", "application/atom+xml":
		return true
	case "application/octet-stream":
		return ext == ".md" || ext == ".markdown" || ext == ".txt" || looksLikeText(content)
	default:
		return false
	}
}

func isUploadedTextMediaType(mediaType string) bool {
	base, _, err := mime.ParseMediaType(mediaType)
	if err == nil {
		mediaType = base
	}
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	if strings.HasPrefix(mediaType, "text/") {
		return true
	}
	switch mediaType {
	case "application/json", "application/ld+json", "application/xml", "application/xhtml+xml", "application/rss+xml", "application/atom+xml":
		return true
	default:
		return false
	}
}

func looksLikeText(content []byte) bool {
	sample := content
	if len(sample) > 4096 {
		sample = sample[:4096]
	}
	for _, r := range string(sample) {
		if r == '\n' || r == '\r' || r == '\t' {
			continue
		}
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func ensureUTF8TextMediaType(mediaType string) string {
	base, params, err := mime.ParseMediaType(mediaType)
	if err != nil {
		return mediaType
	}
	base = strings.ToLower(strings.TrimSpace(base))
	if strings.HasPrefix(base, "text/") {
		if _, ok := params["charset"]; !ok {
			params["charset"] = "utf-8"
		}
		return mime.FormatMediaType(base, params)
	}
	return base
}
