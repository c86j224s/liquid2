package directdraft

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
)

// RunPlanned는 canonical plan session을 이어 planned Markdown provider 후보를 만든다.
func (runner Runner) RunPlanned(ctx context.Context, input PlannedInput) (PlannedCandidate, error) {
	artifactID := strings.TrimSpace(input.ArtifactID)
	if artifactID == "" {
		artifactID = runner.id("art")
	}
	toolSessionID := runner.id("ses")
	reportStarted := time.Now()
	started := input.WorkflowStartedAt
	if started.IsZero() {
		started = reportStarted
	}
	result, err := runner.Executor.Run(ctx, agentexec.AgentRequest{
		UserText:          "generate markdown report artifact",
		Prompt:            reportprompt.WithReportDirection(reportprompt.PlannedMarkdownReportPrompt(input.Title, input.MissionID, toolSessionID, input.Rigor, input.Plan, input.GenerationGuidanceProfile), input.DirectionHint),
		Model:             input.AgentModel,
		ReasoningEffort:   input.AgentReasoningEffort,
		MissionID:         input.MissionID,
		ToolSessionID:     toolSessionID,
		PreviousSessionID: input.ReportPlanSessionID,
		AgentExecutor:     input.AgentExecutor,
		MCPMode:           input.MCPMode,
		ExtraMCPTools:     ReadMCPTools(),
		ReplaceMCPTools:   true,
	})
	reportDurationMS := time.Since(reportStarted).Milliseconds()
	if err != nil {
		return PlannedCandidate{}, fmt.Errorf("report agent failed: %w", reportAgentFailure(err, result, "report_markdown", reportDurationMS, input.ReportPlanSessionID))
	}
	returnedSessionID := strings.TrimSpace(result.SessionID)
	result, err = validateSameSessionResult(result, input.ReportPlanSessionID)
	if err != nil {
		return PlannedCandidate{}, reportAgentFailure(err, result, "report_markdown", reportDurationMS, input.ReportPlanSessionID)
	}
	markdown := strings.TrimSpace(result.Text)
	if markdown == "" {
		return PlannedCandidate{}, reportAgentFailure(fmt.Errorf("%w: report agent returned empty Markdown", producterror.ErrInvalidInput), result, "report_markdown", reportDurationMS, input.ReportPlanSessionID)
	}
	return PlannedCandidate{
		ArtifactID: artifactID, ToolSessionID: toolSessionID,
		PlanEventID: input.PlanEventID, PlanToolSessionID: input.PlanToolSessionID,
		ReportPlanSessionID: input.ReportPlanSessionID, SessionChainKind: input.SessionChainKind,
		PreReportResearchSessionID: input.PreReportResearchSessionID, ForkSourceSessionID: input.ForkSourceSessionID,
		ReturnedSessionID: returnedSessionID, ReportSessionID: result.SessionID, Markdown: markdown,
		WorkflowStartedAt: started, AgentDurationMS: reportDurationMS,
		AgentUsage: result.Usage, AgentResumed: result.Resumed,
	}, nil
}
