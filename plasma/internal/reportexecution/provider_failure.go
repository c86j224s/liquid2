package reportexecution

import "github.com/c86j224s/liquid2/plasma/internal/agentusage"

// ProviderFailureReason is a closed, durable classification that never includes
// provider output or validator detail.
type ProviderFailureReason string

const (
	ProviderFailureReasonTransport          ProviderFailureReason = "transport"
	ProviderFailureReasonDecode             ProviderFailureReason = "decode"
	ProviderFailureReasonSemanticValidation ProviderFailureReason = "semantic_validation"
)

// Valid reports whether the reason is safe to persist.
func (reason ProviderFailureReason) Valid() bool {
	switch reason {
	case ProviderFailureReasonTransport,
		ProviderFailureReasonDecode,
		ProviderFailureReasonSemanticValidation:
		return true
	default:
		return false
	}
}

// ProviderValidationCode is a closed, non-content-bearing explanation for a
// semantic rejection. It never contains provider output, source text, terms,
// locators, or validator prose.
type ProviderValidationCode string

const (
	ProviderValidationCodeSourceReadContract                ProviderValidationCode = "source_read_contract"
	ProviderValidationCodeLanguageReview                    ProviderValidationCode = "language_review"
	ProviderValidationCodeTerminologyInventory              ProviderValidationCode = "terminology_inventory"
	ProviderValidationCodeTerminologySourceGrounding        ProviderValidationCode = "terminology_source_grounding"
	ProviderValidationCodeTerminologySourceFormGrounding    ProviderValidationCode = "terminology_source_form_grounding"
	ProviderValidationCodeTerminologySourceReadingGrounding ProviderValidationCode = "terminology_source_reading_grounding"
	ProviderValidationCodeTerminologyPresentation           ProviderValidationCode = "terminology_presentation"
	ProviderValidationCodeTerminologyRemoval                ProviderValidationCode = "terminology_removal"
	ProviderValidationCodeTerminologyRename                 ProviderValidationCode = "terminology_rename"
	ProviderValidationCodeTerminologyReaderForm             ProviderValidationCode = "terminology_reader_form"
	ProviderValidationCodeTerminologyFirstUse               ProviderValidationCode = "terminology_first_use"
	ProviderValidationCodeTerminologySourceForm             ProviderValidationCode = "terminology_source_form"
	ProviderValidationCodeTerminologySourceReading          ProviderValidationCode = "terminology_source_reading"
	ProviderValidationCodeTerminologyAliasPlacement         ProviderValidationCode = "terminology_alias_placement"
	ProviderValidationCodeTerminologyAliasCollision         ProviderValidationCode = "terminology_alias_collision"
	ProviderValidationCodeTerminologyScriptCoverage         ProviderValidationCode = "terminology_script_coverage"
	ProviderValidationCodeReaderFacingContent               ProviderValidationCode = "reader_facing_content"
	ProviderValidationCodeReaderOpening                     ProviderValidationCode = "reader_opening"
	ProviderValidationCodeReaderAuditVoice                  ProviderValidationCode = "reader_audit_voice"
	ProviderValidationCodeReaderOrdinarySI                  ProviderValidationCode = "reader_ordinary_si"
	ProviderValidationCodeReaderProcess                     ProviderValidationCode = "reader_process"
	ProviderValidationCodeReaderMetadata                    ProviderValidationCode = "reader_metadata"
	ProviderValidationCodeReaderInternalMachinery           ProviderValidationCode = "reader_internal_machinery"
	ProviderValidationCodeReaderUnexplainedTerm             ProviderValidationCode = "reader_unexplained_term"
	ProviderValidationCodeSupportedDetail                   ProviderValidationCode = "supported_detail"
	ProviderValidationCodeReportDepth                       ProviderValidationCode = "report_depth"
	ProviderValidationCodeClaimStrength                     ProviderValidationCode = "claim_strength"
	ProviderValidationCodeEvidenceSupportInventory          ProviderValidationCode = "evidence_support_inventory"
	ProviderValidationCodeEvidenceSupportReceipt            ProviderValidationCode = "evidence_support_receipt"
	ProviderValidationCodeEvidenceSupportTarget             ProviderValidationCode = "evidence_support_target"
	ProviderValidationCodeEvidenceSupportBinding            ProviderValidationCode = "evidence_support_binding"
	ProviderValidationCodeEvidenceSupportQuote              ProviderValidationCode = "evidence_support_quote"
	ProviderValidationCodeEvidenceSupportUnsupported        ProviderValidationCode = "evidence_support_unsupported"
	ProviderValidationCodeEvidenceSupportLevel              ProviderValidationCode = "evidence_support_level"
	ProviderValidationCodeEvidenceSupportCoverage           ProviderValidationCode = "evidence_support_coverage"
	ProviderValidationCodeEvidencePacketInventory           ProviderValidationCode = "evidence_packet_inventory"
	ProviderValidationCodeEvidencePacketTarget              ProviderValidationCode = "evidence_packet_target"
	ProviderValidationCodeEvidencePacketSource              ProviderValidationCode = "evidence_packet_source"
	ProviderValidationCodeEvidencePacketBinding             ProviderValidationCode = "evidence_packet_binding"
	ProviderValidationCodeEvidencePacketCoverage            ProviderValidationCode = "evidence_packet_coverage"
	ProviderValidationCodeDocumentContract                  ProviderValidationCode = "document_contract"
	ProviderValidationCodeSemanticContract                  ProviderValidationCode = "semantic_contract"
)

// Valid reports whether the validation code is safe to persist.
func (code ProviderValidationCode) Valid() bool {
	switch code {
	case ProviderValidationCodeSourceReadContract,
		ProviderValidationCodeLanguageReview,
		ProviderValidationCodeTerminologyInventory,
		ProviderValidationCodeTerminologySourceGrounding,
		ProviderValidationCodeTerminologySourceFormGrounding,
		ProviderValidationCodeTerminologySourceReadingGrounding,
		ProviderValidationCodeTerminologyPresentation,
		ProviderValidationCodeTerminologyRemoval,
		ProviderValidationCodeTerminologyRename,
		ProviderValidationCodeTerminologyReaderForm,
		ProviderValidationCodeTerminologyFirstUse,
		ProviderValidationCodeTerminologySourceForm,
		ProviderValidationCodeTerminologySourceReading,
		ProviderValidationCodeTerminologyAliasPlacement,
		ProviderValidationCodeTerminologyAliasCollision,
		ProviderValidationCodeTerminologyScriptCoverage,
		ProviderValidationCodeReaderFacingContent,
		ProviderValidationCodeReaderOpening,
		ProviderValidationCodeReaderAuditVoice,
		ProviderValidationCodeReaderOrdinarySI,
		ProviderValidationCodeReaderProcess,
		ProviderValidationCodeReaderMetadata,
		ProviderValidationCodeReaderInternalMachinery,
		ProviderValidationCodeReaderUnexplainedTerm,
		ProviderValidationCodeSupportedDetail,
		ProviderValidationCodeReportDepth,
		ProviderValidationCodeClaimStrength,
		ProviderValidationCodeEvidenceSupportInventory,
		ProviderValidationCodeEvidenceSupportReceipt,
		ProviderValidationCodeEvidenceSupportTarget,
		ProviderValidationCodeEvidenceSupportBinding,
		ProviderValidationCodeEvidenceSupportQuote,
		ProviderValidationCodeEvidenceSupportUnsupported,
		ProviderValidationCodeEvidenceSupportLevel,
		ProviderValidationCodeEvidenceSupportCoverage,
		ProviderValidationCodeEvidencePacketInventory,
		ProviderValidationCodeEvidencePacketTarget,
		ProviderValidationCodeEvidencePacketSource,
		ProviderValidationCodeEvidencePacketBinding,
		ProviderValidationCodeEvidencePacketCoverage,
		ProviderValidationCodeDocumentContract,
		ProviderValidationCodeSemanticContract:
		return true
	default:
		return false
	}
}

// ProviderUsageUnavailableReason is a closed explanation for missing token data.
type ProviderUsageUnavailableReason string

const ProviderUsageUnavailableReasonNotEmitted ProviderUsageUnavailableReason = "provider_usage_not_emitted"

// ProviderFailureStage is the closed set of provider-backed Experimental stages.
type ProviderFailureStage string

const (
	ProviderFailureStageSourceSelection  ProviderFailureStage = "il_source_selection"
	ProviderFailureStageEditorialMemory  ProviderFailureStage = "il_editorial_memory"
	ProviderFailureStageNarrative        ProviderFailureStage = "il_narrative"
	ProviderFailureStageLongFormPlan     ProviderFailureStage = "il_long_form_plan"
	ProviderFailureStageLongFormSections ProviderFailureStage = "il_long_form_sections"
	ProviderFailureStageLongFormParts    ProviderFailureStage = "il_long_form_parts"
	ProviderFailureStageLongFormFinal    ProviderFailureStage = "il_long_form_final"
	ProviderFailureStageContinuity       ProviderFailureStage = "il_continuity"
	ProviderFailureStageReader           ProviderFailureStage = "il_reader"
	ProviderFailureStageImages           ProviderFailureStage = "il_images"
	ProviderFailureStageDocument         ProviderFailureStage = "il_document"
	ProviderFailureStageFlow             ProviderFailureStage = "il_flow"
)

// SafeProviderTokenUsage keeps only numeric provider usage fields.
type SafeProviderTokenUsage struct {
	InputTokens           int `json:"input_tokens,omitempty"`
	CachedInputTokens     int `json:"cached_input_tokens,omitempty"`
	UncachedInputTokens   int `json:"uncached_input_tokens,omitempty"`
	OutputTokens          int `json:"output_tokens,omitempty"`
	ReasoningOutputTokens int `json:"reasoning_output_tokens,omitempty"`
	TotalTokens           int `json:"total_tokens,omitempty"`
}

// SafeContextWindow keeps only numeric telemetry and drops provider-owned source text.
type SafeContextWindow struct {
	UsedTokens   int `json:"used_tokens"`
	WindowTokens int `json:"window_tokens"`
}

// SafeProviderUsage is the durable subset of one provider attempt. Prompt,
// session, and provider-owned free-form strings are structurally absent.
type SafeProviderUsage struct {
	SchemaVersion          int                            `json:"schema_version"`
	ProviderUsage          *SafeProviderTokenUsage        `json:"provider_usage,omitempty"`
	ContextWindow          *SafeContextWindow             `json:"context_window,omitempty"`
	DurationMS             int64                          `json:"duration_ms,omitempty"`
	UsageUnavailable       bool                           `json:"usage_unavailable"`
	UsageUnavailableReason ProviderUsageUnavailableReason `json:"usage_unavailable_reason,omitempty"`
}

// ProviderAttemptUsageReceipt binds safe provider usage to one stage attempt.
type ProviderAttemptUsageReceipt struct {
	Stage             ProviderFailureStage           `json:"stage"`
	Attempt           int                            `json:"attempt"`
	Usage             SafeProviderUsage              `json:"usage"`
	UsageUnavailable  bool                           `json:"usage_unavailable"`
	UnavailableReason ProviderUsageUnavailableReason `json:"unavailable_reason,omitempty"`
}

// ProviderFailureUsageReceipt preserves external cost evidence when a report
// stage fails before a success manifest can be stored.
type ProviderFailureUsageReceipt struct {
	Attempts          []ProviderAttemptUsageReceipt  `json:"attempts"`
	DurationMS        int64                          `json:"duration_ms"`
	UsageUnavailable  bool                           `json:"usage_unavailable"`
	UnavailableReason ProviderUsageUnavailableReason `json:"unavailable_reason,omitempty"`
}

// NewProviderAttemptUsageReceipt removes prompt and session fields while
// preserving provider-emitted token and duration evidence.
func NewProviderAttemptUsageReceipt(stage string, attempt int, usage agentusage.AgentUsage) ProviderAttemptUsageReceipt {
	durationMS := usage.DurationMS
	if durationMS < 0 {
		durationMS = 0
	}
	safe := SafeProviderUsage{
		SchemaVersion: agentusage.SchemaVersion,
		ProviderUsage: safeProviderUsage(usage.ProviderUsage),
		ContextWindow: safeContextWindow(usage.ContextWindow),
		DurationMS:    durationMS,
	}
	receipt := ProviderAttemptUsageReceipt{Stage: ProviderFailureStage(stage), Attempt: attempt, Usage: safe}
	if safe.ProviderUsage == nil {
		receipt.UsageUnavailable = true
		receipt.UnavailableReason = ProviderUsageUnavailableReasonNotEmitted
		receipt.Usage.UsageUnavailable = true
		receipt.Usage.UsageUnavailableReason = receipt.UnavailableReason
	}
	return receipt
}

// NewProviderFailureUsageReceipt aggregates already-sanitized attempt receipts.
func NewProviderFailureUsageReceipt(attempts []ProviderAttemptUsageReceipt) ProviderFailureUsageReceipt {
	receipt := ProviderFailureUsageReceipt{Attempts: make([]ProviderAttemptUsageReceipt, 0, len(attempts))}
	for _, source := range attempts {
		stage, ok := validProviderFailureStage(source.Stage)
		if !ok || source.Attempt < 1 || source.Attempt > 2 {
			continue
		}
		attempt := source
		attempt.Stage = stage
		attempt.Usage.SchemaVersion = agentusage.SchemaVersion
		attempt.Usage.ProviderUsage = cloneSafeProviderUsage(attempt.Usage.ProviderUsage)
		attempt.Usage.ContextWindow = cloneSafeContextWindow(attempt.Usage.ContextWindow)
		if attempt.Usage.DurationMS < 0 {
			attempt.Usage.DurationMS = 0
		}
		if attempt.Usage.ProviderUsage == nil {
			attempt.UsageUnavailable = true
			attempt.UnavailableReason = ProviderUsageUnavailableReasonNotEmitted
			attempt.Usage.UsageUnavailable = true
			attempt.Usage.UsageUnavailableReason = attempt.UnavailableReason
		} else {
			attempt.UsageUnavailable = false
			attempt.UnavailableReason = ""
			attempt.Usage.UsageUnavailable = false
			attempt.Usage.UsageUnavailableReason = ""
		}
		receipt.Attempts = append(receipt.Attempts, attempt)
		receipt.DurationMS += attempt.Usage.DurationMS
		if attempt.UsageUnavailable {
			receipt.UsageUnavailable = true
			if receipt.UnavailableReason == "" {
				receipt.UnavailableReason = attempt.UnavailableReason
			}
		}
	}
	return receipt
}

func validProviderFailureStage(stage ProviderFailureStage) (ProviderFailureStage, bool) {
	switch stage {
	case ProviderFailureStageSourceSelection, ProviderFailureStageEditorialMemory, ProviderFailureStageNarrative, ProviderFailureStageLongFormPlan, ProviderFailureStageLongFormSections, ProviderFailureStageLongFormParts, ProviderFailureStageLongFormFinal, ProviderFailureStageContinuity, ProviderFailureStageReader, ProviderFailureStageImages, ProviderFailureStageDocument, ProviderFailureStageFlow:
		return stage, true
	default:
		return "", false
	}
}

func safeProviderUsage(usage *agentusage.ProviderUsage) *SafeProviderTokenUsage {
	if usage == nil {
		return nil
	}
	return &SafeProviderTokenUsage{
		InputTokens:           nonNegative(usage.InputTokens),
		CachedInputTokens:     nonNegative(usage.CachedInputTokens),
		UncachedInputTokens:   nonNegative(usage.UncachedInputTokens),
		OutputTokens:          nonNegative(usage.OutputTokens),
		ReasoningOutputTokens: nonNegative(usage.ReasoningOutputTokens),
		TotalTokens:           nonNegative(usage.TotalTokens),
	}
}

func cloneSafeProviderUsage(usage *SafeProviderTokenUsage) *SafeProviderTokenUsage {
	if usage == nil {
		return nil
	}
	cloned := *usage
	return &cloned
}

func safeContextWindow(metrics *agentusage.ContextWindowMetrics) *SafeContextWindow {
	if metrics == nil {
		return nil
	}
	return &SafeContextWindow{UsedTokens: nonNegative(metrics.UsedTokens), WindowTokens: nonNegative(metrics.WindowTokens)}
}

func nonNegative(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func cloneSafeContextWindow(metrics *SafeContextWindow) *SafeContextWindow {
	if metrics == nil {
		return nil
	}
	cloned := *metrics
	return &cloned
}
