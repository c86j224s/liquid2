package app

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"reflect"
	"testing"
)

type metadataStore struct {
	fakeStore
	events []ledger.Event
	saved  mission.Projection
}

func (s *metadataStore) AppendLedgerEvent(_ context.Context, event ledger.Event) (ledger.Event, error) {
	event.Sequence = int64(len(s.events) + 1)
	s.events = append(s.events, event)
	return event, nil
}
func (s *metadataStore) ListLedgerEvents(context.Context, string) ([]ledger.Event, error) {
	return s.events, nil
}
func (s *metadataStore) SaveMissionProjection(_ context.Context, projection mission.Projection) error {
	s.saved = projection
	return nil
}

func ptr(value string) *string { return &value }

func TestUpdateMissionMetadataValidation(t *testing.T) {
	svc := NewService(&metadataStore{})
	tests := []mission.UpdateMissionMetadataRequest{
		{EventID: "evt_1", MissionID: "mis_1", Producer: ledger.Producer{Type: "user", ID: "u"}},
		{EventID: "evt_1", MissionID: "mis_1", Producer: ledger.Producer{Type: "agent", ID: "a"}, Title: ptr("Title")},
		{EventID: "evt_1", MissionID: "mis_1", Producer: ledger.Producer{Type: "user", ID: "u"}, Title: ptr(" \t ")},
	}
	for _, req := range tests {
		if _, err := svc.UpdateMissionMetadata(context.Background(), req); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("expected invalid input for %#v, got %v", req, err)
		}
	}
}

func TestUpdateMissionMetadataSparsePayloadAndRebuild(t *testing.T) {
	store := &metadataStore{events: []ledger.Event{{EventID: "evt_created", MissionID: "mis_1", Sequence: 1, EventType: "mission.created", Producer: ledger.Producer{Type: "user", ID: "u"}, Payload: json.RawMessage(`{"title":"Old","objective":"Keep"}`)}}}
	svc := NewService(store)
	result, err := svc.UpdateMissionMetadata(context.Background(), mission.UpdateMissionMetadataRequest{
		EventID: "evt_update", MissionID: "mis_1", Producer: ledger.Producer{Type: "user", ID: "u"}, Title: ptr(" New "),
		Scope: &mission.Scope{Included: []string{" A ", " ", "B"}, Excluded: []string{" X ", ""}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(result.Event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"title": "New", "scope": map[string]any{"included": []any{"A", "B"}, "excluded": []any{"X"}}}
	if !reflect.DeepEqual(payload, want) {
		t.Fatalf("payload = %#v, want %#v", payload, want)
	}
	if result.Projection.Title != "New" || result.Projection.Objective != "Keep" || store.saved.LastEventID != "evt_update" {
		t.Fatalf("unexpected rebuilt projection: %#v", result.Projection)
	}
}

func TestUpdateMissionMetadataAllowsExplicitClears(t *testing.T) {
	store := &metadataStore{events: []ledger.Event{{EventID: "evt_created", MissionID: "mis_1", Sequence: 1, EventType: "mission.created", Producer: ledger.Producer{Type: "user", ID: "u"}, Payload: json.RawMessage(`{"title":"Old","objective":"Keep","scope":{"included":["A"]}}`)}}}
	result, err := NewService(store).UpdateMissionMetadata(context.Background(), mission.UpdateMissionMetadataRequest{EventID: "evt_update", MissionID: "mis_1", Producer: ledger.Producer{Type: "user", ID: "u"}, Objective: ptr(""), Scope: &mission.Scope{Included: []string{}, Excluded: []string{}}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Projection.Objective != "" || len(result.Projection.Scope.Included) != 0 {
		t.Fatalf("clear failed: %#v", result.Projection)
	}
}
