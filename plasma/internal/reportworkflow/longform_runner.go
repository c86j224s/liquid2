package reportworkflow

import (
	"context"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/plan"
)

// RunLongFormPrefix는 finalization tail 직전까지 long-form prefix graph를 typed stage로 실행한다.
func (runner Runner) RunLongFormPrefix(ctx context.Context, input DraftInput) (PrefixOutput, error) {
	family, err := SelectFamily(input.ReportMode, input.ExecutionStrategy)
	if err != nil {
		return PrefixOutput{}, err
	}
	if family != FamilyLongFormSerial && family != FamilyLongFormSectionFanout {
		return PrefixOutput{}, fmt.Errorf("%w: long-form prefix requires long_form mode", producterror.ErrInvalidInput)
	}
	fanout := family == FamilyLongFormSectionFanout
	planInput := plan.LongFormInput{Input: planInput(input), SectionFanout: fanout}
	donePlan := runner.observeStart(NodePlan)
	planOut, err := runner.planRunner.RunLongForm(ctx, planInput)
	donePlan(err, planOut.Recovered)
	if err != nil {
		return PrefixOutput{}, err
	}
	repairOut, repaired, err := runner.planRunner.RecoverLongFormSectionRepair(ctx, planInput, planOut)
	if err != nil {
		return PrefixOutput{}, err
	}
	if repaired {
		planOut.Plan = repairOut.Plan
	}
	progress, err := runner.loadLongFormProgress(ctx, input, planOut, repairOut)
	if err != nil {
		return PrefixOutput{}, err
	}
	doneRequirements := runner.observeStart(NodeRequirements)
	reqOut, err := runner.requirementsRunner.Run(ctx, requirementsInput(input, planOut, progress))
	doneRequirements(err, reqOut.Recovered)
	if err != nil {
		return PrefixOutput{}, err
	}
	if planOut.PartPlanningEnabled {
		done := runner.observeStart(NodePartPlan)
		progress.partPlans, err = runner.runPartPlans(ctx, input, planOut, reqOut, progress)
		done(err, false)
		if err != nil {
			return PrefixOutput{}, err
		}
	}
	planOut = fillPlanRuntime(input, planOut)
	doneSection := runner.observeStart(NodeSectionDraft)
	sectionOut, err := runner.runLongFormSectionStage(ctx, input, planInput, planOut, reqOut, progress, repairOut, fanout)
	doneSection(err, false)
	if err != nil {
		return PrefixOutput{}, err
	}
	planOut = sectionOut.plan
	progress = sectionOut.progress
	sections := sectionOut.sections
	sectionArtifactIDs := sectionOut.artifactIDs
	sectionWordTotal := sectionOut.wordTotal
	baseParts := partAssemblyBase(input, planOut)
	donePart := runner.observeStart(NodePartAssembly)
	parts, partArtifactIDs, currentSessionID, err := runner.runPartAssemblies(ctx, baseParts, planOut, progress, sections, fanout)
	donePart(err, false)
	if err != nil {
		return PrefixOutput{}, err
	}
	if currentSessionID != "" {
		progress.currentSessionID = currentSessionID
	}
	finalReasoningEffort := input.AgentReasoningEffort
	if planOut.FinalEditPipeline != "" {
		finalReasoningEffort = firstNonEmpty(input.AgentReasoningEffort, "default")
	}
	if planOut.PartEditEnabled {
		doneEdit := runner.observeStart(NodePartEdit)
		parts, partArtifactIDs, err = runner.runPartEdits(ctx, input, planOut, reqOut, progress, parts, finalReasoningEffort)
		doneEdit(err, false)
		if err != nil {
			return PrefixOutput{}, err
		}
	}
	return runner.prefixOutput(input, planOut, reqOut, sections, parts, sectionArtifactIDs, partArtifactIDs, sectionWordTotal, finalReasoningEffort, progress.currentSessionID)
}
