package reporting

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

const (
	FinalEditPipelineReaderStyleGateV1                                 = "reader_style_gate_v1"
	FinalEditPipelineAssemblyWriterReaderStyleGateV2                   = "assembly_writer_reader_style_gate_v2"
	FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 = "assembly_writer_reader_style_validation_evidence_gate_v3"
)

const (
	FinalEditHumanizeEnabled  = "enabled"
	FinalEditHumanizeDisabled = "disabled"
)

// FinalEditPipelinePlanState는 계산한 읽기 모델이다. 원천 상태는 장부와 저장소에 남아 있다.
type FinalEditPipelinePlanState struct {
	Pipeline           string
	PendingEventID     string
	PlanEventID        string
	ArtifactID         string
	ReportMode         string
	PostReportHumanize string
}

// FinalEditStageBinding는 재실행과 검증에 쓰는 binding 계약이다.
type FinalEditStageBinding struct {
	MissionID                    string       `json:"mission_id"`
	PendingEventID               string       `json:"pending_event_id"`
	PlanEventID                  string       `json:"plan_event_id"`
	FinalEditPipeline            string       `json:"final_edit_pipeline,omitempty"`
	Title                        string       `json:"title"`
	Stage                        string       `json:"stage"`
	SourceArtifactID             string       `json:"source_artifact_id"`
	EditedArtifactID             string       `json:"edited_artifact_id"`
	Filename                     string       `json:"filename"`
	ToolSessionID                string       `json:"tool_session_id"`
	ProviderSessionID            string       `json:"provider_session_id"`
	PreviousProviderSessionID    string       `json:"previous_provider_session_id"`
	IdempotencyKey               string       `json:"idempotency_key"`
	AgentExecutor                string       `json:"agent_executor"`
	AgentModel                   string       `json:"agent_model"`
	AgentReasoningEffort         string       `json:"agent_reasoning_effort"`
	AgentSelectionSource         string       `json:"agent_selection_source"`
	MCPMode                      string       `json:"mcp_mode"`
	RigorLevel                   string       `json:"rigor_level"`
	RigorLabel                   string       `json:"rigor_label"`
	ReportSessionPolicy          string       `json:"report_session_policy"`
	ReportSessionPolicySelection string       `json:"report_session_policy_selection"`
	PostReportHumanize           string       `json:"post_report_humanize"`
	GenerationGuidanceProfile    string       `json:"generation_guidance_profile"`
	GenerationGuidanceSHA256     string       `json:"generation_guidance_sha256"`
	SessionChainKind             string       `json:"session_chain_kind"`
	PreReportResearchSessionID   string       `json:"pre_report_research_session_id"`
	ReportPlanSessionID          string       `json:"report_plan_session_id"`
	ForkSourceAgentSessionID     string       `json:"fork_source_agent_session_id"`
	Producer                     app.Producer `json:"producer"`
}

// FinalEditStageResult는 final edit stage 제출 이벤트와 artifact를 함께 반환한다.
type FinalEditStageResult struct {
	Binding                        FinalEditStageBinding
	Artifact                       app.RawArtifact
	Event                          app.LedgerEvent
	Replay                         bool
	OperationCount                 int
	Changed                        bool
	GateFindings                   []StoredFinalEditGateFinding
	SemanticReview                 FinalEditSemanticAttestation
	StyleOperationDiagnoses        []FinalEditStyleOperationDiagnosis
	StyleOperationDiagnosesPresent bool
}

type finalEditSubmittedPayload struct {
	Kind                         string                              `json:"kind"`
	PendingEventID               string                              `json:"pending_event_id"`
	PlanEventID                  string                              `json:"plan_event_id"`
	FinalEditPipeline            string                              `json:"final_edit_pipeline"`
	Title                        string                              `json:"title"`
	Stage                        string                              `json:"stage"`
	StageID                      string                              `json:"stage_id"`
	SourceArtifactID             string                              `json:"source_artifact_id"`
	ArtifactID                   string                              `json:"artifact_id"`
	EditedArtifactID             string                              `json:"edited_artifact_id"`
	Filename                     string                              `json:"filename"`
	ToolSessionID                string                              `json:"tool_session_id"`
	ProviderSessionID            string                              `json:"provider_session_id"`
	PreviousProviderSessionID    string                              `json:"previous_provider_session_id"`
	IdempotencyKey               string                              `json:"idempotency_key"`
	AgentExecutor                string                              `json:"agent_executor"`
	AgentModel                   string                              `json:"agent_model,omitempty"`
	AgentReasoningEffort         string                              `json:"agent_reasoning_effort,omitempty"`
	AgentSelectionSource         string                              `json:"agent_selection_source,omitempty"`
	MCPMode                      string                              `json:"mcp_mode,omitempty"`
	RigorLevel                   string                              `json:"rigor_level,omitempty"`
	RigorLabel                   string                              `json:"rigor_label,omitempty"`
	ReportSessionPolicy          string                              `json:"report_session_policy,omitempty"`
	ReportSessionPolicySelection string                              `json:"report_session_policy_selection,omitempty"`
	PostReportHumanize           string                              `json:"post_report_humanize,omitempty"`
	GenerationGuidanceProfile    string                              `json:"generation_guidance_profile,omitempty"`
	GenerationGuidanceSHA256     string                              `json:"generation_guidance_sha256,omitempty"`
	SessionChainKind             string                              `json:"session_chain_kind,omitempty"`
	PreReportResearchSessionID   string                              `json:"pre_report_research_session_id,omitempty"`
	ReportPlanSessionID          string                              `json:"report_plan_session_id,omitempty"`
	ForkSourceAgentSessionID     string                              `json:"fork_source_agent_session_id,omitempty"`
	OperationCount               int                                 `json:"operation_count"`
	StyleDiagnosesVersion        int                                 `json:"style_operation_diagnoses_version,omitempty"`
	SourceWordCount              int                                 `json:"source_word_count"`
	EditedWordCount              int                                 `json:"edited_word_count"`
	SourceSHA256                 string                              `json:"source_sha256"`
	ArtifactSHA256               string                              `json:"artifact_sha256"`
	Changed                      bool                                `json:"changed"`
	StyleOperationDiagnoses      *[]FinalEditStyleOperationDiagnosis `json:"style_operation_diagnoses,omitempty"`
	GateFindings                 []StoredFinalEditGateFinding        `json:"gate_findings,omitempty"`
	SemanticAcceptance           []StoredFinalEditSemanticAcceptance `json:"semantic_acceptance,omitempty"`
	SemanticAcceptanceDigest     string                              `json:"semantic_acceptance_digest,omitempty"`
	SemanticAcceptanceCount      int                                 `json:"semantic_acceptance_count,omitempty"`
	Text                         string                              `json:"text"`
}

func longFormCanonicalRequestForFinalEdit(eventID string, binding LongFormFinalizeBinding, artifact app.RawArtifact, finalWords int, req LongFormFinalizeRequest) app.AppendEventRequest {
	request := longFormCanonicalRequest(eventID, binding, artifact, finalWords)
	if !isSupportedFinalEditPipeline(strings.TrimSpace(req.FinalEditPipeline)) {
		return request
	}
	payload := eventPayload(app.LedgerEvent{Payload: request.Payload})
	putFinalEditCanonicalFields(payload, binding, artifact, finalEditCanonicalFields{
		Pipeline:         strings.TrimSpace(req.FinalEditPipeline),
		GateFindings:     req.GateFindings,
		SemanticReview:   req.SemanticReview,
		ActualArtifactID: strings.TrimSpace(req.FinalEditActualArtifactID),
		GateEventID:      strings.TrimSpace(req.FinalEditGateEventID),
		GateChanged:      req.FinalEditGateChanged,
	})
	request.Payload = mustJSON(payload)
	return request
}

func putFinalEditCanonicalFields(payload map[string]any, binding LongFormFinalizeBinding, artifact app.RawArtifact, fields finalEditCanonicalFields) {
	if strings.TrimSpace(fields.Pipeline) == "" && len(fields.GateFindings) == 0 {
		return
	}
	putReportNonEmpty(payload, "final_edit_pipeline", strings.TrimSpace(fields.Pipeline))
	if isSupportedFinalEditPipeline(strings.TrimSpace(fields.Pipeline)) {
		payload["artifact_id"] = artifact.ArtifactID
		payload["planned_final_artifact_id"] = binding.ArtifactID
		payload["final_edit_gate_event_id"] = strings.TrimSpace(fields.GateEventID)
		payload["final_edit_gate_changed"] = fields.GateChanged
		payload["artifact_sha256"] = artifact.SHA256
	}
	if len(fields.GateFindings) > 0 {
		payload["final_edit_gate_findings"] = fields.GateFindings
	}
	if fields.SemanticReview.Count > 0 {
		payload["final_edit_semantic_acceptance"] = fields.SemanticReview.Records
		payload["final_edit_semantic_acceptance_digest"] = fields.SemanticReview.Digest
		payload["final_edit_semantic_acceptance_count"] = fields.SemanticReview.Count
	}
}

func canonicalArtifactIDForFinalizeRequest(binding LongFormFinalizeBinding, req LongFormFinalizeRequest) string {
	if isSupportedFinalEditPipeline(strings.TrimSpace(req.FinalEditPipeline)) && strings.TrimSpace(req.FinalEditActualArtifactID) != "" {
		return strings.TrimSpace(req.FinalEditActualArtifactID)
	}
	return binding.ArtifactID
}

// FinalEditPipelineFromPlanEvent는 plan 이벤트 payload에서 최종 편집 pipeline 선택을 읽는다.
func FinalEditPipelineFromPlanEvent(event app.LedgerEvent) (FinalEditPipelinePlanState, bool, error) {
	if event.EventType != "report.plan.created" {
		return FinalEditPipelinePlanState{}, false, nil
	}
	var payload struct {
		PendingEventID     string `json:"pending_event_id"`
		ArtifactID         string `json:"artifact_id"`
		ReportMode         string `json:"report_mode"`
		FinalEditPipeline  string `json:"final_edit_pipeline"`
		PostReportHumanize string `json:"post_report_humanize"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return FinalEditPipelinePlanState{}, false, fmt.Errorf("%w: report plan payload is invalid", app.ErrConflict)
	}
	pipeline := strings.TrimSpace(payload.FinalEditPipeline)
	if pipeline != "" && !isSupportedFinalEditPipeline(pipeline) {
		return FinalEditPipelinePlanState{}, false, fmt.Errorf("%w: unsupported final edit pipeline", app.ErrConflict)
	}
	postReportHumanize := strings.TrimSpace(payload.PostReportHumanize)
	if isSupportedFinalEditPipeline(pipeline) {
		normalized, err := normalizeFinalEditHumanize(postReportHumanize)
		if err != nil {
			return FinalEditPipelinePlanState{}, false, err
		}
		postReportHumanize = normalized
	}
	return FinalEditPipelinePlanState{
		Pipeline:           pipeline,
		PendingEventID:     strings.TrimSpace(payload.PendingEventID),
		PlanEventID:        strings.TrimSpace(event.EventID),
		ArtifactID:         strings.TrimSpace(payload.ArtifactID),
		ReportMode:         strings.TrimSpace(payload.ReportMode),
		PostReportHumanize: postReportHumanize,
	}, true, nil
}

func isSupportedFinalEditPipeline(value string) bool {
	switch strings.TrimSpace(value) {
	case FinalEditPipelineReaderStyleGateV1, FinalEditPipelineAssemblyWriterReaderStyleGateV2, FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3:
		return true
	default:
		return false
	}
}

func normalizeFinalEditHumanize(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case FinalEditHumanizeEnabled:
		return FinalEditHumanizeEnabled, nil
	case FinalEditHumanizeDisabled:
		return FinalEditHumanizeDisabled, nil
	default:
		return "", fmt.Errorf("%w: final edit post_report_humanize must be enabled or disabled", app.ErrConflict)
	}
}
