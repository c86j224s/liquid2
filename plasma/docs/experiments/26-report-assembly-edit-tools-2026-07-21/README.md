# Report Assembly Edit-Tools Experiment - 2026-07-21

This experiment follows the long-form fanout, section-brief, visual-plan, and
final-report MCP work. Those experiments improved how Plasma plans and writes
long-form reports, but the final assembly stage still has one fragile spot:
part assembly agents return JSON that contains the Part intro, inter-section
transitions, and Part closing in one response.

This experiment asks a narrower question:

> If a Part assembly agent edits only the connective tissue through MCP tools,
> does the final long-form report keep or improve flow without changing section
> substance?

## Status

Completed as a fixed-plan quality experiment. The first product-path smoke
proved that the candidate path ran, but it let each arm create its own report
plan. That mixed the Part assembly change with different Part/Section shapes.
The follow-up therefore froze one product-created plan per topic and resumed
both arms from the same cloned plan state.

The experiment implementation added `part-assembly-edit-tools` as a
long-form-only profile and a product-shaped runner. During the experiment the
profile was hidden from the Web UI. The productization follow-up exposes it as
a browser long-form writing option while keeping the default report path
unchanged.

Smoke result:

| Topic | Baseline status | Candidate status | Candidate MCP submissions | Notes |
| --- | --- | --- | ---: | --- |
| `public-health-guidance-a` | completed | completed | 3 of 3 Parts | Product path completed in both arms. Candidate produced all expected Part assembly submission events. |

Observed smoke metrics:

| Metric | Value |
| --- | ---: |
| Candidate word ratio over baseline | 0.745 |
| Candidate wall-time ratio over baseline | 1.163 |
| Candidate preservation-ratio delta | +0.134 |

These smoke metrics confirm only that the candidate path runs and leaves the
expected ledger evidence. They are not enough to decide report quality because
the baseline and candidate produced different plan shapes in that smoke.

The fixed-plan run keeps source snapshots, Part count, Section count, Section
titles, Section purposes, and target refs identical before the Part assembly
behavior diverges. It completed 12 paired topics.

Fixed-plan result:

| Metric | Result |
| --- | ---: |
| Completed paired topics | 12 |
| Candidate MCP assembly completion | 12 / 12 |
| Matching plan signature | 12 / 12 |
| Median candidate word ratio over baseline | 1.017 |
| Median preservation-ratio delta | +0.063 |
| Median wall-time ratio over baseline | 1.582 |
| Median total-token ratio over baseline | 1.267 |
| Median uncached-input-token ratio over baseline | 1.034 |
| Word-ratio sign-test p, one-sided | 0.019 |
| Preservation-delta sign-test p, one-sided | 0.019 |

Direct reading favored the candidate. The difference was clearest in reports
where the baseline left heading, emphasis, or conclusion structure feeling
mechanically assembled. The candidate usually gave each Part a more useful
opening frame, made Part-to-Part movement easier to follow, and kept caveats
attached to the relevant argument. The advantage was smaller in topics where
the baseline was already good, so the result should be read as a report-flow
quality improvement, not as a universal prose rewrite.

The cost is also clear. The candidate adds MCP Part assembly work and therefore
increases wall-clock time and provider usage. In this run the median wall-time
ratio was about 1.58x and total-token ratio about 1.27x. Because quality was the
primary criterion for Issue #152, the candidate is considered a meaningful
improvement candidate, with the cost trade-off explicitly noted.

## Scope

The experiment uses the existing product-shaped long-form report path:

- isolated Plasma database per run;
- local source snapshots attached through the normal source path;
- the browser/API long-form report endpoint;
- Codex report agents with MCP source-reading tools;
- `section_fanout` as the default long-form execution strategy;
- `generation_guidance_profile` as the only experiment selector;
- raw reports, ledgers, prompt traces, judging packets, local databases, and
  provider state kept outside Git.

Raw material belongs under the local experiment archive:

`research-artifacts/liquid2/plasma/experiments/26-report-assembly-edit-tools-2026-07-21/`

## Non-Goals

This experiment deliberately does not:

- edit or rewrite section bodies;
- add a full-report patch surface;
- change the report plan schema;
- change source-reading behavior;
- expose a new Web UI option;
- make `part-assembly-edit-tools` a product default;
- use prompt-only source dumps instead of product source attachment.

The candidate must be judged as a report-quality experiment, not as a new
recovery workflow or a general editing API.

The fixed-plan runner also excludes volatile Codex provider temp directories
when cloning seed state. Those files are runtime leftovers, not product state,
and copying them can fail if the provider removes a temp file during clone.

## Arms

| Arm | `generation_guidance_profile` | Meaning |
| --- | --- | --- |
| `visual_plan` | `visual-plan` | Current long-form baseline with sparse visual-aid planning and existing JSON-return Part assembly. |
| `part_assembly_edit_tools` | `part-assembly-edit-tools` | Candidate: keep the same visual-plan planning/writing guidance, but require Part assembly agents to submit only connective tissue through MCP tools. |

The candidate keeps planning and section writing aligned with `visual-plan` so
the comparison focuses on Part assembly mechanics.

## Candidate MCP Boundary

The candidate exposes four MCP tools only to the bound Part assembly agent
session:

| Tool | Purpose |
| --- | --- |
| `plasma.report.part_assembly.start` | Open an in-process connective draft for the current Part binding. |
| `plasma.report.part_assembly.read` | Inspect that draft in the same MCP process. |
| `plasma.report.part_assembly.patch` | Set only `intro`, `transition`, or `closing` Markdown. |
| `plasma.report.part_assembly.submit` | Append one `report.part_assembly.submitted` event for the bound Part. |

The submitted event stores connective tissue and binding metadata. It does not
store section bodies, and the product runner still mechanically combines the
immutable section drafts with the submitted connective tissue.

## Decision Criteria

The candidate is useful only if it improves or preserves the report as a whole.

Evaluate:

- whether every candidate Part has exactly one successful MCP assembly
  submission;
- whether section bodies remain the source of section substance;
- whether Part intros, transitions, and closings improve flow without smoothing
  away concrete evidence;
- whether Korean prose reads naturally across Part boundaries;
- whether caveats and source limits remain attached to the relevant argument;
- whether report length, preservation ratio, and terminal success do not regress
  enough to outweigh any flow gain.

Automatic metrics are operational signals. They cannot replace direct reading.

## Runner

The public runner is:

`plasma/scripts/experiments/report_part_assembly_tools_experiment.py`

Prepare the isolated binary and fixture lock:

```bash
python3 plasma/scripts/experiments/report_part_assembly_tools_experiment.py --action prepare
```

Smoke one topic with both arms:

```bash
python3 plasma/scripts/experiments/report_part_assembly_tools_experiment.py --action run --limit 1 --workers 2
python3 plasma/scripts/experiments/report_part_assembly_tools_experiment.py --action analyze
python3 plasma/scripts/experiments/report_part_assembly_tools_experiment.py --action packets
```

If smoke passes, run a broader pass:

```bash
python3 plasma/scripts/experiments/report_part_assembly_tools_experiment.py --action run --limit 24 --workers 4
python3 plasma/scripts/experiments/report_part_assembly_tools_experiment.py --action analyze
python3 plasma/scripts/experiments/report_part_assembly_tools_experiment.py --action packets
```

Use fewer workers if local provider or machine stability becomes the limiting
factor.

Run the controlled fixed-plan comparison:

```bash
python3 plasma/scripts/experiments/report_part_assembly_tools_experiment.py --action run --fixed-plan --limit 1 --workers 1
python3 plasma/scripts/experiments/report_part_assembly_tools_experiment.py --action analyze --fixed-plan
python3 plasma/scripts/experiments/report_part_assembly_tools_experiment.py --action packets --fixed-plan
```

Broaden only after the fixed-plan smoke confirms matching `plan_signature`
values for each completed pair.

The 12-pair run used:

```bash
python3 plasma/scripts/experiments/report_part_assembly_tools_experiment.py --action run --fixed-plan --limit 12 --workers 2 --timeout-seconds 7200
python3 plasma/scripts/experiments/report_part_assembly_tools_experiment.py --action analyze --fixed-plan
python3 plasma/scripts/experiments/report_part_assembly_tools_experiment.py --action packets --fixed-plan
```
