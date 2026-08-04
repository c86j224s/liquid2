package sqlite

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

// CreateEvidenceRecord는 evidence record를 저장하고 관련 이벤트를 남긴다.
func (s *Store) CreateEvidenceRecord(ctx context.Context, record app.EvidenceRecord) error {
	return s.research.CreateEvidenceRecord(ctx, record)
}

// GetEvidenceRecord는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) GetEvidenceRecord(ctx context.Context, evidenceID string) (app.EvidenceRecord, error) {
	return s.research.GetEvidenceRecord(ctx, evidenceID)
}

// ListEvidenceRecords는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) ListEvidenceRecords(ctx context.Context, missionID string) ([]app.EvidenceRecord, error) {
	return s.research.ListEvidenceRecords(ctx, missionID)
}

// CreateClaimRecord는 claim record를 저장하고 관련 이벤트를 남긴다.
func (s *Store) CreateClaimRecord(ctx context.Context, record app.ClaimRecord) error {
	return s.research.CreateClaimRecord(ctx, record)
}

// GetClaimRecord는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) GetClaimRecord(ctx context.Context, claimID string) (app.ClaimRecord, error) {
	return s.research.GetClaimRecord(ctx, claimID)
}

// ListClaimRecords는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) ListClaimRecords(ctx context.Context, missionID string) ([]app.ClaimRecord, error) {
	return s.research.ListClaimRecords(ctx, missionID)
}

// CreateQuestionRecord는 question record를 저장하고 관련 이벤트를 남긴다.
func (s *Store) CreateQuestionRecord(ctx context.Context, record app.QuestionRecord) error {
	return s.research.CreateQuestionRecord(ctx, record)
}

// GetQuestionRecord는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) GetQuestionRecord(ctx context.Context, questionID string) (app.QuestionRecord, error) {
	return s.research.GetQuestionRecord(ctx, questionID)
}

// ListQuestionRecords는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) ListQuestionRecords(ctx context.Context, missionID string) ([]app.QuestionRecord, error) {
	return s.research.ListQuestionRecords(ctx, missionID)
}

// CreateOptionRecord는 option record를 저장하고 관련 이벤트를 남긴다.
func (s *Store) CreateOptionRecord(ctx context.Context, record app.OptionRecord) error {
	return s.research.CreateOptionRecord(ctx, record)
}

// GetOptionRecord는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) GetOptionRecord(ctx context.Context, optionID string) (app.OptionRecord, error) {
	return s.research.GetOptionRecord(ctx, optionID)
}

// ListOptionRecords는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) ListOptionRecords(ctx context.Context, missionID string) ([]app.OptionRecord, error) {
	return s.research.ListOptionRecords(ctx, missionID)
}

// CreateProposalBundle는 proposal 묶음과 구성 record를 한 단위로 저장한다.
func (s *Store) CreateProposalBundle(ctx context.Context, bundle app.ProposalBundle) error {
	return s.research.CreateProposalBundle(ctx, bundle)
}

// GetProposalBundle는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) GetProposalBundle(ctx context.Context, proposalID string) (app.ProposalBundle, error) {
	return s.research.GetProposalBundle(ctx, proposalID)
}

// ListProposalBundles는 SQLite 저장소 어댑터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Store) ListProposalBundles(ctx context.Context, missionID string) ([]app.ProposalBundle, error) {
	return s.research.ListProposalBundles(ctx, missionID)
}

// UpdateProposalBundleState는 SQLite 저장소 어댑터의 명시적 상태 전이를 수행한다. 결과는 SQLite 기록으로 확인한다.
func (s *Store) UpdateProposalBundleState(ctx context.Context, update app.ProposalBundleStateUpdate) error {
	return s.research.UpdateProposalBundleState(ctx, update)
}
