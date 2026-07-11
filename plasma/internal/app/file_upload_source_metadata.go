package app

import (
	"mime"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/sources/pdftext"
)

func sanitizeUploadedFilename(filename string, mediaType string) string {
	filename = filepath.Base(strings.TrimSpace(filename))
	filename = strings.Map(func(r rune) rune {
		switch {
		case r == '-' || r == '_' || r == '.':
			return r
		case r >= '0' && r <= '9':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= 'a' && r <= 'z':
			return r
		default:
			return '-'
		}
	}, filename)
	filename = strings.Trim(filename, ".- _")
	if filename == "" {
		filename = "uploaded-source" + uploadedFileExtension(mediaType)
	}
	if !strings.Contains(filepath.Base(filename), ".") {
		filename += uploadedFileExtension(mediaType)
	}
	return filename
}

func uploadedFileExtension(mediaType string) string {
	base, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		base = mediaType
	}
	switch strings.ToLower(strings.TrimSpace(base)) {
	case "text/markdown":
		return ".md"
	case "application/json", "application/ld+json":
		return ".json"
	case "application/xml", "application/rss+xml", "application/atom+xml":
		return ".xml"
	case "text/html", "application/xhtml+xml":
		return ".html"
	case "application/pdf":
		return ".pdf"
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	default:
		return ".txt"
	}
}

func UploadedArtifactReadKind(artifact RawArtifact) string {
	if pdftext.IsPDFMediaType(artifact.MediaType) || pdftext.IsPDFBytes(artifact.Content) {
		return UploadedContentKindPDF
	}
	if isUploadedArtifactMetadataOnly(artifact) {
		return "metadata"
	}
	if isUploadedArtifactText(artifact) || utf8.Valid(artifact.Content) {
		return UploadedContentKindText
	}
	return "unsupported_binary"
}

func UploadedArtifactMetadata(artifact RawArtifact) map[string]any {
	return map[string]any{
		"artifact_id": artifact.ArtifactID,
		"mission_id":  artifact.MissionID,
		"media_type":  artifact.MediaType,
		"byte_size":   artifact.ByteSize,
		"sha256":      artifact.SHA256,
		"storage_uri": artifact.StorageURI,
		"filename":    artifact.Filename,
		"producer":    artifact.Producer,
		"created_at":  artifact.CreatedAt,
		"read_kind":   UploadedArtifactReadKind(artifact),
	}
}
