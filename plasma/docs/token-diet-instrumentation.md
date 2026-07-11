# Plasma Token Diet Instrumentation

Status: design record for `plasma/token-diet-experiment`.
Date: 2026-07-01.

## Decision

This branch is re-scoped to instrumentation before token-diet experiments.

The immediate goal is not to reduce token usage, change memory behavior,
rewrite prompts, or change report generation. The goal is to make Plasma record
enough usage data from real product paths that later token-diet experiments can
compare against a trustworthy baseline.

This decision follows the workflow-step instruction experiment. That experiment
showed that a resumed Codex session can report much larger input-token usage
than the small prompt Plasma sends for the current turn. The likely cause is
accumulated provider-session context and prior tool output, not prompt stuffing
by Plasma. The product cannot currently prove that distinction because its
agent result model does not store provider usage.

## Current Evidence

Current product surfaces already keep prompts relatively thin:

- Browser turns build a short prompt and call the configured agent executor.
- Workflow steps use a bounded instruction prompt and do not inject full
  transcript or source bodies.
- Report generation calls the same executor repeatedly for one-take, planned,
  and long-form report phases.
- MCP calls are already recorded as `mcp.tool.called` trace events with bounded
  argument and result summaries.

The missing piece is usage telemetry:

- `AgentResult` stores text, session id, resumed flag, and log, but not token
  usage.
- `turn.agent.response`, `report.plan.created`, `report.section.created`,
  `report.part.created`, and `report.artifact.created` store duration/session
  metadata, but not provider-reported input/output/cached-token counts.
- MCP trace events store timing and summaries, but not enough byte-size fields
  to connect tool-result volume with later token growth.

The workflow-step experiment captured Codex JSONL externally. Across the
available `turn.completed` events from that experiment, the observed aggregate
was:

- 966 usage rows
- 511,647,549 total input tokens
- 426,710,144 cached input tokens
- 84,937,405 uncached input tokens
- about 83.4% cached-input ratio
- median input tokens about 410,203 per reported turn
- p90 input tokens about 1,159,225 per reported turn
- maximum input tokens 2,647,292 in a final-result turn

These numbers are experiment artifacts, not product telemetry. They are useful
as a warning sign and as a target for the first instrumentation baseline.

## Measurement Questions

Instrumentation must answer five separate questions.

1. What did Plasma explicitly send?

Record local prompt metrics for every agent call:

- prompt byte count
- prompt character count
- approximate local token estimate
- prompt SHA-256
- whether the call was new, resumed, compacting, or retrying after compaction
- mission id, user event id, workflow run id, workflow step id, pending report
  event id, and report phase when present

The product should not store full prompts by default. Prompt bodies can contain
user text or operational details; a hash and size are enough for token-diet
analysis. Experiments may opt into full prompt artifacts outside the product
database.

2. What did the provider report?

When the provider exposes usage data, record the normalized fields:

- input tokens
- cached input tokens
- uncached input tokens
- output tokens
- reasoning output tokens
- total tokens if provided
- provider name and executor name
- model and reasoning effort when known
- usage source, for example `codex_jsonl_turn_completed`

If a provider does not expose usage, record `usage_unavailable` with a reason.

3. What did the agent session do?

Record session lifecycle fields with every agent call:

- previous agent session id
- returned agent session id
- resumed flag
- compaction attempted flag
- compaction event id when present
- duration in milliseconds
- exit status and normalized error kind when the call fails

This separates three different causes of large usage: explicit prompt size,
provider session accumulation, and failure/retry behavior.

4. What tool and source data moved through MCP?

Extend MCP trace metadata so each call records bounded size information:

- tool name
- started/finished/duration
- success flag
- argument summary byte count
- result summary byte count
- raw result byte count when available without storing raw content
- truncation flag when summaries are shortened
- for source reads: source id, source kind, offset, requested max bytes,
  returned bytes, observed event id when present, and media type

The trace remains provenance and debugging telemetry. It is not source material,
evidence, saved knowledge, or a report.

5. Which product surface caused the usage?

Every usage record must identify the surface:

- `turn`: user-driven conversation turn
- `workflow_step`: bounded autonomous workflow step
- `report_plan`: report outline/plan call
- `report_markdown`: planned Markdown body generation call
- `report_section`: long-form section drafting call
- `report_part`: long-form part assembly call
- `report_frame`: long-form front matter and closing call
- `report_one_take`: one-pass Markdown report call
- `report_design`: designed HTML artifact call
- `compaction`: manual or automatic provider-session compaction

Without this field, later analysis will blur user chat, workflow, and report
costs together.

Follow-up candidate surface:

- `proposal_extraction`: post-turn source-backed proposal extraction. This is
  not part of the first implementation slice because it currently returns a
  nested status object rather than a separate terminal event.

## Event Model

Use a shared telemetry envelope called `agent_usage` inside the existing
terminal events. A separate event type should be reserved for delayed or
corrective telemetry only.

Preferred placement:

- `turn.agent.response` gets `agent_usage`.
- `turn.agent.compacted` gets `agent_usage`.
- `report.plan.created`, `report.section.created`, `report.part.created`, and
  `report.artifact.created` get `agent_usage` for the agent call that produced
  that event.
- `report.design.failed` and `report.draft.failed` get partial `agent_usage`
  when a provider call already returned usage before the failure was recorded.
- Agent error responses get partial `agent_usage` when the provider emitted
  usage before failure.
- `mcp.tool.called` gets `io_metrics`, not `agent_usage`.

Recommended shape:

```json
{
  "agent_usage": {
    "schema_version": 1,
    "surface": "workflow_step",
    "provider": "codex",
    "executor": "codex",
    "model": "gpt-5.5",
    "reasoning_effort": "xhigh",
    "prompt": {
      "bytes": 1946,
      "chars": 1946,
      "estimated_tokens": 487,
      "sha256": "..."
    },
    "session": {
      "previous_agent_session_id": "...",
      "agent_session_id": "...",
      "resumed": true,
      "compaction_attempted": false
    },
    "provider_usage": {
      "input_tokens": 2647292,
      "cached_input_tokens": 2351360,
      "uncached_input_tokens": 295932,
      "output_tokens": 27195,
      "reasoning_output_tokens": 8230
    },
    "duration_ms": 123456,
    "usage_source": "codex_jsonl_turn_completed",
    "usage_unavailable": false
  }
}
```

The exact fields may be absent when unavailable, but field names should remain
stable once written.

## Capture Strategy

Codex should be instrumented first because the current product uses Codex as
the main executor and the local CLI supports `--json` for both `codex exec` and
`codex exec resume`.

Implementation direction:

1. Add usage fields to the shared agent request/result model.
2. Run Codex with `--json` while keeping `--output-last-message` for the final
   answer body.
3. Parse `turn.completed.usage` events from stdout JSONL.
4. Continue extracting the session id from JSONL or from the existing log path.
5. Preserve bounded log excerpts for errors, but do not store complete JSONL
   logs in normal product events.
6. Include prompt metrics computed locally before execution.

For providers that do not expose usage yet, the executor should still return
prompt metrics, session metadata, and `usage_unavailable`.

## Implementation Notes

The first implementation slice records:

- Codex `--json` output parsed from `thread.started` and
  `turn.completed.usage`;
- local prompt metrics for Codex executor calls;
- `agent_usage` on browser turn responses, workflow step responses, compaction
  events, report planning events, report section/part events, final Markdown
  report artifact events, designed HTML export events, and report failure
  events when partial provider usage is available;
- `io_metrics` on `mcp.tool.called` events, including normalized read metrics
  for `plasma.sources.read` and `plasma.research.read`:
  requested offset/max bytes, returned content bytes, response truncation,
  next offset, source/object identifiers, media type when known, extraction
  metadata for PDF text reads, and observation event metadata for live local
  path reads.

The implementation intentionally does not store full prompts or full provider
JSONL logs in success events. Agent turn failure events continue to store
bounded log excerpts; report failure events store normalized usage payloads
without provider log excerpts.

Known follow-up:

- Codex execution still captures combined stdout/stderr in memory before
  parsing usage and session metadata. That does not leak full JSONL into normal
  product events, but it can increase worker memory use on very large JSONL
  streams. A later hardening pass should replace this with a streaming parser
  plus a bounded ring/head-tail log excerpt.

## July 1 Baseline Interpretation

The first active-browser measurement after instrumentation showed an important
distinction:

- Plasma's explicit prompts were small, usually a few kilobytes for workflow
  steps and about 13 KB for planned Markdown report generation.
- Provider-reported input tokens were much larger because Codex resumed a
  provider session that already contained transcript, tool calls, tool outputs,
  reasoning records, and repeated internal model calls.
- Cached input tokens were high, so prompt caching was not simply failing. The
  issue is that cached context still counts toward provider usage and context
  pressure; caching reduces repeated computation/cost, not the amount of
  session context the agent carries.
- MCP keeps Plasma from pasting source bodies into prompts, but it does not make
  read content free. If the agent reads source or ledger content through a tool,
  the tool result can still enter the provider session transcript.

This means the first token-diet question is not "are prompts too long?" but
"which product work should share one provider session, and which work should be
isolated?"

Current conclusion:

- Research conversation and autonomous investigation can share a session while
  they are part of the same investigation thread. That session may grow because
  the agent genuinely reads sources and compares findings.
- Report generation should be isolated from the research session by forking the
  provider session at report start. The fork still inherits context up to the
  fork point, so it does not reduce the report's starting context. It does stop
  report planning, drafting, section assembly, and export work from inflating
  the later research conversation.
- A stronger token-diet experiment can still compare same-session resume against
  a fresh session that reconstructs state through Plasma MCP, but that is a
  larger product decision than report-session isolation.

MCP savings still need a controlled comparison. Current telemetry can show MCP
tool result sizes and provider token growth, but it cannot prove the percentage
saved versus a source-body-in-prompt baseline without an A/B run.

## July 2 Phase 1 Closeout

Token-diet phase 1 is closed as an instrumentation and isolation phase.

What is proven:

- Plasma now records enough `agent_usage` and MCP `io_metrics` to inspect real
  product token growth by surface, provider session, prompt size, cached input,
  uncached input, output, duration, and failure state.
- Non-one-take report generation now runs in an isolated fork when a fork-capable
  executor and pre-report research session are available.
- The live Gemma4 mission measurement confirmed the intended session split:
  research/autonomous investigation used session `019f2096...`, long-form
  Markdown report generation used report fork `b5c9e863...`, designed HTML used
  session `019f2129...`, and the post-report research turn resumed the original
  research session `019f2096...`.
- In that Gemma4 measurement, long-form report generation added about 10.42M
  input tokens and 3.42M uncached input tokens to the report fork instead of the
  research session. The first post-report research turn added about 907K input
  tokens and 201K uncached input tokens to the original research session. This
  supports the isolation claim: report work no longer inflates later research
  turns with report planning, drafting, part assembly, and framing context.
- Designed HTML export was also isolated from the research session. The observed
  Gemma4 designed export used about 44K input tokens, 39K uncached input tokens,
  26K output tokens, and 70K total tokens in its own session.

What is not proven:

- Phase 1 does not prove that total report-generation cost is lower. It proves
  cost and context isolation.
- Phase 1 does not prove that MCP has reduced research-token usage versus a
  non-MCP or source-body-in-prompt baseline.
- Phase 1 does not solve research-session growth. The same Gemma4 mission showed
  the research/autonomous investigation session itself growing by about 17.7M
  input tokens and 4.12M uncached input tokens before report generation.
- Phase 1 does not decide whether Plasma should use short-lived agent sessions,
  ledger-reconstructed context, automatic compaction, or external memory
  summaries for research turns.

Phase 1 product conclusion:

- Keep report-session isolation as the default for non-one-take reports.
- Keep usage instrumentation active.
- Pause token-diet work here. Further reductions belong to a separate phase 2
  effort, not to this branch's phase 1 closure.

## Phase 2 Token-Diet Candidates

Phase 2 should start from research-session growth, not from report-session
isolation.

Primary candidate areas:

1. Research-session growth control.

   Measure why long autonomous investigation sessions grow quickly even when
   Plasma prompts are small. Separate source reads, ledger reads, MCP tool
   outputs, repeated workflow instructions, provider internal transcript growth,
   and repeated reasoning work. Do not assume prompt stuffing until the recorded
   prompt metrics and provider usage contradict each other.

2. MCP savings A/B.

   Run a controlled comparison between MCP-first source access and an explicit
   source-body-in-prompt baseline. Current telemetry can show bounded MCP result
   sizes, but only an A/B run can estimate the actual savings and quality tradeoff.

3. Ledger-reconstructed or fresh-session research turns.

   Compare a resumed provider session against a fresh agent session that
   reconstructs mission state through Plasma MCP tools. This must preserve the
   direct-agent MCP scenario: an agent using Plasma without the web UI should
   still be able to recover mission state from the ledger and sources.

4. Conversation and memory storage policy.

   Decide whether user/agent conversation turns should be stored as retrievable
   mission memory, how much of them should be exposed through MCP, and how to
   prevent result text from being misclassified as source material. This is a
   product semantics decision before it is a token optimization.

5. Compaction strategy.

   Measure manual and automatic compaction as a separate intervention. Compaction
   may reduce context pressure, but it can also discard useful nuance. It must be
   tested for answer quality, not only token count.

6. Budget and stop-condition visibility.

   Add product-visible summaries that let the user see when a mission is spending
   tokens on source discovery, source reading, repeated workflow steps, report
   drafting, or design export. This should support user steering without adding
   friction to normal investigation.

Phase 2 should not begin by rewriting prompts blindly. The first step is a
measurement plan that uses phase 1 telemetry to select the highest-impact
surface and defines quality checks alongside token checks.

## Baseline Collection

After implementation, collect a small product baseline before changing memory
or context behavior:

- one normal conversation turn on an existing mission
- one resumed conversation turn on the same mission
- one bounded workflow run with several steps
- one long-form report generation
- one designed HTML report generation
- one manual or automatic compaction attempt if context pressure appears

Each run should leave ledger-visible records with:

- explicit prompt size
- provider-reported usage when available
- cached and uncached input split when available
- output and reasoning-output usage when available
- session resume status
- MCP call count and result-size summary
- duration and failure state

The first baseline should be summarized in a follow-up experiment note under
`plasma/docs/experiments/`, not inferred from console logs alone.

## Non-Goals

This branch should not yet:

- switch the product to short-session memory
- stop resuming provider sessions
- compact automatically beyond existing behavior
- rewrite prompts for token reduction
- change report composition strategy
- alter source, evidence, saved knowledge, or report semantics
- store raw prompts or full provider JSONL logs in normal ledger events

Those are future experiments that should use this instrumentation as evidence.

## Acceptance Criteria

The instrumentation work is ready for token-diet experiments when:

- at least one browser turn records `agent_usage`;
- at least one workflow step records `agent_usage`;
- report generation records usage per agent call, not only one total;
- MCP traces expose enough byte-size metadata to relate tool output volume to
  later provider input growth;
- unavailable usage is explicit, not silently omitted;
- a documented baseline run exists with real product numbers.
