package confluence

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestClientSearchConfluenceSourcesUsesCQLSearchEndpoint(t *testing.T) {
	var gotPath string
	var gotQuery string
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotAuth = r.Header.Get("Authorization")
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"results": [{
				"content": {
					"id": "123",
					"type": "page",
					"status": "current",
					"title": "Roadmap",
					"space": {"id": 987, "key": "ENG", "name": "Engineering"},
					"version": {"when": "2026-07-02T05:10:00.000Z", "number": 7},
					"_links": {"webui": "/spaces/ENG/pages/123/Roadmap"}
				},
				"excerpt": "<p>Roadmap <strong>planning</strong></p>",
				"url": "/spaces/ENG/pages/123/Roadmap"
			}],
			"_links": {
				"base": "https://example.atlassian.net/wiki",
				"next": "/wiki/rest/api/search?cql=type%3Dpage&limit=5&cursor=next"
			}
		}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL+"/wiki", "cloud_1", WithBearerToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	result, err := client.SearchConfluenceSources(context.Background(), app.ConfluenceSourceSearchRequest{
		MissionID: "mis_1",
		CloudID:   "cloud_1",
		Query:     "roadmap",
		Limit:     5,
		SpaceKey:  "ENG",
	})
	if err != nil {
		t.Fatalf("SearchConfluenceSources returned error: %v", err)
	}
	if gotPath != "/wiki/rest/api/search" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	values, err := url.ParseQuery(gotQuery)
	if err != nil {
		t.Fatalf("could not parse query %q: %v", gotQuery, err)
	}
	if values.Get("limit") != "5" {
		t.Fatalf("unexpected limit query: %s", gotQuery)
	}
	if values.Get("cql") != `type=page and space = "ENG" and text ~ "roadmap"` {
		t.Fatalf("unexpected CQL query: %s", values.Get("cql"))
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("unexpected authorization header: %q", gotAuth)
	}
	if result.NextCursor != "next" || len(result.Candidates) != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	candidate := result.Candidates[0]
	if candidate.Connector.ExternalSourceID != app.ConfluenceExternalSourceID("cloud_1", "123") ||
		candidate.SourceURI != "https://example.atlassian.net/wiki/spaces/ENG/pages/123/Roadmap" ||
		candidate.Summary != "" ||
		!candidate.CanSnapshot {
		t.Fatalf("unexpected candidate: %#v", candidate)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if strings.Contains(string(encoded), "Roadmap planning") {
		t.Fatalf("search result leaked provider excerpt: %s", string(encoded))
	}
}

func TestClientReadConfluenceSourceUsesV2PageEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wiki/api/v2/pages/123" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("body-format") != "storage" {
			t.Fatalf("missing body-format=storage query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "123",
			"status": "current",
			"title": "Roadmap",
			"spaceId": "987",
			"createdAt": "2026-07-01T01:00:00.000Z",
			"version": {
				"createdAt": "2026-07-02T05:10:00.000Z",
				"message": "update",
				"number": 7,
				"minorEdit": false,
				"authorId": "abc"
			},
			"body": {
				"storage": {
					"value": "<p>Hello <strong>research</strong></p>",
					"representation": "storage"
				}
			},
			"_links": {
				"base": "https://example.atlassian.net/wiki",
				"webui": "/spaces/ENG/pages/123/Roadmap"
			}
		}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL+"/wiki", "cloud_1", WithConnectorVersion("confluence.test"))
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	page, err := client.ReadConfluenceSource(context.Background(), app.ConfluenceSourceReadRequest{
		CloudID: "cloud_1",
		PageID:  "123",
	})
	if err != nil {
		t.Fatalf("ReadConfluenceSource returned error: %v", err)
	}
	if page.Connector.ConnectorVersion != "confluence.test" ||
		page.Connector.ExternalURI != app.ConfluenceExternalURI("cloud_1", "123") ||
		page.WebURL != "https://example.atlassian.net/wiki/spaces/ENG/pages/123/Roadmap" {
		t.Fatalf("unexpected connector metadata: %#v", page)
	}
	if page.BodyStorage != "<p>Hello <strong>research</strong></p>" || page.PlainText != "Hello research" {
		t.Fatalf("unexpected body conversion: %#v", page)
	}
	if !strings.Contains(string(page.Metadata), `"space_id":"987"`) {
		t.Fatalf("metadata did not preserve page fields: %s", string(page.Metadata))
	}
}

func TestClientGetConfluenceSourceVersionDoesNotRequestBodyFormat(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wiki/api/v2/pages/123" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "123",
			"title": "Roadmap",
			"spaceId": "987",
			"version": {"createdAt": "2026-07-03T02:00:00.000Z", "number": 8},
			"_links": {"base": "https://example.atlassian.net/wiki", "webui": "/spaces/ENG/pages/123/Roadmap"}
		}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL+"/wiki", "cloud_1")
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	version, err := client.GetConfluenceSourceVersion(context.Background(), app.ConfluenceSourceReadRequest{CloudID: "cloud_1", PageID: "123"})
	if err != nil {
		t.Fatalf("GetConfluenceSourceVersion returned error: %v", err)
	}
	if gotQuery != "" {
		t.Fatalf("metadata request should not request body-format, got %q", gotQuery)
	}
	if version.Version != 8 || version.Title != "Roadmap" {
		t.Fatalf("unexpected version: %#v", version)
	}
}

func TestClientBrowseSpacesPagesAndChildren(t *testing.T) {
	paths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/wiki/api/v2/spaces":
			_, _ = w.Write([]byte(`{
				"results": [{"id":"sp_1","key":"ENG","name":"Engineering","type":"global","status":"current","_links":{"webui":"/spaces/ENG"}}],
				"_links": {"base":"https://example.atlassian.net/wiki","next":"/wiki/api/v2/spaces?cursor=next-space"}
			}`))
		case "/wiki/api/v2/spaces/sp_1/pages":
			_, _ = w.Write([]byte(`{
				"results": [{"id":"123","title":"Roadmap","spaceId":"sp_1","version":{"createdAt":"2026-07-02T05:10:00.000Z","number":7},"_links":{"webui":"/spaces/ENG/pages/123/Roadmap"}}],
				"_links": {"base":"https://example.atlassian.net/wiki","next":"/wiki/api/v2/spaces/sp_1/pages?cursor=next-page"}
			}`))
		case "/wiki/api/v2/pages/123/children":
			_, _ = w.Write([]byte(`{
				"results": [{"id":"456","title":"Child","spaceId":"sp_1","version":{"number":1},"_links":{"webui":"/spaces/ENG/pages/456/Child"}}],
				"_links": {"base":"https://example.atlassian.net/wiki"}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL+"/wiki", "cloud_1")
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	spaces, err := client.ListConfluenceSpaces(context.Background(), app.ConfluenceSpaceListRequest{CloudID: "cloud_1", Limit: 5})
	if err != nil {
		t.Fatalf("ListConfluenceSpaces returned error: %v", err)
	}
	pages, err := client.ListConfluenceSpacePages(context.Background(), app.ConfluenceSpacePagesRequest{CloudID: "cloud_1", SpaceID: "sp_1", Limit: 5})
	if err != nil {
		t.Fatalf("ListConfluenceSpacePages returned error: %v", err)
	}
	children, err := client.ListConfluencePageChildren(context.Background(), app.ConfluencePageChildrenRequest{CloudID: "cloud_1", PageID: "123", Limit: 5})
	if err != nil {
		t.Fatalf("ListConfluencePageChildren returned error: %v", err)
	}
	if spaces.NextCursor != "next-space" || spaces.Spaces[0].WebURL != "https://example.atlassian.net/wiki/spaces/ENG" {
		t.Fatalf("unexpected spaces: %#v", spaces)
	}
	if pages.NextCursor != "next-page" || pages.Pages[0].PageID != "123" || pages.Pages[0].Version != 7 {
		t.Fatalf("unexpected pages: %#v", pages)
	}
	if children.Pages[0].ParentID != "123" || children.Pages[0].Title != "Child" {
		t.Fatalf("unexpected children: %#v", children)
	}
	if strings.Join(paths, ",") != "/wiki/api/v2/spaces,/wiki/api/v2/spaces/sp_1/pages,/wiki/api/v2/pages/123/children" {
		t.Fatalf("unexpected paths: %#v", paths)
	}
}

func TestSafeOperationRedactsConfluencePathIDs(t *testing.T) {
	cases := map[string]string{
		"/api/v2/pages/123/children?cursor=abc":      "GET /api/v2/pages/{page_id}/children",
		"/api/v2/spaces/sp_1/pages?limit=20":         "GET /api/v2/spaces/{space_id}/pages",
		"/api/v2/spaces/sp_1/pages/123?limit=20":     "GET /api/v2/spaces/{space_id}/pages/{page_id}",
		"/wiki/api/v2/pages/123?body-format=storage": "GET /wiki/api/v2/pages/{page_id}",
	}
	for endpoint, want := range cases {
		got := safeOperation(http.MethodGet, endpoint)
		if got != want {
			t.Fatalf("safeOperation(%q) = %q, want %q", endpoint, got, want)
		}
		for _, leaked := range []string{"123", "sp_1", "cursor=abc", "body-format"} {
			if strings.Contains(got, leaked) {
				t.Fatalf("safeOperation leaked %q in %q", leaked, got)
			}
		}
	}
}

func TestClientTransportErrorIsRedacted(t *testing.T) {
	client, err := NewClient(
		"https://docs.atlassian.net/wiki",
		"cloud_1",
		WithHTTPClient(&http.Client{Transport: failingRoundTripper{}}),
	)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	_, err = client.ReadConfluenceSource(context.Background(), app.ConfluenceSourceReadRequest{
		CloudID: "cloud_1",
		PageID:  "secret-page",
	})
	var confluenceErr *app.ConfluenceError
	if err == nil || !errors.As(err, &confluenceErr) || confluenceErr.Code != app.ConfluenceErrorCodeUpstream {
		t.Fatalf("expected redacted Confluence transport error, got %v", err)
	}
	if confluenceErr.Operation != "GET /api/v2/pages/{page_id}" {
		t.Fatalf("unexpected redacted operation: %q", confluenceErr.Operation)
	}
	visible := err.Error() + " " + confluenceErr.Operation
	for _, leaked := range []string{"secret-page", "body-format", "docs.atlassian.net", "transport-secret"} {
		if strings.Contains(visible, leaked) {
			t.Fatalf("transport error leaked %q in %q", leaked, visible)
		}
	}
}

type failingRoundTripper struct{}

func (f failingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, &url.Error{
		Op:  req.Method,
		URL: req.URL.String(),
		Err: errors.New("transport-secret"),
	}
}

func TestClientWithBasicAuthUsesAuthorizationHeader(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"123","title":"Roadmap","version":{"number":1}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "cloud_1", WithBasicAuth("person@example.com", "api-token"))
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	if _, err := client.GetConfluenceSourceVersion(context.Background(), app.ConfluenceSourceReadRequest{CloudID: "cloud_1", PageID: "123"}); err != nil {
		t.Fatalf("GetConfluenceSourceVersion returned error: %v", err)
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("person@example.com:api-token"))
	if gotAuth != want {
		t.Fatalf("unexpected authorization header: %q", gotAuth)
	}
}

func TestClientRejectsInvalidSiteURLOption(t *testing.T) {
	if _, err := NewClient("https://example.atlassian.net/wiki", "cloud_1", WithSiteURL("://bad-url")); err == nil {
		t.Fatal("expected invalid site URL option to fail")
	}
}

func TestDiscoveryClientListsConfluenceSites(t *testing.T) {
	var gotPath string
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":"cloud_1","name":"Docs","url":"https://docs.atlassian.net","scopes":["read:page:confluence"]},
			{"id":"cloud_2","name":"Jira","url":"https://jira.atlassian.net","scopes":["read:jira-work"]}
		]`))
	}))
	defer server.Close()

	client, err := NewDiscoveryClient(WithDiscoveryBaseURL(server.URL), WithDiscoveryBearerToken("token"))
	if err != nil {
		t.Fatalf("NewDiscoveryClient returned error: %v", err)
	}
	result, err := client.ListConfluenceSites(context.Background())
	if err != nil {
		t.Fatalf("ListConfluenceSites returned error: %v", err)
	}
	if gotPath != "/oauth/token/accessible-resources" || gotAuth != "Bearer token" {
		t.Fatalf("unexpected request path/auth: %s %q", gotPath, gotAuth)
	}
	if len(result.Sites) != 1 || result.Sites[0].CloudID != "cloud_1" {
		t.Fatalf("unexpected sites: %#v", result.Sites)
	}
}

func TestDiscoveryClientRejectsSensitiveBaseURL(t *testing.T) {
	for _, baseURL := range []string{
		"https://person:secret@docs.atlassian.net",
		"https://docs.atlassian.net?token=secret",
		"https://docs.atlassian.net#secret",
	} {
		if _, err := NewDiscoveryClient(WithDiscoveryBaseURL(baseURL)); err == nil {
			t.Fatalf("expected sensitive discovery base URL %q to be rejected", baseURL)
		}
	}
}

func TestDiscoveryClientTransportErrorIsRedacted(t *testing.T) {
	client, err := NewDiscoveryClient(
		WithDiscoveryBaseURL("https://discovery-secret.example"),
		WithDiscoveryHTTPClient(&http.Client{Transport: failingRoundTripper{}}),
		WithDiscoveryBearerToken("secret-access-token"),
	)
	if err != nil {
		t.Fatalf("NewDiscoveryClient returned error: %v", err)
	}
	_, err = client.ListConfluenceSites(context.Background())
	var confluenceErr *app.ConfluenceError
	if err == nil || !errors.As(err, &confluenceErr) || confluenceErr.Code != app.ConfluenceErrorCodeUpstream {
		t.Fatalf("expected redacted discovery transport error, got %v", err)
	}
	if confluenceErr.Operation != "GET /oauth/token/accessible-resources" {
		t.Fatalf("unexpected operation: %q", confluenceErr.Operation)
	}
	visible := err.Error() + " " + confluenceErr.Operation
	for _, leaked := range []string{"discovery-secret.example", "secret-access-token", "transport-secret"} {
		if strings.Contains(visible, leaked) {
			t.Fatalf("discovery transport error leaked %q in %q", leaked, visible)
		}
	}
}

func TestOAuthClientBuildsAuthorizeURLAndExchangesCode(t *testing.T) {
	var tokenRequest map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&tokenRequest); err != nil {
			t.Fatalf("decode token request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"access_token": "access-secret",
			"refresh_token": "refresh-secret",
			"token_type": "Bearer",
			"expires_in": 3600,
			"scope": "read:confluence-content.all offline_access"
		}`))
	}))
	defer server.Close()

	client, err := NewOAuthClient(OAuthConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURI:  "http://127.0.0.1/callback",
		AuthorizeURL: server.URL + "/authorize",
		TokenURL:     server.URL + "/oauth/token",
		Scopes:       []string{"read:confluence-content.all", "offline_access"},
	})
	if err != nil {
		t.Fatalf("NewOAuthClient returned error: %v", err)
	}
	authorizeURL, err := client.AuthorizationURL(OAuthAuthorizationRequest{State: "state-1"})
	if err != nil {
		t.Fatalf("AuthorizationURL returned error: %v", err)
	}
	parsed, err := url.Parse(authorizeURL)
	if err != nil {
		t.Fatalf("parse authorize URL: %v", err)
	}
	query := parsed.Query()
	if query.Get("audience") != "api.atlassian.com" ||
		query.Get("client_id") != "client-id" ||
		query.Get("redirect_uri") != "http://127.0.0.1/callback" ||
		query.Get("state") != "state-1" ||
		query.Get("response_type") != "code" ||
		query.Get("prompt") != "consent" {
		t.Fatalf("unexpected authorize query: %s", parsed.RawQuery)
	}
	if query.Get("scope") != "read:confluence-content.all offline_access" {
		t.Fatalf("unexpected scope query: %q", query.Get("scope"))
	}

	token, err := client.ExchangeCode(context.Background(), OAuthCodeExchangeRequest{Code: "code-1"})
	if err != nil {
		t.Fatalf("ExchangeCode returned error: %v", err)
	}
	if token.AccessToken != "access-secret" || token.RefreshToken != "refresh-secret" || len(token.Scopes) != 2 {
		t.Fatalf("unexpected token result: %#v", token)
	}
	if tokenRequest["grant_type"] != "authorization_code" ||
		tokenRequest["client_id"] != "client-id" ||
		tokenRequest["client_secret"] != "client-secret" ||
		tokenRequest["code"] != "code-1" ||
		tokenRequest["redirect_uri"] != "http://127.0.0.1/callback" {
		t.Fatalf("unexpected token request: %#v", tokenRequest)
	}
}

func TestOAuthClientReturnsStatusErrorsWithoutResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "sensitive token error", http.StatusUnauthorized)
	}))
	defer server.Close()

	client, err := NewOAuthClient(OAuthConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURI:  "http://127.0.0.1/callback",
		TokenURL:     server.URL,
	})
	if err != nil {
		t.Fatalf("NewOAuthClient returned error: %v", err)
	}
	_, err = client.ExchangeCode(context.Background(), OAuthCodeExchangeRequest{Code: "code-1"})
	var confluenceErr *app.ConfluenceError
	if err == nil || !errors.As(err, &confluenceErr) || confluenceErr.Code != app.ConfluenceErrorCodeUnauthorized {
		t.Fatalf("expected typed 401 error, got %v", err)
	}
	if strings.Contains(err.Error(), "sensitive token error") {
		t.Fatalf("error leaked provider body: %v", err)
	}
}

func TestOAuthClientRejectsSensitiveTokenURL(t *testing.T) {
	for _, tokenURL := range []string{
		"https://person:secret@auth.atlassian.com/oauth/token",
		"https://auth.atlassian.com/oauth/token?code=secret",
		"https://auth.atlassian.com/oauth/token#secret",
	} {
		_, err := NewOAuthClient(OAuthConfig{
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			TokenURL:     tokenURL,
		})
		if err == nil {
			t.Fatalf("expected sensitive OAuth token URL %q to be rejected", tokenURL)
		}
	}
}

func TestOAuthClientTransportErrorIsRedacted(t *testing.T) {
	client, err := NewOAuthClient(OAuthConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURI:  "http://127.0.0.1/callback",
		TokenURL:     "https://token-secret.example/oauth/token",
		HTTPClient:   &http.Client{Transport: failingRoundTripper{}},
	})
	if err != nil {
		t.Fatalf("NewOAuthClient returned error: %v", err)
	}
	_, err = client.ExchangeCode(context.Background(), OAuthCodeExchangeRequest{Code: "code-secret"})
	var confluenceErr *app.ConfluenceError
	if err == nil || !errors.As(err, &confluenceErr) || confluenceErr.Code != app.ConfluenceErrorCodeUpstream {
		t.Fatalf("expected redacted OAuth transport error, got %v", err)
	}
	if confluenceErr.Operation != "POST /oauth/token" {
		t.Fatalf("unexpected operation: %q", confluenceErr.Operation)
	}
	visible := err.Error() + " " + confluenceErr.Operation
	for _, leaked := range []string{"token-secret.example", "client-secret", "code-secret", "transport-secret"} {
		if strings.Contains(visible, leaked) {
			t.Fatalf("OAuth transport error leaked %q in %q", leaked, visible)
		}
	}
}

func TestOAuthClientRefreshesAccessToken(t *testing.T) {
	var tokenRequest map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&tokenRequest); err != nil {
			t.Fatalf("decode token request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"access_token": "refreshed-access",
			"refresh_token": "rotated-refresh",
			"token_type": "Bearer",
			"expires_in": 3600,
			"scope": "read:confluence-content.all offline_access"
		}`))
	}))
	defer server.Close()

	client, err := NewOAuthClient(OAuthConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		TokenURL:     server.URL + "/oauth/token",
	})
	if err != nil {
		t.Fatalf("NewOAuthClient returned error: %v", err)
	}
	token, err := client.RefreshAccessToken(context.Background(), "refresh-secret")
	if err != nil {
		t.Fatalf("RefreshAccessToken returned error: %v", err)
	}
	if token.AccessToken != "refreshed-access" || token.RefreshToken != "rotated-refresh" || token.TokenExpiresAt.IsZero() {
		t.Fatalf("unexpected token result: %#v", token)
	}
	if tokenRequest["grant_type"] != "refresh_token" ||
		tokenRequest["client_id"] != "client-id" ||
		tokenRequest["client_secret"] != "client-secret" ||
		tokenRequest["refresh_token"] != "refresh-secret" {
		t.Fatalf("unexpected refresh token request: %#v", tokenRequest)
	}
}

func TestClientReturnsHTTPStatusErrorsWithoutResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "7")
		http.Error(w, "sensitive provider details", http.StatusUnauthorized)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "cloud_1")
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	_, err = client.ReadConfluenceSource(context.Background(), app.ConfluenceSourceReadRequest{
		CloudID: "cloud_1",
		PageID:  "123",
	})
	var confluenceErr *app.ConfluenceError
	if err == nil || !errors.As(err, &confluenceErr) || confluenceErr.Code != app.ConfluenceErrorCodeUnauthorized {
		t.Fatalf("expected status error, got %v", err)
	}
	if confluenceErr.Operation == "" {
		t.Fatalf("expected safe operation in confluence error: %#v", confluenceErr)
	}
	if strings.Contains(err.Error(), "sensitive provider details") {
		t.Fatalf("error leaked provider body: %v", err)
	}
}

func TestClientRejectsCloudIDMismatch(t *testing.T) {
	client, err := NewClient("https://example.atlassian.net/wiki", "cloud_1")
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	_, err = client.SearchConfluenceSources(context.Background(), app.ConfluenceSourceSearchRequest{CloudID: "cloud_2"})
	var confluenceErr *app.ConfluenceError
	if err == nil || !errors.As(err, &confluenceErr) || confluenceErr.Code != app.ConfluenceErrorCodeCloudMismatch {
		t.Fatalf("expected cloud id mismatch, got %v", err)
	}
}

func TestNewClientRejectsInvalidBaseURL(t *testing.T) {
	if _, err := NewClient("localhost:8080", "cloud_1"); err == nil {
		t.Fatal("expected invalid base URL error")
	}
}
