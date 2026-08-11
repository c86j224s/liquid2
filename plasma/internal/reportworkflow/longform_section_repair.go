package reportworkflow

import (
	"context"
	"errors"
	"sort"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/longformutil"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/plan"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/requirements"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/sectiondraft"
)

type longFormSectionStageOutput struct {
	plan        plan.LongFormOutput
	progress    longFormProgress
	sections    [][]sectiondraft.Draft
	artifactIDs []string
	wordTotal   int
}

func (runner Runner) runLongFormSectionStage(ctx context.Context, input DraftInput, planInput plan.LongFormInput, planOut plan.LongFormOutput, reqOut requirements.Output, progress longFormProgress, repairOut plan.LongFormSectionRepairOutput, fanout bool) (longFormSectionStageOutput, error) {
	sections, artifactIDs, wordTotal, currentSessionID, err := runner.runLongFormSectionSet(ctx, input, planOut, reqOut, progress, fanout)
	if err == nil {
		progress.currentSessionID = firstNonEmpty(currentSessionID, progress.currentSessionID)
		return longFormSectionStageOutput{plan: planOut, progress: progress, sections: sections, artifactIDs: artifactIDs, wordTotal: wordTotal}, nil
	}
	var gapFailure *sectionEvidenceGapError
	if !errors.As(err, &gapFailure) || repairOut.Event.EventID != "" {
		return longFormSectionStageOutput{}, err
	}

	// All fan-out workers have finished before runLongFormSectionSet returns.
	// Reload their durable successes and terminal gaps before repairing the plan.
	progress, loadErr := runner.loadLongFormProgress(ctx, input, planOut, plan.LongFormSectionRepairOutput{})
	if loadErr != nil {
		return longFormSectionStageOutput{}, loadErr
	}
	gaps := terminalSectionGapCoordinates(progress)
	if len(gaps) == 0 {
		return longFormSectionStageOutput{}, err
	}
	repairOut, repairErr := runner.planRunner.RunLongFormSectionRepair(ctx, plan.LongFormSectionRepairInput{
		Request: planInput, Plan: planOut, RequirementMap: reqOut.RequirementMap, Gaps: gaps,
	})
	if repairErr != nil {
		first := gaps[0]
		return longFormSectionStageOutput{}, longformutil.StageFailure("section", planOut.Event.EventID, first.PartIndex, first.SectionIndex, repairErr)
	}
	planOut.Plan = repairOut.Plan
	progress, loadErr = runner.loadLongFormProgress(ctx, input, planOut, repairOut)
	if loadErr != nil {
		return longFormSectionStageOutput{}, loadErr
	}
	sections, artifactIDs, wordTotal, currentSessionID, err = runner.runLongFormSectionSet(ctx, input, planOut, reqOut, progress, fanout)
	if err != nil {
		return longFormSectionStageOutput{}, err
	}
	progress.currentSessionID = firstNonEmpty(currentSessionID, progress.currentSessionID)
	return longFormSectionStageOutput{plan: planOut, progress: progress, sections: sections, artifactIDs: artifactIDs, wordTotal: wordTotal}, nil
}

func (runner Runner) runLongFormSectionSet(ctx context.Context, input DraftInput, planOut plan.LongFormOutput, reqOut requirements.Output, progress longFormProgress, fanout bool) ([][]sectiondraft.Draft, []string, int, string, error) {
	base := sectionBase(input, planOut, reqOut)
	if fanout {
		sections, artifactIDs, wordTotal, err := runner.runFanoutSections(ctx, base, planOut, progress)
		return sections, artifactIDs, wordTotal, "", err
	}
	return runner.runSerialSections(ctx, base, planOut, progress)
}

func terminalSectionGapCoordinates(progress longFormProgress) []reporting.ReportSectionCoordinate {
	result := []reporting.ReportSectionCoordinate{}
	for index, gap := range progress.sectionGaps {
		if gap.Attempt >= sectiondraft.MaxEvidenceGapAttempts {
			result = append(result, reporting.ReportSectionCoordinate{PartIndex: index.part + 1, SectionIndex: index.section + 1})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].PartIndex == result[j].PartIndex {
			return result[i].SectionIndex < result[j].SectionIndex
		}
		return result[i].PartIndex < result[j].PartIndex
	})
	return result
}
