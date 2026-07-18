package web

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func (server *Server) draftSectionFanoutSections(ctx context.Context, req sectionFanoutLongFormRequest, state sectionFanoutPlanState, progress sectionalReportProgress, forker AgentSessionForker, executor AgentExecutor) ([][]sectionalReportDraft, []string, int, error) {
	sectionDraftsByPart := make([][]sectionalReportDraft, len(state.plan.Parts))
	sectionArtifactIDs := []string{}
	sectionWordTotal := 0
	tasks := []sectionFanoutTask{}

	for partIndex, part := range state.plan.Parts {
		if recoveredPart, ok := progress.parts[partIndex]; ok {
			recoveredSections := make([]sectionalReportDraft, 0, len(part.Sections))
			for sectionIndex := range part.Sections {
				if draft, exists := progress.sections[sectionalReportIndex{part: partIndex, section: sectionIndex}]; exists {
					recoveredSections = append(recoveredSections, draft)
				}
			}
			if len(recoveredSections) != 0 && len(recoveredSections) != len(part.Sections) {
				return nil, nil, 0, fmt.Errorf("%w: recovered long-form part has partial section provenance", app.ErrConflict)
			}
			for _, draft := range recoveredSections {
				sectionArtifactIDs = append(sectionArtifactIDs, draft.ArtifactID)
				sectionWordTotal += draft.WordCount
			}
			if len(recoveredSections) == 0 {
				sectionWordTotal += recoveredPart.WordCount
			}
			continue
		}

		sectionDraftsByPart[partIndex] = make([]sectionalReportDraft, len(part.Sections))
		for sectionIndex, section := range part.Sections {
			if draft, ok := progress.sections[sectionalReportIndex{part: partIndex, section: sectionIndex}]; ok {
				sectionDraftsByPart[partIndex][sectionIndex] = draft
				continue
			}
			previousSessionID, sourceSessionID, err := forkSectionFanoutSession(ctx, forker, state.reportPlanSessionID)
			if err != nil {
				return nil, nil, 0, longFormStageFailure("section", state.planEvent.EventID, partIndex+1, sectionIndex+1, err)
			}
			tasks = append(tasks, sectionFanoutTask{
				partIndex:       partIndex,
				sectionIndex:    sectionIndex,
				part:            part,
				section:         section,
				previousSession: previousSessionID,
				sourceSessionID: firstNonEmpty(sourceSessionID, state.reportPlanSessionID),
				toolSessionID:   newID("ses"),
			})
		}
	}

	results, err := server.runSectionFanoutTasks(ctx, req, state, tasks, executor)
	if err != nil {
		return nil, nil, 0, err
	}
	for _, item := range results {
		task := item.task
		sectionDraftsByPart[task.partIndex][task.sectionIndex] = item.draft
	}

	for partIndex, part := range state.plan.Parts {
		if _, ok := progress.parts[partIndex]; ok {
			continue
		}
		for sectionIndex := range part.Sections {
			draft := sectionDraftsByPart[partIndex][sectionIndex]
			if strings.TrimSpace(draft.ArtifactID) == "" {
				return nil, nil, 0, fmt.Errorf("%w: section fanout left a section incomplete", app.ErrConflict)
			}
			sectionArtifactIDs = append(sectionArtifactIDs, draft.ArtifactID)
			sectionWordTotal += draft.WordCount
		}
	}
	return sectionDraftsByPart, sectionArtifactIDs, sectionWordTotal, nil
}

func (server *Server) persistSectionFanoutResult(ctx context.Context, req sectionFanoutLongFormRequest, state sectionFanoutPlanState, item sectionFanoutResult) (sectionFanoutResult, error) {
	task := item.task
	artifact, err := server.service.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: newID("art"),
		MissionID:  req.missionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   safeFilename(fmt.Sprintf("%s part %02d section %02d", req.title, task.partIndex+1, task.sectionIndex+1), ".md"),
		Producer:   app.Producer{Type: "agent_session", ID: fallbackSessionID(item.result.SessionID, task.toolSessionID)},
		Content:    []byte(item.markdown),
	})
	if err != nil {
		return sectionFanoutResult{}, longFormStageFailure("section", state.planEvent.EventID, task.partIndex+1, task.sectionIndex+1, err)
	}
	wordCount := reportWordCount(item.markdown)
	_, err = server.service.AppendEvent(ctx, reporting.BuildMarkdownReportSectionCreatedAppendRequest(reporting.MarkdownReportSectionCreatedEventRequest{
		MarkdownReportStageEventBase: reporting.MarkdownReportStageEventBase{
			EventID:                      newID("evt"),
			MissionID:                    req.missionID,
			PendingEventID:               req.pendingEventID,
			PlanEventID:                  state.planEvent.EventID,
			Title:                        task.section.Title,
			Artifact:                     artifact,
			AgentExecutor:                req.executorName,
			AgentModel:                   req.agentModel,
			AgentReasoningEffort:         req.agentReasoningEffort,
			AgentSelectionSource:         req.agentSelectionSource,
			AgentSessionID:               item.result.SessionID,
			PreviousAgentSessionID:       task.previousSession,
			ReturnedAgentSessionID:       item.returnedSessionID,
			ToolSessionID:                task.toolSessionID,
			ReportMode:                   reportModeLongForm,
			ReportModeLabel:              reportModeLabel(reportModeLongForm),
			ReportSessionPolicy:          state.reportSessionPolicy,
			ReportSessionPolicySelection: state.reportSessionPolicySelection,
			PostReportHumanize:           req.postReportHumanize,
			HumanizeEnabled:              req.postReportHumanize != "disabled",
			GenerationGuidanceProfile:    req.generationGuidanceProfile,
			GenerationGuidanceSHA256:     req.generationGuidanceSHA256,
			SessionChainKind:             state.sessionChainKind,
			PreReportResearchSessionID:   state.preReportResearchSessionID,
			ReportPlanSessionID:          state.reportPlanSessionID,
			ReportSessionID:              item.result.SessionID,
			ForkSourceAgentSessionID:     task.sourceSessionID,
			CompositionStrategy:          "sectional_preserve_markdown",
			AssemblyStrategy:             "c4_normalized_section_headings",
			DurationMS:                   item.durationMS,
			Text:                         "장문 리포트 섹션 Markdown을 병렬 생성했습니다.",
			AgentUsage:                   item.result.Usage,
			AgentUsageSurface:            "report_section",
			AgentUsageDurationMS:         item.durationMS,
			AgentResumed:                 item.result.Resumed,
			Producer:                     app.Producer{Type: "agent_session", ID: fallbackSessionID(item.result.SessionID, task.toolSessionID)},
		},
		PartIndex:    task.partIndex + 1,
		SectionIndex: task.sectionIndex + 1,
		WordCount:    wordCount,
	}))
	if err != nil {
		return sectionFanoutResult{}, longFormStageFailure("section", state.planEvent.EventID, task.partIndex+1, task.sectionIndex+1, err)
	}
	item.draft = sectionalReportDraft{Title: task.section.Title, Markdown: item.markdown, ArtifactID: artifact.ArtifactID, WordCount: wordCount}
	return item, nil
}
