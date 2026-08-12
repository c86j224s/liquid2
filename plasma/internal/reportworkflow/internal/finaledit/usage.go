package finaledit

import (
	"context"
	"log"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func recordStageUsage(ctx context.Context, store reporting.FinalEditStageStore, input Input, binding reporting.FinalEditStageBinding, stage reporting.FinalEditStageResult, result agentexec.AgentResult, durationMS int64) {
	agentSessionID := strings.TrimSpace(result.SessionID)
	if agentSessionID == "" {
		agentSessionID = binding.ProviderSessionID
	}
	surface := "report_" + binding.Stage
	if _, _, err := reporting.RecordReportAgentUsage(context.WithoutCancel(ctx), store, reporting.ReportAgentUsageRequest{
		MissionID: input.MissionID, PendingEventID: input.PendingEventID, CanonicalEventID: stage.Event.EventID,
		ForkSourceAgentSessionID: binding.ForkSourceAgentSessionID, Surface: surface,
		PreviousAgentSessionID: binding.ProviderSessionID, AgentSessionID: agentSessionID,
		DurationMS: durationMS, Resumed: result.Resumed, Usage: result.Usage,
	}); err != nil {
		log.Printf("report_agent_usage_write_failed mission_id=%q canonical_event_id=%q surface=%q err=%q", input.MissionID, stage.Event.EventID, surface, err)
	}
}
