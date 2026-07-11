package app

import (
	"context"
	"strings"
	"testing"
)

func TestMissionConnectorAccessDefaultsOffAndReplaysLedger(t *testing.T) {
	store := newConnectorAccessFakeStore()
	svc := NewService(store)
	got, err := svc.GetMissionConnectorAccess(context.Background(), "mis_access", ConfluenceConnectorID)
	if err != nil {
		t.Fatalf("GetMissionConnectorAccess returned error: %v", err)
	}
	if got.Enabled || got.Status != ConnectorAccessStatusDisabled || got.LastEventID != "" {
		t.Fatalf("expected default-off access, got %#v", got)
	}

	result, err := svc.SetMissionConnectorAccess(context.Background(), SetConnectorAccessRequest{
		EventID:      "evt_enable",
		MissionID:    "mis_access",
		ConnectorID:  ConfluenceConnectorID,
		Enabled:      true,
		ConnectionID: "cnf_docs",
		CloudID:      "cloud_1",
		SpaceKey:     "ENG",
		Producer:     Producer{Type: "user", ID: "plasma-ui"},
	})
	if err != nil {
		t.Fatalf("SetMissionConnectorAccess enable returned error: %v", err)
	}
	if result.Event.EventType != ConnectorAccessEventEnabled || result.Access.Status != ConnectorAccessStatusEnabled {
		t.Fatalf("unexpected enable result: %#v", result)
	}
	if result.Access.ConnectionID != "cnf_docs" || result.Access.CloudID != "cloud_1" || result.Access.SpaceKey != "ENG" {
		t.Fatalf("unexpected projected grant: %#v", result.Access)
	}

	replayed, err := svc.GetMissionConnectorAccess(context.Background(), "mis_access", ConfluenceConnectorID)
	if err != nil {
		t.Fatalf("replay access: %v", err)
	}
	if replayed.LastEventID != "evt_enable" || replayed.LastSequence != 1 || !replayed.Enabled {
		t.Fatalf("expected replay from ledger event, got %#v", replayed)
	}
}

func TestMissionConnectorAccessUpdateDisableAndNoCredentialPayload(t *testing.T) {
	store := newConnectorAccessFakeStore()
	svc := NewService(store)
	if _, err := svc.SetMissionConnectorAccess(context.Background(), SetConnectorAccessRequest{
		EventID:      "evt_enable",
		MissionID:    "mis_access",
		ConnectorID:  ConfluenceConnectorID,
		Enabled:      true,
		ConnectionID: "cnf_docs",
		CloudID:      "cloud_1",
		Producer:     Producer{Type: "user", ID: "plasma-ui"},
	}); err != nil {
		t.Fatalf("enable grant: %v", err)
	}
	updated, err := svc.SetMissionConnectorAccess(context.Background(), SetConnectorAccessRequest{
		EventID:      "evt_update",
		MissionID:    "mis_access",
		ConnectorID:  ConfluenceConnectorID,
		Enabled:      true,
		ConnectionID: "cnf_docs",
		CloudID:      "cloud_1",
		SpaceKey:     "OPS",
		Producer:     Producer{Type: "user", ID: "plasma-ui"},
	})
	if err != nil {
		t.Fatalf("update grant: %v", err)
	}
	if updated.Event.EventType != ConnectorAccessEventUpdated || updated.Access.SpaceKey != "OPS" {
		t.Fatalf("unexpected update result: %#v", updated)
	}
	disabled, err := svc.SetMissionConnectorAccess(context.Background(), SetConnectorAccessRequest{
		EventID:     "evt_disable",
		MissionID:   "mis_access",
		ConnectorID: ConfluenceConnectorID,
		Producer:    Producer{Type: "user", ID: "plasma-ui"},
	})
	if err != nil {
		t.Fatalf("disable grant: %v", err)
	}
	if disabled.Event.EventType != ConnectorAccessEventDisabled || disabled.Access.Enabled || disabled.Access.Status != ConnectorAccessStatusDisabled {
		t.Fatalf("unexpected disable result: %#v", disabled)
	}
	for _, event := range store.events {
		raw := string(event.Payload)
		for _, leaked := range []string{"secret", "token", "Authorization", "Bearer"} {
			if strings.Contains(raw, leaked) {
				t.Fatalf("grant event leaked %q: %s", leaked, raw)
			}
		}
	}
}

func TestMissionConnectorAccessValidationAndInvalidProjection(t *testing.T) {
	store := newConnectorAccessFakeStore()
	svc := NewService(store)
	cases := []struct {
		name string
		req  SetConnectorAccessRequest
		want string
	}{
		{
			name: "agent producer",
			req: SetConnectorAccessRequest{
				EventID:      "evt_agent",
				MissionID:    "mis_access",
				ConnectorID:  ConfluenceConnectorID,
				Enabled:      true,
				ConnectionID: "cnf_docs",
				CloudID:      "cloud_1",
				Producer:     Producer{Type: "agent_session", ID: "ses_1"},
			},
			want: "user action",
		},
		{
			name: "steering chat producer",
			req: SetConnectorAccessRequest{
				EventID:      "evt_steering",
				MissionID:    "mis_access",
				ConnectorID:  ConfluenceConnectorID,
				Enabled:      true,
				ConnectionID: "cnf_docs",
				CloudID:      "cloud_1",
				Producer:     Producer{Type: "steering_chat", ID: "chat_1"},
			},
			want: "user action",
		},
		{
			name: "revoked connection",
			req: SetConnectorAccessRequest{
				EventID:      "evt_revoked",
				MissionID:    "mis_access",
				ConnectorID:  ConfluenceConnectorID,
				Enabled:      true,
				ConnectionID: "cnf_revoked",
				CloudID:      "cloud_1",
				Producer:     Producer{Type: "user", ID: "plasma-ui"},
			},
			want: "revoked",
		},
		{
			name: "wrong cloud",
			req: SetConnectorAccessRequest{
				EventID:      "evt_cloud",
				MissionID:    "mis_access",
				ConnectorID:  ConfluenceConnectorID,
				Enabled:      true,
				ConnectionID: "cnf_docs",
				CloudID:      "cloud_other",
				Producer:     Producer{Type: "user", ID: "plasma-ui"},
			},
			want: "cloud_id",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.SetMissionConnectorAccess(context.Background(), tc.req)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got %v", tc.want, err)
			}
		})
	}

	if _, err := svc.SetMissionConnectorAccess(context.Background(), SetConnectorAccessRequest{
		EventID:      "evt_enable",
		MissionID:    "mis_access",
		ConnectorID:  ConfluenceConnectorID,
		Enabled:      true,
		ConnectionID: "cnf_docs",
		CloudID:      "cloud_1",
		Producer:     Producer{Type: "user", ID: "plasma-ui"},
	}); err != nil {
		t.Fatalf("enable grant: %v", err)
	}
	store.connections["cnf_docs"] = ConfluenceConnection{
		ConnectionID: "cnf_docs",
		DisplayName:  "Docs",
		AuthType:     ConfluenceAuthTypeOAuth,
		AccessToken:  "secret-oauth-token",
		Revoked:      true,
		Sites:        []ConfluenceSite{{CloudID: "cloud_1", URL: "https://docs.atlassian.net"}},
	}
	invalid, err := svc.GetMissionConnectorAccess(context.Background(), "mis_access", ConfluenceConnectorID)
	if err != nil {
		t.Fatalf("get invalid projection: %v", err)
	}
	if !invalid.Enabled || invalid.Status != ConnectorAccessStatusInvalid || invalid.InvalidReason != "connection_revoked" {
		t.Fatalf("expected invalid revoked projection, got %#v", invalid)
	}
}

type connectorAccessFakeStore struct {
	fakeStore
	events      []LedgerEvent
	connections map[string]ConfluenceConnection
}

func newConnectorAccessFakeStore() *connectorAccessFakeStore {
	return &connectorAccessFakeStore{
		connections: map[string]ConfluenceConnection{
			"cnf_docs": {
				ConnectionID: "cnf_docs",
				DisplayName:  "Docs",
				AuthType:     ConfluenceAuthTypeOAuth,
				AccessToken:  "secret-oauth-token",
				Sites:        []ConfluenceSite{{CloudID: "cloud_1", Name: "Docs", URL: "https://docs.atlassian.net"}},
			},
			"cnf_revoked": {
				ConnectionID: "cnf_revoked",
				DisplayName:  "Revoked",
				AuthType:     ConfluenceAuthTypeOAuth,
				Revoked:      true,
				Sites:        []ConfluenceSite{{CloudID: "cloud_1"}},
			},
		},
	}
}

func (f *connectorAccessFakeStore) AppendLedgerEvent(_ context.Context, event LedgerEvent) (LedgerEvent, error) {
	event.Sequence = int64(len(f.events) + 1)
	f.events = append(f.events, event)
	return event, nil
}

func (f *connectorAccessFakeStore) ListLedgerEvents(_ context.Context, missionID string) ([]LedgerEvent, error) {
	events := []LedgerEvent{}
	for _, event := range f.events {
		if event.MissionID == missionID {
			events = append(events, event)
		}
	}
	return events, nil
}

func (f *connectorAccessFakeStore) UpsertConfluenceConnection(_ context.Context, connection ConfluenceConnection) error {
	f.connections[connection.ConnectionID] = connection
	return nil
}

func (f *connectorAccessFakeStore) GetConfluenceConnection(_ context.Context, connectionID string) (ConfluenceConnection, error) {
	if connection, ok := f.connections[connectionID]; ok {
		return connection, nil
	}
	return ConfluenceConnection{}, ErrInvalidInput
}

func (f *connectorAccessFakeStore) ListConfluenceConnections(context.Context) ([]ConfluenceConnection, error) {
	connections := make([]ConfluenceConnection, 0, len(f.connections))
	for _, connection := range f.connections {
		connections = append(connections, connection)
	}
	return connections, nil
}

func (f *connectorAccessFakeStore) DeleteConfluenceConnection(_ context.Context, connectionID string) error {
	delete(f.connections, connectionID)
	return nil
}
