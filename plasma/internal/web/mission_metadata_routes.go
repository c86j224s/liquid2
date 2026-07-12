package web

import (
	"net/http"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type updateMissionMetadataRequest struct {
	Title     *string           `json:"title"`
	Objective *string           `json:"objective"`
	Scope     *app.MissionScope `json:"scope"`
}

func (server *Server) handleMissionMetadataUpdate(w http.ResponseWriter, r *http.Request, missionID string) {
	var req updateMissionMetadataRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := server.service.UpdateMissionMetadata(r.Context(), app.UpdateMissionMetadataRequest{
		EventID: newID("evt"), MissionID: missionID, Producer: app.Producer{Type: "user", ID: "plasma-ui"},
		Title: req.Title, Objective: req.Objective, Scope: req.Scope,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
