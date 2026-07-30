package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

type reportRedpenSaveRequest struct {
	Content                   string `json:"content"`
	ExpectedCurrentArtifactID string `json:"expected_current_artifact_id"`
}

func (server *Server) handleReportRedpenRoute(w http.ResponseWriter, r *http.Request, missionID string, rest []string) {
	if len(rest) == 3 && rest[2] == "download" {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		server.downloadReportRedpenWorkcopy(w, r, missionID, rest[0])
		return
	}
	if len(rest) != 2 {
		http.NotFound(w, r)
		return
	}
	source, ok := server.reportRedpenSourceArtifact(w, r, missionID, rest[0])
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		workcopy, err := server.service.GetReportRedpenWorkcopy(r.Context(), missionID, source.ArtifactID)
		if err != nil {
			writeAppError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, reportRedpenResponse(source, workcopy))
	case http.MethodPost:
		var req reportRedpenSaveRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		workcopy, err := server.service.SaveReportRedpenWorkcopy(r.Context(), app.SaveReportRedpenRequest{
			EventID:                   newID("evt"),
			ArtifactID:                newID("art"),
			NewWorkcopyID:             newID("rwc"),
			MissionID:                 missionID,
			SourceArtifactID:          source.ArtifactID,
			ExpectedCurrentArtifactID: req.ExpectedCurrentArtifactID,
			Producer:                  app.Producer{Type: "user", ID: "plasma-ui"},
			Content:                   []byte(req.Content),
		})
		if err != nil {
			writeAppError(w, err)
			return
		}
		status := http.StatusOK
		if workcopy.Changed && workcopy.Revision == 1 {
			status = http.StatusCreated
		}
		writeJSON(w, status, reportRedpenResponse(source, workcopy))
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (server *Server) downloadReportRedpenWorkcopy(w http.ResponseWriter, r *http.Request, missionID, sourceArtifactID string) {
	source, ok := server.reportRedpenSourceArtifact(w, r, missionID, sourceArtifactID)
	if !ok {
		return
	}
	workcopy, err := server.service.GetReportRedpenWorkcopy(r.Context(), missionID, source.ArtifactID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	if !workcopy.Exists {
		writeError(w, http.StatusNotFound, "redpen workcopy not found")
		return
	}
	writeRawArtifactDownload(w, reportRedpenProjectedArtifact(workcopy))
}

func (server *Server) reportRedpenSourceArtifact(w http.ResponseWriter, r *http.Request, missionID, artifactID string) (app.RawArtifact, bool) {
	artifact, err := server.service.GetRawArtifact(r.Context(), artifactID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "artifact not found")
			return app.RawArtifact{}, false
		}
		writeAppError(w, err)
		return app.RawArtifact{}, false
	}
	if artifact.MissionID != missionID || !isMarkdownMediaType(artifact.MediaType) {
		writeError(w, http.StatusNotFound, "artifact not found")
		return app.RawArtifact{}, false
	}
	eligible, err := server.isReportRedpenSource(r.Context(), missionID, artifact.ArtifactID)
	if err != nil {
		writeAppError(w, err)
		return app.RawArtifact{}, false
	}
	if !eligible {
		writeError(w, http.StatusNotFound, "artifact not found")
		return app.RawArtifact{}, false
	}
	return artifact, true
}

func (server *Server) isReportRedpenSource(ctx context.Context, missionID, artifactID string) (bool, error) {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return false, err
	}
	for _, event := range events {
		if event.EventType != "report.artifact.created" && event.EventType != "report.artifact.exported" {
			continue
		}
		var payload struct {
			ArtifactID string `json:"artifact_id"`
			Kind       string `json:"kind"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		kind := strings.TrimSpace(payload.Kind)
		if strings.TrimSpace(payload.ArtifactID) == artifactID && (kind == "markdown_report_artifact" || kind == reporting.ExportKindHumanizedMarkdown) {
			return true, nil
		}
	}
	return false, nil
}

func reportRedpenResponse(source app.RawArtifact, workcopy app.ReportRedpenWorkcopy) map[string]any {
	response := map[string]any{
		"exists":          workcopy.Exists,
		"changed":         workcopy.Changed,
		"source_artifact": rawArtifactMetadata(source),
	}
	if !workcopy.Exists {
		return response
	}
	response["workcopy"] = map[string]any{
		"workcopy_id":          workcopy.WorkcopyID,
		"source_artifact_id":   workcopy.SourceArtifactID,
		"previous_artifact_id": workcopy.PreviousArtifactID,
		"revision":             workcopy.Revision,
		"artifact":             rawArtifactMetadata(reportRedpenProjectedArtifact(workcopy)),
		"updated_at":           workcopy.Event.CreatedAt,
	}
	response["event"] = workcopy.Event
	response["content"] = string(workcopy.Artifact.Content)
	return response
}

func reportRedpenProjectedArtifact(workcopy app.ReportRedpenWorkcopy) app.RawArtifact {
	artifact := workcopy.Artifact
	artifact.MediaType = workcopy.MediaType
	artifact.Filename = workcopy.Filename
	return artifact
}
