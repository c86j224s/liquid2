package app

import "github.com/c86j224s/liquid2/plasma/internal/reporting/reportdocument"

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/c86j224s/liquid2/plasma/internal/researchproposal"
	"github.com/c86j224s/liquid2/plasma/internal/researchrecords"
	"reflect"
	"testing"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
)

type fakeStore struct {
	healthErr     error
	migrationErr  error
	migrationList []string
}

func (f fakeStore) Health(context.Context) error {
	return f.healthErr
}

func (f fakeStore) MigrationVersions(context.Context) ([]string, error) {
	if f.migrationErr != nil {
		return nil, f.migrationErr
	}
	return f.migrationList, nil
}

func (f fakeStore) CreateMission(context.Context, mission.Mission) error {
	return nil
}

func (f fakeStore) AppendLedgerEvent(_ context.Context, event ledger.Event) (ledger.Event, error) {
	event.Sequence = 1
	return event, nil
}

func (f fakeStore) ListLedgerEvents(context.Context, string) ([]ledger.Event, error) {
	return []ledger.Event{{EventID: "evt_1"}}, nil
}

func (f fakeStore) SaveMissionProjection(context.Context, mission.Projection) error {
	return nil
}

func (f fakeStore) GetMissionProjection(context.Context, string) (mission.Projection, error) {
	return mission.Projection{MissionID: "mis_1"}, nil
}

func (f fakeStore) CreateRawArtifact(context.Context, artifactcontract.Raw) error {
	return nil
}

func (f fakeStore) GetRawArtifact(context.Context, string) (artifactcontract.Raw, error) {
	return artifactcontract.Raw{ArtifactID: "art_1", MissionID: "mis_1"}, nil
}

func (f fakeStore) CreateSourceSnapshot(context.Context, sourcecontract.Snapshot) error {
	return nil
}

func (f fakeStore) GetSourceSnapshot(context.Context, string) (sourcecontract.Snapshot, error) {
	return sourcecontract.Snapshot{SnapshotID: "src_1", MissionID: "mis_1"}, nil
}

func (f fakeStore) CreateEvidenceRecord(context.Context, researchrecords.EvidenceRecord) error {
	return nil
}

func (f fakeStore) GetEvidenceRecord(context.Context, string) (researchrecords.EvidenceRecord, error) {
	return researchrecords.EvidenceRecord{EvidenceID: "evd_1", MissionID: "mis_1"}, nil
}

func (f fakeStore) CreateClaimRecord(context.Context, researchrecords.ClaimRecord) error {
	return nil
}

func (f fakeStore) GetClaimRecord(context.Context, string) (researchrecords.ClaimRecord, error) {
	return researchrecords.ClaimRecord{ClaimID: "clm_1", MissionID: "mis_1"}, nil
}

func (f fakeStore) CreateQuestionRecord(context.Context, researchrecords.QuestionRecord) error {
	return nil
}

func (f fakeStore) GetQuestionRecord(context.Context, string) (researchrecords.QuestionRecord, error) {
	return researchrecords.QuestionRecord{QuestionID: "qst_1", MissionID: "mis_1"}, nil
}

func (f fakeStore) CreateOptionRecord(context.Context, researchrecords.OptionRecord) error {
	return nil
}

func (f fakeStore) GetOptionRecord(context.Context, string) (researchrecords.OptionRecord, error) {
	return researchrecords.OptionRecord{OptionID: "opt_1", MissionID: "mis_1"}, nil
}

func (f fakeStore) CreateProposalBundle(context.Context, researchproposal.ProposalBundle) error {
	return nil
}

func (f fakeStore) GetProposalBundle(context.Context, string) (researchproposal.ProposalBundle, error) {
	return researchproposal.ProposalBundle{ProposalID: "prp_1", MissionID: "mis_1", State: "pending_review"}, nil
}

func (f fakeStore) UpdateProposalBundleState(context.Context, researchproposal.ProposalBundleStateUpdate) error {
	return nil
}

func (f fakeStore) CreateReport(context.Context, reportdocument.Report) error {
	return nil
}

func (f fakeStore) GetReport(context.Context, string) (reportdocument.Report, error) {
	return reportdocument.Report{ReportID: "rpt_1", MissionID: "mis_1"}, nil
}

func (f fakeStore) CreateReportVersion(context.Context, reportdocument.ReportVersion, []reportdocument.ReportBlock) error {
	return nil
}

func (f fakeStore) GetReportVersion(context.Context, string) (reportdocument.ReportVersion, error) {
	return reportdocument.ReportVersion{ReportVersionID: "rvn_1", ReportID: "rpt_1", MissionID: "mis_1"}, nil
}

func (f fakeStore) ListReportBlocks(context.Context, string) ([]reportdocument.ReportBlock, error) {
	return []reportdocument.ReportBlock{}, nil
}

func (f fakeStore) PromoteReportVersion(context.Context, reportdocument.ReportVersionPromotion) error {
	return nil
}

func TestHealthReturnsMigrationVersions(t *testing.T) {
	svc := NewService(fakeStore{migrationList: []string{"0001_bootstrap"}})
	health, err := svc.Health(context.Background())
	if err != nil {
		t.Fatalf("Health returned error: %v", err)
	}
	if health.Status != "ok" {
		t.Fatalf("expected ok status, got %q", health.Status)
	}
	if len(health.Migrations) != 1 || health.Migrations[0] != "0001_bootstrap" {
		t.Fatalf("unexpected migrations: %#v", health.Migrations)
	}
}

func TestHealthPropagatesStoreError(t *testing.T) {
	want := errors.New("boom")
	svc := NewService(fakeStore{healthErr: want})
	if _, err := svc.Health(context.Background()); !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

func TestMissionUseCasesValidateAndDelegate(t *testing.T) {
	svc := NewService(fakeStore{})
	mission, err := svc.CreateMission(context.Background(), mission.CreateRequest{
		MissionID: "mis_1",
		Title:     " Test mission ",
	})
	if err != nil {
		t.Fatalf("CreateMission returned error: %v", err)
	}
	if mission.Title != "Test mission" {
		t.Fatalf("expected trimmed title, got %q", mission.Title)
	}

	event, err := svc.AppendEvent(context.Background(), ledger.AppendRequest{
		EventID:   "evt_1",
		MissionID: "mis_1",
		EventType: "mission.created",
		Producer:  ledger.Producer{Type: "user", ID: "ses_1"},
	})
	if err != nil {
		t.Fatalf("AppendEvent returned error: %v", err)
	}
	if event.Sequence != 1 {
		t.Fatalf("expected delegated sequence, got %d", event.Sequence)
	}

	events, err := svc.ListEvents(context.Background(), "mis_1")
	if err != nil {
		t.Fatalf("ListEvents returned error: %v", err)
	}
	if !reflect.DeepEqual([]ledger.Event{{EventID: "evt_1"}}, events) {
		t.Fatalf("unexpected events: %#v", events)
	}
}

func TestBuildMissionCreatedAppendRequestPreservesPayloadContract(t *testing.T) {
	req := BuildMissionCreatedAppendRequest(mission.CreatedEventRequest{
		EventID:   "evt_mission",
		MissionID: "mis_1",
		Title:     "Mission title",
		Objective: "Mission objective",
		Scope: mission.Scope{
			Included: []string{"include-a", "include-b"},
			Excluded: []string{"exclude-a"},
		},
		Producer: ledger.Producer{Type: "user", ID: "plasma-ui"},
	})
	if req.EventID != "evt_mission" || req.MissionID != "mis_1" || req.EventType != "mission.created" ||
		req.Producer.Type != "user" || req.Producer.ID != "plasma-ui" {
		t.Fatalf("unexpected mission created event shell: %#v", req)
	}
	var payload struct {
		Title     string        `json:"title"`
		Objective string        `json:"objective"`
		Scope     mission.Scope `json:"scope"`
	}
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Title != "Mission title" || payload.Objective != "Mission objective" ||
		!reflect.DeepEqual(payload.Scope.Included, []string{"include-a", "include-b"}) ||
		!reflect.DeepEqual(payload.Scope.Excluded, []string{"exclude-a"}) {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestAppendEventRejectsInvalidPayload(t *testing.T) {
	svc := NewService(fakeStore{})
	_, err := svc.AppendEvent(context.Background(), ledger.AppendRequest{
		EventID:   "evt_1",
		MissionID: "mis_1",
		EventType: "mission.created",
		Producer:  ledger.Producer{Type: "user", ID: "ses_1"},
		Payload:   []byte(`{bad`),
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input error, got %v", err)
	}
}

func TestValidateIDRejectsBarePrefix(t *testing.T) {
	if err := validateID("mis_", "mis_"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input error, got %v", err)
	}
}
