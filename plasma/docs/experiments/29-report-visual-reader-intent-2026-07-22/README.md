# Report Visual Reader-Intent Experiment - 2026-07-22

This experiment follows experiments 27 and 28 on issue 174.

Experiment 28 showed that a broad "prefer compact visual aids" instruction can
help structure-heavy reports, but it also exposed a failure mode. In a numeric
dashboard fixture, the candidate used a flowchart to explain interpretation
boundaries even though the reader needed source-near charts, timelines, and
tables for the actual values.

This follow-up tests a narrower intent:

> Use a visual aid when it helps the reader understand the section's central
> source-backed material better than prose alone.

The candidate is deliberately not phrased as "use these visual types." It asks
the report agent to identify the reader's task first, then decide whether a
visual makes that task easier.

## Status

Completed. Product default has not been changed.

## Scope

The experiment uses the same product-shaped report path as experiments 27 and
28:

- isolated Plasma database per run;
- local source snapshot attached through the product source path;
- the browser/API report endpoint;
- Codex report agents with MCP source-reading and Mermaid validation tools;
- `generation_guidance_profile` as the only report-writing selector;
- no prompt-only source dump;
- no report schema, renderer, MCP, durable artifact, UI, or product default
  change.

Raw reports, ledgers, prompt traces, judging packets, and local databases must
remain outside the repository under:

`research-artifacts/liquid2/plasma/experiments/29-report-visual-reader-intent-2026-07-22/`

## Product Question

The product question is whether the report writer can be guided toward visual
aids by reader value rather than by visual-type pressure.

The candidate should:

- identify what the reader must understand in each section;
- use visuals when they make the section's central source material easier to
  inspect;
- keep numeric and ordered material close to source values, timing, or
  comparison axes;
- avoid replacing source-near visuals with meta-level diagrams about caution,
  methodology, or unsupported inference;
- keep caveats and boundaries in prose unless they are the actual subject of
  the section.

## Arms

| Arm | `generation_guidance_profile` | Meaning |
| --- | --- | --- |
| `visual_plan` | `visual-plan` | Current product baseline: sparse visual-aid planning plus visual type selection. |
| `visual_reader_intent` | `visual-reader-intent` | Candidate: current visual plan plus reader-task guidance that favors visuals only when they improve inspection of the section's central source material. |

## Fixture Families

The runner reuses the six synthetic fixture families from experiments 27 and
28:

| Fixture | Structure under test |
| --- | --- |
| `fictional-equity-dashboard` | ordered values and event markers |
| `industry-capacity-statistics` | regional comparison and bottleneck timing |
| `agent-benchmark-matrix` | multi-axis trade-offs |
| `architecture-dependency-graph` | service dependencies and blast radius |
| `protocol-lifecycle` | actors, handoffs, state transitions |
| `scenario-risk-portfolio` | scenario bands and uncertainty |

## Decision Criteria

The candidate is useful only if it improves the report as a report.

Evaluate:

- whether the chosen visuals reflect the reader's central task in each section;
- whether numeric or ordered fixtures preserve source-near charts, timelines, or
  compact tables instead of meta-level explanation diagrams;
- whether structure-heavy fixtures still benefit from compact diagrams;
- whether Mermaid diagrams validate before final Markdown submission;
- whether the report remains readable and source-grounded;
- whether visual aids reduce reader effort instead of adding decorative blocks.

## Result

The candidate was not adopted as a product default.

The paired run produced 24 records and 11 completed pairs. One candidate run,
`protocol-lifecycle / planned / visual_reader_intent`, failed during the
`report_plan` stage before a report was written. The failure is counted as an
operational reliability limit for this experiment, but the completed pairs are
still useful for judging visual-selection behavior.

Aggregate automatic metrics:

| Metric | `visual_plan` | `visual_reader_intent` |
| --- | ---: | ---: |
| Completed runs | 12 | 11 |
| Median visual aids | 6.0 | 6.0 |
| Median visual alignment score | 2.0 | 2.0 |
| Unvalidated Mermaid signals | 0 | 0 |

Paired candidate deltas:

| Metric | Value |
| --- | ---: |
| Completed pairs | 11 |
| Median visual-aid delta | 0.0 |
| One-sided sign p-value for visual increase | 0.6367 |
| Median alignment-score delta | 0.0 |
| One-sided sign p-value for alignment increase | 0.75 |
| Median word ratio over baseline | 0.898 |

Manual reading found a mixed result.

Positive case:

- On `fictional-equity-dashboard / long_form`, the candidate avoided the earlier
  meta-level flowchart failure and used source-near aids: compact value tables,
  a `xychart-beta` for close versus 5-day average close, and a Mermaid timeline
  for event markers. Alignment improved from 2 to 3 while the report became
  shorter.

Neutral or negative cases:

- On architecture-dependency material, both arms produced useful dependency
  flowcharts. The candidate did not clearly improve the report.
- On agent benchmark material, the candidate leaned almost entirely on tables.
  That was readable, but it did not add the source-backed chart or quadrant-like
  comparison that the fixture could plausibly support.
- On `scenario-risk-portfolio / planned`, the baseline used a timeline for
  observation anchors, while the candidate reduced the same material to tables.
  Alignment dropped from 2 to 1.

Interpretation:

The reader-intent wording helped suppress an unhelpful meta-diagram in the
numeric dashboard case, but it was too conservative to improve visual use
overall. It behaves more like a restraint layer than an adoption layer. Future
work should not replace `visual_plan` with this candidate. A better successor
would combine the existing visual preference with a narrow rule that keeps
cautions, methodology, and inference boundaries in prose when they would
displace a more useful source-near visual.

## Runner

The public runner is:

`plasma/scripts/experiments/report_visual_reader_intent_experiment.py`

Prepare the isolated binary and fixture lock:

```bash
python3 plasma/scripts/experiments/report_visual_reader_intent_experiment.py --action prepare
```

Smoke the numeric dashboard case first because it exposed the previous failure:

```bash
python3 plasma/scripts/experiments/report_visual_reader_intent_experiment.py --action run --topics fictional-equity-dashboard --modes long_form --workers 1
python3 plasma/scripts/experiments/report_visual_reader_intent_experiment.py --action analyze
python3 plasma/scripts/experiments/report_visual_reader_intent_experiment.py --action packets
```

If smoke passes, run the paired comparison:

```bash
python3 plasma/scripts/experiments/report_visual_reader_intent_experiment.py --action run --limit 6 --workers 2
python3 plasma/scripts/experiments/report_visual_reader_intent_experiment.py --action analyze
python3 plasma/scripts/experiments/report_visual_reader_intent_experiment.py --action packets
```

Use fewer workers if local provider stability becomes the limiting factor.
