package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	htmlpkg "html"
	"strings"
	"time"
)

type ReportStore interface {
	CreateReport(context.Context, Report) error
	GetReport(context.Context, string) (Report, error)
	CreateReportVersion(context.Context, ReportVersion, []ReportBlock) error
	GetReportVersion(context.Context, string) (ReportVersion, error)
	ListReportBlocks(context.Context, string) ([]ReportBlock, error)
	PromoteReportVersion(context.Context, ReportVersionPromotion) error
}

type ReportListStore interface {
	ListReports(context.Context, string) ([]Report, error)
	ListReportVersions(context.Context, string) ([]ReportVersion, error)
}

type reportScopeRecords struct {
	Claims    []ClaimRecord
	Evidence  []EvidenceRecord
	Questions []QuestionRecord
	Options   []OptionRecord
}

func (s *Service) CreateReportDraft(ctx context.Context, req CreateReportDraftRequest) (ReportDraftResult, error) {
	event, report, version, blocks, err := s.buildReportDraft(ctx, req)
	if err != nil {
		return ReportDraftResult{}, err
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:         []LedgerEvent{event},
		Reports:        []Report{report},
		ReportVersions: []ReportVersion{version},
		ReportBlocks:   blocks,
	})
	if err != nil {
		return ReportDraftResult{}, err
	}
	return ReportDraftResult{
		Report:  report,
		Version: version,
		Blocks:  blocks,
		Event:   committed.Events[0],
	}, nil
}

func (s *Service) buildReportDraft(
	ctx context.Context,
	req CreateReportDraftRequest,
) (LedgerEvent, Report, ReportVersion, []ReportBlock, error) {
	reportID := strings.TrimSpace(req.ReportID)
	versionID := strings.TrimSpace(req.ReportVersionID)
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("rpt_", reportID); err != nil {
		return LedgerEvent{}, Report{}, ReportVersion{}, nil, err
	}
	if err := validateID("rvn_", versionID); err != nil {
		return LedgerEvent{}, Report{}, ReportVersion{}, nil, err
	}
	if err := validateID("mis_", missionID); err != nil {
		return LedgerEvent{}, Report{}, ReportVersion{}, nil, err
	}
	if err := validateProducer(req.Producer); err != nil {
		return LedgerEvent{}, Report{}, ReportVersion{}, nil, err
	}
	baseVersionID := strings.TrimSpace(req.BaseVersionID)
	if baseVersionID != "" {
		return LedgerEvent{}, Report{}, ReportVersion{}, nil, fmt.Errorf("%w: report revisions are not implemented yet", ErrInvalidInput)
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "Research report"
	}
	formatIntent := strings.TrimSpace(req.FormatIntent)
	if formatIntent == "" {
		formatIntent = "briefing"
	}
	if !allowedReportFormatIntents[formatIntent] {
		return LedgerEvent{}, Report{}, ReportVersion{}, nil, fmt.Errorf("%w: unsupported report format intent", ErrInvalidInput)
	}

	scope, err := s.normalizeReportScope(ctx, missionID, req.Scope)
	if err != nil {
		return LedgerEvent{}, Report{}, ReportVersion{}, nil, err
	}
	records, err := s.resolveReportScope(ctx, missionID, scope)
	if err != nil {
		return LedgerEvent{}, Report{}, ReportVersion{}, nil, err
	}
	if len(records.Claims)+len(records.Evidence)+len(records.Questions)+len(records.Options) == 0 {
		return LedgerEvent{}, Report{}, ReportVersion{}, nil, fmt.Errorf("%w: report draft requires scoped records", ErrInvalidInput)
	}

	event, err := buildLedgerEvent(AppendEventRequest{
		EventID:   req.CreatedEventID,
		MissionID: missionID,
		EventType: "report.drafted",
		Producer:  normalizeProducer(req.Producer),
		Payload:   mustMarshalJSON(reportDraftPayload(reportID, versionID, formatIntent, scope, req.Generation)),
	})
	if err != nil {
		return LedgerEvent{}, Report{}, ReportVersion{}, nil, err
	}

	now := time.Now().UTC()
	var blocks []ReportBlock
	if len(req.Blocks) > 0 {
		for _, block := range req.Blocks {
			if err := validateReportBlockRefs(block.SourceRefs, records); err != nil {
				return LedgerEvent{}, Report{}, ReportVersion{}, nil, err
			}
		}
		blocks, err = buildReportBlocksFromDraftInputs(versionID, missionID, normalizeProducer(req.Producer), req.Blocks)
	} else {
		blocks, err = buildDraftReportBlocks(versionID, missionID, title, normalizeProducer(req.Producer), records)
	}
	if err != nil {
		return LedgerEvent{}, Report{}, ReportVersion{}, nil, err
	}
	blockIDs := make([]string, 0, len(blocks))
	for _, block := range blocks {
		blockIDs = append(blockIDs, block.BlockID)
	}
	report := Report{
		SchemaVersion:   ReportSchemaVersion,
		ObjectKind:      ReportObjectKind,
		ReportID:        reportID,
		MissionID:       missionID,
		Title:           title,
		ActiveVersionID: versionID,
		State:           "draft",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	version := ReportVersion{
		SchemaVersion:         ReportVersionSchemaVersion,
		ObjectKind:            ReportVersionObjectKind,
		ReportVersionID:       versionID,
		ReportID:              reportID,
		MissionID:             missionID,
		BaseVersionID:         baseVersionID,
		State:                 "draft",
		RootBlockID:           blocks[0].BlockID,
		BlockIDs:              blockIDs,
		IncludedEvidenceScope: scope,
		CreatedEventID:        event.EventID,
		CreatedAt:             now,
	}
	return event, report, version, blocks, nil
}

func reportDraftPayload(reportID string, versionID string, formatIntent string, scope ReportEvidenceScope, generation map[string]any) map[string]any {
	payload := map[string]any{
		"report_id":         reportID,
		"report_version_id": versionID,
		"format_intent":     formatIntent,
		"evidence_scope":    scope,
	}
	if len(generation) > 0 {
		payload["generation"] = generation
	}
	return payload
}

func (s *Service) GetReport(ctx context.Context, reportID string) (Report, error) {
	trimmed := strings.TrimSpace(reportID)
	if err := validateID("rpt_", trimmed); err != nil {
		return Report{}, err
	}
	return s.store.GetReport(ctx, trimmed)
}

func (s *Service) ListReports(ctx context.Context, missionID string) ([]Report, error) {
	trimmed := strings.TrimSpace(missionID)
	if err := validateID("mis_", trimmed); err != nil {
		return nil, err
	}
	store, ok := s.store.(ReportListStore)
	if !ok {
		return nil, fmt.Errorf("%w: report list store is required", ErrInvalidInput)
	}
	return store.ListReports(ctx, trimmed)
}

func (s *Service) GetReportVersion(ctx context.Context, versionID string) (ReportVersion, error) {
	trimmed := strings.TrimSpace(versionID)
	if err := validateID("rvn_", trimmed); err != nil {
		return ReportVersion{}, err
	}
	return s.store.GetReportVersion(ctx, trimmed)
}

func (s *Service) ListReportVersions(ctx context.Context, missionID string) ([]ReportVersion, error) {
	trimmed := strings.TrimSpace(missionID)
	if err := validateID("mis_", trimmed); err != nil {
		return nil, err
	}
	store, ok := s.store.(ReportListStore)
	if !ok {
		return nil, fmt.Errorf("%w: report list store is required", ErrInvalidInput)
	}
	return store.ListReportVersions(ctx, trimmed)
}

func (s *Service) ListReportBlocks(ctx context.Context, versionID string) ([]ReportBlock, error) {
	trimmed := strings.TrimSpace(versionID)
	if err := validateID("rvn_", trimmed); err != nil {
		return nil, err
	}
	return s.store.ListReportBlocks(ctx, trimmed)
}

func (s *Service) PromoteReportVersion(ctx context.Context, req PromoteReportVersionRequest) (ReportVersion, error) {
	versionID := strings.TrimSpace(req.ReportVersionID)
	if err := validateID("rvn_", versionID); err != nil {
		return ReportVersion{}, err
	}
	version, err := s.store.GetReportVersion(ctx, versionID)
	if err != nil {
		return ReportVersion{}, err
	}
	if version.State != "draft" && version.State != "review" {
		return ReportVersion{}, fmt.Errorf("%w: report version cannot be promoted from current state", ErrInvalidInput)
	}
	event, err := s.requireMissionEvent(ctx, version.MissionID, req.ApprovalEventID)
	if err != nil {
		return ReportVersion{}, err
	}
	if err := requireReportPromotionEvent(event, version.ReportVersionID); err != nil {
		return ReportVersion{}, err
	}
	update := ReportVersionPromotion{
		ReportID:        version.ReportID,
		ReportVersionID: version.ReportVersionID,
		FromState:       version.State,
		ToState:         "export_candidate",
		ReportState:     "export_candidate",
		ApprovalEventID: event.EventID,
		UpdatedAt:       time.Now().UTC(),
	}
	if err := s.store.PromoteReportVersion(ctx, update); err != nil {
		return ReportVersion{}, err
	}
	return s.store.GetReportVersion(ctx, versionID)
}

func BuildReportPromotionAppendRequest(req ReportPromotionAppendRequest) AppendEventRequest {
	version := req.Version
	return AppendEventRequest{
		EventID:   strings.TrimSpace(req.EventID),
		MissionID: strings.TrimSpace(version.MissionID),
		EventType: "report.promoted",
		Producer:  req.Producer,
		Payload: mustMarshalJSON(map[string]any{
			"report_version_id": version.ReportVersionID,
		}),
	}
}

func (s *Service) ExportReportVersion(ctx context.Context, req ExportReportVersionRequest) (ReportExportResult, error) {
	exportID := strings.TrimSpace(req.ExportID)
	versionID := strings.TrimSpace(req.ReportVersionID)
	artifactID := strings.TrimSpace(req.ArtifactID)
	eventID := strings.TrimSpace(req.EventID)
	if err := validateID("exp_", exportID); err != nil {
		return ReportExportResult{}, err
	}
	if err := validateID("rvn_", versionID); err != nil {
		return ReportExportResult{}, err
	}
	if err := validateID("art_", artifactID); err != nil {
		return ReportExportResult{}, err
	}
	if err := validateID("evt_", eventID); err != nil {
		return ReportExportResult{}, err
	}
	if err := validateProducer(req.Producer); err != nil {
		return ReportExportResult{}, err
	}
	producer := normalizeProducer(req.Producer)
	if !isApprovalProducer(producer) {
		return ReportExportResult{}, fmt.Errorf("%w: report export requires user or steering_chat producer", ErrInvalidInput)
	}
	target := strings.TrimSpace(req.Target)
	if !allowedReportExportTargets[target] {
		return ReportExportResult{}, fmt.Errorf("%w: unsupported report export target", ErrInvalidInput)
	}

	version, err := s.store.GetReportVersion(ctx, versionID)
	if err != nil {
		return ReportExportResult{}, err
	}
	if version.State != "export_candidate" {
		return ReportExportResult{}, fmt.Errorf("%w: report version must be an export_candidate", ErrInvalidInput)
	}
	approvalEvent, err := s.requireMissionEvent(ctx, version.MissionID, req.ApprovalEventID)
	if err != nil {
		return ReportExportResult{}, err
	}
	if err := requireReportPromotionEvent(approvalEvent, version.ReportVersionID); err != nil {
		return ReportExportResult{}, err
	}
	blocks, err := s.store.ListReportBlocks(ctx, versionID)
	if err != nil {
		return ReportExportResult{}, err
	}
	content, mediaType, filename, err := renderReportExport(version, blocks, target)
	if err != nil {
		return ReportExportResult{}, err
	}
	artifact, err := buildRawArtifact(CreateRawArtifactRequest{
		ArtifactID: artifactID,
		MissionID:  version.MissionID,
		MediaType:  mediaType,
		Filename:   filename,
		Producer:   producer,
		Content:    content,
	})
	if err != nil {
		return ReportExportResult{}, err
	}
	event, err := buildLedgerEvent(AppendEventRequest{
		EventID:   eventID,
		MissionID: version.MissionID,
		EventType: "report.exported",
		Producer:  producer,
		Payload: mustMarshalJSON(map[string]any{
			"export_id":         exportID,
			"report_version_id": version.ReportVersionID,
			"target":            target,
			"artifact_id":       artifact.ArtifactID,
			"approval_event_id": approvalEvent.EventID,
		}),
	})
	if err != nil {
		return ReportExportResult{}, err
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:       []LedgerEvent{event},
		RawArtifacts: []RawArtifact{artifact},
	})
	if err != nil {
		return ReportExportResult{}, err
	}
	return ReportExportResult{Artifact: artifact, Event: committed.Events[0]}, nil
}

func (s *Service) ReportAST(ctx context.Context, versionID string) (ReportASTExport, error) {
	version, err := s.GetReportVersion(ctx, versionID)
	if err != nil {
		return ReportASTExport{}, err
	}
	blocks, err := s.ListReportBlocks(ctx, version.ReportVersionID)
	if err != nil {
		return ReportASTExport{}, err
	}
	return ReportASTExport{
		SchemaVersion: "plasma.report_ast_export.v1",
		ObjectKind:    "report_ast_export",
		Version:       version,
		Blocks:        blocks,
	}, nil
}

func (s *Service) normalizeReportScope(ctx context.Context, missionID string, scope ReportEvidenceScope) (ReportEvidenceScope, error) {
	var err error
	scope.EvidenceIDs, err = normalizeIDList("evd_", scope.EvidenceIDs)
	if err != nil {
		return ReportEvidenceScope{}, err
	}
	scope.ClaimIDs, err = normalizeIDList("clm_", scope.ClaimIDs)
	if err != nil {
		return ReportEvidenceScope{}, err
	}
	scope.QuestionIDs, err = normalizeIDList("qst_", scope.QuestionIDs)
	if err != nil {
		return ReportEvidenceScope{}, err
	}
	scope.OptionIDs, err = normalizeIDList("opt_", scope.OptionIDs)
	if err != nil {
		return ReportEvidenceScope{}, err
	}
	if scope.IncludeProposed {
		return ReportEvidenceScope{}, fmt.Errorf("%w: report drafts cannot include proposed records", ErrInvalidInput)
	}
	if len(scope.QuestionIDs)+len(scope.OptionIDs) > 0 {
		return ReportEvidenceScope{}, fmt.Errorf("%w: report drafts do not support question or option records until approval semantics are defined", ErrInvalidInput)
	}
	scope.AcceptedOnly = true
	scope.IncludeProposed = false
	if len(scope.EvidenceIDs)+len(scope.ClaimIDs)+len(scope.QuestionIDs)+len(scope.OptionIDs) == 0 {
		projection, err := s.GetProjection(ctx, missionID)
		if err != nil {
			return ReportEvidenceScope{}, err
		}
		scope.AcceptedOnly = true
		scope.IncludeProposed = false
		scope.ClaimIDs = append([]string(nil), projection.AcceptedClaimIDs...)
	}
	return scope, nil
}

func (s *Service) resolveReportScope(ctx context.Context, missionID string, scope ReportEvidenceScope) (reportScopeRecords, error) {
	records := reportScopeRecords{}
	evidenceByID := map[string]EvidenceRecord{}
	addEvidence := func(evidenceID string, direct bool) error {
		if _, ok := evidenceByID[evidenceID]; ok {
			return nil
		}
		evidence, err := s.store.GetEvidenceRecord(ctx, evidenceID)
		if err != nil {
			return err
		}
		if evidence.MissionID != missionID {
			return fmt.Errorf("%w: report evidence belongs to another mission", ErrInvalidInput)
		}
		if evidence.State == "rejected" || evidence.State == "superseded" || evidence.State == "archived" {
			return fmt.Errorf("%w: report evidence is not active", ErrInvalidInput)
		}
		if scope.AcceptedOnly {
			if err := s.requireReportApprovedEvidence(ctx, evidence); err != nil {
				return err
			}
		}
		evidenceByID[evidenceID] = evidence
		records.Evidence = append(records.Evidence, evidence)
		return nil
	}

	for _, evidenceID := range scope.EvidenceIDs {
		if err := addEvidence(evidenceID, true); err != nil {
			return reportScopeRecords{}, err
		}
	}
	for _, claimID := range scope.ClaimIDs {
		claim, err := s.store.GetClaimRecord(ctx, claimID)
		if err != nil {
			return reportScopeRecords{}, err
		}
		if claim.MissionID != missionID {
			return reportScopeRecords{}, fmt.Errorf("%w: report claim belongs to another mission", ErrInvalidInput)
		}
		if scope.AcceptedOnly && claim.State != "approved" {
			if err := s.requireApprovedProposalObject(ctx, missionID, claim.CreatedEventID, claim.ClaimID); err != nil {
				return reportScopeRecords{}, fmt.Errorf("%w: accepted-only report claim must be approved", err)
			}
		}
		if !scope.AcceptedOnly && !scope.IncludeProposed && claim.State != "approved" {
			if err := s.requireApprovedProposalObject(ctx, missionID, claim.CreatedEventID, claim.ClaimID); err != nil {
				return reportScopeRecords{}, fmt.Errorf("%w: report scope excludes proposed claims", err)
			}
		}
		if claim.State == "rejected" || claim.State == "superseded" || claim.State == "archived" {
			return reportScopeRecords{}, fmt.Errorf("%w: report claim is not active", ErrInvalidInput)
		}
		records.Claims = append(records.Claims, claim)
		for _, evidenceID := range append(append([]string{}, claim.SupportingEvidenceIDs...), claim.OpposingEvidenceIDs...) {
			if err := addEvidence(evidenceID, false); err != nil {
				return reportScopeRecords{}, err
			}
		}
	}
	for _, questionID := range scope.QuestionIDs {
		question, err := s.store.GetQuestionRecord(ctx, questionID)
		if err != nil {
			return reportScopeRecords{}, err
		}
		if question.MissionID != missionID {
			return reportScopeRecords{}, fmt.Errorf("%w: report question belongs to another mission", ErrInvalidInput)
		}
		if question.State == "rejected" || question.State == "superseded" {
			return reportScopeRecords{}, fmt.Errorf("%w: report question is not active", ErrInvalidInput)
		}
		records.Questions = append(records.Questions, question)
	}
	for _, optionID := range scope.OptionIDs {
		option, err := s.store.GetOptionRecord(ctx, optionID)
		if err != nil {
			return reportScopeRecords{}, err
		}
		if option.MissionID != missionID {
			return reportScopeRecords{}, fmt.Errorf("%w: report option belongs to another mission", ErrInvalidInput)
		}
		if option.State == "rejected" || option.State == "superseded" || option.State == "archived" {
			return reportScopeRecords{}, fmt.Errorf("%w: report option is not active", ErrInvalidInput)
		}
		records.Options = append(records.Options, option)
	}
	return records, nil
}

func (s *Service) requireReportApprovedEvidence(ctx context.Context, evidence EvidenceRecord) error {
	if evidence.State == "approved" {
		return nil
	}
	if err := s.requireApprovedProposalObject(ctx, evidence.MissionID, evidence.CreatedEventID, evidence.EvidenceID); err != nil {
		return fmt.Errorf("%w: report evidence must be approved", err)
	}
	return nil
}

func (s *Service) requireApprovedProposalObject(ctx context.Context, missionID, createdEventID, objectID string) error {
	createdEvent, err := s.requireMissionEvent(ctx, missionID, createdEventID)
	if err != nil {
		return err
	}
	if isApprovalProducer(createdEvent.Producer) {
		return nil
	}
	proposalID, err := proposalIDFromCreatedEvent(createdEvent, objectID)
	if err != nil {
		return err
	}
	events, err := s.store.ListLedgerEvents(ctx, missionID)
	if err != nil {
		return err
	}
	for _, event := range events {
		if event.EventType != "proposal.approved" && event.EventType != "proposal.partially_approved" {
			continue
		}
		if !isApprovalProducer(event.Producer) {
			continue
		}
		var payload proposalDecisionPayload
		if err := unmarshalEventPayload(event, &payload); err != nil {
			return err
		}
		if strings.TrimSpace(payload.ProposalID) == proposalID && containsString(trimStringList(payload.ApprovedObjectIDs), objectID) {
			return nil
		}
	}
	return fmt.Errorf("%w: proposal object is not approved", ErrInvalidInput)
}

func proposalIDFromCreatedEvent(event LedgerEvent, objectID string) (string, error) {
	var payload struct {
		ProposalID string `json:"proposal_id"`
		EvidenceID string `json:"evidence_id"`
		ClaimID    string `json:"claim_id"`
		QuestionID string `json:"question_id"`
		OptionID   string `json:"option_id"`
	}
	if err := unmarshalEventPayload(event, &payload); err != nil {
		return "", err
	}
	if payload.EvidenceID != "" && strings.TrimSpace(payload.EvidenceID) != objectID {
		return "", fmt.Errorf("%w: event does not reference report object", ErrInvalidInput)
	}
	if payload.ClaimID != "" && strings.TrimSpace(payload.ClaimID) != objectID {
		return "", fmt.Errorf("%w: event does not reference report object", ErrInvalidInput)
	}
	if payload.QuestionID != "" && strings.TrimSpace(payload.QuestionID) != objectID {
		return "", fmt.Errorf("%w: event does not reference report object", ErrInvalidInput)
	}
	if payload.OptionID != "" && strings.TrimSpace(payload.OptionID) != objectID {
		return "", fmt.Errorf("%w: event does not reference report object", ErrInvalidInput)
	}
	proposalID := strings.TrimSpace(payload.ProposalID)
	if proposalID == "" {
		return "", fmt.Errorf("%w: created event does not reference a proposal", ErrInvalidInput)
	}
	return proposalID, nil
}

func buildDraftReportBlocks(
	versionID string,
	missionID string,
	title string,
	producer Producer,
	records reportScopeRecords,
) ([]ReportBlock, error) {
	blockID := func(label string, index int) string {
		suffix := strings.TrimPrefix(versionID, "rvn_")
		if index == 0 {
			return "blk_" + suffix + "_" + label
		}
		return fmt.Sprintf("blk_%s_%s_%03d", suffix, label, index)
	}
	childBlocks := []ReportBlock{}
	addBlock := func(blockType string, order int, content any, refs ReportBlockSourceRefs) error {
		id := blockID(blockType, len(childBlocks)+1)
		if err := validateID("blk_", id); err != nil {
			return err
		}
		encoded, err := json.Marshal(content)
		if err != nil {
			return err
		}
		childBlocks = append(childBlocks, ReportBlock{
			SchemaVersion:   ReportBlockSchemaVersion,
			ObjectKind:      ReportBlockObjectKind,
			BlockID:         id,
			ReportVersionID: versionID,
			MissionID:       missionID,
			BlockType:       blockType,
			ParentBlockID:   blockID("root", 0),
			Order:           order,
			Content:         encoded,
			SourceRefs:      refs,
			Authorship:      ReportBlockAuthorship{Mode: "generated", Producer: producer},
			Approval:        Approval{State: "pending", Required: true},
		})
		return nil
	}

	if err := addBlock("title", 10, map[string]string{"text": title}, ReportBlockSourceRefs{}); err != nil {
		return nil, err
	}
	summary := fmt.Sprintf("Draft generated from %d claim(s), %d evidence record(s), %d question(s), and %d option(s).",
		len(records.Claims), len(records.Evidence), len(records.Questions), len(records.Options))
	if err := addBlock("abstract", 20, map[string]string{"text": summary}, ReportBlockSourceRefs{}); err != nil {
		return nil, err
	}
	order := 30
	if len(records.Claims) > 0 {
		if err := addBlock("heading", order, map[string]any{"level": 2, "text": "Claims"}, ReportBlockSourceRefs{}); err != nil {
			return nil, err
		}
		order += 10
		for _, claim := range records.Claims {
			refs := refsForClaim(claim, records.Evidence)
			if len(refs.EvidenceIDs) == 0 {
				return nil, fmt.Errorf("%w: generated claim block requires evidence links", ErrInvalidInput)
			}
			if err := addBlock("claim", order, map[string]string{
				"claim_id":      claim.ClaimID,
				"rendered_text": claim.Text,
			}, refs); err != nil {
				return nil, err
			}
			order += 10
		}
	}
	if len(records.Evidence) > 0 {
		if err := addBlock("heading", order, map[string]any{"level": 2, "text": "Evidence"}, ReportBlockSourceRefs{}); err != nil {
			return nil, err
		}
		order += 10
		for _, evidence := range records.Evidence {
			if err := addBlock("evidence_summary", order, map[string]any{
				"evidence_ids": []string{evidence.EvidenceID},
				"text":         evidence.Summary,
			}, refsForEvidence(evidence)); err != nil {
				return nil, err
			}
			order += 10
		}
	}
	if len(records.Questions) > 0 {
		if err := addBlock("heading", order, map[string]any{"level": 2, "text": "Open Questions"}, ReportBlockSourceRefs{}); err != nil {
			return nil, err
		}
		order += 10
		for _, question := range records.Questions {
			if err := addBlock("unresolved_question", order, map[string]string{
				"question_id": question.QuestionID,
				"text":        question.Text,
			}, ReportBlockSourceRefs{
				QuestionIDs: []string{question.QuestionID},
				EvidenceIDs: append([]string(nil), question.RelatedEvidenceIDs...),
				ClaimIDs:    append([]string(nil), question.RelatedClaimIDs...),
			}); err != nil {
				return nil, err
			}
			order += 10
		}
	}
	if len(records.Options) > 0 {
		if err := addBlock("heading", order, map[string]any{"level": 2, "text": "Options"}, ReportBlockSourceRefs{}); err != nil {
			return nil, err
		}
		order += 10
		for _, option := range records.Options {
			if err := addBlock("option", order, map[string]string{
				"option_id": option.OptionID,
				"text":      option.Title,
			}, ReportBlockSourceRefs{
				OptionIDs: []string{option.OptionID},
				ClaimIDs:  append([]string(nil), option.SupportingClaimIDs...),
			}); err != nil {
				return nil, err
			}
			order += 10
		}
	}

	childIDs := make([]string, 0, len(childBlocks))
	for _, block := range childBlocks {
		childIDs = append(childIDs, block.BlockID)
	}
	rootID := blockID("root", 0)
	rootContent, err := json.Marshal(map[string]any{"children": childIDs})
	if err != nil {
		return nil, err
	}
	root := ReportBlock{
		SchemaVersion:   ReportBlockSchemaVersion,
		ObjectKind:      ReportBlockObjectKind,
		BlockID:         rootID,
		ReportVersionID: versionID,
		MissionID:       missionID,
		BlockType:       "document",
		Order:           0,
		Content:         rootContent,
		SourceRefs:      ReportBlockSourceRefs{},
		Authorship:      ReportBlockAuthorship{Mode: "generated", Producer: producer},
		Approval:        Approval{State: "pending", Required: true},
	}
	return append([]ReportBlock{root}, childBlocks...), nil
}

func buildReportBlocksFromDraftInputs(
	versionID string,
	missionID string,
	producer Producer,
	inputs []ReportBlockDraftInput,
) ([]ReportBlock, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("%w: report draft requires blocks", ErrInvalidInput)
	}
	blockID := func(label string, index int) string {
		suffix := strings.TrimPrefix(versionID, "rvn_")
		if index == 0 {
			return "blk_" + suffix + "_" + label
		}
		return fmt.Sprintf("blk_%s_%s_%03d", suffix, label, index)
	}
	childBlocks := make([]ReportBlock, 0, len(inputs))
	for index, input := range inputs {
		blockType := strings.TrimSpace(input.BlockType)
		if !allowedReportBlockTypes[blockType] || blockType == "document" {
			return nil, fmt.Errorf("%w: unsupported report block type", ErrInvalidInput)
		}
		content := append(json.RawMessage(nil), input.Content...)
		if len(content) == 0 {
			content = json.RawMessage(`{}`)
		}
		block := ReportBlock{
			SchemaVersion:   ReportBlockSchemaVersion,
			ObjectKind:      ReportBlockObjectKind,
			BlockID:         blockID(blockType, index+1),
			ReportVersionID: versionID,
			MissionID:       missionID,
			BlockType:       blockType,
			ParentBlockID:   blockID("root", 0),
			Order:           (index + 1) * 10,
			Content:         content,
			SourceRefs:      input.SourceRefs,
			Authorship:      ReportBlockAuthorship{Mode: "generated", Producer: producer},
			Approval:        Approval{State: "pending", Required: true},
		}
		if err := validateReportBlock(block); err != nil {
			return nil, err
		}
		childBlocks = append(childBlocks, block)
	}

	childIDs := make([]string, 0, len(childBlocks))
	for _, block := range childBlocks {
		childIDs = append(childIDs, block.BlockID)
	}
	rootID := blockID("root", 0)
	rootContent, err := json.Marshal(map[string]any{"children": childIDs})
	if err != nil {
		return nil, err
	}
	root := ReportBlock{
		SchemaVersion:   ReportBlockSchemaVersion,
		ObjectKind:      ReportBlockObjectKind,
		BlockID:         rootID,
		ReportVersionID: versionID,
		MissionID:       missionID,
		BlockType:       "document",
		Order:           0,
		Content:         rootContent,
		SourceRefs:      ReportBlockSourceRefs{},
		Authorship:      ReportBlockAuthorship{Mode: "generated", Producer: producer},
		Approval:        Approval{State: "pending", Required: true},
	}
	return append([]ReportBlock{root}, childBlocks...), nil
}

func refsForClaim(claim ClaimRecord, evidence []EvidenceRecord) ReportBlockSourceRefs {
	refs := ReportBlockSourceRefs{ClaimIDs: []string{claim.ClaimID}}
	evidenceIDs := append(append([]string{}, claim.SupportingEvidenceIDs...), claim.OpposingEvidenceIDs...)
	for _, evidenceID := range evidenceIDs {
		addUnique(&refs.EvidenceIDs, evidenceID)
		for _, record := range evidence {
			if record.EvidenceID == evidenceID {
				for _, snapshotRef := range record.SnapshotRefs {
					addUnique(&refs.SnapshotIDs, snapshotRef.SnapshotID)
				}
			}
		}
	}
	return refs
}

func refsForEvidence(evidence EvidenceRecord) ReportBlockSourceRefs {
	refs := ReportBlockSourceRefs{EvidenceIDs: []string{evidence.EvidenceID}}
	for _, snapshotRef := range evidence.SnapshotRefs {
		addUnique(&refs.SnapshotIDs, snapshotRef.SnapshotID)
	}
	return refs
}

func renderReportExport(version ReportVersion, blocks []ReportBlock, target string) ([]byte, string, string, error) {
	switch target {
	case ReportExportTargetJSONAST:
		export := ReportASTExport{
			SchemaVersion: "plasma.report_ast_export.v1",
			ObjectKind:    "report_ast_export",
			Version:       version,
			Blocks:        blocks,
		}
		content, err := json.MarshalIndent(export, "", "  ")
		if err != nil {
			return nil, "", "", err
		}
		content = append(content, '\n')
		return content, "application/json", version.ReportVersionID + ".json", nil
	case ReportExportTargetMarkdown:
		content, err := renderReportMarkdown(blocks)
		if err != nil {
			return nil, "", "", err
		}
		return content, "text/markdown", version.ReportVersionID + ".md", nil
	case ReportExportTargetHTML:
		content, err := renderReportHTML(blocks)
		if err != nil {
			return nil, "", "", err
		}
		return content, "text/html; charset=utf-8", version.ReportVersionID + ".html", nil
	default:
		return nil, "", "", fmt.Errorf("%w: unsupported report export target", ErrInvalidInput)
	}
}

func renderReportMarkdown(blocks []ReportBlock) ([]byte, error) {
	var out bytes.Buffer
	footnotes := newReportFootnotes()
	for _, block := range blocks {
		switch block.BlockType {
		case "document":
			continue
		case "title":
			text, err := blockText(block)
			if err != nil {
				return nil, err
			}
			out.WriteString("# " + text + "\n\n")
		case "abstract", "paragraph":
			text, err := blockText(block)
			if err != nil {
				return nil, err
			}
			out.WriteString(text + footnotes.markdownMarker(block.SourceRefs) + "\n\n")
		case "heading":
			heading, err := blockHeading(block)
			if err != nil {
				return nil, err
			}
			out.WriteString(strings.Repeat("#", heading.Level) + " " + heading.Text + footnotes.markdownMarker(block.SourceRefs) + "\n\n")
		case "claim":
			var content struct {
				ClaimID      string `json:"claim_id"`
				RenderedText string `json:"rendered_text"`
			}
			if err := json.Unmarshal(block.Content, &content); err != nil {
				return nil, err
			}
			claimID := strings.TrimSpace(content.ClaimID)
			text := strings.TrimSpace(content.RenderedText)
			if claimID == "" || text == "" {
				return nil, fmt.Errorf("%w: claim block requires claim id and rendered text", ErrInvalidInput)
			}
			refs := block.SourceRefs
			if len(reportRefValues(refs)) == 0 {
				refs.ClaimIDs = []string{claimID}
			}
			out.WriteString("- " + text + footnotes.markdownMarker(refs) + "\n")
		case "evidence_summary":
			text, err := blockText(block)
			if err != nil {
				return nil, err
			}
			out.WriteString("- " + text + footnotes.markdownMarker(block.SourceRefs) + "\n")
		case "bullet_list":
			items, err := blockListItems(block)
			if err != nil {
				return nil, err
			}
			for index, item := range items {
				if index == len(items)-1 {
					item += footnotes.markdownMarker(block.SourceRefs)
				}
				out.WriteString("- " + item + "\n")
			}
		case "quote":
			text, err := blockText(block)
			if err != nil {
				return nil, err
			}
			lines := strings.Split(text, "\n")
			for index, line := range lines {
				if index == len(lines)-1 {
					line += footnotes.markdownMarker(block.SourceRefs)
				}
				out.WriteString("> " + line + "\n")
			}
		case "unresolved_question":
			text, err := blockText(block)
			if err != nil {
				return nil, err
			}
			out.WriteString("- " + text + footnotes.markdownMarker(block.SourceRefs) + "\n")
		case "option":
			text, err := blockText(block)
			if err != nil {
				return nil, err
			}
			out.WriteString("- " + text + footnotes.markdownMarker(block.SourceRefs) + "\n")
		default:
			return nil, fmt.Errorf("%w: unsupported report block type", ErrInvalidInput)
		}
		if block.BlockType == "claim" || block.BlockType == "evidence_summary" || block.BlockType == "bullet_list" || block.BlockType == "quote" || block.BlockType == "unresolved_question" || block.BlockType == "option" {
			out.WriteString("\n")
		}
	}
	footnotes.writeMarkdownDefinitions(&out)
	return bytes.TrimSpace(out.Bytes()), nil
}

func renderReportHTML(blocks []ReportBlock) ([]byte, error) {
	var out bytes.Buffer
	footnotes := newReportFootnotes()
	out.WriteString("<!doctype html>\n<html lang=\"ko\">\n<head>\n<meta charset=\"utf-8\">\n")
	out.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	out.WriteString("<title>Plasma Report</title>\n")
	out.WriteString("<style>body{margin:0;font:16px/1.65 -apple-system,BlinkMacSystemFont,\"Segoe UI\",sans-serif;color:#1f2937;background:#f8fafc}main{max-width:880px;margin:0 auto;padding:48px 24px 72px;background:#fff;min-height:100vh}h1{font-size:2rem;line-height:1.2;margin:0 0 24px}h2{font-size:1.35rem;margin:36px 0 12px}h3{font-size:1.1rem;margin:28px 0 10px}p{margin:0 0 16px}.lead{font-size:1.08rem;color:#475569;border-left:4px solid #0f766e;padding-left:14px}blockquote{margin:20px 0;padding:12px 16px;background:#f1f5f9;border-left:4px solid #64748b}ul{padding-left:1.4rem}.footnote-refs{font-size:.78em;vertical-align:super;margin-left:2px}.footnote-refs a{color:#0f766e;text-decoration:none}.footnotes{margin-top:44px;padding-top:18px;border-top:1px solid #e2e8f0;color:#475569;font-size:.9rem}.footnotes h2{font-size:1rem;margin:0 0 10px}.footnotes li{margin:4px 0;word-break:break-all}</style>\n")
	out.WriteString("</head>\n<body>\n<main>\n")
	for _, block := range blocks {
		switch block.BlockType {
		case "document":
			continue
		case "title":
			text, err := blockText(block)
			if err != nil {
				return nil, err
			}
			out.WriteString("<h1>" + htmlpkg.EscapeString(text) + "</h1>\n")
		case "abstract":
			text, err := blockText(block)
			if err != nil {
				return nil, err
			}
			out.WriteString("<p class=\"lead\">" + htmlpkg.EscapeString(text) + footnotes.htmlMarker(block.SourceRefs) + "</p>\n")
		case "paragraph":
			text, err := blockText(block)
			if err != nil {
				return nil, err
			}
			out.WriteString("<p>" + htmlpkg.EscapeString(text) + footnotes.htmlMarker(block.SourceRefs) + "</p>\n")
		case "heading":
			heading, err := blockHeading(block)
			if err != nil {
				return nil, err
			}
			level := heading.Level
			if level < 2 {
				level = 2
			}
			if level > 4 {
				level = 4
			}
			tag := fmt.Sprintf("h%d", level)
			out.WriteString("<" + tag + ">" + htmlpkg.EscapeString(heading.Text) + footnotes.htmlMarker(block.SourceRefs) + "</" + tag + ">\n")
		case "bullet_list":
			items, err := blockListItems(block)
			if err != nil {
				return nil, err
			}
			out.WriteString("<ul>\n")
			for index, item := range items {
				marker := ""
				if index == len(items)-1 {
					marker = footnotes.htmlMarker(block.SourceRefs)
				}
				out.WriteString("<li>" + htmlpkg.EscapeString(item) + marker + "</li>\n")
			}
			out.WriteString("</ul>\n")
		case "quote":
			text, err := blockText(block)
			if err != nil {
				return nil, err
			}
			out.WriteString("<blockquote>" + htmlpkg.EscapeString(text) + footnotes.htmlMarker(block.SourceRefs) + "</blockquote>\n")
		case "claim":
			var content struct {
				RenderedText string `json:"rendered_text"`
			}
			if err := json.Unmarshal(block.Content, &content); err != nil {
				return nil, err
			}
			out.WriteString("<p>" + htmlpkg.EscapeString(strings.TrimSpace(content.RenderedText)) + footnotes.htmlMarker(block.SourceRefs) + "</p>\n")
		case "evidence_summary":
			text, err := blockText(block)
			if err != nil {
				return nil, err
			}
			out.WriteString("<p>" + htmlpkg.EscapeString(text) + footnotes.htmlMarker(block.SourceRefs) + "</p>\n")
		case "unresolved_question", "option":
			text, err := blockText(block)
			if err != nil {
				return nil, err
			}
			out.WriteString("<p>" + htmlpkg.EscapeString(text) + footnotes.htmlMarker(block.SourceRefs) + "</p>\n")
		default:
			return nil, fmt.Errorf("%w: unsupported report block type", ErrInvalidInput)
		}
	}
	footnotes.writeHTMLDefinitions(&out)
	out.WriteString("</main>\n</body>\n</html>\n")
	return out.Bytes(), nil
}

func reportRefValues(refs ReportBlockSourceRefs) []string {
	values := []string{}
	seen := map[string]struct{}{}
	appendValue := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	for _, value := range refs.ClaimIDs {
		appendValue(value)
	}
	for _, value := range refs.EvidenceIDs {
		appendValue(value)
	}
	for _, value := range refs.SnapshotIDs {
		appendValue(value)
	}
	for _, value := range refs.QuestionIDs {
		appendValue(value)
	}
	for _, value := range refs.OptionIDs {
		appendValue(value)
	}
	return values
}

type reportFootnotes struct {
	index  map[string]int
	values []string
}

func newReportFootnotes() *reportFootnotes {
	return &reportFootnotes{index: map[string]int{}}
}

func (footnotes *reportFootnotes) numbers(refs ReportBlockSourceRefs) []int {
	values := reportRefValues(refs)
	numbers := make([]int, 0, len(values))
	for _, value := range values {
		number, ok := footnotes.index[value]
		if !ok {
			footnotes.values = append(footnotes.values, value)
			number = len(footnotes.values)
			footnotes.index[value] = number
		}
		numbers = append(numbers, number)
	}
	return numbers
}

func (footnotes *reportFootnotes) markdownMarker(refs ReportBlockSourceRefs) string {
	numbers := footnotes.numbers(refs)
	if len(numbers) == 0 {
		return ""
	}
	parts := make([]string, 0, len(numbers))
	for _, number := range numbers {
		parts = append(parts, fmt.Sprintf("[^%d]", number))
	}
	return " " + strings.Join(parts, " ")
}

func (footnotes *reportFootnotes) htmlMarker(refs ReportBlockSourceRefs) string {
	numbers := footnotes.numbers(refs)
	if len(numbers) == 0 {
		return ""
	}
	var out bytes.Buffer
	out.WriteString("<sup class=\"footnote-refs\">")
	for _, number := range numbers {
		out.WriteString(fmt.Sprintf("<a href=\"#fn-%d\" id=\"fnref-%d\">[%d]</a>", number, number, number))
	}
	out.WriteString("</sup>")
	return out.String()
}

func (footnotes *reportFootnotes) writeMarkdownDefinitions(out *bytes.Buffer) {
	if len(footnotes.values) == 0 {
		return
	}
	out.WriteString("\n## 각주\n\n")
	for index, value := range footnotes.values {
		out.WriteString(fmt.Sprintf("[^%d]: `%s`\n", index+1, value))
	}
}

func (footnotes *reportFootnotes) writeHTMLDefinitions(out *bytes.Buffer) {
	if len(footnotes.values) == 0 {
		return
	}
	out.WriteString("<section class=\"footnotes\"><h2>각주</h2><ol>\n")
	for index, value := range footnotes.values {
		number := index + 1
		out.WriteString(fmt.Sprintf("<li id=\"fn-%d\"><code>%s</code> <a href=\"#fnref-%d\" aria-label=\"본문으로 돌아가기\">back</a></li>\n", number, htmlpkg.EscapeString(value), number))
	}
	out.WriteString("</ol></section>\n")
}

func validateReportBlockRefs(refs ReportBlockSourceRefs, records reportScopeRecords) error {
	claimIDs := map[string]struct{}{}
	for _, claim := range records.Claims {
		claimIDs[claim.ClaimID] = struct{}{}
	}
	evidenceIDs := map[string]struct{}{}
	snapshotIDs := map[string]struct{}{}
	for _, evidence := range records.Evidence {
		evidenceIDs[evidence.EvidenceID] = struct{}{}
		for _, snapshotRef := range evidence.SnapshotRefs {
			snapshotIDs[snapshotRef.SnapshotID] = struct{}{}
		}
	}
	questionIDs := map[string]struct{}{}
	for _, question := range records.Questions {
		questionIDs[question.QuestionID] = struct{}{}
	}
	optionIDs := map[string]struct{}{}
	for _, option := range records.Options {
		optionIDs[option.OptionID] = struct{}{}
	}
	if err := requireRefsInScope("claim", refs.ClaimIDs, claimIDs); err != nil {
		return err
	}
	if err := requireRefsInScope("evidence", refs.EvidenceIDs, evidenceIDs); err != nil {
		return err
	}
	if err := requireRefsInScope("source snapshot", refs.SnapshotIDs, snapshotIDs); err != nil {
		return err
	}
	if err := requireRefsInScope("question", refs.QuestionIDs, questionIDs); err != nil {
		return err
	}
	if err := requireRefsInScope("option", refs.OptionIDs, optionIDs); err != nil {
		return err
	}
	return nil
}

func requireRefsInScope(kind string, refs []string, allowed map[string]struct{}) error {
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		if _, ok := allowed[ref]; !ok {
			return fmt.Errorf("%w: report block references out-of-scope %s %q", ErrInvalidInput, kind, ref)
		}
	}
	return nil
}

func validateReportBlock(block ReportBlock) error {
	switch block.BlockType {
	case "document":
		return nil
	case "title", "abstract", "paragraph", "quote", "evidence_summary", "unresolved_question", "option":
		_, err := blockText(block)
		return err
	case "heading":
		_, err := blockHeading(block)
		return err
	case "bullet_list":
		_, err := blockListItems(block)
		return err
	case "claim":
		var content struct {
			ClaimID      string `json:"claim_id"`
			RenderedText string `json:"rendered_text"`
		}
		if err := json.Unmarshal(block.Content, &content); err != nil {
			return err
		}
		if strings.TrimSpace(content.RenderedText) == "" {
			return fmt.Errorf("%w: claim block requires rendered text", ErrInvalidInput)
		}
		return nil
	default:
		return fmt.Errorf("%w: unsupported report block type", ErrInvalidInput)
	}
}

func blockText(block ReportBlock) (string, error) {
	var content struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(block.Content, &content); err != nil {
		return "", err
	}
	text := strings.TrimSpace(content.Text)
	if text == "" {
		return "", fmt.Errorf("%w: report block text is required", ErrInvalidInput)
	}
	return text, nil
}

func blockListItems(block ReportBlock) ([]string, error) {
	var content struct {
		Items []string `json:"items"`
	}
	if err := json.Unmarshal(block.Content, &content); err != nil {
		return nil, err
	}
	items := make([]string, 0, len(content.Items))
	for _, item := range content.Items {
		item = strings.TrimSpace(item)
		if item != "" {
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("%w: report list items are required", ErrInvalidInput)
	}
	return items, nil
}

type reportHeading struct {
	Level int
	Text  string
}

func blockHeading(block ReportBlock) (reportHeading, error) {
	var content reportHeading
	if err := json.Unmarshal(block.Content, &content); err != nil {
		return reportHeading{}, err
	}
	if content.Level <= 0 {
		content.Level = 2
	}
	if content.Level > 6 {
		content.Level = 6
	}
	content.Text = strings.TrimSpace(content.Text)
	if content.Text == "" {
		return reportHeading{}, fmt.Errorf("%w: report heading text is required", ErrInvalidInput)
	}
	return content, nil
}

func requireReportPromotionEvent(event LedgerEvent, reportVersionID string) error {
	if !isApprovalProducer(event.Producer) {
		return fmt.Errorf("%w: report promotion requires user or steering_chat event", ErrInvalidInput)
	}
	if event.EventType != "report.promoted" {
		return fmt.Errorf("%w: report promotion requires report.promoted event", ErrInvalidInput)
	}
	var payload struct {
		ReportVersionID string `json:"report_version_id"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("%w: report.promoted payload is invalid", ErrInvalidInput)
	}
	if strings.TrimSpace(payload.ReportVersionID) != reportVersionID {
		return fmt.Errorf("%w: report.promoted event does not reference report version", ErrInvalidInput)
	}
	return nil
}

var allowedReportFormatIntents = map[string]bool{
	"briefing":    true,
	"full_report": true,
	"outline":     true,
}

var allowedReportExportTargets = map[string]bool{
	ReportExportTargetMarkdown: true,
	ReportExportTargetJSONAST:  true,
	ReportExportTargetHTML:     true,
}

var allowedReportBlockTypes = map[string]bool{
	"document":            true,
	"title":               true,
	"abstract":            true,
	"heading":             true,
	"paragraph":           true,
	"bullet_list":         true,
	"quote":               true,
	"claim":               true,
	"evidence_summary":    true,
	"unresolved_question": true,
	"option":              true,
}
