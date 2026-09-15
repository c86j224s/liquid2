package app

import "github.com/c86j224s/liquid2/plasma/internal/reporting/reportdocument"

import (
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/researchproposal"
	"github.com/c86j224s/liquid2/plasma/internal/researchrecords"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

// AtomicWriteStore는 서로 연관된 event/object 생성을 한 transaction으로 commit하는
// 저장소 port다.
type AtomicWriteStore interface {
	CommitAtomicWrite(context.Context, AtomicWrite) (AtomicWriteResult, error)
}

// AtomicWrite는 하나의 제품 전이가 함께 기록해야 하는 저장 객체 묶음이다.
//
// 여러 slice 중 일부가 비어 있을 수 있지만, commit은 all-or-nothing이어야 한다.
type AtomicWrite struct {
	Events          []ledger.Event
	RawArtifacts    []artifactcontract.Raw
	SourceSnapshots []sourcecontract.Snapshot
	EvidenceRecords []researchrecords.EvidenceRecord
	ClaimRecords    []researchrecords.ClaimRecord
	QuestionRecords []researchrecords.QuestionRecord
	ProposalBundles []researchproposal.ProposalBundle
	Reports         []reportdocument.Report
	ReportVersions  []reportdocument.ReportVersion
	ReportBlocks    []reportdocument.ReportBlock
}

// AtomicWriteResult는 commit 후 저장소가 확정한 event 목록을 반환한다.
type AtomicWriteResult struct {
	Events []ledger.Event
}

// SubmitProposalRequest는 object 없이 proposal bundle만 제출하는 요청이다.
type SubmitProposalRequest struct {
	ProposalEvent ledger.AppendRequest
	Proposal      researchproposal.CreateProposalBundleRequest
}

// SubmitProposalResult는 proposal bundle 제출 결과다.
type SubmitProposalResult struct {
	Proposal      researchproposal.ProposalBundle
	ProposalEvent ledger.Event
}
