package reportprompt

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

// RigorProfile은 보고서 prompt가 노출하는 증거 엄격도 계약이다.
//
// 값은 HTTP나 CLI 표시 정책이 아니라 prompt bytes에 들어가는 정규화된 설명이다. 빈 값은
// caller가 이미 정규화하지 않았다는 뜻이므로 이 패키지에서 임의 기본값으로 대체하지 않는다.
type RigorProfile struct {
	Level        string
	Label        string
	Description  string
	Instructions string
}

// OneTakeMarkdownReportPrompt는 one_take 보고서 본문 작성 prompt를 만든다.
func OneTakeMarkdownReportPrompt(title string, missionID string, toolSessionID string, rigor RigorProfile, generationGuidanceProfile string) string {
	guidance := strings.TrimSpace(ReportGenerationGuidance(generationGuidanceProfile))
	if guidance != "" {
		guidance = "\n" + guidance + "\n"
	}
	return fmt.Sprintf(`You are writing a quick Plasma report as a Markdown artifact.

Write a useful Korean Markdown report or article in one pass. This is the fast path: do not create a separate plan artifact first, but still use Plasma MCP research tools when needed.

Mission ID: %s
Report title: %s
Plasma tool binding: use mission_id %s. If a tool requires session_id or producer, use session_id %s and producer {"type":"agent_session","id":"%s"}.

Report rigor:
- Level: %s (%s)
- Meaning: %s
%s
%s

Rules:
- Use MCP/source read tools to inspect original materials when the report needs grounding. Do not assume source bodies are present in this prompt.
- %s
- Sources are original materials. Your report is a result artifact, not a source.
- Use prior investigation answers, normal conversation, and controller questions as working memory only. They may guide themes, gaps, structure, and practical implications, but they are not sources and must not be cited.
- Main conclusions must be grounded in original sources or clearly labeled as interpretation, hypothesis, practical implication, rumor, weak signal, or unresolved uncertainty according to the rigor level.
- Prefer a coherent article over a checklist. Include context, comparison, consequences, and tensions where the available material supports them.
- Do not create evidence, claims, confidence updates, source candidates, report blocks, report plans, or report AST JSON.
- Cite source titles, URLs, and human-readable locators when useful. Do not expose internal evidence, claim, or report block IDs as public citations.
- Do not mention this prompt, prompt variant names, experiment labels, tool session IDs, run identifiers, temporary paths, or working directories. Code/source file paths may be cited only when they are original source locators relevant to the user's topic.
- Return only the Markdown report body.`, missionID, title, missionID, toolSessionID, toolSessionID, rigor.Level, rigor.Label, rigor.Description, rigor.Instructions, guidance, MermaidValidationRuleText)
}

// PlannedMarkdownReportPrompt는 canonical plan을 이어받은 planned 본문 작성 prompt를 만든다.
func PlannedMarkdownReportPrompt(title string, missionID string, toolSessionID string, rigor RigorProfile, plan reporting.ReportPlan, generationGuidanceProfile string) string {
	planJSON := reportPlanJSON(plan)
	guidance := strings.TrimSpace(PlannedReportGenerationGuidance(generationGuidanceProfile))
	if guidance != "" {
		guidance = "\n" + guidance + "\n"
	}
	return fmt.Sprintf(`You are writing a Plasma report as a Markdown artifact.

Write a polished public-facing Korean Markdown report or article, not a thin stitched summary.

Mission ID: %s
Report title: %s
Plasma tool binding: use mission_id %s. If a tool requires session_id or producer, use session_id %s and producer {"type":"agent_session","id":"%s"}.

Report rigor:
- Level: %s (%s)
- Meaning: %s
%s
%s

Rules:
- Use MCP/source read tools to inspect original materials. Do not assume source bodies are present in this prompt.
- Start with plasma.research.outline, then use plasma.research.list, plasma.research.read, plasma.research.grep, and plasma.research.references as needed.
- %s
- Distinguish snapshot_only pinned sources, PDF documents, and live_reference local_path sources. PDF reads return extracted text and metadata, not raw PDF bytes. Live local path reads produce source.observed events; when a report sentence depends on them, cite the human locator plus observation_event_id, observed_at, sha256, and git metadata when available.
- Sources are original materials. Your report is a result artifact, not a source.
- Use prior investigation answers, normal conversation, and controller questions as working memory only. They may guide themes, gaps, structure, and practical implications, but they are not sources and must not be cited.
- The visible generation plan below was created in the previous step. Follow it as the coverage contract for this draft. If additional reads show that a planned topic is unsupported or should be changed, reflect that in the report instead of silently dropping it.
- Before writing, re-read the source-backed clusters needed for the planned sections. Do not rely on the plan text alone as a source.
- Main conclusions must be grounded in original sources or clearly labeled as interpretation, hypothesis, practical implication, rumor, weak signal, or unresolved uncertainty according to the rigor level.
- Make the writing rich where the material supports it. Include context, comparison, consequences, and tensions, but do not invent facts.
- Do not create evidence, claims, confidence updates, source candidates, report blocks, or report AST JSON.
- Cite source titles, URLs, and human-readable locators when useful. Do not expose internal evidence, claim, or report block IDs as public citations.
- Do not mention this prompt, prompt variant names, experiment labels, tool session IDs, run identifiers, temporary paths, or working directories. Code/source file paths may be cited only when they are original source locators relevant to the user's topic.
- Return only the Markdown report body.

	Visible generation plan:
	%s`, missionID, title, missionID, toolSessionID, toolSessionID, rigor.Level, rigor.Label, rigor.Description, rigor.Instructions, guidance, MermaidValidationRuleText, planJSON)
}

func reportPlanJSON(plan reporting.ReportPlan) string {
	encoded, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(encoded)
}
