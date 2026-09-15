package app

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/researchproposal"
	"github.com/c86j224s/liquid2/plasma/internal/researchrecords"
	"strings"
)

// ResearchRecordStore는 app-owned record와 evidence 저장 계약을 모은 포트다.
type ResearchRecordStore interface {
	researchrecords.EvidenceStore
	researchrecords.ClaimStore
	researchrecords.QuestionStore
	researchrecords.OptionStore
	researchproposal.Store
	CreateClaimRecord(context.Context, researchrecords.ClaimRecord) error
	GetClaimRecord(context.Context, string) (researchrecords.ClaimRecord, error)
}

// ResearchRecordListStore는 research record 목록 조회 전용 저장소 포트다.
type ResearchRecordListStore interface {
	researchrecords.EvidenceListStore
	researchrecords.ClaimListStore
	researchrecords.QuestionListStore
	researchrecords.OptionListStore
	ListClaimRecords(context.Context, string) ([]researchrecords.ClaimRecord, error)
	researchproposal.ListStore
}

// CreateEvidenceRecord는 evidence record를 저장하고 관련 이벤트를 남긴다.
func (s *Service) CreateEvidenceRecord(ctx context.Context, req researchrecords.CreateEvidenceRecordRequest) (researchrecords.EvidenceRecord, error) {
	missionID := strings.TrimSpace(req.MissionID)
	createdEvent, err := s.requireMissionEvent(ctx, missionID, req.CreatedEventID)
	if err != nil {
		return researchrecords.EvidenceRecord{}, err
	}
	record, err := researchrecords.BuildEvidenceRecord(ctx, s.store, researchrecords.CreateEvidenceRecordRequest(req), createdEvent)
	if err != nil {
		return researchrecords.EvidenceRecord{}, err
	}
	if err := s.store.CreateEvidenceRecord(ctx, record); err != nil {
		return researchrecords.EvidenceRecord{}, err
	}
	return record, nil
}

// GetEvidenceRecord는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) GetEvidenceRecord(ctx context.Context, evidenceID string) (researchrecords.EvidenceRecord, error) {
	trimmed := strings.TrimSpace(evidenceID)
	if err := validateID("evd_", trimmed); err != nil {
		return researchrecords.EvidenceRecord{}, err
	}
	return s.store.GetEvidenceRecord(ctx, trimmed)
}

// ListEvidenceRecords는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) ListEvidenceRecords(ctx context.Context, missionID string) ([]researchrecords.EvidenceRecord, error) {
	trimmed := strings.TrimSpace(missionID)
	if err := validateID("mis_", trimmed); err != nil {
		return nil, err
	}
	store, ok := s.store.(ResearchRecordListStore)
	if !ok {
		return nil, fmt.Errorf("%w: research record list store is required", ErrInvalidInput)
	}
	return store.ListEvidenceRecords(ctx, trimmed)
}

// CreateClaimRecord는 claim record를 저장하고 관련 이벤트를 남긴다.
func (s *Service) CreateClaimRecord(ctx context.Context, req researchrecords.CreateClaimRecordRequest) (researchrecords.ClaimRecord, error) {
	missionID := strings.TrimSpace(req.MissionID)
	createdEvent, err := s.requireMissionEvent(ctx, missionID, req.CreatedEventID)
	if err != nil {
		return researchrecords.ClaimRecord{}, err
	}
	record, err := researchrecords.BuildClaimRecord(ctx, researchrecords.ClaimRequirements{
		RequireEvidenceRecords: s.requireEvidenceRecords,
		RequireQuestionRecords: s.requireQuestionRecords,
		RequireMissionEvent:    s.requireMissionEvent,
	}, req, createdEvent)
	if err != nil {
		return researchrecords.ClaimRecord{}, err
	}
	if err := s.store.CreateClaimRecord(ctx, record); err != nil {
		return researchrecords.ClaimRecord{}, err
	}
	return record, nil
}

// GetClaimRecord는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) GetClaimRecord(ctx context.Context, claimID string) (researchrecords.ClaimRecord, error) {
	trimmed := strings.TrimSpace(claimID)
	if err := validateID("clm_", trimmed); err != nil {
		return researchrecords.ClaimRecord{}, err
	}
	return s.store.GetClaimRecord(ctx, trimmed)
}

// ListClaimRecords는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) ListClaimRecords(ctx context.Context, missionID string) ([]researchrecords.ClaimRecord, error) {
	trimmed := strings.TrimSpace(missionID)
	if err := validateID("mis_", trimmed); err != nil {
		return nil, err
	}
	store, ok := s.store.(ResearchRecordListStore)
	if !ok {
		return nil, fmt.Errorf("%w: research record list store is required", ErrInvalidInput)
	}
	return store.ListClaimRecords(ctx, trimmed)
}

// CreateQuestionRecord는 question record를 저장하고 관련 이벤트를 남긴다.
func (s *Service) CreateQuestionRecord(ctx context.Context, req researchrecords.CreateQuestionRecordRequest) (researchrecords.QuestionRecord, error) {
	missionID := strings.TrimSpace(req.MissionID)
	createdEvent, err := s.requireMissionEvent(ctx, missionID, req.CreatedEventID)
	if err != nil {
		return researchrecords.QuestionRecord{}, err
	}
	record, err := researchrecords.BuildQuestionRecord(ctx, researchrecords.QuestionRequirements{
		RequireEvidenceRecords: s.requireEvidenceRecords,
		RequireClaimRecords:    s.requireClaimRecords,
	}, req, createdEvent)
	if err != nil {
		return researchrecords.QuestionRecord{}, err
	}
	if err := s.store.CreateQuestionRecord(ctx, record); err != nil {
		return researchrecords.QuestionRecord{}, err
	}
	return record, nil
}

// GetQuestionRecord는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) GetQuestionRecord(ctx context.Context, questionID string) (researchrecords.QuestionRecord, error) {
	trimmed := strings.TrimSpace(questionID)
	if err := validateID("qst_", trimmed); err != nil {
		return researchrecords.QuestionRecord{}, err
	}
	return s.store.GetQuestionRecord(ctx, trimmed)
}

// ListQuestionRecords는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) ListQuestionRecords(ctx context.Context, missionID string) ([]researchrecords.QuestionRecord, error) {
	trimmed := strings.TrimSpace(missionID)
	if err := validateID("mis_", trimmed); err != nil {
		return nil, err
	}
	store, ok := s.store.(ResearchRecordListStore)
	if !ok {
		return nil, fmt.Errorf("%w: research record list store is required", ErrInvalidInput)
	}
	return store.ListQuestionRecords(ctx, trimmed)
}

// CreateOptionRecord는 option record를 저장하고 관련 이벤트를 남긴다.
func (s *Service) CreateOptionRecord(ctx context.Context, req researchrecords.CreateOptionRecordRequest) (researchrecords.OptionRecord, error) {
	record, err := researchrecords.BuildOptionRecord(ctx, researchrecords.OptionRequirements{
		RequireMissionEvent: s.requireMissionEvent,
		RequireClaimRecords: s.requireClaimRecords,
	}, req)
	if err != nil {
		return researchrecords.OptionRecord{}, err
	}
	if err := s.store.CreateOptionRecord(ctx, record); err != nil {
		return researchrecords.OptionRecord{}, err
	}
	return record, nil
}

// GetOptionRecord는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) GetOptionRecord(ctx context.Context, optionID string) (researchrecords.OptionRecord, error) {
	trimmed := strings.TrimSpace(optionID)
	if err := validateID("opt_", trimmed); err != nil {
		return researchrecords.OptionRecord{}, err
	}
	return s.store.GetOptionRecord(ctx, trimmed)
}

// ListOptionRecords는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) ListOptionRecords(ctx context.Context, missionID string) ([]researchrecords.OptionRecord, error) {
	trimmed := strings.TrimSpace(missionID)
	if err := validateID("mis_", trimmed); err != nil {
		return nil, err
	}
	store, ok := s.store.(ResearchRecordListStore)
	if !ok {
		return nil, fmt.Errorf("%w: research record list store is required", ErrInvalidInput)
	}
	return store.ListOptionRecords(ctx, trimmed)
}

// CreateProposalBundle는 proposal 묶음과 구성 record를 한 단위로 저장한다.
func (s *Service) CreateProposalBundle(ctx context.Context, req researchproposal.CreateProposalBundleRequest) (researchproposal.ProposalBundle, error) {
	missionID := strings.TrimSpace(req.MissionID)
	createdEvent, err := s.requireMissionEvent(ctx, missionID, req.CreatedEventID)
	if err != nil {
		return researchproposal.ProposalBundle{}, err
	}
	bundle, err := researchproposal.BuildProposalBundle(ctx, s.requireObjectRef, req, createdEvent, nil)
	if err != nil {
		return researchproposal.ProposalBundle{}, err
	}
	if err := s.store.CreateProposalBundle(ctx, bundle); err != nil {
		return researchproposal.ProposalBundle{}, err
	}
	return bundle, nil
}

// GetProposalBundle는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) GetProposalBundle(ctx context.Context, proposalID string) (researchproposal.ProposalBundle, error) {
	trimmed := strings.TrimSpace(proposalID)
	if err := validateID("prp_", trimmed); err != nil {
		return researchproposal.ProposalBundle{}, err
	}
	return s.store.GetProposalBundle(ctx, trimmed)
}

// ListProposalBundles는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) ListProposalBundles(ctx context.Context, missionID string) ([]researchproposal.ProposalBundle, error) {
	trimmed := strings.TrimSpace(missionID)
	if err := validateID("mis_", trimmed); err != nil {
		return nil, err
	}
	store, ok := s.store.(ResearchRecordListStore)
	if !ok {
		return nil, fmt.Errorf("%w: research record list store is required", ErrInvalidInput)
	}
	return store.ListProposalBundles(ctx, trimmed)
}

// UpdateProposalBundleState는 애플리케이션 서비스 계층의 명시적 상태 전이를 수행한다. 결과는 장부나 저장소 기록으로 확인한다.
func (s *Service) UpdateProposalBundleState(ctx context.Context, req researchproposal.UpdateProposalBundleStateRequest) (researchproposal.ProposalBundle, error) {
	proposalID, nextState, err := researchproposal.ValidateStateChangeRequest(req)
	if err != nil {
		return researchproposal.ProposalBundle{}, err
	}
	current, err := s.store.GetProposalBundle(ctx, proposalID)
	if err != nil {
		return researchproposal.ProposalBundle{}, err
	}
	if err := researchproposal.ValidateTransition(current, nextState); err != nil {
		return researchproposal.ProposalBundle{}, err
	}
	event, err := s.requireMissionEvent(ctx, current.MissionID, req.DecisionEventID)
	if err != nil {
		return researchproposal.ProposalBundle{}, err
	}
	update, err := researchproposal.BuildStateUpdate(req, current, event, nextState)
	if err != nil {
		return researchproposal.ProposalBundle{}, err
	}
	if err := s.store.UpdateProposalBundleState(ctx, update); err != nil {
		return researchproposal.ProposalBundle{}, err
	}
	return s.store.GetProposalBundle(ctx, proposalID)
}

func (s *Service) requireObjectRef(ctx context.Context, missionID, objectKind, objectID string) error {
	switch objectKind {
	case researchrecords.EvidenceRecordObjectKind:
		record, err := s.store.GetEvidenceRecord(ctx, objectID)
		if err != nil {
			return err
		}
		if record.MissionID != missionID {
			return fmt.Errorf("%w: proposal evidence belongs to another mission", ErrInvalidInput)
		}
	case researchrecords.ClaimRecordObjectKind:
		record, err := s.store.GetClaimRecord(ctx, objectID)
		if err != nil {
			return err
		}
		if record.MissionID != missionID {
			return fmt.Errorf("%w: proposal claim belongs to another mission", ErrInvalidInput)
		}
	case researchrecords.QuestionRecordObjectKind:
		record, err := s.store.GetQuestionRecord(ctx, objectID)
		if err != nil {
			return err
		}
		if record.MissionID != missionID {
			return fmt.Errorf("%w: proposal question belongs to another mission", ErrInvalidInput)
		}
	case researchrecords.OptionRecordObjectKind:
		record, err := s.store.GetOptionRecord(ctx, objectID)
		if err != nil {
			return err
		}
		if record.MissionID != missionID {
			return fmt.Errorf("%w: proposal option belongs to another mission", ErrInvalidInput)
		}
	default:
		return fmt.Errorf("%w: unsupported proposal object kind", ErrInvalidInput)
	}
	return nil
}

func (s *Service) requireEvidenceRecords(ctx context.Context, missionID string, evidenceIDs []string) error {
	for _, evidenceID := range evidenceIDs {
		record, err := s.store.GetEvidenceRecord(ctx, evidenceID)
		if err != nil {
			return err
		}
		if record.MissionID != missionID {
			return fmt.Errorf("%w: evidence belongs to another mission", ErrInvalidInput)
		}
	}
	return nil
}

func (s *Service) requireClaimRecords(ctx context.Context, missionID string, claimIDs []string) error {
	for _, claimID := range claimIDs {
		record, err := s.store.GetClaimRecord(ctx, claimID)
		if err != nil {
			return err
		}
		if record.MissionID != missionID {
			return fmt.Errorf("%w: claim belongs to another mission", ErrInvalidInput)
		}
	}
	return nil
}

func (s *Service) requireQuestionRecords(ctx context.Context, missionID string, questionIDs []string) error {
	for _, questionID := range questionIDs {
		record, err := s.store.GetQuestionRecord(ctx, questionID)
		if err != nil {
			return err
		}
		if record.MissionID != missionID {
			return fmt.Errorf("%w: question belongs to another mission", ErrInvalidInput)
		}
	}
	return nil
}

func (s *Service) requireMissionEvent(ctx context.Context, missionID, eventID string) (ledger.Event, error) {
	trimmed := strings.TrimSpace(eventID)
	if err := validateID("evt_", trimmed); err != nil {
		return ledger.Event{}, err
	}
	events, err := s.store.ListLedgerEvents(ctx, missionID)
	if err != nil {
		return ledger.Event{}, err
	}
	for _, event := range events {
		if event.EventID == trimmed && event.MissionID == missionID {
			return event, nil
		}
	}
	return ledger.Event{}, fmt.Errorf("%w: referenced ledger event does not exist in mission", ErrInvalidInput)
}

func normalizeLifecycleState(state, defaultState string) (string, error) {
	trimmed := strings.TrimSpace(state)
	if trimmed == "" {
		trimmed = defaultState
	}
	if !allowedLifecycleStates[trimmed] {
		return "", fmt.Errorf("%w: unsupported lifecycle state", ErrInvalidInput)
	}
	return trimmed, nil
}

func normalizeProposedLifecycleState(state, defaultState string) (string, error) {
	trimmed, err := normalizeLifecycleState(state, defaultState)
	if err != nil {
		return "", err
	}
	if !allowedProposedLifecycleStates[trimmed] {
		return "", fmt.Errorf("%w: terminal lifecycle state requires a transition event", ErrInvalidInput)
	}
	return trimmed, nil
}

func normalizeProducer(producer ledger.Producer) ledger.Producer {
	return ledger.Producer{
		Type: strings.TrimSpace(producer.Type),
		ID:   strings.TrimSpace(producer.ID),
	}
}

func validateProducer(producer ledger.Producer) error {
	if strings.TrimSpace(producer.Type) == "" || strings.TrimSpace(producer.ID) == "" {
		return fmt.Errorf("%w: producer type and id are required", ErrInvalidInput)
	}
	return nil
}

func normalizeIDList(prefix string, ids []string) ([]string, error) {
	normalized := make([]string, 0, len(ids))
	seen := map[string]struct{}{}
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		if err := validateID(prefix, trimmed); err != nil {
			return nil, err
		}
		if _, ok := seen[trimmed]; ok {
			return nil, fmt.Errorf("%w: duplicate id", ErrInvalidInput)
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized, nil
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func unmarshalEventPayload(event ledger.Event, target any) error {
	payload := event.Payload
	if len(payload) == 0 {
		payload = []byte(`{}`)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("%w: invalid ledger event payload", ErrInvalidInput)
	}
	return nil
}

func trimStringList(values []string) []string {
	trimmed := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		candidate := strings.TrimSpace(value)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		trimmed = append(trimmed, candidate)
	}
	return trimmed
}

var allowedLifecycleStates = map[string]bool{
	"draft":        true,
	"proposed":     true,
	"needs_review": true,
	"approved":     true,
	"rejected":     true,
	"superseded":   true,
	"archived":     true,
}

var allowedProposedLifecycleStates = map[string]bool{
	"draft":        true,
	"proposed":     true,
	"needs_review": true,
}
