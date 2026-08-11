package directdraft

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
)

// RunOneTake는 기존 research session을 이어 one_take Markdown provider 후보를 만든다.
func (runner Runner) RunOneTake(ctx context.Context, input BaseInput) (OneTakeCandidate, error) {
	artifactID := runner.id("art")
	toolSessionID := runner.id("ses")
	reportSessionPolicy := firstNonEmpty(input.ReportSessionPolicy, reportexecution.SessionPolicySameSession)
	previousSessionID := latestSession(ctx, runner, input.MissionID, input.AgentExecutor)
	started := time.Now()
	result, err := runner.Executor.Run(ctx, agentexec.AgentRequest{
		UserText:          "generate quick markdown report artifact",
		Prompt:            reportprompt.WithReportDirection(reportprompt.OneTakeMarkdownReportPrompt(input.Title, input.MissionID, toolSessionID, input.Rigor, input.GenerationGuidanceProfile), input.DirectionHint),
		Model:             input.AgentModel,
		ReasoningEffort:   input.AgentReasoningEffort,
		MissionID:         input.MissionID,
		ToolSessionID:     toolSessionID,
		PreviousSessionID: previousSessionID,
		AgentExecutor:     input.AgentExecutor,
		MCPMode:           input.MCPMode,
		ExtraMCPTools:     ReadMCPTools(),
		ReplaceMCPTools:   true,
	})
	agentDurationMS := time.Since(started).Milliseconds()
	if err != nil {
		return OneTakeCandidate{}, fmt.Errorf("quick report agent failed: %w", reportAgentFailure(err, result, "report_one_take", agentDurationMS, previousSessionID))
	}
	returnedSessionID := strings.TrimSpace(result.SessionID)
	result, err = validateSameSessionResult(result, previousSessionID)
	if err != nil {
		return OneTakeCandidate{}, reportAgentFailure(err, result, "report_one_take", agentDurationMS, previousSessionID)
	}
	markdown := strings.TrimSpace(result.Text)
	if markdown == "" {
		return OneTakeCandidate{}, reportAgentFailure(fmt.Errorf("%w: report agent returned empty Markdown", producterror.ErrInvalidInput), result, "report_one_take", agentDurationMS, previousSessionID)
	}
	return OneTakeCandidate{
		ArtifactID: artifactID, ToolSessionID: toolSessionID,
		PreviousSessionID: previousSessionID, ReturnedSessionID: returnedSessionID,
		ReportSessionID: result.SessionID, ReportSessionPolicy: reportSessionPolicy,
		Markdown: markdown, StartedAt: started, AgentDurationMS: agentDurationMS,
		AgentUsage: result.Usage, AgentResumed: result.Resumed,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
