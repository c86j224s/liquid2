package reportpatch

import (
	"reflect"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
)

func TestMCPToolsOrder(t *testing.T) {
	expected := []string{
		mcptools.ToolReportPatchStart,
		mcptools.ToolReportPatchRead,
		mcptools.ToolReportPatchApply,
		mcptools.ToolReportPatchFinalize,
	}
	if !reflect.DeepEqual(MCPTools(), expected) {
		t.Fatalf("MCPTools() = %#v, want %#v", MCPTools(), expected)
	}
}

func TestPromptBytes(t *testing.T) {
	req := reportexecution.PatchRequest{
		AgentExecutor:                "codex",
		AgentModel:                   "gpt-test",
		AgentReasoningEffort:         "medium",
		MCPMode:                      "auto",
		ReportSessionID:              "ses_report",
		PreviousAgentSessionID:       "ses_prev",
		ForkSourceAgentSessionID:     "ses_source",
		ReportSessionPolicy:          reportexecution.SessionPolicyIsolatedFork,
		ReportSessionPolicySelection: reportexecution.SessionPolicySelectionAutoIsolatedFork,
		SessionChainKind:             "isolated_fork_report_patch",
	}
	got := Prompt("Test Report", "mis_1", "ses_tool", "evt_pending", "art_base", "Change \"summary\"", req)
	want := `You are patching an existing Plasma Markdown report artifact.

Do not rewrite the full report in your response. Use the report patch MCP tools to read and modify the report in bounded chunks, then finalize the patched report into a new artifact version.

Mission ID: mis_1
Base report artifact ID: art_base
Patched report title: "Test Report"
Patch instruction: "Change \"summary\""

Plasma tool binding:
- Use mission_id mis_1.
- Use session_id ses_tool and producer {"type":"agent_session","id":"ses_tool"} for all report patch tool calls.

Required MCP flow:
1. Call plasma.report.patch.start with base_artifact_id art_base, title "Test Report", and the patch instruction. Do not provide patch_id; use the patch_id returned by this call for later patch tool calls.
2. Use plasma.report.patch.read to inspect the relevant report ranges. Read more chunks when needed; do not assume the whole report is in the prompt.
3. Use plasma.report.patch.apply with small replace, insert_after, or append operations. Prefer exact targeted edits over broad rewrites.
4. Call plasma.report.patch.finalize exactly once after edits are complete.

Finalize metadata is server-bound Plasma lineage. Do not infer it from the report text, previous pending events, or tool responses. When the finalize schema asks for these fields, use these exact values:
- pending_event_id: evt_pending
- agent_executor: codex
- agent_model: gpt-test
- agent_reasoning_effort: medium
- mcp_mode: auto
- agent_session_id: ses_report
- previous_agent_session_id: ses_prev
- returned_agent_session_id: ses_report
- report_session_id: ses_report
- fork_source_agent_session_id: ses_source
- report_session_policy: isolated_fork
- report_session_policy_selection: auto_isolated_fork
- session_chain_kind: isolated_fork_report_patch

Rules:
- Keep the source and citation structure intact unless the user explicitly asked to change it.
- Preserve useful detail. Do not compress the report just because you are editing it.
- This patch session only exposes report patch tools. If the requested change requires source verification that cannot be done from the current artifact, stop and explain the blocker briefly instead of guessing.
- If you cannot make the requested change safely, do not finalize a fake artifact; explain the blocker briefly.
- After successful finalization, return only a short Korean summary of what changed and the new artifact ID if the tool returned one.`
	if got != want {
		t.Fatalf("prompt mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
