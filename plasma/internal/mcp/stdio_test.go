package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestStdioResearchPromptAndResourcesStaySmall(t *testing.T) {
	server := NewServer(&fakeMCPService{}, WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_1"}))

	init := handleRPC(context.Background(), server, rpcMessage{
		ID:     json.RawMessage(`1`),
		Method: "initialize",
		Params: json.RawMessage(`{"protocolVersion":"2024-11-05"}`),
	})
	initJSON := mustMarshalForTest(t, init.Result)
	if !strings.Contains(initJSON, "plasma.research.outline") ||
		!strings.Contains(initJSON, "source.observed") ||
		!strings.Contains(initJSON, "observation_event_id") ||
		!strings.Contains(initJSON, "plasma.sources.candidates.propose") ||
		!strings.Contains(initJSON, "copy source_uri into url and title into title") ||
		!strings.Contains(initJSON, "mis_1") ||
		!strings.Contains(initJSON, "ses_1") {
		t.Fatalf("initialize instructions missing research workflow or binding: %s", initJSON)
	}
	for _, forbidden := range []string{"plasma.agent_recall_preview", "plasma.evidence.propose", "plasma.claims.propose", "plasma.claims.confidence.update", "plasma.proposals.submit"} {
		if strings.Contains(initJSON, forbidden) {
			t.Fatalf("initialize instructions contain forbidden legacy marker %q: %s", forbidden, initJSON)
		}
	}

	resources := handleRPC(context.Background(), server, rpcMessage{ID: json.RawMessage(`2`), Method: "resources/list"})
	resourcesJSON := mustMarshalForTest(t, resources.Result)
	if !strings.Contains(resourcesJSON, `"resources":[]`) {
		t.Fatalf("resources/list should not expose mission data resources: %s", resourcesJSON)
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
}

func mustMarshalForTest(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal test value: %v", err)
	}
	return string(encoded)
}
