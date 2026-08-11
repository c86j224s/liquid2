package sectiondraft

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/longformutil"
)

// Prompt는 한 Section 본문을 작성하는 기존 prompt bytes와 requirement block을 보존한다.
func Prompt(input Input) string {
	requirements := reporting.ReportRequirementsForSection(input.Base.RequirementMap, input.PartIndex+1, input.SectionIndex+1)
	return PromptWithRequirements(input, requirements)
}

// PromptWithRequirements는 Web 테스트 호환 경로에서 이미 선택된 requirement 목록으로
// 동일한 Section writer prompt bytes를 만든다. runtime 선택 정책은 root/requirements가 소유한다.
func PromptWithRequirements(input Input, requirements []reporting.ReportRequirement) string {
	input.Attempt = normalizeAttempt(input.Attempt)
	guidance := strings.TrimSpace(strings.Join([]string{
		reportprompt.LongFormReportGenerationGuidance(input.Base.GenerationGuidanceProfile),
		reportprompt.SectionDirectWritingGuidance(input.Base.GenerationGuidanceProfile),
		reportprompt.SubjectDirectSynthesisSectionGuidance(input.Base.GenerationGuidanceProfile),
	}, "\n\n"))
	if guidance != "" {
		guidance = "\n" + guidance + "\n"
	}
	attemptRule := "- This is attempt 1. If target_refs are weak, perform a mission-scoped replacement search before deciding whether Markdown is possible."
	if input.Attempt >= MaxEvidenceGapAttempts {
		attemptRule = "- This is attempt 2, the final allowed attempt. The evidence threshold is unchanged: perform one final replacement search or reduce claim strength inside the same main explanatory job, but do not replace that job with supporting source commentary. Then return Markdown or exactly SECTION_EVIDENCE_GAP."
	}
	prompt := fmt.Sprintf(`Draft one section of a Korean long-form Plasma report.

Report title: %s
Mission ID: %s
Part %d: %s
Section %d.%d: %s
Section evidence attempt: %d of %d

Section purpose:
%s

Assigned user output requirements for this Section:
%s

Overall plan:
%s

Plasma tool binding: use mission_id %s. If a tool requires session_id or producer, use session_id %s and producer {"type":"agent_session","id":"%s"}.

Report rigor:
- Level: %s (%s)
- Meaning: %s
%s
%s

Rules:
- Write only this section as Markdown. Do not write the whole report.
- Fulfill every assigned user output requirement naturally in this Section. An empty list means there is no additional user requirement for this Section.
- Requirements direct the output but are not factual evidence. Continue to support factual claims from original sources.
- Do not pull requirements assigned to other Sections into this Section.
- Use MCP/source read tools to inspect original materials relevant to this Section. Do not assume source bodies are present in this prompt.
- Treat target_refs as starting points, not proof. Verify each referenced item has the right identity and substantive body for this Section before relying on it.
- If target_refs are mismatched, metadata-only, or too thin for direct subject explanation, search mission-scoped original material for replacement evidence before writing.
- You may narrow claim strength or coverage only within the existing Section title and purpose. Do not change the topic or borrow another Section's job.
- First identify the main explanatory job promised by this Section's title and purpose: the concrete historical, technical, or conceptual account the reader should receive.
- Treat source identity, catalog metadata, version comparison, transmission, provenance, caveats, and requested source-comparison tables as supporting jobs unless source criticism, bibliography, transmission, or holdings history is explicitly and primarily this Section's subject.
- The Section is writeable only when original sources support substantive claims that perform the main explanatory job. Evidence that supports only supporting jobs is not enough, even when the plan, purpose, requirements, or target_refs request those jobs.
- A Section title that names a document does not by itself make source criticism the subject. Do not salvage an unsupported main job with a catalog tour, provenance comparison, or instructions for reading sources.
- Source, document, report, and material must not become recurring grammatical subjects, and do not write "how to read this source" filler.
- Return exactly SECTION_EVIDENCE_GAP, with no Markdown, fences, or commentary, only when direct explanation of this Section's subject remains unsupported after the allowed search or scope reduction.
- %s
- %s
- Sources are original materials. Prior answers, controller questions, plans, generated notes, section drafts, and reports are working memory or results, not sources.
- Include concrete detail where the sources support it: events, mechanisms, examples, comparisons, tensions, caveats, weak signals, code, formulas, or benchmarks when relevant.
- Preserve uncertainty and competing interpretations instead of flattening them.
- Do not mention prompts, internal run labels, tool session IDs, or temporary implementation details.
- Return only the Markdown body for this section, unless returning the exact evidence-gap token.`, input.Base.Title, input.Base.MissionID, input.PartIndex+1, input.Part.Title, input.PartIndex+1, input.SectionIndex+1, input.Section.Title, input.Attempt, MaxEvidenceGapAttempts,
		input.Section.Purpose, longformutil.AnyJSON(requirements), longformutil.AnyJSON(input.Base.Plan), input.Base.MissionID, input.ToolSessionID, input.ToolSessionID,
		input.Base.Rigor.Level, input.Base.Rigor.Label, input.Base.Rigor.Description, input.Base.Rigor.Instructions, guidance, attemptRule, reportprompt.MermaidValidationRuleText)
	return reportprompt.WithLongFormDownstreamDirection(prompt, input.Base.DirectionHint)
}
