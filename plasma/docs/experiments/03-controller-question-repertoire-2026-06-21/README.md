# Controller Question Repertoire Pilot - 2026-06-21

This directory records a pilot run for the Plasma controller question repertoire
experiment.

## Current Decision - 2026-06-26

The latest valid investigation-controller experiment is
[`11-question-navigator-cwd-fixed-2026-06-26`](11-question-navigator-cwd-fixed-2026-06-26/README.md).
It ran C0, PAL2, and NAV across three mission classes and five seeds per
mission/variant pair, for 45 valid runs.

The result does not justify productizing a new controller. NAV was worse than
C0 in the aggregate, and PAL2 remained inconclusive. The current Plasma default
should stay close to C0: continue the same agent session, provide thin mission
guidance, and let the agent use MCP/source read tools. Future controller work
should be conditional and weak, not an always-on second researcher.

The earlier `10-question-navigator-2026-06-26` run is an invalid audit artifact:
resume turns ran from the repository root instead of the fixed source corpus.
The corrected run fixed this working-directory contamination and recorded a
zero-count contamination audit for investigation turn logs.

The pilot is intentionally smaller than the full experiment plan. It runs one
code-analysis mission across the four planned variants to validate the artifact
layout, controller question-only contract, contamination audit, and first-pass
judge flow before scaling to the full mission set.

The controller is not a researcher, source producer, judge, or report writer.
It only emits user-style steering questions for the main agent.

## Pilot Result

This pilot completed one code-analysis mission across all four variants:

- V0: baseline continuation
- V1: stagnation detect-only
- V2: creative switch on stagnation
- V3: scheduled divergent question

The pilot did not create a useful comparison between controller variants. After
the first main-agent turn, no stagnation trigger fired, and all four variants
received effectively the same follow-up question. The run is therefore recorded
as a harness and code-analysis smoke test, not as evidence that one controller
variant is better.

The useful product finding is consistent across the four reports: C1 Markdown
report generation stores the returned Markdown as a raw artifact and records a
`report.artifact.created` event, but the browser currently lists only artifact
metadata. The stored Markdown body is durable and readable internally, but the
C1 browser UI has no view or download action for that artifact.
