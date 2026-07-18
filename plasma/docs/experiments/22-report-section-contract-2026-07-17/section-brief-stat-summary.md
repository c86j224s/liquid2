# Section Brief Statistical Expansion - 2026-07-17

This note records the statistical-scale follow-up for the section-brief line of
the long-form section contract experiment.

The experiment stayed inside the existing product-shaped long-form path:

- isolated Plasma databases per run;
- local source snapshots attached through the product source path;
- long-form report generation through the product report endpoint;
- MCP source-reading tools available to the report agents;
- `section_fanout` execution strategy;
- no new report artifact type;
- no plan schema change.

Raw reports, ledgers, prompt traces, and judging packets remain in the local
archive:

`research-artifacts/liquid2/plasma/experiments/22-report-section-contract-2026-07-17-section-brief-stat/`

## Arms

| Arm | Profile | Purpose |
| --- | --- | --- |
| `baseline` | `g2` | Current long-form baseline guidance. |
| `section_brief` | `section-brief` | Use the existing section `purpose` as a light prose writing brief: reader movement, concrete details, tension/caveat, and adjacent-topic boundary. |
| `section_brief_cluster_memory` | `section-brief-cluster-memory` | Add source-backed cluster memory to the section brief so the writer keeps mechanisms, examples, numbers, caveats, comparisons, and omissions visible. |

## Run Shape

- Topics: 24
- Arms per topic: 3
- Total reports: 72
- Completed reports: 72
- Terminal failures: 0
- Model: `gpt-5.5`
- Effort: `medium`
- Execution strategy: `section_fanout`
- Workers: 4

## Quantitative Guardrails

The table uses paired comparisons against `baseline`.

| Metric | `section_brief` | `section_brief_cluster_memory` |
| --- | ---: | ---: |
| Mean word ratio vs baseline | 0.918 | 1.113 |
| Median word ratio vs baseline | 0.894 | 1.071 |
| Longer / shorter than baseline | 7 / 17 | 18 / 6 |
| Word-count sign-test p-value | 0.063915 | 0.022656 |
| Mean section ratio vs baseline | 0.979 | 1.057 |
| Median section ratio vs baseline | 0.919 | 1.000 |
| More / fewer / equal sections | 7 / 13 / 4 | 9 / 8 / 7 |
| Section-count sign-test p-value | 0.263176 | 1.000000 |
| Mean runtime ratio vs baseline | 1.041 | 1.110 |
| Median runtime ratio vs baseline | 1.041 | 1.100 |

The cluster-memory arm was also compared directly with `section_brief`.

| Metric | Result |
| --- | --- |
| Mean word ratio, cluster memory over section brief | 1.223 |
| Median word ratio, cluster memory over section brief | 1.187 |
| Longer / shorter than section brief | 21 / 3 |
| Word-count sign-test p-value | 0.000277 |

Interpretation:

- `section_brief` tends to shorten the report, but the paired word-count result
  does not cross the usual 0.05 threshold in this 24-topic run.
- `section_brief_cluster_memory` is statistically distinguishable from baseline
  on length. It makes reports longer in 18 of 24 topics.
- `section_brief_cluster_memory` is very clearly longer than `section_brief`.
  This confirms that the cluster-memory prompt changes behavior, but it does
  not by itself prove higher report quality.

## Direct Reading Assessment

Manual reading focused on complete reports and representative passages from
education policy, climate adaptation, public procurement, disaster
preparedness, consumer finance, open-source governance, and labor statistics.

Observed strengths of `section_brief`:

- Introductions often have a clearer local center.
- The first paragraphs more often state what the section is trying to help the
  reader understand.
- Repeated caveat frames such as "with only this material..." appear less
  frequently than in baseline.
- The prose usually feels more compact and less like a source inventory.

Observed risks of `section_brief`:

- Some reports lose density by compressing examples and quantitative details.
- In topics where baseline already had a strong flow, the improvement is small.
- The arm can make a section feel polished but slightly under-supplied.

Observed strengths of `section_brief_cluster_memory`:

- It preserves more source-backed clusters and concrete examples.
- It is better at keeping mechanisms, tensions, and limits visible across the
  report.
- In some cases, especially policy topics, it makes the report feel more
  complete than `section_brief`.

Observed risks of `section_brief_cluster_memory`:

- It often becomes longer without a proportional readability gain.
- The prose can over-explain adjacent ideas.
- It does not eliminate the existing heading-normalization issue; duplicated
  headings such as `Part 1. Part 1.` still appear in some outputs. This issue is
  shared across arms and should be treated separately from the prompt arm.

AI-ish phrase check:

- The problematic phrases "이 세션" and "공통적으로 확인" did not appear in any
  of the 72 reports.
- Evidence-boundary phrases still appear. `section_brief` reduces them relative
  to baseline, while `section_brief_cluster_memory` returns to roughly baseline
  frequency.

## Decision

Productize both candidate arms as explicit long-form report writing options,
while keeping the existing `g2` guidance as the default.

`section_brief` is the safer focused-writing option because it improves local
section focus in manual reading without changing schema or workflow shape.
However, this run does not prove a statistically significant universal quality
preference, so it should be presented as an option rather than as a proven
replacement for every report.

`section_brief_cluster_memory` should not become the default. It provides a real
behavioral change, but the confirmed change is more length and more coverage
pressure, not a statistically proven quality gain. It is still useful as an
explicit "richer coverage" option for cases where the user wants more source
clusters kept visible.

Conservative interpretation:

- Keep default long-form generation on `g2`.
- Expose `section_brief` for cleaner section focus and less source-inventory
  prose.
- Expose `section_brief_cluster_memory` for richer, more coverage-preserving
  long-form reports.
- Do not describe either option as a proven universal quality upgrade.

## Follow-Up

Recommended next steps:

1. If productizing, keep the change narrow: add `section_brief` and
   `section_brief_cluster_memory` as long-form options only. Do not add cluster
   memory to the default path.
2. Track report length and section count after productization, because mild
   shortening is the main measurable risk.
3. Handle heading duplication as a separate report assembly/rendering issue.
4. If stronger evidence is needed, add a blinded quality-judging step to the
   existing packets instead of expanding the report-generation harness again.
