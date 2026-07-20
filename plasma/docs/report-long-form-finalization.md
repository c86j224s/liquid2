# Long-Form Report Finalization

Web long-form reports keep the existing plan, section drafting, Part assembly,
session policy, H5, and designed HTML workflow. The default execution strategy
is serial. A separate long-form-only "fast parallel" option may fan out section
drafting from the canonical plan session, then returns to the same Part
assembly and finalization contract.

The final handoff is unchanged in both strategies: the report agent submits the
opening and closing through `plasma.report.long_form.finalize`; the server
assembles them with the durable Part artifacts and atomically creates the
existing raw Markdown artifact and `report.artifact.created` event.

Planned reports and CLI report behavior do not use this command.

## Execution Strategies

`serial` is the default long-form strategy. It chains planning, each section,
each Part, and finalization through the existing report session sequence.

`section_fanout` is an explicit browser long-form option. It creates one
canonical plan through the existing `plasma.report.plan.submit` boundary, then
forks the report-plan provider session for independent section workers. Each
section still uses the normal section prompt and bounded source tools. The
browser runner executes at most eight section workers at once. Part assembly
waits for the section artifacts in that Part and preserves their bodies.
Finalization still uses `plasma.report.long_form.finalize`; the agent does not
submit full final Markdown.

The strategy is stored on `report.draft.pending` as `execution_strategy` so
restart and stale recovery use the same path. Omitted or `serial` values keep
the default serial behavior. `section_fanout` is invalid for planned,
one-take, CLI, H5, patch, or designed HTML requests.

## Public Tool Contract

The tool is exposed only in a long-form final session with a complete hidden
runner binding and explicit tool enablement. Its closed input contains exactly:

- `mission_id`, `session_id`, `pending_event_id`, and `plan_event_id`
- `idempotency_key`
- `producer`, fixed to the bound MCP tool session
- `opening_markdown` and `closing_markdown`

The agent cannot select the final artifact ID, filename, title, report mode,
Part order, section order, provider provenance, model settings, or full report
Markdown. These values are server-bound and checked against the mission ledger
and raw artifacts before commit.

The raw final artifact and existing canonical event are committed in one SQLite
transaction. An identical binding and assembled SHA replays the canonical
result. A different identity, provenance value, Part order, idempotency key, or
assembled content conflicts, including after restart or a concurrent call.
The conditional transaction also decides against the current ledger state, so a
terminal event for the pending report cannot race with creation of the final
canonical artifact and event.

## Completion And Retry

A final provider invocation succeeds only when a matching canonical artifact
and event exist and the normalized provider response is exactly
`REPORT_FINALIZED`. The final stage may be invoked at most twice. Both
invocations reuse the logical tool session, idempotency key, durable artifact
binding, and report provider-session chain; plan, section, and Part work is not
repeated.

For `resume_failed`, the runner reuses only plan, section, and Part artifacts
validated from the failed attempt's ancestor chain. It does not reopen or alter
the failed attempt; a restart does not reuse ancestor output.

The first response may supply a retry hint only when it is one legacy object
with exactly `front_matter` and `closing` string fields and exactly one root
trailing comma. The scanner removes that comma only. It rejects valid JSON,
fences, surrounding prose, extra values, unknown or duplicate fields, nested
trailing commas, and truncated input. Recovered text is non-durable guidance
for the second provider invocation and is never used by Web code to create the
artifact or event.

If the command commits but the exact sentinel is missing, the retry performs a
durable replay. A second missing sentinel is an acknowledgment anomaly; it does
not roll back the canonical report or append a contradictory report failure.

## Provenance And Observation

The public tool `producer` follows the existing MCP tool-session convention.
The final artifact and canonical event producer instead use the server-bound
report provider session. The canonical payload preserves the existing report
metadata and records the final tool session separately. Provider usage that is
known only after the tool call is not fabricated into the canonical event or a
conversation ledger event. The redacted operational log records only whether a
returned session exists and matches the bound session, together with token
aggregates and duration; it does not record the returned session ID or provider
usage details in canonical state.

The generic `mcp.tool.called` payload is unchanged. Tool name, success, and
created event IDs can be joined with canonical report provenance without
recording opening, closing, prompts, or full report text in trace summaries.
