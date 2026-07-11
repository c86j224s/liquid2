package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ResearchRecordStore interface {
	CreateEvidenceRecord(context.Context, EvidenceRecord) error
	GetEvidenceRecord(context.Context, string) (EvidenceRecord, error)
	CreateClaimRecord(context.Context, ClaimRecord) error
	GetClaimRecord(context.Context, string) (ClaimRecord, error)
	CreateQuestionRecord(context.Context, QuestionRecord) error
	GetQuestionRecord(context.Context, string) (QuestionRecord, error)
	CreateOptionRecord(context.Context, OptionRecord) error
	GetOptionRecord(context.Context, string) (OptionRecord, error)
	CreateProposalBundle(context.Context, ProposalBundle) error
	GetProposalBundle(context.Context, string) (ProposalBundle, error)
	UpdateProposalBundleState(context.Context, ProposalBundleStateUpdate) error
}

type ResearchRecordListStore interface {
	ListEvidenceRecords(context.Context, string) ([]EvidenceRecord, error)
	ListClaimRecords(context.Context, string) ([]ClaimRecord, error)
	ListQuestionRecords(context.Context, string) ([]QuestionRecord, error)
	ListOptionRecords(context.Context, string) ([]OptionRecord, error)
	ListProposalBundles(context.Context, string) ([]ProposalBundle, error)
}

func (s *Service) CreateEvidenceRecord(ctx context.Context, req CreateEvidenceRecordRequest) (EvidenceRecord, error) {
	missionID := strings.TrimSpace(req.MissionID)
	createdEvent, err := s.requireMissionEvent(ctx, missionID, req.CreatedEventID)
	if err != nil {
		return EvidenceRecord{}, err
	}
	record, err := s.buildEvidenceRecord(ctx, req, createdEvent)
	if err != nil {
		return EvidenceRecord{}, err
	}
	if err := s.store.CreateEvidenceRecord(ctx, record); err != nil {
		return EvidenceRecord{}, err
	}
	return record, nil
}

func (s *Service) buildEvidenceRecord(
	ctx context.Context,
	req CreateEvidenceRecordRequest,
	createdEvent LedgerEvent,
) (EvidenceRecord, error) {
	evidenceID := strings.TrimSpace(req.EvidenceID)
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("evd_", evidenceID); err != nil {
		return EvidenceRecord{}, err
	}
	if err := validateID("mis_", missionID); err != nil {
		return EvidenceRecord{}, err
	}
	if strings.TrimSpace(req.Summary) == "" {
		return EvidenceRecord{}, fmt.Errorf("%w: evidence summary is required", ErrInvalidInput)
	}
	evidenceType := strings.TrimSpace(req.EvidenceType)
	if !allowedEvidenceTypes[evidenceType] {
		return EvidenceRecord{}, fmt.Errorf("%w: unsupported evidence type", ErrInvalidInput)
	}
	state, err := normalizeProposedLifecycleState(req.State, "proposed")
	if err != nil {
		return EvidenceRecord{}, err
	}
	if err := validateProducer(req.Producer); err != nil {
		return EvidenceRecord{}, err
	}
	if createdEvent.MissionID != missionID || strings.TrimSpace(createdEvent.EventID) != strings.TrimSpace(req.CreatedEventID) {
		return EvidenceRecord{}, fmt.Errorf("%w: evidence creation event mismatch", ErrInvalidInput)
	}
	if evidenceType == "user_assertion" && !isApprovalProducer(createdEvent.Producer) {
		return EvidenceRecord{}, fmt.Errorf("%w: user assertion evidence requires a user or steering_chat event", ErrInvalidInput)
	}
	if evidenceType != "user_assertion" && len(req.SnapshotRefs) == 0 {
		return EvidenceRecord{}, fmt.Errorf("%w: evidence requires snapshot refs unless it is a user assertion", ErrInvalidInput)
	}
	snapshotRefs, err := s.normalizeSnapshotRefs(ctx, missionID, req.SnapshotRefs)
	if err != nil {
		return EvidenceRecord{}, err
	}
	confidence, err := normalizeConfidence(req.Confidence)
	if err != nil {
		return EvidenceRecord{}, err
	}

	record := EvidenceRecord{
		SchemaVersion:  EvidenceRecordSchemaVersion,
		ObjectKind:     EvidenceRecordObjectKind,
		EvidenceID:     evidenceID,
		MissionID:      missionID,
		State:          state,
		Summary:        strings.TrimSpace(req.Summary),
		EvidenceType:   evidenceType,
		SnapshotRefs:   snapshotRefs,
		Confidence:     confidence,
		Producer:       normalizeProducer(req.Producer),
		CreatedEventID: strings.TrimSpace(req.CreatedEventID),
		CreatedAt:      time.Now().UTC(),
	}
	return record, nil
}

func (s *Service) GetEvidenceRecord(ctx context.Context, evidenceID string) (EvidenceRecord, error) {
	trimmed := strings.TrimSpace(evidenceID)
	if err := validateID("evd_", trimmed); err != nil {
		return EvidenceRecord{}, err
	}
	return s.store.GetEvidenceRecord(ctx, trimmed)
}

func (s *Service) ListEvidenceRecords(ctx context.Context, missionID string) ([]EvidenceRecord, error) {
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

func (s *Service) CreateClaimRecord(ctx context.Context, req CreateClaimRecordRequest) (ClaimRecord, error) {
	missionID := strings.TrimSpace(req.MissionID)
	createdEvent, err := s.requireMissionEvent(ctx, missionID, req.CreatedEventID)
	if err != nil {
		return ClaimRecord{}, err
	}
	record, err := s.buildClaimRecord(ctx, req, createdEvent)
	if err != nil {
		return ClaimRecord{}, err
	}
	if err := s.store.CreateClaimRecord(ctx, record); err != nil {
		return ClaimRecord{}, err
	}
	return record, nil
}

func (s *Service) buildClaimRecord(
	ctx context.Context,
	req CreateClaimRecordRequest,
	createdEvent LedgerEvent,
) (ClaimRecord, error) {
	claimID := strings.TrimSpace(req.ClaimID)
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("clm_", claimID); err != nil {
		return ClaimRecord{}, err
	}
	if err := validateID("mis_", missionID); err != nil {
		return ClaimRecord{}, err
	}
	if strings.TrimSpace(req.Text) == "" {
		return ClaimRecord{}, fmt.Errorf("%w: claim text is required", ErrInvalidInput)
	}
	if createdEvent.MissionID != missionID || strings.TrimSpace(createdEvent.EventID) != strings.TrimSpace(req.CreatedEventID) {
		return ClaimRecord{}, fmt.Errorf("%w: claim creation event mismatch", ErrInvalidInput)
	}
	state, err := normalizeLifecycleState(req.State, "proposed")
	if err != nil {
		return ClaimRecord{}, err
	}
	if state != "approved" && !allowedProposedLifecycleStates[state] {
		return ClaimRecord{}, fmt.Errorf("%w: claim terminal state requires a transition event", ErrInvalidInput)
	}
	claimProposalID, err := claimProposalIDFromCreatedEvent(createdEvent, claimID)
	if err != nil {
		return ClaimRecord{}, err
	}
	claimType := strings.TrimSpace(req.ClaimType)
	if claimType == "" {
		claimType = "descriptive"
	}
	if !allowedClaimTypes[claimType] {
		return ClaimRecord{}, fmt.Errorf("%w: unsupported claim type", ErrInvalidInput)
	}

	supportingIDs, err := normalizeIDList("evd_", req.SupportingEvidenceIDs)
	if err != nil {
		return ClaimRecord{}, err
	}
	opposingIDs, err := normalizeIDList("evd_", req.OpposingEvidenceIDs)
	if err != nil {
		return ClaimRecord{}, err
	}
	if err := s.requireEvidenceRecords(ctx, missionID, append(append([]string{}, supportingIDs...), opposingIDs...)); err != nil {
		return ClaimRecord{}, err
	}
	questionIDs, err := normalizeIDList("qst_", req.DependsOnQuestionIDs)
	if err != nil {
		return ClaimRecord{}, err
	}
	if err := s.requireQuestionRecords(ctx, missionID, questionIDs); err != nil {
		return ClaimRecord{}, err
	}

	userAssertionEventID := strings.TrimSpace(req.UserAssertionEventID)
	if len(supportingIDs)+len(opposingIDs) == 0 && userAssertionEventID == "" {
		return ClaimRecord{}, fmt.Errorf("%w: claim requires evidence ids or a user assertion event", ErrInvalidInput)
	}
	if userAssertionEventID != "" {
		event, err := s.requireMissionEvent(ctx, missionID, userAssertionEventID)
		if err != nil {
			return ClaimRecord{}, err
		}
		if !isApprovalProducer(event.Producer) {
			return ClaimRecord{}, fmt.Errorf("%w: user assertion claim requires a user or steering_chat event", ErrInvalidInput)
		}
	}

	confidence, err := normalizeConfidence(req.Confidence)
	if err != nil {
		return ClaimRecord{}, err
	}
	approval, err := normalizeClaimApproval(req.Approval)
	if err != nil {
		return ClaimRecord{}, err
	}
	if state == "approved" || approval.State == "approved" {
		event, err := s.requireMissionEvent(ctx, missionID, approval.ApprovalEventID)
		if err != nil {
			return ClaimRecord{}, err
		}
		if err := requireClaimApprovalEvent(event, claimID, claimProposalID); err != nil {
			return ClaimRecord{}, err
		}
		state = "approved"
		approval.State = "approved"
		if approval.ApprovedAt.IsZero() {
			approval.ApprovedAt = event.CreatedAt
		}
	}

	record := ClaimRecord{
		SchemaVersion:         ClaimRecordSchemaVersion,
		ObjectKind:            ClaimRecordObjectKind,
		ClaimID:               claimID,
		MissionID:             missionID,
		State:                 state,
		Text:                  strings.TrimSpace(req.Text),
		ClaimType:             claimType,
		SupportingEvidenceIDs: supportingIDs,
		OpposingEvidenceIDs:   opposingIDs,
		DependsOnQuestionIDs:  questionIDs,
		UserAssertionEventID:  userAssertionEventID,
		Confidence:            confidence,
		Approval:              approval,
		CreatedEventID:        strings.TrimSpace(req.CreatedEventID),
		CreatedAt:             time.Now().UTC(),
	}
	return record, nil
}

func (s *Service) GetClaimRecord(ctx context.Context, claimID string) (ClaimRecord, error) {
	trimmed := strings.TrimSpace(claimID)
	if err := validateID("clm_", trimmed); err != nil {
		return ClaimRecord{}, err
	}
	return s.store.GetClaimRecord(ctx, trimmed)
}

func (s *Service) ListClaimRecords(ctx context.Context, missionID string) ([]ClaimRecord, error) {
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

func (s *Service) CreateQuestionRecord(ctx context.Context, req CreateQuestionRecordRequest) (QuestionRecord, error) {
	missionID := strings.TrimSpace(req.MissionID)
	createdEvent, err := s.requireMissionEvent(ctx, missionID, req.CreatedEventID)
	if err != nil {
		return QuestionRecord{}, err
	}
	record, err := s.buildQuestionRecord(ctx, req, createdEvent)
	if err != nil {
		return QuestionRecord{}, err
	}
	if err := s.store.CreateQuestionRecord(ctx, record); err != nil {
		return QuestionRecord{}, err
	}
	return record, nil
}

func (s *Service) buildQuestionRecord(
	ctx context.Context,
	req CreateQuestionRecordRequest,
	createdEvent LedgerEvent,
) (QuestionRecord, error) {
	questionID := strings.TrimSpace(req.QuestionID)
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("qst_", questionID); err != nil {
		return QuestionRecord{}, err
	}
	if err := validateID("mis_", missionID); err != nil {
		return QuestionRecord{}, err
	}
	if strings.TrimSpace(req.Text) == "" {
		return QuestionRecord{}, fmt.Errorf("%w: question text is required", ErrInvalidInput)
	}
	if createdEvent.MissionID != missionID || strings.TrimSpace(createdEvent.EventID) != strings.TrimSpace(req.CreatedEventID) {
		return QuestionRecord{}, fmt.Errorf("%w: question creation event mismatch", ErrInvalidInput)
	}
	state, err := normalizeQuestionCreateState(req.State)
	if err != nil {
		return QuestionRecord{}, err
	}
	if state == "answered" && strings.TrimSpace(req.Resolution) == "" {
		return QuestionRecord{}, fmt.Errorf("%w: answered question requires a resolution", ErrInvalidInput)
	}
	priority := strings.TrimSpace(req.Priority)
	if priority == "" {
		priority = "medium"
	}
	if !allowedPriorities[priority] {
		return QuestionRecord{}, fmt.Errorf("%w: unsupported question priority", ErrInvalidInput)
	}
	evidenceIDs, err := normalizeIDList("evd_", req.RelatedEvidenceIDs)
	if err != nil {
		return QuestionRecord{}, err
	}
	if err := s.requireEvidenceRecords(ctx, missionID, evidenceIDs); err != nil {
		return QuestionRecord{}, err
	}
	claimIDs, err := normalizeIDList("clm_", req.RelatedClaimIDs)
	if err != nil {
		return QuestionRecord{}, err
	}
	if err := s.requireClaimRecords(ctx, missionID, claimIDs); err != nil {
		return QuestionRecord{}, err
	}

	record := QuestionRecord{
		SchemaVersion:      QuestionRecordSchemaVersion,
		ObjectKind:         QuestionRecordObjectKind,
		QuestionID:         questionID,
		MissionID:          missionID,
		State:              state,
		Text:               strings.TrimSpace(req.Text),
		Priority:           priority,
		Blocking:           req.Blocking,
		RelatedEvidenceIDs: evidenceIDs,
		RelatedClaimIDs:    claimIDs,
		Resolution:         strings.TrimSpace(req.Resolution),
		CreatedEventID:     strings.TrimSpace(req.CreatedEventID),
		CreatedAt:          time.Now().UTC(),
	}
	return record, nil
}

func (s *Service) GetQuestionRecord(ctx context.Context, questionID string) (QuestionRecord, error) {
	trimmed := strings.TrimSpace(questionID)
	if err := validateID("qst_", trimmed); err != nil {
		return QuestionRecord{}, err
	}
	return s.store.GetQuestionRecord(ctx, trimmed)
}

func (s *Service) ListQuestionRecords(ctx context.Context, missionID string) ([]QuestionRecord, error) {
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

func (s *Service) CreateOptionRecord(ctx context.Context, req CreateOptionRecordRequest) (OptionRecord, error) {
	optionID := strings.TrimSpace(req.OptionID)
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("opt_", optionID); err != nil {
		return OptionRecord{}, err
	}
	if err := validateID("mis_", missionID); err != nil {
		return OptionRecord{}, err
	}
	if strings.TrimSpace(req.Title) == "" {
		return OptionRecord{}, fmt.Errorf("%w: option title is required", ErrInvalidInput)
	}
	if _, err := s.requireMissionEvent(ctx, missionID, req.CreatedEventID); err != nil {
		return OptionRecord{}, err
	}
	state, err := normalizeProposedLifecycleState(req.State, "proposed")
	if err != nil {
		return OptionRecord{}, err
	}
	claimIDs, err := normalizeIDList("clm_", req.SupportingClaimIDs)
	if err != nil {
		return OptionRecord{}, err
	}
	if err := s.requireClaimRecords(ctx, missionID, claimIDs); err != nil {
		return OptionRecord{}, err
	}
	riskLevel := strings.TrimSpace(req.RiskLevel)
	if riskLevel == "" {
		riskLevel = "unknown"
	}
	if !allowedRiskLevels[riskLevel] {
		return OptionRecord{}, fmt.Errorf("%w: unsupported option risk level", ErrInvalidInput)
	}

	record := OptionRecord{
		SchemaVersion:      OptionRecordSchemaVersion,
		ObjectKind:         OptionRecordObjectKind,
		OptionID:           optionID,
		MissionID:          missionID,
		State:              state,
		Title:              strings.TrimSpace(req.Title),
		Description:        strings.TrimSpace(req.Description),
		Pros:               normalizeStringList(req.Pros),
		Cons:               normalizeStringList(req.Cons),
		SupportingClaimIDs: claimIDs,
		RiskLevel:          riskLevel,
		CreatedEventID:     strings.TrimSpace(req.CreatedEventID),
		CreatedAt:          time.Now().UTC(),
	}
	if err := s.store.CreateOptionRecord(ctx, record); err != nil {
		return OptionRecord{}, err
	}
	return record, nil
}

func (s *Service) GetOptionRecord(ctx context.Context, optionID string) (OptionRecord, error) {
	trimmed := strings.TrimSpace(optionID)
	if err := validateID("opt_", trimmed); err != nil {
		return OptionRecord{}, err
	}
	return s.store.GetOptionRecord(ctx, trimmed)
}

func (s *Service) ListOptionRecords(ctx context.Context, missionID string) ([]OptionRecord, error) {
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

func (s *Service) CreateProposalBundle(ctx context.Context, req CreateProposalBundleRequest) (ProposalBundle, error) {
	missionID := strings.TrimSpace(req.MissionID)
	createdEvent, err := s.requireMissionEvent(ctx, missionID, req.CreatedEventID)
	if err != nil {
		return ProposalBundle{}, err
	}
	bundle, err := s.buildProposalBundle(ctx, req, createdEvent, nil)
	if err != nil {
		return ProposalBundle{}, err
	}
	if err := s.store.CreateProposalBundle(ctx, bundle); err != nil {
		return ProposalBundle{}, err
	}
	return bundle, nil
}

func (s *Service) buildProposalBundle(
	ctx context.Context,
	req CreateProposalBundleRequest,
	createdEvent LedgerEvent,
	pendingRefs []ObjectRef,
) (ProposalBundle, error) {
	proposalID := strings.TrimSpace(req.ProposalID)
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("prp_", proposalID); err != nil {
		return ProposalBundle{}, err
	}
	if err := validateID("mis_", missionID); err != nil {
		return ProposalBundle{}, err
	}
	if strings.TrimSpace(req.Title) == "" {
		return ProposalBundle{}, fmt.Errorf("%w: proposal title is required", ErrInvalidInput)
	}
	if createdEvent.MissionID != missionID || strings.TrimSpace(createdEvent.EventID) != strings.TrimSpace(req.CreatedEventID) {
		return ProposalBundle{}, fmt.Errorf("%w: proposal creation event mismatch", ErrInvalidInput)
	}
	if err := requireProposalSubmittedEvent(createdEvent, proposalID); err != nil {
		return ProposalBundle{}, err
	}
	state := strings.TrimSpace(req.State)
	if state == "" {
		state = "pending_review"
	}
	if state != "pending_review" {
		return ProposalBundle{}, fmt.Errorf("%w: proposal bundle must start pending_review", ErrInvalidInput)
	}
	requestedDecision := strings.TrimSpace(req.RequestedDecision)
	if !allowedRequestedDecisions[requestedDecision] {
		return ProposalBundle{}, fmt.Errorf("%w: unsupported requested decision", ErrInvalidInput)
	}
	objectRefs, err := s.normalizeObjectRefs(ctx, missionID, req.ObjectRefs, pendingRefs)
	if err != nil {
		return ProposalBundle{}, err
	}
	if len(objectRefs) == 0 {
		return ProposalBundle{}, fmt.Errorf("%w: proposal bundle requires object refs", ErrInvalidInput)
	}

	now := time.Now().UTC()
	bundle := ProposalBundle{
		SchemaVersion:     ProposalBundleSchemaVersion,
		ObjectKind:        ProposalBundleObjectKind,
		ProposalID:        proposalID,
		MissionID:         missionID,
		State:             state,
		Title:             strings.TrimSpace(req.Title),
		ObjectRefs:        objectRefs,
		RequestedDecision: requestedDecision,
		CreatedEventID:    strings.TrimSpace(req.CreatedEventID),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	return bundle, nil
}

func (s *Service) GetProposalBundle(ctx context.Context, proposalID string) (ProposalBundle, error) {
	trimmed := strings.TrimSpace(proposalID)
	if err := validateID("prp_", trimmed); err != nil {
		return ProposalBundle{}, err
	}
	return s.store.GetProposalBundle(ctx, trimmed)
}

func (s *Service) ListProposalBundles(ctx context.Context, missionID string) ([]ProposalBundle, error) {
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

func (s *Service) UpdateProposalBundleState(ctx context.Context, req UpdateProposalBundleStateRequest) (ProposalBundle, error) {
	proposalID := strings.TrimSpace(req.ProposalID)
	if err := validateID("prp_", proposalID); err != nil {
		return ProposalBundle{}, err
	}
	nextState := strings.TrimSpace(req.State)
	if !allowedProposalStates[nextState] || nextState == "pending_review" {
		return ProposalBundle{}, fmt.Errorf("%w: unsupported proposal state transition", ErrInvalidInput)
	}
	current, err := s.store.GetProposalBundle(ctx, proposalID)
	if err != nil {
		return ProposalBundle{}, err
	}
	if !validProposalTransition(current.State, nextState) {
		return ProposalBundle{}, fmt.Errorf("%w: invalid proposal state transition", ErrInvalidInput)
	}
	event, err := s.requireMissionEvent(ctx, current.MissionID, req.DecisionEventID)
	if err != nil {
		return ProposalBundle{}, err
	}
	if !isApprovalProducer(event.Producer) {
		return ProposalBundle{}, fmt.Errorf("%w: proposal decision requires a user or steering_chat event", ErrInvalidInput)
	}
	if err := requireProposalDecisionEvent(event, current, nextState); err != nil {
		return ProposalBundle{}, err
	}

	now := time.Now().UTC()
	update := ProposalBundleStateUpdate{
		ProposalID:      proposalID,
		FromState:       current.State,
		ToState:         nextState,
		DecisionEventID: strings.TrimSpace(req.DecisionEventID),
		DecidedAt:       event.CreatedAt,
		UpdatedAt:       now,
	}
	if update.DecidedAt.IsZero() {
		update.DecidedAt = now
	}
	if err := s.store.UpdateProposalBundleState(ctx, update); err != nil {
		return ProposalBundle{}, err
	}
	return s.store.GetProposalBundle(ctx, proposalID)
}

func (s *Service) normalizeSnapshotRefs(ctx context.Context, missionID string, refs []SnapshotRef) ([]SnapshotRef, error) {
	normalized := make([]SnapshotRef, 0, len(refs))
	seen := map[string]struct{}{}
	for _, ref := range refs {
		snapshotID := strings.TrimSpace(ref.SnapshotID)
		artifactID := strings.TrimSpace(ref.ArtifactID)
		if err := validateID("src_", snapshotID); err != nil {
			return nil, err
		}
		if err := validateID("art_", artifactID); err != nil {
			return nil, err
		}
		key := snapshotID + "\x00" + artifactID
		if _, ok := seen[key]; ok {
			return nil, fmt.Errorf("%w: duplicate snapshot ref", ErrInvalidInput)
		}
		seen[key] = struct{}{}
		snapshot, err := s.store.GetSourceSnapshot(ctx, snapshotID)
		if err != nil {
			return nil, err
		}
		if snapshot.MissionID != missionID {
			return nil, fmt.Errorf("%w: evidence snapshot belongs to another mission", ErrInvalidInput)
		}
		if !containsString(snapshot.ArtifactIDs, artifactID) {
			return nil, fmt.Errorf("%w: evidence artifact is not linked to snapshot", ErrInvalidInput)
		}
		locator := append(json.RawMessage(nil), ref.Locator...)
		if len(locator) == 0 {
			locator = json.RawMessage(`{}`)
		}
		if !json.Valid(locator) {
			return nil, fmt.Errorf("%w: evidence locator must be valid JSON", ErrInvalidInput)
		}
		normalized = append(normalized, SnapshotRef{
			SnapshotID: snapshotID,
			ArtifactID: artifactID,
			Locator:    locator,
		})
	}
	return normalized, nil
}

func (s *Service) normalizeObjectRefs(ctx context.Context, missionID string, refs []ObjectRef, pendingRefs []ObjectRef) ([]ObjectRef, error) {
	normalized := make([]ObjectRef, 0, len(refs))
	seen := map[string]struct{}{}
	pending := map[string]struct{}{}
	for _, ref := range pendingRefs {
		objectKind := strings.TrimSpace(ref.ObjectKind)
		objectID := strings.TrimSpace(ref.ObjectID)
		if objectKind != "" && objectID != "" {
			pending[objectKind+"\x00"+objectID] = struct{}{}
		}
	}
	for _, ref := range refs {
		objectKind := strings.TrimSpace(ref.ObjectKind)
		objectID := strings.TrimSpace(ref.ObjectID)
		key := objectKind + "\x00" + objectID
		if _, ok := seen[key]; ok {
			return nil, fmt.Errorf("%w: duplicate proposal object ref", ErrInvalidInput)
		}
		seen[key] = struct{}{}
		if _, ok := pending[key]; ok {
			if err := validateObjectRefID(objectKind, objectID); err != nil {
				return nil, err
			}
		} else {
			if err := s.requireObjectRef(ctx, missionID, objectKind, objectID); err != nil {
				return nil, err
			}
		}
		normalized = append(normalized, ObjectRef{ObjectKind: objectKind, ObjectID: objectID})
	}
	return normalized, nil
}

func (s *Service) requireObjectRef(ctx context.Context, missionID, objectKind, objectID string) error {
	if err := validateObjectRefID(objectKind, objectID); err != nil {
		return err
	}
	switch objectKind {
	case EvidenceRecordObjectKind:
		record, err := s.store.GetEvidenceRecord(ctx, objectID)
		if err != nil {
			return err
		}
		if record.MissionID != missionID {
			return fmt.Errorf("%w: proposal evidence belongs to another mission", ErrInvalidInput)
		}
	case ClaimRecordObjectKind:
		record, err := s.store.GetClaimRecord(ctx, objectID)
		if err != nil {
			return err
		}
		if record.MissionID != missionID {
			return fmt.Errorf("%w: proposal claim belongs to another mission", ErrInvalidInput)
		}
	case QuestionRecordObjectKind:
		record, err := s.store.GetQuestionRecord(ctx, objectID)
		if err != nil {
			return err
		}
		if record.MissionID != missionID {
			return fmt.Errorf("%w: proposal question belongs to another mission", ErrInvalidInput)
		}
	case OptionRecordObjectKind:
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

func validateObjectRefID(objectKind, objectID string) error {
	switch objectKind {
	case EvidenceRecordObjectKind:
		return validateID("evd_", objectID)
	case ClaimRecordObjectKind:
		return validateID("clm_", objectID)
	case QuestionRecordObjectKind:
		return validateID("qst_", objectID)
	case OptionRecordObjectKind:
		return validateID("opt_", objectID)
	default:
		return fmt.Errorf("%w: unsupported proposal object kind", ErrInvalidInput)
	}
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

func (s *Service) requireMissionEvent(ctx context.Context, missionID, eventID string) (LedgerEvent, error) {
	trimmed := strings.TrimSpace(eventID)
	if err := validateID("evt_", trimmed); err != nil {
		return LedgerEvent{}, err
	}
	events, err := s.store.ListLedgerEvents(ctx, missionID)
	if err != nil {
		return LedgerEvent{}, err
	}
	for _, event := range events {
		if event.EventID == trimmed && event.MissionID == missionID {
			return event, nil
		}
	}
	return LedgerEvent{}, fmt.Errorf("%w: referenced ledger event does not exist in mission", ErrInvalidInput)
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

func normalizeQuestionState(state string) (string, error) {
	trimmed := strings.TrimSpace(state)
	if trimmed == "" {
		trimmed = "open"
	}
	if !allowedQuestionStates[trimmed] {
		return "", fmt.Errorf("%w: unsupported question state", ErrInvalidInput)
	}
	return trimmed, nil
}

func normalizeQuestionCreateState(state string) (string, error) {
	trimmed, err := normalizeQuestionState(state)
	if err != nil {
		return "", err
	}
	if !allowedQuestionCreateStates[trimmed] {
		return "", fmt.Errorf("%w: terminal question state requires a transition event", ErrInvalidInput)
	}
	return trimmed, nil
}

func normalizeConfidence(confidence Confidence) (Confidence, error) {
	level := strings.TrimSpace(confidence.Level)
	if level == "" {
		level = "unknown"
	}
	if !allowedConfidenceLevels[level] {
		return Confidence{}, fmt.Errorf("%w: unsupported confidence level", ErrInvalidInput)
	}
	return Confidence{
		Level:             level,
		Rationale:         strings.TrimSpace(confidence.Rationale),
		OpenRisks:         normalizeStringList(confidence.OpenRisks),
		NeedsVerification: confidence.NeedsVerification,
	}, nil
}

func normalizeClaimApproval(approval Approval) (Approval, error) {
	state := strings.TrimSpace(approval.State)
	if state == "" {
		state = "pending"
	}
	if !allowedApprovalStates[state] {
		return Approval{}, fmt.Errorf("%w: unsupported approval state", ErrInvalidInput)
	}
	return Approval{
		State:           state,
		Required:        true,
		ApprovalEventID: strings.TrimSpace(approval.ApprovalEventID),
		ApprovedAt:      approval.ApprovedAt,
	}, nil
}

func normalizeProducer(producer Producer) Producer {
	return Producer{
		Type: strings.TrimSpace(producer.Type),
		ID:   strings.TrimSpace(producer.ID),
	}
}

func validateProducer(producer Producer) error {
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

func normalizeStringList(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return normalized
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func claimProposalIDFromCreatedEvent(event LedgerEvent, claimID string) (string, error) {
	if event.EventType != "claim.proposed" {
		return "", fmt.Errorf("%w: claim requires a claim.proposed creation event", ErrInvalidInput)
	}
	var payload struct {
		ClaimID    string `json:"claim_id"`
		ProposalID string `json:"proposal_id"`
	}
	if err := unmarshalEventPayload(event, &payload); err != nil {
		return "", err
	}
	if strings.TrimSpace(payload.ClaimID) != claimID {
		return "", fmt.Errorf("%w: claim.proposed event does not reference claim", ErrInvalidInput)
	}
	return strings.TrimSpace(payload.ProposalID), nil
}

func requireClaimApprovalEvent(event LedgerEvent, claimID, proposalID string) error {
	if !isApprovalProducer(event.Producer) {
		return fmt.Errorf("%w: approved claim requires a user or steering_chat event", ErrInvalidInput)
	}
	switch event.EventType {
	case "claim.approved":
		var payload struct {
			ClaimID string `json:"claim_id"`
		}
		if err := unmarshalEventPayload(event, &payload); err != nil {
			return err
		}
		if strings.TrimSpace(payload.ClaimID) != claimID {
			return fmt.Errorf("%w: claim.approved event does not reference claim", ErrInvalidInput)
		}
		return nil
	case "proposal.approved":
		var payload proposalDecisionPayload
		if err := unmarshalEventPayload(event, &payload); err != nil {
			return err
		}
		if proposalID == "" || strings.TrimSpace(payload.ProposalID) != proposalID {
			return fmt.Errorf("%w: proposal.approved event does not reference claim proposal", ErrInvalidInput)
		}
		if !containsString(trimStringList(payload.ApprovedObjectIDs), claimID) || containsString(trimStringList(payload.RejectedObjectIDs), claimID) {
			return fmt.Errorf("%w: proposal.approved event does not approve claim", ErrInvalidInput)
		}
		return nil
	default:
		return fmt.Errorf("%w: approved claim requires a claim or proposal approval event", ErrInvalidInput)
	}
}

func requireProposalSubmittedEvent(event LedgerEvent, proposalID string) error {
	if event.EventType != "proposal.submitted" {
		return fmt.Errorf("%w: proposal bundle requires a proposal.submitted creation event", ErrInvalidInput)
	}
	var payload struct {
		ProposalID string `json:"proposal_id"`
	}
	if err := unmarshalEventPayload(event, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.ProposalID) != proposalID {
		return fmt.Errorf("%w: proposal.submitted event does not reference proposal", ErrInvalidInput)
	}
	return nil
}

func validProposalTransition(from, to string) bool {
	return from == "pending_review" && (to == "approved" || to == "partially_approved" || to == "rejected" || to == "withdrawn")
}

type proposalDecisionPayload struct {
	ProposalID        string   `json:"proposal_id"`
	ApprovedObjectIDs []string `json:"approved_object_ids"`
	RejectedObjectIDs []string `json:"rejected_object_ids"`
}

func requireProposalDecisionEvent(event LedgerEvent, bundle ProposalBundle, targetState string) error {
	if !proposalDecisionEventTypeMatches(event, targetState) {
		return fmt.Errorf("%w: proposal decision event does not match target state", ErrInvalidInput)
	}
	if targetState == "withdrawn" {
		return requireProposalIDPayload(event, bundle.ProposalID)
	}
	var payload proposalDecisionPayload
	if err := unmarshalEventPayload(event, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.ProposalID) != bundle.ProposalID {
		return fmt.Errorf("%w: proposal decision event does not reference proposal", ErrInvalidInput)
	}
	approvedIDs := trimStringList(payload.ApprovedObjectIDs)
	rejectedIDs := trimStringList(payload.RejectedObjectIDs)
	refIDs := objectRefIDs(bundle.ObjectRefs)
	if err := requireDecisionIDsInRefs(approvedIDs, refIDs); err != nil {
		return err
	}
	if err := requireDecisionIDsInRefs(rejectedIDs, refIDs); err != nil {
		return err
	}
	if overlaps(approvedIDs, rejectedIDs) {
		return fmt.Errorf("%w: proposal decision ids overlap", ErrInvalidInput)
	}
	switch targetState {
	case "approved":
		if !sameStringSet(approvedIDs, refIDs) || len(rejectedIDs) != 0 {
			return fmt.Errorf("%w: proposal.approved payload must approve all refs", ErrInvalidInput)
		}
	case "rejected":
		if !sameStringSet(rejectedIDs, refIDs) || len(approvedIDs) != 0 {
			return fmt.Errorf("%w: proposal.rejected payload must reject all refs", ErrInvalidInput)
		}
	case "partially_approved":
		if len(approvedIDs) == 0 || len(rejectedIDs) == 0 {
			return fmt.Errorf("%w: proposal.partially_approved payload needs approved and rejected ids", ErrInvalidInput)
		}
		if !sameStringSet(append(append([]string{}, approvedIDs...), rejectedIDs...), refIDs) {
			return fmt.Errorf("%w: proposal.partially_approved payload must cover all refs", ErrInvalidInput)
		}
	}
	return nil
}

func proposalDecisionEventTypeMatches(event LedgerEvent, targetState string) bool {
	switch targetState {
	case "approved":
		return event.EventType == "proposal.approved"
	case "partially_approved":
		return event.EventType == "proposal.partially_approved"
	case "rejected":
		return event.EventType == "proposal.rejected"
	case "withdrawn":
		return event.EventType == "proposal.withdrawn" || event.EventType == "proposal.rejected"
	default:
		return false
	}
}

func requireProposalIDPayload(event LedgerEvent, proposalID string) error {
	var payload struct {
		ProposalID string `json:"proposal_id"`
	}
	if err := unmarshalEventPayload(event, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.ProposalID) != proposalID {
		return fmt.Errorf("%w: proposal event does not reference proposal", ErrInvalidInput)
	}
	return nil
}

func unmarshalEventPayload(event LedgerEvent, target any) error {
	payload := event.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
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

func objectRefIDs(refs []ObjectRef) []string {
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		ids = append(ids, ref.ObjectID)
	}
	return trimStringList(ids)
}

func requireDecisionIDsInRefs(decisionIDs, refIDs []string) error {
	for _, id := range decisionIDs {
		if !containsString(refIDs, id) {
			return fmt.Errorf("%w: proposal decision references unknown object", ErrInvalidInput)
		}
	}
	return nil
}

func sameStringSet(left, right []string) bool {
	left = trimStringList(left)
	right = trimStringList(right)
	if len(left) != len(right) {
		return false
	}
	for _, value := range left {
		if !containsString(right, value) {
			return false
		}
	}
	return true
}

func overlaps(left, right []string) bool {
	for _, value := range left {
		if containsString(right, value) {
			return true
		}
	}
	return false
}

var allowedEvidenceTypes = map[string]bool{
	"quote":          true,
	"fact":           true,
	"table_row":      true,
	"statistic":      true,
	"observation":    true,
	"interpretation": true,
	"reaction":       true,
	"rumor":          true,
	"controversy":    true,
	"market_signal":  true,
	"code":           true,
	"formula":        true,
	"benchmark":      true,
	"open_question":  true,
	"user_assertion": true,
}

var allowedClaimTypes = map[string]bool{
	"descriptive":    true,
	"evaluative":     true,
	"recommendation": true,
	"risk":           true,
	"decision":       true,
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

var allowedQuestionStates = map[string]bool{
	"open":        true,
	"in_progress": true,
	"answered":    true,
	"rejected":    true,
	"superseded":  true,
	"reopened":    true,
}

var allowedQuestionCreateStates = map[string]bool{
	"open":        true,
	"in_progress": true,
}

var allowedPriorities = map[string]bool{
	"low":    true,
	"medium": true,
	"high":   true,
}

var allowedRiskLevels = map[string]bool{
	"low":     true,
	"medium":  true,
	"high":    true,
	"unknown": true,
}

var allowedConfidenceLevels = map[string]bool{
	"low":     true,
	"medium":  true,
	"high":    true,
	"unknown": true,
}

var allowedApprovalStates = map[string]bool{
	"not_required": true,
	"pending":      true,
	"approved":     true,
	"rejected":     true,
}

var allowedProposalStates = map[string]bool{
	"pending_review":     true,
	"approved":           true,
	"partially_approved": true,
	"rejected":           true,
	"withdrawn":          true,
}

var allowedRequestedDecisions = map[string]bool{
	"approve": true,
	"reject":  true,
	"revise":  true,
	"split":   true,
}
