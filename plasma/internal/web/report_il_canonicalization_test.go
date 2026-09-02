package web

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"github.com/c86j224s/liquid2/plasma/internal/reportpipeline"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func TestStartReportDraftCanonicalizesExperimentalPendingAgainstClassicMission(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	server := NewServer(svc, Options{AgentExecutor: errorAgentExecutor{err: context.Canceled}}).(*Server)
	missionID := createMissionForTest(t, ctx, svc)
	if _, err := svc.AppendEvent(ctx, app.AppendEventRequest{
		EventID: "evt_classic_lock", MissionID: missionID, EventType: "turn.agent.response",
		Producer: app.Producer{Type: "agent", ID: "claude"},
		Payload:  mustJSON(map[string]any{"agent_executor": "claude"}),
	}); err != nil {
		t.Fatal(err)
	}
	result, err := server.startReportDraft(ctx, missionID, reportDraftRequest{
		Title: "experimental", AgentExecutor: "claude", AgentModel: "wrong", AgentReasoningEffort: "low",
		AgentSelectionSource: "mission", MCPMode: "auto", RigorLevel: "balanced", ReportMode: reportModeLongForm,
		PipelineFamily: reportilcontract.PipelineFamily, ExecutionStrategy: reportExecutionStrategySectionFanout,
		ReportSessionPolicy: reportSessionPolicySameSession, ReportSessionPolicySelection: "default", PostReportHumanize: "enabled",
	})
	if err != nil {
		t.Fatal(err)
	}
	pending, ok := result["pending_event"].(app.LedgerEvent)
	if !ok {
		t.Fatalf("pending event missing: %#v", result)
	}
	var payload map[string]any
	if err := json.Unmarshal(pending.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["pipeline_family"] != reportilcontract.PipelineFamily || payload["pipeline_graph"] != reportpipeline.ExperimentalILValidationProfilesGraph || payload["report_mode"] != reportModeLongForm || payload["agent_executor"] != "codex" || payload["agent_model"] != "gpt-5.6-luna" || payload["agent_reasoning_effort"] != "xhigh" || payload["agent_selection_source"] != "experimental_fixed" || payload["mcp_mode"] != "source_read_only" || payload["report_session_policy"] != reportSessionPolicyFreshSession || payload["report_session_policy_selection"] != "experimental_fixed" || payload["post_report_humanize"] != "disabled" || payload["humanize_enabled"] != false || payload["rigor_level"] != "strict" || payload["rigor_label"] != "검증형" || payload["execution_strategy"] != nil {
		t.Fatalf("experimental pending payload is not canonical: %#v", payload)
	}
}

func TestStartReportDraftPreservesExperimentalAuthoringAndValidationProfiles(t *testing.T) {
	for _, mode := range []string{reportModePlanned, reportModeLongForm} {
		for _, tc := range []struct {
			level string
			label string
		}{
			{level: "unverified", label: "무검증"},
			{level: "exploratory", label: "탐색형"},
			{level: "strict", label: "검증형"},
		} {
			t.Run(mode+"/"+tc.level, func(t *testing.T) {
				ctx := context.Background()
				store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
				if err != nil {
					t.Fatal(err)
				}
				defer store.Close()
				svc := app.NewService(store)
				server := NewServer(svc, Options{AgentExecutor: errorAgentExecutor{err: context.Canceled}}).(*Server)
				missionID := createMissionForTest(t, ctx, svc)
				result, err := server.startReportDraft(ctx, missionID, reportDraftRequest{
					Title: "experimental", ReportMode: mode,
					PipelineFamily: reportilcontract.PipelineFamily, RigorLevel: tc.level,
				})
				if err != nil {
					t.Fatal(err)
				}
				pending, ok := result["pending_event"].(app.LedgerEvent)
				if !ok {
					t.Fatalf("pending event missing: %#v", result)
				}
				var payload map[string]any
				if err := json.Unmarshal(pending.Payload, &payload); err != nil {
					t.Fatal(err)
				}
				if payload["pipeline_family"] != reportilcontract.PipelineFamily ||
					payload["pipeline_graph"] != reportpipeline.ExperimentalILValidationProfilesGraph ||
					payload["report_mode"] != mode || payload["rigor_level"] != tc.level || payload["rigor_label"] != tc.label {
					t.Fatalf("experimental profiles = %#v", payload)
				}
			})
		}
	}
}
