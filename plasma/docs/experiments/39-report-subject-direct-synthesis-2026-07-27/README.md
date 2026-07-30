# Report Subject-Direct Synthesis Experiment

## Status

Wave 1 completed for issue #189. The candidate reduced source-as-narrator prose,
but its length and repetition varied too much by topic, so it was not promoted
to the default profile.

## Question

Can Plasma keep the current long-form report density and citation discipline
while writing the subject directly instead of narrating what source material
says?

## Arms

| Arm | Guidance profile | Meaning |
|---|---|---|
| Current default | `part-connective-economy-voice` | Current long-form default lineage. |
| Rich control | `section-brief-cluster-memory-narrative-contract` | Separate rich-control arm with Section brief cluster memory. |
| Subject-direct candidate | `part-connective-subject-direct-synthesis-voice` | Current default lineage plus planning and Section-only subject-direct synthesis guidance. |

The candidate inherits the long-form narrative contract, curiosity-led
explanation, natural voice, edited reading voice, Section direct-writing
guidance, and Part connective-economy guidance. It does not add Section brief
cluster-memory behavior, Part assembly behavior, final-edit behavior, schema
fields, storage changes, API changes, word limits, Section limits, or acceptance
keyword gates.

## Runner

Use:

```bash
cd plasma
python3 scripts/experiments/report_subject_direct_synthesis_experiment.py --action prepare --limit 6
python3 scripts/experiments/report_subject_direct_synthesis_experiment.py --action run --limit 6 --workers 2
python3 scripts/experiments/report_subject_direct_synthesis_experiment.py --action analyze --limit 6
python3 scripts/experiments/report_subject_direct_synthesis_experiment.py --action packets --limit 6
```

The archive root is
`~/research-artifacts/liquid2/plasma/experiments/39-report-subject-direct-synthesis-2026-07-27`.
Raw reports, databases, logs, judging packets, and copied sources stay there.

## Corpus Plan

Wave 1 uses the six experiment-38 fixtures as causal controls. Those fixtures
are English single-source material, so they can isolate whether the prompt
change affects the current default lineage, but they do not satisfy issue #189's
full acceptance corpus.

Before any default-promotion decision, Wave 2 must add local archived fixtures
for Korean original sources, multi-source conflict, market research, and
source-sparse material. Those fixtures must remain in the local archive unless a
small redacted summary is needed for the public decision record.

## Measurement

The analysis writes ordinary paired report metrics and
`analysis/stage-aggregate.json`. Stage metrics cover Section drafts, assembled
Parts, and final artifacts:

- source-as-narrator advisory locator counts;
- citation counts, including external Markdown inline links, parenthetical citation
  shapes, footnote references, and footnote definition lines;
- characters, words, headings, Part/Section counts, and Section-to-final size
  ratios;
- Part connective character deltas and final character deltas.

Source-as-narrator detection is advisory locator output only. It can help a
human reader find passages to inspect, but it is never a pass/fail acceptance
rule, keyword policy, or authoritative classifier. The locator intentionally
includes named-source narration shapes such as product documents, books, guides,
and official documentation explaining a claim.

## Human Acceptance

Reject the candidate if it loses any source-backed mechanism, example, number,
caveat, comparison, unresolved tension, necessary source identity, or citation
versus the relevant control. Source identity remains necessary when version
differences, inter-source disagreement, source-flagged uncertainty or
instability, measurement method, or authority scope materially changes the
claim.

Advance only if the candidate preserves information density and citation
discipline while reducing source-as-narrator prose in human reading.

## Wave 1 Result

All 18 planned runs completed: six topics across the current default, rich
control, and subject-direct candidate. The candidate produced 24,257 final
words versus 25,712 for the current default, while the rich control produced
32,162. The advisory source-as-narrator count fell from 37 to 21 and citation
locator occurrences rose from 13 to 44.

The aggregate hides material topic variance:

| Topic | Candidate words / current-default words | Human reading |
|---|---:|---|
| Accessibility | 0.74 | Clearer and shorter, but still repeats evidence limits. |
| Climate adaptation | 1.20 | Longer, more sectioned, and repetitive. |
| Consumer finance | 1.44 | More direct, but substantially more verbose. |
| Labor statistics | 0.95 | Similar length; the definition and formula repeat across sections. |
| Vaccination | 0.90 | More direct and easier to enter while retaining inspected facts. |
| Public procurement | 0.67 | More focused and less prescriptive than the current default. |

The single-source controls contained 74-480 source words, while candidate
reports expanded them by 7.8-35.5 times. Human reading found that the candidate
improved the writing stance in several topics but did not reliably control
evidence-relative length. The profile remains available for Wave 2 experiments;
the long-form default remains `part-connective-economy-voice`. Issue #189 keeps
tracking density and repetition before any default-promotion decision.

## Public Artifact Policy

Commit only this protocol and small redacted decisions. Keep raw runs and bulky
reproducibility artifacts under the local archive described in
`plasma/docs/artifact-archive.md`.
