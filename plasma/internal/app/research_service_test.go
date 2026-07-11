package app

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCreateEvidenceRecordRequiresSnapshotUnlessUserAssertion(t *testing.T) {
	store := newResearchFakeStore()
	svc := NewService(store)

	_, err := svc.CreateEvidenceRecord(context.Background(), CreateEvidenceRecordRequest{
		EvidenceID:     "evd_missing_snapshot",
		MissionID:      "mis_1",
		Summary:        "External fact without a pinned source",
		EvidenceType:   "fact",
		Producer:       Producer{Type: "autopilot", ID: "ses_auto"},
		CreatedEventID: "evt_evidence",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}

	record, err := svc.CreateEvidenceRecord(context.Background(), CreateEvidenceRecordRequest{
		EvidenceID:     "evd_user_assertion",
		MissionID:      "mis_1",
		Summary:        "The user wants Plasma storage isolated from Liquid2.",
		EvidenceType:   "user_assertion",
		Producer:       Producer{Type: "user", ID: "ses_user"},
		CreatedEventID: "evt_user",
	})
	if err != nil {
		t.Fatalf("CreateEvidenceRecord user assertion returned error: %v", err)
	}
	if record.State != "proposed" || record.Confidence.Level != "unknown" {
		t.Fatalf("unexpected user assertion defaults: %#v", record)
	}
}

func TestCreateEvidenceRecordAcceptsResearchSignalTypes(t *testing.T) {
	store := newResearchFakeStore()
	svc := NewService(store)
	store.events = append(store.events, LedgerEvent{
		EventID:   "evt_evidence_reaction",
		MissionID: "mis_1",
		EventType: "evidence.proposed",
		Producer:  Producer{Type: "autopilot", ID: "ses_auto"},
		Payload:   []byte(`{"evidence_id":"evd_reaction"}`),
		CreatedAt: time.Now().UTC(),
	})

	record, err := svc.CreateEvidenceRecord(context.Background(), CreateEvidenceRecordRequest{
		EvidenceID:   "evd_reaction",
		MissionID:    "mis_1",
		Summary:      "Audience reactions consistently mention the same concern.",
		EvidenceType: "reaction",
		SnapshotRefs: []SnapshotRef{{
			SnapshotID: "src_1",
			ArtifactID: "art_1",
			Locator:    []byte(`{"locator_type":"text_quote","exact":"same concern"}`),
		}},
		Confidence:     Confidence{Level: "low", Rationale: "Useful signal, not a strict fact."},
		Producer:       Producer{Type: "autopilot", ID: "ses_auto"},
		CreatedEventID: "evt_evidence_reaction",
	})
	if err != nil {
		t.Fatalf("CreateEvidenceRecord reaction returned error: %v", err)
	}
	if record.EvidenceType != "reaction" || record.Confidence.Level != "low" {
		t.Fatalf("unexpected reaction evidence record: %#v", record)
	}
}

func TestCreateClaimRecordKeepsConfidenceSeparateFromApproval(t *testing.T) {
	store := newResearchFakeStore()
	svc := NewService(store)
	evidence := createSnapshotEvidence(t, svc)

	_, err := svc.CreateClaimRecord(context.Background(), CreateClaimRecordRequest{
		ClaimID:               "clm_approved_without_event",
		MissionID:             "mis_1",
		State:                 "approved",
		Text:                  "This should not be accepted without an approval event.",
		ClaimType:             "decision",
		SupportingEvidenceIDs: []string{evidence.EvidenceID},
		Confidence:            Confidence{Level: "high"},
		Approval:              Approval{State: "approved"},
		CreatedEventID:        "evt_claim_without_event",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}

	claim, err := svc.CreateClaimRecord(context.Background(), CreateClaimRecordRequest{
		ClaimID:               "clm_high_confidence",
		MissionID:             "mis_1",
		Text:                  "High confidence is still only proposed.",
		ClaimType:             "decision",
		SupportingEvidenceIDs: []string{evidence.EvidenceID},
		Confidence:            Confidence{Level: "high", Rationale: "Pinned snapshot."},
		CreatedEventID:        "evt_claim_high",
	})
	if err != nil {
		t.Fatalf("CreateClaimRecord high confidence returned error: %v", err)
	}
	if claim.State != "proposed" || claim.Approval.State != "pending" || !claim.Approval.Required {
		t.Fatalf("confidence changed acceptance state: %#v", claim)
	}

	_, err = svc.CreateClaimRecord(context.Background(), CreateClaimRecordRequest{
		ClaimID:               "clm_unrelated_approval",
		MissionID:             "mis_1",
		State:                 "approved",
		Text:                  "Unrelated approval events must not accept this claim.",
		ClaimType:             "decision",
		SupportingEvidenceIDs: []string{evidence.EvidenceID},
		Approval:              Approval{State: "approved", ApprovalEventID: "evt_unrelated_approval"},
		CreatedEventID:        "evt_claim_unrelated",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected unrelated approval rejection, got %v", err)
	}

	approved, err := svc.CreateClaimRecord(context.Background(), CreateClaimRecordRequest{
		ClaimID:               "clm_approved",
		MissionID:             "mis_1",
		State:                 "approved",
		Text:                  "Approval requires a ledger approval event.",
		ClaimType:             "decision",
		SupportingEvidenceIDs: []string{evidence.EvidenceID},
		Approval:              Approval{State: "approved", ApprovalEventID: "evt_approval"},
		CreatedEventID:        "evt_claim_approved",
	})
	if err != nil {
		t.Fatalf("CreateClaimRecord approved returned error: %v", err)
	}
	if approved.State != "approved" || approved.Approval.ApprovalEventID != "evt_approval" {
		t.Fatalf("unexpected approved claim: %#v", approved)
	}
}

func TestUpdateClaimConfidenceRecordsAdvisoryEventOnly(t *testing.T) {
	store := newResearchFakeStore()
	svc := NewService(store)
	evidence := createSnapshotEvidence(t, svc)
	store.events = append(store.events, LedgerEvent{
		EventID:   "evt_claim_confidence_update",
		MissionID: "mis_1",
		EventType: "claim.proposed",
		Producer:  Producer{Type: "autopilot", ID: "ses_auto"},
		Payload:   []byte(`{"claim_id":"clm_confidence_update","proposal_id":"prp_confidence_update"}`),
		CreatedAt: time.Now().UTC(),
	})
	claim, err := svc.CreateClaimRecord(context.Background(), CreateClaimRecordRequest{
		ClaimID:               "clm_confidence_update",
		MissionID:             "mis_1",
		Text:                  "Confidence can change without approval.",
		ClaimType:             "descriptive",
		SupportingEvidenceIDs: []string{evidence.EvidenceID},
		Confidence:            Confidence{Level: "low", Rationale: "Initial weak support."},
		CreatedEventID:        "evt_claim_confidence_update",
	})
	if err != nil {
		t.Fatalf("CreateClaimRecord returned error: %v", err)
	}

	event, err := svc.UpdateClaimConfidence(context.Background(), UpdateClaimConfidenceRequest{
		EventID:          "evt_confidence_update",
		MissionID:        "mis_1",
		ClaimID:          claim.ClaimID,
		Confidence:       Confidence{Level: "high", Rationale: "New pinned evidence directly supports it."},
		BasisEvidenceIDs: []string{evidence.EvidenceID},
		Producer:         Producer{Type: "agent_session", ID: "ses_auto"},
	})
	if err != nil {
		t.Fatalf("UpdateClaimConfidence returned error: %v", err)
	}
	if event.EventType != ClaimConfidenceUpdatedEvent {
		t.Fatalf("unexpected event type: %#v", event)
	}
	updates := ClaimConfidenceUpdatesFromEvents([]LedgerEvent{event})
	if len(updates) != 1 || updates[0].ClaimID != claim.ClaimID || updates[0].Confidence.Level != "high" {
		t.Fatalf("unexpected confidence update projection: %#v", updates)
	}
	storedClaim, err := svc.GetClaimRecord(context.Background(), claim.ClaimID)
	if err != nil {
		t.Fatal(err)
	}
	if storedClaim.State != "proposed" || storedClaim.Approval.State != "pending" {
		t.Fatalf("confidence update changed approval state: %#v", storedClaim)
	}

	second, err := svc.UpdateClaimConfidence(context.Background(), UpdateClaimConfidenceRequest{
		EventID:    "evt_confidence_update_second",
		MissionID:  "mis_1",
		ClaimID:    claim.ClaimID,
		Confidence: Confidence{Level: "medium", Rationale: "Same turn tried again."},
		Producer:   Producer{Type: "agent_session", ID: "ses_auto"},
	})
	if err != nil {
		t.Fatalf("UpdateClaimConfidence second update returned error: %v", err)
	}
	if second.EventID == event.EventID || second.EventType != ClaimConfidenceUpdatedEvent {
		t.Fatalf("expected a second confidence event, got %#v", second)
	}
}

func TestProposalBundleStateTransitionsAreTerminal(t *testing.T) {
	store := newResearchFakeStore()
	svc := NewService(store)
	evidence := createSnapshotEvidence(t, svc)
	claim, err := svc.CreateClaimRecord(context.Background(), CreateClaimRecordRequest{
		ClaimID:               "clm_for_proposal",
		MissionID:             "mis_1",
		Text:                  "Store evidence before claims.",
		ClaimType:             "decision",
		SupportingEvidenceIDs: []string{evidence.EvidenceID},
		CreatedEventID:        "evt_claim_for_proposal",
	})
	if err != nil {
		t.Fatalf("CreateClaimRecord returned error: %v", err)
	}

	bundle, err := svc.CreateProposalBundle(context.Background(), CreateProposalBundleRequest{
		ProposalID:        "prp_1",
		MissionID:         "mis_1",
		Title:             "Accept claim",
		ObjectRefs:        []ObjectRef{{ObjectKind: ClaimRecordObjectKind, ObjectID: claim.ClaimID}},
		RequestedDecision: "approve",
		CreatedEventID:    "evt_proposal",
	})
	if err != nil {
		t.Fatalf("CreateProposalBundle returned error: %v", err)
	}
	if bundle.State != "pending_review" {
		t.Fatalf("unexpected initial proposal state: %#v", bundle)
	}

	_, err = svc.UpdateProposalBundleState(context.Background(), UpdateProposalBundleStateRequest{
		ProposalID:      "prp_1",
		State:           "approved",
		DecisionEventID: "evt_approval",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected unrelated proposal decision rejection, got %v", err)
	}

	approved, err := svc.UpdateProposalBundleState(context.Background(), UpdateProposalBundleStateRequest{
		ProposalID:      "prp_1",
		State:           "approved",
		DecisionEventID: "evt_proposal_approval",
	})
	if err != nil {
		t.Fatalf("UpdateProposalBundleState approved returned error: %v", err)
	}
	if approved.State != "approved" || approved.DecisionEventID != "evt_proposal_approval" {
		t.Fatalf("unexpected approved proposal: %#v", approved)
	}

	_, err = svc.UpdateProposalBundleState(context.Background(), UpdateProposalBundleStateRequest{
		ProposalID:      "prp_1",
		State:           "rejected",
		DecisionEventID: "evt_reject",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected terminal transition rejection, got %v", err)
	}
}

func createSnapshotEvidence(t *testing.T, svc *Service) EvidenceRecord {
	t.Helper()
	record, err := svc.CreateEvidenceRecord(context.Background(), CreateEvidenceRecordRequest{
		EvidenceID:   "evd_1",
		MissionID:    "mis_1",
		Summary:      "Pinned source snapshot.",
		EvidenceType: "quote",
		SnapshotRefs: []SnapshotRef{{
			SnapshotID: "src_1",
			ArtifactID: "art_1",
			Locator:    []byte(`{"locator_type":"text_quote","exact":"source"}`),
		}},
		Confidence:     Confidence{Level: "medium"},
		Producer:       Producer{Type: "autopilot", ID: "ses_auto"},
		CreatedEventID: "evt_evidence",
	})
	if err != nil {
		t.Fatalf("CreateEvidenceRecord returned error: %v", err)
	}
	return record
}

type researchFakeStore struct {
	fakeStore
	events    []LedgerEvent
	snapshots map[string]SourceSnapshot
	evidence  map[string]EvidenceRecord
	claims    map[string]ClaimRecord
	questions map[string]QuestionRecord
	options   map[string]OptionRecord
	proposals map[string]ProposalBundle
}

func newResearchFakeStore() *researchFakeStore {
	now := time.Now().UTC()
	return &researchFakeStore{
		events: []LedgerEvent{
			{EventID: "evt_user", MissionID: "mis_1", EventType: "mission.steered", Producer: Producer{Type: "user", ID: "ses_user"}, CreatedAt: now},
			{EventID: "evt_evidence", MissionID: "mis_1", EventType: "evidence.proposed", Producer: Producer{Type: "autopilot", ID: "ses_auto"}, Payload: []byte(`{"evidence_id":"evd_1"}`), CreatedAt: now},
			{EventID: "evt_claim_without_event", MissionID: "mis_1", EventType: "claim.proposed", Producer: Producer{Type: "autopilot", ID: "ses_auto"}, Payload: []byte(`{"claim_id":"clm_approved_without_event","proposal_id":"prp_without_event"}`), CreatedAt: now},
			{EventID: "evt_claim_high", MissionID: "mis_1", EventType: "claim.proposed", Producer: Producer{Type: "autopilot", ID: "ses_auto"}, Payload: []byte(`{"claim_id":"clm_high_confidence","proposal_id":"prp_high"}`), CreatedAt: now},
			{EventID: "evt_claim_unrelated", MissionID: "mis_1", EventType: "claim.proposed", Producer: Producer{Type: "autopilot", ID: "ses_auto"}, Payload: []byte(`{"claim_id":"clm_unrelated_approval","proposal_id":"prp_unrelated_claim"}`), CreatedAt: now},
			{EventID: "evt_claim_approved", MissionID: "mis_1", EventType: "claim.proposed", Producer: Producer{Type: "autopilot", ID: "ses_auto"}, Payload: []byte(`{"claim_id":"clm_approved","proposal_id":"prp_approved"}`), CreatedAt: now},
			{EventID: "evt_claim_for_proposal", MissionID: "mis_1", EventType: "claim.proposed", Producer: Producer{Type: "autopilot", ID: "ses_auto"}, Payload: []byte(`{"claim_id":"clm_for_proposal","proposal_id":"prp_1"}`), CreatedAt: now},
			{EventID: "evt_proposal", MissionID: "mis_1", EventType: "proposal.submitted", Producer: Producer{Type: "autopilot", ID: "ses_auto"}, Payload: []byte(`{"proposal_id":"prp_1"}`), CreatedAt: now},
			{EventID: "evt_approval", MissionID: "mis_1", EventType: "proposal.approved", Producer: Producer{Type: "user", ID: "ses_user"}, Payload: []byte(`{"proposal_id":"prp_approved","approved_object_ids":["clm_approved"],"rejected_object_ids":[]}`), CreatedAt: now},
			{EventID: "evt_unrelated_approval", MissionID: "mis_1", EventType: "proposal.approved", Producer: Producer{Type: "user", ID: "ses_user"}, Payload: []byte(`{"proposal_id":"prp_other","approved_object_ids":["clm_other"],"rejected_object_ids":[]}`), CreatedAt: now},
			{EventID: "evt_proposal_approval", MissionID: "mis_1", EventType: "proposal.approved", Producer: Producer{Type: "user", ID: "ses_user"}, Payload: []byte(`{"proposal_id":"prp_1","approved_object_ids":["clm_for_proposal"],"rejected_object_ids":[]}`), CreatedAt: now},
			{EventID: "evt_reject", MissionID: "mis_1", EventType: "proposal.rejected", Producer: Producer{Type: "user", ID: "ses_user"}, Payload: []byte(`{"proposal_id":"prp_1","approved_object_ids":[],"rejected_object_ids":["clm_for_proposal"]}`), CreatedAt: now},
		},
		snapshots: map[string]SourceSnapshot{
			"src_1": {SnapshotID: "src_1", MissionID: "mis_1", ArtifactIDs: []string{"art_1"}},
		},
		evidence:  map[string]EvidenceRecord{},
		claims:    map[string]ClaimRecord{},
		questions: map[string]QuestionRecord{},
		options:   map[string]OptionRecord{},
		proposals: map[string]ProposalBundle{},
	}
}

func (f *researchFakeStore) ListLedgerEvents(_ context.Context, missionID string) ([]LedgerEvent, error) {
	events := []LedgerEvent{}
	for _, event := range f.events {
		if event.MissionID == missionID {
			events = append(events, event)
		}
	}
	return events, nil
}

func (f *researchFakeStore) AppendLedgerEvent(_ context.Context, event LedgerEvent) (LedgerEvent, error) {
	event.Sequence = int64(len(f.events) + 1)
	f.events = append(f.events, event)
	return event, nil
}

func (f *researchFakeStore) GetSourceSnapshot(_ context.Context, snapshotID string) (SourceSnapshot, error) {
	snapshot, ok := f.snapshots[snapshotID]
	if !ok {
		return SourceSnapshot{}, errors.New("missing snapshot")
	}
	return snapshot, nil
}

func (f *researchFakeStore) CreateEvidenceRecord(_ context.Context, record EvidenceRecord) error {
	f.evidence[record.EvidenceID] = record
	return nil
}

func (f *researchFakeStore) GetEvidenceRecord(_ context.Context, evidenceID string) (EvidenceRecord, error) {
	record, ok := f.evidence[evidenceID]
	if !ok {
		return EvidenceRecord{}, errors.New("missing evidence")
	}
	return record, nil
}

func (f *researchFakeStore) CreateClaimRecord(_ context.Context, record ClaimRecord) error {
	f.claims[record.ClaimID] = record
	return nil
}

func (f *researchFakeStore) GetClaimRecord(_ context.Context, claimID string) (ClaimRecord, error) {
	record, ok := f.claims[claimID]
	if !ok {
		return ClaimRecord{}, errors.New("missing claim")
	}
	return record, nil
}

func (f *researchFakeStore) CreateQuestionRecord(_ context.Context, record QuestionRecord) error {
	f.questions[record.QuestionID] = record
	return nil
}

func (f *researchFakeStore) GetQuestionRecord(_ context.Context, questionID string) (QuestionRecord, error) {
	record, ok := f.questions[questionID]
	if !ok {
		return QuestionRecord{}, errors.New("missing question")
	}
	return record, nil
}

func (f *researchFakeStore) CreateOptionRecord(_ context.Context, record OptionRecord) error {
	f.options[record.OptionID] = record
	return nil
}

func (f *researchFakeStore) GetOptionRecord(_ context.Context, optionID string) (OptionRecord, error) {
	record, ok := f.options[optionID]
	if !ok {
		return OptionRecord{}, errors.New("missing option")
	}
	return record, nil
}

func (f *researchFakeStore) CreateProposalBundle(_ context.Context, bundle ProposalBundle) error {
	f.proposals[bundle.ProposalID] = bundle
	return nil
}

func (f *researchFakeStore) GetProposalBundle(_ context.Context, proposalID string) (ProposalBundle, error) {
	bundle, ok := f.proposals[proposalID]
	if !ok {
		return ProposalBundle{}, errors.New("missing proposal")
	}
	return bundle, nil
}

func (f *researchFakeStore) UpdateProposalBundleState(_ context.Context, update ProposalBundleStateUpdate) error {
	bundle, ok := f.proposals[update.ProposalID]
	if !ok || bundle.State != update.FromState {
		return errors.New("missing proposal")
	}
	bundle.State = update.ToState
	bundle.DecisionEventID = update.DecisionEventID
	bundle.DecidedAt = update.DecidedAt
	bundle.UpdatedAt = update.UpdatedAt
	f.proposals[update.ProposalID] = bundle
	return nil
}
