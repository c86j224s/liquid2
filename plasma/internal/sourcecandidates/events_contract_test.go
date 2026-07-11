package sourcecandidates

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestBuildSourceCandidateProposalEventRequestPreservesWebPayload(t *testing.T) {
	event, ok, err := BuildSourceCandidateProposalEventRequest(SourceCandidateProposalEventRequest{
		EventID:       "evt_1",
		MissionID:     "mis_1",
		UserEventID:   "evt_user",
		AgentEventID:  "evt_agent",
		ExecutorName:  "codex",
		MCPMode:       "auto",
		ToolSessionID: "ses_tool",
		StrategyID:    "v2",
		Producer:      app.Producer{Type: "agent", ID: "codex"},
		Candidates: []SourceCandidateProposal{{
			URL:    "https://example.com/a",
			Title:  "Example",
			Reason: "useful",
			State:  "proposed",
		}},
	})
	if err != nil || !ok {
		t.Fatalf("BuildSourceCandidateProposalEventRequest returned ok=%v err=%v", ok, err)
	}
	if event.EventID != "evt_1" || event.MissionID != "mis_1" || event.EventType != "source.candidate.proposed" {
		t.Fatalf("unexpected event identity: %#v", event)
	}
	assertJSONPayload(t, event.Payload, map[string]any{
		"kind":            "source_candidate_proposed",
		"user_event_id":   "evt_user",
		"agent_event_id":  "evt_agent",
		"agent_executor":  "codex",
		"mcp_mode":        "auto",
		"tool_session_id": "ses_tool",
		"strategy_id":     "v2",
		"candidate_count": float64(1),
		"candidates": []any{map[string]any{
			"url":    "https://example.com/a",
			"title":  "Example",
			"reason": "useful",
			"state":  "proposed",
		}},
	})
}

func TestBuildSourceCandidateMCPProposalEventRequestPreservesMCPPayload(t *testing.T) {
	event, err := BuildSourceCandidateMCPProposalEventRequest(SourceCandidateMCPProposalEventRequest{
		EventID:            "evt_1",
		MissionID:          "mis_1",
		SessionID:          "ses_1",
		CurrentUserEventID: "evt_user",
		AgentExecutor:      "codex",
		Producer:           app.Producer{Type: "agent_session", ID: "ses_1"},
		Candidates: []SourceCandidateProposal{{
			URL:    "https://example.com/a",
			Title:  "Example",
			Reason: "useful",
			State:  "proposed",
		}},
	})
	if err != nil {
		t.Fatalf("BuildSourceCandidateMCPProposalEventRequest returned error: %v", err)
	}
	if event.EventID != "evt_1" || event.MissionID != "mis_1" || event.EventType != "source.candidate.proposed" || event.CorrelationID != "ses_1" {
		t.Fatalf("unexpected event identity: %#v", event)
	}
	assertJSONPayload(t, event.Payload, map[string]any{
		"kind":             "source_candidate_proposed",
		"source":           "mcp",
		"tool_session_id":  "ses_1",
		"agent_session_id": "ses_1",
		"user_event_id":    "evt_user",
		"agent_executor":   "codex",
		"candidate_count":  float64(1),
		"candidates": []any{map[string]any{
			"url":    "https://example.com/a",
			"title":  "Example",
			"reason": "useful",
			"state":  "proposed",
		}},
	})
}

func TestBuildWorkflowSourceCandidateProposalEventRequestPreservesWorkflowPayload(t *testing.T) {
	event, ok, err := BuildWorkflowSourceCandidateProposalEventRequest(WorkflowSourceCandidateProposalEventRequest{
		EventID:        "evt_1",
		MissionID:      "mis_1",
		WorkflowRunID:  "wfr_1",
		WorkflowStepID: "wfs_1",
		UserEventID:    "evt_user",
		AgentEventID:   "evt_agent",
		Producer:       app.Producer{Type: "agent", ID: "codex"},
		Candidates: []WorkflowSourceCandidateProposal{{
			URL:    "https://example.com/a",
			Title:  "Example",
			Reason: "useful",
			State:  "proposed",
		}, {
			URL:    "https://example.com/untitled",
			Reason: "useful untitled source",
			State:  "proposed",
		}},
	})
	if err != nil || !ok {
		t.Fatalf("BuildWorkflowSourceCandidateProposalEventRequest returned ok=%v err=%v", ok, err)
	}
	if event.EventID != "evt_1" || event.MissionID != "mis_1" || event.EventType != "source.candidate.proposed" ||
		event.Producer.Type != "agent" || event.Producer.ID != "codex" {
		t.Fatalf("unexpected event identity: %#v", event)
	}
	assertJSONPayload(t, event.Payload, map[string]any{
		"kind":             "source_candidate_proposed",
		"workflow_run_id":  "wfr_1",
		"workflow_step_id": "wfs_1",
		"user_event_id":    "evt_user",
		"agent_event_id":   "evt_agent",
		"candidates": []any{map[string]any{
			"url":    "https://example.com/a",
			"title":  "Example",
			"reason": "useful",
			"state":  "proposed",
		}, map[string]any{
			"url":    "https://example.com/untitled",
			"title":  "",
			"reason": "useful untitled source",
			"state":  "proposed",
		}},
	})
}

func TestSourceCandidateTerminalEventsPreserveSurfaceSpecificAgentExecutor(t *testing.T) {
	artifact := app.RawArtifact{
		ArtifactID: "art_1",
		MissionID:  "mis_1",
		MediaType:  "text/plain; charset=utf-8",
		ByteSize:   12,
		SHA256:     "abc",
	}
	fetched := SourceCandidateFetched{
		MediaType:       "text/plain; charset=utf-8",
		TextLengthKnown: true,
	}
	webJob := SourceCandidateStagingJob{
		MissionID:       "mis_1",
		SessionID:       "ses_1",
		ProposalEventID: "evt_proposed",
		Candidate: SourceCandidateProposal{
			URL:   "https://example.com/a",
			Title: "Example",
		},
		Producer:       app.Producer{Type: "agent", ID: "codex"},
		StartedEventID: "evt_started",
		AgentExecutor:  "codex",
	}
	assertPayloadLacksKey(t, sourceCandidateStagedEventRequest(webJob, "evt_staged", artifact, "Example", fetched).Payload, "agent_executor")
	assertPayloadLacksKey(t, sourceCandidateStagingFailedEventRequest(webJob, "evt_failed", errSourceCandidateTest).Payload, "agent_executor")

	mcpJob := webJob
	mcpJob.Producer = app.Producer{Type: "agent_session", ID: "ses_1"}
	mcpJob.EmitAgentExecutorInTerminalEvents = true
	assertJSONPayloadIncludes(t, sourceCandidateStagedEventRequest(mcpJob, "evt_staged", artifact, "Example", fetched).Payload, map[string]any{
		"agent_executor": "codex",
	})
	assertJSONPayloadIncludes(t, sourceCandidateStagingFailedEventRequest(mcpJob, "evt_failed", errSourceCandidateTest).Payload, map[string]any{
		"agent_executor": "codex",
	})
}

func assertJSONPayload(t *testing.T, raw []byte, want map[string]any) {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("payload mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func assertPayloadLacksKey(t *testing.T, raw []byte, key string) {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}
	if _, ok := got[key]; ok {
		t.Fatalf("payload unexpectedly contains %q: %#v", key, got)
	}
}

func assertJSONPayloadIncludes(t *testing.T, raw []byte, want map[string]any) {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}
	for key, value := range want {
		if !reflect.DeepEqual(got[key], value) {
			t.Fatalf("payload key %q mismatch: got %#v want %#v in %#v", key, got[key], value, got)
		}
	}
}

type sourceCandidateTestError string

func (e sourceCandidateTestError) Error() string {
	return string(e)
}

const errSourceCandidateTest = sourceCandidateTestError("candidate fetch failed")
