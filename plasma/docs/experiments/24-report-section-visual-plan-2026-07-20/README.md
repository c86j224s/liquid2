# Report Section Visual Plan Experiment - 2026-07-20

This experiment follows the section-writing work in experiment 22 and the
visual-aid planning work in experiment 23.

The user decision from experiment 23 was to fold normal "basic writing" into the
`visual-plan` default. This experiment does **not** re-test that default. It
asks a narrower question: whether the long-form-only section writing options
should also receive the same sparse visual-aid planning guidance.

## Status

Completed full 24-topic pass. The follow-up product decision is split by arm:

- `section_brief_visual_plan` replaces the previous `section-brief` Web UI
  option. It preserved length at roughly the same scale as `section_brief` while
  adding useful tables and occasional Mermaid diagrams.
- `section_brief_cluster_memory_visual_plan` is not a clear default replacement
  for `section_brief_cluster_memory`. It reliably increased visual aids, but it
  also changed the "rich coverage" option's character more often by shortening
  some reports or increasing table density in others. The user chose to expose
  it as a separate rich long-form option for product testing rather than folding
  it into every rich report.

## Scope

The experiment uses the existing product-shaped long-form report path:

- isolated Plasma databases per run;
- local source snapshots attached through the product source path;
- the browser/API long-form report endpoint;
- Codex report agents with MCP source-reading tools;
- `section_fanout` as the default long-form execution strategy;
- no report schema change and no new durable product artifact type.

Raw reports, ledgers, prompt traces, judging packets, and local databases remain
outside the repository under:

`research-artifacts/liquid2/plasma/experiments/24-report-section-visual-plan-2026-07-20/`

## Arms

| Arm | `generation_guidance_profile` | Meaning |
| --- | --- | --- |
| `section_brief` | `section-brief` | Existing long-form option: each Section purpose acts as a light prose writing brief. |
| `section_brief_visual_plan` | `section-brief-visual-plan` | Candidate: the existing section brief plus visual-aid intent during planning and writing. |
| `section_brief_cluster_memory` | `section-brief-cluster-memory` | Existing long-form option: section brief plus source-backed cluster memory. |
| `section_brief_cluster_memory_visual_plan` | `section-brief-cluster-memory-visual-plan` | Candidate: the existing cluster-memory brief plus visual-aid intent during planning and writing. |

The two candidate profiles are accepted only for long-form reports. After the
experiment, the Web UI sends `section-brief-visual-plan` for the `섹션 중심`
choice and sends `section-brief-cluster-memory-visual-plan` for the richer
`섹션 중심 + 풍부하게` choice. The older non-visual profile values remain
accepted for compatibility with API callers and older report metadata.

## Comparison

The experiment compares candidates against their matching current option:

| Pair | Baseline | Candidate |
| --- | --- | --- |
| Focused section writing | `section_brief` | `section_brief_visual_plan` |
| Rich coverage writing | `section_brief_cluster_memory` | `section_brief_cluster_memory_visual_plan` |

## Decision Criteria

The candidate should not be productized merely because it increases visual count.
It needs to preserve or improve the report as a whole.

Evaluate:

- whether the section stays centered;
- whether prose still reads as coherent Korean long-form writing;
- whether visual aids help understanding rather than decorate the report;
- whether tables or Mermaid diagrams supplement rather than replace explanation;
- whether concrete source-backed detail and caveats remain visible;
- whether length growth remains readable rather than padded.

## Result

The full pass ran 96 report generations across 24 topics and four arms. It
completed 95 runs and had one terminal failure in the
`section_brief_cluster_memory` baseline arm. That failure removes one rich-arm
pair from paired comparison, but it does not affect the focused-arm comparison.

| Candidate | Completed pairs | Median word ratio vs baseline | Median visual delta | Mermaid validation warnings |
| --- | ---: | ---: | ---: | ---: |
| `section_brief_visual_plan` | 24 | 1.029 | +4 | 0 |
| `section_brief_cluster_memory_visual_plan` | 23 | 0.929 | +5 | 0 |

Interpretation:

- `section_brief_visual_plan` gave a clean signal. It increased visual aids
  without materially inflating report length. Direct reading found that tables
  and Mermaid diagrams often clarified comparison, sequence, dependency, and
  trade-off structures instead of merely decorating the prose.
- `section_brief_cluster_memory_visual_plan` gave a mixed signal. The visual
  aids were present and usually valid, but the candidate shortened 16 of 23
  paired reports and sometimes made the already rich writing mode feel more
  compressed or table-heavy. This does not prove harm, but it is not strong
  enough to silently replace the existing rich-coverage option.

Product decision:

- Replace the focused `section-brief` Web UI option with
  `section-brief-visual-plan`.
- Keep the rich cluster-memory path as an explicit user choice rather than a
  silent default for all reports; expose the visual variant as
  `section-brief-cluster-memory-visual-plan` so it can be tested in real use.

## Runner

The public runner is:

`plasma/scripts/experiments/report_section_visual_plan_experiment.py`

Prepare the isolated binary and fixture lock:

```bash
python3 plasma/scripts/experiments/report_section_visual_plan_experiment.py --action prepare
```

Smoke one topic with two workers:

```bash
python3 plasma/scripts/experiments/report_section_visual_plan_experiment.py --action run --limit 1 --workers 2
python3 plasma/scripts/experiments/report_section_visual_plan_experiment.py --action analyze
python3 plasma/scripts/experiments/report_section_visual_plan_experiment.py --action packets
```

If smoke passes, run a statistical pass with diverse topics:

```bash
python3 plasma/scripts/experiments/report_section_visual_plan_experiment.py --action run --limit 24 --workers 4
python3 plasma/scripts/experiments/report_section_visual_plan_experiment.py --action analyze
python3 plasma/scripts/experiments/report_section_visual_plan_experiment.py --action packets
```

Use fewer workers if local provider or machine stability becomes the limiting
factor.
