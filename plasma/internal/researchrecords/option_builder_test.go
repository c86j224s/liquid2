package researchrecords

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

func TestBuildOptionRecordOrdersValidationAndCallbacks(t *testing.T) {
	var calls []string
	eventErr := errors.New("event failed")
	requirements := OptionRequirements{
		RequireMissionEvent: func(_ context.Context, missionID, eventID string) (ledger.Event, error) {
			calls = append(calls, "event:"+missionID+":"+eventID)
			return ledger.Event{}, eventErr
		},
		RequireClaimRecords: func(_ context.Context, missionID string, ids []string) error {
			calls = append(calls, "claim:"+missionID+":"+strings.Join(ids, ","))
			return nil
		},
	}
	if _, err := BuildOptionRecord(context.Background(), requirements, CreateOptionRecordRequest{OptionID: "bad", MissionID: "mis_1", Title: "title", CreatedEventID: "evt_1"}); err == nil || len(calls) != 0 {
		t.Fatalf("invalid ID should precede callbacks: err=%v calls=%v", err, calls)
	}
	if _, err := BuildOptionRecord(context.Background(), requirements, CreateOptionRecordRequest{OptionID: "opt_1", MissionID: "mis_1", Title: "", CreatedEventID: "evt_1"}); err == nil || len(calls) != 0 {
		t.Fatalf("invalid title should precede callbacks: err=%v calls=%v", err, calls)
	}
	if _, err := BuildOptionRecord(context.Background(), requirements, CreateOptionRecordRequest{OptionID: "opt_1", MissionID: "mis_1", Title: "title", CreatedEventID: "evt_1"}); !errors.Is(err, eventErr) {
		t.Fatalf("expected event error, got %v", err)
	}
	if want := []string{"event:mis_1:evt_1"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("callback order mismatch: got=%v want=%v", calls, want)
	}
}

func TestBuildOptionRecordOrdersEventStateClaimRiskAndNormalizes(t *testing.T) {
	var calls []string
	requirements := OptionRequirements{
		RequireMissionEvent: func(_ context.Context, missionID, eventID string) (ledger.Event, error) {
			calls = append(calls, "event:"+missionID+":"+eventID)
			return ledger.Event{MissionID: missionID, EventID: eventID}, nil
		},
		RequireClaimRecords: func(_ context.Context, missionID string, ids []string) error {
			calls = append(calls, "claim:"+missionID+":"+strings.Join(ids, ","))
			return errors.New("claim failed")
		},
	}
	if _, err := BuildOptionRecord(context.Background(), requirements, CreateOptionRecordRequest{
		OptionID: "opt_1", MissionID: "mis_1", State: "bad", Title: "title", SupportingClaimIDs: []string{"clm_1"}, CreatedEventID: "evt_1",
	}); err == nil || !strings.Contains(err.Error(), "lifecycle") {
		t.Fatalf("expected state error before claims, got %v", err)
	}
	if want := []string{"event:mis_1:evt_1"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("state callback order mismatch: got=%v want=%v", calls, want)
	}
	calls = nil
	if _, err := BuildOptionRecord(context.Background(), requirements, CreateOptionRecordRequest{
		OptionID: "opt_1", MissionID: "mis_1", Title: "title", SupportingClaimIDs: []string{"clm_1"}, CreatedEventID: "evt_1", RiskLevel: "invalid",
	}); err == nil || !strings.Contains(err.Error(), "claim failed") {
		t.Fatalf("expected claim error before risk, got %v", err)
	}
	if want := []string{"event:mis_1:evt_1", "claim:mis_1:clm_1"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("claim callback order mismatch: got=%v want=%v", calls, want)
	}

	calls = nil
	requirements.RequireClaimRecords = func(_ context.Context, missionID string, ids []string) error {
		calls = append(calls, "claim:"+missionID+":"+strings.Join(ids, ","))
		return nil
	}
	record, err := BuildOptionRecord(context.Background(), requirements, CreateOptionRecordRequest{
		OptionID: " opt_1 ", MissionID: " mis_1 ", Title: " title ", Description: " description ", Pros: []string{" pro ", " ", "second "}, Cons: nil, CreatedEventID: " evt_1 ",
	})
	if err != nil {
		t.Fatalf("BuildOptionRecord returned error: %v", err)
	}
	if record.RiskLevel != "unknown" || record.State != "proposed" || record.Title != "title" || record.Description != "description" || record.CreatedAt.Location() != time.UTC {
		t.Fatalf("unexpected normalized option: %#v", record)
	}
	if !reflect.DeepEqual(record.Pros, []string{"pro", "second"}) || record.Cons == nil || len(record.Cons) != 0 {
		t.Fatalf("unexpected nil-list normalization: %#v", record)
	}
	if want := []string{"event:mis_1: evt_1 ", "claim:mis_1:"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("empty claim callback mismatch: got=%v want=%v", calls, want)
	}
}
