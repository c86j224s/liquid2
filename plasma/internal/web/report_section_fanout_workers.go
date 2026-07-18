package web

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type sectionFanoutTaskOutcome struct {
	index  int
	result sectionFanoutResult
	err    error
}

func (server *Server) runSectionFanoutTasks(ctx context.Context, req sectionFanoutLongFormRequest, state sectionFanoutPlanState, tasks []sectionFanoutTask, executor AgentExecutor) ([]sectionFanoutResult, error) {
	if len(tasks) == 0 {
		return nil, nil
	}
	limit := sectionFanoutWorkerLimit
	if limit > len(tasks) {
		limit = len(tasks)
	}
	results := make([]sectionFanoutResult, len(tasks))
	errs := make([]error, len(tasks))
	sem := make(chan struct{}, limit)
	outcomes := make(chan sectionFanoutTaskOutcome, len(tasks))
	var wg sync.WaitGroup
	for index, task := range tasks {
		wg.Add(1)
		sem <- struct{}{}
		go func(index int, task sectionFanoutTask) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := server.appendSectionFanoutStartedEvent(ctx, req, state, task); err != nil {
				outcomes <- sectionFanoutTaskOutcome{index: index, err: err}
				return
			}
			started := time.Now()
			result, err := executor.Run(ctx, AgentRequest{
				UserText:          fmt.Sprintf("draft section %d.%d for section-fanout long-form markdown report", task.partIndex+1, task.sectionIndex+1),
				Prompt:            withReportDirection(agentSectionDraftPrompt(req.title, req.missionID, task.toolSessionID, req.rigor, state.plan, task.part, task.section, task.partIndex, task.sectionIndex, req.generationGuidanceProfile), req.directionHint),
				Model:             req.agentModel,
				ReasoningEffort:   req.agentReasoningEffort,
				MissionID:         req.missionID,
				ToolSessionID:     task.toolSessionID,
				PreviousSessionID: task.previousSession,
				AgentExecutor:     req.executorName,
				MCPMode:           req.mcpMode,
			})
			durationMS := time.Since(started).Milliseconds()
			if err != nil {
				outcomes <- sectionFanoutTaskOutcome{index: index, err: longFormStageFailure("section", state.planEvent.EventID, task.partIndex+1, task.sectionIndex+1, reportAgentFailure(err, result, "report_section", durationMS, task.previousSession))}
				return
			}
			returnedSessionID := strings.TrimSpace(result.SessionID)
			validated, err := validatedSameSessionResult(result, task.previousSession)
			if err != nil {
				outcomes <- sectionFanoutTaskOutcome{index: index, err: longFormStageFailure("section", state.planEvent.EventID, task.partIndex+1, task.sectionIndex+1, reportAgentFailure(err, result, "report_section", durationMS, task.previousSession))}
				return
			}
			markdown := strings.TrimSpace(validated.Text)
			if markdown == "" {
				outcomes <- sectionFanoutTaskOutcome{index: index, err: longFormStageFailure("section", state.planEvent.EventID, task.partIndex+1, task.sectionIndex+1, reportAgentFailure(fmt.Errorf("%w: section report agent returned empty Markdown", app.ErrInvalidInput), validated, "report_section", durationMS, task.previousSession))}
				return
			}
			item, err := server.persistSectionFanoutResult(ctx, req, state, sectionFanoutResult{task: task, result: validated, returnedSessionID: returnedSessionID, durationMS: durationMS, markdown: markdown})
			if err != nil {
				outcomes <- sectionFanoutTaskOutcome{index: index, err: err}
				return
			}
			outcomes <- sectionFanoutTaskOutcome{index: index, result: item}
		}(index, task)
	}
	go func() {
		wg.Wait()
		close(outcomes)
	}()
	for outcome := range outcomes {
		if outcome.err != nil {
			errs[outcome.index] = outcome.err
			continue
		}
		results[outcome.index] = outcome.result
	}
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return results, nil
}
