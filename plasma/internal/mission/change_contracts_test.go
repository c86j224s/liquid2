package mission

import (
	"bytes"
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func TestBuildMetadataUpdateSparseAndClearPayloads(t *testing.T) {
	title := " New title "
	objective := "  "
	req := UpdateMissionMetadataRequest{
		EventID:   "evt_update",
		MissionID: "mis_1",
		Producer:  ledger.Producer{Type: "user", ID: "u"},
		Title:     &title,
		Objective: &objective,
		Scope:     &Scope{Included: []string{" A ", " ", "B"}, Excluded: []string{" X ", ""}},
	}
	appendReq, err := BuildMetadataUpdate(req)
	if err != nil {
		t.Fatal(err)
	}
	if appendReq.EventID != req.EventID || appendReq.MissionID != req.MissionID || appendReq.EventType != "mission.metadata.updated" || appendReq.Producer != req.Producer {
		t.Fatalf("append request = %#v", appendReq)
	}
	want := `{"objective":"","scope":{"included":["A","B"],"excluded":["X"]},"title":"New title"}`
	if string(appendReq.Payload) != want {
		t.Fatalf("payload = %s, want %s", appendReq.Payload, want)
	}
}

func TestBuildMetadataUpdateValidationAndPresence(t *testing.T) {
	title := "  "
	tests := []struct {
		name string
		req  UpdateMissionMetadataRequest
	}{
		{name: "no fields", req: UpdateMissionMetadataRequest{Producer: ledger.Producer{Type: "user"}}},
		{name: "blank title", req: UpdateMissionMetadataRequest{Title: &title, Producer: ledger.Producer{Type: "user"}}},
		{name: "non-user producer", req: UpdateMissionMetadataRequest{Objective: stringPtr("x"), Producer: ledger.Producer{Type: "agent", ID: "a"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := BuildMetadataUpdate(test.req); !errors.Is(err, producterror.ErrInvalidInput) {
				t.Fatalf("error = %v, want invalid input", err)
			}
		})
	}
}

func TestBuildMetadataUpdateUserOnlyAndNilEmptyScope(t *testing.T) {
	objective := " objective "
	appendReq, err := BuildMetadataUpdate(UpdateMissionMetadataRequest{
		EventID: "evt_update", MissionID: "mis_1", Producer: ledger.Producer{Type: "user", ID: "u"}, Objective: &objective,
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(appendReq.Payload) != `{"objective":"objective"}` {
		t.Fatalf("user-only payload = %s", appendReq.Payload)
	}

	for _, scope := range []*Scope{nil, &Scope{Included: []string{}, Excluded: []string{}}} {
		t.Run(scopeName(scope), func(t *testing.T) {
			req := UpdateMissionMetadataRequest{EventID: "evt_update", MissionID: "mis_1", Producer: ledger.Producer{Type: "user", ID: "u"}, Objective: &objective, Scope: scope}
			appendReq, err := BuildMetadataUpdate(req)
			if err != nil {
				t.Fatal(err)
			}
			if scope == nil && bytes.Contains(appendReq.Payload, []byte(`"scope"`)) {
				t.Fatalf("nil scope unexpectedly present: %s", appendReq.Payload)
			}
			if scope != nil && !bytes.Contains(appendReq.Payload, []byte(`"scope":{"included":[],"excluded":[]}`)) {
				t.Fatalf("empty scope missing: %s", appendReq.Payload)
			}
		})
	}
}

func TestBuildLifecycleChangeValidationPriorityAndPayload(t *testing.T) {
	tests := []struct {
		name    string
		req     MissionLifecycleChangeRequest
		wantMsg string
	}{
		{name: "event id first", req: MissionLifecycleChangeRequest{EventID: "bad", MissionID: "bad", Producer: ledger.Producer{Type: "agent", ID: "a"}}, wantMsg: "evt_"},
		{name: "mission id second", req: MissionLifecycleChangeRequest{EventID: "evt_1", MissionID: "bad", Producer: ledger.Producer{Type: "agent", ID: "a"}}, wantMsg: "mis_"},
		{name: "producer third", req: MissionLifecycleChangeRequest{EventID: "evt_1", MissionID: "mis_1", Producer: ledger.Producer{Type: "agent", ID: "a"}}, wantMsg: "user producer"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := BuildLifecycleChange(test.req, LifecycleArchived, ArchivedEvent)
			if !errors.Is(err, producterror.ErrInvalidInput) || !bytes.Contains([]byte(err.Error()), []byte(test.wantMsg)) {
				t.Fatalf("error = %v", err)
			}
		})
	}

	req := MissionLifecycleChangeRequest{EventID: " evt_1 ", MissionID: " mis_1 ", Producer: ledger.Producer{Type: "user", ID: "u"}, Reason: " done "}
	appendReq, err := BuildLifecycleChange(req, LifecycleArchived, ArchivedEvent)
	if err != nil {
		t.Fatal(err)
	}
	if appendReq.EventID != req.EventID || appendReq.MissionID != req.MissionID {
		t.Fatalf("raw ids changed: %#v", appendReq)
	}
	if string(appendReq.Payload) != `{"lifecycle_state":"archived","reason":"done"}` {
		t.Fatalf("payload = %s", appendReq.Payload)
	}
}

func TestLifecycleChangeNeededMissingMissionIdempotentAndNormalization(t *testing.T) {
	if _, err := LifecycleChangeNeeded("mis_1", nil, LifecycleArchived); !errors.Is(err, producterror.ErrInvalidInput) {
		t.Fatalf("missing mission error = %v", err)
	}
	events := []ledger.Event{{EventID: "evt_created", MissionID: "mis_1", EventType: "mission.created", Payload: []byte(`{"title":"Mission"}`)}}
	needed, err := LifecycleChangeNeeded("mis_1", events, LifecycleActive)
	if err != nil || needed {
		t.Fatalf("active idempotency = %v, %v", needed, err)
	}
	needed, err = LifecycleChangeNeeded("mis_1", events, LifecycleArchived)
	if err != nil || !needed {
		t.Fatalf("archive needed = %v, %v", needed, err)
	}
	if got := NormalizeLifecycleState(" archived "); got != LifecycleArchived {
		t.Fatalf("archived normalization = %q", got)
	}
	if got := NormalizeLifecycleState("garbage"); got != LifecycleActive {
		t.Fatalf("non-archived normalization = %q", got)
	}
}

func stringPtr(value string) *string { return &value }

func scopeName(scope *Scope) string {
	if scope == nil {
		return "nil scope"
	}
	return "empty scope"
}
