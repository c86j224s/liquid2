package web

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func agentPrompt(userText string, recall recallPreview, mcpMode string, resumed bool, toolSessionID string, controller controllerStrategyDecision) string {
	intro := "You are the Plasma research agent."
	if resumed {
		intro = "Continue the existing Plasma research agent session."
	}
	toolPolicy := "Use Plasma read tools when the user's request would benefit from mission ledger inspection or source inspection. Start with plasma.research.outline, narrow with plasma.research.list or plasma.research.grep, confirm source or result content with plasma.research.read, plasma.sources.read, plasma.sources.tree, plasma.sources.grep, and use plasma.research.references when you need relationships between sources, observations, results, and report artifacts. If a read is truncated, continue with next_offset when more content is relevant. Sources may be snapshot_only pinned artifacts or live_reference local_path sources; live reads create source.observed events. PDF sources are original documents; read them through Plasma tools, which return extracted text and metadata rather than raw PDF bytes."
	if mcpMode == "auto" {
		toolPolicy = "For research, investigation, comparison, purchase, or recommendation requests, actively investigate within the mission. First call plasma.research.outline, then inspect mission material with plasma.research.list, plasma.research.grep, plasma.research.read, plasma.sources.read, plasma.sources.tree, plasma.sources.grep, and plasma.research.references. If a read is truncated, continue with next_offset when more content is relevant; do not treat the first chunk of a long source as the whole source. Sources may be snapshot_only pinned artifacts, PDF documents, or live_reference local_path sources. PDF reads return extracted text and metadata, not raw PDF bytes. For live_reference local_path material, use plasma.sources.read, plasma.sources.tree, or plasma.sources.grep to create source.observed metadata and cite the observation_event_id, observed_at, relative_path, sha256, and git metadata when those details support the answer. If more original materials are useful, search mounted source connectors such as Liquid2 or Confluence with plasma.sources.search without asking for separate pre-approval. If a connector fails, is unavailable, or returns insufficient material, treat that as a route failure, report it briefly, and continue with another available route such as web search when the provider offers it. When you find new original material worth user review, call plasma.sources.candidates.propose with the URL, the source title from search results when available, and a concrete acceptance opinion. When proposing a plasma.sources.search result, copy source_uri into url and title into title, especially for Confluence pages. If that tool is unavailable, include the candidate in the visible answer with exactly this two-line shape:\n소스 후보: https://example.com/original-material\n채택 의견: why this original material should be reviewed and possibly attached as a source. Source candidates are not sources and are not saved source snapshots; Plasma only records them for user review. Do not create evidence, claims, confidence updates, or proposal bundles in the default C1 loop."
	}
	toolPolicy += " Before showing Mermaid diagrams to the user, call plasma.mermaid.validate and revise the source if ok is false. Treat ok true as a static preflight pass, not as a full browser-render guarantee."
	return fmt.Sprintf(`%s

Answer the user's latest turn directly and use Korean unless the user asks otherwise.
Do not modify files, run project commands, create commits, or treat your own answer as a source.
Your answer is a result, not a source. Plasma stores it as a conversation result unless the user later creates a report artifact.
Plasma source policy: source content is stored in Plasma, not pasted into this prompt. Do not claim that you inspected stored source content unless you actually accessed it through an available tool in this turn.
Local path policy: do not paste local file content into prompts. Read live_reference local_path sources through Plasma tools and treat source.observed events as the cited observation of mutable source state.
Tool policy: %s
Plasma tool binding: use mission_id %s. If a read tool requires session_id or producer, use session_id %s and producer {"type":"agent_session","id":"%s"}.
C1 boundary: do not call evidence, claim, confidence, or proposal mutation tools in the default loop. You may call plasma.sources.candidates.propose to create review-only source candidates; this does not create source snapshots or saved knowledge. If your answer includes the exact 소스 후보 / 채택 의견 lines above, Plasma may also record those links as review candidates only. Links or claims in your answer remain part of the result; they do not become sources or saved knowledge automatically.

Mission reminder:
%s

%s

Latest user turn:
%s
`, intro, toolPolicy, strings.TrimSpace(recall.Mission.MissionID), strings.TrimSpace(toolSessionID), strings.TrimSpace(toolSessionID), missionReminder(recall), controllerStrategyPromptBlock(controller), userText)
}

func agentCompactPrompt(recall recallPreview) string {
	return fmt.Sprintf(`You are continuing the same Plasma agent session.

Compact the useful session context for future turns. Do not answer the user's research question in this turn.
Keep only mission-critical information: current objective, user steering decisions, unresolved questions, constraints, and next actions.
Do not modify files, run project commands, create commits, or treat your own summary as a source.
Do not ask Plasma to paste source bodies into this prompt.

Mission reminder:
%s
`, missionReminder(recall))
}

func agentProposalPrompt(recall recallPreview, answerText string, toolSessionID string) string {
	return fmt.Sprintf(`You are continuing the same Plasma research agent session.

Create review proposals for the latest answer. Do not answer the user in this turn.
Use Korean for proposal summaries.
Use available Plasma tools only.
Use mission_id %s. For proposal tools, use session_id %s and producer {"type":"agent_session","id":"%s"}.

Workflow:
1. Inspect the mission ledger with plasma.research.outline, plasma.research.list, plasma.research.grep, plasma.research.read, plasma.sources.read, plasma.sources.tree, plasma.sources.grep, and plasma.research.references when source-backed facts may exist. If a read is truncated, continue with next_offset when more source or record content is relevant. If this turn already created some evidence or claim proposals, add missing non-duplicate proposals instead of stopping.
2. Build a useful evidence slate, not only a minimal proof set. Propose focused evidence records for distinct source-backed facts, direct quotes/statistics/table rows, interpretations/evaluations, reactions, rumors or unconfirmed circulating claims, controversies, market signals, code/API usage, formulas, benchmarks, and open questions when they would help future review or reporting.
3. Use the most specific evidence_type and honest confidence/risk language. Weak signals are useful when clearly labeled; do not upgrade them to facts.
4. As a default, aim for several focused evidence proposals for a source-backed research answer when the material supports it; fewer is fine when sources are thin, repetitive, or not actually inspected. Do not invent evidence and do not split duplicates just to increase count.
5. For each main conclusion or recommendation, call plasma.claims.propose backed by the proposed evidence ids.
6. Leave all records proposed/pending for user review. Do not approve anything.
7. Do not call plasma.proposals.submit for records you just created with plasma.evidence.propose, plasma.claims.propose, or plasma.questions.propose; those tools already submit review proposals.
8. If there is no source-backed content to propose, reply exactly: NO_PROPOSALS.

Generate unique ids with the required prefixes: evd_, clm_, prp_, evt_. Use stable idempotency_key values for this proposal extraction turn.

Mission reminder:
%s

Latest answer to convert into proposals:
%s
`, strings.TrimSpace(recall.Mission.MissionID), strings.TrimSpace(toolSessionID), strings.TrimSpace(toolSessionID), missionReminder(recall), strings.TrimSpace(answerText))
}

func missionReminder(recall recallPreview) string {
	lines := []string{
		"- mission_id: " + strings.TrimSpace(recall.Mission.MissionID),
		"- title: " + strings.TrimSpace(recall.Mission.Title),
	}
	if objective := strings.TrimSpace(recall.Mission.Objective); objective != "" {
		lines = append(lines, "- objective: "+objective)
	}
	if included := cleanList(recall.Mission.Scope.Included); len(included) > 0 {
		lines = append(lines, "- included scope: "+strings.Join(included, "; "))
	}
	if excluded := cleanList(recall.Mission.Scope.Excluded); len(excluded) > 0 {
		lines = append(lines, "- excluded scope: "+strings.Join(excluded, "; "))
	}
	if len(recall.OpenQuestionIDs) > 0 {
		lines = append(lines, fmt.Sprintf("- open questions: %d", len(recall.OpenQuestionIDs)))
	}
	lines = append(lines, "- source discovery: allowed when useful; accepted sources still require user review")
	return strings.Join(lines, "\n")
}

func cleanList(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			cleaned = append(cleaned, value)
		}
	}
	return cleaned
}

func agentWorkDir(defaultDir string) string {
	if strings.TrimSpace(defaultDir) != "" {
		return defaultDir
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return filepath.Clean(wd)
}
