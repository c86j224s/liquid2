package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestReportPartEditToolsRequireBindingAndExplicitEnablement(t *testing.T) {
	binding := testPartEditBinding()
	session := Binding{MissionID: binding.MissionID, AgentSessionID: binding.ToolSessionID, AgentExecutor: binding.AgentExecutor}
	allTools := []string{ToolReportPartEditStart, ToolReportPartEditRead, ToolReportPartEditPatch, ToolReportPartEditSubmit}
	for _, tc := range []struct {
		name    string
		options []Option
		want    bool
	}{
		{"default", []Option{WithBinding(session)}, false},
		{"binding only", []Option{WithBinding(session), WithPartEditBinding(binding)}, false},
		{"enable only", []Option{WithBinding(session), WithEnabledTools(allTools)}, false},
		{"bound Part editor", []Option{WithBinding(session), WithPartEditBinding(binding), WithEnabledTools(allTools)}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			listed := toolNames(NewServer(&fakeMCPService{}, tc.options...).ListTools())
			for _, name := range allTools {
				if containsString(listed, name) != tc.want {
					t.Fatalf("tool %s visibility=%v, want %v: %#v", name, containsString(listed, name), tc.want, listed)
				}
			}
		})
	}
}

func TestReportPartEditToolsSubmitChangedAndNoOpBoundPart(t *testing.T) {
	for _, tc := range []struct {
		name       string
		patch      bool
		wantID     string
		wantText   string
		wantChange bool
	}{
		{name: "no-op", wantID: "art_part", wantText: "# Part 1\n\nSource body.\n", wantChange: false},
		{name: "changed", patch: true, wantID: "art_part_edit", wantText: "# Part 1\n\nEdited body.\n", wantChange: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			binding := testPartEditBinding()
			service := seededPartEditService(binding)
			server := NewServer(service,
				WithBinding(Binding{MissionID: binding.MissionID, AgentSessionID: binding.ToolSessionID, AgentExecutor: binding.AgentExecutor}),
				WithPartEditBinding(binding),
				WithEnabledTools([]string{ToolReportPartEditStart, ToolReportPartEditRead, ToolReportPartEditPatch, ToolReportPartEditSubmit}),
			)
			common := map[string]any{
				"mission_id": binding.MissionID,
				"session_id": binding.ToolSessionID,
				"producer":   map[string]any{"type": "agent_session", "id": binding.ToolSessionID},
			}
			startArgs := cloneMap(common)
			startArgs["idempotency_key"] = "part-edit-start"
			startArgs["draft_id"] = "rpe_test"
			startArgs["pending_event_id"] = binding.PendingEventID
			startArgs["plan_event_id"] = binding.PlanEventID
			startArgs["part_index"] = binding.PartIndex
			startArgs["source_artifact_id"] = binding.SourceArtifactID
			start := server.Call(context.Background(), ToolCall{Name: ToolReportPartEditStart, Arguments: mustArgs(t, startArgs)})
			if start.Error != nil {
				t.Fatalf("start failed: %#v", start.Error)
			}
			if countMCPEvents(service.ledgerEvents, reporting.PartEditStartedEventType) != 1 {
				t.Fatalf("start did not persist exactly one durable event: %#v", service.ledgerEvents)
			}
			read := server.Call(context.Background(), ToolCall{Name: ToolReportPartEditRead, Arguments: mustArgs(t, map[string]any{
				"mission_id": binding.MissionID, "session_id": binding.ToolSessionID, "draft_id": "rpe_test", "max_bytes": 65536,
			})})
			if read.Error != nil || !strings.Contains(read.Content.(map[string]any)["content"].(string), "Source body.") {
				t.Fatalf("read failed: %#v", read)
			}
			if tc.patch {
				patchArgs := cloneMap(common)
				patchArgs["idempotency_key"] = "part-edit-patch"
				patchArgs["draft_id"] = "rpe_test"
				patchArgs["operation"] = "replace"
				patchArgs["match_text"] = "Source body."
				patchArgs["replacement"] = "Edited body."
				patchArgs["summary"] = "tighten Part"
				if result := server.Call(context.Background(), ToolCall{Name: ToolReportPartEditPatch, Arguments: mustArgs(t, patchArgs)}); result.Error != nil {
					t.Fatalf("patch failed: %#v", result.Error)
				}
			}
			submitArgs := cloneMap(common)
			submitArgs["idempotency_key"] = "part-edit-submit"
			submitArgs["draft_id"] = "rpe_test"
			submitArgs["pending_event_id"] = binding.PendingEventID
			submitArgs["plan_event_id"] = binding.PlanEventID
			submit := server.Call(context.Background(), ToolCall{Name: ToolReportPartEditSubmit, Arguments: mustArgs(t, submitArgs)})
			if submit.Error != nil || len(submit.CreatedEventIDs) != 1 {
				t.Fatalf("submit failed: %#v", submit)
			}
			state := submit.Content.(map[string]any)
			if state["artifact_id"] != tc.wantID || state["submitted"] != true {
				t.Fatalf("unexpected submitted state: %#v", state)
			}
			artifact := service.artifacts[tc.wantID]
			if string(artifact.Content) != tc.wantText {
				t.Fatalf("unexpected artifact content: %q", artifact.Content)
			}
			payload := map[string]any{}
			for _, event := range service.ledgerEvents {
				if event.EventType == reporting.PartEditedEventType {
					if err := json.Unmarshal(event.Payload, &payload); err != nil {
						t.Fatal(err)
					}
				}
			}
			if payload["artifact_id"] != tc.wantID || payload["changed"] != tc.wantChange {
				t.Fatalf("unexpected Part edit event payload: %#v", payload)
			}
			if countMCPEvents(service.ledgerEvents, reporting.PartEditStartedEventType) != 1 {
				t.Fatalf("submit changed durable start count: %#v", service.ledgerEvents)
			}
		})
	}
}

func TestReportPartEditStartPersistsAndReplaysDurableBinding(t *testing.T) {
	ctx := context.Background()
	binding := testPartEditBinding()
	service := seededPartEditService(binding)
	server := boundPartEditMCPServer(service, binding)
	startArgs := partEditStartArgs(binding, "rpe_direct")

	start := server.Call(ctx, ToolCall{Name: ToolReportPartEditStart, Arguments: mustArgs(t, startArgs)})
	if start.Error != nil {
		t.Fatalf("direct start failed: %#v", start.Error)
	}
	if countMCPEvents(service.ledgerEvents, reporting.PartEditStartedEventType) != 1 {
		t.Fatalf("direct start event count differs: %#v", service.ledgerEvents)
	}
	payload := lastMCPEventPayload(t, service.ledgerEvents, reporting.PartEditStartedEventType)
	for key, want := range map[string]any{
		"pending_event_id":                binding.PendingEventID,
		"plan_event_id":                   binding.PlanEventID,
		"source_part_event_id":            binding.SourcePartEventID,
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
		"report_session_policy":           binding.ReportSessionPolicy,
		"report_session_policy_selection": binding.ReportSessionPolicySelection,
		"generation_guidance_profile":     binding.GenerationGuidanceProfile,
		"generation_guidance_sha256":      binding.GenerationGuidanceSHA256,
		"session_chain_kind":              binding.SessionChainKind,
		"report_plan_session_id":          binding.ReportPlanSessionID,
		"fork_source_agent_session_id":    binding.ForkSourceAgentSessionID,
	} {
		if got := payload[key]; got != want {
			t.Fatalf("payload[%s]=%#v, want %#v; payload=%#v", key, got, want, payload)
		}
	}

	prestartedService := seededPartEditService(binding)
	started, _, err := reporting.StartPartEdit(ctx, prestartedService, "evt_web_prestarted", binding)
	if err != nil {
		t.Fatal(err)
	}
	prestartedServer := boundPartEditMCPServer(prestartedService, binding)
	replayArgs := partEditStartArgs(binding, "rpe_replay")
	replay := prestartedServer.Call(ctx, ToolCall{Name: ToolReportPartEditStart, Arguments: mustArgs(t, replayArgs)})
	if replay.Error != nil {
		t.Fatalf("prestarted replay failed: %#v", replay.Error)
	}
	if countMCPEvents(prestartedService.ledgerEvents, reporting.PartEditStartedEventType) != 1 ||
		lastMCPEvent(prestartedService.ledgerEvents, reporting.PartEditStartedEventType).EventID != started.EventID {
		t.Fatalf("prestarted MCP replay did not preserve canonical start: %#v", prestartedService.ledgerEvents)
	}
}

func TestReportPartEditStartPropagatesDurableMismatch(t *testing.T) {
	ctx := context.Background()
	binding := testPartEditBinding()
	service := seededPartEditService(binding)
	conflict := binding
	conflict.ToolSessionID = "ses_other_tool"
	if _, err := service.AppendEvent(ctx, reporting.BuildPartEditStartedAppendRequest("evt_conflicting_start", conflict)); err != nil {
		t.Fatal(err)
	}
	result := boundPartEditMCPServer(service, binding).Call(ctx, ToolCall{Name: ToolReportPartEditStart, Arguments: mustArgs(t, partEditStartArgs(binding, "rpe_conflict"))})
	if result.Error == nil || result.Error.ErrorKind != "conflict" {
		t.Fatalf("mismatch error=%#v, want conflict", result.Error)
	}
	if countMCPEvents(service.ledgerEvents, reporting.PartEditStartedEventType) != 1 {
		t.Fatalf("mismatch appended a new start: %#v", service.ledgerEvents)
	}
}

func TestReportPartEditStartDoesNotPersistBeforeSourceValidation(t *testing.T) {
	binding := testPartEditBinding()
	service := seededPartEditService(binding)
	artifact := service.artifacts[binding.SourceArtifactID]
	artifact.Content = []byte{0xff, 0xfe}
	service.artifacts[binding.SourceArtifactID] = artifact

	result := boundPartEditMCPServer(service, binding).Call(context.Background(), ToolCall{Name: ToolReportPartEditStart, Arguments: mustArgs(t, partEditStartArgs(binding, "rpe_invalid_source"))})
	if result.Error == nil || result.Error.ErrorKind != "validation" {
		t.Fatalf("error=%#v, want validation", result.Error)
	}
	if countMCPEvents(service.ledgerEvents, reporting.PartEditStartedEventType) != 0 {
		t.Fatalf("invalid source appended durable start: %#v", service.ledgerEvents)
	}
}

func TestReportPartEditSchemasAreClosed(t *testing.T) {
	for name, schema := range map[string]json.RawMessage{
		ToolReportPartEditStart:  schemaReportPartEditStart,
		ToolReportPartEditRead:   schemaReportPartEditRead,
		ToolReportPartEditPatch:  schemaReportPartEditPatch,
		ToolReportPartEditSubmit: schemaReportPartEditSubmit,
	} {
		var value struct {
			AdditionalProperties bool `json:"additionalProperties"`
		}
		if err := json.Unmarshal(schema, &value); err != nil || value.AdditionalProperties {
			t.Fatalf("%s schema is not closed: value=%#v err=%v", name, value, err)
		}
	}
}

func testPartEditBinding() reporting.PartEditBinding {
	return reporting.PartEditBinding{
		MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan",
		SourcePartEventID: "evt_part", SourceArtifactID: "art_part", EditedArtifactID: "art_part_edit",
		Filename: "part-1-edited.md", ToolSessionID: "ses_tool", ProviderSessionID: "provider-editor",
		PreviousProviderSessionID: "provider-editor", IdempotencyKey: "report-part-edit:evt_pending:evt_plan:1", PartIndex: 1,
		AgentExecutor: "codex", AgentModel: "model", AgentReasoningEffort: "medium",
		AgentSelectionSource: "request", MCPMode: "auto", ReportSessionPolicy: "isolated_fork",
		ReportSessionPolicySelection: "default", GenerationGuidanceProfile: "narrative-contract",
		GenerationGuidanceSHA256: "guidance-sha", SessionChainKind: "section_fanout_report",
		ReportPlanSessionID: "provider-plan", ForkSourceAgentSessionID: "provider-plan",
	}
}

func boundPartEditMCPServer(service *fakeMCPService, binding reporting.PartEditBinding) *Server {
	return NewServer(service,
		WithBinding(Binding{MissionID: binding.MissionID, AgentSessionID: binding.ToolSessionID, AgentExecutor: binding.AgentExecutor}),
		WithPartEditBinding(binding),
		WithEnabledTools([]string{ToolReportPartEditStart, ToolReportPartEditRead, ToolReportPartEditPatch, ToolReportPartEditSubmit}),
	)
}

func partEditStartArgs(binding reporting.PartEditBinding, draftID string) map[string]any {
	return map[string]any{
		"mission_id": binding.MissionID, "session_id": binding.ToolSessionID,
		"producer":        map[string]any{"type": "agent_session", "id": binding.ToolSessionID},
		"idempotency_key": binding.IdempotencyKey + ":start", "draft_id": draftID,
		"pending_event_id": binding.PendingEventID, "plan_event_id": binding.PlanEventID,
		"part_index": binding.PartIndex, "source_artifact_id": binding.SourceArtifactID,
	}
}

func countMCPEvents(events []app.LedgerEvent, eventType string) int {
	count := 0
	for _, event := range events {
		if event.EventType == eventType {
			count++
		}
	}
	return count
}

func lastMCPEvent(events []app.LedgerEvent, eventType string) app.LedgerEvent {
	var found app.LedgerEvent
	for _, event := range events {
		if event.EventType == eventType {
			found = event
		}
	}
	return found
}

func lastMCPEventPayload(t *testing.T, events []app.LedgerEvent, eventType string) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(lastMCPEvent(events, eventType).Payload, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func seededPartEditService(binding reporting.PartEditBinding) *fakeMCPService {
	part := app.RawArtifact{
		ArtifactID: binding.SourceArtifactID, MissionID: binding.MissionID,
		MediaType: "text/markdown; charset=utf-8", Filename: "part-1.md",
		Producer: app.Producer{Type: "agent_session", ID: "provider-part"}, Content: []byte("# Part 1\n\nSource body.\n"),
	}
	return &fakeMCPService{
		artifacts: map[string]app.RawArtifact{part.ArtifactID: part},
		ledgerEvents: []app.LedgerEvent{
			{EventID: binding.PendingEventID, MissionID: binding.MissionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: mustJSON(map[string]any{"report_mode": "long_form"})},
			{EventID: binding.PlanEventID, MissionID: binding.MissionID, EventType: "report.plan.created", Producer: app.Producer{Type: "agent_session", ID: "provider-plan"}, Payload: mustJSON(map[string]any{"pending_event_id": binding.PendingEventID, "report_mode": "long_form", "artifact_id": "art_final", "part_edit_enabled": true})},
			{EventID: binding.SourcePartEventID, MissionID: binding.MissionID, EventType: "report.part.created", Producer: app.Producer{Type: "agent_session", ID: "provider-part"}, Payload: mustJSON(map[string]any{"pending_event_id": binding.PendingEventID, "plan_event_id": binding.PlanEventID, "artifact_id": binding.SourceArtifactID, "part_index": binding.PartIndex})},
		},
	}
}
