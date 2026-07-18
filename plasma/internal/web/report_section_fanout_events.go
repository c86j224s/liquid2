package web

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func (server *Server) appendSectionFanoutStartedEvent(ctx context.Context, req sectionFanoutLongFormRequest, state sectionFanoutPlanState, task sectionFanoutTask) error {
	sessionID := fallbackSessionID(task.previousSession, task.toolSessionID)
	_, err := server.service.AppendEvent(ctx, reporting.BuildMarkdownReportSectionStartedAppendRequest(reporting.MarkdownReportSectionStartedEventRequest{
		MarkdownReportStageEventBase: reporting.MarkdownReportStageEventBase{
			EventID:                      newID("evt"),
			MissionID:                    req.missionID,
			PendingEventID:               req.pendingEventID,
			PlanEventID:                  state.planEvent.EventID,
			Title:                        task.section.Title,
			AgentExecutor:                req.executorName,
			AgentModel:                   req.agentModel,
			AgentReasoningEffort:         req.agentReasoningEffort,
			AgentSelectionSource:         req.agentSelectionSource,
			AgentSessionID:               sessionID,
			PreviousAgentSessionID:       task.previousSession,
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
			ReportSessionID:              sessionID,
			ForkSourceAgentSessionID:     task.sourceSessionID,
			CompositionStrategy:          "sectional_preserve_markdown",
			AssemblyStrategy:             "c4_normalized_section_headings",
			Text:                         "장문 리포트 섹션 Markdown 생성을 시작했습니다.",
			Producer:                     app.Producer{Type: "agent_session", ID: sessionID},
		},
		PartIndex:    task.partIndex + 1,
		SectionIndex: task.sectionIndex + 1,
	}))
	if err != nil {
		return longFormStageFailure("section", state.planEvent.EventID, task.partIndex+1, task.sectionIndex+1, err)
	}
	return nil
}
