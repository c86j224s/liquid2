package web

import "net/http"

func (server *Server) handleMissionUsage(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	usage, err := server.service.MissionUsage(r.Context(), missionID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"usage": usage})
}
