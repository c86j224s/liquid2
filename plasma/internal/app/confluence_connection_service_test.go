package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestUpsertConfluenceConnectionNormalizesAndDoesNotMarshalTokens(t *testing.T) {
	store := &confluenceConnectionFakeStore{}
	svc := NewService(store)
	connection, err := svc.UpsertConfluenceConnection(context.Background(), UpsertConfluenceConnectionRequest{
		ConnectionID: " cnf_1 ",
		DisplayName:  " Workspace ",
		AuthType:     ConfluenceAuthTypeOAuth,
		AccountID:    " acct_1 ",
		AccessToken:  " access-secret ",
		RefreshToken: " refresh-secret ",
		Scopes:       []string{"read:page:confluence", "read:page:confluence"},
		Sites: []ConfluenceSite{{
			CloudID: " cloud_1 ",
			Name:    " Example ",
			URL:     "https://example.atlassian.net/",
		}},
	})
	if err != nil {
		t.Fatalf("UpsertConfluenceConnection returned error: %v", err)
	}
	if connection.ConnectionID != "cnf_1" || len(connection.Scopes) != 1 || connection.Sites[0].URL != "https://example.atlassian.net" {
		t.Fatalf("connection was not normalized: %#v", connection)
	}
	raw, err := json.Marshal(connection)
	if err != nil {
		t.Fatalf("marshal connection: %v", err)
	}
	for _, leaked := range []string{"access-secret", "refresh-secret"} {
		if strings.Contains(string(raw), leaked) {
			t.Fatalf("connection JSON leaked token %q: %s", leaked, string(raw))
		}
	}
}

func TestUpsertConfluenceConnectionRejectsMissingToken(t *testing.T) {
	svc := NewService(&confluenceConnectionFakeStore{})
	_, err := svc.UpsertConfluenceConnection(context.Background(), UpsertConfluenceConnectionRequest{
		ConnectionID: "cnf_1",
		AuthType:     ConfluenceAuthTypeOAuth,
	})
	if err == nil {
		t.Fatal("expected missing token error")
	}
}

func TestUpsertConfluenceConnectionRejectsAPITokenWithoutEmail(t *testing.T) {
	svc := NewService(&confluenceConnectionFakeStore{})
	_, err := svc.UpsertConfluenceConnection(context.Background(), UpsertConfluenceConnectionRequest{
		ConnectionID: "cnf_1",
		AuthType:     ConfluenceAuthTypeAPIToken,
		AccessToken:  "api-token",
	})
	if err == nil || !strings.Contains(err.Error(), "account email") {
		t.Fatalf("expected account email error, got %v", err)
	}
}

func TestUpsertConfluenceConnectionRejectsUnsafeAPITokenSiteURL(t *testing.T) {
	svc := NewService(&confluenceConnectionFakeStore{})
	for _, siteURL := range []string{
		"http://docs.atlassian.net",
		"https://evil.example",
		"https://person:secret@docs.atlassian.net/wiki",
		"https://docs.atlassian.net/wiki/spaces/ENG/pages/123/Roadmap",
	} {
		_, err := svc.UpsertConfluenceConnection(context.Background(), UpsertConfluenceConnectionRequest{
			ConnectionID: "cnf_1",
			AuthType:     ConfluenceAuthTypeAPIToken,
			AccountName:  "person@example.com",
			AccessToken:  "api-token",
			Sites:        []ConfluenceSite{{CloudID: "cloud_1", URL: siteURL}},
		})
		if err == nil {
			t.Fatalf("expected unsafe site URL %q to be rejected", siteURL)
		}
	}
}

func TestUpsertConfluenceConnectionRejectsAPITokenCloudIDMismatch(t *testing.T) {
	svc := NewService(&confluenceConnectionFakeStore{})
	_, err := svc.UpsertConfluenceConnection(context.Background(), UpsertConfluenceConnectionRequest{
		ConnectionID: "cnf_1",
		AuthType:     ConfluenceAuthTypeAPIToken,
		AccountName:  "person@example.com",
		AccessToken:  "api-token",
		Sites:        []ConfluenceSite{{CloudID: "cloud_1", URL: "https://docs.atlassian.net/wiki/"}},
	})
	if err == nil || !strings.Contains(err.Error(), "cloud id must match the site URL") {
		t.Fatalf("expected API token cloud id mismatch error, got %v", err)
	}
}

func TestUpsertConfluenceConnectionAllowsAPITokenAtlassianSiteURL(t *testing.T) {
	svc := NewService(&confluenceConnectionFakeStore{})
	connection, err := svc.UpsertConfluenceConnection(context.Background(), UpsertConfluenceConnectionRequest{
		ConnectionID: "cnf_1",
		AuthType:     ConfluenceAuthTypeAPIToken,
		AccountName:  "person@example.com",
		AccessToken:  "api-token",
		Sites:        []ConfluenceSite{{URL: "https://docs.atlassian.net/wiki/"}},
	})
	if err != nil {
		t.Fatalf("UpsertConfluenceConnection returned error: %v", err)
	}
	if connection.Sites[0].URL != "https://docs.atlassian.net/wiki" {
		t.Fatalf("unexpected normalized API token site URL: %#v", connection.Sites[0])
	}
	if connection.Sites[0].CloudID != "site_docs.atlassian.net" || connection.Sites[0].Name != "docs.atlassian.net" {
		t.Fatalf("expected API token site metadata to be derived from URL, got %#v", connection.Sites[0])
	}
}

func TestConfluenceAPITokenSiteCloudIDRejectsUnsafeURL(t *testing.T) {
	if got, err := ConfluenceAPITokenSiteCloudID("https://docs.atlassian.net/wiki/"); err != nil || got != "site_docs.atlassian.net" {
		t.Fatalf("ConfluenceAPITokenSiteCloudID returned %q, %v", got, err)
	}
	if _, err := ConfluenceAPITokenSiteCloudID("https://evil.example/wiki"); err == nil {
		t.Fatal("expected unsafe site URL to be rejected")
	}
	if _, err := ConfluenceAPITokenSiteCloudID("https://docs.atlassian.net/wiki/spaces/ENG/pages/123/Roadmap"); err == nil {
		t.Fatal("expected page URL to be rejected")
	}
}

func TestNormalizeConfluenceAPITokenAPIBaseURL(t *testing.T) {
	got, err := NormalizeConfluenceAPITokenAPIBaseURL("https://docs.atlassian.net/wiki/?ignored=1#fragment")
	if err != nil {
		t.Fatalf("NormalizeConfluenceAPITokenAPIBaseURL returned error: %v", err)
	}
	if got != "https://docs.atlassian.net/wiki" {
		t.Fatalf("unexpected normalized API base URL: %q", got)
	}
	for _, value := range []string{
		"http://docs.atlassian.net/wiki",
		"https://evil.example/wiki",
		"https://person:secret@docs.atlassian.net/wiki",
	} {
		if _, err := NormalizeConfluenceAPITokenAPIBaseURL(value); err == nil {
			t.Fatalf("expected API base URL %q to be rejected", value)
		}
	}
}

func TestNormalizeConfluenceAPITokenAPIBaseURLForSiteRejectsCrossTenant(t *testing.T) {
	got, err := NormalizeConfluenceAPITokenAPIBaseURLForSite(
		"https://docs.atlassian.net/wiki",
		"https://docs.atlassian.net/wiki/",
	)
	if err != nil {
		t.Fatalf("NormalizeConfluenceAPITokenAPIBaseURLForSite returned error: %v", err)
	}
	if got != "https://docs.atlassian.net/wiki" {
		t.Fatalf("unexpected normalized API base URL: %q", got)
	}
	if _, err := NormalizeConfluenceAPITokenAPIBaseURLForSite(
		"https://other.atlassian.net/wiki",
		"https://docs.atlassian.net/wiki",
	); err == nil {
		t.Fatal("expected cross-tenant API base URL to be rejected")
	}
}

type confluenceConnectionFakeStore struct {
	fakeStore
	connections map[string]ConfluenceConnection
}

func (f *confluenceConnectionFakeStore) UpsertConfluenceConnection(_ context.Context, connection ConfluenceConnection) error {
	if f.connections == nil {
		f.connections = map[string]ConfluenceConnection{}
	}
	f.connections[connection.ConnectionID] = connection
	return nil
}

func (f *confluenceConnectionFakeStore) GetConfluenceConnection(_ context.Context, connectionID string) (ConfluenceConnection, error) {
	if connection, ok := f.connections[connectionID]; ok {
		return connection, nil
	}
	return ConfluenceConnection{}, ErrInvalidInput
}

func (f *confluenceConnectionFakeStore) ListConfluenceConnections(context.Context) ([]ConfluenceConnection, error) {
	connections := make([]ConfluenceConnection, 0, len(f.connections))
	for _, connection := range f.connections {
		connections = append(connections, connection)
	}
	return connections, nil
}

func (f *confluenceConnectionFakeStore) DeleteConfluenceConnection(_ context.Context, connectionID string) error {
	delete(f.connections, connectionID)
	return nil
}
