package web

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/sources/pdftext"
)

func (err reportFailureWithPayload) Error() string {
	return err.cause.Error()
}

func (err reportFailureWithPayload) Unwrap() error {
	return err.cause
}

func (err reportFailureWithPayload) FailurePayload() map[string]any {
	return err.payload
}

func (server *Server) exportReportVersion(w http.ResponseWriter, r *http.Request, versionID string) {
	var req reportExportRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	target := strings.TrimSpace(req.Target)
	if target == "" {
		target = app.ReportExportTargetMarkdown
	}
	if !allowedReportExportTarget(target) {
		writeAppError(w, fmt.Errorf("%w: unsupported report export target", app.ErrInvalidInput))
		return
	}
	version, err := server.service.GetReportVersion(r.Context(), versionID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	if cached, ok, err := server.existingReportExport(r.Context(), version.MissionID, version.ReportVersionID, target); err != nil {
		writeAppError(w, err)
		return
	} else if ok {
		writeReportExportResponse(w, http.StatusOK, cached)
		return
	}
	approvalEvent, err := server.service.AppendEvent(r.Context(), app.BuildReportPromotionAppendRequest(app.ReportPromotionAppendRequest{
		EventID:  newID("evt"),
		Version:  version,
		Producer: app.Producer{Type: "user", ID: "plasma-ui"},
	}))
	if err != nil {
		writeAppError(w, err)
		return
	}
	if version.State != "export_candidate" {
		if _, err := server.service.PromoteReportVersion(r.Context(), app.PromoteReportVersionRequest{
			ReportVersionID: version.ReportVersionID,
			ApprovalEventID: approvalEvent.EventID,
		}); err != nil {
			writeAppError(w, err)
			return
		}
	}
	result, err := server.service.ExportReportVersion(r.Context(), app.ExportReportVersionRequest{
		ExportID:        newID("exp"),
		ReportVersionID: version.ReportVersionID,
		Target:          target,
		ArtifactID:      newID("art"),
		EventID:         newID("evt"),
		ApprovalEventID: approvalEvent.EventID,
		Producer:        app.Producer{Type: "user", ID: "plasma-ui"},
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeReportExportResponse(w, http.StatusCreated, result)
}

func (server *Server) existingReportExport(ctx context.Context, missionID string, versionID string, target string) (app.ReportExportResult, bool, error) {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return app.ReportExportResult{}, false, err
	}
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.EventType != "report.exported" {
			continue
		}
		var payload struct {
			ReportVersionID string `json:"report_version_id"`
			Target          string `json:"target"`
			ArtifactID      string `json:"artifact_id"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if strings.TrimSpace(payload.ReportVersionID) != versionID || strings.TrimSpace(payload.Target) != target {
			continue
		}
		artifactID := strings.TrimSpace(payload.ArtifactID)
		if artifactID == "" {
			continue
		}
		artifact, err := server.service.GetRawArtifact(ctx, artifactID)
		if err != nil {
			return app.ReportExportResult{}, false, err
		}
		return app.ReportExportResult{Artifact: artifact, Event: event}, true, nil
	}
	return app.ReportExportResult{}, false, nil
}

func writeReportExportResponse(w http.ResponseWriter, status int, result app.ReportExportResult) {
	writeJSON(w, status, map[string]any{
		"artifact": result.Artifact,
		"event":    result.Event,
		"content":  string(result.Artifact.Content),
	})
}

func writeRawArtifactFullPreview(w http.ResponseWriter, artifact app.RawArtifact) {
	if !utf8.Valid(artifact.Content) {
		writeAppError(w, fmt.Errorf("%w: artifact preview is not UTF-8 text", app.ErrInvalidInput))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"artifact":             rawArtifactMetadata(artifact),
		"content":              string(artifact.Content),
		"content_length":       len(artifact.Content),
		"content_length_known": true,
		"truncated":            false,
	})
}

func writeRawArtifactRead(w http.ResponseWriter, artifact app.RawArtifact, offset int, maxBytes int) {
	switch app.UploadedArtifactReadKind(artifact) {
	case app.UploadedContentKindText:
		content, normalizedOffset, nextOffset, truncated, err := boundedUTF8Content(artifact.Content, offset, maxBytes)
		if err != nil {
			writeAppError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"artifact":             rawArtifactMetadata(artifact),
			"content":              content,
			"offset":               normalizedOffset,
			"next_offset":          nextOffset,
			"content_length":       len(artifact.Content),
			"content_length_known": true,
			"truncated":            truncated,
		})
		return
	case "metadata":
		writeJSON(w, http.StatusOK, map[string]any{
			"artifact":             rawArtifactMetadata(artifact),
			"content":              "",
			"content_length":       artifact.ByteSize,
			"content_length_known": true,
			"truncated":            false,
			"metadata_only":        true,
			"message":              "binary media source read returns metadata only",
		})
		return
	}
	writeAppError(w, fmt.Errorf("%w: source artifact is not readable text", app.ErrInvalidInput))
}

func writePDFArtifactRead(w http.ResponseWriter, artifact app.RawArtifact, offset int, maxBytes int) {
	chunk, err := pdftext.ExtractChunk(artifact.Content, offset, maxBytes)
	if err != nil {
		writeAppError(w, fmt.Errorf("%w: PDF text extraction failed: %v", app.ErrInvalidInput, err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"artifact":             rawArtifactMetadata(artifact),
		"content":              chunk.Text,
		"offset":               chunk.Offset,
		"next_offset":          chunk.NextOffset,
		"content_length":       chunk.ContentLength,
		"content_length_known": chunk.ContentLengthKnown,
		"truncated":            chunk.Truncated,
		"extraction": map[string]any{
			"type":                 "pdf_text",
			"page_count":           chunk.PageCount,
			"text_length":          chunk.ContentLength,
			"text_length_known":    chunk.ContentLengthKnown,
			"suggested_read_bytes": pdftext.DefaultChunkMaxBytes,
			"max_read_bytes":       pdftext.MaxChunkBytes,
		},
	})
}

func writeRawArtifactDownload(w http.ResponseWriter, artifact app.RawArtifact) {
	mediaType := strings.TrimSpace(artifact.MediaType)
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	filename := strings.TrimSpace(artifact.Filename)
	if filename == "" {
		filename = artifact.ArtifactID
	}
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(artifact.Content)))
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(artifact.Content)
}

func writeRawArtifactHTMLPreview(w http.ResponseWriter, artifact app.RawArtifact) {
	if !isHTMLMediaType(artifact.MediaType) {
		writeError(w, http.StatusUnsupportedMediaType, "artifact is not previewable as HTML")
		return
	}
	mediaType := strings.TrimSpace(artifact.MediaType)
	filename := strings.TrimSpace(artifact.Filename)
	if filename == "" {
		filename = artifact.ArtifactID + ".html"
	}
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(artifact.Content)))
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": filename}))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "sandbox allow-scripts; default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src data: blob:; font-src data:; media-src data: blob:; connect-src 'none'; base-uri 'none'; form-action 'none'")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(artifact.Content)
}

func isHTMLMediaType(mediaType string) bool {
	base, _, err := mime.ParseMediaType(strings.TrimSpace(mediaType))
	if err != nil {
		base = mediaType
	}
	return strings.EqualFold(strings.TrimSpace(base), "text/html")
}

func rawArtifactMetadata(artifact app.RawArtifact) map[string]any {
	return app.UploadedArtifactMetadata(artifact)
}

type rawArtifactAPIResponse struct {
	ArtifactID string
	MissionID  string
	MediaType  string
	ByteSize   int64
	SHA256     string
	StorageURI string
	Filename   string
	Producer   app.Producer
	CreatedAt  time.Time
}

func rawArtifactResponse(artifact app.RawArtifact) rawArtifactAPIResponse {
	return rawArtifactAPIResponse{
		ArtifactID: artifact.ArtifactID,
		MissionID:  artifact.MissionID,
		MediaType:  artifact.MediaType,
		ByteSize:   artifact.ByteSize,
		SHA256:     artifact.SHA256,
		StorageURI: artifact.StorageURI,
		Filename:   artifact.Filename,
		Producer:   artifact.Producer,
		CreatedAt:  artifact.CreatedAt,
	}
}

func boundedUTF8Content(content []byte, offset int, maxBytes int) (string, int, int, bool, error) {
	if offset < 0 {
		return "", 0, 0, false, fmt.Errorf("%w: source artifact offset must be non-negative", app.ErrInvalidInput)
	}
	if offset > len(content) {
		return "", 0, 0, false, fmt.Errorf("%w: source artifact offset is beyond content length", app.ErrInvalidInput)
	}
	if offset < len(content) && !utf8.RuneStart(content[offset]) {
		return "", 0, 0, false, fmt.Errorf("%w: source artifact offset must align to UTF-8 boundary", app.ErrInvalidInput)
	}
	limit := maxBytes
	if limit <= 0 {
		limit = 20000
	} else if limit > 50000 {
		limit = 50000
	}
	remaining := content[offset:]
	if len(remaining) <= limit {
		return string(remaining), offset, 0, false, nil
	}
	end := offset + limit
	for end > offset && !utf8.RuneStart(content[end]) {
		end--
	}
	if end == offset {
		return "", 0, 0, false, fmt.Errorf("%w: source artifact max_bytes is too small for next UTF-8 rune", app.ErrInvalidInput)
	}
	return string(content[offset:end]), offset, end, true, nil
}

func allowedReportExportTarget(target string) bool {
	switch target {
	case app.ReportExportTargetMarkdown, app.ReportExportTargetJSONAST, app.ReportExportTargetHTML:
		return true
	default:
		return false
	}
}
