package web

import (
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportpatch"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"strings"
)

func agentOneTakeMarkdownReportPrompt(title string, missionID string, toolSessionID string, rigor reportRigorProfile, generationGuidanceProfile string) string {
	return reportprompt.OneTakeMarkdownReportPrompt(title, missionID, toolSessionID, reportWorkflowRigor(rigor), generationGuidanceProfile)
}

func agentMarkdownReportPrompt(title string, missionID string, toolSessionID string, rigor reportRigorProfile, plan agentReportPlan, generationGuidanceProfile string) string {
	return reportprompt.PlannedMarkdownReportPrompt(title, missionID, toolSessionID, reportWorkflowRigor(rigor), plan, generationGuidanceProfile)
}

func agentReportPatchPrompt(title string, missionID string, toolSessionID string, pendingEventID string, baseArtifactID string, instruction string, req reportexecution.PatchRequest) string {
	return AgentReportPatchPrompt(title, missionID, toolSessionID, pendingEventID, baseArtifactID, instruction, req)
}

// AgentReportPatchPrompt는 patch agent에게 전달할 report 수정 지시문을 조립한다.
func AgentReportPatchPrompt(title string, missionID string, toolSessionID string, pendingEventID string, baseArtifactID string, instruction string, req reportexecution.PatchRequest) string {
	return reportpatch.Prompt(title, missionID, toolSessionID, pendingEventID, baseArtifactID, instruction, req)
}

func agentReportPlanPrompt(title string, missionID string, toolSessionID string, pendingEventID string, idempotencyKey string, rigor reportRigorProfile, generationGuidanceProfile string) string {
	return reportprompt.MarkdownReportPlanPrompt(title, missionID, toolSessionID, pendingEventID, idempotencyKey, reportWorkflowRigor(rigor), generationGuidanceProfile)
}

func agentReportPrompt(title string, missionID string, toolSessionID string, rigor reportRigorProfile, plan agentReportPlan) string {
	planJSON := agentReportPlanJSON(plan)
	return fmt.Sprintf(`You are the Plasma report writer.

Write a polished Korean report or article for the current mission.
The canonical output must be a structured AST JSON object. Markdown and HTML will be rendered from this AST later.
Do not output Markdown fences, commentary, or prose outside the JSON object.
Do not invent source references. Use Plasma MCP research tools to inspect pinned sources, live local_path observations, evidence, saved claims, questions, and report blocks when needed. Source bodies, evidence arrays, and mission recall JSON are not pasted into this prompt.

Evidence rigor:
- Level: %s (%s)
- Meaning: %s
%s

General evidence handling:
- First call plasma.research.outline for the mission overview.
- Use plasma.research.list and plasma.research.grep to find candidate source snapshots, evidence records, claims, questions, and report blocks.
- Use plasma.research.read to confirm saved knowledge, evidence details, source chunks, and long payloads with offset/max_bytes.
- For PDF sources, rely on extracted text and extraction metadata returned by Plasma tools.
- For live_reference local_path sources, use read observations rather than source IDs alone. When a sentence depends on mutable local material, include the relevant human locator and observation_event_id/observed_at/sha256/git details in the text or refs context available to the renderer.
- Use plasma.research.references to verify source-evidence-claim-report links before relying on them.
- Treat grep matches as candidates only. A final report sentence that depends on mission material must be grounded in saved evidence, saved claims, or explicit source reads.
- Evidence can include facts, observations, interpretations, reactions, rumors, controversies, market signals, code, formulas, benchmarks, and open questions.
- Treat evidence_type and confidence as writing constraints, not as obstacles. The report should become richer without flattening weak signals into facts.
- If a sentence depends on a specific saved claim, evidence record, or source snapshot, include the relevant refs in that AST block.
- References are rendered as visible footnotes in Markdown and HTML exports. Include refs for every source-backed paragraph, list, and quote.
- You may inspect proposed, pending, or rejected material while researching, but final AST refs must only contain approved claim_ids and approved evidence_ids that are inside the report scope.
- If unapproved material is useful background, either replace it with approved refs that support the same point or describe it clearly as an unapproved candidate without using its claim_id or evidence_id as a final ref.
- Before returning the AST, check every refs/source_refs object. Any proposed, pending, rejected, missing, or out-of-scope claim_id/evidence_id will be rejected and you will need to repair the AST.

User-visible generation plan created in the previous step:
%s

Follow the plan unless your additional reads reveal that a section should be changed. If you change it, keep the final article coherent and evidence-grounded.

Report title requested by the user interface:
%s

Plasma tool binding: use mission_id %s. If a tool requires session_id or producer, use session_id %s and producer {"type":"agent_session","id":"%s"}.

Return exactly this JSON shape:
{
  "title": "short report title",
  "summary": "executive summary paragraph",
  "blocks": [
    {"type": "heading", "level": 2, "text": "section title"},
    {"type": "paragraph", "text": "article paragraph", "refs": {"claim_ids": ["clm_..."], "evidence_ids": ["evd_..."], "snapshot_ids": ["src_..."]}},
    {"type": "bullet_list", "items": ["item"], "refs": {"evidence_ids": ["evd_..."]}},
    {"type": "quote", "text": "short callout"}
  ]
}

Allowed block types are heading, paragraph, bullet_list, and quote.
Use refs only when a block depends on specific saved knowledge or evidence. Omit refs for narrative transitions.
Write a complete, readable article that covers the planned evidence clusters. Synthesize the material, but do not shrink away planned source-backed substance.
`, rigor.level, rigor.label, rigor.description, rigor.instructions, planJSON, strings.TrimSpace(title), strings.TrimSpace(missionID), toolSessionID, toolSessionID)
}

func parseAgentReportAST(text string) (agentReportAST, error) {
	raw, err := extractAgentJSONObject(text)
	if err != nil {
		return agentReportAST{}, fmt.Errorf("%w: report agent did not return JSON AST", producterror.ErrInvalidInput)
	}
	var ast agentReportAST
	decoder := json.NewDecoder(strings.NewReader(raw))
	if err := decoder.Decode(&ast); err != nil {
		return agentReportAST{}, fmt.Errorf("%w: invalid report AST JSON: %v", producterror.ErrInvalidInput, err)
	}
	if strings.TrimSpace(ast.Title) == "" && strings.TrimSpace(ast.Summary) == "" && len(ast.Blocks) == 0 {
		return agentReportAST{}, fmt.Errorf("%w: report AST is empty", producterror.ErrInvalidInput)
	}
	return ast, nil
}

func extractAgentJSONObject(text string) (string, error) {
	raw := strings.TrimSpace(text)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSpace(strings.TrimSuffix(raw, "```"))
	if strings.HasPrefix(raw, "{") {
		return raw, nil
	}
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return "", fmt.Errorf("%w: JSON object not found", producterror.ErrInvalidInput)
	}
	return raw[start : end+1], nil
}

func agentReportPlanJSON(plan agentReportPlan) string {
	encoded, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func agentReportASTJSON(ast agentReportAST) string {
	encoded, err := json.MarshalIndent(ast, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(encoded)
}
