package app

import (
	"context"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/reporting/reportdocument"
	"strings"
	"time"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/researchproposal"
)

// ReportStore는 report와 version 상태 전이를 저장하는 포트다.
type ReportStore interface {
	CreateReport(context.Context, reportdocument.Report) error
	GetReport(context.Context, string) (reportdocument.Report, error)
	CreateReportVersion(context.Context, reportdocument.ReportVersion, []reportdocument.ReportBlock) error
	GetReportVersion(context.Context, string) (reportdocument.ReportVersion, error)
	ListReportBlocks(context.Context, string) ([]reportdocument.ReportBlock, error)
	PromoteReportVersion(context.Context, reportdocument.ReportVersionPromotion) error
}

// ReportListStore는 미션별 report 목록 조회 전용 저장소 포트다.
type ReportListStore interface {
	ListReports(context.Context, string) ([]reportdocument.Report, error)
	ListReportVersions(context.Context, string) ([]reportdocument.ReportVersion, error)
}

// CreateReportDraft는 초안 artifact를 report의 새 version으로 저장한다.
func (s *Service) CreateReportDraft(ctx context.Context, req reportdocument.CreateReportDraftRequest) (reportdocument.ReportDraftResult, error) {
	event, report, version, blocks, err := s.buildReportDraft(ctx, req)
	if err != nil {
		return reportdocument.ReportDraftResult{}, err
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:         []ledger.Event{event},
		Reports:        []reportdocument.Report{report},
		ReportVersions: []reportdocument.ReportVersion{version},
		ReportBlocks:   blocks,
	})
	if err != nil {
		return reportdocument.ReportDraftResult{}, err
	}
	return reportdocument.ReportDraftResult{
		Report:  report,
		Version: version,
		Blocks:  blocks,
		Event:   committed.Events[0],
	}, nil
}

func (s *Service) buildReportDraft(
	ctx context.Context,
	req reportdocument.CreateReportDraftRequest,
) (ledger.Event, reportdocument.Report, reportdocument.ReportVersion, []reportdocument.ReportBlock, error) {
	reportID := strings.TrimSpace(req.ReportID)
	versionID := strings.TrimSpace(req.ReportVersionID)
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("rpt_", reportID); err != nil {
		return ledger.Event{}, reportdocument.Report{}, reportdocument.ReportVersion{}, nil, err
	}
	if err := validateID("rvn_", versionID); err != nil {
		return ledger.Event{}, reportdocument.Report{}, reportdocument.ReportVersion{}, nil, err
	}
	if err := validateID("mis_", missionID); err != nil {
		return ledger.Event{}, reportdocument.Report{}, reportdocument.ReportVersion{}, nil, err
	}
	if err := validateProducer(req.Producer); err != nil {
		return ledger.Event{}, reportdocument.Report{}, reportdocument.ReportVersion{}, nil, err
	}
	baseVersionID := strings.TrimSpace(req.BaseVersionID)
	if baseVersionID != "" {
		return ledger.Event{}, reportdocument.Report{}, reportdocument.ReportVersion{}, nil, fmt.Errorf("%w: report revisions are not implemented yet", ErrInvalidInput)
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "Research report"
	}
	formatIntent := strings.TrimSpace(req.FormatIntent)
	if formatIntent == "" {
		formatIntent = "briefing"
	}
	if !reportdocument.FormatIntentAllowed(formatIntent) {
		return ledger.Event{}, reportdocument.Report{}, reportdocument.ReportVersion{}, nil, fmt.Errorf("%w: unsupported report format intent", ErrInvalidInput)
	}

	scope, err := s.normalizeReportScope(ctx, missionID, req.Scope)
	if err != nil {
		return ledger.Event{}, reportdocument.Report{}, reportdocument.ReportVersion{}, nil, err
	}
	records, err := s.resolveReportScope(ctx, missionID, scope)
	if err != nil {
		return ledger.Event{}, reportdocument.Report{}, reportdocument.ReportVersion{}, nil, err
	}
	if len(records.Claims)+len(records.Evidence)+len(records.Questions)+len(records.Options) == 0 {
		return ledger.Event{}, reportdocument.Report{}, reportdocument.ReportVersion{}, nil, fmt.Errorf("%w: report draft requires scoped records", ErrInvalidInput)
	}

	event, err := buildLedgerEvent(ledger.AppendRequest{
		EventID:   req.CreatedEventID,
		MissionID: missionID,
		EventType: "report.drafted",
		Producer:  normalizeProducer(req.Producer),
		Payload:   mustMarshalJSON(reportDraftPayload(reportID, versionID, formatIntent, scope, req.Generation)),
	})
	if err != nil {
		return ledger.Event{}, reportdocument.Report{}, reportdocument.ReportVersion{}, nil, err
	}

	now := time.Now().UTC()
	var blocks []reportdocument.ReportBlock
	if len(req.Blocks) > 0 {
		for _, block := range req.Blocks {
			if err := reportdocument.ValidateReportBlockRefs(block.SourceRefs, records); err != nil {
				return ledger.Event{}, reportdocument.Report{}, reportdocument.ReportVersion{}, nil, err
			}
		}
		blocks, err = reportdocument.BuildReportBlocksFromDraftInputs(versionID, missionID, normalizeProducer(req.Producer), req.Blocks)
	} else {
		blocks, err = reportdocument.BuildDraftReportBlocks(versionID, missionID, title, normalizeProducer(req.Producer), records)
	}
	if err != nil {
		return ledger.Event{}, reportdocument.Report{}, reportdocument.ReportVersion{}, nil, err
	}
	blockIDs := make([]string, 0, len(blocks))
	for _, block := range blocks {
		blockIDs = append(blockIDs, block.BlockID)
	}
	report := reportdocument.Report{
		SchemaVersion:   reportdocument.ReportSchemaVersion,
		ObjectKind:      reportdocument.ReportObjectKind,
		ReportID:        reportID,
		MissionID:       missionID,
		Title:           title,
		ActiveVersionID: versionID,
		State:           "draft",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	version := reportdocument.ReportVersion{
		SchemaVersion:         reportdocument.ReportVersionSchemaVersion,
		ObjectKind:            reportdocument.ReportVersionObjectKind,
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

func reportDraftPayload(reportID string, versionID string, formatIntent string, scope reportdocument.ReportEvidenceScope, generation map[string]any) map[string]any {
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

// GetReport는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) GetReport(ctx context.Context, reportID string) (reportdocument.Report, error) {
	trimmed := strings.TrimSpace(reportID)
	if err := validateID("rpt_", trimmed); err != nil {
		return reportdocument.Report{}, err
	}
	return s.store.GetReport(ctx, trimmed)
}

// ListReports는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) ListReports(ctx context.Context, missionID string) ([]reportdocument.Report, error) {
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

// GetReportVersion는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) GetReportVersion(ctx context.Context, versionID string) (reportdocument.ReportVersion, error) {
	trimmed := strings.TrimSpace(versionID)
	if err := validateID("rvn_", trimmed); err != nil {
		return reportdocument.ReportVersion{}, err
	}
	return s.store.GetReportVersion(ctx, trimmed)
}

// ListReportVersions는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) ListReportVersions(ctx context.Context, missionID string) ([]reportdocument.ReportVersion, error) {
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

// ListReportBlocks는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) ListReportBlocks(ctx context.Context, versionID string) ([]reportdocument.ReportBlock, error) {
	trimmed := strings.TrimSpace(versionID)
	if err := validateID("rvn_", trimmed); err != nil {
		return nil, err
	}
	return s.store.ListReportBlocks(ctx, trimmed)
}

// PromoteReportVersion는 선택한 report version을 현재 version으로 승격한다.
func (s *Service) PromoteReportVersion(ctx context.Context, req reportdocument.PromoteReportVersionRequest) (reportdocument.ReportVersion, error) {
	versionID := strings.TrimSpace(req.ReportVersionID)
	if err := validateID("rvn_", versionID); err != nil {
		return reportdocument.ReportVersion{}, err
	}
	version, err := s.store.GetReportVersion(ctx, versionID)
	if err != nil {
		return reportdocument.ReportVersion{}, err
	}
	if version.State != "draft" && version.State != "review" {
		return reportdocument.ReportVersion{}, fmt.Errorf("%w: report version cannot be promoted from current state", ErrInvalidInput)
	}
	event, err := s.requireMissionEvent(ctx, version.MissionID, req.ApprovalEventID)
	if err != nil {
		return reportdocument.ReportVersion{}, err
	}
	if err := reportdocument.RequirePromotionEvent(event, version.ReportVersionID); err != nil {
		return reportdocument.ReportVersion{}, err
	}
	update := reportdocument.ReportVersionPromotion{
		ReportID:        version.ReportID,
		ReportVersionID: version.ReportVersionID,
		FromState:       version.State,
		ToState:         "export_candidate",
		ReportState:     "export_candidate",
		ApprovalEventID: event.EventID,
		UpdatedAt:       time.Now().UTC(),
	}
	if err := s.store.PromoteReportVersion(ctx, update); err != nil {
		return reportdocument.ReportVersion{}, err
	}
	return s.store.GetReportVersion(ctx, versionID)
}

// BuildReportPromotionAppendRequest는 애플리케이션 서비스 계층에서 장부에 기록할 append 요청을 조립한다. 실제 저장과 조건부 append 결정은 호출자가 소유한다.
func BuildReportPromotionAppendRequest(req reportdocument.ReportPromotionAppendRequest) ledger.AppendRequest {
	version := req.Version
	return ledger.AppendRequest{
		EventID:   strings.TrimSpace(req.EventID),
		MissionID: strings.TrimSpace(version.MissionID),
		EventType: "report.promoted",
		Producer:  req.Producer,
		Payload: mustMarshalJSON(map[string]any{
			"report_version_id": version.ReportVersionID,
		}),
	}
}

// ExportReportVersion는 애플리케이션 서비스 계층 결과를 외부 artifact 형태로 내보낸다. 원본 report version은 변경하지 않는다.
func (s *Service) ExportReportVersion(ctx context.Context, req reportdocument.ExportReportVersionRequest) (reportdocument.ReportExportResult, error) {
	exportID := strings.TrimSpace(req.ExportID)
	versionID := strings.TrimSpace(req.ReportVersionID)
	artifactID := strings.TrimSpace(req.ArtifactID)
	eventID := strings.TrimSpace(req.EventID)
	if err := validateID("exp_", exportID); err != nil {
		return reportdocument.ReportExportResult{}, err
	}
	if err := validateID("rvn_", versionID); err != nil {
		return reportdocument.ReportExportResult{}, err
	}
	if err := validateID("art_", artifactID); err != nil {
		return reportdocument.ReportExportResult{}, err
	}
	if err := validateID("evt_", eventID); err != nil {
		return reportdocument.ReportExportResult{}, err
	}
	if err := validateProducer(req.Producer); err != nil {
		return reportdocument.ReportExportResult{}, err
	}
	producer := normalizeProducer(req.Producer)
	if !isApprovalProducer(producer) {
		return reportdocument.ReportExportResult{}, fmt.Errorf("%w: report export requires user or steering_chat producer", ErrInvalidInput)
	}
	target := strings.TrimSpace(req.Target)
	if !reportdocument.ExportTargetAllowed(target) {
		return reportdocument.ReportExportResult{}, fmt.Errorf("%w: unsupported report export target", ErrInvalidInput)
	}

	version, err := s.store.GetReportVersion(ctx, versionID)
	if err != nil {
		return reportdocument.ReportExportResult{}, err
	}
	if version.State != "export_candidate" {
		return reportdocument.ReportExportResult{}, fmt.Errorf("%w: report version must be an export_candidate", ErrInvalidInput)
	}
	approvalEvent, err := s.requireMissionEvent(ctx, version.MissionID, req.ApprovalEventID)
	if err != nil {
		return reportdocument.ReportExportResult{}, err
	}
	if err := reportdocument.RequirePromotionEvent(approvalEvent, version.ReportVersionID); err != nil {
		return reportdocument.ReportExportResult{}, err
	}
	blocks, err := s.store.ListReportBlocks(ctx, versionID)
	if err != nil {
		return reportdocument.ReportExportResult{}, err
	}
	content, mediaType, filename, err := reportdocument.RenderReportExport(version, blocks, target)
	if err != nil {
		return reportdocument.ReportExportResult{}, err
	}
	artifact, err := artifactcontract.Build(artifactcontract.CreateRequest{
		ArtifactID: artifactID,
		MissionID:  version.MissionID,
		MediaType:  mediaType,
		Filename:   filename,
		Producer:   producer,
		Content:    content,
	})
	if err != nil {
		return reportdocument.ReportExportResult{}, err
	}
	event, err := buildLedgerEvent(ledger.AppendRequest{
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
		return reportdocument.ReportExportResult{}, err
	}
	committed, err := s.commitAtomicWrite(ctx, AtomicWrite{
		Events:       []ledger.Event{event},
		RawArtifacts: []artifactcontract.Raw{artifact},
	})
	if err != nil {
		return reportdocument.ReportExportResult{}, err
	}
	return reportdocument.ReportExportResult{Artifact: artifact, Event: committed.Events[0]}, nil
}

// ReportAST는 저장된 report artifact를 구조화된 Markdown AST로 파싱한다.
func (s *Service) ReportAST(ctx context.Context, versionID string) (reportdocument.ReportASTExport, error) {
	version, err := s.GetReportVersion(ctx, versionID)
	if err != nil {
		return reportdocument.ReportASTExport{}, err
	}
	blocks, err := s.ListReportBlocks(ctx, version.ReportVersionID)
	if err != nil {
		return reportdocument.ReportASTExport{}, err
	}
	return reportdocument.ReportASTExport{
		SchemaVersion: "plasma.report_ast_export.v1",
		ObjectKind:    "report_ast_export",
		Version:       version,
		Blocks:        blocks,
	}, nil
}

func (s *Service) normalizeReportScope(ctx context.Context, missionID string, scope reportdocument.ReportEvidenceScope) (reportdocument.ReportEvidenceScope, error) {
	var err error
	scope, err = reportdocument.NormalizeScope(scope)
	if err != nil {
		return reportdocument.ReportEvidenceScope{}, err
	}
	if len(scope.EvidenceIDs)+len(scope.ClaimIDs)+len(scope.QuestionIDs)+len(scope.OptionIDs) == 0 {
		projection, err := s.GetProjection(ctx, missionID)
		if err != nil {
			return reportdocument.ReportEvidenceScope{}, err
		}
		scope.AcceptedOnly = true
		scope.IncludeProposed = false
		scope.ClaimIDs = append([]string(nil), projection.AcceptedClaimIDs...)
	}
	return scope, nil
}

func (s *Service) resolveReportScope(ctx context.Context, missionID string, scope reportdocument.ReportEvidenceScope) (reportdocument.ScopeRecords, error) {
	return reportdocument.ResolveScope(ctx, reportdocument.ScopeReaders{GetEvidenceRecord: s.store.GetEvidenceRecord, GetClaimRecord: s.store.GetClaimRecord, GetQuestionRecord: s.store.GetQuestionRecord, GetOptionRecord: s.store.GetOptionRecord, RequireApprovedObject: s.requireApprovedProposalObject}, missionID, scope)
}
func (s *Service) requireApprovedProposalObject(ctx context.Context, missionID, createdEventID, objectID string) error {
	return researchproposal.RequireApprovedObject(ctx, researchproposal.ApprovalReaders{RequireMissionEvent: s.requireMissionEvent, ListMissionEvents: s.store.ListLedgerEvents}, missionID, createdEventID, objectID)
}
