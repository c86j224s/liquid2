# Report Section-Direct Reading Experiment

## Status

Completed as an issue #179 follow-up to experiment 36. All 12 report runs
finished, producing six complete blinded pairs. The Section-only candidate met
the experiment's advancement rules, but it is not a product default.

This experiment changes only the Section writer. The plan writer, Part
assembler, final editor, report schema, citations, and long-form assembly path
remain the same as `edited-reading-voice`.

## Result

The candidate was preferred in four of six blinded comparisons: accessibility,
climate adaptation, labor statistics, and public-health guidance. The baseline
was preferred for consumer finance and public procurement. Preferences were
recorded before the private arm mapping was revealed.

The tracked document-position phrases also fell after normalizing for output
size. This matters because candidate length ranged from 0.92x to 1.32x of its
paired baseline.

| Artifact stage | Baseline markers per 10k chars | Candidate markers per 10k chars | Raw counts |
|---|---:|---:|---:|
| Section drafts | 2.55 | 1.01 | 25 -> 11 |
| Assembled Parts | 4.82 | 3.96 | 55 -> 50 |
| Final reports | 4.83 | 3.89 | 55 -> 49 |

Final-report marker density was lower in four of six topics and lower in
aggregate. The larger reduction at the Section stage shows that the instruction
worked where it was applied. Part assembly then reintroduced much of the
outline narration, so the final-report gain was smaller.

| Measure | Baseline | Candidate |
|---|---:|---:|
| Completed reports | 6 | 6 |
| Average words | 4,592 | 5,085 |
| Average wall time | 608.9 s | 607.8 s |

Direct reading found no systematic loss of citations, caveats, mechanisms, or
concrete detail. It also found no general brevity benefit: the candidate was
about 10.7% longer by average word count, with two shorter and four longer
reports.

## Decision

Advance `section-direct-reading-voice` as the next controlled experiment
baseline. Do not make it the product default yet. The result supports assigning
direct subject writing to the Section writer, but it does not show that a
Section-only change is sufficient for the assembled report. A later experiment
should isolate Part assembly without changing planning, Section generation, or
final editing in the same run.

## Stage Diagnosis

Experiment 36 preserved Section drafts, assembled Parts, and final reports for
all three edited candidate topics. Counting the same small set of document-
position markers at each stage produced this progression:

| Stage | Tracked marker count | Net change from previous stage |
|---|---:|---:|
| Section drafts | 21 | - |
| Assembled Parts | 29 | +8 |
| Final reports | 32 | +3 |

The count is a diagnostic signal, not a complete prose-quality score. It shows
that Part assembly and final editing can add outline narration, but most of the
remaining markers already exist in the immutable Section drafts. The first
controlled correction therefore belongs to the Section writer.

## Question

> Can a Section-only instruction reduce outline narration while preserving the
> concrete detail and reading texture of `edited-reading-voice`?

## Arms

| Arm | Guidance profile | Meaning |
|---|---|---|
| Baseline | `edited-reading-voice` | Experiment 36 candidate with curiosity-led, natural, and edited-reading guidance. |
| Candidate | `section-direct-reading-voice` | The same guidance plus a Section-only instruction to write the subject directly instead of explaining the Section's place in the outline. |

The candidate does not add a plan field or a later cleanup pass. Its policy
hash includes the Section-only instruction, but that instruction is injected
only into Section drafting prompts.

## Topics

The six selected topics broaden subject matter within the existing locked
fixture corpus:

1. vaccination and public communication;
2. credit-card costs and consumer decisions;
3. unemployment measurement and interpretation;
4. climate adaptation, governance, uncertainty, and equity;
5. Web Content Accessibility Guidelines and implementation priorities;
6. request-for-proposal design, incentives, and failure modes.

All six fixtures remain English Wikipedia single-source bundles. This run can
test the writing-stage correction across subject areas, but it cannot establish
that the result generalizes to Korean original sources, multi-source conflict,
market research, or source-sparse investigations.

## Runner

Use:

```bash
cd plasma
python3 scripts/experiments/report_section_direct_reading_experiment.py --action prepare --limit 6
python3 scripts/experiments/report_section_direct_reading_experiment.py --action run --limit 6 --workers 2
python3 scripts/experiments/report_section_direct_reading_experiment.py --action analyze --limit 6
python3 scripts/experiments/report_section_direct_reading_experiment.py --action packets --limit 6
```

With `--limit 6`, the runner creates six topics across two long-form arms, or
12 report generations before packet preparation.

## Reading Focus

Prefer the candidate only if direct reading shows that it:

- begins Sections with the claim, mechanism, scene, contrast, consequence, or
  evidence-backed question;
- reduces document-position narration in most topics and in aggregate;
- preserves examples, numbers, mechanisms, caveats, citations, and unresolved
  tensions;
- does not become shorter mainly by dropping explanation;
- remains preferable in at least four of six blinded topic comparisons.

Reject or revise the candidate if it merely swaps phrases, creates abrupt
Section boundaries, hides source limits, or loses useful reading texture.

## Public Artifact Policy

Commit only this protocol, small aggregate metrics, and redacted conclusions.
Keep generated reports, intermediate manuscripts, judging packets, databases,
logs, and copied source bundles under the local experiment archive described in
`plasma/docs/artifact-archive.md`.
