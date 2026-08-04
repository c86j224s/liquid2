package reporthumanize

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportpatch"
)

func TestPromptGoldenBytes(t *testing.T) {
	req := reportexecution.PatchRequest{
		Title:                        "Report humanized",
		Instruction:                  PatchInstruction(),
		AgentExecutor:                "codex",
		AgentModel:                   "gpt-test",
		AgentReasoningEffort:         "medium",
		MCPMode:                      "auto",
		ReportSessionID:              "ses_report",
		PreviousAgentSessionID:       "ses_report",
		ReportSessionPolicy:          reportexecution.SessionPolicySameSession,
		ReportSessionPolicySelection: "auto_same_report_session_h5",
		SessionChainKind:             "same_report_session_h5_humanize_patch",
	}
	got := Prompt("Report", "mis_1", "ses_tool", "evt_h5", "art_source", req)
	want := `You are applying the approved Plasma H5 Korean report humanize pass through MCP report patch tools.

This is a post-report tone pass after Markdown report generation. It is not a planner, source selector, content model rewrite, AST redesign, or Designed HTML improvement.

Do not rewrite the full report in your response. Use the report patch MCP tools to inspect bounded chunks, apply small targeted edits, and finalize a new Markdown artifact.

Mission ID: mis_1
Base report artifact ID: art_source
Report title: "Report"
Patch instruction: "Apply the H5 Korean tone pass to this Markdown report. Smooth stiff AI-like Korean phrasing, repetitive transitions, and unnatural sentence endings, but preserve the report structure, claims, citations, numbers, tables, code, links, headings, paragraph boundaries, and useful detail. Use bounded MCP reads and small targeted patch operations. Do not rewrite or summarize the whole report."

Plasma tool binding:
- Use mission_id mis_1.
- Use session_id ses_tool and producer {"type":"agent_session","id":"ses_tool"} for all report patch tool calls.

Required MCP flow:
1. Call plasma.report.patch.start with base_artifact_id art_source, title "Report humanized", and the patch instruction. Do not provide patch_id; use the patch_id returned by this call for later patch tool calls.
2. Use plasma.report.patch.read to inspect relevant ranges. Continue with next_offset when a chunk is truncated and more content is needed.
3. Use plasma.report.patch.apply with small replace operations only. Do not use append, insert_after, or replace_all. Prefer exact targeted edits over broad rewrites.
4. Call plasma.report.patch.finalize exactly once after edits are complete.

Finalize metadata is server-bound Plasma lineage. Do not infer it from the report text, previous pending events, or tool responses. When the finalize schema asks for these fields, use these exact values:
- pending_event_id: evt_h5
- agent_executor: codex
- agent_model: gpt-test
- agent_reasoning_effort: medium
- mcp_mode: auto
- agent_session_id: ses_report
- previous_agent_session_id: ses_report
- returned_agent_session_id: ses_report
- report_session_id: ses_report
- fork_source_agent_session_id:
- report_session_policy: same_session
- report_session_policy_selection: auto_same_report_session_h5
- session_chain_kind: same_report_session_h5_humanize_patch

Rules:
- Keep the same report register: clear, public-facing Korean report prose, not casual chat.
- Smooth stiff Korean phrasing and transitions where possible.
- Do not add, remove, merge, split, reorder, or summarize sections or paragraphs.
- Preserve heading levels and order exactly.
- Preserve tables, code fences, links, footnotes, source labels, citations, quotes, numbers, dates, model names, technical terms, and uncertainty/caveat wording.
- Do not introduce new claims, sources, evidence, recommendations, or caveats.
- If a sentence cannot be improved without risking fidelity, keep it unchanged.
- If the report is already natural enough or every possible change would risk fidelity, do not finalize. Return exactly: NO_H5_CHANGES
- After successful finalization, return only a short Korean summary and the artifact ID returned by the tool.
`
	want = strings.Replace(want, "- fork_source_agent_session_id:\n", "- fork_source_agent_session_id: \n", 1)
	if got != want {
		t.Fatalf("prompt mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
	sum := sha256.Sum256([]byte(got))
	if hash := hex.EncodeToString(sum[:]); hash != "846f650cc9ef40eab56fe19a3170944070c6b63881568451e644f8f47a9496fd" {
		t.Fatalf("prompt sha256 = %s", hash)
	}
	for _, tool := range reportpatch.MCPTools() {
		if !strings.Contains(got, tool) {
			t.Fatalf("prompt missing tool %s", tool)
		}
	}
}
