package reportprompt

import (
	"fmt"
	"strings"
)

// MarkdownReportPlanPrompt는 planned 보고서의 canonical plan 제출 prompt를 만든다.
func MarkdownReportPlanPrompt(title string, missionID string, toolSessionID string, pendingEventID string, idempotencyKey string, rigor RigorProfile, generationGuidanceProfile string) string {
	experimentalGuidance := strings.TrimSpace(ReportGenerationPlanningGuidance(generationGuidanceProfile))
	if experimentalGuidance != "" {
		experimentalGuidance = "\n" + experimentalGuidance + "\n"
	}
	return fmt.Sprintf(`You are planning a Plasma report before writing it.

Create a user-visible Korean report generation plan for the current mission.
Do not write the article yet. Submit the plan through plasma.report.plan.submit.
Use Plasma MCP research tools to inspect the mission before planning. Source bodies, evidence arrays, and mission recall JSON are not pasted into this prompt. PDF reads return extracted text and metadata, not raw PDF bytes.
Live local_path sources are mutable origins. Use read tools to create source.observed events before relying on them, and plan to cite observation metadata rather than only source IDs.

Evidence rigor:
- Level: %s (%s)
- Meaning: %s
%s

Planning workflow:
- First call plasma.research.outline for the mission overview.
- Use plasma.research.list and plasma.research.grep to find relevant source snapshots, evidence records, saved claims, prior report blocks, prior user turns, agent responses, controller questions, and unresolved questions.
- Use plasma.research.read for the objects or source chunks you intend to rely on. If a read is truncated, continue with next_offset when that material matters.
- For PDF sources, rely on Plasma's extracted text reads and extraction metadata. Do not ask for raw PDF bytes in the prompt.
- For live_reference local_path sources, final report support should come from explicit read observations with observation_event_id, observed_at, relative_path, sha256, and git metadata when available.
- Use plasma.research.references when you need to understand source-evidence-claim-report links.
- General research may inspect proposed, pending, or rejected material as context, but the plan's target_refs should name only approved records you expect the final report to rely on.
- Treat repeated or explicit user questions as coverage signals. If the user steered the mission toward a person, event, comparison, dispute, or source cluster, include it in sections or planned_omissions after checking source support.
- Plan for richness. Include facts, interpretations, reactions, rumors, disputes, code, formulas, benchmarks, and open questions when the mission and rigor level allow them.
- The plan is visible to the user. Be concrete enough that the user can tell what the report will cover and what evidence clusters it will use.
%s

Report title requested by the user interface:
%s

Plasma tool binding: use mission_id %s, session_id %s, pending_event_id %s, report_mode planned, idempotency_key %s, and producer {"type":"agent_session","id":"%s"}.

Submit one accepted plan with this plan shape. If the tool returns a retryable validation error, correct the plan and resubmit; make at most three parsed submission calls total. Every parsed call consumes this budget, including a success or replay; protocol/envelope parse failures do not:
{
  "summary": "what this report will try to produce",
  "sections": [
    {
      "title": "planned section title",
      "purpose": "why this section belongs in the report",
      "target_refs": {"claim_ids": ["clm_..."], "evidence_ids": ["evd_..."], "snapshot_ids": ["src_..."]}
    }
  ],
  "coverage_notes": ["what source or evidence clusters were inspected and will be used"],
  "planned_omissions": ["known gaps, weak areas, or items intentionally left out"]
}

After the tool succeeds, return exactly PLAN_SUBMITTED as the complete final response. Do not return plan JSON, fences, or commentary.
`, rigor.Level, rigor.Label, rigor.Description, rigor.Instructions, experimentalGuidance, strings.TrimSpace(title), strings.TrimSpace(missionID), toolSessionID, pendingEventID, idempotencyKey, toolSessionID)
}
