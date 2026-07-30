# Report Direct Explanation Writing Experiment

## Status

Full-run generation, aggregate analysis, and judging-packet preparation
completed for issue #179. The experiment compares the current baseline with both
the supporting `reader-paragraph-contract` candidate and the primary
`curiosity-led-explanation` candidate. No product default has changed.

This experiment tests whether the current `narrative-contract` baseline can be
improved by making Plasma outputs feel less like source annotations or
investigation logs and more like processed reading artifacts. Any candidate must
keep the report plan schema, UI choices, storage model, citation model, and
report assembly pipeline unchanged unless a later issue explicitly changes that
scope.

## Question

Plasma reports became more source-grounded after the narrative-contract work,
but some reports still read like assembled research material rather than
processed writing. The product goal is broader than source coverage: users want
to learn new information, understand it well, connect multiple sources into
insight, read a gathered version that is enjoyable enough to continue, and use
some outputs for market research or decisions. The release-server
writing-methodology research suggests that readability comes from reader intent,
information gaps, curiosity, ordering, paragraph purpose, and editing discipline
before sentence polish.

The experiment therefore asks:

> Can Plasma turn source-grounded material into a processed reading artifact
> that the user wants to continue reading, while preserving source-backed detail
> and useful caveats?

## Arms

| Arm | Guidance profile | Meaning |
|---|---|---|
| Baseline | `narrative-contract` | Current product writing baseline. |
| Supporting candidate | `reader-paragraph-contract` | Adds reader-path planning, section-level paragraph planning, compact claim-source memory, and a paragraph-quality self-check. This was implemented first and smoke-tested, but it is not the strongest expression of the product problem. |
| Primary candidate | `curiosity-led-explanation` | Makes the information gap, reader tension, surprising connection, insight path, and reason to keep reading explicit in existing plan fields. |

Candidates must keep the submitted plan schema unchanged. They should use
existing fields only:

- `writing_contract` carries the report-level reader question, takeaway,
  reading path, must-keep facts, compressible material, supporting-layer
  candidates, visual role, and tone.
- Part and Section `purpose` strings carry reader movement and paragraph-plan
  intent.
- `coverage_notes` carries compact claim-source memory and source-boundary
  reminders.

For the curiosity-led candidate, the same fields carry the opening question,
information gap, expected payoff, next-question chain, and places where evidence
should slow the prose down without turning the artifact into a source inventory.

## Runner

Use:

```bash
cd plasma
python3 scripts/experiments/report_direct_explanation_writing_experiment.py --action prepare --limit 8
python3 scripts/experiments/report_direct_explanation_writing_experiment.py --action run --limit 8 --workers 2
python3 scripts/experiments/report_direct_explanation_writing_experiment.py --action analyze --limit 8
python3 scripts/experiments/report_direct_explanation_writing_experiment.py --action packets --limit 8
```

With `--limit 8`, the runner creates eight topics across three arms: baseline,
supporting paragraph contract, and primary curiosity-led explanation. That is
24 long-form reports before judging packets. The runner reuses the existing
report-section experiment harness. It creates isolated databases and new
missions under the local experiment archive, builds a local `plasma` binary for
the run, and stores raw prompts, logs, reports, private mappings, and judging
packets outside the public repository.

## Initial Smoke Check

A one-topic smoke run completed on the shared fixture corpus across all three
arms. This is only an execution check, not a quality or product-direction
decision.

| Metric | Baseline | Paragraph Candidate | Curiosity Candidate |
|---|---:|---:|---:|
| Status | completed | completed | completed |
| Parts | 4 | 3 | 4 |
| Sections | 8 | 8 | 10 |
| Words | 3,332 | 3,207 | 4,148 |
| Wall seconds | 516.5 | 557.8 | 758.8 |
| Word ratio over baseline | 1.000 | 0.962 | 1.245 |
| Section ratio over baseline | 1.000 | 1.000 | 1.250 |

Two blind judging packets were created, one for each candidate against the
baseline. The result proves that the current three-arm harness and profile paths
can complete. It is too small to decide readability or adoption. The curiosity
candidate's longer output is an early signal to inspect for richer explanation
versus avoidable length, not an automatic win.

## Full Run Check

The full run completed across eight topics and three arms, producing 24 report
records and eight complete baseline-versus-candidate pairs. No arm failed. The
packet step produced 16 candidate comparison files, one packet per topic and
candidate arm.

| Metric | Baseline | Paragraph Candidate | Curiosity Candidate |
|---|---:|---:|---:|
| Completed reports | 8 | 8 | 8 |
| Mean words | 5,255.6 | 4,341.1 | 4,706.9 |
| Mean word ratio over baseline | 1.000 | 0.832 | 0.919 |
| Word-ratio range | 1.000-1.000 | 0.587-0.964 | 0.588-1.245 |
| Mean sections | 10.5 | 9.0 | 9.5 |
| Mean wall seconds | 744.8 | 681.8 | 654.9 |

The aggregate result is an execution and sizing check only. It shows that both
candidates can complete at full-run size and that the curiosity-led candidate
does not generally lengthen outputs in this fixture set. It does not decide
which candidate reads better; that requires direct reading and blind-packet
review.

## Decision Boundary

Adopt nothing from this experiment unless direct reading and blind-packet review
show all of the following:

- The candidate is easier to read as a full artifact, not merely smoother at the
  sentence level or more orderly as a report.
- The candidate makes the user want to continue reading by creating and
  resolving useful information gaps, tensions, contrasts, or insight paths.
- The candidate preserves concrete source-backed facts, caveats, distinctions,
  citations, examples, and unresolved tensions.
- The candidate does not show the shortening or omission regression seen in
  earlier rewrite-oriented experiments.
- The candidate does not introduce custom plan fields, report-visible prompt
  terms, source-by-source inventory prose, or repeated meta-openers.
- Operational completion remains comparable to the baseline.

Failure should leave the product default unchanged. A mixed result may still
produce a narrower follow-up issue if one part of the contract is useful and the
rest is too heavy.

## Public Artifact Policy

Commit only this protocol, small aggregate metrics, and a redacted conclusion.
Keep raw run directories, generated reports, prompt packets, judging packets,
databases, logs, and copied source bundles under the local experiment archive
described in `plasma/docs/artifact-archive.md`.
