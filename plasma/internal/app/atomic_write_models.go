package app

import "context"

// AtomicWriteStore는 서로 연관된 event/object 생성을 한 transaction으로 commit하는
// 저장소 port다.
type AtomicWriteStore interface {
	CommitAtomicWrite(context.Context, AtomicWrite) (AtomicWriteResult, error)
}

// AtomicWrite는 하나의 제품 전이가 함께 기록해야 하는 저장 객체 묶음이다.
//
// 여러 slice 중 일부가 비어 있을 수 있지만, commit은 all-or-nothing이어야 한다.
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

// AtomicWriteResult는 commit 후 저장소가 확정한 event 목록을 반환한다.
type AtomicWriteResult struct {
	Events []LedgerEvent
}

// SnapshotLiquid2SourceWithEventRequest는 Liquid2 source snapshot과 source event를
// 함께 생성하기 위한 입력이다.
type SnapshotLiquid2SourceWithEventRequest struct {
	Snapshot SnapshotLiquid2SourceRequest
	EventID  string
	Producer Producer
}

// Liquid2SnapshotWithEventResult는 Liquid2 snapshot 생성 결과와 장부 event다.
type Liquid2SnapshotWithEventResult struct {
	Artifact RawArtifact
	Snapshot SourceSnapshot
	Event    LedgerEvent
}

// CreateSourceSnapshotWithEventRequest는 새 raw artifact와 source snapshot, event를
// 같은 transaction으로 생성하는 입력이다.
type CreateSourceSnapshotWithEventRequest struct {
	Artifact CreateRawArtifactRequest
	Snapshot CreateSourceSnapshotRequest
	Event    AppendEventRequest
}

// SourceSnapshotWithEventResult는 artifact-backed source snapshot 생성 결과다.
type SourceSnapshotWithEventResult struct {
	Artifact RawArtifact
	Snapshot SourceSnapshot
	Event    LedgerEvent
}

// CreateExistingArtifactSourceSnapshotWithEventRequest는 이미 저장된 artifact를 source
// snapshot으로 연결하는 입력이다.
type CreateExistingArtifactSourceSnapshotWithEventRequest struct {
	Snapshot CreateSourceSnapshotRequest
	Event    AppendEventRequest
}

// ExistingArtifactSourceSnapshotWithEventResult는 existing artifact 기반 snapshot 생성 결과다.
type ExistingArtifactSourceSnapshotWithEventResult struct {
	Snapshot SourceSnapshot
	Event    LedgerEvent
}

// CreateLiveSourceSnapshotWithEventRequest는 live reference source snapshot과 event를
// 함께 생성하는 입력이다.
type CreateLiveSourceSnapshotWithEventRequest struct {
	Snapshot CreateSourceSnapshotRequest
	Event    AppendEventRequest
}

// LiveSourceSnapshotWithEventResult는 live reference source snapshot 생성 결과다.
type LiveSourceSnapshotWithEventResult struct {
	Snapshot SourceSnapshot
	Event    LedgerEvent
}

// CreateEvidenceProposalRequest는 evidence record와 proposal event/bundle을 함께 만드는
// 요청이다.
type CreateEvidenceProposalRequest struct {
	EvidenceEvent AppendEventRequest
	Evidence      CreateEvidenceRecordRequest
	ProposalEvent AppendEventRequest
	Proposal      CreateProposalBundleRequest
}

// EvidenceProposalResult는 evidence proposal 생성 결과다.
type EvidenceProposalResult struct {
	Evidence      EvidenceRecord
	Proposal      ProposalBundle
	EvidenceEvent LedgerEvent
	ProposalEvent LedgerEvent
}

// CreateQuestionProposalRequest는 question record와 proposal event/bundle을 함께 만드는 요청이다.
type CreateQuestionProposalRequest struct {
	QuestionEvent AppendEventRequest
	Question      CreateQuestionRecordRequest
	ProposalEvent AppendEventRequest
	Proposal      CreateProposalBundleRequest
}

// QuestionProposalResult는 question proposal 생성 결과다.
type QuestionProposalResult struct {
	Question      QuestionRecord
	Proposal      ProposalBundle
	QuestionEvent LedgerEvent
	ProposalEvent LedgerEvent
}

// CreateClaimProposalRequest는 claim record와 proposal event/bundle을 함께 만드는 요청이다.
type CreateClaimProposalRequest struct {
	ClaimEvent    AppendEventRequest
	Claim         CreateClaimRecordRequest
	ProposalEvent AppendEventRequest
	Proposal      CreateProposalBundleRequest
}

// ClaimProposalResult는 claim proposal 생성 결과다.
type ClaimProposalResult struct {
	Claim         ClaimRecord
	Proposal      ProposalBundle
	ClaimEvent    LedgerEvent
	ProposalEvent LedgerEvent
}

// SubmitProposalRequest는 object 없이 proposal bundle만 제출하는 요청이다.
type SubmitProposalRequest struct {
	ProposalEvent AppendEventRequest
	Proposal      CreateProposalBundleRequest
}

// SubmitProposalResult는 proposal bundle 제출 결과다.
type SubmitProposalResult struct {
	Proposal      ProposalBundle
	ProposalEvent LedgerEvent
}
