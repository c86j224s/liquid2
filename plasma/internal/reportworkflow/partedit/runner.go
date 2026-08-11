package partedit

import (
	"context"
	"fmt"
	"strings"

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
	result, runErr := runner.Executor.Run(ctx, agentexec.AgentRequest{
		UserText: userText,
		Prompt:   reportprompt.WithLongFormDownstreamDirection(Prompt(input, binding, draftID), input.Base.DirectionHint),
		Model:    input.Base.AgentModel, ReasoningEffort: input.Base.AgentReasoningEffort,
		MissionID: input.Base.MissionID, ToolSessionID: input.ToolSessionID, PreviousSessionID: input.PreviousSessionID,
		AgentExecutor: input.Base.AgentExecutor, MCPMode: input.Base.MCPMode,
		ExtraMCPTools: MCPTools(), ReplaceMCPTools: true, PartEdit: &binding,
	})
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
