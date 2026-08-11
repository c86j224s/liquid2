package partassembly

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestRecoverValidatesRangeArtifactAndMission(t *testing.T) {
	plan := reporting.SectionalReportPlan{Parts: []reporting.ReportPlanPart{{Title: "Part", Sections: []reporting.ReportPlanSection{{Title: "Section"}}}}}
	service := partRecoverStore{artifacts: map[string]artifact.Raw{
		"art_part":    {ArtifactID: "art_part", MissionID: "mis_1", MediaType: "text/markdown; charset=utf-8", Content: []byte("# Part\n\nBody text.")},
		"art_html":    {ArtifactID: "art_html", MissionID: "mis_1", MediaType: "text/html", Content: []byte("<p>Part</p>")},
		"art_foreign": {ArtifactID: "art_foreign", MissionID: "mis_other", MediaType: "text/markdown; charset=utf-8", Content: []byte("# Foreign")},
	}}

	out, ok, err := Recover(context.Background(), RecoverInput{
		Service: service, Event: partEvent("art_part", 1), MissionID: "mis_1",
		PendingEventID: "evt_pending", PlanEventID: "evt_plan", Plan: plan,
	})
	if err != nil || !ok {
		t.Fatalf("Recover valid ok=%t err=%v", ok, err)
	}
	if out.PartIndex != 0 || out.Draft.ArtifactID != "art_part" || out.Draft.WordCount == 0 {
		t.Fatalf("unexpected recovered Part: %#v", out)
	}

	for _, tc := range []struct {
		name  string
		event ledger.Event
	}{
		{name: "outside plan", event: partEvent("art_part", 2)},
		{name: "non markdown artifact", event: partEvent("art_html", 1)},
		{name: "foreign artifact", event: partEvent("art_foreign", 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, ok, err := Recover(context.Background(), RecoverInput{
				Service: service, Event: tc.event, MissionID: "mis_1",
				PendingEventID: "evt_pending", PlanEventID: "evt_plan", Plan: plan,
			})
			if err != nil || ok {
				t.Fatalf("Recover ok=%t err=%v, want ignored", ok, err)
			}
		})
	}
}

type partRecoverStore struct {
	artifacts map[string]artifact.Raw
}

func (store partRecoverStore) GetRawArtifact(_ context.Context, artifactID string) (artifact.Raw, error) {
	raw, ok := store.artifacts[artifactID]
	if !ok {
		return artifact.Raw{}, errors.New("artifact missing")
	}
	return raw, nil
}

func partEvent(artifactID string, partIndex int) ledger.Event {
	return ledger.Event{
		EventID: "evt_part", MissionID: "mis_1", EventType: CreatedEventType,
		Payload: mustPartRecoverJSON(map[string]any{
			"pending_event_id": "evt_pending", "plan_event_id": "evt_plan",
			"artifact_id": artifactID, "title": "Part", "agent_session_id": "provider-part",
			"part_index": partIndex,
		}),
	}
}

func mustPartRecoverJSON(value any) []byte {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}
