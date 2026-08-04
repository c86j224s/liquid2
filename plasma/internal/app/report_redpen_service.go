package app

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"
)

// GetReportRedpenWorkcopy는 애플리케이션 서비스 계층의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (s *Service) GetReportRedpenWorkcopy(ctx context.Context, missionID, sourceArtifactID string) (ReportRedpenWorkcopy, error) {
	source, events, err := s.reportRedpenSource(ctx, missionID, sourceArtifactID)
	if err != nil {
		return ReportRedpenWorkcopy{}, err
	}
	payload, event, exists, err := latestReportRedpenEvent(events, source.ArtifactID)
	if err != nil || !exists {
		return ReportRedpenWorkcopy{SourceArtifactID: source.ArtifactID}, err
	}
	artifact, err := s.GetRawArtifact(ctx, payload.ArtifactID)
	if err != nil {
		return ReportRedpenWorkcopy{}, err
	}
	if artifact.MissionID != missionID || !isMarkdownArtifactMediaType(artifact.MediaType) || artifact.SHA256 != payload.SHA256 {
		return ReportRedpenWorkcopy{}, fmt.Errorf("%w: redpen workcopy artifact is inconsistent", ErrInvalidInput)
	}
	return reportRedpenWorkcopy(payload, artifact, event, false), nil
}

// SaveReportRedpenWorkcopy는 redpen 작업본을 최신 workcopy로 저장한다.
func (s *Service) SaveReportRedpenWorkcopy(ctx context.Context, req SaveReportRedpenRequest) (ReportRedpenWorkcopy, error) {
	store, ok := s.store.(reportRedpenRevisionStore)
	if !ok {
		return ReportRedpenWorkcopy{}, fmt.Errorf("%w: report redpen revision store is required", ErrInvalidInput)
	}
	source, _, err := s.reportRedpenSource(ctx, req.MissionID, req.SourceArtifactID)
	if err != nil {
		return ReportRedpenWorkcopy{}, err
	}
	if err := validateID("rwc_", req.NewWorkcopyID); err != nil {
		return ReportRedpenWorkcopy{}, err
	}
	if expected := strings.TrimSpace(req.ExpectedCurrentArtifactID); expected != "" {
		if err := validateID("art_", expected); err != nil {
			return ReportRedpenWorkcopy{}, err
		}
	}
	if !utf8.Valid(req.Content) {
		return ReportRedpenWorkcopy{}, fmt.Errorf("%w: redpen content must be UTF-8", ErrInvalidInput)
	}
	candidate, err := buildRawArtifact(CreateRawArtifactRequest{
		ArtifactID: req.ArtifactID,
		MissionID:  req.MissionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   reportRedpenFilename(source),
		Producer:   req.Producer,
		Content:    req.Content,
	})
	if err != nil {
		return ReportRedpenWorkcopy{}, err
	}

	artifact, event, changed, err := store.CommitReportRedpenRevision(ctx, candidate, func(events []LedgerEvent, target RawArtifact) (LedgerEvent, bool, error) {
		currentPayload, currentEvent, exists, err := latestReportRedpenEvent(events, source.ArtifactID)
		if err != nil {
			return LedgerEvent{}, false, err
		}
		expected := strings.TrimSpace(req.ExpectedCurrentArtifactID)
		if !exists && expected != "" {
			return LedgerEvent{}, false, fmt.Errorf("%w: redpen workcopy does not exist", ErrConflict)
		}
		if exists && expected != currentPayload.ArtifactID {
			return LedgerEvent{}, false, fmt.Errorf("%w: redpen workcopy changed in another session", ErrConflict)
		}
		if exists && candidate.SHA256 == currentPayload.SHA256 {
			return currentEvent, false, nil
		}
		if !isMarkdownArtifactMediaType(target.MediaType) {
			return LedgerEvent{}, false, fmt.Errorf("%w: redpen content matches a non-Markdown artifact", ErrConflict)
		}

		workcopyID := req.NewWorkcopyID
		revision := 1
		previousArtifactID := source.ArtifactID
		if exists {
			workcopyID = currentPayload.WorkcopyID
			revision = currentPayload.Revision + 1
			previousArtifactID = currentPayload.ArtifactID
		}
		payload := reportRedpenEventPayload{
			Kind:               ReportRedpenArtifactKind,
			WorkcopyID:         workcopyID,
			SourceArtifactID:   source.ArtifactID,
			ArtifactID:         target.ArtifactID,
			PreviousArtifactID: previousArtifactID,
			Revision:           revision,
			SHA256:             target.SHA256,
			MediaType:          candidate.MediaType,
			Filename:           candidate.Filename,
		}
		eventReq, err := payload.appendRequest(req, currentEvent)
		if err != nil {
			return LedgerEvent{}, false, err
		}
		built, err := buildLedgerEvent(eventReq)
		return built, err == nil, err
	})
	if err != nil {
		return ReportRedpenWorkcopy{}, err
	}
	payload, err := decodeReportRedpenPayload(event)
	if err != nil {
		return ReportRedpenWorkcopy{}, err
	}
	return reportRedpenWorkcopy(payload, artifact, event, changed), nil
}

func (s *Service) reportRedpenSource(ctx context.Context, missionID, sourceArtifactID string) (RawArtifact, []LedgerEvent, error) {
	if err := validateID("mis_", missionID); err != nil {
		return RawArtifact{}, nil, err
	}
	if err := validateID("art_", sourceArtifactID); err != nil {
		return RawArtifact{}, nil, err
	}
	artifact, err := s.GetRawArtifact(ctx, sourceArtifactID)
	if err != nil {
		return RawArtifact{}, nil, err
	}
	if artifact.MissionID != missionID || !isMarkdownArtifactMediaType(artifact.MediaType) {
		return RawArtifact{}, nil, fmt.Errorf("%w: redpen source must be a Markdown report artifact in the mission", ErrInvalidInput)
	}
	events, err := s.ListEvents(ctx, missionID)
	if err != nil {
		return RawArtifact{}, nil, err
	}
	if !hasReportRedpenSourceEvent(events, artifact.ArtifactID) {
		return RawArtifact{}, nil, fmt.Errorf("%w: redpen source must be a Markdown report artifact in the mission", ErrInvalidInput)
	}
	return artifact, events, nil
}
