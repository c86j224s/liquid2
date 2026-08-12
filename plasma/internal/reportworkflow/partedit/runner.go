package partedit

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/longformutil"
)

// Run은 durable start를 기록한 뒤 Part edit/author agent 결과를 durable replay에서 채택한다.
func (runner Runner) Run(ctx context.Context, input Input) (Output, error) {
	binding, err := runner.binding(ctx, input)
	if err != nil {
		return Output{}, err
	}
	if _, _, err := reporting.StartPartEdit(ctx, runner.Service, runner.id("evt"), binding); err != nil {
		return Output{}, err
	}
	draftID := runner.id("rpe")
	userText := fmt.Sprintf("edit assembled part %d of the long-form report", input.PartIndex+1)
	if input.AuthorMode {
		userText = fmt.Sprintf("write final part %d of the long-form report", input.PartIndex+1)
	}
	started := time.Now()
	result, runErr := runner.Executor.Run(ctx, agentexec.AgentRequest{
		UserText: userText,
		Prompt:   reportprompt.WithLongFormDownstreamDirection(Prompt(input, binding, draftID), input.Base.DirectionHint),
		Model:    input.Base.AgentModel, ReasoningEffort: input.Base.AgentReasoningEffort,
		MissionID: input.Base.MissionID, ToolSessionID: input.ToolSessionID, PreviousSessionID: input.PreviousSessionID,
		AgentExecutor: input.Base.AgentExecutor, MCPMode: input.Base.MCPMode,
		ExtraMCPTools: MCPTools(), ReplaceMCPTools: true, PartEdit: &binding,
	})
	durationMS := time.Since(started).Milliseconds()
	if runErr == nil {
		result, runErr = longformutil.ValidateSameSessionResult(result, input.PreviousSessionID)
	}
	edited, exists, loadErr := reporting.LoadPartEdit(context.WithoutCancel(ctx), runner.Service, binding)
	if loadErr != nil {
		return Output{}, loadErr
	}
	if !exists {
		if runErr != nil {
			return Output{Result: result}, runErr
		}
		message := "Part editor did not submit a durable edit"
		if input.AuthorMode {
			message = "final Part author did not submit a durable Part"
		}
		return Output{Result: result}, fmt.Errorf("%w: %s", producterror.ErrConflict, message)
	}
	agentSessionID := strings.TrimSpace(result.SessionID)
	if agentSessionID == "" {
		agentSessionID = binding.ProviderSessionID
	}
	if _, _, usageErr := reporting.RecordReportAgentUsage(context.WithoutCancel(ctx), runner.Service, reporting.ReportAgentUsageRequest{
		MissionID: input.Base.MissionID, PendingEventID: input.Base.PendingEventID, CanonicalEventID: edited.Event.EventID,
		ForkSourceAgentSessionID: binding.ForkSourceAgentSessionID, Surface: "report_part_edit",
		PreviousAgentSessionID: input.PreviousSessionID, AgentSessionID: agentSessionID,
		DurationMS: durationMS, Resumed: result.Resumed, Usage: result.Usage,
	}); usageErr != nil {
		log.Printf("report_agent_usage_write_failed mission_id=%q canonical_event_id=%q surface=%q err=%q", input.Base.MissionID, edited.Event.EventID, "report_part_edit", usageErr)
	}
	if runErr == nil && strings.TrimSpace(result.Text) != reporting.PartEditSubmittedSentinel {
		message := "Part editor acknowledgement was not exact"
		if input.AuthorMode {
			message = "final Part author acknowledgement was not exact"
		}
		return Output{Result: result}, fmt.Errorf("%w: %s", producterror.ErrConflict, message)
	}
	markdown := strings.TrimSpace(string(edited.Artifact.Content))
	return Output{
		Draft:  PartDraft{Title: input.Source.Title, Markdown: markdown, ArtifactID: edited.Artifact.ArtifactID, WordCount: longformutil.WordCount(markdown)},
		Result: result,
	}, nil
}

func (runner Runner) id(prefix string) string {
	if runner.NewID != nil {
		return runner.NewID(prefix)
	}
	return prefix + "_missing"
}
