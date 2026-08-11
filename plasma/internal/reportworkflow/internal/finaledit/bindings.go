package finaledit

import (
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

// SupportsFinalEditPlanEvent는 durable plan event가 현재 Runner가 실행하는
// V1/V2/V3 final edit pipeline인지 읽기 전용으로 판정한다.
func SupportsFinalEditPlanEvent(event ledger.Event) bool {
	state, ok, err := reporting.FinalEditPipelineFromPlanEvent(event)
	return err == nil && ok && supportsFinalEditPipeline(state.Pipeline)
}

// FinalEditPipeline은 canonical plan event에 저장된 final edit pipeline만 읽는다.
func (input Input) FinalEditPipeline() string {
	state, ok, err := reporting.FinalEditPipelineFromPlanEvent(input.PlanEvent)
	if err != nil || !ok {
		return ""
	}
	return strings.TrimSpace(state.Pipeline)
}

// FinalEditAgentReasoningEffort는 final edit binding에 저장할 effort를 정규화한다.
func (input Input) FinalEditAgentReasoningEffort() string {
	return LongFormFinalEditContractReasoningEffort(input.AgentReasoningEffort)
}

// LongFormFinalBinding은 canonical finalization replay에 쓰는 durable binding을 만든다.
func (input Input) LongFormFinalBinding(toolSessionID string, providerSessionID string, previousProviderSessionID string, forkSourceAgentSessionID string) reporting.LongFormFinalizeBinding {
	return reporting.LongFormFinalizeBinding{
		MissionID:                    input.MissionID,
		PendingEventID:               input.PendingEventID,
		PlanEventID:                  input.PlanEvent.EventID,
		ArtifactID:                   input.ArtifactID,
		Filename:                     SafeFilename(input.Title, ".md"),
		Title:                        input.Title,
		ToolSessionID:                toolSessionID,
		IdempotencyKey:               "report-long-form-finalize:" + input.PendingEventID + ":" + input.PlanEvent.EventID,
		ProviderSessionID:            providerSessionID,
		PreviousProviderSessionID:    previousProviderSessionID,
		PartArtifactIDs:              input.PartArtifactIDs,
		SectionArtifactIDs:           input.SectionArtifactIDs,
		SectionWordCount:             input.SectionWordTotal,
		CompositionStrategy:          reporting.LongFormCompositionNarrativeEdit,
		AgentExecutor:                input.ExecutorName,
		AgentModel:                   input.AgentModel,
		AgentReasoningEffort:         input.FinalEditAgentReasoningEffort(),
		AgentSelectionSource:         input.AgentSelectionSource,
		MCPMode:                      input.MCPMode,
		RigorLevel:                   input.Rigor.Level,
		RigorLabel:                   input.Rigor.Label,
		ReportSessionPolicy:          input.ReportSessionPolicy,
		ReportSessionPolicySelection: input.ReportSessionPolicySelection,
		PostReportHumanize:           input.PostReportHumanize,
		GenerationGuidanceProfile:    input.GenerationGuidanceProfile,
		GenerationGuidanceSHA256:     input.GenerationGuidanceSHA256,
		SessionChainKind:             input.SessionChainKind,
		PreReportResearchSessionID:   input.PreReportResearchSessionID,
		ReportPlanSessionID:          input.ReportPlanSessionID,
		ForkSourceAgentSessionID:     forkSourceAgentSessionID,
		PlanToolSessionID:            ReportEventString(input.PlanEvent, "tool_session_id"),
		StartedAt:                    input.Started,
		Producer:                     ledger.Producer{Type: "agent_session", ID: providerSessionID},
	}
}

// FinalEditStageBinding은 한 final edit stage의 durable start/submit 계약을 만든다.
func (input Input) FinalEditStageBinding(stage string, sourceArtifactID string, editedArtifactID string, toolSessionID string, providerSessionID string, previousProviderSessionID string, forkSourceAgentSessionID string) reporting.FinalEditStageBinding {
	binding := reporting.FinalEditStageBinding{
		MissionID:                    input.MissionID,
		PendingEventID:               input.PendingEventID,
		PlanEventID:                  input.PlanEvent.EventID,
		Title:                        input.Title,
		Stage:                        stage,
		SourceArtifactID:             sourceArtifactID,
		EditedArtifactID:             editedArtifactID,
		Filename:                     SafeFilename(input.Title, ".md"),
		ToolSessionID:                toolSessionID,
		ProviderSessionID:            providerSessionID,
		PreviousProviderSessionID:    previousProviderSessionID,
		IdempotencyKey:               reporting.FinalEditStageIdempotencyKey(stage, input.PendingEventID, input.PlanEvent.EventID),
		AgentExecutor:                input.ExecutorName,
		AgentModel:                   input.AgentModel,
		AgentReasoningEffort:         input.FinalEditAgentReasoningEffort(),
		AgentSelectionSource:         input.AgentSelectionSource,
		MCPMode:                      input.MCPMode,
		RigorLevel:                   input.Rigor.Level,
		RigorLabel:                   input.Rigor.Label,
		ReportSessionPolicy:          input.ReportSessionPolicy,
		ReportSessionPolicySelection: input.ReportSessionPolicySelection,
		PostReportHumanize:           input.PostReportHumanize,
		GenerationGuidanceProfile:    input.GenerationGuidanceProfile,
		GenerationGuidanceSHA256:     input.GenerationGuidanceSHA256,
		SessionChainKind:             input.SessionChainKind,
		PreReportResearchSessionID:   input.PreReportResearchSessionID,
		ReportPlanSessionID:          input.ReportPlanSessionID,
		ForkSourceAgentSessionID:     forkSourceAgentSessionID,
		Producer:                     ledger.Producer{Type: "agent_session", ID: providerSessionID},
	}
	if pipeline := input.FinalEditPipeline(); pipeline == reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2 || pipeline == reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 {
		binding.FinalEditPipeline = pipeline
	}
	return binding
}

func supportsFinalEditPipeline(value string) bool {
	switch strings.TrimSpace(value) {
	case reporting.FinalEditPipelineReaderStyleGateV1,
		reporting.FinalEditPipelineAssemblyWriterReaderStyleGateV2,
		reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3:
		return true
	default:
		return false
	}
}
