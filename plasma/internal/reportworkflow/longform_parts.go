package reportworkflow

import (
	"context"
	"fmt"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/longformutil"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partassembly"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partedit"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/plan"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/requirements"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/sectiondraft"
)

func (runner Runner) runPartAssemblies(ctx context.Context, base partassembly.BaseInput, planOut plan.LongFormOutput, progress longFormProgress, sections [][]sectiondraft.Draft, fanout bool) ([]partassembly.PartDraft, []string, string, error) {
	partDrafts := make([]partassembly.PartDraft, 0, len(planOut.Plan.Parts))
	partArtifactIDs := []string{}
	currentSessionID := firstNonEmpty(progress.currentSessionID, planOut.ReportPlanSessionID)
	forker, canFork := runner.partAssemblyRunner.Executor.(agentexec.AgentSessionForker)
	for partIndex, part := range planOut.Plan.Parts {
		if draft, ok := progress.parts[partIndex]; ok {
			partDrafts = append(partDrafts, draft)
			partArtifactIDs = append(partArtifactIDs, draft.ArtifactID)
			currentSessionID = firstNonEmpty(draft.SessionID, currentSessionID)
			continue
		}
		previousSessionID := currentSessionID
		forkSourceID := planOut.ForkSourceSessionID
		if fanout {
			if !canFork {
				return nil, nil, "", fmt.Errorf("%w: section fanout requires an agent session forker", producterror.ErrInvalidInput)
			}
			source, err := partSourceSession(planOut, progress, partIndex)
			if err != nil {
				return nil, nil, "", err
			}
			previousSessionID, forkSourceID, err = forkLongFormSession(ctx, forker, source)
			if err != nil {
				return nil, nil, "", longformutil.StageFailure("part", planOut.Event.EventID, partIndex+1, 0, err)
			}
			forkSourceID = firstNonEmpty(forkSourceID, source)
		}
		sectionInputs := make([]partassembly.SectionDraft, len(sections[partIndex]))
		for index, draft := range sections[partIndex] {
			sectionInputs[index] = partassembly.SectionDraft{Title: draft.Title, Markdown: draft.Markdown, ArtifactID: draft.ArtifactID, WordCount: draft.WordCount}
		}
		out, err := runner.partAssemblyRunner.Run(ctx, partassembly.Input{
			Base: base, Part: part, PartIndex: partIndex, Sections: sectionInputs,
			ToolSessionID: runner.id("ses"), PreviousSessionID: previousSessionID, ForkSourceSessionID: forkSourceID,
		})
		if err != nil {
			return nil, nil, "", err
		}
		partDrafts = append(partDrafts, out.Draft)
		partArtifactIDs = append(partArtifactIDs, out.Draft.ArtifactID)
		if !fanout {
			currentSessionID = out.Draft.SessionID
		}
	}
	return partDrafts, partArtifactIDs, currentSessionID, nil
}

func (runner Runner) runPartEdits(ctx context.Context, input DraftInput, planOut plan.LongFormOutput, reqOut requirements.Output, progress longFormProgress, parts []partassembly.PartDraft, effort string) ([]partassembly.PartDraft, []string, error) {
	editedParts := make([]partassembly.PartDraft, 0, len(parts))
	editedArtifactIDs := make([]string, 0, len(parts))
	forker, canFork := runner.partEditRunner.Executor.(agentexec.AgentSessionForker)
	for partIndex, source := range parts {
		if edited, ok := progress.editedParts[partIndex]; ok {
			editedParts = append(editedParts, partassembly.PartDraft{Title: edited.Title, Markdown: edited.Markdown, ArtifactID: edited.ArtifactID, WordCount: edited.WordCount})
			editedArtifactIDs = append(editedArtifactIDs, edited.ArtifactID)
			continue
		}
		editInput := partedit.Input{
			Base: partEditBase(input, planOut, reqOut, effort),
			Part: planOut.Plan.Parts[partIndex], PartIndex: partIndex,
			Source:                   partedit.PartDraft{Title: source.Title, Markdown: source.Markdown, ArtifactID: source.ArtifactID, WordCount: source.WordCount},
			ForkSourceAgentSessionID: planOut.ReportPlanSessionID,
		}
		expected := ""
		if planOut.PartPlanningEnabled {
			partPlan, ok := progress.partPlans[partIndex]
			if !ok || partPlan.ProviderSessionID == "" || partPlan.Brief == "" {
				return nil, nil, longformutil.StageFailure("part_edit", planOut.Event.EventID, partIndex+1, 0,
					fmt.Errorf("%w: final Part author session is missing", producterror.ErrConflict))
			}
			editInput.AuthorMode = true
			editInput.PartPlanningBrief = partPlan.Brief
			editInput.PreviousSessionID = partPlan.ProviderSessionID
			expected = partPlan.ProviderSessionID
		}
		recovered, ok, err := runner.partEditRunner.CurrentStart(ctx, editInput, expected)
		if err != nil {
			return nil, nil, longformutil.StageFailure("part_edit", planOut.Event.EventID, partIndex+1, 0, err)
		}
		if ok {
			editInput.ToolSessionID = recovered.ToolSessionID
			editInput.PreviousSessionID = recovered.ProviderSessionID
			editInput.EditedArtifactID = recovered.EditedArtifactID
			editInput.Filename = recovered.Filename
			editInput.ForkSourceAgentSessionID = recovered.ForkSourceAgentSessionID
		} else {
			if !editInput.AuthorMode {
				if !canFork {
					return nil, nil, longformutil.StageFailure("part_edit", planOut.Event.EventID, partIndex+1, 0,
						fmt.Errorf("%w: Part editor requires an agent session forker", producterror.ErrInvalidInput))
				}
				previous, forkSource, err := forkLongFormSession(ctx, forker, planOut.ReportPlanSessionID)
				if err != nil {
					return nil, nil, longformutil.StageFailure("part_edit", planOut.Event.EventID, partIndex+1, 0, err)
				}
				editInput.PreviousSessionID = previous
				editInput.ForkSourceAgentSessionID = firstNonEmpty(forkSource, planOut.ReportPlanSessionID)
			}
			editInput.ToolSessionID = runner.id("ses")
			editInput.EditedArtifactID = runner.id("art")
			editInput.Filename = longformutil.SafeFilename(fmt.Sprintf("%s part %02d edited", input.Title, partIndex+1), ".md")
		}
		started := time.Now()
		out, err := runner.partEditRunner.Run(ctx, editInput)
		if err != nil {
			return nil, nil, longformutil.StageFailure("part_edit", planOut.Event.EventID, partIndex+1, 0,
				longformutil.AgentFailure(err, out.Result, "report_part_edit", time.Since(started).Milliseconds(), editInput.PreviousSessionID))
		}
		if out.Draft.ArtifactID == "" {
			message := "edited Part artifact is missing"
			if editInput.AuthorMode {
				message = "final Part author artifact is missing"
			}
			return nil, nil, longformutil.StageFailure("part_edit", planOut.Event.EventID, partIndex+1, 0,
				fmt.Errorf("%w: %s", producterror.ErrConflict, message))
		}
		editedParts = append(editedParts, partassembly.PartDraft{Title: out.Draft.Title, Markdown: out.Draft.Markdown, ArtifactID: out.Draft.ArtifactID, WordCount: out.Draft.WordCount})
		editedArtifactIDs = append(editedArtifactIDs, out.Draft.ArtifactID)
	}
	return editedParts, editedArtifactIDs, nil
}

func partSourceSession(planOut plan.LongFormOutput, progress longFormProgress, partIndex int) (string, error) {
	if !planOut.PartPlanningEnabled {
		return planOut.ReportPlanSessionID, nil
	}
	partPlan, ok := progress.partPlans[partIndex]
	if !ok || partPlan.ProviderSessionID == "" {
		return "", fmt.Errorf("%w: Part planning session is missing", producterror.ErrConflict)
	}
	return partPlan.ProviderSessionID, nil
}
