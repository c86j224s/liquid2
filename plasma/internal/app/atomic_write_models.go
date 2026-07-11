package app

import "context"

type AtomicWriteStore interface {
	CommitAtomicWrite(context.Context, AtomicWrite) (AtomicWriteResult, error)
}

type AtomicWrite struct {
	Events          []LedgerEvent
	RawArtifacts    []RawArtifact
	SourceSnapshots []SourceSnapshot
	EvidenceRecords []EvidenceRecord
	ClaimRecords    []ClaimRecord
	QuestionRecords []QuestionRecord
	ProposalBundles []ProposalBundle
	Reports         []Report
	ReportVersions  []ReportVersion
	ReportBlocks    []ReportBlock
}

type AtomicWriteResult struct {
	Events []LedgerEvent
}

type SnapshotLiquid2SourceWithEventRequest struct {
	Snapshot SnapshotLiquid2SourceRequest
	EventID  string
	Producer Producer
}

type Liquid2SnapshotWithEventResult struct {
	Artifact RawArtifact
	Snapshot SourceSnapshot
	Event    LedgerEvent
}

type CreateSourceSnapshotWithEventRequest struct {
	Artifact CreateRawArtifactRequest
	Snapshot CreateSourceSnapshotRequest
	Event    AppendEventRequest
}

type SourceSnapshotWithEventResult struct {
	Artifact RawArtifact
	Snapshot SourceSnapshot
	Event    LedgerEvent
}

type CreateExistingArtifactSourceSnapshotWithEventRequest struct {
	Snapshot CreateSourceSnapshotRequest
	Event    AppendEventRequest
}

type ExistingArtifactSourceSnapshotWithEventResult struct {
	Snapshot SourceSnapshot
	Event    LedgerEvent
}

type CreateLiveSourceSnapshotWithEventRequest struct {
	Snapshot CreateSourceSnapshotRequest
	Event    AppendEventRequest
}

type LiveSourceSnapshotWithEventResult struct {
	Snapshot SourceSnapshot
	Event    LedgerEvent
}

type CreateEvidenceProposalRequest struct {
	EvidenceEvent AppendEventRequest
	Evidence      CreateEvidenceRecordRequest
	ProposalEvent AppendEventRequest
	Proposal      CreateProposalBundleRequest
}

type EvidenceProposalResult struct {
	Evidence      EvidenceRecord
	Proposal      ProposalBundle
	EvidenceEvent LedgerEvent
	ProposalEvent LedgerEvent
}

type CreateQuestionProposalRequest struct {
	QuestionEvent AppendEventRequest
	Question      CreateQuestionRecordRequest
	ProposalEvent AppendEventRequest
	Proposal      CreateProposalBundleRequest
}

type QuestionProposalResult struct {
	Question      QuestionRecord
	Proposal      ProposalBundle
	QuestionEvent LedgerEvent
	ProposalEvent LedgerEvent
}

type CreateClaimProposalRequest struct {
	ClaimEvent    AppendEventRequest
	Claim         CreateClaimRecordRequest
	ProposalEvent AppendEventRequest
	Proposal      CreateProposalBundleRequest
}

type ClaimProposalResult struct {
	Claim         ClaimRecord
	Proposal      ProposalBundle
	ClaimEvent    LedgerEvent
	ProposalEvent LedgerEvent
}

type SubmitProposalRequest struct {
	ProposalEvent AppendEventRequest
	Proposal      CreateProposalBundleRequest
}

type SubmitProposalResult struct {
	Proposal      ProposalBundle
	ProposalEvent LedgerEvent
}
