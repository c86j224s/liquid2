package reporting

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestRecordReportAgentUsageIsContentFreeAndIdempotent(t *testing.T) {
	store := &reportAgentUsageTestStore{}
	usage := agentusage.New("openai", "codex", "model", "high", "private prompt body").WithProviderUsage(agentusage.ProviderUsage{
		Scope: agentusage.UsageScopeSessionCumulative, InputTokens: 120, CachedInputTokens: 80, OutputTokens: 9,
	}, "provider")
	req := ReportAgentUsageRequest{
		MissionID: "mis_1", PendingEventID: "evt_pending", CanonicalEventID: "evt_part_edited",
		ForkSourceAgentSessionID: "session-plan", Surface: "report_part_edit",
		PreviousAgentSessionID: "session-part", AgentSessionID: "session-part", DurationMS: 42, Resumed: true, Usage: usage,
	}

	event, recorded, err := RecordReportAgentUsage(context.Background(), store, req)
	if err != nil || !recorded {
		t.Fatalf("RecordReportAgentUsage recorded=%t err=%v", recorded, err)
	}
	if event.EventID != "evt_report_usage_part_edited" || event.EventType != ReportAgentUsageRecordedEventType ||
		event.CausationEventID != req.CanonicalEventID || event.CorrelationID != req.PendingEventID {
		t.Fatalf("unexpected usage event: %#v", event)
	}
	if strings.Contains(string(event.Payload), "private prompt body") {
		t.Fatalf("usage event retained prompt content: %s", event.Payload)
	}
	var payload struct {
		CorrelationEventID       string                `json:"correlation_event_id"`
		ForkSourceAgentSessionID string                `json:"fork_source_agent_session_id"`
		AgentUsage               agentusage.AgentUsage `json:"agent_usage"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.CorrelationEventID != req.CanonicalEventID || payload.ForkSourceAgentSessionID != "session-plan" ||
		payload.AgentUsage.Surface != "report_part_edit" || payload.AgentUsage.Session.AgentSessionID != "session-part" ||
		payload.AgentUsage.ProviderUsage == nil || payload.AgentUsage.ProviderUsage.TotalTokens != 129 {
		t.Fatalf("unexpected usage payload: %#v", payload)
	}
	replayed, recorded, err := RecordReportAgentUsage(context.Background(), store, req)
	if err != nil || !recorded || replayed.EventID != event.EventID || len(store.events) != 1 {
		t.Fatalf("usage replay differs: event=%#v recorded=%t count=%d err=%v", replayed, recorded, len(store.events), err)
	}
}

func TestRecordReportAgentUsageSkipsEmptyEnvelope(t *testing.T) {
	store := &reportAgentUsageTestStore{}
	_, recorded, err := RecordReportAgentUsage(context.Background(), store, ReportAgentUsageRequest{
		MissionID: "mis_1", PendingEventID: "evt_pending", CanonicalEventID: "evt_result", AgentSessionID: "session-result",
	})
	if err != nil || recorded || len(store.events) != 0 {
		t.Fatalf("empty usage should not create an event: recorded=%t count=%d err=%v", recorded, len(store.events), err)
	}
}

type reportAgentUsageTestStore struct {
	events []app.LedgerEvent
}

func (store *reportAgentUsageTestStore) AppendEventConditionally(_ context.Context, missionID string, decide func([]app.LedgerEvent) (app.AppendEventRequest, app.LedgerEvent, bool, error)) (app.LedgerEvent, bool, error) {
	request, existing, appendEvent, err := decide(append([]app.LedgerEvent(nil), store.events...))
	if err != nil || !appendEvent {
		return existing, false, err
	}
	event := app.LedgerEvent{
		EventID: request.EventID, MissionID: missionID, EventType: request.EventType, Producer: request.Producer,
		CausationEventID: request.CausationEventID, CorrelationID: request.CorrelationID, Payload: request.Payload,
	}
	store.events = append(store.events, event)
	return event, true, nil
}
