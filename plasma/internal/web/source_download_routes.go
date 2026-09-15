package web

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strings"
	"unicode"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
)

func (server *Server) handleSourceCandidateDownload(w http.ResponseWriter, r *http.Request, missionID, stagedEventID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	artifactID := strings.TrimSpace(r.URL.Query().Get("artifact_id"))
	artifact, err := server.service.ResolveSourceCandidateDownloadByIDs(r.Context(), missionID, stagedEventID, artifactID)
	if err != nil {
		writeSourceDownloadError(w, err)
		return
	}
	writeStoredArtifactDownload(w, artifact)
}

func (server *Server) handleSourceSnapshotDownload(w http.ResponseWriter, r *http.Request, missionID, snapshotID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	artifact, err := server.service.ResolveSourceSnapshotDownloadByIDs(r.Context(), missionID, snapshotID, strings.TrimSpace(r.URL.Query().Get("artifact_id")))
	if err != nil {
		writeSourceDownloadError(w, err)
		return
	}
	writeStoredArtifactDownload(w, artifact)
}

func writeSourceDownloadError(w http.ResponseWriter, err error) {
	if errors.Is(err, app.ErrSourceDownloadNotFound) {
		writeError(w, http.StatusNotFound, "source artifact not found")
		return
	}
	writeAppError(w, err)
}

func writeStoredArtifactDownload(w http.ResponseWriter, artifact artifactcontract.Raw) {
	mediaType := strings.TrimSpace(artifact.MediaType)
	if mediaType == "" || strings.ContainsAny(mediaType, "\r\n\x00") {
		mediaType = "application/octet-stream"
	} else if parsed, _, err := mime.ParseMediaType(mediaType); err != nil || strings.TrimSpace(parsed) == "" || strings.ContainsAny(parsed, "\r\n\x00") {
		mediaType = "application/octet-stream"
	}
	filename := safeStoredArtifactFilename(artifact.Filename, artifact.ArtifactID)
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(artifact.Content)))
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(artifact.Content)
}

func safeStoredArtifactFilename(raw, artifactID string) string {
	name := sanitizeStoredArtifactFilename(raw)
	if name == "" {
		name = sanitizeStoredArtifactFilename(artifactID)
	}
	if name == "" {
		name = "download"
	}
	const maxFilenameRunes = 120
	runes := []rune(name)
	if len(runes) > maxFilenameRunes {
		ext := ""
		if dot := strings.LastIndex(name, "."); dot > 0 && dot < len(name)-1 {
			ext = name[dot:]
		}
		if ext != "" && len([]rune(ext)) < maxFilenameRunes-1 {
			name = string(runes[:maxFilenameRunes-len([]rune(ext))-1]) + "_" + ext
		} else {
			name = string(runes[:maxFilenameRunes])
		}
	}
	return name
}

func sanitizeStoredArtifactFilename(raw string) string {
	var builder strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		switch {
		case r == '/' || r == '\\' || r == '"' || unicode.IsControl(r) || unicode.IsSpace(r) || unicode.In(r, unicode.Cf):
			builder.WriteByte('_')
		default:
			builder.WriteRune(r)
		}
	}
	name := strings.Trim(builder.String(), " ._")
	if name == "." || name == ".." {
		return ""
	}
	return name
}
