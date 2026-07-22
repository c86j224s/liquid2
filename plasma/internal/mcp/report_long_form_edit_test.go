package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestReportLongFormEditToolsRequireNarrativeBindingAndExplicitEnablement(t *testing.T) {
	tools := []string{ToolReportLongFormEditStart, ToolReportLongFormEditRead, ToolReportLongFormEditPatch, ToolReportLongFormEditSubmit}
	binding := testNarrativeLongFormFinalizeBinding()
	session := Binding{MissionID: binding.MissionID, AgentSessionID: binding.ToolSessionID, AgentExecutor: binding.AgentExecutor}
	for _, tc := range []struct {
		name    string
		binding reporting.LongFormFinalizeBinding
		enabled []string
		want    bool
	}{
		{name: "default"},
		{name: "binding only", binding: binding},
		{name: "enable only", enabled: tools},
		{name: "narrative bound", binding: binding, enabled: tools, want: true},
		{name: "preserve bound", binding: testLongFormFinalizeBinding(), enabled: tools},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := NewServer(&fakeMCPService{}, WithBinding(session), WithLongFormFinalizeBinding(tc.binding), WithEnabledTools(tc.enabled))
			listed := toolNames(server.ListTools())
			for _, name := range tools {
				if containsString(listed, name) != tc.want {
					t.Fatalf("tool %s visibility=%v, want %v: %#v", name, containsString(listed, name), tc.want, listed)
				}
			}
		})
	}
	server := NewServer(&fakeMCPService{}, WithBinding(session), WithLongFormFinalizeBinding(binding), WithEnabledTools([]string{ToolReportLongFormEditStart}))
	listed := toolNames(server.ListTools())
	if !containsString(listed, ToolReportLongFormEditStart) {
		t.Fatalf("explicitly enabled start tool was hidden: %#v", listed)
	}
	for _, name := range []string{ToolReportLongFormEditRead, ToolReportLongFormEditPatch, ToolReportLongFormEditSubmit} {
		if containsString(listed, name) {
			t.Fatalf("disabled tool %s was listed: %#v", name, listed)
		}
	}
}

func TestReportLongFormEditToolsFinalizeEditedBoundManuscript(t *testing.T) {
	binding := testNarrativeLongFormFinalizeBinding()
	service := seededLongFormEditService(binding)
	server := NewServer(service,
		WithBinding(Binding{MissionID: binding.MissionID, AgentSessionID: binding.ToolSessionID, AgentExecutor: binding.AgentExecutor}),
		WithLongFormFinalizeBinding(binding),
		WithEnabledTools([]string{ToolReportLongFormEditStart, ToolReportLongFormEditRead, ToolReportLongFormEditPatch, ToolReportLongFormEditSubmit}),
	)
	common := map[string]any{
		"mission_id": binding.MissionID,
		"session_id": binding.ToolSessionID,
		"producer":   map[string]any{"type": "agent_session", "id": binding.ToolSessionID},
	}
	startArgs := cloneMap(common)
	startArgs["idempotency_key"] = "edit-start"
	startArgs["draft_id"] = "rfe_test"
	startArgs["pending_event_id"] = binding.PendingEventID
	startArgs["plan_event_id"] = binding.PlanEventID
	start := server.Call(context.Background(), ToolCall{Name: ToolReportLongFormEditStart, Arguments: mustArgs(t, startArgs)})
	if start.Error != nil {
		t.Fatalf("start failed: %#v", start.Error)
	}

	read := server.Call(context.Background(), ToolCall{Name: ToolReportLongFormEditRead, Arguments: mustArgs(t, map[string]any{
		"mission_id": binding.MissionID, "session_id": binding.ToolSessionID, "draft_id": "rfe_test", "max_bytes": 65536,
	})})
	if read.Error != nil || !strings.Contains(read.Content.(map[string]any)["content"].(string), "Preserved body.") {
		t.Fatalf("read did not expose bound manuscript: %#v", read)
	}

	patchArgs := cloneMap(common)
	patchArgs["idempotency_key"] = "edit-patch-1"
	patchArgs["draft_id"] = "rfe_test"
	patchArgs["operation"] = "replace"
	patchArgs["match_text"] = "Preserved body."
	patchArgs["replacement"] = "Edited body."
	patchArgs["summary"] = "explain the Part directly"
	if result := server.Call(context.Background(), ToolCall{Name: ToolReportLongFormEditPatch, Arguments: mustArgs(t, patchArgs)}); result.Error != nil {
		t.Fatalf("patch failed: %#v", result.Error)
	}

	submitArgs := cloneMap(common)
	submitArgs["idempotency_key"] = "edit-submit"
	submitArgs["draft_id"] = "rfe_test"
	submitArgs["pending_event_id"] = binding.PendingEventID
	submitArgs["plan_event_id"] = binding.PlanEventID
	submit := server.Call(context.Background(), ToolCall{Name: ToolReportLongFormEditSubmit, Arguments: mustArgs(t, submitArgs)})
	if submit.Error != nil || len(submit.CreatedEventIDs) != 1 {
		t.Fatalf("submit failed: %#v", submit)
	}
	artifact := service.artifacts[binding.ArtifactID]
	if !strings.Contains(string(artifact.Content), "Edited body.") || strings.Contains(string(artifact.Content), "Preserved body.") {
		t.Fatalf("final artifact did not preserve exact edit: %q", artifact.Content)
	}
	var canonical app.LedgerEvent
	for _, event := range service.ledgerEvents {
		if event.EventType == "report.artifact.created" {
			canonical = event
		}
	}
	if canonical.EventID == "" {
		t.Fatalf("canonical report event missing: %#v", service.ledgerEvents)
	}
	var payload map[string]any
	if err := json.Unmarshal(canonical.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["composition_strategy"] != reporting.LongFormCompositionNarrativeEdit || payload["assembly_strategy"] != "narrative_contract_final_edit" {
		t.Fatalf("unexpected final metadata: %#v", payload)
	}
}

func TestReportLongFormEditSchemasAreClosed(t *testing.T) {
	for name, schema := range map[string]json.RawMessage{
		ToolReportLongFormEditStart: schemaReportLongFormEditStart, ToolReportLongFormEditRead: schemaReportLongFormEditRead,
		ToolReportLongFormEditPatch: schemaReportLongFormEditPatch, ToolReportLongFormEditSubmit: schemaReportLongFormEditSubmit,
	} {
		var value struct {
			AdditionalProperties bool `json:"additionalProperties"`
		}
		if err := json.Unmarshal(schema, &value); err != nil || value.AdditionalProperties {
			t.Fatalf("%s schema is not closed: value=%#v err=%v", name, value, err)
		}
	}
}

func testNarrativeLongFormFinalizeBinding() reporting.LongFormFinalizeBinding {
	binding := testLongFormFinalizeBinding()
	binding.SectionArtifactIDs = []string{"art_section"}
	binding.SectionWordCount = 2
	binding.CompositionStrategy = reporting.LongFormCompositionNarrativeEdit
	binding.PreviousProviderSessionID = binding.ProviderSessionID
	binding.GenerationGuidanceProfile = "narrative-contract"
	return binding
}

func seededLongFormEditService(binding reporting.LongFormFinalizeBinding) *fakeMCPService {
	producer := binding.Producer
	part := app.RawArtifact{ArtifactID: binding.PartArtifactIDs[0], MissionID: binding.MissionID, MediaType: "text/markdown; charset=utf-8", Filename: "part.md", Producer: producer, Content: []byte("# Part 1\n\nPreserved body.\n")}
	return &fakeMCPService{
		artifacts: map[string]app.RawArtifact{part.ArtifactID: part},
		ledgerEvents: []app.LedgerEvent{
			{EventID: binding.PendingEventID, MissionID: binding.MissionID, EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: mustJSON(map[string]any{"report_mode": "long_form"})},
			{EventID: binding.PlanEventID, MissionID: binding.MissionID, EventType: "report.plan.created", Producer: producer, Payload: mustJSON(map[string]any{"pending_event_id": binding.PendingEventID, "report_mode": "long_form", "artifact_id": binding.ArtifactID})},
			{EventID: "evt_part", MissionID: binding.MissionID, EventType: "report.part.created", Producer: producer, Payload: mustJSON(map[string]any{"pending_event_id": binding.PendingEventID, "plan_event_id": binding.PlanEventID, "artifact_id": part.ArtifactID, "part_index": 1})},
			{EventID: "evt_section", MissionID: binding.MissionID, EventType: "report.section.created", Producer: producer, Payload: mustJSON(map[string]any{"pending_event_id": binding.PendingEventID, "plan_event_id": binding.PlanEventID, "artifact_id": binding.SectionArtifactIDs[0], "part_index": 1, "section_index": 1})},
		},
	}
}
