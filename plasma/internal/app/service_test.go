package app

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
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

func (f fakeStore) CreateMission(context.Context, Mission) error {
	return nil
}

func (f fakeStore) AppendLedgerEvent(_ context.Context, event LedgerEvent) (LedgerEvent, error) {
	event.Sequence = 1
	return event, nil
}

func (f fakeStore) ListLedgerEvents(context.Context, string) ([]LedgerEvent, error) {
	return []LedgerEvent{{EventID: "evt_1"}}, nil
}

func (f fakeStore) SaveMissionProjection(context.Context, MissionProjection) error {
	return nil
}

func (f fakeStore) GetMissionProjection(context.Context, string) (MissionProjection, error) {
	return MissionProjection{MissionID: "mis_1"}, nil
}

func (f fakeStore) CreateRawArtifact(context.Context, RawArtifact) error {
	return nil
}

func (f fakeStore) GetRawArtifact(context.Context, string) (RawArtifact, error) {
	return RawArtifact{ArtifactID: "art_1", MissionID: "mis_1"}, nil
}

func (f fakeStore) CreateSourceSnapshot(context.Context, SourceSnapshot) error {
	return nil
}

func (f fakeStore) GetSourceSnapshot(context.Context, string) (SourceSnapshot, error) {
	return SourceSnapshot{SnapshotID: "src_1", MissionID: "mis_1"}, nil
}

func (f fakeStore) CreateEvidenceRecord(context.Context, EvidenceRecord) error {
	return nil
}

func (f fakeStore) GetEvidenceRecord(context.Context, string) (EvidenceRecord, error) {
	return EvidenceRecord{EvidenceID: "evd_1", MissionID: "mis_1"}, nil
}

func (f fakeStore) CreateClaimRecord(context.Context, ClaimRecord) error {
	return nil
}

func (f fakeStore) GetClaimRecord(context.Context, string) (ClaimRecord, error) {
	return ClaimRecord{ClaimID: "clm_1", MissionID: "mis_1"}, nil
}

func (f fakeStore) CreateQuestionRecord(context.Context, QuestionRecord) error {
	return nil
}

func (f fakeStore) GetQuestionRecord(context.Context, string) (QuestionRecord, error) {
	return QuestionRecord{QuestionID: "qst_1", MissionID: "mis_1"}, nil
}

func (f fakeStore) CreateOptionRecord(context.Context, OptionRecord) error {
	return nil
}

func (f fakeStore) GetOptionRecord(context.Context, string) (OptionRecord, error) {
	return OptionRecord{OptionID: "opt_1", MissionID: "mis_1"}, nil
}

func (f fakeStore) CreateProposalBundle(context.Context, ProposalBundle) error {
	return nil
}

func (f fakeStore) GetProposalBundle(context.Context, string) (ProposalBundle, error) {
	return ProposalBundle{ProposalID: "prp_1", MissionID: "mis_1", State: "pending_review"}, nil
}

func (f fakeStore) UpdateProposalBundleState(context.Context, ProposalBundleStateUpdate) error {
	return nil
}

func (f fakeStore) CreateReport(context.Context, Report) error {
	return nil
}

func (f fakeStore) GetReport(context.Context, string) (Report, error) {
	return Report{ReportID: "rpt_1", MissionID: "mis_1"}, nil
}

func (f fakeStore) CreateReportVersion(context.Context, ReportVersion, []ReportBlock) error {
	return nil
}

func (f fakeStore) GetReportVersion(context.Context, string) (ReportVersion, error) {
	return ReportVersion{ReportVersionID: "rvn_1", ReportID: "rpt_1", MissionID: "mis_1"}, nil
}

func (f fakeStore) ListReportBlocks(context.Context, string) ([]ReportBlock, error) {
	return []ReportBlock{}, nil
}

func (f fakeStore) PromoteReportVersion(context.Context, ReportVersionPromotion) error {
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
	mission, err := svc.CreateMission(context.Background(), CreateMissionRequest{
		MissionID: "mis_1",
		Title:     " Test mission ",
	})
	if err != nil {
		t.Fatalf("CreateMission returned error: %v", err)
	}
	if mission.Title != "Test mission" {
		t.Fatalf("expected trimmed title, got %q", mission.Title)
	}

	event, err := svc.AppendEvent(context.Background(), AppendEventRequest{
		EventID:   "evt_1",
		MissionID: "mis_1",
		EventType: "mission.created",
		Producer:  Producer{Type: "user", ID: "ses_1"},
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
	if !reflect.DeepEqual([]LedgerEvent{{EventID: "evt_1"}}, events) {
		t.Fatalf("unexpected events: %#v", events)
	}
}

func TestBuildMissionCreatedAppendRequestPreservesPayloadContract(t *testing.T) {
	req := BuildMissionCreatedAppendRequest(MissionCreatedEventRequest{
		EventID:   "evt_mission",
		MissionID: "mis_1",
		Title:     "Mission title",
		Objective: "Mission objective",
		Scope: MissionScope{
			Included: []string{"include-a", "include-b"},
			Excluded: []string{"exclude-a"},
		},
		Producer: Producer{Type: "user", ID: "plasma-ui"},
	})
	if req.EventID != "evt_mission" || req.MissionID != "mis_1" || req.EventType != "mission.created" ||
		req.Producer.Type != "user" || req.Producer.ID != "plasma-ui" {
		t.Fatalf("unexpected mission created event shell: %#v", req)
	}
	var payload struct {
		Title     string       `json:"title"`
		Objective string       `json:"objective"`
		Scope     MissionScope `json:"scope"`
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
	_, err := svc.AppendEvent(context.Background(), AppendEventRequest{
		EventID:   "evt_1",
		MissionID: "mis_1",
		EventType: "mission.created",
		Producer:  Producer{Type: "user", ID: "ses_1"},
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
