package ingest

import (
	"bytes"
	"net/http"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const MaxUploadBytes = 1024 * 1024

type UploadInput struct {
	Title       string
	Filename    string
	ContentType string
	Data        []byte
}

type PreparedUpload struct {
	Title    string
	Filename string
	MimeType string
	Data     []byte
	Content  string
	Format   string
}

func PrepareUpload(input UploadInput) (PreparedUpload, error) {
	filename := strings.TrimSpace(filepath.Base(input.Filename))
	if filename == "" || filename == "." {
		return PreparedUpload{}, unsupportedMedia("filename is required")
	}
	if len(input.Data) > MaxUploadBytes {
		return PreparedUpload{}, payloadTooLarge("file exceeds 1MB")
	}
	if len(input.Data) == 0 {
		return PreparedUpload{}, unsupportedMedia("file is empty")
	}
	mimeType, err := validatedUploadMime(filename, input.ContentType, input.Data)
	if err != nil {
		return PreparedUpload{}, err
	}
	upload := PreparedUpload{
		Title: strings.TrimSpace(input.Title), Filename: filename,
		MimeType: mimeType, Data: append([]byte(nil), input.Data...),
	}
	if extracted, ok := extractUploadText(mimeType, input.Data); ok {
		upload.Content = extracted.Content
		upload.Format = extracted.Format
	}
	return upload, nil
}

func allowedUploadMime(mimeType string) bool {
	switch mimeType {
	case "text/plain", "text/markdown", "text/html", "application/pdf":
		return true
	default:
		return false
	}
}

func validatedUploadMime(filename string, contentType string, data []byte) (string, error) {
	declaredType := mediaType(contentType)
	extensionType := uploadMimeFromExtension(filename)
	if declaredType != "" && declaredType != "application/octet-stream" {
		if !allowedUploadMime(declaredType) {
			return "", unsupportedMedia("file type is not supported")
		}
		if extensionType != "" && !compatibleUploadMime(declaredType, extensionType) {
			return "", unsupportedMedia("file extension does not match content type")
		}
		if bytes.HasPrefix(data, []byte("%PDF-")) && declaredType != "application/pdf" {
			return "", unsupportedMedia("pdf content must use application/pdf")
		}
		if !validUploadPayload(declaredType, data) {
			return "", unsupportedMedia("file content does not match content type")
		}
		return declaredType, nil
	}

	mimeType := extensionType
	if mimeType == "" {
		mimeType = mediaType(http.DetectContentType(data))
	}
	if !allowedUploadMime(mimeType) {
		return "", unsupportedMedia("file type is not supported")
	}
	if !validUploadPayload(mimeType, data) {
		return "", unsupportedMedia("file content does not match content type")
	}
	return mimeType, nil
}

func uploadMimeFromExtension(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".txt":
		return "text/plain"
	case ".md", ".markdown":
		return "text/markdown"
	case ".html", ".htm":
		return "text/html"
	case ".pdf":
		return "application/pdf"
	default:
		return ""
	}
}

func compatibleUploadMime(declaredType string, extensionType string) bool {
	return declaredType == extensionType ||
		(declaredType == "text/plain" && strings.HasPrefix(extensionType, "text/"))
}

func validUploadPayload(mimeType string, data []byte) bool {
	switch mimeType {
	case "application/pdf":
		return bytes.HasPrefix(data, []byte("%PDF-"))
	case "text/plain", "text/markdown", "text/html":
		return utf8.Valid(data) && !bytes.Contains(data, []byte{0})
	default:
		return false
	}
}

func extractUploadText(mimeType string, data []byte) (ExtractedContent, bool) {
	switch mimeType {
	case "text/plain", "text/markdown", "text/html":
		extracted, err := Extract(mimeType, data)
		return extracted, err == nil
	default:
		return ExtractedContent{}, false
	}
}
