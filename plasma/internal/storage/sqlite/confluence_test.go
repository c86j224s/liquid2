package sqlite

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestConfluenceConnectionRoundTripDoesNotMarshalTokens(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()

	expiresAt := time.Date(2026, 7, 3, 3, 0, 0, 0, time.UTC)
	err = store.UpsertConfluenceConnection(ctx, app.ConfluenceConnection{
		ConnectionID:   "cnf_1",
		DisplayName:    "Docs",
		AuthType:       app.ConfluenceAuthTypeOAuth,
		AccountID:      "acct_1",
		AccessToken:    "access-secret",
		RefreshToken:   "refresh-secret",
		TokenExpiresAt: expiresAt,
		Scopes:         []string{"read:page:confluence"},
		Sites: []app.ConfluenceSite{{
			CloudID: "cloud_1",
			Name:    "Docs",
			URL:     "https://docs.atlassian.net",
		}},
	})
	if err != nil {
		t.Fatalf("UpsertConfluenceConnection returned error: %v", err)
	}
	connection, err := store.GetConfluenceConnection(ctx, "cnf_1")
	if err != nil {
		t.Fatalf("GetConfluenceConnection returned error: %v", err)
	}
	if connection.AccessToken != "access-secret" || connection.RefreshToken != "refresh-secret" {
		t.Fatalf("tokens were not stored for connector use: %#v", connection)
	}
	if !connection.TokenExpiresAt.Equal(expiresAt) || len(connection.Sites) != 1 {
		t.Fatalf("unexpected connection: %#v", connection)
	}
	raw, err := json.Marshal(connection)
	if err != nil {
		t.Fatalf("marshal connection: %v", err)
	}
	for _, leaked := range []string{"access-secret", "refresh-secret"} {
		if strings.Contains(string(raw), leaked) {
			t.Fatalf("connection JSON leaked %q: %s", leaked, string(raw))
		}
	}
}

func TestConfluenceConnectionListAndDelete(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()

	for _, id := range []string{"cnf_1", "cnf_2"} {
		if err := store.UpsertConfluenceConnection(ctx, app.ConfluenceConnection{
			ConnectionID: id,
			DisplayName:  id,
			AuthType:     app.ConfluenceAuthTypeAPIToken,
			AccessToken:  "token",
		}); err != nil {
			t.Fatalf("upsert %s: %v", id, err)
		}
	}
	connections, err := store.ListConfluenceConnections(ctx)
	if err != nil {
		t.Fatalf("ListConfluenceConnections returned error: %v", err)
	}
	if len(connections) != 2 {
		t.Fatalf("expected two connections, got %#v", connections)
	}
	if err := store.DeleteConfluenceConnection(ctx, "cnf_1"); err != nil {
		t.Fatalf("DeleteConfluenceConnection returned error: %v", err)
	}
	connections, err = store.ListConfluenceConnections(ctx)
	if err != nil {
		t.Fatalf("ListConfluenceConnections returned error: %v", err)
	}
	if len(connections) != 1 || connections[0].ConnectionID != "cnf_2" {
		t.Fatalf("unexpected remaining connections: %#v", connections)
	}
}
