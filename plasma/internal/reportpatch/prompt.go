package reportpatch

import (
	"fmt"
	"strconv"

	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
)

// Prompt builds the ordinary report patch agent instruction. The wording is a
// transport-neutral contract shared by Web and CLI patch callers.
func Prompt(title string, missionID string, toolSessionID string, pendingEventID string, baseArtifactID string, instruction string, req reportexecution.PatchRequest) string {
	return fmt.Sprintf(`You are patching an existing Plasma Markdown report artifact.

Do not rewrite the full report in your response. Use the report patch MCP tools to read and modify the report in bounded chunks, then finalize the patched report into a new artifact version.

Mission ID: %s
Base report artifact ID: %s
Patched report title: %s
Patch instruction: %s

Plasma tool binding:
- Use mission_id %s.
- Use session_id %s and producer {"type":"agent_session","id":"%s"} for all report patch tool calls.

Required MCP flow:
1. Call %s with base_artifact_id %s, title %s, and the patch instruction. Do not provide patch_id; use the patch_id returned by this call for later patch tool calls.
2. Use %s to inspect the relevant report ranges. Read more chunks when needed; do not assume the whole report is in the prompt.
3. Use %s with small replace, insert_after, or append operations. Prefer exact targeted edits over broad rewrites.
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
- Keep the source and citation structure intact unless the user explicitly asked to change it.
- Preserve useful detail. Do not compress the report just because you are editing it.
- This patch session only exposes report patch tools. If the requested change requires source verification that cannot be done from the current artifact, stop and explain the blocker briefly instead of guessing.
- If you cannot make the requested change safely, do not finalize a fake artifact; explain the blocker briefly.
- After successful finalization, return only a short Korean summary of what changed and the new artifact ID if the tool returned one.`, missionID, baseArtifactID, strconv.Quote(title), strconv.Quote(instruction), missionID, toolSessionID, toolSessionID,
		mcptools.ToolReportPatchStart, baseArtifactID, strconv.Quote(title),
		mcptools.ToolReportPatchRead,
		mcptools.ToolReportPatchApply,
		mcptools.ToolReportPatchFinalize,
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
