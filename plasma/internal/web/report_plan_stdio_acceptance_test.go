package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	plasmamcp "github.com/c86j224s/liquid2/plasma/internal/mcp"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

type stdioReportPlanExecutor struct {
	binary                         string
	database                       string
	delegate                       *fakeAgentExecutor
	malformedFirstLongFormFinalize bool
	longFormFinalizeCalls          int
}

func (executor *stdioReportPlanExecutor) Run(ctx context.Context, req AgentRequest) (AgentResult, error) {
	result, err := executor.delegate.Run(ctx, req)
	if err != nil {
		return result, err
	}
	if req.LongFormFinalize != nil {
		executor.longFormFinalizeCalls++
		frame, parseErr := parseFixtureSectionalFrame(result.Text)
		if parseErr != nil {
			return result, parseErr
		}
		encoded, _ := json.Marshal(req.LongFormFinalize)
		args := []string{"mcp", "-db", executor.database, "-mission-id", req.MissionID, "-agent-session-id", req.ToolSessionID, "-agent-executor", req.AgentExecutor, "-enabled-tool", plasmamcp.ToolReportLongFormFinalize, "-report-long-form-finalize-binding-json", string(encoded)}
		arguments := map[string]any{
			"mission_id": req.MissionID, "session_id": req.ToolSessionID, "pending_event_id": req.LongFormFinalize.PendingEventID, "plan_event_id": req.LongFormFinalize.PlanEventID,
			"idempotency_key": req.LongFormFinalize.IdempotencyKey, "producer": map[string]any{"type": "agent_session", "id": req.ToolSessionID}, "opening_markdown": frame.FrontMatter, "closing_markdown": frame.Closing,
		}
		if executor.malformedFirstLongFormFinalize && executor.longFormFinalizeCalls == 1 {
			arguments["unexpected_field"] = true
		}
		call := map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": map[string]any{"name": plasmamcp.ToolReportLongFormFinalize, "arguments": arguments}}
		callJSON, _ := json.Marshal(call)
		command := exec.CommandContext(ctx, executor.binary, args...)
		command.Stdin = bytes.NewReader(append(callJSON, '\n'))
		output, runErr := command.CombinedOutput()
		if runErr != nil || !strings.Contains(string(output), "artifact_sha256") || strings.Contains(string(output), `"isError":true`) {
			return result, fmt.Errorf("plasma final MCP stdio failed: %v: %s", runErr, output)
		}
		result.Text = "REPORT_FINALIZED"
		return result, nil
	}
	if req.PartAssembly != nil {
		assembly, parseErr := parseAgentPartAssembly(result.Text)
		if parseErr != nil {
			return result, parseErr
		}
		encoded, _ := json.Marshal(req.PartAssembly)
		args := []string{
			"mcp", "-db", executor.database, "-mission-id", req.MissionID, "-agent-session-id", req.ToolSessionID, "-agent-executor", req.AgentExecutor,
			"-enabled-tool", plasmamcp.ToolReportPartAssemblyStart,
			"-enabled-tool", plasmamcp.ToolReportPartAssemblyRead,
			"-enabled-tool", plasmamcp.ToolReportPartAssemblyPatch,
			"-enabled-tool", plasmamcp.ToolReportPartAssemblySubmit,
			"-report-part-assembly-binding-json", string(encoded),
		}
		draftID := "rpa_stdio"
		base := map[string]any{"mission_id": req.MissionID, "session_id": req.ToolSessionID, "producer": map[string]any{"type": "agent_session", "id": req.ToolSessionID}}
		partArgs := func(extra map[string]any) map[string]any {
			arguments := make(map[string]any, len(base)+len(extra))
			for key, value := range base {
				arguments[key] = value
			}
			for key, value := range extra {
				arguments[key] = value
			}
			return arguments
		}
		calls := []map[string]any{
			{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": plasmamcp.ToolReportPartAssemblyStart, "arguments": partArgs(map[string]any{"pending_event_id": req.PartAssembly.PendingEventID, "plan_event_id": req.PartAssembly.PlanEventID, "draft_id": draftID, "part_index": req.PartAssembly.PartIndex, "section_count": req.PartAssembly.SectionCount, "idempotency_key": "part_start_key"})}},
		}
		if strings.TrimSpace(assembly.Intro) != "" {
			calls = append(calls, map[string]any{"jsonrpc": "2.0", "id": len(calls) + 1, "method": "tools/call", "params": map[string]any{"name": plasmamcp.ToolReportPartAssemblyPatch, "arguments": partArgs(map[string]any{"draft_id": draftID, "field": "intro", "markdown": assembly.Intro, "summary": "intro", "idempotency_key": "part_intro_key"})}})
		}
		for index, transition := range assembly.Transitions {
			calls = append(calls, map[string]any{"jsonrpc": "2.0", "id": len(calls) + 1, "method": "tools/call", "params": map[string]any{"name": plasmamcp.ToolReportPartAssemblyPatch, "arguments": partArgs(map[string]any{"draft_id": draftID, "field": "transition", "after_section_index": transition.AfterSectionIndex, "markdown": transition.Markdown, "summary": "transition", "idempotency_key": fmt.Sprintf("part_transition_%d_key", index+1)})}})
		}
		if strings.TrimSpace(assembly.Closing) != "" {
			calls = append(calls, map[string]any{"jsonrpc": "2.0", "id": len(calls) + 1, "method": "tools/call", "params": map[string]any{"name": plasmamcp.ToolReportPartAssemblyPatch, "arguments": partArgs(map[string]any{"draft_id": draftID, "field": "closing", "markdown": assembly.Closing, "summary": "closing", "idempotency_key": "part_closing_key"})}})
		}
		calls = append(calls, map[string]any{"jsonrpc": "2.0", "id": len(calls) + 1, "method": "tools/call", "params": map[string]any{"name": plasmamcp.ToolReportPartAssemblySubmit, "arguments": partArgs(map[string]any{"pending_event_id": req.PartAssembly.PendingEventID, "plan_event_id": req.PartAssembly.PlanEventID, "draft_id": draftID, "idempotency_key": "part_submit_key"})}})
		var input bytes.Buffer
		for _, call := range calls {
			if err := json.NewEncoder(&input).Encode(call); err != nil {
				return result, err
			}
		}
		command := exec.CommandContext(ctx, executor.binary, args...)
		command.Stdin = &input
		output, runErr := command.CombinedOutput()
		if runErr != nil || !strings.Contains(string(output), "event_id") || strings.Contains(string(output), `"isError":true`) {
			return result, fmt.Errorf("plasma part assembly MCP stdio failed: %v: %s", runErr, output)
		}
		result.Text = reporting.PartAssemblySubmittedSentinel
		return result, nil
	}
	if req.ReportPlan == nil {
		return result, nil
	}
	var plan json.RawMessage
	if json.Unmarshal([]byte(result.Text), &plan) != nil {
		return result, fmt.Errorf("fixture plan is not JSON")
	}
	args := []string{"mcp", "-db", executor.database, "-mission-id", req.MissionID, "-agent-session-id", req.ToolSessionID, "-agent-executor", req.AgentExecutor, "-enabled-tool", plasmamcp.ToolMissionGet, "-enabled-tool", plasmamcp.ToolReportPlanSubmit}
	args = appendReportPlanMCPArgs(args, req.ToolSessionID, *req.ReportPlan)
	call := map[string]any{
		"jsonrpc": "2.0", "id": 2, "method": "tools/call",
		"params": map[string]any{"name": plasmamcp.ToolReportPlanSubmit, "arguments": map[string]any{
			"mission_id": req.MissionID, "session_id": req.ToolSessionID, "pending_event_id": req.ReportPlan.PendingEventID,
			"report_mode": req.ReportPlan.ReportMode, "idempotency_key": req.ReportPlan.IdempotencyKey,
			"producer": map[string]any{"type": "agent_session", "id": req.ToolSessionID}, "plan": plan,
		}},
	}
	callJSON, _ := json.Marshal(call)
	input := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}` + "\n" + string(callJSON) + "\n")
	command := exec.CommandContext(ctx, executor.binary, args...)
	command.Stdin = bytes.NewReader(input)
	output, runErr := command.CombinedOutput()
	if runErr != nil {
		return result, fmt.Errorf("plasma mcp stdio failed: %w: %s", runErr, output)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], plasmamcp.ToolReportPlanSubmit) || !strings.Contains(lines[1], "submission_event_id") || strings.Contains(lines[1], `"isError":true`) {
		return result, fmt.Errorf("unexpected plasma mcp stdio output: %s", output)
	}
	result.Text = reporting.ReportPlanSubmittedSentinel
	return result, nil
}

func TestWebReportRoutesUseRealPlasmaMCPStdio(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	binary := filepath.Join(t.TempDir(), "plasma")
	build := exec.Command("go", "build", "-o", binary, "./cmd/plasma")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build plasma acceptance binary: %v: %s", err, output)
	}

	t.Run("partial binding fails closed", func(t *testing.T) {
		command := exec.Command(binary, "mcp", "-db", filepath.Join(t.TempDir(), "partial.db"), "-mission-id", "mis_partial", "-agent-session-id", "ses_partial", "-agent-executor", "codex", "-report-plan-mode", "planned", "-report-plan-tool-session-id", "ses_partial", "-report-plan-idempotency-key", "key")
		if output, err := command.CombinedOutput(); err == nil || !strings.Contains(string(output), "binding is incomplete") {
			t.Fatalf("missing pending binding did not fail at startup: err=%v output=%s", err, output)
		}
	})

	for _, test := range []struct {
		name, mode, executorName, providerSession string
		responses                                 []AgentResult
	}{
		{name: "planned codex", mode: reportModePlanned, executorName: "codex", providerSession: "provider-planned", responses: []AgentResult{
			{Text: agentReportAnyJSON(agentReportPlan{Summary: "Plan", Sections: []agentReportSection{{Title: "Section"}}}), SessionID: "provider-planned"},
			{Text: "# Planned report\n\nBody.", SessionID: "provider-planned"},
		}},
		{name: "long form codex", mode: reportModeLongForm, executorName: "codex", providerSession: "provider-long", responses: []AgentResult{
			{Text: agentReportAnyJSON(agentSectionalReportPlan{Summary: "Plan", Parts: []agentReportPart{{Title: "Part", Sections: []agentReportSection{{Title: "Section"}}}}}), SessionID: "provider-long"},
			{Text: "Section body.", SessionID: "provider-long"},
			{Text: `{"intro":"Part intro","transitions":[],"closing":"Part close"}`, SessionID: "provider-long"},
			{Text: `{"front_matter":"# Long report","closing":"## Close"}`, SessionID: "provider-long"},
			{Text: `{"front_matter":"# Long report","closing":"## Close"}`, SessionID: "provider-long"},
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			database := filepath.Join(t.TempDir(), "plasma.db")
			store, err := sqlite.Open(ctx, database)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			service := app.NewService(store)
			delegate := &fakeAgentExecutor{responses: test.responses}
			executor := &stdioReportPlanExecutor{binary: binary, database: database, delegate: delegate, malformedFirstLongFormFinalize: test.mode == reportModeLongForm}
			server := httptest.NewServer(NewServer(service, Options{AgentExecutors: map[string]AgentExecutor{test.executorName: executor}}))
			defer server.Close()
			mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "MCP acceptance"})
			missionID := nestedString(t, mission, "projection", "mission_id")
			postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", map[string]any{
				"title": "Report", "report_mode": test.mode, "agent_executor": test.executorName,
				"generation_guidance_profile": reportGenerationGuidanceProfileVisualPlan,
			})
			detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
			if countEvents(detail, "report.plan.submitted") != 1 || countEvents(detail, "report.plan.created") != 1 {
				t.Fatalf("stdio path did not create one submission and canonical: %#v", detail["events"])
			}
			if test.mode == reportModeLongForm && (executor.longFormFinalizeCalls != 2 || countEvents(detail, "report.draft.failed") != 0 || countEvents(detail, "turn.agent.response") != 0) {
				t.Fatalf("built stdio validation retry did not recover cleanly: calls=%d events=%#v", executor.longFormFinalizeCalls, detail["events"])
			}
			events, _ := detail["events"].([]any)
			submittedIndex, canonicalIndex := -1, -1
			for index, raw := range events {
				event, _ := raw.(map[string]any)
				switch event["EventType"] {
				case "report.plan.submitted":
					submittedIndex = index
					producer := nestedMap(t, event, "Producer")
					if producer["type"] != "mcp_server" || producer["id"] != plasmamcp.ToolReportPlanSubmit {
						t.Fatalf("submission producer is not server-owned: %#v", producer)
					}
					payload := nestedMap(t, event, "Payload")
					if _, exists := payload["provider_session_id"]; exists {
						t.Fatalf("submission falsely recorded returned provider session: %#v", payload)
					}
				case "report.plan.created":
					canonicalIndex = index
					payload := nestedMap(t, event, "Payload")
					if payload["agent_session_id"] != test.providerSession || payload["report_plan_session_id"] != test.providerSession {
						t.Fatalf("canonical did not record validated provider session: %#v", payload)
					}
				}
			}
			if submittedIndex < 0 || canonicalIndex <= submittedIndex {
				t.Fatalf("canonical ordering is invalid: submitted=%d canonical=%d", submittedIndex, canonicalIndex)
			}
			if len(delegate.requests) == 0 || delegate.requests[0].ReportPlan == nil {
				t.Fatal("Web did not produce a report-plan MCP context")
			}
			planRequest := delegate.requests[0]
			for _, expected := range []string{missionID, planRequest.ToolSessionID, planRequest.ReportPlan.PendingEventID, test.mode, planRequest.ReportPlan.IdempotencyKey, `producer {"type":"agent_session","id":"` + planRequest.ToolSessionID + `"}`} {
				if expected == "" || !strings.Contains(planRequest.Prompt, expected) {
					t.Fatalf("real Web plan prompt missing binding %q: %s", expected, planRequest.Prompt)
				}
			}
		})
	}
}
