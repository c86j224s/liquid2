package reporthumanize

import (
	"fmt"
	"strconv"

	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportpatch"
)

// Prompt builds the exact H5 report humanize MCP patch prompt.
func Prompt(title string, missionID string, toolSessionID string, pendingEventID string, baseArtifactID string, req reportexecution.PatchRequest) string {
	tools := reportpatch.MCPTools()
	return fmt.Sprintf(`You are applying the approved Plasma H5 Korean report humanize pass through MCP report patch tools.

This is a post-report tone pass after Markdown report generation. It is not a planner, source selector, content model rewrite, AST redesign, or Designed HTML improvement.

Do not rewrite the full report in your response. Use the report patch MCP tools to inspect bounded chunks, apply small targeted edits, and finalize a new Markdown artifact.

Mission ID: %s
Base report artifact ID: %s
Report title: %s
Patch instruction: %s

Plasma tool binding:
- Use mission_id %s.
- Use session_id %s and producer {"type":"agent_session","id":"%s"} for all report patch tool calls.

Required MCP flow:
1. Call %s with base_artifact_id %s, title %s, and the patch instruction. Do not provide patch_id; use the patch_id returned by this call for later patch tool calls.
2. Use %s to inspect relevant ranges. Continue with next_offset when a chunk is truncated and more content is needed.
3. Use %s with small replace operations only. Do not use append, insert_after, or replace_all. Prefer exact targeted edits over broad rewrites.
4. Call %s exactly once after edits are complete.

Finalize metadata is server-bound Plasma lineage. Do not infer it from the report text, previous pending events, or tool responses. When the finalize schema asks for these fields, use these exact values:
- pending_event_id: %s
- agent_executor: %s
- agent_model: %s
- agent_reasoning_effort: %s
- mcp_mode: %s
- agent_session_id: %s
- previous_agent_session_id: %s
- returned_agent_session_id: %s
- report_session_id: %s
- fork_source_agent_session_id: %s
- report_session_policy: %s
- report_session_policy_selection: %s
- session_chain_kind: %s

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
`, missionID, baseArtifactID, strconv.Quote(title), strconv.Quote(req.Instruction), missionID, toolSessionID, toolSessionID,
		tools[0], baseArtifactID, strconv.Quote(req.Title),
		tools[1],
		tools[2],
		tools[3],
		pendingEventID,
		req.AgentExecutor,
		req.AgentModel,
		req.AgentReasoningEffort,
		req.MCPMode,
		req.ReportSessionID,
		req.PreviousAgentSessionID,
		req.ReportSessionID,
		req.ReportSessionID,
		req.ForkSourceAgentSessionID,
		req.ReportSessionPolicy,
		req.ReportSessionPolicySelection,
		req.SessionChainKind)
}
