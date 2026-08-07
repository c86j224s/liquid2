# Report Direction Layering

## Question

Issue #210 asked whether a request-specific report direction should influence
the whole long-form report instead of only the plan and individual Section
drafts. The intended direction is a weak editorial axis: it may focus the
report, prohibit a particular interpretation, or request a presentation form,
but it must not replace source-grounded reporting or narrow the report so far
that mission-relevant coverage disappears.

## Candidate

The candidate keeps the user's original wording as the authority and adds only
a light planner interpretation:

1. The long-form planner receives the original direction before plan freeze.
2. It reflects that direction in the existing report structure and
   `ReportWritingContract`; no new persisted type or protocol field is added.
3. Content-writing stages receive both the original wording and the interpreted
   contract.
4. Style, semantic validation, evidence-gate, patch, and export stages remain
   outside the direction boundary.

The priority remains source facts, explicit user wording, planner
interpretation, and local writing discretion, in that order.

## Method

The comparison used the actual HTTP report route, product binaries, MCP source
reads, isolated databases and provider homes, and the existing strict long-form
configuration. A fixed source snapshot and one report direction were used for
all arms. The matrix compared baseline and candidate behavior in both serial
and `section_fanout` strategies.

The direction asked the report to focus on source lifecycle and durable work
state, avoid interpreting MCP as the owner of domain policy, and include a
comparison table without letting the table replace the prose.

Raw runs remain in the policy-defined local experiment archive. This document
contains only the redacted protocol, aggregate observations, and adoption
decision.

## First Run

| Arm | Parts | Sections | Approximate words | Full-reading result |
| --- | ---: | ---: | ---: | --- |
| baseline serial | 4 | 16 | 11,337 | Broad and rich, but less focused and more encyclopedic. |
| candidate serial | 3 | 9 | 6,704 | Clear and readable, but too compressed; relevant breadth was lost. |
| baseline fanout | 4 | 14 | 9,838 | Best initial balance; naturally aligned with much of the requested direction. |
| candidate fanout | 4 | 14 | 9,269 | Strong direction adherence with slightly denser implementation detail. |

All four reports respected the prohibited interpretation and presentation
request. The important failure was the candidate serial planner treating
"focus" as permission to reduce the outline. This was a real quality regression,
so the first candidate did not pass productization.

## Narrow Correction and Intermediate Check

The prompt contract gained one invariant: request direction may adjust emphasis,
interpretation, ordering, and presentation, but it must not reduce the
mission-relevant coverage or depth required by the objective and sources. The
contract deliberately does not prescribe fixed Part or Section counts.

Only the two candidate paths were rerun after this correction.

| Arm | Parts | Sections | Approximate words | Full-reading result |
| --- | ---: | ---: | ---: | --- |
| corrected candidate serial | 4 | 14 | 9,331 | Recovered breadth and depth while keeping the requested center. |
| corrected candidate fanout | 5 | 16 | 11,092 | Preserved the requested center and covered the full product boundary. |

Both corrected reports were read from beginning to end. They retained source
lifecycle and durable state as the organizing axis, avoided the prohibited MCP
interpretation, and used tables and diagrams as reading aids rather than
substitutes for explanation. The serial report recovered the areas lost in the
first candidate. The fanout report remained longer and more implementation
dense, which matches the existing long-form strategy rather than a new
direction-induced regression.

After this run, the prompt boundary was narrowed so that final writers and
reader editors receive only the `writing_contract`, not the complete plan JSON.
The downstream contract also states that existing requirement mapping owns each
concrete output, preventing report-wide wording from making every Section
repeat the same item. This was a final containment change rather than a new
candidate behavior.

## Final Product-Path Confirmation

The final candidate was rerun through both long-form product strategies using
the same binary built from the final issue #210 candidate after the containment
change. The source fixture, direction, model, reasoning effort, and strict
long-form configuration remained fixed. The planner was not given a target Part
or Section count.

| Arm | Parts | Sections | Approximate words | Full-reading result |
| --- | ---: | ---: | ---: | --- |
| final candidate serial | 4 | 15 | 9,976 | Kept source approval, observation, reuse, and durable work state at the center while preserving the broader product boundary. It repeated some boundary principles more often than fanout, but without damaging flow or coverage. |
| final candidate fanout | 4 | 14 | 10,800 | Divided product boundaries, source lifecycle, durable work, and display responsibilities cleanly; tables and diagrams appeared only where they supported distinct explanations. |

Both final reports were read again from beginning to end. Both avoided the
prohibited interpretation that MCP owns domain policy and retained source
lifecycle and durable work state as the report's organizing axis. Serial did
not reproduce the first candidate's coverage loss. Fanout did not duplicate a
report-wide presentation requirement across every Section. Its tables covered
different comparisons and did not replace the prose.

This is not a statistical claim that the candidate is universally superior.
It is a productization judgment that the final candidate, on the actual product
path and both execution strategies, applies the direction without the observed
coverage regression or a new readability regression.

After the experiment, the branch was rebased onto a newer `main` that added
planned-general-report prose guidance and hid report outputs from research
discovery. Neither change alters long-form guidance selection, `source_snapshot`
reads, or this direction-layering boundary, and the full suite passed after the
rebase. The candidate reports were therefore not regenerated.

## Runtime Caveat

Both final runs completed, but existing finalization MCP validation errors were
observed and recovered automatically. The recurring categories included
producer/session mismatch, UTF-8 offset alignment, incorrect draft IDs, and
submission before a read-only packet was complete. Serial also completed after
removing an unapproved evidence reference. The direction prompt is not sent to
style, semantic-validation, or evidence-gate stages, so these failures are
recorded as existing finalization fragility rather than attributed to this
candidate. Issue #210 does not change that boundary.

## Decision

Adopt the corrected layering for long-form reports. The decision is based on
full-report reading of the final product-path rerun, not a claim that the
candidate is universally better than the baseline. The accepted conclusion is
narrower: request direction now survives planning and downstream content
assembly without the observed coverage-loss or output-duplication regression.

The change does not add a new API, MCP schema, UI control, persistence model, or
report type. It does not route report direction into conversation, workflow,
style correction, semantic validation, evidence checking, patching, or export.
