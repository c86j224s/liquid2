package sourcecandidates

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

const sourceCandidateFetchingMessage = "후보 원문을 백그라운드에서 가져오는 중입니다. 완료되면 source.candidate.staged 또는 source.candidate.staging_failed 이벤트가 장부에 남습니다."
const defaultSourceCandidateRejectReason = "사용자가 이 URL을 이번 미션의 소스로 쓰지 않기로 했습니다."
const defaultSourceCandidateRestoreReason = "사용자가 기각했던 URL을 다시 소스 후보로 검토하기로 했습니다."

type Store interface {
	appender
	ListRawArtifacts(context.Context, string) ([]app.RawArtifact, error)
	ListEvents(context.Context, string) ([]app.LedgerEvent, error)
	ListSourceSnapshotsWithState(context.Context, app.ListSourceSnapshotsRequest) ([]app.SourceSnapshot, error)
	CreateRawArtifactWithEvent(context.Context, app.CreateRawArtifactRequest, func(app.RawArtifact) app.AppendEventRequest) (app.RawArtifact, app.LedgerEvent, error)
}

type appender interface {
	AppendEvent(context.Context, app.AppendEventRequest) (app.LedgerEvent, error)
}

func StartStaging(ctx context.Context, store Store, req SourceCandidateStagingStartRequest) (SourceCandidateStagingStartResult, error) {
	normalizedURL, _, err := NormalizeSourceCandidateURL(req.Candidate.URL)
	if err != nil {
		return SourceCandidateStagingStartResult{}, err
	}
	candidate := req.Candidate
	candidate.URL = normalizedURL
	payload := map[string]any{
		"kind":               "source_candidate_staging_started",
		"candidate_kind":     sourceCandidateKind(req.CandidateKind),
		"url":                candidate.URL,
		"title":              candidate.Title,
		"proposal_event_id":  strings.TrimSpace(req.ProposalEventID),
		"approval_state":     "unapproved_candidate",
		"not_report_default": true,
	}
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID != "" {
		payload["tool_session_id"] = sessionID
		payload["agent_session_id"] = sessionID
	}
	if strings.TrimSpace(req.AgentExecutor) != "" {
		payload["agent_executor"] = strings.TrimSpace(req.AgentExecutor)
	}
	event, err := store.AppendEvent(ctx, app.AppendEventRequest{
		EventID:          strings.TrimSpace(req.EventID),
		MissionID:        strings.TrimSpace(req.MissionID),
		EventType:        "source.candidate.staging_started",
		Producer:         req.Producer,
		CausationEventID: strings.TrimSpace(req.CausationEventID),
		CorrelationID:    sessionID,
		Payload:          mustMarshalJSON(payload),
	})
	if err != nil {
		return SourceCandidateStagingStartResult{}, err
	}
	return SourceCandidateStagingStartResult{
		Event: event,
		Output: SourceCandidateStagingOutput{
			URL:             candidate.URL,
			ProposalEventID: strings.TrimSpace(req.ProposalEventID),
			StagingEventID:  event.EventID,
			StagingState:    "fetching",
			Message:         sourceCandidateFetchingMessage,
		},
	}, nil
}

func Stage(ctx context.Context, store Store, req SourceCandidateStageRequest) error {
	if req.Fetcher == nil {
		return fmt.Errorf("%w: source candidate fetcher is required", app.ErrInvalidInput)
	}
	if req.NewArtifactID == nil || req.NewEventID == nil {
		return fmt.Errorf("%w: source candidate id generators are required", app.ErrInvalidInput)
	}
	job := req.Job
	normalizedURL, _, err := NormalizeSourceCandidateURL(job.Candidate.URL)
	if err != nil {
		_ = AppendStagingFailed(ctx, store, job, req.NewEventID, err)
		return err
	}
	job.Candidate.URL = normalizedURL
	fetched, err := req.Fetcher(ctx, normalizedURL)
	if err != nil {
		_ = AppendStagingFailed(ctx, store, job, req.NewEventID, err)
		return err
	}
	title := strings.TrimSpace(job.Candidate.Title)
	if title == "" {
		title = strings.TrimSpace(fetched.Title)
	}
	if title == "" {
		title = normalizedURL
	}
	contentSHA := sha256HexBytes(fetched.Content)
	if existing, ok, err := reusableArtifactBySHA(ctx, store, job.MissionID, contentSHA); err != nil {
		_ = AppendStagingFailed(ctx, store, job, req.NewEventID, err)
		return err
	} else if ok {
		_, err := store.AppendEvent(ctx, sourceCandidateStagedEventRequest(job, req.NewEventID("evt"), existing, title, fetched))
		if err != nil {
			_ = AppendStagingFailed(ctx, store, job, req.NewEventID, err)
		}
		return err
	}
	_, _, err = store.CreateRawArtifactWithEvent(ctx, app.CreateRawArtifactRequest{
		ArtifactID:     req.NewArtifactID("art"),
		MissionID:      job.MissionID,
		MediaType:      fetched.MediaType,
		Filename:       sourceCandidateFilename(title, fetched.MediaType, req.FilenameFallback),
		Producer:       job.Producer,
		Content:        fetched.Content,
		ExpectedSHA256: contentSHA,
	}, func(artifact app.RawArtifact) app.AppendEventRequest {
		return sourceCandidateStagedEventRequest(job, req.NewEventID("evt"), artifact, title, fetched)
	})
	if err != nil {
		_ = AppendStagingFailed(ctx, store, job, req.NewEventID, err)
	}
	return err
}

func AppendStagingFailed(ctx context.Context, store appender, job SourceCandidateStagingJob, newEventID SourceCandidateIDFunc, cause error) error {
	if newEventID == nil {
		return fmt.Errorf("%w: source candidate event id generator is required", app.ErrInvalidInput)
	}
	_, err := store.AppendEvent(ctx, sourceCandidateStagingFailedEventRequest(job, newEventID("evt"), cause))
	return err
}

func sourceCandidateKind(kind string) string {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return "url"
	}
	return kind
}

func Reject(ctx context.Context, store appender, req SourceCandidateDecisionRequest) (app.LedgerEvent, error) {
	return appendDecision(ctx, store, req, "source.candidate.rejected", "source_candidate_rejected", defaultSourceCandidateRejectReason)
}

func Restore(ctx context.Context, store appender, req SourceCandidateDecisionRequest) (app.LedgerEvent, error) {
	return appendDecision(ctx, store, req, "source.candidate.restored", "source_candidate_restored", defaultSourceCandidateRestoreReason)
}

func appendDecision(ctx context.Context, store appender, req SourceCandidateDecisionRequest, eventType string, kind string, defaultReason string) (app.LedgerEvent, error) {
	normalizedURL, err := normalizeSourceCandidateDecisionURL(req.URL)
	if err != nil {
		return app.LedgerEvent{}, err
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = defaultReason
	}
	return store.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   strings.TrimSpace(req.EventID),
		MissionID: strings.TrimSpace(req.MissionID),
		EventType: eventType,
		Producer:  req.Producer,
		Payload: mustMarshalJSON(map[string]any{
			"kind":   kind,
			"url":    normalizedURL,
			"reason": reason,
		}),
	})
}
