# Long-Form Report Finalization

Web long-form reports keep the existing plan, section drafting, Part assembly,
session policy, H5, and designed HTML workflow. The default execution strategy
is serial. A separate long-form-only "fast parallel" option may fan out section
drafting from the canonical plan session, then returns to the same Part
assembly and finalization contract.

The active Web writing choices use the same final editorial handoff in both
strategies. The server assembles durable Part artifacts into an in-process
manuscript. A bound final editor reads and patches that manuscript, then submits
it atomically as the existing raw Markdown artifact and
`report.artifact.created` event. Stored legacy profiles retain the previous
opening/closing-only finalization semantics for replay and interrupted work.

Planned reports and CLI report behavior do not use this command.

## Part Assembly Edit Tools

The browser keeps three visible writing choices: visual planning,
section-centered writing, and section-centered writing with richer cluster
memory. The reader-facing writing contract is a common baseline under all
three, not a fourth choice. Internally, new requests use distinct composite
profile values so stored legacy profile values are not reinterpreted.

All three choices keep the same visual-aid default: source shape should suggest
the aid before the writer falls back to prose, so chronology tends toward
timeline, dependency toward flowchart, actor handoff toward sequence diagram,
lifecycle toward state diagram, ordered values toward source-backed chart, and
scenario or trade-off toward matrix/table. They also use the same Part assembly
MCP handoff. The Part agent must bounded-read every immutable Section bound to
that Part before editing its intro, transitions, and closing, then returns the
`PART_ASSEMBLY_SUBMITTED` sentinel.

The older `part-assembly-edit-tools` profile remains accepted for experiment
replay and stored-event compatibility, but it is not a separate browser choice.

This handoff does not let the agent rewrite Section bodies or submit complete
Part Markdown. The server still inserts the immutable Section artifacts and
creates the canonical Part artifact. Planned reports use the same writing
contract without Part or final stages. The compatibility one-take Web API uses
the shared writing guidance without inventing a plan. CLI reports, H5 patching,
designed HTML, and cost policy are unchanged.

## Execution Strategies

`serial` is the default long-form strategy. It chains planning, each section,
each Part, and finalization through the existing report session sequence.

`section_fanout` is an explicit browser long-form option. It creates one
canonical plan through the existing `plasma.report.plan.submit` boundary, then
forks the report-plan provider session for independent section workers. Each
section still uses the normal section prompt and bounded source tools. The
browser runner executes at most eight section workers at once. Part assembly
waits for the section artifacts in that Part and preserves their bodies.
Active choices return to the same bound manuscript editor and atomic submit
contract after Part assembly. Stored legacy profiles continue to use
`plasma.report.long_form.finalize` and do not submit full final Markdown.

The strategy is stored on `report.draft.pending` as `execution_strategy` so
restart and stale recovery use the same path. Omitted or `serial` values keep
the default serial behavior. `section_fanout` is invalid for planned,
one-take, CLI, H5, patch, or designed HTML requests.

## Public Tool Contract

The active final editor tools are exposed only in a long-form final session with
a complete hidden runner binding and explicit tool enablement:

- `plasma.report.long_form.final_edit.start` creates the server-owned manuscript
  from the ordered bound Parts.
- `plasma.report.long_form.final_edit.read` returns bounded UTF-8 slices.
- `plasma.report.long_form.final_edit.patch` applies bounded exact replace,
  insert-after, or append operations.
- `plasma.report.long_form.final_edit.submit` commits the edited manuscript
  through the canonical finalization transaction.

The agent cannot select the final artifact ID, filename, title, report mode,
Part order, section order, provider provenance, or model settings. It cannot
read sources or research during final editing, and it cannot mutate Section or
Part artifacts. Legacy `plasma.report.long_form.finalize` remains bound to its
closed opening/closing input for stored-profile compatibility only.

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
