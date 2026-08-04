package confluenceaccess

import (
	"encoding/json"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

func TestProjectReplaysEnableUpdateAndDisable(t *testing.T) {
	events := []ledger.Event{
		accessEvent(t, "evt_enable", 1, AccessEventEnabled, "cnf_docs", "cloud_1", "ENG"),
		accessEvent(t, "evt_update", 2, AccessEventUpdated, "cnf_docs", "cloud_1", "OPS"),
	}

	projection := Project("mis_1", ConnectorID, events)
	if !projection.Enabled || projection.Status != AccessStatusEnabled {
		t.Fatalf("expected enabled projection, got %#v", projection)
	}
	if projection.SpaceKey != "OPS" || projection.LastEventID != "evt_update" || projection.LastSequence != 2 {
		t.Fatalf("expected latest update, got %#v", projection)
	}

	events = append(events, accessEvent(t, "evt_disable", 3, AccessEventDisabled, "", "", ""))
	projection = Project("mis_1", ConnectorID, events)
	if projection.Enabled || projection.Status != AccessStatusDisabled {
		t.Fatalf("expected disabled projection, got %#v", projection)
	}
	if projection.ConnectionID != "" || projection.CloudID != "" || projection.SpaceKey != "" {
		t.Fatalf("disabled projection retained access target: %#v", projection)
	}
}

func TestProjectIgnoresOtherConnectorsAndMalformedEvents(t *testing.T) {
	malformed := ledger.Event{
		EventID:   "evt_bad",
		Sequence:  1,
		EventType: AccessEventEnabled,
		Payload:   json.RawMessage(`{"connector_id":`),
	}
	otherPayload, err := json.Marshal(map[string]string{"connector_id": "other"})
	if err != nil {
		t.Fatal(err)
	}
	other := ledger.Event{EventID: "evt_other", Sequence: 2, EventType: AccessEventEnabled, Payload: otherPayload}

	projection := Project("mis_1", ConnectorID, []ledger.Event{malformed, other})
	if projection.Enabled || projection.LastEventID != "" || projection.Status != AccessStatusDisabled {
		t.Fatalf("unexpected projection from ignored events: %#v", projection)
	}
}

func accessEvent(t *testing.T, eventID string, sequence int64, eventType, connectionID, cloudID, spaceKey string) ledger.Event {
	t.Helper()
	payload, err := json.Marshal(accessPayload{
		ConnectorID:  ConnectorID,
		ConnectionID: connectionID,
		CloudID:      cloudID,
		SpaceKey:     spaceKey,
	})
	if err != nil {
		t.Fatal(err)
	}
	return ledger.Event{EventID: eventID, Sequence: sequence, EventType: eventType, Payload: payload}
}
