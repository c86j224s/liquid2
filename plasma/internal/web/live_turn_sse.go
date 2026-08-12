package web

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (server *Server) handleMissionTurnLive(w http.ResponseWriter, r *http.Request, missionID, userEventID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	snapshot, updates, unsubscribe, ok := server.liveTurns.subscribe(missionID, userEventID)
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	defer unsubscribe()

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming is not supported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	if !writeLiveTurnSSE(w, flusher, snapshot) {
		return
	}
	for {
		select {
		case update, ok := <-updates:
			if !ok {
				return
			}
			if !writeLiveTurnSSE(w, flusher, update) || update.Terminal {
				return
			}
		case <-r.Context().Done():
			return
		}
	}
}

func writeLiveTurnSSE(w http.ResponseWriter, flusher http.Flusher, snapshot liveTurnSnapshot) bool {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return false
	}
	if _, err := fmt.Fprintf(w, "event: snapshot\ndata: %s\n\n", payload); err != nil {
		return false
	}
	flusher.Flush()
	return true
}
