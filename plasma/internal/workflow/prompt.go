package workflow

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

const controlMarker = "PLASMA_WORKFLOW_CONTROL:"

// StepPrompt는 workflow 단계용 에이전트 프롬프트를 만든다. 사용자의 원 요청,
// 파생된 run goal, 현재 단계 지시를 구분해서 전달하고, 마지막 control marker
// 제출 계약을 프롬프트에 포함한다.
func StepPrompt(view workflowstate.WorkflowRunView, instruction string, toolSessionID string, resumed bool) string {
	intro := "You are the Plasma research agent running one bounded workflow step."
	orientation := "Use Plasma read tools when useful. Start with plasma.research.outline and retain its last_sequence for later change checks. Then use plasma.research.list, plasma.research.grep, plasma.research.read, plasma.sources.read, plasma.sources.tree, plasma.sources.grep, and plasma.research.references as needed. If more original materials are useful, use plasma.sources.search."
	if resumed {
		intro = "Continue the existing Plasma research agent session for one bounded workflow step."
		if view.CompletedStepCount == 0 {
			orientation = "This is the first step of a new workflow run in the existing conversation. Reorient to the current mission and this run's user request before continuing from prior session context. Start with plasma.research.outline and retain its last_sequence for later change checks. Then use plasma.research.list, plasma.research.grep, plasma.research.read, plasma.sources.read, plasma.sources.tree, plasma.sources.grep, and plasma.research.references as needed. If more original materials are useful, use plasma.sources.search."
		} else {
			orientation = "Use Plasma read tools when useful. Continue from the existing session context for this workflow run. Do not re-read plasma.research.outline or plasma.research.list only to regain orientation. When you have a concrete reason to check mission changes, call plasma.research.changes with the last confirmed sequence from an earlier outline or changes response, then retain current_sequence. If no confirmed sequence is available or resync_required is true, read plasma.research.outline instead. Use plasma.research.grep, plasma.research.read, plasma.sources.read, plasma.sources.tree, plasma.sources.grep, plasma.research.references, and plasma.sources.search as needed."
		}
	}
	return layeredStepPrompt(intro, orientation, view, instruction, toolSessionID)
}

func layeredStepPrompt(intro string, orientation string, view workflowstate.WorkflowRunView, instruction string, toolSessionID string) string {
	raw := strings.TrimSpace(firstNonEmptyWorkflowPrompt(view.UserInstructionRaw, view.Instruction, instruction))
	goal := strings.TrimSpace(firstNonEmptyWorkflowPrompt(view.RunGoal, view.Instruction, instruction))
	return fmt.Sprintf(`%s

Use Korean unless the user asked otherwise. Make one concrete progress step for the mission.
%s

Mission ID: %s
Tool session ID: %s
User's original autonomous-run request. Preserve its breadth and ambiguity; it outranks the derived goal and the current step instruction:
%s

Derived autonomous-run goal. This is a working interpretation, not a replacement for the user's original request:
%s

Instruction for this step:
%s

Rules:
- This is a bounded workflow step, not a separate mission and not a permanent product mode.
- If the step instruction is narrower than the user's original request, use it as the next concrete move without forgetting the original breadth.
- Do not let the derived goal close off possibilities that the user's original request intentionally left open.
- Sources are original materials. Your answer is a result, not a source. A report is a later Markdown artifact assembled from the conversation, source references, and saved mission material.
- Sources may be snapshot_only pinned artifacts, PDF documents, or live_reference local_path sources. PDF reads return extracted text and metadata, not raw PDF bytes. Live local path reads create source.observed events; cite observation_event_id, observed_at, relative_path, sha256, and git metadata when using mutable local material.
- Do not bulk-paste original material, local file content, conversation history, large recall JSON, or report corpora into the answer.
- Do not call evidence, claim, confidence, proposal, report-block, or report-AST mutation tools in the default C1 path.
- If you find new original material worth user review, include it in the visible result as an explicit source candidate. It does not need to be fully verified yet; make uncertainty clear in the acceptance opinion. Use exactly this two-line shape for each candidate:
  소스 후보: https://example.com/original-material
  채택 의견: why this original material should be reviewed and possibly attached as a source
- Source candidates are not sources. Plasma only stores them for user approval; do not say they were saved as sources.
- Return a concise user-visible result first.
- End with exactly one control marker line:
%s {"decision":"continue|stop","reason":"short reason","next_instruction":"optional next step"}

Use decision "continue" when the current step is complete but the user's original autonomous-run request or derived run goal still has useful remaining work that you can start now with the current mission state and available tools. Include a concrete, immediately executable next_instruction.
Before stopping, use current material for any useful synthesis, comparison, organization, or concise user-visible summary that still advances the run goal.
Use decision "stop" when the user's original autonomous-run request and derived run goal are satisfied, or no useful investigation or organization step remains executable now. Do not use stop merely because the current step instruction is complete.
Waiting for user approval or a reply, a source attachment, credentials, or a future external event is not executable workflow work. A pending source candidate, its staging, or its user review does not by itself keep the workflow open. Explain the current result and the condition for resuming, then use "stop" instead of polling.
Do not use "continue" to re-check the same unchanged condition or to wait for a future event.`, intro, orientation, strings.TrimSpace(view.MissionID), strings.TrimSpace(toolSessionID), raw, goal, strings.TrimSpace(instruction), controlMarker)
}

func firstNonEmptyWorkflowPrompt(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
