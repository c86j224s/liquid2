# Report Plan Submission

Web planned and long-form reports use a durable MCP submission boundary between
planning and writing. The report runner creates a tool-session binding for the
provider planning turn and exposes `plasma.report.plan.submit` in addition to
the normal research tools. The binding does not create or select a provider
session; the existing `same_session` or `isolated_fork` policy remains the
provider-session authority.

The transition is:

```text
report.draft.pending
  -> report.plan.submitted
  -> validated provider exit (exact PLAN_SUBMITTED response and session lineage)
  -> report.plan.created
  -> existing writing and assembly
```

The strict tool input has common fields `mission_id`, `session_id`,
`pending_event_id`, `report_mode`, `idempotency_key`, `producer`, and `plan`.
Unknown fields are rejected at the root and in every nested object. A `planned`
plan keeps the current `summary`, `sections`, `coverage_notes`, and
`planned_omissions` fields. Sections keep `title`, `purpose`, and `target_refs`;
either summary or sections must be present, and existing planned whitespace
semantics remain unchanged. A `long_form` plan keeps `summary`, `parts`,
`coverage_notes`, and `planned_omissions`; each part has `title`, optional
`purpose`, and sections with `title`, optional `purpose`, and `target_refs`.
Long-form strings are trimmed, empty entries are removed, and coverage and
omission lists retain at most 24 entries.

The intentional issue #110 changes are limited to rejecting incomplete
long-form part/section structure instead of synthesizing it, and validating
every referenced claim, evidence, snapshot, question, and option with the
existing mission and report-eligibility rules. No new approval state, plan IDs,
or semantic size/count policy is introduced.

The public input `session_id` and producer identify the server-bound MCP tool
session; they are not a provider session. The server also binds the mission,
pending event, mode, idempotency key, executor/model/effort, and, when one
truthfully exists before the turn, an optional previous provider session. The
tool validates those values, the plan shape, and all source-reference kinds. It
records provenance only with a server-owned event producer and cannot create
the canonical plan. The runner selects
one valid submission for its current tool session and atomically promotes it
after `Invoke` returns, validates the actual returned provider-session lineage,
and writes that actual session only into canonical `report.plan.created`
provenance. A submitted-only attempt cannot advance
report progress; a later attempt uses a new tool session and ignores the stale
submission. Recovery after canonical creation continues from the canonical
event under the existing rules.

The planning prompt includes the concrete mission, tool session, pending event,
mode, idempotency key, and tool-session producer. The agent must obtain one
accepted submission. Every successfully parsed submit call consumes the
three-call tool-session budget, including successful calls and idempotent
replays. Before dispatch, stdio requires JSON-RPC exactly `2.0` and a request ID
that is a string or number. Missing or wrong versions and invalid ID shapes
return JSON-RPC invalid-request errors and never consume the tool budget.
Existing notifications remain response-free. Other envelope/protocol parse
failures also do not consume the budget. Retryable
validation failures may be corrected and resubmitted; the third validation
failure and every fourth parsed call are non-retryable and cannot reach
storage. Binding, conflict, and storage failures are non-retryable.

This boundary converts only Web `planned` and Web `long_form` response-JSON
planning. CLI planned reports retain their Markdown `plan_text` contract, and
CLI long-form reports remain rejected. Report writing, long-form section and
part assembly, source-read policy, report-session selection, one-take behavior,
H5 patching, G2 guidance, and designed HTML are unchanged.

Any future CLI or MCP report-start adapter must call the same reporting
lifecycle. It must not promote a tool submission directly or parse the final
provider response as a fallback plan.
