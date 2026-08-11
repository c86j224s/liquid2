package plan

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
)

// LongFormPrompt는 section-first long-form plan 제출 prompt의 기존 bytes를 보존한다.
func LongFormPrompt(title string, missionID string, toolSessionID string, pendingEventID string, idempotencyKey string, rigor reportprompt.RigorProfile, generationGuidanceProfile string) string {
	experimentalGuidance := strings.TrimSpace(reportprompt.LongFormExperimentalPlanningGuidance(generationGuidanceProfile))
	if experimentalGuidance != "" {
		experimentalGuidance = "\n" + experimentalGuidance + "\n"
	}
	return fmt.Sprintf(`You are planning a section-first Korean long-form Plasma report.

Do not write the report yet. Submit the plan through plasma.report.plan.submit.
Use Plasma MCP research tools to inspect the mission before planning. Source bodies, evidence arrays, and mission recall JSON are not pasted into this prompt.

Mission ID: %s
Report title: %s
Plasma tool binding: use mission_id %s, session_id %s, pending_event_id %s, report_mode long_form, idempotency_key %s, and producer {"type":"agent_session","id":"%s"}.

Report rigor:
- Level: %s (%s)
- Meaning: %s
%s

Planning rules:
- First call plasma.research.outline for the mission overview.
- Use plasma.research.list, plasma.research.grep, plasma.research.read, and plasma.research.references to find the source-backed clusters the report should cover.
- Do not read or incorporate turn.user or report.draft.pending events for additional planning material. When a request_direction block is present, it is the only request-scoped direction the planner may use before outline freeze.
- Treat pending_event_id only as a submission binding; do not read it as a ledger event.
- Plan for long-form richness, not a short summary. Include concrete episodes, mechanisms, comparisons, tensions, caveats, weak signals, code/formulas/benchmarks when relevant.
- Group the report into Parts and Sections. A normal mission should usually have 2-5 Parts and 6-14 Sections total. Use fewer only when the mission material is genuinely small.
- Each Section must be specific enough to be drafted independently.
- Sources are original materials. Prior answers, controller questions, plans, generated notes, section drafts, and reports are working memory or results, not sources.
- target_refs should name the source snapshots, evidence records, or saved claims the Section should inspect when available.
%s

Submit one accepted plan with this plan shape. If the tool returns a retryable validation error, correct the plan and resubmit; make at most three parsed submission calls total. Every parsed call consumes this budget, including a success or replay; protocol/envelope parse failures do not:
{
  "summary": "what this long-form report will produce",
  "parts": [
    {
      "title": "part title",
      "purpose": "why this part belongs",
      "sections": [
        {
          "title": "section title",
          "purpose": "what this section must explain",
          "target_refs": {"claim_ids": ["clm_..."], "evidence_ids": ["evd_..."], "snapshot_ids": ["src_..."]}
        }
      ]
    }
  ],
  "coverage_notes": ["source-backed clusters inspected"],
  "planned_omissions": ["known gaps or intentionally omitted areas"]
}

After the tool succeeds, return exactly PLAN_SUBMITTED as the complete final response. Do not return plan JSON, fences, or commentary.`, missionID, title, missionID, toolSessionID, pendingEventID, idempotencyKey, toolSessionID, rigor.Level, rigor.Label, rigor.Description, rigor.Instructions, experimentalGuidance)
}
