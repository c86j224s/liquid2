# H3 Judgment

Status update: this judgment led to an H3 experiment, and H3 was rejected after
the blind preference test. See [`h3-result.md`](h3-result.md) for the completed
result.

## Question

Should the experiment continue from H2 to H3?

Short answer: yes, but only as a conservative loss-reduction profile. H3 should
not be a stronger style pass.

## Basis

The blind preference test found a statistically significant positive signal for
H2:

- 76 changed text blocks;
- 3 independent blind judge passes;
- 228 total decisions;
- H2 preferred in 181 decisions;
- original preferred in 39 decisions;
- 8 ties;
- exact one-sided sign-test p-value: `2.32e-23`.

The remaining improvement target is narrow. By case majority, H2 lost 13 cases
and tied 3 cases. Those 16 cases are the useful input for H3.

## Failure Patterns

The H2 losses were not random. They mostly came from five patterns.

### 1. Report Register Became Too Soft

Some H2 edits made Korean more casual or less report-like. The problem is not
that the sentence became unreadable; the problem is that the original was already
the better report sentence.

H3 implication:

- Do not replace formal report collocations with softer everyday phrasing unless
  the original is clearly awkward.
- Prefer technical-report register over conversational smoothness.

### 2. Explanatory Framing Was Weakened

Several losses came from replacing phrases that carry explanation stance or
interpretive caution. These phrases often look a little stiff, but they tell the
reader how to treat the sentence.

H3 implication:

- Preserve framing phrases when they mark interpretation, definition, or safe
  reading.
- Be careful with edits around "understand as", "read as", "the point is", and
  similar report scaffolding.

### 3. Local Rewrite Hurt Korean Collocation

Some edits were locally plausible but less natural in Korean after the
surrounding noun phrase, particle, or verb was considered.

H3 implication:

- Add a collocation check after local edits.
- If the rewrite only changes a particle or connective and the original is more
  idiomatic, revert to the original.

### 4. List Items Prefer Parallel Shape

One loss class came from smoothing list-item text into a more prose-like shape.
For report lists, noun-phrase parallelism can be better than sentence-level
smoothness.

H3 implication:

- Preserve list-item grammar and parallelism unless the original list item is
  clearly broken.
- Do not optimize each bullet as if it were a standalone paragraph.

### 5. Some Differences Are Not Worth Editing

Several tied or near-tied cases had very small wording differences. In those
cases, a humanize pass adds risk without meaningful reader benefit.

H3 implication:

- Add a minimum-benefit rule: if the improvement is marginal, leave the original
  unchanged.
- H3 should produce fewer edits than H2, not more.

## H3 Profile

H3 should be:

```text
H2 + conservative loss guards
```

The candidate profile:

1. Start from the H2 tone profile.
2. Make only local, meaning-preserving Korean tone edits.
3. After drafting, compare each changed block against the original.
4. Revert a changed block when the original is better on report register,
   explanatory framing, technical precision, list parallelism, or Korean
   collocation.
5. Keep the H2 structure and source-safety gates unchanged.

H3 should explicitly avoid:

- stronger casualization;
- stronger prose polishing;
- new structure;
- new examples;
- source or evidence changes;
- changes whose only benefit is "slightly smoother".

## Next Experiment

The next experiment should generate H3 candidates for the same three samples and
run three comparisons:

1. **H3 vs original** over all changed blocks.
2. **H3 vs H2** over all changed blocks.
3. **H3 vs H2** over the 16 H2 loss/tie cases only.

Success criteria:

- H3 passes the same structure gate as H2.
- H3 does not introduce a content-fidelity failure.
- H3 keeps the overall H2 preference signal.
- H3 reduces original-majority cases below the H2 baseline of 13.
- H3 improves the 16 attention cases without losing many of H2's existing wins.

## Decision

Proceed to an H3 experiment.

Do not productize H2 or H3 yet.
