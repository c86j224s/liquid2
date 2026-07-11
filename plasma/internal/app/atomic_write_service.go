package app

import (
	"context"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/sourceevents"
)

func (s *Service) CreateRawArtifactWithEvent(
	ctx context.Context,
	artifactReq CreateRawArtifactRequest,
	eventReqForArtifact func(RawArtifact) AppendEventRequest,
) (RawArtifact, LedgerEvent, error) {
	if eventReqForArtifact == nil {
		return RawArtifact{}, LedgerEvent{}, fmt.Errorf("%w: event builder is required", ErrInvalidInput)
	}
	artifact, err := buildRawArtifact(artifactReq)
	if err != nil {
		return RawArtifact{}, LedgerEvent{}, err
	}
	event, err := buildLedgerEvent(eventReqForArtifact(artifact))
	if err != nil {
		return RawArtifact{}, LedgerEvent{}, err
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:       []LedgerEvent{event},
		RawArtifacts: []RawArtifact{artifact},
	})
	if err != nil {
		return RawArtifact{}, LedgerEvent{}, err
	}
	return artifact, committed.Events[0], nil
}

func (s *Service) CreateSourceSnapshotWithEvent(
	ctx context.Context,
	req CreateSourceSnapshotWithEventRequest,
) (SourceSnapshotWithEventResult, error) {
	artifact, err := buildRawArtifact(req.Artifact)
	if err != nil {
		return SourceSnapshotWithEventResult{}, err
	}
	snapshotReq := req.Snapshot
	if len(snapshotReq.ArtifactIDs) == 0 {
		snapshotReq.ArtifactIDs = []string{artifact.ArtifactID}
	}
	snapshot, err := s.buildSourceSnapshot(ctx, snapshotReq, []RawArtifact{artifact})
	if err != nil {
		return SourceSnapshotWithEventResult{}, err
	}
	eventReq := req.Event
	if len(eventReq.Payload) == 0 {
		eventReq.Payload = sourceevents.BuildSourceSnapshottedPayload(sourceevents.SourceSnapshottedPayloadRequest{
			SnapshotID:         snapshot.SnapshotID,
			ArtifactIDs:        snapshot.ArtifactIDs,
			Connector:          sourceEventConnectorRef(snapshot.Connector),
			IncludeArtifactIDs: true,
		})
	}
	event, err := buildLedgerEvent(eventReq)
	if err != nil {
		return SourceSnapshotWithEventResult{}, err
	}
	if event.EventType != sourceevents.SourceSnapshottedEventType {
		return SourceSnapshotWithEventResult{}, fmt.Errorf("%w: source snapshot requires source.snapshotted event", ErrInvalidInput)
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:          []LedgerEvent{event},
		RawArtifacts:    []RawArtifact{artifact},
		SourceSnapshots: []SourceSnapshot{snapshot},
	})
	if err != nil {
		return SourceSnapshotWithEventResult{}, err
	}
	return SourceSnapshotWithEventResult{
		Artifact: artifact,
		Snapshot: snapshot,
		Event:    committed.Events[0],
	}, nil
}

func (s *Service) CreateExistingArtifactSourceSnapshotWithEvent(
	ctx context.Context,
	req CreateExistingArtifactSourceSnapshotWithEventRequest,
) (ExistingArtifactSourceSnapshotWithEventResult, error) {
	snapshot, err := s.buildSourceSnapshot(ctx, req.Snapshot, nil)
	if err != nil {
		return ExistingArtifactSourceSnapshotWithEventResult{}, err
	}
	eventReq := req.Event
	if len(eventReq.Payload) == 0 {
		eventReq.Payload = sourceevents.BuildSourceSnapshottedPayload(sourceevents.SourceSnapshottedPayloadRequest{
			SnapshotID:         snapshot.SnapshotID,
			ArtifactIDs:        snapshot.ArtifactIDs,
			Connector:          sourceEventConnectorRef(snapshot.Connector),
			IncludeArtifactIDs: true,
		})
	}
	event, err := buildLedgerEvent(eventReq)
	if err != nil {
		return ExistingArtifactSourceSnapshotWithEventResult{}, err
	}
	if event.EventType != sourceevents.SourceSnapshottedEventType {
		return ExistingArtifactSourceSnapshotWithEventResult{}, fmt.Errorf("%w: source snapshot requires source.snapshotted event", ErrInvalidInput)
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:          []LedgerEvent{event},
		SourceSnapshots: []SourceSnapshot{snapshot},
	})
	if err != nil {
		return ExistingArtifactSourceSnapshotWithEventResult{}, err
	}
	return ExistingArtifactSourceSnapshotWithEventResult{
		Snapshot: snapshot,
		Event:    committed.Events[0],
	}, nil
}

func (s *Service) CreateLiveSourceSnapshotWithEvent(
	ctx context.Context,
	req CreateLiveSourceSnapshotWithEventRequest,
) (LiveSourceSnapshotWithEventResult, error) {
	snapshot, err := s.buildSourceSnapshot(ctx, req.Snapshot, nil)
	if err != nil {
		return LiveSourceSnapshotWithEventResult{}, err
	}
	eventReq := req.Event
	if len(eventReq.Payload) == 0 {
		eventReq.Payload = sourceevents.BuildSourceSnapshottedPayload(sourceevents.SourceSnapshottedPayloadRequest{
			SnapshotID: snapshot.SnapshotID,
			Connector:  sourceEventConnectorRef(snapshot.Connector),
		})
	}
	event, err := buildLedgerEvent(eventReq)
	if err != nil {
		return LiveSourceSnapshotWithEventResult{}, err
	}
	if event.EventType != sourceevents.SourceSnapshottedEventType {
		return LiveSourceSnapshotWithEventResult{}, fmt.Errorf("%w: source snapshot requires source.snapshotted event", ErrInvalidInput)
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:          []LedgerEvent{event},
		SourceSnapshots: []SourceSnapshot{snapshot},
	})
	if err != nil {
		return LiveSourceSnapshotWithEventResult{}, err
	}
	return LiveSourceSnapshotWithEventResult{
		Snapshot: snapshot,
		Event:    committed.Events[0],
	}, nil
}

func (s *Service) CreateEvidenceProposal(
	ctx context.Context,
	req CreateEvidenceProposalRequest,
) (EvidenceProposalResult, error) {
	evidenceEvent, err := buildLedgerEvent(req.EvidenceEvent)
	if err != nil {
		return EvidenceProposalResult{}, err
	}
	if evidenceEvent.EventType != "evidence.proposed" {
		return EvidenceProposalResult{}, fmt.Errorf("%w: evidence proposal requires evidence.proposed event", ErrInvalidInput)
	}
	proposalEvent, err := buildLedgerEvent(req.ProposalEvent)
	if err != nil {
		return EvidenceProposalResult{}, err
	}
	evidence, err := s.buildEvidenceRecord(ctx, req.Evidence, evidenceEvent)
	if err != nil {
		return EvidenceProposalResult{}, err
	}
	proposal, err := s.buildProposalBundle(
		ctx,
		req.Proposal,
		proposalEvent,
		[]ObjectRef{{ObjectKind: EvidenceRecordObjectKind, ObjectID: evidence.EvidenceID}},
	)
	if err != nil {
		return EvidenceProposalResult{}, err
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:          []LedgerEvent{evidenceEvent, proposalEvent},
		EvidenceRecords: []EvidenceRecord{evidence},
		ProposalBundles: []ProposalBundle{proposal},
	})
	if err != nil {
		return EvidenceProposalResult{}, err
	}
	return EvidenceProposalResult{
		Evidence:      evidence,
		Proposal:      proposal,
		EvidenceEvent: committed.Events[0],
		ProposalEvent: committed.Events[1],
	}, nil
}

func (s *Service) CreateQuestionProposal(
	ctx context.Context,
	req CreateQuestionProposalRequest,
) (QuestionProposalResult, error) {
	questionEvent, err := buildLedgerEvent(req.QuestionEvent)
	if err != nil {
		return QuestionProposalResult{}, err
	}
	if questionEvent.EventType != "question.proposed" {
		return QuestionProposalResult{}, fmt.Errorf("%w: question proposal requires question.proposed event", ErrInvalidInput)
	}
	proposalEvent, err := buildLedgerEvent(req.ProposalEvent)
	if err != nil {
		return QuestionProposalResult{}, err
	}
	question, err := s.buildQuestionRecord(ctx, req.Question, questionEvent)
	if err != nil {
		return QuestionProposalResult{}, err
	}
	proposal, err := s.buildProposalBundle(
		ctx,
		req.Proposal,
		proposalEvent,
		[]ObjectRef{{ObjectKind: QuestionRecordObjectKind, ObjectID: question.QuestionID}},
	)
	if err != nil {
		return QuestionProposalResult{}, err
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:          []LedgerEvent{questionEvent, proposalEvent},
		QuestionRecords: []QuestionRecord{question},
		ProposalBundles: []ProposalBundle{proposal},
	})
	if err != nil {
		return QuestionProposalResult{}, err
	}
	return QuestionProposalResult{
		Question:      question,
		Proposal:      proposal,
		QuestionEvent: committed.Events[0],
		ProposalEvent: committed.Events[1],
	}, nil
}

func (s *Service) CreateClaimProposal(
	ctx context.Context,
	req CreateClaimProposalRequest,
) (ClaimProposalResult, error) {
	claimEvent, err := buildLedgerEvent(req.ClaimEvent)
	if err != nil {
		return ClaimProposalResult{}, err
	}
	if claimEvent.EventType != "claim.proposed" {
		return ClaimProposalResult{}, fmt.Errorf("%w: claim proposal requires claim.proposed event", ErrInvalidInput)
	}
	proposalEvent, err := buildLedgerEvent(req.ProposalEvent)
	if err != nil {
		return ClaimProposalResult{}, err
	}
	claim, err := s.buildClaimRecord(ctx, req.Claim, claimEvent)
	if err != nil {
		return ClaimProposalResult{}, err
	}
	proposal, err := s.buildProposalBundle(
		ctx,
		req.Proposal,
		proposalEvent,
		[]ObjectRef{{ObjectKind: ClaimRecordObjectKind, ObjectID: claim.ClaimID}},
	)
	if err != nil {
		return ClaimProposalResult{}, err
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:          []LedgerEvent{claimEvent, proposalEvent},
		ClaimRecords:    []ClaimRecord{claim},
		ProposalBundles: []ProposalBundle{proposal},
	})
	if err != nil {
		return ClaimProposalResult{}, err
	}
	return ClaimProposalResult{
		Claim:         claim,
		Proposal:      proposal,
		ClaimEvent:    committed.Events[0],
		ProposalEvent: committed.Events[1],
	}, nil
}

func (s *Service) SubmitProposal(
	ctx context.Context,
	req SubmitProposalRequest,
) (SubmitProposalResult, error) {
	proposalEvent, err := buildLedgerEvent(req.ProposalEvent)
	if err != nil {
		return SubmitProposalResult{}, err
	}
	if proposalEvent.EventType != "proposal.submitted" {
		return SubmitProposalResult{}, fmt.Errorf("%w: proposal submit requires proposal.submitted event", ErrInvalidInput)
	}
	proposal, err := s.buildProposalBundle(ctx, req.Proposal, proposalEvent, nil)
	if err != nil {
		return SubmitProposalResult{}, err
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:          []LedgerEvent{proposalEvent},
		ProposalBundles: []ProposalBundle{proposal},
	})
	if err != nil {
		return SubmitProposalResult{}, err
	}
	return SubmitProposalResult{
		Proposal:      proposal,
		ProposalEvent: committed.Events[0],
	}, nil
}

func (s *Service) commitAtomicWrite(ctx context.Context, write AtomicWrite) (AtomicWriteResult, error) {
	store, ok := s.store.(AtomicWriteStore)
	if !ok {
		return AtomicWriteResult{}, fmt.Errorf("%w: atomic write store is required", ErrInvalidInput)
	}
	return store.CommitAtomicWrite(ctx, write)
}
