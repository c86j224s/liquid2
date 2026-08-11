package requirements

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/longformutil"
)

// Prompt는 고정된 long-form outline에 사용자 출력 요구사항을 배정하는 기존 prompt bytes다.
func Prompt(title, directionHint string, plan reporting.SectionalReportPlan, reviewEventIDs []string, binding reporting.ReportRequirementMapBinding) string {
	return fmt.Sprintf(`You are mapping explicit user output requirements onto a fixed Korean long-form report outline.

Do not draft the report and do not modify, replace, extend, or reorder the outline. Submit one requirement map through plasma.report.requirements.submit.

Mission ID: %s
Report title: %s
Current report request direction_hint from pending event %s: %s

Fixed outline:
%s

Indexed destinations (the indices are 1-based and must be copied exactly into owner):
%s

Eligible user-authored events to review, oldest first:
%s

Plasma tool binding: use mission_id %s, session_id %s, pending_event_id %s, plan_event_id %s, idempotency_key %s, and producer {"type":"agent_session","id":"%s"}.

Method:
- Use plasma.research.read to inspect every eligible event listed above. Do not inspect or cite any other ledger event.
- The current pending event must appear in reviewed_event_ids. Include only eligible event IDs that you actually read.
- Extract only explicit, still-active requirements for the report output: requested subject emphasis, comparison, format, audience treatment, inclusion, or exclusion.
- Do not turn acknowledgements, workflow commands, research-process instructions, source facts, inferred preferences, or agent-generated text into requirements.
- When a later user instruction supersedes an earlier one, preserve only the active instruction and cite the event IDs needed to trace it.
- Assign each requirement to exactly one existing Section that can satisfy it naturally. Requirements are writing instructions, not sources or evidence.
- Before submitting, verify that each owner's indexed title and purpose match the instruction. Do not default to the first Section.
- If an explicit requirement cannot fit any existing Section, record a concise unmapped_reason. Never create a Section or distort an unrelated Section to absorb it.
- Use stable req_ identifiers within this submission.

Submit this shape. Every source_event_id must also occur in reviewed_event_ids. If the tool returns a retryable validation error, correct and resubmit; make at most three parsed submission calls total:
{
  "mission_id": %q,
  "session_id": %q,
  "pending_event_id": %q,
  "plan_event_id": %q,
  "idempotency_key": %q,
  "producer": {"type":"agent_session","id":%q},
  "requirement_map": {
    "reviewed_event_ids": ["evt_..."],
    "requirements": [
      {
        "requirement_id": "req_...",
        "instruction": "observable output requirement",
        "source_event_ids": ["evt_..."],
        "owner": {"part_index": 1, "section_index": 1}
      },
      {
        "requirement_id": "req_...",
        "instruction": "requirement that does not fit the outline",
        "source_event_ids": ["evt_..."],
        "unmapped_reason": "why no existing Section can own it"
      }
    ]
  }
}

After the tool succeeds, return exactly REQUIREMENTS_MAPPED as the complete final response. Do not return JSON, fences, or commentary.`,
		binding.MissionID, strconv.Quote(title), binding.PendingEventID, strconv.Quote(strings.TrimSpace(directionHint)),
		longformutil.AnyJSON(plan), indexedDestinations(plan), longformutil.AnyJSON(reviewEventIDs), binding.MissionID, binding.ToolSessionID, binding.PendingEventID, binding.PlanEventID, binding.IdempotencyKey, binding.ToolSessionID,
		binding.MissionID, binding.ToolSessionID, binding.PendingEventID, binding.PlanEventID, binding.IdempotencyKey, binding.ToolSessionID)
}

func indexedDestinations(plan reporting.SectionalReportPlan) string {
	var output strings.Builder
	for partIndex, part := range plan.Parts {
		fmt.Fprintf(&output, "Part %d: %s\n", partIndex+1, part.Title)
		for sectionIndex, section := range part.Sections {
			fmt.Fprintf(&output, "- Section %d.%d: %s | %s\n", partIndex+1, sectionIndex+1, section.Title, section.Purpose)
		}
	}
	return strings.TrimSpace(output.String())
}
