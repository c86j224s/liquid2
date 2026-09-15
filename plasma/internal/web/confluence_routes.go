package web

import (
	"context"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/confluenceaccess"
	"github.com/c86j224s/liquid2/plasma/internal/source/confluencesource"
	"github.com/c86j224s/liquid2/plasma/internal/source/liquid2source"
	"net/http"
	"net/url"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	confluenceconnector "github.com/c86j224s/liquid2/plasma/internal/connectors/confluence"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/sourcecandidateevents"
	"github.com/c86j224s/liquid2/plasma/internal/sourceingest"
)

func errConfluenceOAuthUnsupported() error {
	return fmt.Errorf("%w: Confluence OAuth is disabled in Plasma 0.0. Use an API token connection instead.", app.ErrInvalidInput)
}

func (server *Server) confluenceClient(ctx context.Context, connectionID string, cloudID string) (confluencesource.ConfluenceSourceConnector, error) {
	connection, err := server.confluenceConnectionForUse(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return server.confluenceClientForConnection(connection, cloudID)
}

func (server *Server) confluenceClientForConnection(connection confluenceaccess.Connection, cloudID string) (confluencesource.ConfluenceSourceConnector, error) {
	cloudID = strings.TrimSpace(cloudID)
	if cloudID == "" {
		return nil, fmt.Errorf("%w: confluence cloud id is required", app.ErrInvalidInput)
	}
	effectiveSiteURL := webConfluenceCachedSiteURL(connection, cloudID)
	baseURL := strings.TrimSpace(server.confluenceAPIBaseURL)
	options := []confluenceconnector.Option{}
	switch connection.AuthType {
	case confluenceaccess.AuthOAuth:
		return nil, errConfluenceOAuthUnsupported()
	case confluenceaccess.AuthAPIToken:
		if effectiveSiteURL == "" {
			return nil, fmt.Errorf("%w: confluence site URL is required for api_token connections", app.ErrInvalidInput)
		}
		normalizedSiteURL, err := confluenceaccess.NormalizeConfluenceAPITokenSiteURL(effectiveSiteURL)
		if err != nil {
			return nil, err
		}
		effectiveSiteURL = normalizedSiteURL
		if baseURL == "" {
			baseURL = webConfluenceWikiBaseURL(effectiveSiteURL)
		} else {
			normalizedBaseURL, err := confluenceaccess.NormalizeConfluenceAPITokenAPIBaseURLForSite(baseURL, effectiveSiteURL)
			if err != nil {
				return nil, err
			}
			baseURL = normalizedBaseURL
		}
		options = append(options, confluenceconnector.WithSiteURL(effectiveSiteURL))
		options = append(options, confluenceconnector.WithBasicAuth(connection.AccountName, connection.AccessToken))
	default:
		return nil, fmt.Errorf("%w: unsupported confluence auth type", app.ErrInvalidInput)
	}
	return confluenceconnector.NewClient(baseURL, cloudID, options...)
}

func (server *Server) confluenceBrowserConnector(ctx context.Context, connectionID string, cloudID string) (confluencesource.ConfluenceBrowserConnector, error) {
	connector, err := server.confluenceClient(ctx, connectionID, cloudID)
	if err != nil {
		return nil, err
	}
	browser, ok := connector.(confluencesource.ConfluenceBrowserConnector)
	if !ok {
		return nil, fmt.Errorf("%w: confluence browser connector is required", app.ErrInvalidInput)
	}
	return browser, nil
}

func (server *Server) confluenceConnectionForUse(ctx context.Context, connectionID string) (confluenceaccess.Connection, error) {
	connection, err := server.service.GetConfluenceConnection(ctx, connectionID)
	if err != nil {
		return confluenceaccess.Connection{}, err
	}
	if connection.Revoked {
		return confluenceaccess.Connection{}, confluencesource.NewConfluenceValidationError(
			confluencesource.ConfluenceErrorCodeRevoked,
			"Confluence 연결이 로컬에서 해제되었습니다. 다시 연결하거나 다른 연결을 선택하세요.",
		)
	}
	if connection.AuthType == confluenceaccess.AuthOAuth {
		return confluenceaccess.Connection{}, errConfluenceOAuthUnsupported()
	}
	return connection, nil
}

func webConfluenceDiscoveryClient(connection confluenceaccess.Connection, discoveryURL string) (*confluenceconnector.DiscoveryClient, error) {
	if connection.AuthType != confluenceaccess.AuthOAuth {
		return nil, fmt.Errorf("%w: confluence site discovery requires an oauth connection", app.ErrInvalidInput)
	}
	options := []confluenceconnector.DiscoveryOption{confluenceconnector.WithDiscoveryBearerToken(connection.AccessToken)}
	if strings.TrimSpace(discoveryURL) != "" {
		options = append(options, confluenceconnector.WithDiscoveryBaseURL(discoveryURL))
	}
	return confluenceconnector.NewDiscoveryClient(options...)
}

func webConfluenceCachedSiteURL(connection confluenceaccess.Connection, cloudID string) string {
	cloudID = strings.TrimSpace(cloudID)
	for _, site := range connection.Sites {
		if strings.TrimSpace(site.CloudID) == cloudID {
			return strings.TrimSpace(site.URL)
		}
	}
	return ""
}

func webConfluenceWikiBaseURL(siteURL string) string {
	siteURL = strings.TrimRight(strings.TrimSpace(siteURL), "/")
	if strings.HasSuffix(siteURL, "/wiki") {
		return siteURL
	}
	return siteURL + "/wiki"
}

func (server *Server) handleLiquid2Search(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if server.liquid2 == nil {
		writeError(w, http.StatusNotImplemented, "liquid2 connector is not configured")
		return
	}
	var req liquid2SearchRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := server.service.SearchLiquid2Sources(r.Context(), server.liquid2, liquid2source.Liquid2SourceSearchRequest{
		MissionID: missionID,
		Query:     req.Query,
		Limit:     req.Limit,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (server *Server) handleLiquid2Snapshot(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if server.liquid2 == nil {
		writeError(w, http.StatusNotImplemented, "liquid2 connector is not configured")
		return
	}
	var req liquid2SnapshotRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	externalSourceID := strings.TrimSpace(req.ExternalSourceID)
	if externalSourceID == "" {
		writeError(w, http.StatusBadRequest, "external_source_id is required")
		return
	}
	result, err := server.service.SnapshotLiquid2SourceWithEvent(r.Context(), server.liquid2, liquid2source.SnapshotLiquid2SourceWithEventRequest{
		Snapshot: liquid2source.SnapshotLiquid2SourceRequest{
			MissionID:        missionID,
			ArtifactID:       newID("art"),
			SnapshotID:       newID("src"),
			ExternalSourceID: externalSourceID,
			Reason:           req.Reason,
		},
		EventID:  newID("evt"),
		Producer: ledger.Producer{Type: "user", ID: "plasma-ui"},
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (server *Server) confluenceOAuthServerConfigured() bool {
	return strings.TrimSpace(server.confluenceOAuth.ClientID) != "" && strings.TrimSpace(server.confluenceOAuth.ClientSecret) != ""
}

func writeConfluenceMissionLifecycleDeprecated(w http.ResponseWriter) {
	writeError(w, http.StatusGone, "Confluence connection lifecycle moved to Settings. Use /api/settings/connectors/confluence routes for API token registration, rename, revoke, delete, and stored site lookup.")
}

func (server *Server) handleConfluenceConnections(w http.ResponseWriter, r *http.Request, missionID string) {
	switch r.Method {
	case http.MethodGet:
		connections, err := server.service.ListConfluenceConnections(r.Context())
		if err != nil {
			writeAppError(w, err)
			return
		}
		response := map[string]any{
			"connections":      connections,
			"api_token_only":   true,
			"oauth_configured": false,
		}
		if strings.TrimSpace(missionID) != "" {
			response["mission_id"] = missionID
		}
		writeJSON(w, http.StatusOK, response)
	case http.MethodPost:
		if strings.TrimSpace(missionID) != "" {
			writeConfluenceMissionLifecycleDeprecated(w)
			return
		}
		var req confluenceConnectionRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		connectionID := strings.TrimSpace(req.ConnectionID)
		if connectionID == "" {
			connectionID = newID("cnf")
		}
		authType := strings.TrimSpace(req.AuthType)
		if authType == "" {
			authType = confluenceaccess.AuthAPIToken
		}
		token := strings.TrimSpace(req.AccessToken)
		if strings.TrimSpace(req.APIToken) != "" {
			authType = confluenceaccess.AuthAPIToken
			token = strings.TrimSpace(req.APIToken)
		}
		if authType == confluenceaccess.AuthOAuth {
			writeAppError(w, errConfluenceOAuthUnsupported())
			return
		}
		if authType != confluenceaccess.AuthAPIToken {
			writeAppError(w, fmt.Errorf("%w: unsupported confluence auth type", app.ErrInvalidInput))
			return
		}
		if len(req.Sites) == 0 {
			writeAppError(w, fmt.Errorf("%w: confluence API token connection requires a site URL", app.ErrInvalidInput))
			return
		}
		expiresAt, err := parseOptionalRFC3339(req.ExpiresAt)
		if err != nil {
			writeAppError(w, err)
			return
		}
		connection, err := server.service.UpsertConfluenceConnection(r.Context(), confluenceaccess.UpsertRequest{
			ConnectionID:   connectionID,
			DisplayName:    req.DisplayName,
			AuthType:       authType,
			AccountID:      req.AccountID,
			AccountName:    req.AccountName,
			AccessToken:    token,
			RefreshToken:   req.RefreshToken,
			TokenExpiresAt: expiresAt,
			Scopes:         req.Scopes,
			Sites:          req.Sites,
		})
		if err != nil {
			writeAppError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"connection": connection})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (server *Server) handleConfluenceConnection(w http.ResponseWriter, r *http.Request, missionID string, connectionID string) {
	switch r.Method {
	case http.MethodPatch:
		var req confluenceConnectionUpdateRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		connection, err := server.service.RenameConfluenceConnection(r.Context(), connectionID, req.DisplayName)
		if err != nil {
			writeAppError(w, err)
			return
		}
		response := map[string]any{"connection": connection}
		if strings.TrimSpace(missionID) != "" {
			response["mission_id"] = missionID
		}
		writeJSON(w, http.StatusOK, response)
	case http.MethodDelete:
		if err := server.service.DeleteConfluenceConnection(r.Context(), connectionID); err != nil {
			writeAppError(w, err)
			return
		}
		response := map[string]any{"connection_id": connectionID, "deleted": true}
		if strings.TrimSpace(missionID) != "" {
			response["mission_id"] = missionID
		}
		writeJSON(w, http.StatusOK, response)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (server *Server) handleConfluenceConnectionRevoke(w http.ResponseWriter, r *http.Request, missionID string, connectionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !decodeJSON(w, r, &struct{}{}) {
		return
	}
	connection, err := server.service.RevokeConfluenceConnection(r.Context(), connectionID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response := map[string]any{"connection": connection, "revoked": true}
	if strings.TrimSpace(missionID) != "" {
		response["mission_id"] = missionID
	}
	writeJSON(w, http.StatusOK, response)
}

func (server *Server) handleConfluenceSites(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if strings.TrimSpace(missionID) != "" && r.Method == http.MethodPost {
		writeConfluenceMissionLifecycleDeprecated(w)
		return
	}
	req := confluenceSitesRequest{}
	if r.Method == http.MethodPost {
		if !decodeJSON(w, r, &req) {
			return
		}
	} else {
		req.ConnectionID = r.URL.Query().Get("connection_id")
	}
	connection, err := server.confluenceConnectionForUse(r.Context(), req.ConnectionID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	if req.Refresh || r.Method == http.MethodPost {
		lister, err := webConfluenceDiscoveryClient(connection, server.confluenceOAuthDiscoveryURL)
		if err != nil {
			writeAppError(w, err)
			return
		}
		connection, err = server.service.RefreshConfluenceConnectionSites(r.Context(), connection.ConnectionID, lister)
		if err != nil {
			writeAppError(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"mission_id": missionID, "connection": connection, "sites": connection.Sites})
}

func (server *Server) handleSettingsConfluenceConnectionSites(w http.ResponseWriter, r *http.Request, connectionID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	connection, err := server.service.GetConfluenceConnection(r.Context(), connectionID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"connection_id": connection.ConnectionID, "sites": connection.Sites})
}

func (server *Server) handleSettingsConfluenceConnectionSitesRefresh(w http.ResponseWriter, r *http.Request, connectionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !decodeJSON(w, r, &struct{}{}) {
		return
	}
	connection, err := server.confluenceConnectionForUse(r.Context(), connectionID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	lister, err := webConfluenceDiscoveryClient(connection, server.confluenceOAuthDiscoveryURL)
	if err != nil {
		writeAppError(w, err)
		return
	}
	connection, err = server.service.RefreshConfluenceConnectionSites(r.Context(), connection.ConnectionID, lister)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"connection": connection, "sites": connection.Sites})
}

func (server *Server) handleConfluenceSpaces(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	req := confluenceSpacesRequest{}
	if r.Method == http.MethodPost {
		if !decodeJSON(w, r, &req) {
			return
		}
	} else {
		req.ConnectionID = r.URL.Query().Get("connection_id")
		req.CloudID = r.URL.Query().Get("cloud_id")
		req.Limit = queryInt(r, "limit")
		req.Cursor = r.URL.Query().Get("cursor")
	}
	connector, err := server.confluenceBrowserConnector(r.Context(), req.ConnectionID, req.CloudID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result, err := server.service.ListConfluenceSpaces(r.Context(), connector, confluencesource.ConfluenceSpaceListRequest{
		MissionID: missionID,
		CloudID:   req.CloudID,
		Limit:     req.Limit,
		Cursor:    req.Cursor,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (server *Server) handleConfluenceSpacePages(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	req := confluenceSpacePagesRequest{}
	if r.Method == http.MethodPost {
		if !decodeJSON(w, r, &req) {
			return
		}
	} else {
		req.ConnectionID = r.URL.Query().Get("connection_id")
		req.CloudID = r.URL.Query().Get("cloud_id")
		req.SpaceID = r.URL.Query().Get("space_id")
		req.Limit = queryInt(r, "limit")
		req.Cursor = r.URL.Query().Get("cursor")
	}
	connector, err := server.confluenceBrowserConnector(r.Context(), req.ConnectionID, req.CloudID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result, err := server.service.ListConfluenceSpacePages(r.Context(), connector, confluencesource.ConfluenceSpacePagesRequest{
		MissionID: missionID,
		CloudID:   req.CloudID,
		SpaceID:   req.SpaceID,
		Limit:     req.Limit,
		Cursor:    req.Cursor,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (server *Server) handleConfluencePageChildren(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	req := confluencePageChildrenRequest{}
	if r.Method == http.MethodPost {
		if !decodeJSON(w, r, &req) {
			return
		}
	} else {
		req.ConnectionID = r.URL.Query().Get("connection_id")
		req.CloudID = r.URL.Query().Get("cloud_id")
		req.PageID = r.URL.Query().Get("page_id")
		req.Limit = queryInt(r, "limit")
		req.Cursor = r.URL.Query().Get("cursor")
	}
	connector, err := server.confluenceBrowserConnector(r.Context(), req.ConnectionID, req.CloudID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result, err := server.service.ListConfluencePageChildren(r.Context(), connector, confluencesource.ConfluencePageChildrenRequest{
		MissionID: missionID,
		CloudID:   req.CloudID,
		PageID:    req.PageID,
		Limit:     req.Limit,
		Cursor:    req.Cursor,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (server *Server) handleConfluenceSearch(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req confluenceSearchRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	connector, err := server.confluenceClient(r.Context(), req.ConnectionID, req.CloudID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result, err := server.service.SearchConfluenceSources(r.Context(), connector, confluencesource.ConfluenceSourceSearchRequest{
		MissionID: missionID,
		CloudID:   req.CloudID,
		Query:     req.Query,
		Limit:     req.Limit,
		Cursor:    req.Cursor,
		SpaceKey:  req.SpaceKey,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (server *Server) handleConfluencePreview(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req confluenceSnapshotRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	connector, err := server.confluenceClient(r.Context(), req.ConnectionID, req.CloudID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result, err := server.service.PreviewConfluenceSource(r.Context(), connector, app.ConfluenceSourcePreviewRequest{
		MissionID:       missionID,
		CloudID:         req.CloudID,
		PageID:          req.PageID,
		ExpectedVersion: req.ExpectedVersion,
		MaxBodyBytes:    req.MaxBodyBytes,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (server *Server) handleConfluenceSnapshot(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req confluenceSnapshotRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	connector, err := server.confluenceClient(r.Context(), req.ConnectionID, req.CloudID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result, err := server.service.SnapshotConfluenceSourceWithEvent(r.Context(), connector, app.SnapshotConfluenceSourceWithEventRequest{
		Snapshot: app.SnapshotConfluenceSourceRequest{
			MissionID:       missionID,
			ArtifactID:      newID("art"),
			SnapshotID:      newID("src"),
			CloudID:         req.CloudID,
			PageID:          req.PageID,
			ExpectedVersion: req.ExpectedVersion,
			MaxBodyBytes:    req.MaxBodyBytes,
			Range: confluencesource.ConfluenceRangeSelection{
				ContentID: req.RangeContentID,
				Start:     req.RangeStart,
				End:       req.RangeEnd,
			},
			Reason: req.Reason,
		},
		EventID:  newID("evt"),
		Producer: ledger.Producer{Type: "user", ID: "plasma-ui"},
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (server *Server) handleConfluenceURLSnapshot(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req confluenceURLSnapshotRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	normalizedURL, err := normalizedHTTPURL(req.URL)
	if err != nil {
		writeAppError(w, err)
		return
	}
	unlock := server.sources.lock(missionID + "\x00" + normalizedURL)
	defer unlock()
	target, ok, err := parseConfluencePageURL(normalizedURL)
	if !ok {
		writeAppError(w, fmt.Errorf("%w: Confluence page URL is required", app.ErrInvalidInput))
		return
	}
	if err != nil {
		writeAppError(w, err)
		return
	}
	if existing, ok, err := sourceingest.ExistingSourceSnapshotForURL(r.Context(), server.service, missionID, normalizedURL); err != nil {
		writeAppError(w, err)
		return
	} else if ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"existing": true,
			"snapshot": existing,
		})
		return
	}
	provenance, err := server.stagedConfluenceCandidateProvenance(r.Context(), missionID, normalizedURL)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result, err := server.snapshotConfluenceURLSourceWithSelection(r.Context(), missionID, target, req.ConnectionID, req.CloudID, req.Title, provenance)
	if err != nil {
		server.recordSourceSnapshotFailure(r.Context(), missionID, "confluence_url", normalizedURL, err)
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

type stagedSourceCandidateProvenance struct {
	ProposalEventID string
	URL             string
}

func (server *Server) snapshotConfluenceURLSource(ctx context.Context, missionID string, target confluencePageURLTarget, title string) (app.ConfluenceSnapshotWithEventResult, error) {
	return server.snapshotConfluenceURLSourceWithSelection(ctx, missionID, target, "", "", title, stagedSourceCandidateProvenance{})
}

func (server *Server) snapshotConfluenceURLSourceWithSelection(ctx context.Context, missionID string, target confluencePageURLTarget, connectionID string, cloudID string, title string, provenance stagedSourceCandidateProvenance) (app.ConfluenceSnapshotWithEventResult, error) {
	connection, err := server.confluenceConnectionForPageURL(ctx, target, connectionID, cloudID)
	if err != nil {
		return app.ConfluenceSnapshotWithEventResult{}, err
	}
	connector, err := server.confluenceClientForConnection(connection, target.CloudID)
	if err != nil {
		return app.ConfluenceSnapshotWithEventResult{}, err
	}
	version, err := confluenceVersionForURLSnapshot(ctx, connector, target)
	if err != nil {
		return app.ConfluenceSnapshotWithEventResult{}, err
	}
	if version.SiteURL != "" && webConfluenceURLHost(version.SiteURL) != "" && webConfluenceURLHost(version.SiteURL) != webConfluenceURLHost(target.SiteURL) {
		return app.ConfluenceSnapshotWithEventResult{}, fmt.Errorf("%w: Confluence page가 붙여넣은 URL의 site와 일치하지 않습니다. 페이지와 연결을 다시 확인하세요.", app.ErrInvalidInput)
	}
	request := app.SnapshotConfluenceSourceWithEventRequest{
		Snapshot: app.SnapshotConfluenceSourceRequest{
			MissionID: missionID, ArtifactID: newID("art"), SnapshotID: newID("src"),
			CloudID: target.CloudID, PageID: target.PageID, Title: title,
			ExpectedVersion: version.Version, Reason: "Confluence URL에서 직접 추가",
		},
		EventID: newID("evt"), Producer: ledger.Producer{Type: "user", ID: "plasma-ui"},
	}
	request.SourceCandidateProposalEventID = strings.TrimSpace(provenance.ProposalEventID)
	request.SourceCandidateURL = strings.TrimSpace(provenance.URL)
	return server.service.SnapshotConfluenceSourceWithEvent(ctx, connector, request)
}

func confluenceVersionForURLSnapshot(ctx context.Context, connector confluencesource.ConfluenceSourceConnector, target confluencePageURLTarget) (confluencesource.ConfluenceSourceVersion, error) {
	req := confluencesource.ConfluenceSourceReadRequest{CloudID: target.CloudID, PageID: target.PageID}
	var version confluencesource.ConfluenceSourceVersion
	var err error
	if versionConnector, ok := connector.(confluencesource.ConfluenceSourceVersionConnector); ok {
		version, err = versionConnector.GetConfluenceSourceVersion(ctx, req)
	} else {
		var page confluencesource.ConfluenceSourcePage
		page, err = connector.ReadConfluenceSource(ctx, req)
		version = confluencesource.ConfluenceSourceVersion{
			Connector: page.Connector,
			CloudID:   page.CloudID,
			SiteURL:   page.SiteURL,
			PageID:    page.PageID,
			SpaceID:   page.SpaceID,
			SpaceKey:  page.SpaceKey,
			Title:     page.Title,
			WebURL:    page.WebURL,
			Version:   page.Version,
			UpdatedAt: page.UpdatedAt,
		}
	}
	if err != nil {
		return confluencesource.ConfluenceSourceVersion{}, err
	}
	if strings.TrimSpace(version.CloudID) != "" && strings.TrimSpace(version.CloudID) != target.CloudID {
		return confluencesource.ConfluenceSourceVersion{}, fmt.Errorf("%w: Confluence page가 붙여넣은 URL의 site와 일치하지 않습니다. 페이지와 연결을 다시 확인하세요.", app.ErrInvalidInput)
	}
	if strings.TrimSpace(version.PageID) != "" && strings.TrimSpace(version.PageID) != target.PageID {
		return confluencesource.ConfluenceSourceVersion{}, fmt.Errorf("%w: Confluence page id가 붙여넣은 URL과 일치하지 않습니다. URL을 다시 확인하세요.", app.ErrInvalidInput)
	}
	if version.Version <= 0 {
		return confluencesource.ConfluenceSourceVersion{}, fmt.Errorf("%w: Confluence page version을 확인할 수 없습니다. Sources의 Confluence 검색에서 page를 찾아 추가하세요.", app.ErrInvalidInput)
	}
	return version, nil
}

func (server *Server) confluenceConnectionForPageURL(ctx context.Context, target confluencePageURLTarget, requestedConnectionID string, requestedCloudID string) (confluenceaccess.Connection, error) {
	requestedConnectionID = strings.TrimSpace(requestedConnectionID)
	requestedCloudID = strings.TrimSpace(requestedCloudID)
	if requestedCloudID != "" && requestedCloudID != target.CloudID {
		return confluenceaccess.Connection{}, fmt.Errorf("%w: 선택한 Confluence site와 붙여넣은 URL의 site가 일치하지 않습니다. 연결과 URL을 다시 확인하세요.", app.ErrInvalidInput)
	}
	connections, err := server.service.ListConfluenceConnections(ctx)
	if err != nil {
		return confluenceaccess.Connection{}, err
	}
	matches := []confluenceaccess.Connection{}
	for _, connection := range connections {
		if connection.Revoked || connection.AuthType != confluenceaccess.AuthAPIToken {
			continue
		}
		if requestedConnectionID != "" && connection.ConnectionID != requestedConnectionID {
			continue
		}
		for _, site := range connection.Sites {
			siteURL, err := confluenceaccess.NormalizeConfluenceAPITokenSiteURL(site.URL)
			if err != nil {
				continue
			}
			siteCloudID, err := confluenceaccess.ConfluenceAPITokenSiteCloudID(siteURL)
			if err != nil {
				continue
			}
			if siteCloudID == target.CloudID && webConfluenceURLHost(siteURL) == webConfluenceURLHost(target.SiteURL) {
				matches = append(matches, connection)
				break
			}
		}
	}
	switch len(matches) {
	case 0:
		if requestedConnectionID != "" || requestedCloudID != "" {
			return confluenceaccess.Connection{}, fmt.Errorf("%w: 선택한 Confluence 연결/site와 붙여넣은 URL이 일치하지 않습니다. 연결과 URL을 다시 확인하세요.", app.ErrInvalidInput)
		}
		return confluenceaccess.Connection{}, fmt.Errorf("%w: 이 Confluence site에 맞는 API token 연결이 없습니다. Settings에서 %s 연결을 만든 뒤 다시 추가하세요.", app.ErrInvalidInput, target.SiteURL)
	case 1:
		return matches[0], nil
	default:
		return confluenceaccess.Connection{}, fmt.Errorf("%w: 이 Confluence site에 맞는 API token 연결이 둘 이상입니다. Sources의 Confluence 영역에서 연결을 직접 선택해 page를 추가하세요.", app.ErrInvalidInput)
	}
}

func (server *Server) handleConfluenceCheckUpdate(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req confluenceUpdateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	connector, err := server.confluenceUpdateConnector(r.Context(), missionID, req.ConnectionID, req.SnapshotID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result, err := server.service.CheckConfluenceSourceUpdateWithEvent(r.Context(), connector, app.CheckConfluenceSourceUpdateRequest{
		MissionID:  missionID,
		SnapshotID: req.SnapshotID,
		EventID:    newID("evt"),
		Producer:   ledger.Producer{Type: "user", ID: "plasma-ui"},
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (server *Server) handleConfluenceUpdatePreview(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req confluenceUpdateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	connector, err := server.confluenceUpdateConnector(r.Context(), missionID, req.ConnectionID, req.SnapshotID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result, err := server.service.PreviewConfluenceSourceUpdate(r.Context(), connector, app.ConfluenceUpdatePreviewRequest{
		MissionID:       missionID,
		SnapshotID:      req.SnapshotID,
		ExpectedVersion: req.ExpectedVersion,
		MaxBodyBytes:    req.MaxBodyBytes,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (server *Server) handleConfluenceUpdate(w http.ResponseWriter, r *http.Request, missionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req confluenceUpdateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	connector, err := server.confluenceUpdateConnector(r.Context(), missionID, req.ConnectionID, req.SnapshotID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result, err := server.service.UpdateConfluenceSourceWithEvent(r.Context(), connector, app.UpdateConfluenceSourceRequest{
		MissionID:          missionID,
		PreviousSnapshotID: req.SnapshotID,
		ArtifactID:         newID("art"),
		SnapshotID:         newID("src"),
		ExpectedVersion:    req.ExpectedVersion,
		MaxBodyBytes:       req.MaxBodyBytes,
		Range: confluencesource.ConfluenceRangeSelection{
			ContentID: req.RangeContentID,
			Start:     req.RangeStart,
			End:       req.RangeEnd,
		},
		Reason:          req.Reason,
		SnapshotEventID: newID("evt"),
		UpdateEventID:   newID("evt"),
		Producer:        ledger.Producer{Type: "user", ID: "plasma-ui"},
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func parseConfluencePageURL(rawURL string) (confluencePageURLTarget, bool, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return confluencePageURLTarget{}, false, nil
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "atlassian.net" && !strings.HasSuffix(host, ".atlassian.net") {
		return confluencePageURLTarget{}, false, nil
	}
	siteURL := "https://" + host + "/wiki"
	cloudID, err := confluenceaccess.ConfluenceAPITokenSiteCloudID(siteURL)
	if err != nil {
		return confluencePageURLTarget{}, true, err
	}
	pageID := confluencePageIDFromURL(parsed)
	if pageID == "" {
		return confluencePageURLTarget{}, true, fmt.Errorf("%w: Confluence page URL에서 page id를 확인할 수 없습니다. Sources의 Confluence 검색에서 page를 찾아 추가하세요.", app.ErrInvalidInput)
	}
	return confluencePageURLTarget{
		RawURL:  rawURL,
		SiteURL: siteURL,
		CloudID: cloudID,
		PageID:  pageID,
	}, true, nil
}

func (server *Server) stagedConfluenceCandidateProvenance(ctx context.Context, missionID, normalizedURL string) (stagedSourceCandidateProvenance, error) {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return stagedSourceCandidateProvenance{}, err
	}
	converted := make([]sourcecandidateevents.Event, 0, len(events))
	for _, event := range events {
		converted = append(converted, sourcecandidateevents.Event{EventID: event.EventID, Sequence: event.Sequence, EventType: event.EventType, Payload: event.Payload, CreatedAt: event.CreatedAt})
	}
	payload, state, ok := sourcecandidateevents.LatestStagingTerminalForURL(converted, normalizedURL, func(value string) (string, error) { return normalizedHTTPURL(value) })
	if !ok || state != sourcecandidateevents.StagedEventType || strings.TrimSpace(payload.ArtifactID) == "" || strings.TrimSpace(payload.ProposalEventID) == "" {
		return stagedSourceCandidateProvenance{}, nil
	}
	return stagedSourceCandidateProvenance{ProposalEventID: payload.ProposalEventID, URL: normalizedURL}, nil
}
