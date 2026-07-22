package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestReportPartAssemblyToolsRequireBindingAndExplicitEnablement(t *testing.T) {
	binding := testPartAssemblyBinding()
	sessionBinding := Binding{MissionID: binding.MissionID, AgentSessionID: binding.ToolSessionID, AgentExecutor: binding.AgentExecutor}
	allTools := []string{
		ToolReportPartAssemblyStart,
		ToolReportPartAssemblyRead,
		ToolReportPartAssemblyPatch,
		ToolReportPartAssemblySubmit,
	}
	cases := []struct {
		name    string
		options []Option
		want    bool
	}{
		{"default", []Option{WithBinding(sessionBinding)}, false},
		{"binding only", []Option{WithBinding(sessionBinding), WithPartAssemblyBinding(binding)}, false},
		{"enable only", []Option{WithBinding(sessionBinding), WithEnabledTools(allTools)}, false},
		{"bound part assembly session", []Option{WithBinding(sessionBinding), WithPartAssemblyBinding(binding), WithEnabledTools(allTools)}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tools := toolNames(NewServer(&fakeMCPService{}, tc.options...).ListTools())
			for _, name := range allTools {
				if containsString(tools, name) != tc.want {
					t.Fatalf("tool %s visibility=%v, want %v: %#v", name, containsString(tools, name), tc.want, tools)
				}
			}
		})
	}
}

func TestReportPartAssemblyToolsSubmitConnectiveEvent(t *testing.T) {
	binding := testPartAssemblyBinding()
	service := &fakeMCPService{}
	server := NewServer(
		service,
		WithBinding(Binding{MissionID: binding.MissionID, AgentSessionID: binding.ToolSessionID, AgentExecutor: binding.AgentExecutor}),
		WithPartAssemblyBinding(binding),
		WithEnabledTools([]string{
			ToolReportPartAssemblyStart,
			ToolReportPartAssemblyRead,
			ToolReportPartAssemblyPatch,
			ToolReportPartAssemblySubmit,
		}),
	)

	start := server.Call(context.Background(), ToolCall{Name: ToolReportPartAssemblyStart, Arguments: mustArgs(t, map[string]any{
		"mission_id":       binding.MissionID,
		"session_id":       binding.ToolSessionID,
		"idempotency_key":  "start",
		"producer":         map[string]any{"type": "agent_session", "id": binding.ToolSessionID},
		"draft_id":         "rpa_test",
		"pending_event_id": binding.PendingEventID,
		"plan_event_id":    binding.PlanEventID,
		"part_index":       binding.PartIndex,
		"section_count":    binding.SectionCount,
	})})
	if start.Error != nil {
		t.Fatalf("start failed: %#v", start.Error)
	}

	patches := []map[string]any{
		{"field": "intro", "markdown": "파트 도입입니다.", "summary": "intro"},
		{"field": "transition", "after_section_index": 1, "markdown": "다음 섹션으로 넘어갑니다.", "summary": "transition"},
		{"field": "closing", "markdown": "파트 마무리입니다.", "summary": "closing"},
	}
	for index, patch := range patches {
		args := map[string]any{
			"mission_id":      binding.MissionID,
			"session_id":      binding.ToolSessionID,
			"idempotency_key": "patch-" + string(rune('a'+index)),
			"producer":        map[string]any{"type": "agent_session", "id": binding.ToolSessionID},
			"draft_id":        "rpa_test",
		}
		for key, value := range patch {
			args[key] = value
		}
		result := server.Call(context.Background(), ToolCall{Name: ToolReportPartAssemblyPatch, Arguments: mustArgs(t, args)})
		if result.Error != nil {
			t.Fatalf("patch %d failed: %#v", index, result.Error)
		}
	}

	submit := server.Call(context.Background(), ToolCall{Name: ToolReportPartAssemblySubmit, Arguments: mustArgs(t, map[string]any{
		"mission_id":       binding.MissionID,
		"session_id":       binding.ToolSessionID,
		"idempotency_key":  "submit",
		"producer":         map[string]any{"type": "agent_session", "id": binding.ToolSessionID},
		"draft_id":         "rpa_test",
		"pending_event_id": binding.PendingEventID,
		"plan_event_id":    binding.PlanEventID,
	})})
	if submit.Error != nil {
		t.Fatalf("submit failed: %#v", submit.Error)
	}
	if len(submit.CreatedEventIDs) != 1 {
		t.Fatalf("expected one created event, got %#v", submit.CreatedEventIDs)
	}

	submission, exists, err := reporting.LoadPartAssemblySubmission(context.Background(), service, binding)
	if err != nil || !exists {
		t.Fatalf("submission not readable: exists=%v err=%v", exists, err)
	}
	if submission.Assembly.Intro != "파트 도입입니다." || submission.Assembly.Closing != "파트 마무리입니다." || len(submission.Assembly.Transitions) != 1 {
		t.Fatalf("unexpected assembly: %#v", submission.Assembly)
	}
	var submitted *app.AppendEventRequest
	for i := range service.events {
		if service.events[i].EventType == reporting.PartAssemblySubmittedEventType {
			submitted = &service.events[i]
		}
	}
	if submitted == nil {
		t.Fatalf("missing part assembly event: %#v", service.events)
	}
	if bytes := string(submitted.Payload); strings.Contains(bytes, "Section body") {
		t.Fatalf("part assembly event leaked section body: %s", bytes)
	}
}

func TestReportPartAssemblySchemasAreClosed(t *testing.T) {
	for name, schema := range map[string]json.RawMessage{
		ToolReportPartAssemblyStart:  schemaReportPartAssemblyStart,
		ToolReportPartAssemblyRead:   schemaReportPartAssemblyRead,
		ToolReportPartSectionRead:    schemaReportPartSectionRead,
		ToolReportPartAssemblyPatch:  schemaReportPartAssemblyPatch,
		ToolReportPartAssemblySubmit: schemaReportPartAssemblySubmit,
	} {
		var value struct {
			AdditionalProperties bool `json:"additionalProperties"`
		}
		if err := json.Unmarshal(schema, &value); err != nil {
			t.Fatalf("%s schema invalid: %v", name, err)
		}
		if value.AdditionalProperties {
			t.Fatalf("%s schema allows unknown properties", name)
		}
	}
}

func TestReportPartSectionReadUsesOnlyBoundArtifactIndexes(t *testing.T) {
	binding := testPartAssemblyBinding()
	binding.SectionArtifactIDs = []string{"art_section_1", "art_section_2", "art_section_3"}
	service := &fakeMCPService{artifacts: map[string]app.RawArtifact{
		"art_section_1": {ArtifactID: "art_section_1", MissionID: binding.MissionID, MediaType: "text/markdown; charset=utf-8", Content: []byte("# 첫 섹션\n\n본문을 직접 읽습니다.")},
		"art_section_2": {ArtifactID: "art_section_2", MissionID: "mis_other", MediaType: "text/markdown; charset=utf-8", Content: []byte("foreign")},
		"art_section_3": {ArtifactID: "art_section_3", MissionID: binding.MissionID, MediaType: "application/octet-stream", Content: []byte("binary")},
	}}
	server := NewServer(
		service,
		WithBinding(Binding{MissionID: binding.MissionID, AgentSessionID: binding.ToolSessionID, AgentExecutor: binding.AgentExecutor}),
		WithPartAssemblyBinding(binding),
		WithEnabledTools([]string{ToolReportPartSectionRead}),
	)
	if !containsString(toolNames(server.ListTools()), ToolReportPartSectionRead) {
		t.Fatal("bound Part Section read tool was not exposed")
	}
	read := server.Call(context.Background(), ToolCall{Name: ToolReportPartSectionRead, Arguments: mustArgs(t, map[string]any{
		"mission_id": binding.MissionID, "session_id": binding.ToolSessionID, "section_index": 1, "max_bytes": 12,
	})})
	if read.Error != nil {
		t.Fatalf("bound Section read failed: %#v", read.Error)
	}
	content, ok := read.Content.(map[string]any)
	if !ok || content["section_index"] != 1 || content["content"] == "" || content["truncated"] != true {
		t.Fatalf("unexpected bounded Section read: %#v", read.Content)
	}
	for _, index := range []int{0, 4} {
		result := server.Call(context.Background(), ToolCall{Name: ToolReportPartSectionRead, Arguments: mustArgs(t, map[string]any{
			"mission_id": binding.MissionID, "session_id": binding.ToolSessionID, "section_index": index,
		})})
		if result.Error == nil || result.Error.ErrorKind != "validation" {
			t.Fatalf("out-of-bound Section index %d was accepted: %#v", index, result)
		}
	}
	foreign := server.Call(context.Background(), ToolCall{Name: ToolReportPartSectionRead, Arguments: mustArgs(t, map[string]any{
		"mission_id": binding.MissionID, "session_id": binding.ToolSessionID, "section_index": 2,
	})})
	if foreign.Error == nil || foreign.Error.ErrorKind != "conflict" {
		t.Fatalf("foreign bound artifact was readable: %#v", foreign)
	}
}

func TestReportPartSectionReadStaysHiddenWithoutCompleteArtifactBinding(t *testing.T) {
	binding := testPartAssemblyBinding()
	server := NewServer(
		&fakeMCPService{},
		WithBinding(Binding{MissionID: binding.MissionID, AgentSessionID: binding.ToolSessionID, AgentExecutor: binding.AgentExecutor}),
		WithPartAssemblyBinding(binding),
		WithEnabledTools([]string{ToolReportPartSectionRead}),
	)
	if containsString(toolNames(server.ListTools()), ToolReportPartSectionRead) {
		t.Fatal("Part Section read tool was exposed without bound Section artifacts")
	}
}

func testPartAssemblyBinding() reporting.PartAssemblyBinding {
	return reporting.PartAssemblyBinding{
		MissionID:            "mis_1",
		PendingEventID:       "evt_pending",
		PlanEventID:          "evt_plan",
		ToolSessionID:        "ses_tool",
		ProviderSessionID:    "provider-1",
		PartIndex:            1,
		SectionCount:         3,
		AgentExecutor:        "codex",
		AgentModel:           "gpt-test",
		AgentReasoningEffort: "medium",
		Producer:             app.Producer{Type: "agent_session", ID: "ses_tool"},
	}
}
