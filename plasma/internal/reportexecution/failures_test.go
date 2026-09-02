package reportexecution

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/source"
)

type failureService struct {
	events []ledger.Event
}

func (s *failureService) AppendEvent(_ context.Context, req ledger.AppendRequest) (ledger.Event, error) {
	return ledger.Event{}, errors.New("unexpected single append")
}
func (s *failureService) AppendEvents(_ context.Context, _ string, _ []ledger.AppendRequest) ([]ledger.Event, error) {
	return nil, errors.New("unexpected append batch")
}
func (s *failureService) AppendReportTerminalIfOpen(_ context.Context, missionID, pendingID string, reqs []ledger.AppendRequest) ([]ledger.Event, bool, error) {
	for _, req := range reqs {
		event := ledger.Event{EventID: req.EventID, MissionID: missionID, EventType: req.EventType, Producer: req.Producer, CausationEventID: req.CausationEventID, CorrelationID: req.CorrelationID, Payload: req.Payload}
		s.events = append(s.events, event)
	}
	return s.events[len(s.events)-len(reqs):], true, nil
}
func (s *failureService) AppendEventsIfNoActiveAgentWork(context.Context, string, []ledger.AppendRequest) ([]ledger.Event, error) {
	return nil, errors.New("unexpected pending append")
}
func (s *failureService) ListEvents(context.Context, string) ([]ledger.Event, error) { return nil, nil }
func (s *failureService) ListSourceSnapshotsWithState(context.Context, source.ListRequest) ([]source.Snapshot, error) {
	return nil, nil
}

func TestStageFailureIDsCoverAllExperimentalStages(t *testing.T) {
	for _, stage := range []string{"source_packet", "il_source_selection", "il_narrative", "il_long_form_plan", "il_long_form_sections", "il_long_form_parts", "il_long_form_final", "il_continuity", "il_reader", "il_images", "il_document", "il_flow", "il_render", "il_store"} {
		if got := stageFailureID(stage, -1, -1); got != stage {
			t.Fatalf("stageFailureID(%q) = %q", stage, got)
		}
		if failure := NewStageFailure(stage, "", -1, -1, errors.New("cause")); failure.Retryable {
			t.Fatalf("experimental stage %q was marked retryable", stage)
		}
	}
}

func TestProviderFailureReceiptPreservesLongFormStages(t *testing.T) {
	usage := agentusage.New("codex", "codex", "gpt-5.6-luna", "xhigh", "private prompt").
		WithProviderUsage(agentusage.ProviderUsage{InputTokens: 13, OutputTokens: 5}, "provider")
	stages := []string{"il_long_form_plan", "il_long_form_sections", "il_long_form_parts", "il_long_form_final"}
	attempts := make([]ProviderAttemptUsageReceipt, 0, len(stages))
	for _, stage := range stages {
		attempts = append(attempts, NewProviderAttemptUsageReceipt(stage, 1, usage))
	}
	receipt := NewProviderFailureUsageReceipt(attempts)
	if len(receipt.Attempts) != len(stages) {
		t.Fatalf("long-form failure attempts = %#v", receipt.Attempts)
	}
	for index, attempt := range receipt.Attempts {
		if string(attempt.Stage) != stages[index] || attempt.Attempt != 1 {
			t.Fatalf("long-form failure attempt %d = %#v", index, attempt)
		}
	}
}

func TestAppendDraftFailedWritesStageCompanionThenDraftTerminal(t *testing.T) {
	service := &failureService{}
	ids := []string{"evt_draft_failed", "evt_stage_failure"}
	runner := Runner{Service: service, NewID: func(string) string {
		id := ids[0]
		ids = ids[1:]
		return id
	}}
	cause := NewStageFailure("il_flow", "", -1, -1, errors.New("provider secret response"))
	terminal, err := runner.AppendDraftFailed(context.Background(), "mis_failure", "evt_pending", "codex", "planned", cause)
	if err != nil {
		t.Fatal(err)
	}
	if len(service.events) != 2 || service.events[0].EventType != "report.il_flow.failed" || service.events[1].EventType != "report.draft.failed" || terminal.EventID != service.events[1].EventID {
		t.Fatalf("events = %#v", service.events)
	}
	var companion, draft map[string]any
	if err := json.Unmarshal(service.events[0].Payload, &companion); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(service.events[1].Payload, &draft); err != nil {
		t.Fatal(err)
	}
	if companion["stage_failure_event_id"] != nil || companion["pending_event_id"] != "evt_pending" || companion["stage_kind"] != "il_flow" || companion["stage_id"] != "il_flow" || companion["retryable"] != false {
		t.Fatalf("companion payload = %#v", companion)
	}
	if service.events[0].CorrelationID != service.events[1].EventID || companion["terminal_event_id"] != service.events[1].EventID || draft["stage_failure_event_id"] != service.events[0].EventID || draft["pending_event_id"] != "evt_pending" || draft["failed_stage_kind"] != "il_flow" || draft["failed_stage_id"] != "il_flow" {
		t.Fatalf("lineage payload/events = %#v / %#v", companion, draft)
	}
	if strings.Contains(string(service.events[0].Payload), "provider secret response") || strings.Contains(string(service.events[1].Payload), "provider secret response") {
		t.Fatal("raw cause leaked")
	}
}

func TestAppendDraftFailedCopiesSafeProviderReceiptToBothTerminals(t *testing.T) {
	service := &failureService{}
	ids := []string{"evt_draft_failed", "evt_stage_failure"}
	runner := Runner{Service: service, NewID: func(string) string {
		id := ids[0]
		ids = ids[1:]
		return id
	}}
	usage := agentusage.New("codex", "codex", "gpt-5.6-luna", "xhigh", "private prompt").
		WithProviderUsage(agentusage.ProviderUsage{InputTokens: 13, OutputTokens: 5}, "provider").
		WithDuration(42).
		WithSession("previous-secret", "returned-secret", true, false)
	receipt := NewProviderFailureUsageReceipt([]ProviderAttemptUsageReceipt{
		NewProviderAttemptUsageReceipt("il_source_selection", 1, usage),
		NewProviderAttemptUsageReceipt("il_document", 1, usage),
	})
	cause := NewStageFailure("il_document", "", -1, -1, errors.New("raw validator detail"))
	cause.SafeFailureReason = ProviderFailureReasonSemanticValidation
	cause.SafeValidationCode = ProviderValidationCodeTerminologySourceGrounding
	cause.ProviderUsage = &receipt
	if _, err := runner.AppendDraftFailed(context.Background(), "mis_failure", "evt_pending", "codex", "planned", cause); err != nil {
		t.Fatal(err)
	}
	if len(service.events) != 2 {
		t.Fatalf("events = %#v", service.events)
	}
	for _, event := range service.events {
		var payload map[string]any
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload["safe_failure_reason"] != string(ProviderFailureReasonSemanticValidation) || payload["safe_validation_code"] != string(ProviderValidationCodeTerminologySourceGrounding) {
			t.Fatalf("safe failure classification missing from %s: %#v", event.EventType, payload)
		}
		encoded := string(event.Payload)
		for _, forbidden := range []string{"raw validator detail", "private prompt", "previous-secret", "returned-secret", "\"prompt\"", "\"session\""} {
			if strings.Contains(encoded, forbidden) {
				t.Fatalf("%s leaked %q: %s", event.EventType, forbidden, encoded)
			}
		}
		var durable struct {
			Receipt ProviderFailureUsageReceipt `json:"provider_usage_receipt"`
		}
		if err := json.Unmarshal(event.Payload, &durable); err != nil {
			t.Fatal(err)
		}
		if len(durable.Receipt.Attempts) != 2 || durable.Receipt.Attempts[0].Stage != ProviderFailureStageSourceSelection || durable.Receipt.Attempts[0].Attempt != 1 || durable.Receipt.Attempts[0].Usage.ProviderUsage == nil || durable.Receipt.Attempts[0].Usage.ProviderUsage.TotalTokens != 18 || durable.Receipt.Attempts[1].Stage != ProviderFailureStageDocument || durable.Receipt.DurationMS != 84 {
			t.Fatalf("provider receipt in %s = %#v", event.EventType, durable.Receipt)
		}
	}
}

func TestProviderFailureReceiptDropsInvalidCoordinatesAndClosesMetadata(t *testing.T) {
	usage := agentusage.New("custom-provider", "custom-executor", "custom-model", "custom-effort", "private prompt").
		WithUnavailable("private unavailable detail")
	usage.UsageSource = "private usage source"
	usage.ContextWindow = &agentusage.ContextWindowMetrics{UsedTokens: 7, WindowTokens: 100, Source: "private context source"}
	receipt := NewProviderFailureUsageReceipt([]ProviderAttemptUsageReceipt{
		NewProviderAttemptUsageReceipt("il_document", 1, usage),
		NewProviderAttemptUsageReceipt("arbitrary-stage", 9, usage),
	})
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.Attempts) != 1 || receipt.Attempts[0].Stage != ProviderFailureStageDocument || receipt.Attempts[0].UnavailableReason != ProviderUsageUnavailableReasonNotEmitted {
		t.Fatalf("closed receipt = %#v", receipt)
	}
	for _, forbidden := range []string{"custom-provider", "custom-executor", "custom-model", "custom-effort", "private prompt", "private unavailable detail", "private usage source", "private context source", "arbitrary-stage", "usage_source"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("closed receipt leaked %q: %s", forbidden, encoded)
		}
	}
	if receipt.Attempts[0].Usage.ContextWindow == nil || receipt.Attempts[0].Usage.ContextWindow.UsedTokens != 7 {
		t.Fatalf("numeric context telemetry missing: %#v", receipt)
	}
}

func TestProviderValidationCodeAllowlistRejectsContentBearingValues(t *testing.T) {
	for _, code := range []ProviderValidationCode{
		ProviderValidationCodeSourceReadContract,
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
		ProviderValidationCodeSemanticContract,
	} {
		if !code.Valid() {
			t.Fatalf("closed validation code %q is invalid", code)
		}
	}
	for _, code := range []ProviderValidationCode{"source_001", "法樹寺", "private validator detail", ""} {
		if code.Valid() {
			t.Fatalf("content-bearing validation code %q was accepted", code)
		}
	}
}

func TestFailurePayloadAllowlistRejectsExperimentalAndUnknownMetadata(t *testing.T) {
	payload := map[string]any{}
	mergeFailurePayload(payload, failurePayloadError{payload: map[string]any{
		"safe_failure_reason":    ProviderFailureReasonDecode,
		"provider_usage_receipt": ProviderFailureUsageReceipt{},
		"raw_provider_response":  "secret",
	}})
	for _, key := range []string{"safe_failure_reason", "provider_usage_receipt", "raw_provider_response"} {
		if _, ok := payload[key]; ok {
			t.Fatalf("untyped payload key %q passed allowlist: %#v", key, payload)
		}
	}
}

type failurePayloadError struct{ payload map[string]any }

func (err failurePayloadError) Error() string                  { return "failed" }
func (err failurePayloadError) FailurePayload() map[string]any { return err.payload }
