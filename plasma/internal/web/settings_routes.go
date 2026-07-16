package web

import (
	"net/http"
	"strings"
)

func (server *Server) handleSettingsRoute(w http.ResponseWriter, r *http.Request) {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/settings/"), "/")
	if rest == "model-defaults" {
		server.handleSettingsModelDefaults(w, r)
		return
	}
	parts := strings.Split(rest, "/")
	if len(parts) < 2 || parts[0] != "connectors" || parts[1] != "confluence" {
		http.NotFound(w, r)
		return
	}
	server.handleSettingsConfluence(w, r, parts[2:])
}

func (server *Server) handleSettingsConfluence(w http.ResponseWriter, r *http.Request, rest []string) {
	if len(rest) == 1 && rest[0] == "connections" {
		server.handleConfluenceConnections(w, r, "")
		return
	}
	if len(rest) == 2 && rest[0] == "connections" {
		server.handleConfluenceConnection(w, r, "", rest[1])
		return
	}
	if len(rest) == 3 && rest[0] == "connections" && rest[2] == "revoke" {
		server.handleConfluenceConnectionRevoke(w, r, "", rest[1])
		return
	}
	if len(rest) == 3 && rest[0] == "connections" && rest[2] == "sites" {
		server.handleSettingsConfluenceConnectionSites(w, r, rest[1])
		return
	}
	if len(rest) == 4 && rest[0] == "connections" && rest[2] == "sites" && rest[3] == "refresh" {
		server.handleSettingsConfluenceConnectionSitesRefresh(w, r, rest[1])
		return
	}
	if len(rest) == 2 && rest[0] == "oauth" {
		switch rest[1] {
		case "start":
			writeAppError(w, errConfluenceOAuthUnsupported())
			return
		case "callback":
			writeAppError(w, errConfluenceOAuthUnsupported())
			return
		}
	}
	http.NotFound(w, r)
}
