# Report Edited Reading Voice Experiment

## Status

Completed as an issue #179 follow-up. Experiments 34 and 35 showed a useful but
incomplete path: `curiosity-natural-voice` made some reports sound less
mechanical, while `curiosity-tight-voice` controlled length but sometimes made
the result feel compressed rather than better edited.

This experiment tests a different correction. The new candidate does not make
compactness the primary goal. It asks the report writer and final editor to
turn research into edited reading material: concrete openings, consistent
heading language, fewer self-framing sentences, fewer "자료는..." style source
frames, and paragraph rhythm that preserves useful detail. No product default
has changed.

## Question

> Can Plasma keep the natural-voice gains while reducing report self-framing,
> title-language drift, mechanical paragraph endings, and over-compression?

## Failure Patterns Carried Forward

The experiment 35 direct read identified these target failures:

- self-framing sentences such as "이 보고서는..." or "이 글은..." when the
  sentence could begin with the subject;
- repeated "자료는..." or equivalent source-frame language that makes the
  report feel like a source inventory;
- English source titles leaking into Korean report headings without a reason;
- repeated section previews, caveats, and tidy mini-conclusions;
- compact output that drops reading texture instead of becoming more edited.

## Arms

| Arm | Guidance profile | Meaning |
|---|---|---|
| Baseline | `narrative-contract` | Current product writing baseline, kept because the shared harness analyzes candidates against a baseline. |
| Voice candidate | `curiosity-natural-voice` | Experiment 34 candidate that reduced visible signposting but expanded some reports. |
| Tight candidate | `curiosity-tight-voice` | Experiment 35 candidate that controlled length but produced a mixed reading result. |
| Edited candidate | `edited-reading-voice` | New candidate that keeps curiosity and natural voice while focusing on edited article-like surface quality. |

The edited candidate still uses the existing report plan schema and existing
long-form `section_fanout` assembly path. It does not add UI choices, database
state, report artifact types, citation behavior, postprocessors, or custom plan
fields.

The small run makes `edited-reading-voice` the strongest current follow-up
candidate, but not a product-default adoption result. It kept more detail than
the tight candidate, avoided the natural candidate's length expansion, used
consistent Korean titles, and removed direct "이 보고서는..." framing in the
three samples. It still used section-navigation phrases such as "이 부는",
"이 절에서", and "다음 질문은" often enough that the prose can sound like it
is explaining its outline. A larger run should test that remaining problem
before productization. No product default changed.

## Runner

Use:

```bash
cd plasma
python3 scripts/experiments/report_edited_reading_voice_experiment.py --action prepare --limit 3
python3 scripts/experiments/report_edited_reading_voice_experiment.py --action run --limit 3 --workers 2
python3 scripts/experiments/report_edited_reading_voice_experiment.py --action analyze --limit 3
python3 scripts/experiments/report_edited_reading_voice_experiment.py --action packets --limit 3
```

With `--limit 3`, the runner creates three topics across four long-form arms,
or 12 report generations before packet preparation.

## Small Run Result

The completed run produced three paired topics, 12 successful long-form report
generations, and nine blinded comparison packets. No arm failed. Every arm used
the existing `section_fanout` path through parallel Section drafting, Part
assembly, and final narrative editing.

| Arm | Average words | Average word ratio vs. baseline | Ratio range | Average section ratio vs. baseline | Average wall seconds |
|---|---:|---:|---:|---:|---:|
| `curiosity-natural-voice` | 5,123 | 1.090 | 0.971-1.212 | 1.037 | 655.7 |
| `curiosity-tight-voice` | 3,364 | 0.705 | 0.618-0.774 | 0.805 | 470.2 |
| `edited-reading-voice` | 4,048 | 0.854 | 0.799-0.934 | 0.862 | 554.4 |

The edited candidate averaged about 79% of the natural candidate's word count,
while the tight candidate averaged about 66%. Direct reading showed that this
middle position was not only a length compromise. All three edited reports used
Korean titles where the natural and tight candidates retained English titles,
and none opened with the direct "이 보고서는..." frame. Repeated "자료는..."
source framing also fell substantially relative to the natural candidate.

The remaining weakness is structural narration. The edited reports still
sometimes announced what a Part or Section would do, previewed the next
question, or restated the source boundary after it was already clear. The
candidate therefore advances as the leading prompt direction for a broader
sample, not as a finished default. The next test should reduce outline
explanation without returning to the tight candidate's loss of reading texture.

## Reading Focus

Judge the edited candidate primarily against `curiosity-natural-voice` and
`curiosity-tight-voice`.

Prefer the edited candidate only if it:

- opens on the subject rather than on report mechanics;
- keeps headings in a deliberate language and depth pattern;
- reduces "자료는..." and "이 보고서는..." style framing without hiding source
  boundaries;
- keeps concrete details, examples, numbers, mechanisms, caveats, and
  unresolved tensions where they make the reading stronger;
- feels more like edited Korean prose, not merely shorter text.

Reject or revise the edited candidate if it hides uncertainty, becomes casual,
drops useful detail, or only swaps one formulaic rhythm for another.

## Public Artifact Policy

Commit only this protocol, small aggregate metrics, and redacted conclusions.
Keep generated reports, prompt packets, judging packets, databases, logs, and
copied source bundles under the local experiment archive described in
`plasma/docs/artifact-archive.md`.
