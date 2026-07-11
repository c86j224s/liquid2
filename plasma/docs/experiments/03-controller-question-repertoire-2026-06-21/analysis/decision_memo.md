# Decision Memo

## Decision

Do not select a controller variant from this pilot.

## Reason

The pilot did not create variant separation. The controller observed useful
progress after the first agent turn and selected the same confirmatory follow-up
question for all four variants. Because V1, V2, and V3 did not exercise their
distinctive behaviors, the run cannot support a ranking.

## Keep

Keep the C1 direction:

- thin prompts,
- same provider session resume,
- source and ledger access through tools,
- controller as question-only steering,
- no default claim/evidence/confidence/proposal machinery in the product path.

## Follow-Up Experiment

Status update, 2026-06-26: the follow-up work has been run. The current anchor
is `../11-question-navigator-cwd-fixed-2026-06-26/`, a corrected 45-run
C0/PAL2/NAV experiment. It did not validate a new default controller; NAV was
rejected as worse than C0 and PAL2 remained inconclusive.

Run a second pilot designed to force at least three controller decisions or a
deliberate stagnation point. The goal should be to observe:

- whether V1 detects stagnation without doing extra work,
- whether V2 asks a useful reframing question only when progress stalls,
- whether V3's scheduled divergence helps or distracts,
- whether the controller can deepen a codebase analysis after the first mapping
  turn.

## Product Follow-Up

Implement browser view/download for direct C1 Markdown report artifacts before
using report generation heavily in product testing. The current artifact is
stored, but the browser action surface is incomplete.
