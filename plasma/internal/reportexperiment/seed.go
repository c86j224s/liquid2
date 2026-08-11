package reportexperiment

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow"
)

const (
	executorCodex                 = "codex"
	experimentSelectionSource     = "experiment_fixed_reviewed_part"
	experimentMCPMode             = "auto"
	experimentExecutionStrategy   = "section_fanout"
	experimentSessionChainKind    = "section_fanout_report"
	experimentPolicySelection     = reportexecution.SessionPolicySelectionExplicitSameSession
	experimentCompositionStrategy = reporting.LongFormCompositionNarrativeEdit
)

type seedResult struct {
	Prefix reportworkflow.PrefixOutput
	Plan   seedPlanSummary
}

type seedPlanSummary struct {
	MissionID                  string `json:"mission_id"`
	PendingEventID             string `json:"pending_event_id"`
	PlanEventID                string `json:"plan_event_id"`
	FinalArtifactID            string `json:"final_artifact_id"`
	PlanSHA256                 string `json:"plan_sha256"`
	GenerationGuidanceProfile  string `json:"generation_guidance_profile"`
	GenerationGuidanceSHA256   string `json:"generation_guidance_sha256"`
	SessionChainKind           string `json:"session_chain_kind"`
	ReportPlanSessionID        string `json:"report_plan_session_id"`
	PreReportResearchSessionID string `json:"pre_report_research_session_id,omitempty"`
	ForkSourceAgentSessionID   string `json:"fork_source_agent_session_id,omitempty"`
	PartCount                  int    `json:"part_count"`
	SectionCount               int    `json:"section_count"`
}

func seedFixedPartPrefix(ctx context.Context, svc Service, loaded LoadedFixture, prepared preparedSeed, runID, agentModel, reasoningEffort, planProviderSessionID string, started time.Time) (seedResult, error) {
	planProviderSessionID = strings.TrimSpace(planProviderSessionID)
	if planProviderSessionID == "" {
		return seedResult{}, fmt.Errorf("%w: bootstrap plan provider session ID is required", producterror.ErrInvalidInput)
	}
	stem := safeIDStem(runID)
	missionID := "mis_reportexperiment_" + stem
	pendingID := "evt_reportexperiment_" + stem + "_pending"
	planID := "evt_reportexperiment_" + stem + "_plan"
	finalArtifactID := "art_reportexperiment_" + stem + "_final"
	planToolSessionID := "ses_reportexperiment_" + stem + "_plan"
	preReportSessionID := ""
	forkSourceSessionID := ""
	guidanceProfile := prepared.GuidanceProfile
	guidanceSHA := prepared.GuidanceSHA256
	producer := ledger.Producer{Type: "agent_session", ID: planProviderSessionID}

	if _, err := svc.CreateMission(ctx, mission.CreateRequest{MissionID: missionID, Title: loaded.Spec.ReportTitle}); err != nil {
		return seedResult{}, err
	}
	if _, err := svc.AppendEvent(ctx, missionCreatedAppendRequest("evt_reportexperiment_"+stem+"_mission", missionID, loaded.Spec.ReportTitle)); err != nil {
		return seedResult{}, err
	}
	if _, err := svc.AppendEvent(ctx, pendingAppendRequest(loaded, missionID, pendingID, agentModel, reasoningEffort, guidanceProfile, guidanceSHA)); err != nil {
		return seedResult{}, err
	}
	planEvent, err := svc.AppendEvent(ctx, reporting.BuildMarkdownReportPlanCreatedAppendRequest(reporting.MarkdownReportPlanCreatedEventRequest{
		MarkdownReportEventBase: markdownEventBase(markdownEventBaseInput{
			EventID: planID, MissionID: missionID, PendingEventID: pendingID, Title: loaded.Spec.ReportTitle,
			AgentModel: agentModel, ReasoningEffort: reasoningEffort, ToolSessionID: planToolSessionID,
			ProviderSessionID: planProviderSessionID, Rigor: loaded.Spec.Rigor,
			PostReportHumanize: prepared.PostReportHumanize, GuidanceProfile: guidanceProfile, GuidanceSHA256: guidanceSHA,
			PreReportSessionID: preReportSessionID, PlanSessionID: planProviderSessionID, ForkSourceSessionID: forkSourceSessionID,
			Text:     "고정 reviewed Part fixture finalization 계획을 기록했습니다.",
			Producer: producer,
		}),
		ArtifactID:          finalArtifactID,
		Plan:                prepared.Plan,
		AssemblyStrategy:    reporting.LongFormCompositionPreserveMarkdown,
		FinalEditPipeline:   prepared.FinalEditPipeline,
		PartEditEnabled:     false,
		PartPlanningEnabled: false,
		PlanReviewRequired:  false,
		PlanReviewState:     "auto_accepted",
	}))
	if err != nil {
		return seedResult{}, err
	}

	parts, sections, partArtifactIDs, sectionArtifactIDs, sectionWordTotal, err := createSeedParts(ctx, seedPartsInput{
		Service: svc, Loaded: loaded, Stem: stem, MissionID: missionID, PendingEventID: pendingID, PlanEventID: planID,
		PlanProviderSessionID: planProviderSessionID, PreReportSessionID: preReportSessionID,
		ForkSourceSessionID: forkSourceSessionID,
		AgentModel:          agentModel, ReasoningEffort: reasoningEffort, GuidanceProfile: guidanceProfile, GuidanceSHA256: guidanceSHA,
		Producer: producer,
	})
	if err != nil {
		return seedResult{}, err
	}

	prefix := reportworkflow.PrefixOutput{
		MissionID: missionID, PendingEventID: pendingID, Title: loaded.Spec.ReportTitle, DirectionHint: loaded.Spec.DirectionHint,
		ExecutionStrategy: experimentExecutionStrategy, AgentExecutor: executorCodex, AgentModel: agentModel, AgentReasoningEffort: reasoningEffort,
		AgentSelectionSource: experimentSelectionSource, MCPMode: experimentMCPMode,
		Rigor:               reportprompt.RigorProfile{Level: loaded.Spec.Rigor.Level, Label: loaded.Spec.Rigor.Label},
		ReportSessionPolicy: reportexecution.SessionPolicySameSession, ReportSessionPolicySelection: experimentPolicySelection,
		PostReportHumanize: prepared.PostReportHumanize, GenerationGuidanceProfile: guidanceProfile, GenerationGuidanceSHA256: guidanceSHA,
		ArtifactID: finalArtifactID, PlanEvent: planEvent, Plan: prepared.Plan, RequirementMap: prepared.RequirementMap,
		Parts: parts, Sections: sections, PartArtifactIDs: partArtifactIDs, SectionArtifactIDs: sectionArtifactIDs, SectionWordTotal: sectionWordTotal,
		SessionChainKind: experimentSessionChainKind, PreReportResearchSessionID: preReportSessionID, ReportPlanSessionID: planProviderSessionID,
		ForkSourceAgentSessionID: forkSourceSessionID, PartEditEnabled: false, PartPlanningEnabled: false,
		FinalEditPipeline: prepared.FinalEditPipeline, FinalTail: prepared.FinalTail, StartedAt: started,
	}
	return seedResult{Prefix: prefix, Plan: seedPlanSummary{
		MissionID: missionID, PendingEventID: pendingID, PlanEventID: planID, FinalArtifactID: finalArtifactID,
		PlanSHA256: prepared.PlanSHA256, GenerationGuidanceProfile: guidanceProfile, GenerationGuidanceSHA256: guidanceSHA,
		SessionChainKind: experimentSessionChainKind, ReportPlanSessionID: planProviderSessionID,
		PreReportResearchSessionID: preReportSessionID, ForkSourceAgentSessionID: forkSourceSessionID,
		PartCount: len(parts), SectionCount: len(sectionArtifactIDs),
	}}, nil
}
