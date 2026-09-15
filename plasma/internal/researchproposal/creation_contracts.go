package researchproposal

import (
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/researchrecords"
)

// CreateEvidenceProposalRequest는 evidence record와 proposal event/bundle을 함께 만드는 요청이다.
type CreateEvidenceProposalRequest struct {
	EvidenceEvent ledger.AppendRequest
	Evidence      researchrecords.CreateEvidenceRecordRequest
	ProposalEvent ledger.AppendRequest
	Proposal      CreateProposalBundleRequest
}

// EvidenceProposalResult는 evidence proposal 생성 결과다.
type EvidenceProposalResult struct {
	Evidence      researchrecords.EvidenceRecord
	Proposal      ProposalBundle
	EvidenceEvent ledger.Event
	ProposalEvent ledger.Event
}

// CreateQuestionProposalRequest는 question record와 proposal event/bundle을 함께 만드는 요청이다.
type CreateQuestionProposalRequest struct {
	QuestionEvent ledger.AppendRequest
	Question      researchrecords.CreateQuestionRecordRequest
	ProposalEvent ledger.AppendRequest
	Proposal      CreateProposalBundleRequest
}

// QuestionProposalResult는 question proposal 생성 결과다.
type QuestionProposalResult struct {
	Question      researchrecords.QuestionRecord
	Proposal      ProposalBundle
	QuestionEvent ledger.Event
	ProposalEvent ledger.Event
}

// CreateClaimProposalRequest는 claim record와 proposal event/bundle을 함께 만드는 요청이다.
type CreateClaimProposalRequest struct {
	ClaimEvent    ledger.AppendRequest
	Claim         researchrecords.CreateClaimRecordRequest
	ProposalEvent ledger.AppendRequest
	Proposal      CreateProposalBundleRequest
}

// ClaimProposalResult는 claim proposal 생성 결과다.
type ClaimProposalResult struct {
	Claim         researchrecords.ClaimRecord
	Proposal      ProposalBundle
	ClaimEvent    ledger.Event
	ProposalEvent ledger.Event
}
