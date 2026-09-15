package reportrun

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/reportusage"
)

func TestCompleteReportRunActualRecordedExactOutcome(t *testing.T) {
	s := completionTestStore{events: baseCompletionEvents()}
	usage := agentusage.New("", "codex", "m", "r", "prompt").WithSurface("report_requirements").WithSession("ses_1", "ses_1", false, false).WithProviderUsage(agentusage.ProviderUsage{Scope: agentusage.UsageScopeCall, InputTokens: 4, OutputTokens: 2}, "provider")
	got, err := CompleteReportRun(context.Background(), &s, ReportCompletionRequest{MissionID: "mis_1", CanonicalEventID: "evt_art", ActualUsage: &reportusage.ReportAgentUsageRequest{MissionID: "mis_1", CanonicalEventID: "evt_target", AgentSessionID: "ses_1", PreviousAgentSessionID: "ses_1", Surface: "report_requirements", Usage: usage}})
	if err != nil {
		t.Fatal(err)
	}
	var completion struct {
		Recorded    int `json:"usage_recorded_count"`
		Unavailable int `json:"usage_unavailable_count"`
	}
	_ = json.Unmarshal(got.Payload, &completion)
	if completion.Recorded != 1 || completion.Unavailable != 0 {
		t.Fatalf("completion=%#v", completion)
	}
	usageEvent, ok := eventByID(s.events, "evt_report_usage_target")
	if !ok {
		t.Fatal("actual usage event missing")
	}
	if usageEvent.EventType != reportusage.ReportAgentUsageRecordedEventType || usageEvent.Producer.ID != "ses_1" || usageEvent.CausationEventID != "evt_target" || usageEvent.CorrelationID != "evt_root" {
		t.Fatalf("usage envelope=%#v", usageEvent)
	}
	if len(s.events) != 5 || s.events[3].EventID != "evt_report_usage_target" || s.events[4].EventID != got.EventID {
		t.Fatalf("completion append order = [%s, %s], want usage before completion", s.events[3].EventID, s.events[4].EventID)
	}
}

func TestCompleteReportRunActualUnavailableValidReason(t *testing.T) {
	s := completionTestStore{events: baseCompletionEvents()}
	usage := agentusage.New("", "codex", "m", "r", "prompt").WithSurface("report_requirements").WithSession("ses_1", "ses_1", false, false).WithUnavailable("provider temporarily unavailable")
	got, err := CompleteReportRun(context.Background(), &s, ReportCompletionRequest{MissionID: "mis_1", CanonicalEventID: "evt_art", ActualUsage: &reportusage.ReportAgentUsageRequest{MissionID: "mis_1", CanonicalEventID: "evt_target", AgentSessionID: "ses_1", PreviousAgentSessionID: "ses_1", Surface: "report_requirements", Usage: usage}})
	if err != nil {
		t.Fatal(err)
	}
	var completion struct {
		Recorded    int `json:"usage_recorded_count"`
		Unavailable int `json:"usage_unavailable_count"`
	}
	_ = json.Unmarshal(got.Payload, &completion)
	if completion.Recorded != 0 || completion.Unavailable != 1 {
		t.Fatalf("completion=%#v", completion)
	}
}

func TestCompleteReportRunRejectsActualInvalidOutcomeAndRollsBack(t *testing.T) {
	cases := []struct {
		name  string
		usage agentusage.AgentUsage
	}{
		{"missing unavailable", agentusage.New("", "codex", "m", "r", "prompt").WithSurface("report_requirements").WithSession("ses_1", "ses_1", false, false)},
		{"provider plus unavailable", func() agentusage.AgentUsage {
			u := agentusage.New("", "codex", "m", "r", "prompt").WithSurface("report_requirements").WithSession("ses_1", "ses_1", false, false).WithProviderUsage(agentusage.ProviderUsage{InputTokens: 1}, "provider")
			u.UsageUnavailable = true
			return u
		}()},
		{"empty unavailable reason", agentusage.New("", "codex", "m", "r", "prompt").WithSurface("report_requirements").WithSession("ses_1", "ses_1", false, false).WithUnavailable("")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := completionTestStore{events: baseCompletionEvents()}
			_, err := CompleteReportRun(context.Background(), &s, ReportCompletionRequest{MissionID: "mis_1", CanonicalEventID: "evt_art", ActualUsage: &reportusage.ReportAgentUsageRequest{MissionID: "mis_1", CanonicalEventID: "evt_target", AgentSessionID: "ses_1", PreviousAgentSessionID: "ses_1", Surface: "report_requirements", Usage: tc.usage}})
			if err == nil || len(s.events) != 3 {
				t.Fatalf("err=%v events=%d", err, len(s.events))
			}
		})
	}
}
