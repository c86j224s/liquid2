package app

import (
	"context"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/researchcatalog"
	"github.com/c86j224s/liquid2/plasma/internal/researchproposal"
	"github.com/c86j224s/liquid2/plasma/internal/researchrecords"
	"github.com/c86j224s/liquid2/plasma/internal/source"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"github.com/c86j224s/liquid2/plasma/internal/sourceevents"
)

// CreateRawArtifactWithEvent는 raw artifact와 causation 이벤트를 함께 기록한다.
func (s *Service) CreateRawArtifactWithEvent(
	ctx context.Context,
	artifactReq artifactcontract.CreateRequest,
	eventReqForArtifact func(artifactcontract.Raw) ledger.AppendRequest,
) (artifactcontract.Raw, ledger.Event, error) {
	if eventReqForArtifact == nil {
		return artifactcontract.Raw{}, ledger.Event{}, fmt.Errorf("%w: event builder is required", ErrInvalidInput)
	}
	artifact, err := artifactcontract.Build(artifactReq)
	if err != nil {
		return artifactcontract.Raw{}, ledger.Event{}, err
	}
	event, err := buildLedgerEvent(eventReqForArtifact(artifact))
	if err != nil {
		return artifactcontract.Raw{}, ledger.Event{}, err
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:       []ledger.Event{event},
		RawArtifacts: []artifactcontract.Raw{artifact},
	})
	if err != nil {
		return artifactcontract.Raw{}, ledger.Event{}, err
	}
	return artifact, committed.Events[0], nil
}

// CreateSourceSnapshotWithEvent는 source snapshot과 causation 이벤트를 함께 기록한다.
func (s *Service) CreateSourceSnapshotWithEvent(
	ctx context.Context,
	req source.CreateSourceSnapshotWithEventRequest,
) (source.SourceSnapshotWithEventResult, error) {
	artifact, err := artifactcontract.Build(req.Artifact)
	if err != nil {
		return source.SourceSnapshotWithEventResult{}, err
	}
	snapshotReq := req.Snapshot
	if len(snapshotReq.ArtifactIDs) == 0 {
		snapshotReq.ArtifactIDs = []string{artifact.ArtifactID}
	}
	snapshot, err := source.BuildSnapshot(ctx, s.store, snapshotReq, []artifactcontract.Raw{artifact})
	if err != nil {
		return source.SourceSnapshotWithEventResult{}, err
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
		return source.SourceSnapshotWithEventResult{}, err
	}
	if event.EventType != sourceevents.SourceSnapshottedEventType {
		return source.SourceSnapshotWithEventResult{}, fmt.Errorf("%w: source snapshot requires source.snapshotted event", ErrInvalidInput)
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:          []ledger.Event{event},
		RawArtifacts:    []artifactcontract.Raw{artifact},
		SourceSnapshots: []sourcecontract.Snapshot{snapshot},
	})
	if err != nil {
		return source.SourceSnapshotWithEventResult{}, err
	}
	return source.SourceSnapshotWithEventResult{
		Artifact: artifact,
		Snapshot: snapshot,
		Event:    committed.Events[0],
	}, nil
}

// CreateExistingArtifactSourceSnapshotWithEvent는 기존 artifact를 새 source snapshot으로 연결한다.
func (s *Service) CreateExistingArtifactSourceSnapshotWithEvent(
	ctx context.Context,
	req source.CreateExistingArtifactSourceSnapshotWithEventRequest,
) (source.ExistingArtifactSourceSnapshotWithEventResult, error) {
	snapshot, err := source.BuildSnapshot(ctx, s.store, req.Snapshot, nil)
	if err != nil {
		return source.ExistingArtifactSourceSnapshotWithEventResult{}, err
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
		return source.ExistingArtifactSourceSnapshotWithEventResult{}, err
	}
	if event.EventType != sourceevents.SourceSnapshottedEventType {
		return source.ExistingArtifactSourceSnapshotWithEventResult{}, fmt.Errorf("%w: source snapshot requires source.snapshotted event", ErrInvalidInput)
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:          []ledger.Event{event},
		SourceSnapshots: []sourcecontract.Snapshot{snapshot},
	})
	if err != nil {
		return source.ExistingArtifactSourceSnapshotWithEventResult{}, err
	}
	return source.ExistingArtifactSourceSnapshotWithEventResult{
		Snapshot: snapshot,
		Event:    committed.Events[0],
	}, nil
}

// CreateLiveSourceSnapshotWithEvent는 live reference source snapshot과 이벤트를 함께 기록한다.
func (s *Service) CreateLiveSourceSnapshotWithEvent(
	ctx context.Context,
	req source.CreateLiveSourceSnapshotWithEventRequest,
) (source.LiveSourceSnapshotWithEventResult, error) {
	snapshot, err := source.BuildSnapshot(ctx, s.store, req.Snapshot, nil)
	if err != nil {
		return source.LiveSourceSnapshotWithEventResult{}, err
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
		return source.LiveSourceSnapshotWithEventResult{}, err
	}
	if event.EventType != sourceevents.SourceSnapshottedEventType {
		return source.LiveSourceSnapshotWithEventResult{}, fmt.Errorf("%w: source snapshot requires source.snapshotted event", ErrInvalidInput)
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:          []ledger.Event{event},
		SourceSnapshots: []sourcecontract.Snapshot{snapshot},
	})
	if err != nil {
		return source.LiveSourceSnapshotWithEventResult{}, err
	}
	return source.LiveSourceSnapshotWithEventResult{
		Snapshot: snapshot,
		Event:    committed.Events[0],
	}, nil
}

// CreateEvidenceProposal는 evidence proposal record와 이벤트를 함께 기록한다.
func (s *Service) CreateEvidenceProposal(
	ctx context.Context,
	req researchproposal.CreateEvidenceProposalRequest,
) (researchproposal.EvidenceProposalResult, error) {
	evidenceEvent, err := buildLedgerEvent(req.EvidenceEvent)
	if err != nil {
		return researchproposal.EvidenceProposalResult{}, err
	}
	if evidenceEvent.EventType != "evidence.proposed" {
		return researchproposal.EvidenceProposalResult{}, fmt.Errorf("%w: evidence proposal requires evidence.proposed event", ErrInvalidInput)
	}
	proposalEvent, err := buildLedgerEvent(req.ProposalEvent)
	if err != nil {
		return researchproposal.EvidenceProposalResult{}, err
	}
	evidence, err := researchrecords.BuildEvidenceRecord(ctx, s.store, researchrecords.CreateEvidenceRecordRequest(req.Evidence), evidenceEvent)
	if err != nil {
		return researchproposal.EvidenceProposalResult{}, err
	}
	proposal, err := researchproposal.BuildProposalBundle(ctx, s.requireObjectRef,
		req.Proposal,
		proposalEvent,
		[]researchcatalog.ObjectRef{{ObjectKind: researchrecords.EvidenceRecordObjectKind, ObjectID: evidence.EvidenceID}},
	)
	if err != nil {
		return researchproposal.EvidenceProposalResult{}, err
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:          []ledger.Event{evidenceEvent, proposalEvent},
		EvidenceRecords: []researchrecords.EvidenceRecord{evidence},
		ProposalBundles: []researchproposal.ProposalBundle{proposal},
	})
	if err != nil {
		return researchproposal.EvidenceProposalResult{}, err
	}
	return researchproposal.EvidenceProposalResult{
		Evidence:      evidence,
		Proposal:      proposal,
		EvidenceEvent: committed.Events[0],
		ProposalEvent: committed.Events[1],
	}, nil
}

// CreateQuestionProposal는 question proposal record와 이벤트를 함께 기록한다.
func (s *Service) CreateQuestionProposal(
	ctx context.Context,
	req researchproposal.CreateQuestionProposalRequest,
) (researchproposal.QuestionProposalResult, error) {
	questionEvent, err := buildLedgerEvent(req.QuestionEvent)
	if err != nil {
		return researchproposal.QuestionProposalResult{}, err
	}
	if questionEvent.EventType != "question.proposed" {
		return researchproposal.QuestionProposalResult{}, fmt.Errorf("%w: question proposal requires question.proposed event", ErrInvalidInput)
	}
	proposalEvent, err := buildLedgerEvent(req.ProposalEvent)
	if err != nil {
		return researchproposal.QuestionProposalResult{}, err
	}
	question, err := researchrecords.BuildQuestionRecord(ctx, researchrecords.QuestionRequirements{
		RequireEvidenceRecords: s.requireEvidenceRecords,
		RequireClaimRecords:    s.requireClaimRecords,
	}, req.Question, questionEvent)
	if err != nil {
		return researchproposal.QuestionProposalResult{}, err
	}
	proposal, err := researchproposal.BuildProposalBundle(ctx, s.requireObjectRef,
		req.Proposal,
		proposalEvent,
		[]researchcatalog.ObjectRef{{ObjectKind: researchrecords.QuestionRecordObjectKind, ObjectID: question.QuestionID}},
	)
	if err != nil {
		return researchproposal.QuestionProposalResult{}, err
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:          []ledger.Event{questionEvent, proposalEvent},
		QuestionRecords: []researchrecords.QuestionRecord{question},
		ProposalBundles: []researchproposal.ProposalBundle{proposal},
	})
	if err != nil {
		return researchproposal.QuestionProposalResult{}, err
	}
	return researchproposal.QuestionProposalResult{
		Question:      question,
		Proposal:      proposal,
		QuestionEvent: committed.Events[0],
		ProposalEvent: committed.Events[1],
	}, nil
}

// CreateClaimProposal는 claim proposal record와 이벤트를 함께 기록한다.
func (s *Service) CreateClaimProposal(
	ctx context.Context,
	req researchproposal.CreateClaimProposalRequest,
) (researchproposal.ClaimProposalResult, error) {
	claimEvent, err := buildLedgerEvent(req.ClaimEvent)
	if err != nil {
		return researchproposal.ClaimProposalResult{}, err
	}
	if claimEvent.EventType != "claim.proposed" {
		return researchproposal.ClaimProposalResult{}, fmt.Errorf("%w: claim proposal requires claim.proposed event", ErrInvalidInput)
	}
	proposalEvent, err := buildLedgerEvent(req.ProposalEvent)
	if err != nil {
		return researchproposal.ClaimProposalResult{}, err
	}
	claim, err := researchrecords.BuildClaimRecord(ctx, researchrecords.ClaimRequirements{
		RequireEvidenceRecords: s.requireEvidenceRecords,
		RequireQuestionRecords: s.requireQuestionRecords,
		RequireMissionEvent:    s.requireMissionEvent,
	}, req.Claim, claimEvent)
	if err != nil {
		return researchproposal.ClaimProposalResult{}, err
	}
	proposal, err := researchproposal.BuildProposalBundle(ctx, s.requireObjectRef,
		req.Proposal,
		proposalEvent,
		[]researchcatalog.ObjectRef{{ObjectKind: researchrecords.ClaimRecordObjectKind, ObjectID: claim.ClaimID}},
	)
	if err != nil {
		return researchproposal.ClaimProposalResult{}, err
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:          []ledger.Event{claimEvent, proposalEvent},
		ClaimRecords:    []researchrecords.ClaimRecord{claim},
		ProposalBundles: []researchproposal.ProposalBundle{proposal},
	})
	if err != nil {
		return researchproposal.ClaimProposalResult{}, err
	}
	return researchproposal.ClaimProposalResult{
		Claim:         claim,
		Proposal:      proposal,
		ClaimEvent:    committed.Events[0],
		ProposalEvent: committed.Events[1],
	}, nil
}

// SubmitProposal는 proposal 제출 결정을 record와 이벤트로 남긴다.
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
	proposal, err := researchproposal.BuildProposalBundle(ctx, s.requireObjectRef, req.Proposal, proposalEvent, nil)
	if err != nil {
		return SubmitProposalResult{}, err
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:          []ledger.Event{proposalEvent},
		ProposalBundles: []researchproposal.ProposalBundle{proposal},
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
