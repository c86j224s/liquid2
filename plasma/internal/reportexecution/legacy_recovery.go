package reportexecution

import (
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"strings"
)

// RecoveryDefaults carries product defaults without importing transport or prompt owners.
type RecoveryDefaults struct{ RigorLevel, ReportMode, SessionPolicy, GuidanceProfile string }

// LegacyRecoveryDraftRequest preserves the Web recovery payload field set, including
// historical omission of rigor_label and pipeline_graph. It intentionally differs
// from the general execution decoder until a separately approved behavior change.
func DraftPendingRecoverable(event ledger.Event) bool {
	if event.EventType != "report.draft.pending" {
		return false
	}
	var payload struct {
		Kind              string `json:"kind"`
		AgentExecutor     string `json:"agent_executor"`
		ReportMode        string `json:"report_mode"`
		MCPMode           string `json:"mcp_mode"`
		ExecutionStrategy string `json:"execution_strategy"`
		PipelineFamily    string `json:"pipeline_family"`
		RetryStrategy     string `json:"retry_strategy"`
		RetryOf           string `json:"retry_of_pending_event_id"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return false
	}
	if strings.TrimSpace(payload.PipelineFamily) == reportilcontract.PipelineFamily &&
		(strings.TrimSpace(payload.RetryStrategy) == "restart" ||
			strings.TrimSpace(payload.RetryStrategy) == "resume_failed") &&
		strings.HasPrefix(strings.TrimSpace(payload.RetryOf), "evt_") {
		return true
	}
	switch strings.TrimSpace(payload.Kind) {
	case "markdown_report_artifact_pending", "report_draft_pending":
		return true
	}
	return strings.TrimSpace(payload.AgentExecutor) != "" &&
		strings.TrimSpace(payload.ReportMode) != ""
}

func LegacyRecoveryDraftRequest(event ledger.Event, defaults RecoveryDefaults) (DraftRequest, error) {
	var payload struct {
		Title                        string        `json:"title"`
		DirectionHint                string        `json:"direction_hint"`
		ExecutionStrategy            string        `json:"execution_strategy"`
		AgentExecutor                string        `json:"agent_executor"`
		AgentModel                   string        `json:"agent_model"`
		AgentReasoningEffort         string        `json:"agent_reasoning_effort"`
		AgentSelectionSource         string        `json:"agent_selection_source"`
		MCPMode                      string        `json:"mcp_mode"`
		RigorLevel                   string        `json:"rigor_level"`
		ReportMode                   string        `json:"report_mode"`
		PipelineFamily               string        `json:"pipeline_family"`
		ReportSessionPolicy          string        `json:"report_session_policy"`
		ReportSessionPolicySelection string        `json:"report_session_policy_selection"`
		PostReportHumanize           string        `json:"post_report_humanize"`
		GenerationGuidanceProfile    string        `json:"generation_guidance_profile"`
		GenerationGuidanceSHA256     string        `json:"generation_guidance_sha256"`
		RetryStrategy                string        `json:"retry_strategy"`
		RetryOfPendingEventID        string        `json:"retry_of_pending_event_id"`
		ResumeStage                  string        `json:"resume_stage"`
		OutputKind                   string        `json:"output_kind"`
		ArticleIntent                ArticleIntent `json:"article_intent"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return DraftRequest{}, fmt.Errorf("%w: invalid report pending payload", producterror.ErrInvalidInput)
	}
	req := DraftRequest{
		Title:                        firstNonEmpty(payload.Title, "Mission report"),
		DirectionHint:                NormalizeDirectionHint(payload.DirectionHint),
		ExecutionStrategy:            strings.TrimSpace(strings.ToLower(payload.ExecutionStrategy)),
		AgentExecutor:                firstNonEmpty(payload.AgentExecutor, "codex"),
		AgentModel:                   strings.TrimSpace(payload.AgentModel),
		AgentReasoningEffort:         strings.TrimSpace(payload.AgentReasoningEffort),
		AgentSelectionSource:         strings.TrimSpace(payload.AgentSelectionSource),
		MCPMode:                      firstNonEmpty(payload.MCPMode, "auto"),
		RigorLevel:                   firstNonEmpty(payload.RigorLevel, defaults.RigorLevel),
		ReportMode:                   firstNonEmpty(payload.ReportMode, defaults.ReportMode),
		PipelineFamily:               strings.TrimSpace(payload.PipelineFamily),
		ReportSessionPolicy:          firstNonEmpty(payload.ReportSessionPolicy, defaults.SessionPolicy),
		ReportSessionPolicySelection: strings.TrimSpace(payload.ReportSessionPolicySelection),
		PostReportHumanize:           strings.TrimSpace(payload.PostReportHumanize),
		GenerationGuidanceProfile:    firstNonEmpty(payload.GenerationGuidanceProfile, defaults.GuidanceProfile),
		GenerationGuidanceSHA256:     strings.TrimSpace(payload.GenerationGuidanceSHA256),
		RetryStrategy:                strings.TrimSpace(payload.RetryStrategy),
		RetryOfPendingEventID:        strings.TrimSpace(payload.RetryOfPendingEventID),
		ResumeStage:                  strings.TrimSpace(payload.ResumeStage),
		OutputKind:                   strings.TrimSpace(payload.OutputKind),
		ArticleIntent:                payload.ArticleIntent,
	}
	canonical := NormalizeDraftRequest(DraftRequest{
		Title: req.Title, DirectionHint: req.DirectionHint, ExecutionStrategy: req.ExecutionStrategy,
		AgentExecutor: req.AgentExecutor, AgentModel: req.AgentModel, AgentReasoningEffort: req.AgentReasoningEffort,
		AgentSelectionSource: req.AgentSelectionSource, MCPMode: req.MCPMode, RigorLevel: req.RigorLevel,
		ReportMode: req.ReportMode, PipelineFamily: req.PipelineFamily, ReportSessionPolicy: req.ReportSessionPolicy,
		ReportSessionPolicySelection: req.ReportSessionPolicySelection, PostReportHumanize: req.PostReportHumanize,
		GenerationGuidanceProfile: req.GenerationGuidanceProfile, GenerationGuidanceSHA256: req.GenerationGuidanceSHA256,
		RetryStrategy: req.RetryStrategy, RetryOfPendingEventID: req.RetryOfPendingEventID, ResumeStage: req.ResumeStage,
		OutputKind: req.OutputKind, ArticleIntent: req.ArticleIntent,
	})
	req.AgentExecutor, req.AgentModel, req.AgentReasoningEffort = canonical.AgentExecutor, canonical.AgentModel, canonical.AgentReasoningEffort
	req.AgentSelectionSource, req.MCPMode, req.RigorLevel = canonical.AgentSelectionSource, canonical.MCPMode, canonical.RigorLevel
	req.ReportMode, req.PipelineFamily = canonical.ReportMode, canonical.PipelineFamily
	req.ReportSessionPolicy, req.ReportSessionPolicySelection = canonical.ReportSessionPolicy, canonical.ReportSessionPolicySelection
	req.PostReportHumanize, req.GenerationGuidanceProfile = canonical.PostReportHumanize, canonical.GenerationGuidanceProfile
	req.GenerationGuidanceSHA256 = canonical.GenerationGuidanceSHA256
	req.RetryStrategy, req.RetryOfPendingEventID, req.ResumeStage = canonical.RetryStrategy, canonical.RetryOfPendingEventID, canonical.ResumeStage
	req.ExecutionStrategy, req.Title, req.DirectionHint = canonical.ExecutionStrategy, canonical.Title, canonical.DirectionHint
	req.OutputKind, req.ArticleIntent = canonical.OutputKind, canonical.ArticleIntent
	return req, nil
}
