package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestStdioInitializeToolsPromptsAndDocsStaySmall(t *testing.T) {
	server := NewServer(&fakeMCPService{}, WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_1"}))

	init := handleRPC(context.Background(), server, rpcMessage{
		ID:     json.RawMessage(`1`),
		Method: "initialize",
		Params: json.RawMessage(`{"protocolVersion":"2024-11-05"}`),
	})
	initJSON := mustMarshalForTest(t, init.Result)
	if !strings.Contains(initJSON, `"protocolVersion":"2024-11-05"`) ||
		!strings.Contains(initJSON, `"tools":{}`) ||
		!strings.Contains(initJSON, `"resources":{}`) ||
		!strings.Contains(initJSON, "plasma://docs/mcp/") ||
		!strings.Contains(initJSON, "plasma.mermaid.validate") ||
		!strings.Contains(initJSON, "mis_1") ||
		!strings.Contains(initJSON, "ses_1") {
		t.Fatalf("initialize result missing capabilities, docs guidance, or binding: %s", initJSON)
	}
	if len(initJSON) > 1400 {
		t.Fatalf("initialize result grew too large: %d bytes", len(initJSON))
	}
	for _, forbidden := range []string{"plasma.agent_recall_preview", "plasma.evidence.propose", "plasma.claims.propose", "plasma.claims.confidence.update", "plasma.proposals.submit"} {
		if strings.Contains(initJSON, forbidden) {
			t.Fatalf("initialize instructions contain forbidden legacy marker %q: %s", forbidden, initJSON)
		}
	}

	resources := handleRPC(context.Background(), server, rpcMessage{ID: json.RawMessage(`2`), Method: "resources/list"})
	resourcesJSON := mustMarshalForTest(t, resources.Result)
	for _, uri := range []string{"plasma://docs/mcp/tools", "plasma://docs/mcp/errors", "plasma://docs/mcp/reporting", "plasma://docs/mcp/sources", "plasma://docs/mcp/mermaid"} {
		if !strings.Contains(resourcesJSON, uri) {
			t.Fatalf("resources/list missing %s: %s", uri, resourcesJSON)
		}
	}
	if strings.Contains(resourcesJSON, "mis_1") || strings.Contains(resourcesJSON, "ses_1") {
		t.Fatalf("resources/list exposed mission/session data: %s", resourcesJSON)
	}

	prompts := handleRPC(context.Background(), server, rpcMessage{ID: json.RawMessage(`3`), Method: "prompts/list"})
	promptsJSON := mustMarshalForTest(t, prompts.Result)
	if !strings.Contains(promptsJSON, "plasma.research.workflow") {
		t.Fatalf("prompts/list missing workflow prompt: %s", promptsJSON)
	}

	prompt := handleRPC(context.Background(), server, rpcMessage{
		ID:     json.RawMessage(`4`),
		Method: "prompts/get",
		Params: json.RawMessage(`{"name":"plasma.research.workflow"}`),
	})
	promptJSON := mustMarshalForTest(t, prompt.Result)
	if (!strings.Contains(promptJSON, "Grep") && !strings.Contains(promptJSON, "grep")) ||
		!strings.Contains(promptJSON, "source.observed") ||
		!strings.Contains(promptJSON, "observation_event_id") ||
		!strings.Contains(promptJSON, "copy source_uri into url and title into title") {
		t.Fatalf("prompt missing workflow guidance: %s", promptJSON)
	}
	for _, forbidden := range []string{"source_snapshot_ids", "evidence_ids", "report pack", "plasma.agent_recall_preview"} {
		if strings.Contains(promptJSON, forbidden) {
			t.Fatalf("prompt contains mission-data or pack marker %q: %s", forbidden, promptJSON)
		}
	}

	tools := handleRPC(context.Background(), server, rpcMessage{ID: json.RawMessage(`5`), Method: "tools/list"})
	toolsJSON := mustMarshalForTest(t, tools.Result)
	if !strings.Contains(toolsJSON, ToolSourcesRead) || !strings.Contains(toolsJSON, ToolMermaidValidate) || !strings.Contains(toolsJSON, "inputSchema") {
		t.Fatalf("tools/list lost default tool metadata: %s", toolsJSON)
	}

	call := handleRPC(context.Background(), server, rpcMessage{
		ID:     json.RawMessage(`6`),
		Method: "tools/call",
		Params: json.RawMessage(`{"name":"plasma.mermaid.validate","arguments":{"mission_id":"mis_1","source":"flowchart LR\nA --> B"}}`),
	})
	callJSON := mustMarshalForTest(t, call.Result)
	if !strings.Contains(callJSON, `"isError":false`) || !strings.Contains(callJSON, `\"tool_name\":\"plasma.mermaid.validate\"`) {
		t.Fatalf("tools/call lost result envelope: %s", callJSON)
	}
}

func TestStdioResourcesReadStaticMarkdown(t *testing.T) {
	server := NewServer(&fakeMCPService{}, WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_1"}))

	read := handleRPC(context.Background(), server, rpcMessage{
		ID:     json.RawMessage(`1`),
		Method: "resources/read",
		Params: json.RawMessage(`{"uri":"plasma://docs/mcp/tools"}`),
	})
	if read.Error != nil {
		t.Fatalf("resources/read returned error: %#v", read.Error)
	}
	readJSON := mustMarshalForTest(t, read.Result)
	if !strings.Contains(readJSON, `"mimeType":"text/markdown"`) ||
		!strings.Contains(readJSON, "# Plasma MCP Tool Guide") ||
		!strings.Contains(readJSON, "plasma.sources.read") {
		t.Fatalf("resources/read returned unexpected content: %s", readJSON)
	}
	if strings.Contains(readJSON, "mis_1") || strings.Contains(readJSON, "ses_1") {
		t.Fatalf("resources/read exposed mission/session data: %s", readJSON)
	}
}

func TestStdioResourcesReadRejectsUnknownAndInvalidURI(t *testing.T) {
	server := NewServer(&fakeMCPService{})

	unknown := handleRPC(context.Background(), server, rpcMessage{
		ID:     json.RawMessage(`1`),
		Method: "resources/read",
		Params: json.RawMessage(`{"uri":"plasma://docs/mcp/unknown"}`),
	})
	if unknown.Error == nil || unknown.Error.Code != resourceNotFoundCode || unknown.Error.Message != "resource not found" {
		t.Fatalf("unknown resource did not use not-found semantics: %#v", unknown)
	}

	invalid := handleRPC(context.Background(), server, rpcMessage{
		ID:     json.RawMessage(`2`),
		Method: "resources/read",
		Params: json.RawMessage(`{"uri":"not a uri"}`),
	})
	if invalid.Error == nil || invalid.Error.Code != -32602 || invalid.Error.Message != "invalid resource uri" {
		t.Fatalf("invalid resource URI did not use invalid params: %#v", invalid)
	}

	malformed := handleRPC(context.Background(), server, rpcMessage{
		ID:     json.RawMessage(`3`),
		Method: "resources/read",
		Params: json.RawMessage(`"invalid"`),
	})
	if malformed.Error == nil || malformed.Error.Code != -32602 {
		t.Fatalf("malformed resources/read params did not use invalid params: %#v", malformed)
	}
}

func mustMarshalForTest(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal test value: %v", err)
	}
	return string(encoded)
}
