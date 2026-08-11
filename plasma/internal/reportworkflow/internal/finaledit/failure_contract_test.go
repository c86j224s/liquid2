package finaledit

import (
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestStageFailurePreservesFinalEditPayloadContract(t *testing.T) {
	usage := agentusage.New("openai", "codex", "model", "high", "prompt").WithProviderUsage(agentusage.ProviderUsage{
		InputTokens: 11, OutputTokens: 7,
	}, "provider")
	cause := AgentFailure(errors.New("provider failed"), agentexec.AgentResult{
		SessionID: "provider-reader", Resumed: true, Usage: usage,
	}, "report_"+reporting.FinalEditStageReader, 42, "provider-previous")
	err := StageFailure(reporting.FinalEditStageReader, "evt_plan", cause)

	var stage *reportexecution.StageFailureError
	if !errors.As(err, &stage) {
		t.Fatalf("error is not StageFailureError: %T %v", err, err)
	}
	if stage.Kind != "final" || stage.PlanEventID != "evt_plan" || stage.ID() != "final" || !stage.Retryable {
		t.Fatalf("stage failure coordinate changed: %#v", stage)
	}
	if stage.Cause == nil || stage.Cause.Error() != reporting.FinalEditStageReader+": provider failed" {
		t.Fatalf("stage cause changed: %v", stage.Cause)
	}

	var payloadProvider reportexecution.FailurePayloadProvider
	if !errors.As(err, &payloadProvider) {
		t.Fatalf("failure payload missing: %T %v", err, err)
	}
	payload := payloadProvider.FailurePayload()
	if payload["failed_surface"] != "report_"+reporting.FinalEditStageReader ||
		payload["agent_session_id"] != "provider-reader" || payload["resumed"] != true {
		t.Fatalf("session payload changed: %#v", payload)
	}
	eventUsage, ok := payload["agent_usage"].(agentusage.AgentUsage)
	if !ok {
		t.Fatalf("agent_usage missing or wrong type: %#v", payload)
	}
	if eventUsage.Surface != "report_"+reporting.FinalEditStageReader ||
		eventUsage.DurationMS != 42 ||
		eventUsage.Session.PreviousAgentSessionID != "provider-previous" ||
		eventUsage.Session.AgentSessionID != "provider-reader" ||
		!eventUsage.Session.Resumed ||
		eventUsage.ProviderUsage == nil ||
		eventUsage.ProviderUsage.TotalTokens != 18 {
		t.Fatalf("agent usage payload changed: %#v", eventUsage)
	}
}

func TestAgentFailureOmitsBlankSessionID(t *testing.T) {
	err := StageFailure(reporting.FinalEditStageReader, "evt_plan", AgentFailure(
		errors.New("provider failed"),
		agentexec.AgentResult{SessionID: "   "},
		"report_"+reporting.FinalEditStageReader,
		3,
		"provider-reader",
	))
	var payloadProvider reportexecution.FailurePayloadProvider
	if !errors.As(err, &payloadProvider) {
		t.Fatalf("failure payload missing: %T %v", err, err)
	}
	if _, ok := payloadProvider.FailurePayload()["agent_session_id"]; ok {
		t.Fatalf("blank returned SessionID must be omitted: %#v", payloadProvider.FailurePayload())
	}
}
