# H5 Full-Report Result

## Purpose

H2 had a strong changed-block readability signal, but that test did not prove
that a whole report reads better end to end. H5 tested a narrower follow-up:
take the H2 report as the baseline, run one full-report tone pass, and compare
H5 against H2.

The purpose was not to redesign the report or change the content model. The
test asked only whether full-report tone smoothing can improve the final reading
experience while preserving the Markdown structure and report register.

## Inputs

Raw inputs and generated candidates are stored outside Git:

```text
~/research-artifacts/liquid2/plasma/experiments/08-report-humanize-2026-07-04/
```

H5 candidates:

```text
candidates/
  s1-h5-full.md
  s2-h5-full.md
analysis/
  s1-h5-full-structure.json
  s2-h5-full-structure.json
  h5_full_report_blind_cases.jsonl
  h5_full_report_blind_cases_manifest.json
  h5_full_report_blind_judge_j1.jsonl
  h5_full_report_blind_judge_j2.jsonl
  h5_full_report_blind_judge_j3.jsonl
  h5_full_report_blind_judge_j4.jsonl
  h5_full_report_blind_judge_j5.jsonl
  h5_full_report_blind_preference_summary.json
  h5_full_report_blind_preference_summary.md
```

## Structure Gate

Both H5 candidates passed the structure check.

| sample | H5 candidate | changed blocks vs H2 | structure gate |
|---|---|---:|---|
| `s1-short-ai-report` | `s1-h5-full.md` | 7 | pass |
| `s2-long-gemma4-report` | `s2-h5-full.md` | 13 | pass |

One generated `s2` candidate had two small style issues after generation:
a repeated "먼저" and a conversational "살펴보자". These were manually fixed
before the final structure check and blind preference run.

## Blind Preference Test

The blind test compared H5 against H2 on two surfaces:

- full report: the whole `s1` and `s2` reports;
- section: sections containing H5 changes, used as supporting local evidence.

Five independent judge passes evaluated 18 cases:

- 2 full-report cases;
- 16 changed-section cases.

Result:

| scope | H5 preferred | H2 preferred | ties | H5 share among non-ties | one-sided sign-test p |
|---|---:|---:|---:|---:|---:|
| all decisions | 59 | 8 | 23 | 0.881 | `5.08e-11` |
| full-report decisions | 9 | 1 | 0 | 0.900 | `0.0107` |
| changed-section decisions | 50 | 7 | 23 | 0.877 | `2.12e-09` |

By sample:

| sample | H5 preferred | H2 preferred | ties | H5 share among non-ties |
|---|---:|---:|---:|---:|
| `s1-short-ai-report` | 31 | 1 | 3 | 0.969 |
| `s2-long-gemma4-report` | 28 | 7 | 20 | 0.800 |

Case majorities:

- overall: H5 12, H2 1, tie 4, split 1;
- full-report cases: H5 2;
- changed-section cases: H5 10, H2 1, tie 4, split 1.

## Interpretation

H5 is the first variant in this experiment family that directly addresses the
full-report reading-experience gap in the H2 result. The combined blind result
strongly favors H5 over H2.

The result should still be read with two limits:

- there are only two full-report cases, so the full-report-only result is
  directional despite being 9 to 1 across five judges;
- section-level cases are useful supporting evidence, but they are not a
  complete substitute for human review of full reports.

Within those limits, H5 is the best product follow-up candidate so far. The
next product step should preserve the H2/H5 fidelity gates and treat H5 as a
post-report tone pass, not as a report planner, source selector, or HTML design
tool.

## Decision

Current decision: `h5_full_report_signal_confirmed`.

H3 remains rejected. H4 selector remains rejected. H5 is accepted as the next
candidate to carry into product-design discussion, with manual full-report
review still required before runtime adoption.
