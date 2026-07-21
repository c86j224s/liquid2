package web

import (
	"context"
	"fmt"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func (server *Server) assembleSectionFanoutParts(ctx context.Context, req sectionFanoutLongFormRequest, state sectionFanoutPlanState, progress sectionalReportProgress, sections [][]sectionalReportDraft, forker AgentSessionForker, executor AgentExecutor) ([]sectionalReportPartDraft, []string, error) {
	partDrafts := make([]sectionalReportPartDraft, 0, len(state.plan.Parts))
	partArtifactIDs := []string{}
	for partIndex, part := range state.plan.Parts {
		if draft, ok := progress.parts[partIndex]; ok {
			partDrafts = append(partDrafts, draft)
			partArtifactIDs = append(partArtifactIDs, draft.ArtifactID)
			continue
		}
		previousSessionID, sourceSessionID, err := forkSectionFanoutSession(ctx, forker, state.reportPlanSessionID)
		if err != nil {
			return nil, nil, longFormStageFailure("part", state.planEvent.EventID, partIndex+1, 0, err)
		}
		toolSessionID := newID("ses")
		partStarted := time.Now()
		assembly, result, returnedSessionID, err := server.runPartAssemblyAgent(ctx, reportPartAssemblyAgentRequest{
			title:                        req.title,
			missionID:                    req.missionID,
			toolSessionID:                toolSessionID,
			previousSessionID:            previousSessionID,
			pendingEventID:               req.pendingEventID,
			planEventID:                  state.planEvent.EventID,
			executorName:                 req.executorName,
			agentModel:                   req.agentModel,
			agentReasoningEffort:         req.agentReasoningEffort,
			agentSelectionSource:         req.agentSelectionSource,
			mcpMode:                      req.mcpMode,
			rigor:                        req.rigor,
			plan:                         state.plan,
			part:                         part,
			drafts:                       sections[partIndex],
			partIndex:                    partIndex,
			reportSessionPolicy:          state.reportSessionPolicy,
			reportSessionPolicySelection: state.reportSessionPolicySelection,
			postReportHumanize:           req.postReportHumanize,
			generationGuidanceProfile:    req.generationGuidanceProfile,
			generationGuidanceSHA256:     req.generationGuidanceSHA256,
			sessionChainKind:             state.sessionChainKind,
			preReportResearchSessionID:   state.preReportResearchSessionID,
			reportPlanSessionID:          state.reportPlanSessionID,
			forkSourceAgentSessionID:     firstNonEmpty(sourceSessionID, state.reportPlanSessionID),
		}, executor)
		partDurationMS := time.Since(partStarted).Milliseconds()
		if err != nil {
			return nil, nil, longFormStageFailure("part", state.planEvent.EventID, partIndex+1, 0, reportAgentFailure(err, result, "report_part", partDurationMS, previousSessionID))
		}
		partMarkdown := assembleSectionalPartMarkdown(part, sections[partIndex], assembly, partIndex)
		artifact, err := server.service.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
			ArtifactID: newID("art"),
			MissionID:  req.missionID,
			MediaType:  "text/markdown; charset=utf-8",
			Filename:   safeFilename(fmt.Sprintf("%s part %02d", req.title, partIndex+1), ".md"),
			Producer:   app.Producer{Type: "agent_session", ID: fallbackSessionID(result.SessionID, toolSessionID)},
			Content:    []byte(partMarkdown),
		})
		if err != nil {
			return nil, nil, longFormStageFailure("part", state.planEvent.EventID, partIndex+1, 0, err)
		}
		partWordCount := reportWordCount(partMarkdown)
		_, err = server.service.AppendEvent(ctx, reporting.BuildMarkdownReportPartCreatedAppendRequest(reporting.MarkdownReportPartCreatedEventRequest{
			MarkdownReportStageEventBase: reporting.MarkdownReportStageEventBase{
				EventID:                      newID("evt"),
				MissionID:                    req.missionID,
				PendingEventID:               req.pendingEventID,
				PlanEventID:                  state.planEvent.EventID,
				Title:                        part.Title,
				Artifact:                     artifact,
				AgentExecutor:                req.executorName,
				AgentModel:                   req.agentModel,
				AgentReasoningEffort:         req.agentReasoningEffort,
				AgentSelectionSource:         req.agentSelectionSource,
				AgentSessionID:               result.SessionID,
				PreviousAgentSessionID:       previousSessionID,
				ReturnedAgentSessionID:       returnedSessionID,
				ToolSessionID:                toolSessionID,
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
				ReportSessionID:              result.SessionID,
				ForkSourceAgentSessionID:     firstNonEmpty(sourceSessionID, state.reportPlanSessionID),
				CompositionStrategy:          "sectional_preserve_markdown",
				AssemblyStrategy:             "c4_normalized_section_headings",
				DurationMS:                   partDurationMS,
				Text:                         "장문 리포트 파트 Markdown을 보존 조립했습니다.",
				AgentUsage:                   result.Usage,
				AgentUsageSurface:            "report_part",
				AgentUsageDurationMS:         partDurationMS,
				AgentResumed:                 result.Resumed,
				Producer:                     app.Producer{Type: "agent_session", ID: fallbackSessionID(result.SessionID, toolSessionID)},
			},
			PartIndex:    partIndex + 1,
			SectionCount: len(sections[partIndex]),
			WordCount:    partWordCount,
		}))
		if err != nil {
			return nil, nil, longFormStageFailure("part", state.planEvent.EventID, partIndex+1, 0, err)
		}
		partArtifactIDs = append(partArtifactIDs, artifact.ArtifactID)
		partDrafts = append(partDrafts, sectionalReportPartDraft{Title: part.Title, Markdown: partMarkdown, ArtifactID: artifact.ArtifactID, WordCount: partWordCount})
	}
	return partDrafts, partArtifactIDs, nil
}
