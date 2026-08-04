package reporting

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

const (
	FinalEditStageWriter                  = "final_write"
	FinalEditStageReader                  = "reader_edit"
	FinalEditStageStyle                   = "style_edit"
	FinalEditStageGate                    = "corrective_gate"
	FinalEditStageStyleSemanticValidation = "style_semantic_validation"
	FinalEditStageEvidenceGate            = "evidence_gate"

	FinalEditWriterStartedEventType                    = "report.final_edit.writer.started"
	FinalEditWriterSubmittedEventType                  = "report.final_edit.writer.submitted"
	FinalEditReaderStartedEventType                    = "report.final_edit.reader.started"
	FinalEditReaderSubmittedEventType                  = "report.final_edit.reader.submitted"
	FinalEditStyleStartedEventType                     = "report.final_edit.style.started"
	FinalEditStyleSubmittedEventType                   = "report.final_edit.style.submitted"
	FinalEditGateStartedEventType                      = "report.final_edit.gate.started"
	FinalEditGateSubmittedEventType                    = "report.final_edit.gate.submitted"
	FinalEditStyleSemanticValidationStartedEventType   = "report.final_edit.style_semantic_validation.started"
	FinalEditStyleSemanticValidationSubmittedEventType = "report.final_edit.style_semantic_validation.submitted"
	FinalEditEvidenceGateStartedEventType              = "report.final_edit.evidence_gate.started"
	FinalEditEvidenceGateSubmittedEventType            = "report.final_edit.evidence_gate.submitted"
)

// BuildFinalEditStageStartedAppendRequest는 보고서 생성 파이프라인에서 장부에 기록할 append 요청을 조립한다. 실제 저장과 조건부 append 결정은 호출자가 소유한다.
func BuildFinalEditStageStartedAppendRequest(eventID string, binding FinalEditStageBinding) app.AppendEventRequest {
	binding = normalizeFinalEditStageBinding(binding)
	payload := finalEditStageBasePayload(binding)
	payload["kind"] = "long_form_final_edit_" + binding.Stage + "_started"
	payload["text"] = fmt.Sprintf("장문 리포트 %s 단계를 시작했습니다.", binding.Stage)
	return app.AppendEventRequest{
		EventID:          strings.TrimSpace(eventID),
		MissionID:        binding.MissionID,
		EventType:        finalEditStartedEventType(binding.Stage),
		Producer:         app.Producer{Type: "agent_session", ID: binding.ProviderSessionID},
		CausationEventID: binding.PlanEventID,
		CorrelationID:    binding.IdempotencyKey,
		Payload:          mustJSON(payload),
	}
}

func buildFinalEditSubmittedAppendRequest(eventID string, binding FinalEditStageBinding, source, artifact app.RawArtifact, operationCount int, changed bool, findings []StoredFinalEditGateFinding, semanticReview FinalEditSemanticAttestation) app.AppendEventRequest {
	return buildFinalEditSubmittedAppendRequestWithStyleDiagnoses(eventID, binding, source, artifact, operationCount, changed, nil, findings, semanticReview)
}

func buildFinalEditSubmittedAppendRequestWithStyleDiagnoses(eventID string, binding FinalEditStageBinding, source, artifact app.RawArtifact, operationCount int, changed bool, diagnoses []FinalEditStyleOperationDiagnosis, findings []StoredFinalEditGateFinding, semanticReview FinalEditSemanticAttestation) app.AppendEventRequest {
	payload := finalEditSubmittedPayload{
		Kind:                         "long_form_final_edit_" + binding.Stage + "_submitted",
		PendingEventID:               binding.PendingEventID,
		PlanEventID:                  binding.PlanEventID,
		FinalEditPipeline:            finalEditStagePayloadPipeline(binding),
		Title:                        binding.Title,
		Stage:                        binding.Stage,
		StageID:                      finalEditStageID(binding.Stage),
		SourceArtifactID:             binding.SourceArtifactID,
		ArtifactID:                   artifact.ArtifactID,
		EditedArtifactID:             binding.EditedArtifactID,
		Filename:                     binding.Filename,
		ToolSessionID:                binding.ToolSessionID,
		ProviderSessionID:            binding.ProviderSessionID,
		PreviousProviderSessionID:    binding.PreviousProviderSessionID,
		IdempotencyKey:               binding.IdempotencyKey,
		AgentExecutor:                binding.AgentExecutor,
		AgentModel:                   binding.AgentModel,
		AgentReasoningEffort:         binding.AgentReasoningEffort,
		AgentSelectionSource:         binding.AgentSelectionSource,
		MCPMode:                      binding.MCPMode,
		RigorLevel:                   binding.RigorLevel,
		RigorLabel:                   binding.RigorLabel,
		ReportSessionPolicy:          binding.ReportSessionPolicy,
		ReportSessionPolicySelection: binding.ReportSessionPolicySelection,
		PostReportHumanize:           binding.PostReportHumanize,
		GenerationGuidanceProfile:    binding.GenerationGuidanceProfile,
		GenerationGuidanceSHA256:     binding.GenerationGuidanceSHA256,
		SessionChainKind:             binding.SessionChainKind,
		PreReportResearchSessionID:   binding.PreReportResearchSessionID,
		ReportPlanSessionID:          binding.ReportPlanSessionID,
		ForkSourceAgentSessionID:     binding.ForkSourceAgentSessionID,
		OperationCount:               operationCount,
		SourceWordCount:              len(strings.Fields(string(source.Content))),
		EditedWordCount:              len(strings.Fields(string(artifact.Content))),
		SourceSHA256:                 source.SHA256,
		ArtifactSHA256:               artifact.SHA256,
		Changed:                      changed,
		GateFindings:                 append([]StoredFinalEditGateFinding(nil), findings...),
		SemanticAcceptance:           append([]StoredFinalEditSemanticAcceptance(nil), semanticReview.Records...),
		SemanticAcceptanceDigest:     semanticReview.Digest,
		SemanticAcceptanceCount:      semanticReview.Count,
		Text:                         fmt.Sprintf("장문 리포트 %s 단계를 durable artifact로 제출했습니다.", binding.Stage),
	}
	if binding.Stage == FinalEditStageStyle {
		copied := make([]FinalEditStyleOperationDiagnosis, len(diagnoses))
		copy(copied, diagnoses)
		payload.StyleDiagnosesVersion = FinalEditStyleOperationDiagnosesVersion
		payload.StyleOperationDiagnoses = &copied
	}
	return app.AppendEventRequest{
		EventID:          strings.TrimSpace(eventID),
		MissionID:        binding.MissionID,
		EventType:        finalEditSubmittedEventType(binding.Stage),
		Producer:         app.Producer{Type: "agent_session", ID: binding.ProviderSessionID},
		CausationEventID: binding.PlanEventID,
		CorrelationID:    binding.IdempotencyKey,
		Payload:          mustJSON(payload),
	}
}

func finalEditStageBasePayload(binding FinalEditStageBinding) map[string]any {
	return map[string]any{
		"pending_event_id":                binding.PendingEventID,
		"plan_event_id":                   binding.PlanEventID,
		"final_edit_pipeline":             finalEditStagePayloadPipeline(binding),
		"title":                           binding.Title,
		"stage":                           binding.Stage,
		"stage_id":                        finalEditStageID(binding.Stage),
		"source_artifact_id":              binding.SourceArtifactID,
		"artifact_id":                     binding.EditedArtifactID,
		"filename":                        binding.Filename,
		"tool_session_id":                 binding.ToolSessionID,
		"provider_session_id":             binding.ProviderSessionID,
		"previous_provider_session_id":    binding.PreviousProviderSessionID,
		"idempotency_key":                 binding.IdempotencyKey,
		"agent_executor":                  binding.AgentExecutor,
		"agent_model":                     binding.AgentModel,
		"agent_reasoning_effort":          binding.AgentReasoningEffort,
		"agent_selection_source":          binding.AgentSelectionSource,
		"mcp_mode":                        binding.MCPMode,
		"rigor_level":                     binding.RigorLevel,
		"rigor_label":                     binding.RigorLabel,
		"report_session_policy":           binding.ReportSessionPolicy,
		"report_session_policy_selection": binding.ReportSessionPolicySelection,
		"post_report_humanize":            binding.PostReportHumanize,
		"generation_guidance_profile":     binding.GenerationGuidanceProfile,
		"generation_guidance_sha256":      binding.GenerationGuidanceSHA256,
		"session_chain_kind":              binding.SessionChainKind,
		"pre_report_research_session_id":  binding.PreReportResearchSessionID,
		"report_plan_session_id":          binding.ReportPlanSessionID,
		"fork_source_agent_session_id":    binding.ForkSourceAgentSessionID,
	}
}

func finalEditStartedEventType(stage string) string {
	switch stage {
	case FinalEditStageWriter:
		return FinalEditWriterStartedEventType
	case FinalEditStageReader:
		return FinalEditReaderStartedEventType
	case FinalEditStageStyle:
		return FinalEditStyleStartedEventType
	case FinalEditStageGate:
		return FinalEditGateStartedEventType
	case FinalEditStageStyleSemanticValidation:
		return FinalEditStyleSemanticValidationStartedEventType
	case FinalEditStageEvidenceGate:
		return FinalEditEvidenceGateStartedEventType
	default:
		return ""
	}
}

func finalEditSubmittedEventType(stage string) string {
	switch stage {
	case FinalEditStageWriter:
		return FinalEditWriterSubmittedEventType
	case FinalEditStageReader:
		return FinalEditReaderSubmittedEventType
	case FinalEditStageStyle:
		return FinalEditStyleSubmittedEventType
	case FinalEditStageGate:
		return FinalEditGateSubmittedEventType
	case FinalEditStageStyleSemanticValidation:
		return FinalEditStyleSemanticValidationSubmittedEventType
	case FinalEditStageEvidenceGate:
		return FinalEditEvidenceGateSubmittedEventType
	default:
		return ""
	}
}

func finalEditStageID(stage string) string {
	switch stage {
	case FinalEditStageWriter:
		return "final-write"
	case FinalEditStageReader:
		return "reader-edit"
	case FinalEditStageStyle:
		return "style-edit"
	case FinalEditStageGate:
		return "corrective-gate"
	case FinalEditStageStyleSemanticValidation:
		return "style-semantic-validation"
	case FinalEditStageEvidenceGate:
		return "evidence-gate"
	default:
		return strings.TrimSpace(stage)
	}
}

func finalEditStagePayloadPipeline(binding FinalEditStageBinding) string {
	pipeline := strings.TrimSpace(binding.FinalEditPipeline)
	if pipeline != "" {
		return pipeline
	}
	if strings.TrimSpace(binding.Stage) == FinalEditStageWriter {
		return FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
	}
	return FinalEditPipelineReaderStyleGateV1
}
