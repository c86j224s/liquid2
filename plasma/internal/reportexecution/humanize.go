package reportexecution

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func (runner Runner) StartHumanize(ctx context.Context, missionID string, req HumanizeRequest, producer ledger.Producer) (ledger.Event, error) {
	req = normalizeHumanizeRequest(req)
	eventID := runner.id("evt")
	toolSessionID := firstNonEmpty(req.ToolSessionID, runner.id("ses"))
	req.ToolSessionID = toolSessionID
	appended, err := runner.Service.AppendEventsIfNoActiveAgentWork(ctx, missionID, []ledger.AppendRequest{{
		EventID:   eventID,
		MissionID: missionID,
		EventType: "report.humanize.pending",
		Producer:  producer,
		Payload: mustJSON(map[string]any{
			"kind":                      "humanized_markdown_report_pending",
			"target":                    "humanized_markdown",
			"profile":                   "h5-full-report-tone-pass",
			"pending_event_id":          eventID,
			"report_pending_event_id":   req.ReportPendingEventID,
			"title":                     req.Title,
			"source_artifact_id":        req.SourceArtifactID,
			"source_artifact_sha256":    req.SourceArtifactSHA256,
			"source_media_type":         req.SourceMediaType,
			"agent_executor":            req.AgentExecutor,
			"agent_model":               req.AgentModel,
			"agent_reasoning_effort":    req.AgentReasoningEffort,
			"previous_agent_session_id": req.PreviousAgentSessionID,
			"tool_session_id":           toolSessionID,
			"mcp_mode":                  req.MCPMode,
			"report_mode":               req.ReportMode,
			"report_mode_label":         ModeLabel(req.ReportMode),
			"humanize_transport":        "mcp_patch",
			"relationship":              "pending_post_report_tone_pass_of_source_artifact",
			"text":                      "H5 말투 보정 Markdown artifact를 생성하는 중입니다.",
			"started_at":                time.Now().UTC().Format(time.RFC3339Nano),
		}),
	}})
	if err != nil {
		return ledger.Event{}, err
	}
	pending := appended[0]
	return pending, runner.RunHumanize(context.Background(), missionID, req, pending.EventID)
}

// ResumeHumanize는 보고서 생성 파이프라인 실행 lifecycle을 다룬다. 중복 실행과 취소는 저장된 pending/terminal 이벤트 기준으로 판정한다.
func (runner Runner) ResumeHumanize(ctx context.Context, missionID string, pending ledger.Event) error {
	req, err := HumanizeRequestFromPendingEvent(pending)
	if err != nil {
		_, failErr := runner.AppendHumanizeFailed(ctx, missionID, pending.EventID, "", "", "", err)
		return failErr
	}
	return runner.RunHumanize(context.Background(), missionID, req, pending.EventID)
}

// RunHumanize는 보고서 생성 파이프라인 실행 lifecycle을 다룬다. 중복 실행과 취소는 저장된 pending/terminal 이벤트 기준으로 판정한다.
func (runner Runner) RunHumanize(ctx context.Context, missionID string, req HumanizeRequest, pendingEventID string) error {
	req = normalizeHumanizeRequest(req)
	if runner.InFlight == nil {
		return fmt.Errorf("%w: report runner requires in-flight registry", producterror.ErrInvalidInput)
	}
	if runner.GenerateHumanize == nil {
		return fmt.Errorf("%w: report runner requires humanize generator", producterror.ErrInvalidInput)
	}
	workerCtx, cancel := context.WithCancel(ctx)
	runID, ok := runner.InFlight.Start(missionID, pendingEventID, cancel)
	if !ok {
		cancel()
		if runner.isSamePendingAlreadyRunning(missionID, pendingEventID) || runner.hasTerminalEvent(context.Background(), missionID, pendingEventID) {
			return nil
		}
		if _, err := runner.AppendHumanizeFailed(context.Background(), missionID, pendingEventID, req.AgentExecutor, req.SourceArtifactID, req.ReportMode, errors.New("report draft is already running for this mission")); err != nil {
			logTerminalWriteFailure(missionID, pendingEventID, "humanize", "report.humanize.failed", err)
		}
		return fmt.Errorf("%w: report draft is already running for this mission", producterror.ErrInvalidInput)
	}
	go func() {
		defer cancel()
		defer runner.InFlight.Finish(missionID, runID)
		if runner.hasTerminalEvent(context.Background(), missionID, pendingEventID) {
			return
		}
		if err := runner.GenerateHumanize(workerCtx, missionID, req, pendingEventID); err != nil {
			failCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if _, appendErr := runner.AppendHumanizeFailed(failCtx, missionID, pendingEventID, req.AgentExecutor, req.SourceArtifactID, req.ReportMode, err); appendErr != nil {
				logTerminalWriteFailure(missionID, pendingEventID, "humanize", "report.humanize.failed", appendErr)
			}
		}
	}()
	return nil
}
