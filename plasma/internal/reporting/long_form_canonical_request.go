package reporting

import (
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func longFormCanonicalRequest(eventID string, binding LongFormFinalizeBinding, artifact app.RawArtifact, finalWords int) app.AppendEventRequest {
	duration := time.Since(binding.StartedAt).Milliseconds()
	if binding.StartedAt.IsZero() || duration < 0 {
		duration = 0
	}
	assemblyStrategy := "c4_normalized_section_headings"
	text := "섹션별 보존 조립 방식으로 장문 Markdown 리포트 artifact를 생성했습니다."
	if binding.CompositionStrategy == LongFormCompositionNarrativeEdit {
		assemblyStrategy = "narrative_contract_final_edit"
		text = "바인딩된 파트 원고를 최종 편집해 장문 Markdown 리포트 artifact를 생성했습니다."
	}
	request := BuildMarkdownReportArtifactCreatedAppendRequest(MarkdownReportArtifactCreatedEventRequest{
		MarkdownReportEventBase: MarkdownReportEventBase{
			EventID: eventID, MissionID: binding.MissionID, PendingEventID: binding.PendingEventID, Title: binding.Title,
			AgentExecutor: binding.AgentExecutor, AgentModel: binding.AgentModel, AgentReasoningEffort: binding.AgentReasoningEffort, AgentSelectionSource: binding.AgentSelectionSource,
			AgentSessionID: binding.ProviderSessionID, PreviousAgentSessionID: binding.PreviousProviderSessionID,
			ToolSessionID: binding.ToolSessionID, MCPMode: binding.MCPMode, RigorLevel: binding.RigorLevel, RigorLabel: binding.RigorLabel,
			ReportMode: ModeLongForm, ReportModeLabel: ModeLabel(ModeLongForm), ReportSessionPolicy: binding.ReportSessionPolicy, ReportSessionPolicySelection: binding.ReportSessionPolicySelection,
			PostReportHumanize: binding.PostReportHumanize, HumanizeEnabled: binding.PostReportHumanize != "disabled",
			GenerationGuidanceProfile: binding.GenerationGuidanceProfile, GenerationGuidanceSHA256: binding.GenerationGuidanceSHA256,
			SessionChainKind: binding.SessionChainKind, PreReportResearchSessionID: binding.PreReportResearchSessionID, ReportPlanSessionID: binding.ReportPlanSessionID,
			ReportSessionID: binding.ProviderSessionID, ForkSourceAgentSessionID: binding.ForkSourceAgentSessionID,
			CompositionStrategy: binding.CompositionStrategy, DurationMS: duration,
			Text: text, Producer: binding.Producer,
		},
		Artifact: artifact, PlanEventID: binding.PlanEventID, PlanToolSessionID: binding.PlanToolSessionID,
		IncludePlanReview: true, PlanReviewState: "auto_accepted", AssemblyStrategy: assemblyStrategy,
		SectionCount: len(binding.SectionArtifactIDs), PartCount: len(binding.PartArtifactIDs), SectionArtifactIDs: binding.SectionArtifactIDs,
		PartArtifactIDs: binding.PartArtifactIDs, SectionWordCount: binding.SectionWordCount, FinalWordCount: finalWords,
		PreservationRatio:     float64(finalWords) / float64(maxReportingInt(1, binding.SectionWordCount)),
		OmitPreservationRatio: binding.CompositionStrategy == LongFormCompositionNarrativeEdit, IncludeLongFormFields: true,
	})
	request.CorrelationID = binding.IdempotencyKey
	return request
}
