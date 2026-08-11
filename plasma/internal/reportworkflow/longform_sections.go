package reportworkflow

import (
	"context"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/longformutil"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/plan"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/sectiondraft"
)

func (runner Runner) runSerialSections(ctx context.Context, base sectiondraft.BaseInput, planOut plan.LongFormOutput, progress longFormProgress) ([][]sectiondraft.Draft, []string, int, string, error) {
	sectionsByPart := make([][]sectiondraft.Draft, 0, len(planOut.Plan.Parts))
	sectionArtifactIDs := []string{}
	wordTotal := 0
	currentSessionID := firstNonEmpty(progress.currentSessionID, planOut.ReportPlanSessionID)
	for partIndex, part := range planOut.Plan.Parts {
		if recoveredPart, ok := progress.parts[partIndex]; ok {
			recoveredSections, err := recoveredSectionsForPart(progress, partIndex, len(part.Sections))
			if err != nil {
				return nil, nil, 0, "", err
			}
			for _, draft := range recoveredSections {
				sectionArtifactIDs = append(sectionArtifactIDs, draft.ArtifactID)
				wordTotal += draft.WordCount
			}
			if len(recoveredSections) == 0 {
				wordTotal += recoveredPart.WordCount
			}
			sectionsByPart = append(sectionsByPart, nil)
			continue
		}
		partDrafts := make([]sectiondraft.Draft, 0, len(part.Sections))
		for sectionIndex, section := range part.Sections {
			if draft, ok := progress.sections[sectionIndexKey(partIndex, sectionIndex)]; ok {
				partDrafts = append(partDrafts, draft)
				sectionArtifactIDs = append(sectionArtifactIDs, draft.ArtifactID)
				wordTotal += draft.WordCount
				currentSessionID = firstNonEmpty(draft.SessionID, currentSessionID)
				continue
			}
			key := sectionIndexKey(partIndex, sectionIndex)
			recoveredGap, hasRecoveredGap := progress.sectionGaps[key]
			if hasRecoveredGap && recoveredGap.Attempt >= sectiondraft.MaxEvidenceGapAttempts {
				return nil, nil, 0, "", sectionEvidenceGapFailure(base.PlanEvent.EventID, recoveredGap)
			}
			toolSessionID := ""
			previous := currentSessionID
			attempt := 1
			if hasRecoveredGap {
				if !sameRecoveredSourceSession(recoveredGap.SourceSessionID, planOut.ForkSourceSessionID) {
					return nil, nil, 0, "", sectionEvidenceGapSourceFailure(base.PlanEvent.EventID, recoveredGap)
				}
				toolSessionID = recoveredGap.ToolSessionID
				previous = firstNonEmpty(recoveredGap.SessionID, recoveredGap.ReturnedSessionID, currentSessionID)
				attempt = recoveredGap.Attempt + 1
			}
			if toolSessionID == "" {
				toolSessionID = runner.id("ses")
			}
			out, err := runner.runSectionDraftWithEvidenceGapRetry(ctx, sectiondraft.Input{
				Base: base, Part: part, Section: section, PartIndex: partIndex, SectionIndex: sectionIndex,
				ToolSessionID: toolSessionID, PreviousSessionID: previous, SourceSessionID: planOut.ForkSourceSessionID,
				Attempt: attempt, UserText: fmt.Sprintf("draft section %d.%d for sectional long-form markdown report", partIndex+1, sectionIndex+1),
				CreatedText: "장문 리포트 섹션 Markdown을 생성했습니다.",
			})
			if err != nil {
				return nil, nil, 0, "", err
			}
			currentSessionID = out.Draft.SessionID
			partDrafts = append(partDrafts, out.Draft)
			sectionArtifactIDs = append(sectionArtifactIDs, out.Draft.ArtifactID)
			wordTotal += out.Draft.WordCount
		}
		sectionsByPart = append(sectionsByPart, partDrafts)
	}
	return sectionsByPart, sectionArtifactIDs, wordTotal, currentSessionID, nil
}

func (runner Runner) runFanoutSections(ctx context.Context, base sectiondraft.BaseInput, planOut plan.LongFormOutput, progress longFormProgress) ([][]sectiondraft.Draft, []string, int, error) {
	sectionsByPart := make([][]sectiondraft.Draft, len(planOut.Plan.Parts))
	sectionArtifactIDs := []string{}
	wordTotal := 0
	type task struct {
		partIndex, sectionIndex int
		input                   sectiondraft.Input
	}
	tasks := []task{}
	forker, ok := runner.sectionDraftRunner.Executor.(agentexec.AgentSessionForker)
	if !ok {
		return nil, nil, 0, fmt.Errorf("%w: section fanout requires an agent session forker", producterror.ErrInvalidInput)
	}
	for partIndex, part := range planOut.Plan.Parts {
		if recoveredPart, ok := progress.parts[partIndex]; ok {
			recoveredSections, err := recoveredSectionsForPart(progress, partIndex, len(part.Sections))
			if err != nil {
				return nil, nil, 0, err
			}
			for _, draft := range recoveredSections {
				sectionArtifactIDs = append(sectionArtifactIDs, draft.ArtifactID)
				wordTotal += draft.WordCount
			}
			if len(recoveredSections) == 0 {
				wordTotal += recoveredPart.WordCount
			}
			continue
		}
		sectionsByPart[partIndex] = make([]sectiondraft.Draft, len(part.Sections))
		partSourceSessionID, err := partSourceSession(planOut, progress, partIndex)
		if err != nil {
			return nil, nil, 0, err
		}
		for sectionIndex, section := range part.Sections {
			if draft, ok := progress.sections[sectionIndexKey(partIndex, sectionIndex)]; ok {
				sectionsByPart[partIndex][sectionIndex] = draft
				continue
			}
			key := sectionIndexKey(partIndex, sectionIndex)
			if recoveredGap, ok := progress.sectionGaps[key]; ok {
				if recoveredGap.Attempt >= sectiondraft.MaxEvidenceGapAttempts {
					return nil, nil, 0, sectionEvidenceGapFailure(base.PlanEvent.EventID, recoveredGap)
				}
				if !sameRecoveredSourceSession(recoveredGap.SourceSessionID, partSourceSessionID) {
					return nil, nil, 0, sectionEvidenceGapSourceFailure(base.PlanEvent.EventID, recoveredGap)
				}
				tasks = append(tasks, task{partIndex: partIndex, sectionIndex: sectionIndex, input: sectiondraft.Input{
					Base: base, Part: part, Section: section, PartIndex: partIndex, SectionIndex: sectionIndex,
					Attempt:           recoveredGap.Attempt + 1,
					ToolSessionID:     recoveredGap.ToolSessionID,
					PreviousSessionID: firstNonEmpty(recoveredGap.SessionID, recoveredGap.ReturnedSessionID),
					SourceSessionID:   partSourceSessionID,
					UserText:          SectionFanoutSectionUserText(partIndex, sectionIndex),
					CreatedText:       "장문 리포트 섹션 Markdown을 병렬 생성했습니다.",
				}})
				continue
			}
			previous, source, err := forkLongFormSession(ctx, forker, partSourceSessionID)
			if err != nil {
				return nil, nil, 0, longformutil.StageFailure("section", planOut.Event.EventID, partIndex+1, sectionIndex+1, err)
			}
			tasks = append(tasks, task{partIndex: partIndex, sectionIndex: sectionIndex, input: sectiondraft.Input{
				Base: base, Part: part, Section: section, PartIndex: partIndex, SectionIndex: sectionIndex,
				ToolSessionID: runner.id("ses"), PreviousSessionID: previous, SourceSessionID: firstNonEmpty(source, partSourceSessionID),
				UserText:     SectionFanoutSectionUserText(partIndex, sectionIndex),
				StartedEvent: true, CreatedText: "장문 리포트 섹션 Markdown을 병렬 생성했습니다.",
			}})
		}
	}
	err := runLimited(len(tasks), longFormWorkerLimit, func(i int) error {
		out, err := runner.runSectionDraftWithEvidenceGapRetry(ctx, tasks[i].input)
		if err != nil {
			return err
		}
		task := tasks[i]
		sectionsByPart[task.partIndex][task.sectionIndex] = out.Draft
		return nil
	})
	if err != nil {
		return nil, nil, 0, err
	}
	for partIndex, part := range planOut.Plan.Parts {
		if _, ok := progress.parts[partIndex]; ok {
			continue
		}
		for sectionIndex := range part.Sections {
			draft := sectionsByPart[partIndex][sectionIndex]
			if draft.ArtifactID == "" {
				return nil, nil, 0, fmt.Errorf("%w: section fanout left a section incomplete", producterror.ErrConflict)
			}
			sectionArtifactIDs = append(sectionArtifactIDs, draft.ArtifactID)
			wordTotal += draft.WordCount
		}
	}
	return sectionsByPart, sectionArtifactIDs, wordTotal, nil
}

func (runner Runner) runSectionDraftWithEvidenceGapRetry(ctx context.Context, input sectiondraft.Input) (sectiondraft.Output, error) {
	input.Attempt = normalizeSectionAttempt(input.Attempt)
	if input.Attempt > sectiondraft.MaxEvidenceGapAttempts {
		return sectiondraft.Output{}, sectionEvidenceGapFailure(input.Base.PlanEvent.EventID, sectiondraft.EvidenceGap{
			PartIndex: input.PartIndex + 1, SectionIndex: input.SectionIndex + 1, Attempt: input.Attempt,
			ReasonCode: sectiondraft.EvidenceGapReasonCode,
		})
	}
	for {
		out, err := runner.sectionDraftRunner.Run(ctx, input)
		if err != nil {
			return sectiondraft.Output{}, err
		}
		if out.EvidenceGap == nil {
			return out, nil
		}
		gap := *out.EvidenceGap
		if gap.Attempt >= sectiondraft.MaxEvidenceGapAttempts {
			return sectiondraft.Output{}, sectionEvidenceGapFailure(input.Base.PlanEvent.EventID, gap)
		}
		input.Attempt = gap.Attempt + 1
		input.PreviousSessionID = firstNonEmpty(gap.SessionID, gap.ReturnedSessionID, input.PreviousSessionID)
		input.ToolSessionID = firstNonEmpty(gap.ToolSessionID, input.ToolSessionID)
		input.SourceSessionID = firstNonEmpty(gap.SourceSessionID, input.SourceSessionID)
		input.StartedEvent = false
	}
}

func sectionEvidenceGapFailure(planEventID string, gap sectiondraft.EvidenceGap) error {
	cause := longformutil.StageFailure("section", planEventID, gap.PartIndex, gap.SectionIndex,
		fmt.Errorf("%w: section evidence remained inadequate after retry", producterror.ErrConflict))
	return &sectionEvidenceGapError{gap: gap, cause: cause}
}

// sectionEvidenceGapError keeps the terminal control outcome distinguishable
// from unrelated Section conflicts without changing the public stage failure.
type sectionEvidenceGapError struct {
	gap   sectiondraft.EvidenceGap
	cause error
}

func (err *sectionEvidenceGapError) Error() string { return err.cause.Error() }

func (err *sectionEvidenceGapError) Unwrap() error { return err.cause }

func sectionEvidenceGapSourceFailure(planEventID string, gap sectiondraft.EvidenceGap) error {
	return longformutil.StageFailure("section", planEventID, gap.PartIndex, gap.SectionIndex,
		fmt.Errorf("%w: recovered section evidence gap source lineage mismatch", producterror.ErrConflict))
}

func sameRecoveredSourceSession(recovered string, canonical string) bool {
	return recovered == canonical
}

func normalizeSectionAttempt(attempt int) int {
	if attempt < 1 {
		return 1
	}
	return attempt
}
