package web

import (
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partassembly"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partedit"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partplan"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/sectiondraft"
)

type sectionFanoutTask struct {
	partIndex       int
	sectionIndex    int
	part            agentReportPart
	section         agentReportSection
	previousSession string
	toolSessionID   string
	sourceSessionID string
}

func sectionDraftPromptInput(title, missionID, toolSessionID string, rigor reportRigorProfile, plan agentSectionalReportPlan, part agentReportPart, section agentReportSection, partIndex, sectionIndex int, profile string) sectiondraft.Input {
	return sectiondraft.Input{
		Base: sectiondraft.BaseInput{
			MissionID: missionID, Title: title, Rigor: reportWorkflowRigor(rigor),
			GenerationGuidanceProfile: profile, Plan: plan,
		},
		Part: part, Section: section, PartIndex: partIndex, SectionIndex: sectionIndex,
		ToolSessionID: toolSessionID,
	}
}

func sectionFanoutDraftInput(req sectionFanoutLongFormRequest, state sectionFanoutPlanState, task sectionFanoutTask) sectiondraft.Input {
	return sectiondraft.Input{
		Base: sectiondraft.BaseInput{
			MissionID: req.missionID, PendingEventID: req.pendingEventID, Title: req.title,
			DirectionHint: req.directionHint, AgentExecutor: req.executorName, AgentModel: req.agentModel,
			AgentReasoningEffort: req.agentReasoningEffort, AgentSelectionSource: req.agentSelectionSource,
			MCPMode: req.mcpMode, Rigor: reportWorkflowRigor(req.rigor),
			ReportSessionPolicy: state.reportSessionPolicy, ReportSessionPolicySelection: state.reportSessionPolicySelection,
			PostReportHumanize: req.postReportHumanize, GenerationGuidanceProfile: req.generationGuidanceProfile,
			GenerationGuidanceSHA256: req.generationGuidanceSHA256, Plan: state.plan,
			PlanEvent: ledger.Event{EventID: state.planEvent.EventID}, ReportPlanSessionID: state.reportPlanSessionID,
			SessionChainKind: state.sessionChainKind, PreReportResearchSessionID: state.preReportResearchSessionID,
			ForkSourceSessionID: task.sourceSessionID, RequirementMap: state.requirementMap,
		},
		Part: task.part, Section: task.section, PartIndex: task.partIndex, SectionIndex: task.sectionIndex,
		ToolSessionID: task.toolSessionID, PreviousSessionID: task.previousSession,
		SourceSessionID: task.sourceSessionID, StartedEvent: true,
		UserText:    reportworkflow.SectionFanoutSectionUserText(task.partIndex, task.sectionIndex),
		CreatedText: "장문 리포트 섹션 Markdown을 병렬 생성했습니다.",
	}
}

func partAssemblyInput(req reportPartAssemblyAgentRequest) partassembly.Input {
	return partassembly.Input{
		Base: partassembly.BaseInput{
			MissionID: req.missionID, PendingEventID: req.pendingEventID, Title: req.title,
			DirectionHint: req.directionHint, AgentExecutor: req.executorName, AgentModel: req.agentModel,
			AgentReasoningEffort: req.agentReasoningEffort, AgentSelectionSource: req.agentSelectionSource,
			MCPMode: req.mcpMode, Rigor: reportWorkflowRigor(req.rigor),
			ReportSessionPolicy: req.reportSessionPolicy, ReportSessionPolicySelection: req.reportSessionPolicySelection,
			PostReportHumanize: req.postReportHumanize, GenerationGuidanceProfile: req.generationGuidanceProfile,
			GenerationGuidanceSHA256: req.generationGuidanceSHA256, Plan: req.plan,
			PlanEvent: ledger.Event{EventID: req.planEventID}, ReportPlanSessionID: req.reportPlanSessionID,
			SessionChainKind: req.sessionChainKind, PreReportResearchSessionID: req.preReportResearchSessionID,
		},
		Part: req.part, PartIndex: req.partIndex, Sections: partAssemblySections(req.drafts),
		ToolSessionID: req.toolSessionID, PreviousSessionID: req.previousSessionID,
		ForkSourceSessionID: req.forkSourceAgentSessionID,
	}
}

func partAssemblySections(drafts []sectionalReportDraft) []partassembly.SectionDraft {
	sections := make([]partassembly.SectionDraft, len(drafts))
	for index, draft := range drafts {
		sections[index] = partassembly.SectionDraft{
			Title: draft.Title, Markdown: draft.Markdown, ArtifactID: draft.ArtifactID, WordCount: draft.WordCount,
		}
	}
	return sections
}

func partEditInput(req reportPartEditorRequest, authorMode bool, brief string) partedit.Input {
	return partedit.Input{
		Base: partedit.BaseInput{
			MissionID: req.missionID, PendingEventID: req.pendingEventID, Title: req.title,
			DirectionHint: req.directionHint, AgentExecutor: req.executorName, AgentModel: req.agentModel,
			AgentReasoningEffort: req.agentReasoningEffort, AgentSelectionSource: req.agentSelectionSource,
			MCPMode: req.mcpMode, Rigor: reportWorkflowRigor(req.rigor),
			ReportSessionPolicy: req.reportSessionPolicy, ReportSessionPolicySelection: req.reportSessionPolicySelection,
			GenerationGuidanceProfile: req.generationGuidanceProfile, GenerationGuidanceSHA256: req.generationGuidanceSHA256,
			Plan: req.plan, PlanEvent: ledger.Event{EventID: req.planEventID},
			RequirementMap: req.requirementMap, RequirementMapEvent: req.requirementMapEvent,
			ReportPlanSessionID: req.reportPlanSessionID, SessionChainKind: req.sessionChainKind,
		},
		Part: req.part, PartIndex: req.partIndex,
		Source: partedit.PartDraft{
			Title: req.source.Title, Markdown: req.source.Markdown, ArtifactID: req.source.ArtifactID, WordCount: req.source.WordCount,
		},
		ToolSessionID: req.toolSessionID, PreviousSessionID: req.previousSessionID,
		EditedArtifactID: req.editedArtifactID, Filename: req.filename,
		ForkSourceAgentSessionID: req.forkSourceAgentSessionID,
		PartPlanningBrief:        brief, AuthorMode: authorMode,
	}
}

func partPlanInput(req sectionFanoutLongFormRequest, state sectionFanoutPlanState, task sectionFanoutPartPlanTask) partplan.Input {
	return partplan.Input{
		Base: partplan.BaseInput{
			MissionID: req.missionID, PendingEventID: req.pendingEventID, Title: req.title,
			DirectionHint: req.directionHint, AgentExecutor: firstNonEmpty(state.agentExecutor, req.executorName),
			AgentModel: state.agentModel, AgentReasoningEffort: state.agentReasoningEffort,
			AgentSelectionSource: state.agentSelectionSource, MCPMode: req.mcpMode, Rigor: reportWorkflowRigor(req.rigor),
			ReportSessionPolicy: state.reportSessionPolicy, ReportSessionPolicySelection: state.reportSessionPolicySelection,
			PostReportHumanize: req.postReportHumanize, GenerationGuidanceProfile: state.generationGuidanceProfile,
			GenerationGuidanceSHA256: state.generationGuidanceSHA256, Plan: state.plan,
			PlanEvent: ledger.Event{EventID: state.planEvent.EventID}, RequirementMap: state.requirementMap,
			ReportPlanSessionID: state.reportPlanSessionID, SessionChainKind: state.sessionChainKind,
			PreReportResearchSessionID: state.preReportResearchSessionID,
		},
		Part: task.part, PartIndex: task.partIndex,
		ProviderSessionID: task.providerSession, ForkSourceSession: task.forkSourceSession,
	}
}

func partEditDraft(out partedit.Output) sectionalReportPartDraft {
	return sectionalReportPartDraft{
		Title: out.Draft.Title, Markdown: out.Draft.Markdown,
		ArtifactID: out.Draft.ArtifactID, WordCount: out.Draft.WordCount,
	}
}

func hasRecoveredSection(progress sectionalReportProgress, partIndex int) bool {
	for key := range progress.sections {
		if key.part == partIndex {
			return true
		}
	}
	return false
}

func sectionFanoutPartSourceSession(state sectionFanoutPlanState, partIndex int) (string, error) {
	if state.partPlanningEnabled {
		partPlan, ok := state.partPlans[partIndex]
		if ok && partPlan.providerSessionID != "" {
			return partPlan.providerSessionID, nil
		}
	}
	return state.reportPlanSessionID, nil
}
