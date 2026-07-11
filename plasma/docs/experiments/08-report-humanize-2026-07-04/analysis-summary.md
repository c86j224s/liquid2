# Analysis Summary

## Run Summary

H1 was tested on two committed-metadata samples with raw material kept in the
local archive:

| sample | candidate | generation tokens | structure gate v2 | review outcome |
|---|---|---:|---|---|
| `s1-short-ai-report` | `s1-h1.md` | 53,541 | fail | tone improved, but final paragraph was dropped |
| `s3-section-gemma4-part03-section02` | `s3-h1.md` | 37,019 | pass | tone improved modestly and content was preserved |

A separate review pass used 34,400 tokens and recommended
`continue_experiment`, not productization.

H2 was then run on the failing `s1` full-report sample. The raw candidate remains
outside Git as `candidates/s1-h2.md`.

| sample | candidate | generation tokens | structure gate v2 | review outcome |
|---|---|---:|---|---|
| `s1-short-ai-report` | `s1-h2.md` | not captured | pass | restored the dropped final paragraph and kept the conservative tone edits |

The same H2 profile was then run on the long-form Gemma 4 report sample. The raw
candidate remains outside Git as `candidates/s2-h2.md`.

| sample | candidate | generation tokens | structure gate v2 | review outcome |
|---|---|---:|---|---|
| `s2-long-gemma4-report` | `s2-h2.md` | not captured | pass | preserved the long report structure; host audit found no heading, table, code fence, source-line, inline-code, number-token, or selected technical-token drift |

## Blind Preference Test

The H2 candidates were compared against their originals with a blind A/B
preference test. The test used only changed text blocks, so unchanged blocks did
not dilute the result.

Method:

- 76 matched changed text blocks were extracted from `s1`, `s2`, and `s3`.
- Each case randomized whether the original or H2 text appeared as A or B.
- Three independent judge passes evaluated the same cases.
- Judges selected A, B, or tie for Korean report readability while preserving
  report tone, technical precision, uncertainty, and source safety.
- H2 preference was decoded only after the judgments were written.
- Statistical test: exact one-sided sign test over non-tie decisions.

Result:

| scope | H2 preferred | original preferred | ties | H2 share among non-ties | one-sided sign-test p |
|---|---:|---:|---:|---:|---:|
| all decisions | 181 | 39 | 8 | 0.823 | `2.32e-23` |
| case majority | 60 | 13 | 3 | 0.822 | `1.15e-08` |
| `s1-short-ai-report` | 38 | 11 | 2 | 0.776 | `7.10e-05` |
| `s2-long-gemma4-report` | 114 | 25 | 5 | 0.820 | `4.45e-15` |
| `s3-section-gemma4-part03-section02` | 29 | 3 | 1 | 0.906 | `1.28e-06` |

Interpretation: within this experiment set, H2 shows a statistically significant
readability preference over the original Plasma report wording.

Limits:

- The sample set has only three source artifacts.
- The judges are model-based, not human readers.
- The test measures local changed-block preference, not full-report reading
  experience.
- A product decision still needs manual review of representative full reports.

## What Improved

H1 reduced stiff Korean report phrasing without making the samples casual. The
stronger examples were local sentence edits:

- "수행되어야 한다" to "수행해야 한다"
- "영향을 미칠 수 있다" to "영향을 줄 수 있다"
- "이 말을 초보자 기준으로 풀면" to "이 문장을 입문자 관점에서 풀면"

The `s3` section is the best result in this run. It preserved headings, table
shape, code fence, numbers, and technical claims while smoothing several awkward
transitions.

## What Failed

The original structure checker passed `s1-h1.md`, but the review found a
content-fidelity failure: the final concluding paragraph was removed.

This exposed a blind spot. Checking headings, links, code fences, tables,
numbers, and quote counts is not enough for a tone-only rewrite. A rewrite can
preserve those signatures while still dropping an ordinary paragraph.

The checker was updated during this experiment to compare non-empty Markdown
block counts. With that change:

- `s1-h1.md` fails with one hard failure, `nonempty_block_count`.
- `s3-h1.md` still passes.
- `s1-h2.md` passes after the missing paragraph is restored.
- `s2-h2.md` passes on the long-form sample.

## Interpretation

The `im-not-ai` reference is useful as a tone profile, but a direct report
post-pass is not ready for product use. The risk is not that it changes report
layout; the immediate risk is quieter: it may compress or omit content while
making the prose smoother.

The H2 repair suggests that a second content-fidelity audit can catch and fix at
least this omission class. The long-form H2 run is encouraging because it kept
the larger report intact under the stronger gate. The blind preference test then
showed a statistically significant local readability preference for H2.

This is enough to say the experiment found a real positive signal. It is not
enough by itself to ship the post-pass. The remaining product question is whether
the gain survives manual full-report review and whether a production
implementation can keep the same fidelity controls.

## Decision

Current decision: `h5_full_report_signal_confirmed`.

Do not productize behind a flag yet without product-design review of the runtime
flow and manual review of representative full reports.

The H3 follow-up was executed as a conservative loss-reduction profile. It
passed structure checks and remained strongly preferred over the original, but
it lost against the previous humanized baseline:

| comparison | H3 preferred | opponent preferred | ties | H3 share among non-ties | one-sided sign-test p |
|---|---:|---:|---:|---:|---:|
| H3 vs original | 173 | 4 | 3 | 0.977 | `2.11e-46` |
| H3 vs previous baseline | 74 | 144 | 17 | 0.339 | `0.999999` |
| H3 vs attention baseline | 6 | 17 | 2 | 0.261 | `0.994689` |

Interpretation: H3 is not adopted. The conservative loss guards removed useful
H2 edits more often than they fixed H2 loss cases. See
[`h3-result.md`](h3-result.md).

H4 then tested a selector approach: keep H2 by default and revert only cases
where the selector judged the original wording better. That path also failed.
The selector majority reverted 29 cases, including 9 prior H2 losses but also
20 prior H2 wins. On the reverted cases, retrospective blind comparison favored
H2 over the H4-selected output: H4 25, H2 58, ties 4, with H4's non-tie share at
0.301 and one-sided p(H4 > H2) at `0.999923`. The oracle upper bound shows a
better selector would help on this corpus, but the actual selector is rejected.

H5 tested a different follow-up: a whole-report tone pass over H2, preserving
the same structure constraints. Both H5 candidates passed the structure gate.
Five blind judge passes compared H5 against H2 on 2 full-report cases and 16
changed-section cases.

| scope | H5 preferred | H2 preferred | ties | H5 share among non-ties | one-sided sign-test p |
|---|---:|---:|---:|---:|---:|
| all decisions | 59 | 8 | 23 | 0.881 | `5.08e-11` |
| full-report decisions | 9 | 1 | 0 | 0.900 | `0.0107` |
| changed-section decisions | 50 | 7 | 23 | 0.877 | `2.12e-09` |

Interpretation: H5 is the strongest follow-up signal so far. It directly
addresses the H2 limitation that changed-block preference did not prove whole
report readability. The full-report sample count is still small, so this is a
product-design candidate rather than an automatic runtime adoption.

See [`h5-full-report-result.md`](h5-full-report-result.md).
