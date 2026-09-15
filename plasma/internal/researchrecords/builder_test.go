package researchrecords

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/source"
)

type fakeSnapshotReader struct {
	snapshots map[string]source.Snapshot
	errors    map[string]error
	reads     []string
}

func (f *fakeSnapshotReader) GetSourceSnapshot(_ context.Context, id string) (source.Snapshot, error) {
	f.reads = append(f.reads, id)
	if err := f.errors[id]; err != nil {
		return source.Snapshot{}, err
	}
	return f.snapshots[id], nil
}

func validRequest() CreateEvidenceRecordRequest {
	return CreateEvidenceRecordRequest{
		EvidenceID:     "evd_1",
		MissionID:      "mis_1",
		State:          " proposed ",
		Summary:        "  grounded summary  ",
		EvidenceType:   " quote ",
		SnapshotRefs:   []SnapshotRef{{SnapshotID: " src_1 ", ArtifactID: " art_1 ", Locator: []byte(`{"line":1}`)}},
		Confidence:     Confidence{Level: " medium ", Rationale: "  rationale ", OpenRisks: []string{"  risk ", "  "}},
		Producer:       ledger.Producer{Type: " agent_session ", ID: " ses_1 "},
		CreatedEventID: " evt_1 ",
	}
}

func validEvent() ledger.Event {
	return ledger.Event{EventID: "evt_1", MissionID: "mis_1", Producer: ledger.Producer{Type: "agent_session", ID: "ses_1"}}
}

func validReader() *fakeSnapshotReader {
	return &fakeSnapshotReader{snapshots: map[string]source.Snapshot{
		"src_1": {SnapshotID: "src_1", MissionID: "mis_1", ArtifactIDs: []string{"art_1"}},
		"src_2": {SnapshotID: "src_2", MissionID: "mis_1", ArtifactIDs: []string{"art_2"}},
	}}
}

func TestBuildEvidenceRecordRequestValidationDoesNotReadSnapshots(t *testing.T) {
	cases := []struct {
		name string
		edit func(*CreateEvidenceRecordRequest, *ledger.Event)
		want string
	}{
		{name: "bad evidence id", edit: func(req *CreateEvidenceRecordRequest, _ *ledger.Event) { req.EvidenceID = "bad" }, want: "invalid input: id must start with evd_"},
		{name: "bad mission id", edit: func(req *CreateEvidenceRecordRequest, _ *ledger.Event) { req.MissionID = "bad" }, want: "invalid input: id must start with mis_"},
		{name: "empty summary", edit: func(req *CreateEvidenceRecordRequest, _ *ledger.Event) { req.Summary = " " }, want: "invalid input: evidence summary is required"},
		{name: "unsupported evidence type", edit: func(req *CreateEvidenceRecordRequest, _ *ledger.Event) { req.EvidenceType = "unsupported" }, want: "invalid input: unsupported evidence type"},
		{name: "invalid state", edit: func(req *CreateEvidenceRecordRequest, _ *ledger.Event) { req.State = "approved" }, want: "invalid input: terminal lifecycle state requires a transition event"},
		{name: "empty producer", edit: func(req *CreateEvidenceRecordRequest, _ *ledger.Event) { req.Producer = ledger.Producer{} }, want: "invalid input: producer type and id are required"},
		{name: "event mismatch", edit: func(_ *CreateEvidenceRecordRequest, event *ledger.Event) { event.MissionID = "mis_other" }, want: "invalid input: evidence creation event mismatch"},
		{name: "user assertion producer", edit: func(req *CreateEvidenceRecordRequest, event *ledger.Event) {
			req.EvidenceType = "user_assertion"
			req.SnapshotRefs = nil
			event.Producer = ledger.Producer{Type: "autopilot", ID: "ses_1"}
		}, want: "invalid input: user assertion evidence requires a user or steering_chat event"},
		{name: "empty refs", edit: func(req *CreateEvidenceRecordRequest, _ *ledger.Event) { req.SnapshotRefs = nil }, want: "invalid input: evidence requires snapshot refs unless it is a user assertion"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, event, reader := validRequest(), validEvent(), validReader()
			tc.edit(&req, &event)
			_, err := BuildEvidenceRecord(context.Background(), reader, req, event)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
			if len(reader.reads) != 0 {
				t.Fatalf("snapshot reads = %#v, want zero", reader.reads)
			}
		})
	}
}

func TestBuildEvidenceRecordDuplicateRefsShortCircuitInOrder(t *testing.T) {
	reader := validReader()
	req := validRequest()
	req.SnapshotRefs = []SnapshotRef{
		{SnapshotID: "src_1", ArtifactID: "art_1"},
		{SnapshotID: "src_1", ArtifactID: "art_1"},
		{SnapshotID: "src_2", ArtifactID: "art_2"},
	}
	_, err := BuildEvidenceRecord(context.Background(), reader, req, validEvent())
	if err == nil || !strings.Contains(err.Error(), "duplicate snapshot ref") {
		t.Fatalf("error = %v, want duplicate snapshot ref", err)
	}
	if got := strings.Join(reader.reads, ","); got != "src_1" {
		t.Fatalf("snapshot reads = %q, want src_1", got)
	}
}

func TestBuildEvidenceRecordSourceValidationPrecedesLocatorAndConfidence(t *testing.T) {
	cases := []struct {
		name   string
		reader *fakeSnapshotReader
		ref    SnapshotRef
		want   string
	}{
		{name: "lookup error", reader: &fakeSnapshotReader{snapshots: map[string]source.Snapshot{}, errors: map[string]error{"src_1": errors.New("lookup failed")}}, ref: SnapshotRef{SnapshotID: "src_1", ArtifactID: "art_1", Locator: []byte("bad")}, want: "lookup failed"},
		{name: "mission mismatch", reader: &fakeSnapshotReader{snapshots: map[string]source.Snapshot{"src_1": {MissionID: "mis_other", ArtifactIDs: []string{"art_1"}}}}, ref: SnapshotRef{SnapshotID: "src_1", ArtifactID: "art_1", Locator: []byte("bad")}, want: "another mission"},
		{name: "artifact mismatch", reader: &fakeSnapshotReader{snapshots: map[string]source.Snapshot{"src_1": {MissionID: "mis_1", ArtifactIDs: []string{"art_other"}}}}, ref: SnapshotRef{SnapshotID: "src_1", ArtifactID: "art_1", Locator: []byte("bad")}, want: "not linked"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := validRequest()
			req.SnapshotRefs = []SnapshotRef{tc.ref}
			req.Confidence = Confidence{Level: "invalid"}
			_, err := BuildEvidenceRecord(context.Background(), tc.reader, req, validEvent())
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
			if len(tc.reader.reads) != 1 || tc.reader.reads[0] != "src_1" {
				t.Fatalf("snapshot reads = %#v, want [src_1]", tc.reader.reads)
			}
		})
	}
}

func TestBuildEvidenceRecordNormalizesLocatorConfidenceAndRefs(t *testing.T) {
	reader := validReader()
	req := validRequest()
	req.SnapshotRefs = []SnapshotRef{{SnapshotID: " src_1 ", ArtifactID: " art_1 "}, {SnapshotID: "src_2", ArtifactID: "art_2", Locator: []byte(`{"quote":"ok"}`)}}
	record, err := BuildEvidenceRecord(context.Background(), reader, req, validEvent())
	if err != nil {
		t.Fatal(err)
	}
	if record.CreatedAt.IsZero() || record.CreatedAt.Location() != time.UTC {
		t.Fatalf("created_at = %#v, want UTC", record.CreatedAt)
	}
	if record.EvidenceID != "evd_1" || record.MissionID != "mis_1" || record.Summary != "grounded summary" || record.State != "proposed" || record.EvidenceType != "quote" {
		t.Fatalf("normalized record = %#v", record)
	}
	if record.Producer.Type != "agent_session" || record.Producer.ID != "ses_1" || record.Confidence.Level != "medium" || record.Confidence.Rationale != "rationale" || len(record.Confidence.OpenRisks) != 1 || record.Confidence.OpenRisks[0] != "risk" {
		t.Fatalf("normalized metadata = %#v", record)
	}
	if string(record.SnapshotRefs[0].Locator) != `{}` || string(record.SnapshotRefs[1].Locator) != `{"quote":"ok"}` {
		t.Fatalf("normalized locators = %#v", record.SnapshotRefs)
	}
	if strings.Join(reader.reads, ",") != "src_1,src_2" {
		t.Fatalf("snapshot reads = %#v", reader.reads)
	}
}

func TestBuildEvidenceRecordAllowedTypesConfidenceAndLifecycleErrors(t *testing.T) {
	for _, evidenceType := range []string{"quote", "fact", "table_row", "statistic", "observation", "interpretation", "reaction", "rumor", "controversy", "market_signal", "code", "formula", "benchmark", "open_question", "user_assertion"} {
		t.Run("type/"+evidenceType, func(t *testing.T) {
			req := validRequest()
			req.EvidenceType = evidenceType
			if evidenceType == "user_assertion" {
				req.SnapshotRefs = nil
				event := validEvent()
				event.Producer.Type = "user"
				if _, err := BuildEvidenceRecord(context.Background(), validReader(), req, event); err != nil {
					t.Fatal(err)
				}
				return
			}
			if _, err := BuildEvidenceRecord(context.Background(), validReader(), req, validEvent()); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, level := range []string{"low", "medium", "high", "unknown", ""} {
		t.Run("confidence/"+level, func(t *testing.T) {
			req := validRequest()
			req.Confidence.Level = level
			if _, err := BuildEvidenceRecord(context.Background(), validReader(), req, validEvent()); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, state := range []string{"draft", "proposed", "needs_review", "", "approved", "rejected", "superseded", "archived", "invalid"} {
		t.Run("lifecycle/"+state, func(t *testing.T) {
			req := validRequest()
			req.State = state
			_, err := BuildEvidenceRecord(context.Background(), validReader(), req, validEvent())
			if state == "approved" || state == "rejected" || state == "superseded" || state == "archived" {
				if err == nil || !strings.Contains(err.Error(), "terminal lifecycle state requires a transition event") {
					t.Fatalf("error = %v", err)
				}
				return
			}
			if state == "invalid" {
				if err == nil || !strings.Contains(err.Error(), "unsupported lifecycle state") {
					t.Fatalf("error = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
	if _, err := NormalizeConfidence(Confidence{Level: "invalid"}); !errors.Is(err, producterror.ErrInvalidInput) {
		t.Fatalf("confidence error = %v", err)
	}
}
