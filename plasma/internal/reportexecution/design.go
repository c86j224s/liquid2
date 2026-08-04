package reportexecution

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func (runner Runner) StartDesign(ctx context.Context, missionID string, req DesignRequest, producer ledger.Producer) (ledger.Event, error) {
	appended, err := runner.Service.AppendEventsIfNoActiveAgentWork(ctx, missionID, []ledger.AppendRequest{{
		EventID:   runner.id("evt"),
		MissionID: missionID,
		EventType: "report.design.pending",
		Producer:  producer,
		Payload: mustJSON(map[string]any{
			"kind":                   "designed_html_report_pending",
			"source_artifact_id":     req.SourceArtifactID,
			"source_media_type":      req.SourceMediaType,
			"title":                  req.Title,
			"agent_executor":         req.AgentExecutor,
			"agent_model":            strings.TrimSpace(req.AgentModel),
			"agent_reasoning_effort": strings.TrimSpace(req.AgentReasoningEffort),
			"target":                 DesignTargetDesigned,
			"renderer_version":       req.RendererVersion,
			"text":                   "Designed HTML 리포트 artifact를 생성 중입니다.",
			"started_at":             time.Now().UTC().Format(time.RFC3339Nano),
		}),
	}})
	if err != nil {
		return ledger.Event{}, err
	}
	pending := appended[0]
	return pending, runner.RunDesign(context.Background(), missionID, req, pending.EventID)
}

// ResumeDesign는 보고서 생성 파이프라인 실행 lifecycle을 다룬다. 중복 실행과 취소는 저장된 pending/terminal 이벤트 기준으로 판정한다.
func (runner Runner) ResumeDesign(ctx context.Context, missionID string, pending ledger.Event) error {
	req, err := DesignRequestFromPendingEvent(pending)
	if err != nil {
		_, failErr := runner.AppendDesignFailed(ctx, missionID, pending.EventID, "", "", "", err)
		return failErr
	}
	return runner.RunDesign(context.Background(), missionID, req, pending.EventID)
}

// RunDesign는 보고서 생성 파이프라인 실행 lifecycle을 다룬다. 중복 실행과 취소는 저장된 pending/terminal 이벤트 기준으로 판정한다.
func (runner Runner) RunDesign(ctx context.Context, missionID string, req DesignRequest, pendingEventID string) error {
	if runner.InFlight == nil {
		return fmt.Errorf("%w: report runner requires in-flight registry", producterror.ErrInvalidInput)
	}
	if runner.GenerateDesign == nil {
		return fmt.Errorf("%w: report runner requires design generator", producterror.ErrInvalidInput)
	}
	workerCtx, cancel := context.WithCancel(ctx)
	runID, ok := runner.InFlight.Start(missionID, pendingEventID, cancel)
	if !ok {
		cancel()
		if runner.isSamePendingAlreadyRunning(missionID, pendingEventID) || runner.hasTerminalEvent(context.Background(), missionID, pendingEventID) {
			return nil
		}
		if _, err := runner.AppendDesignFailed(context.Background(), missionID, pendingEventID, req.AgentExecutor, req.SourceArtifactID, req.RendererVersion, errors.New("report draft is already running for this mission")); err != nil {
			logTerminalWriteFailure(missionID, pendingEventID, "design", "report.design.failed", err)
		}
		return fmt.Errorf("%w: report draft is already running for this mission", producterror.ErrInvalidInput)
	}
	go func() {
		defer cancel()
		defer runner.InFlight.Finish(missionID, runID)
		if runner.hasTerminalEvent(context.Background(), missionID, pendingEventID) {
			return
		}
		if err := runner.GenerateDesign(workerCtx, missionID, req, pendingEventID); err != nil {
			failCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if _, appendErr := runner.AppendDesignFailed(failCtx, missionID, pendingEventID, req.AgentExecutor, req.SourceArtifactID, req.RendererVersion, err); appendErr != nil {
				logTerminalWriteFailure(missionID, pendingEventID, "design", "report.design.failed", appendErr)
			}
		}
	}()
	return nil
}
