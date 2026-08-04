package web

import (
	"net/http"
	"strings"
)

// ServeHTTP는 Plasma 웹 서버의 HTTP 진입점이며, 라우팅과 프로토콜 적응만 수행하고 제품 정책의 source of truth가 되지 않는다.
func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "same-origin")

	if strings.HasPrefix(r.URL.Path, "/api/") {
		w.Header().Set("Cache-Control", "no-store")
		if !sameOriginWrite(r) {
			writeError(w, http.StatusForbidden, "cross-origin write is not allowed")
			return
		}
		if !jsonWrite(w, r) {
			return
		}
		server.serveAPI(w, r)
		return
	}
	server.serveStatic(w, r)
}

func (server *Server) serveAPI(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/health":
		server.handleHealth(w, r)
	case r.URL.Path == "/api/runtime":
		server.handleRuntime(w, r)
	case r.URL.Path == "/api/local_path/roots":
		server.handleLocalPathRoots(w, r, "")
	case strings.HasPrefix(r.URL.Path, "/api/settings/"):
		server.handleSettingsRoute(w, r)
	case r.URL.Path == "/api/missions":
		server.handleMissions(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/missions/"):
		server.handleMissionRoute(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/report_versions/"):
		server.handleReportVersionRoute(w, r)
	default:
		http.NotFound(w, r)
	}
}
