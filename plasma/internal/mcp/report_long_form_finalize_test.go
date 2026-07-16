package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestReportLongFormFinalizeSchemaIsClosed(t *testing.T) {
	var schema struct {
		AdditionalProperties bool                       `json:"additionalProperties"`
		Required             []string                   `json:"required"`
		Properties           map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(schemaReportLongFormFinalize, &schema); err != nil {
		t.Fatal(err)
	}
	want := []string{"mission_id", "session_id", "pending_event_id", "plan_event_id", "idempotency_key", "producer", "opening_markdown", "closing_markdown"}
	if schema.AdditionalProperties || len(schema.Properties) != len(want) || len(schema.Required) != len(want) {
		t.Fatalf("schema is not closed: %#v", schema)
	}
	for _, key := range want {
		if _, ok := schema.Properties[key]; !ok {
			t.Fatalf("missing schema property %q", key)
		}
	}
	for _, forbidden := range []string{"front_matter", "full_markdown", "parts", "part_artifact_ids", "title", "artifact_id", "report_mode"} {
		if _, ok := schema.Properties[forbidden]; ok {
			t.Fatalf("forbidden schema property %q", forbidden)
		}
	}
}

func TestReportLongFormFinalizeIsHiddenWithoutExactBinding(t *testing.T) {
	server := NewServer(&fakeMCPService{})
	for _, tool := range server.ListTools() {
		if tool.Name == ToolReportLongFormFinalize {
			t.Fatal("finalization tool leaked into the default tool list")
		}
	}
	binding := testLongFormFinalizeBinding()
	server = NewServer(&fakeMCPService{}, WithBinding(Binding{MissionID: binding.MissionID, AgentSessionID: binding.ToolSessionID, AgentExecutor: binding.AgentExecutor}), WithLongFormFinalizeBinding(binding), WithEnabledTools([]string{ToolReportLongFormFinalize}))
	count := 0
	for _, tool := range server.ListTools() {
		if tool.Name == ToolReportLongFormFinalize {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("finalization tool count=%d, want 1", count)
	}
}

func TestReportLongFormFinalizeRejectsUnknownArgumentsBeforeStorage(t *testing.T) {
	binding := testLongFormFinalizeBinding()
	server := NewServer(&fakeMCPService{}, WithBinding(Binding{MissionID: binding.MissionID, AgentSessionID: binding.ToolSessionID, AgentExecutor: binding.AgentExecutor}), WithLongFormFinalizeBinding(binding), WithEnabledTools([]string{ToolReportLongFormFinalize}))
	result := server.Call(context.Background(), ToolCall{Name: ToolReportLongFormFinalize, Arguments: json.RawMessage(`{"mission_id":"mis_1","session_id":"ses_1","pending_event_id":"evt_pending","plan_event_id":"evt_plan","idempotency_key":"key","producer":{"type":"agent_session","id":"ses_1"},"opening_markdown":"open","closing_markdown":"close","full_markdown":"no"}`)})
	if result.Error == nil || result.Error.ErrorKind != "validation" {
		t.Fatalf("unknown argument result=%#v", result)
	}
}

func testLongFormFinalizeBinding() reporting.LongFormFinalizeBinding {
	return reporting.LongFormFinalizeBinding{
		MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan", ArtifactID: "art_final", Filename: "report.md", Title: "Report",
		ToolSessionID: "ses_1", IdempotencyKey: "key", ProviderSessionID: "provider-1", PartArtifactIDs: []string{"art_part"},
		AgentExecutor: "codex", Producer: app.Producer{Type: "agent_session", ID: "provider-1"},
	}
}
