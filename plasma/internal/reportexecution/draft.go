package reportexecution

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportpipeline"
	"github.com/c86j224s/liquid2/plasma/internal/source"
)

func (runner Runner) StartDraft(ctx context.Context, missionID string, req DraftRequest, producer ledger.Producer) (ledger.Event, error) {
	if _, err := NormalizePipelineFamily(req.PipelineFamily); err != nil {
		return ledger.Event{}, err
	}
	req = normalizeDraftRequest(req)
	sources, err := runner.Service.ListSourceSnapshotsWithState(ctx, source.ListRequest{MissionID: missionID})
	if err != nil {
		return ledger.Event{}, err
	}
	startedAt := time.Now().UTC()
	pendingEventID := runner.id("evt")
	payload := map[string]any{
		"kind":                            "markdown_report_artifact_pending",
		"title":                           req.Title,
		"agent_executor":                  req.AgentExecutor,
		"agent_model":                     req.AgentModel,
		"agent_reasoning_effort":          req.AgentReasoningEffort,
		"agent_selection_source":          req.AgentSelectionSource,
		"mcp_mode":                        req.MCPMode,
		"rigor_level":                     req.RigorLevel,
		"rigor_label":                     req.RigorLabel,
		"report_mode":                     req.ReportMode,
		"report_mode_label":               ModeLabel(req.ReportMode),
		"pipeline_family":                 req.PipelineFamily,
		"report_session_policy":           req.ReportSessionPolicy,
		"report_session_policy_selection": req.ReportSessionPolicySelection,
		"post_report_humanize":            req.PostReportHumanize,
		"humanize_enabled":                req.PostReportHumanize != "disabled",
		"generation_guidance_profile":     req.GenerationGuidanceProfile,
		"generation_guidance_sha256":      req.GenerationGuidanceSHA256,
		"text":                            "리포트 초안 생성 중입니다.",
		"started_at":                      startedAt.Format(time.RFC3339Nano),
		"source_context":                  buildReportSourceContext(sources, startedAt),
		"origin_pending_event_id":         pendingEventID,
		"attempt_number":                  1,
		"retry_strategy":                  "initial",
	}
	if req.DirectionHint != "" {
		payload["direction_hint"] = req.DirectionHint
	}
	if req.PipelineGraph != "" {
		payload["pipeline_graph"] = req.PipelineGraph
	}
	if req.ExecutionStrategy != "" {
		payload["execution_strategy"] = req.ExecutionStrategy
	}
	appended, err := runner.Service.AppendEventsIfNoActiveAgentWork(ctx, missionID, []ledger.AppendRequest{{
		EventID:   pendingEventID,
		MissionID: missionID,
		EventType: "report.draft.pending",
		Producer:  producer,
		Payload:   mustJSON(payload),
	}})
	if err != nil {
		return ledger.Event{}, err
	}
	pending := appended[0]
	return pending, runner.RunDraft(context.Background(), missionID, req, pending.EventID)
}

func (runner Runner) generateDraft(ctx context.Context, missionID string, req DraftRequest, pendingEventID string) error {
	family, err := NormalizePipelineFamily(req.PipelineFamily)
	if err != nil {
		return err
	}
	switch family {
	case reportpipeline.ExperimentalIL:
		if runner.GenerateExperimental == nil {
			return fmt.Errorf("%w: report IL experimental generator is unavailable", producterror.ErrInvalidInput)
		}
		return runner.GenerateExperimental(ctx, missionID, req, pendingEventID)
	case reportpipeline.Unverified:
		if runner.GenerateUnverified == nil {
			return fmt.Errorf("%w: unverified report generator is unavailable", producterror.ErrInvalidInput)
		}
		return runner.GenerateUnverified(ctx, missionID, req, pendingEventID)
	}
	if runner.GenerateDraft == nil {
		return fmt.Errorf("%w: report runner requires draft generator", producterror.ErrInvalidInput)
	}
	return runner.GenerateDraft(ctx, missionID, req, pendingEventID)
}

// ResumeDraft는 보고서 생성 파이프라인 실행 lifecycle을 다룬다. 중복 실행과 취소는 저장된 pending/terminal 이벤트 기준으로 판정한다.
func (runner Runner) ResumeDraft(ctx context.Context, missionID string, pending ledger.Event) error {
	req, err := DraftRequestFromPendingEvent(pending)
	if err != nil {
		_, failErr := runner.AppendDraftFailed(ctx, missionID, pending.EventID, "", "", err)
		return failErr
	}
	return runner.RunDraft(context.Background(), missionID, req, pending.EventID)
}

// RunDraft는 보고서 생성 파이프라인 실행 lifecycle을 다룬다. 중복 실행과 취소는 저장된 pending/terminal 이벤트 기준으로 판정한다.
func (runner Runner) RunDraft(ctx context.Context, missionID string, req DraftRequest, pendingEventID string) error {
	req = NormalizeDraftRequest(req)
	if runner.InFlight == nil {
		return fmt.Errorf("%w: report runner requires in-flight registry", producterror.ErrInvalidInput)
	}
	if runner.GenerateDraft == nil && runner.GenerateExperimental == nil && runner.GenerateUnverified == nil {
		return fmt.Errorf("%w: report runner requires draft generator", producterror.ErrInvalidInput)
	}
	workerCtx, cancel := context.WithCancel(ctx)
	runID, ok := runner.InFlight.Start(missionID, pendingEventID, cancel)
	if !ok {
		cancel()
		if runner.isSamePendingAlreadyRunning(missionID, pendingEventID) || runner.hasTerminalEvent(context.Background(), missionID, pendingEventID) {
			return nil
		}
		if _, err := runner.AppendDraftFailed(context.Background(), missionID, pendingEventID, req.AgentExecutor, req.ReportMode, errors.New("report draft is already running for this mission")); err != nil {
			logTerminalWriteFailure(missionID, pendingEventID, "draft", "report.draft.failed", err)
		}
		return fmt.Errorf("%w: report draft is already running for this mission", producterror.ErrInvalidInput)
	}
	go func() {
		defer cancel()
		defer runner.InFlight.Finish(missionID, runID)
		if runner.hasTerminalEvent(context.Background(), missionID, pendingEventID) {
			return
		}
		if err := runner.generateDraft(workerCtx, missionID, req, pendingEventID); err != nil {
			failCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if _, appendErr := runner.AppendDraftFailed(failCtx, missionID, pendingEventID, req.AgentExecutor, req.ReportMode, err); appendErr != nil {
				logTerminalWriteFailure(missionID, pendingEventID, "draft", "report.draft.failed", appendErr)
			}
		}
	}()
	return nil
}
