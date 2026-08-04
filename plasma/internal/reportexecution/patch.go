package reportexecution

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func (runner Runner) StartPatch(ctx context.Context, missionID string, req PatchRequest, producer ledger.Producer) (ledger.Event, error) {
	req = normalizePatchRequest(req)
	appended, err := runner.Service.AppendEventsIfNoActiveAgentWork(ctx, missionID, []ledger.AppendRequest{{
		EventID:   runner.id("evt"),
		MissionID: missionID,
		EventType: "report.patch.pending",
		Producer:  producer,
		Payload: mustJSON(map[string]any{
			"kind":                            "markdown_report_patch_pending",
			"base_artifact_id":                req.BaseArtifactID,
			"title":                           req.Title,
			"instruction":                     req.Instruction,
			"agent_executor":                  req.AgentExecutor,
			"agent_model":                     req.AgentModel,
			"agent_reasoning_effort":          req.AgentReasoningEffort,
			"mcp_mode":                        req.MCPMode,
			"report_session_id":               req.ReportSessionID,
			"previous_agent_session_id":       req.PreviousAgentSessionID,
			"fork_source_agent_session_id":    req.ForkSourceAgentSessionID,
			"report_session_policy":           req.ReportSessionPolicy,
			"report_session_policy_selection": req.ReportSessionPolicySelection,
			"session_chain_kind":              req.SessionChainKind,
			"text":                            "MCP 패치 방식으로 리포트 수정 중입니다.",
			"started_at":                      time.Now().UTC().Format(time.RFC3339Nano),
		}),
	}})
	if err != nil {
		return ledger.Event{}, err
	}
	pending := appended[0]
	return pending, runner.RunPatch(context.Background(), missionID, req, pending.EventID)
}

// ResumePatch는 보고서 생성 파이프라인 실행 lifecycle을 다룬다. 중복 실행과 취소는 저장된 pending/terminal 이벤트 기준으로 판정한다.
func (runner Runner) ResumePatch(ctx context.Context, missionID string, pending ledger.Event) error {
	req, err := PatchRequestFromPendingEvent(pending)
	if err != nil {
		_, failErr := runner.AppendPatchFailed(ctx, missionID, pending.EventID, "", "", err)
		return failErr
	}
	return runner.RunPatch(context.Background(), missionID, req, pending.EventID)
}

// RunPatch는 보고서 생성 파이프라인 실행 lifecycle을 다룬다. 중복 실행과 취소는 저장된 pending/terminal 이벤트 기준으로 판정한다.
func (runner Runner) RunPatch(ctx context.Context, missionID string, req PatchRequest, pendingEventID string) error {
	if runner.InFlight == nil {
		return fmt.Errorf("%w: report runner requires in-flight registry", producterror.ErrInvalidInput)
	}
	if runner.GeneratePatch == nil {
		return fmt.Errorf("%w: report runner requires patch generator", producterror.ErrInvalidInput)
	}
	workerCtx, cancel := context.WithCancel(ctx)
	runID, ok := runner.InFlight.Start(missionID, pendingEventID, cancel)
	if !ok {
		cancel()
		if runner.isSamePendingAlreadyRunning(missionID, pendingEventID) || runner.hasTerminalEvent(context.Background(), missionID, pendingEventID) {
			return nil
		}
		if _, err := runner.AppendPatchFailed(context.Background(), missionID, pendingEventID, req.AgentExecutor, req.BaseArtifactID, errors.New("report draft is already running for this mission")); err != nil {
			logTerminalWriteFailure(missionID, pendingEventID, "patch", "report.patch.failed", err)
		}
		return fmt.Errorf("%w: report draft is already running for this mission", producterror.ErrInvalidInput)
	}
	go func() {
		defer cancel()
		defer runner.InFlight.Finish(missionID, runID)
		if runner.hasTerminalEvent(context.Background(), missionID, pendingEventID) {
			return
		}
		if err := runner.GeneratePatch(workerCtx, missionID, req, pendingEventID); err != nil {
			failCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if _, appendErr := runner.AppendPatchFailed(failCtx, missionID, pendingEventID, req.AgentExecutor, req.BaseArtifactID, err); appendErr != nil {
				logTerminalWriteFailure(missionID, pendingEventID, "patch", "report.patch.failed", appendErr)
			}
		}
	}()
	return nil
}
