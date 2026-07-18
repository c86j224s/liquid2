# Long-Form Report Fanout Experiment - 2026-07-16

This experiment evaluated issue #103: compare the current long-form report path
with a section-level fanout variant before changing product behavior.

The product question is narrow. Plasma already has a working long-form report
path that preserves section bodies and assembles a coherent final report. The
experiment asks whether parallel section or part execution can reduce wall-clock
time without making the report read worse.

## Status

- Issue: #103
- State: completed; productized as an explicit long-form option
- Public plan: [`protocol.md`](protocol.md)
- Raw archive target:
  `research-artifacts/liquid2/plasma/experiments/21-report-fanout-2026-07-16/`

## Product Question

Can Plasma fan out long-form report drafting across sections or parts while
preserving source fidelity, narrative flow, tone consistency, and safe retry
boundaries?

## Compared Arms

| Arm | Meaning |
| --- | --- |
| `A-serial-current` | Current product-equivalent long-form path. Planning, section drafting, part assembly, and final assembly run through the existing serial lifecycle and session chain. |
| `B-section-fanout` | One canonical long-form plan is created, then each section is drafted independently from the plan boundary. Part assembly waits for all sections in that part and still uses the existing section-preserving assembly contract. |

The first execution deliberately excludes part-level fanout and recursive
merge/tree synthesis. Those ideas remain useful follow-ups, but this run keeps
one product question isolated: whether section drafting can fan out without
making the final long-form report read worse.

The confirmatory run uses up to 24 diverse topic packets, paired by topic across
the two arms. That is the largest currently practical preregistered fixture set
for this family of product-path report experiments.

## Success Shape

A fanout arm is worth productization only if it clears both gates:

1. It materially improves wall-clock time for long-form generation.
2. It is non-inferior to the serial baseline on report quality.

Quality is not judged by length alone. Human-readable review must check flow,
tone, repetition, omission, source fidelity, and whether the final report still
feels like one authored document rather than stitched fragments.

## Result Summary

The confirmatory run completed 24 paired topics, 48 total report runs, with no
terminal failures.

`B-section-fanout` was faster than `A-serial-current` on every paired topic:

- paired topics: 24
- candidate faster: 24
- baseline faster: 0
- median wall-clock improvement: 263.5 seconds
- mean wall-clock improvement: 263.9 seconds
- one-sided exact sign-test p-value for speed: `5.96e-08`

The speed result is statistically decisive for this fixture set. The smallest
observed win was 22.8 seconds and the largest was 517.7 seconds.

Quality was not reduced to a single automatic score in this run. The public
guardrail metrics were:

- final report length ratio, candidate divided by baseline: median `0.93`,
  mean `0.97`;
- candidate report was longer on 10 topics and shorter on 14 topics;
- section preservation ratio was higher for the candidate on 20 of 24 topics,
  with a median delta of `+0.039`.

These metrics do not prove prose quality by themselves. They do indicate that
the speed win did not come from a uniform truncation of reports, and the
section-preserving assembly contract remained intact. Blind judge packets were
written under the local archive for later editorial review.

Manual reading gave the same directional conclusion but with more nuance. The
fanout reports were often easier to read because they were shorter, less
repetitive, and paced better. Labor-statistics and climate-adaptation samples
favored fanout. An open-source-governance sample favored the serial baseline for
depth and a more slowly developed authorial argument. This means fanout is not a
strict replacement for the serial path, but it is good enough to expose as an
explicit long-form option.

Decision: productize `B-section-fanout` as an opt-in "fast parallel" long-form
execution strategy. Keep the current serial long-form path as the default.

## Non-Goals

- Do not change the default long-form product path.
- Do not add a separate report mode. Fanout is an execution strategy for
  long-form reports only.
- Do not use development or release Plasma databases.
- Do not commit raw reports, prompt packets, provider state, screenshots,
  session identifiers, or copied source corpora.
- Do not treat agent-produced section drafts as sources.

## Files

- Protocol: [`protocol.md`](protocol.md)
- Public aggregate: [`aggregate.json`](aggregate.json)
- Local blind packets: `judging/packets/` under the raw archive target.

## Productization Boundary

The accepted product change is deliberately narrow:

- Browser long-form report controls expose a "serial" versus "fast parallel"
  execution choice.
- The request stores the selected execution strategy on `report.draft.pending`
  so restart and stale recovery can use the same path.
- `section_fanout` is accepted only when `report_mode` is `long_form`; planned
  and one-take reports keep their current behavior.
- Planning still uses the existing `plasma.report.plan.submit` MCP boundary.
- Final assembly still uses the existing `plasma.report.long_form.finalize` MCP
  boundary. The agent does not submit full final Markdown.
- Section drafting is the only fanout point. Each section worker starts from a
  fork of the canonical report-plan provider session and uses the same section
  prompt shape and bounded source tools as the serial long-form path.
- Part assembly and final assembly continue to preserve section bodies instead
  of rewriting them.

The implementation intentionally does not productize part-level fanout,
recursive tree merging, new tone prompts, new source classifications, or a new
report artifact type.
