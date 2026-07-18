package web

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type conversationExportRequest struct {
	Title string `json:"title"`
}

func (server *Server) handleMissionConversationExports(w http.ResponseWriter, r *http.Request, missionID string, rest []string) {
	if len(rest) != 0 {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req conversationExportRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := server.service.ExportConversation(r.Context(), app.ConversationExportRequest{
		EventID:    newID("evt"),
		ArtifactID: newID("art"),
		MissionID:  missionID,
		Title:      req.Title,
		Producer:   app.Producer{Type: "user", ID: "plasma-ui"},
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"artifact":    rawArtifactMetadata(result.Artifact),
		"event":       result.Event,
		"content":     string(result.Artifact.Content),
		"entry_count": result.EntryCount,
	})
}

func (server *Server) isReadableArtifact(ctx context.Context, missionID string, artifactID string) (bool, error) {
	if ok, err := server.isReportArtifact(ctx, missionID, artifactID); err != nil || ok {
		return ok, err
	}
	return server.isConversationExportArtifact(ctx, missionID, artifactID)
}

func (server *Server) isConversationExportArtifact(ctx context.Context, missionID string, artifactID string) (bool, error) {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return false, err
	}
	for _, event := range events {
		if event.EventType != app.ConversationExportedEvent {
			continue
		}
		var payload struct {
			Kind       string `json:"kind"`
			ArtifactID string `json:"artifact_id"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if strings.TrimSpace(payload.ArtifactID) == artifactID && strings.TrimSpace(payload.Kind) == app.ConversationExportKindMarkdown {
			return true, nil
		}
	}
	return false, nil
}
