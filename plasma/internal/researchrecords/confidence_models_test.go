package researchrecords

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func TestBuildClaimConfidenceUpdatePreservesValidationAndNormalizationOrder(t *testing.T) {
	var calls []string
	requirements := ConfidenceRequirements{
		GetClaimRecord: func(_ context.Context, claimID string) (ClaimRecord, error) {
			calls = append(calls, "claim:"+claimID)
			return ClaimRecord{ClaimID: claimID, MissionID: "mis_1"}, nil
		},
		RequireEvidenceRecords: func(_ context.Context, missionID string, ids []string) error {
			calls = append(calls, "evidence:"+missionID+":"+strings.Join(ids, ","))
			return nil
		},
	}
	appendRequest, err := BuildClaimConfidenceUpdate(context.Background(), requirements, UpdateClaimConfidenceRequest{
		EventID:          " evt_1 ",
		MissionID:        " mis_1 ",
		ClaimID:          " clm_1 ",
		Confidence:       Confidence{Level: " high ", Rationale: "  supported  ", OpenRisks: []string{" risk "}},
		BasisEvidenceIDs: []string{" evd_1 "},
		Origin:           "  ",
		Producer:         ledger.Producer{Type: " agent_session ", ID: " ses_1 "},
		CausationEventID: " evt_cause ",
		CorrelationID:    " corr_1 ",
	})
	if err != nil {
		t.Fatalf("BuildClaimConfidenceUpdate returned error: %v", err)
	}
	if got, want := calls, []string{"claim:clm_1", "evidence:mis_1:evd_1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("callback order = %#v, want %#v", got, want)
	}
	if appendRequest.EventID != "evt_1" || appendRequest.MissionID != "mis_1" || appendRequest.EventType != ClaimConfidenceUpdatedEvent {
		t.Fatalf("unexpected append identity: %#v", appendRequest)
	}
	if appendRequest.Producer != (ledger.Producer{Type: "agent_session", ID: "ses_1"}) || appendRequest.CausationEventID != "evt_cause" || appendRequest.CorrelationID != "corr_1" {
		t.Fatalf("unexpected append metadata: %#v", appendRequest)
	}
	var payload ClaimConfidenceUpdatePayload
	if err := json.Unmarshal(appendRequest.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Origin != "agent" || payload.ClaimID != "clm_1" || payload.Confidence.Rationale != "supported" || !reflect.DeepEqual(payload.BasisEvidenceIDs, []string{"evd_1"}) {
		t.Fatalf("unexpected normalized payload: %#v", payload)
	}
}

func TestBuildClaimConfidenceUpdateInvalidIDsDoNotLookup(t *testing.T) {
	lookups := 0
	requirements := ConfidenceRequirements{
		GetClaimRecord:         func(context.Context, string) (ClaimRecord, error) { lookups++; return ClaimRecord{}, nil },
		RequireEvidenceRecords: func(context.Context, string, []string) error { lookups++; return nil },
	}
	_, err := BuildClaimConfidenceUpdate(context.Background(), requirements, UpdateClaimConfidenceRequest{
		EventID: "bad", MissionID: "mis_1", ClaimID: "clm_1", Producer: ledger.Producer{Type: "user", ID: "u_1"},
	})
	if err == nil || lookups != 0 {
		t.Fatalf("expected invalid ID before lookup, err=%v lookups=%d", err, lookups)
	}
}

func TestBuildClaimConfidenceUpdateCrossMissionAndRationaleOrdering(t *testing.T) {
	evidenceLookups := 0
	requirements := ConfidenceRequirements{
		GetClaimRecord: func(context.Context, string) (ClaimRecord, error) {
			return ClaimRecord{ClaimID: "clm_1", MissionID: "mis_other"}, nil
		},
		RequireEvidenceRecords: func(context.Context, string, []string) error { evidenceLookups++; return nil },
	}
	_, err := BuildClaimConfidenceUpdate(context.Background(), requirements, UpdateClaimConfidenceRequest{
		EventID: "evt_1", MissionID: "mis_1", ClaimID: "clm_1", Confidence: Confidence{Level: "high", Rationale: "reason"}, Producer: ledger.Producer{Type: "user", ID: "u_1"},
	})
	if !errors.Is(err, producterror.ErrInvalidInput) || evidenceLookups != 0 {
		t.Fatalf("expected cross-mission rejection before evidence lookup, err=%v evidenceLookups=%d", err, evidenceLookups)
	}

	requirements.GetClaimRecord = func(context.Context, string) (ClaimRecord, error) {
		return ClaimRecord{ClaimID: "clm_1", MissionID: "mis_1"}, nil
	}
	_, err = BuildClaimConfidenceUpdate(context.Background(), requirements, UpdateClaimConfidenceRequest{
		EventID: "evt_1", MissionID: "mis_1", ClaimID: "clm_1", Confidence: Confidence{Level: "high"}, Producer: ledger.Producer{Type: "user", ID: "u_1"},
	})
	if !errors.Is(err, producterror.ErrInvalidInput) || evidenceLookups != 0 {
		t.Fatalf("expected rationale rejection before evidence lookup, err=%v evidenceLookups=%d", err, evidenceLookups)
	}
}

func TestBuildClaimConfidenceUpdateEmptyEvidenceInvokesCallback(t *testing.T) {
	called := false
	requirements := ConfidenceRequirements{
		GetClaimRecord: func(context.Context, string) (ClaimRecord, error) { return ClaimRecord{MissionID: "mis_1"}, nil },
		RequireEvidenceRecords: func(_ context.Context, missionID string, ids []string) error {
			called = true
			if missionID != "mis_1" || len(ids) != 0 {
				t.Fatalf("unexpected empty evidence callback arguments: %q %#v", missionID, ids)
			}
			return nil
		},
	}
	_, err := BuildClaimConfidenceUpdate(context.Background(), requirements, UpdateClaimConfidenceRequest{
		EventID: "evt_1", MissionID: "mis_1", ClaimID: "clm_1", Confidence: Confidence{Level: "medium", Rationale: "reason"}, Producer: ledger.Producer{Type: "user", ID: "u_1"},
	})
	if err != nil || !called {
		t.Fatalf("empty evidence callback was not invoked, err=%v called=%v", err, called)
	}
}

func TestClaimConfidenceProjectionDecodesOnlyKnownValidEventsAndPreservesMetadata(t *testing.T) {
	createdAt := time.Date(2026, 9, 8, 10, 11, 12, 0, time.FixedZone("test", 3600))
	validPayload, err := json.Marshal(ClaimConfidenceUpdatePayload{
		ClaimID: " clm_1 ", Confidence: Confidence{Level: " high ", Rationale: " reason "}, BasisEvidenceIDs: []string{" evd_1 "},
	})
	if err != nil {
		t.Fatal(err)
	}
	valid := ledger.Event{EventID: "evt_1", MissionID: "mis_1", Sequence: 7, EventType: ClaimConfidenceUpdatedEvent, Producer: ledger.Producer{Type: "user ", ID: "ui"}, Payload: validPayload, CreatedAt: createdAt}
	update, ok := ClaimConfidenceUpdateFromEvent(valid)
	if !ok {
		t.Fatal("valid confidence event was rejected")
	}
	if update.EventID != "evt_1" || update.MissionID != "mis_1" || update.Sequence != 7 || !update.CreatedAt.Equal(createdAt) || update.Origin != "user" {
		t.Fatalf("projection metadata mismatch: %#v", update)
	}
	if update.ClaimID != "clm_1" || update.Confidence.Level != "high" || update.Confidence.Rationale != "reason" || !reflect.DeepEqual(update.BasisEvidenceIDs, []string{"evd_1"}) {
		t.Fatalf("projection normalization mismatch: %#v", update)
	}

	for _, event := range []ledger.Event{
		{EventType: ClaimConfidenceUpdatedEvent},
		{EventType: ClaimConfidenceUpdatedEvent, Payload: []byte("{")},
		{EventType: "unknown.event", Payload: validPayload},
	} {
		if _, ok := ClaimConfidenceUpdateFromEvent(event); ok {
			t.Fatalf("malformed or unknown event decoded: %#v", event)
		}
	}
	updates := ClaimConfidenceUpdatesFromEvents([]ledger.Event{
		{EventID: "evt_bad", EventType: "unknown.event", Payload: validPayload},
		valid,
		{EventID: "evt_2", MissionID: "mis_1", Sequence: 8, EventType: ClaimConfidenceUpdatedEvent, Producer: ledger.Producer{Type: "agent", ID: "a_1"}, Payload: validPayload, CreatedAt: createdAt.Add(time.Minute)},
	})
	if len(updates) != 2 || updates[0].EventID != "evt_1" || updates[1].EventID != "evt_2" || !updates[1].CreatedAt.After(updates[0].CreatedAt) {
		t.Fatalf("projection order mismatch: %#v", updates)
	}
}
