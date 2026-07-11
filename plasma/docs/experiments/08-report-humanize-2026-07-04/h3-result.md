# H3 Result

## Question

Did the H3 conservative loss-reduction profile improve on H2?

Short answer: no. H3 preserved structure and still improved over the original
wording, but it lost clearly against the previous humanized baseline.

## Setup

H3 was generated for the same three archived samples:

| sample | previous humanized baseline | H3 candidate | structure gate |
|---|---|---|---|
| `s1-short-ai-report` | `s1-h2.md` | `s1-h3.md` | pass |
| `s2-long-gemma4-report` | `s2-h2.md` | `s2-h3.md` | pass |
| `s3-section-gemma4-part03-section02` | `s3-h1.md` | `s3-h3.md` | pass |

`s3` did not have a separate H2 candidate, so `s3-h1.md` was used as the
previous humanized baseline for that sample.

The H3 profile was intentionally conservative:

- keep only clearly useful H2-style tone edits;
- revert changes that made the prose softer, less report-like, or weaker in
  explanatory framing;
- prefer the original when the difference was marginal;
- preserve headings, blocks, tables, code fences, source-bearing tokens,
  numbers, and quoted text.

## Blind Preference Test

The H3 test used five blind judge passes over 88 randomized cases:

- `h3_vs_original`: 36 cases where H3 differed from the original.
- `h3_vs_baseline`: 47 cases where H3 differed from the previous humanized
  baseline.
- `h3_vs_baseline_attention`: 5 cases from the prior H2 loss/tie attention set
  that H3 actually changed.

The judges selected A, B, or tie using Korean report readability plus fidelity
as the decision criterion. H3 identity was decoded only after judgments were
written.

## Result

| comparison | cases | H3 wins | opponent wins | ties | H3 share among non-ties | one-sided sign-test p |
|---|---:|---:|---:|---:|---:|---:|
| H3 vs original | 36 | 173 | 4 | 3 | 0.977 | `2.11e-46` |
| H3 vs previous baseline | 47 | 74 | 144 | 17 | 0.339 | `0.999999` |
| H3 vs attention baseline | 5 | 6 | 17 | 2 | 0.261 | `0.994689` |

Case-majority results:

| comparison | H3 majority | opponent majority | tie majority |
|---|---:|---:|---:|
| H3 vs original | 35 | 0 | 1 |
| H3 vs previous baseline | 14 | 30 | 3 |
| H3 vs attention baseline | 1 | 3 | 1 |

## Interpretation

H3 proves that the conservative profile can still beat the original wording, but
that is not enough. The relevant question was whether H3 improves on H2 by
reducing H2's losses. It did not.

The result suggests that the H3 loss guards were too conservative. They removed
many useful H2 edits while fixing only a small part of the prior loss/tie set.
The attention comparison is especially important: H3 changed only 5 of the 16
prior H2 loss/tie cases, and even there the previous baseline was preferred.

## Decision

Reject H3 as the next product direction.

Keep the H2 result as the positive statistical signal, but do not productize the
post-generation tone pass yet. If this line of work continues, the next useful
step is not a more conservative pass. It should either:

1. productize H2 only behind stronger fidelity checks and manual review, or
2. run a new profile that targets full-report reading quality rather than local
   block-level loss reduction.
