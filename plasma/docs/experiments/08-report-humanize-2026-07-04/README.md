# Report Humanize Experiment - 2026-07-04

This experiment checks whether the external `im-not-ai` Humanize Korean rules
can improve the Korean tone of Plasma reports without changing report
structure, source handling, or report mode.

## Status

- Issue: <https://github.com/c86j224s/liquid2/issues/17>
- Branch: `feat/plasma-report-humanize-experiment`
- PR boundary: one issue and one PR cover the design, run, result summary, and
  adoption decision for this experiment.
- User gate: `docs` unless this branch later adds product runtime behavior.
- Product runtime: unchanged at the start of the experiment.
- Raw archive root:
  `~/research-artifacts/liquid2/plasma/experiments/08-report-humanize-2026-07-04/`

## Product Question

Can Plasma add a conservative post-generation tone pass for Korean reports that
reduces AI-like wording while preserving the report as a report?

The goal is tone correction only. This experiment does not try to improve report
layout, section strategy, source selection, designed HTML quality, or long-form
composition.

## Boundaries

In scope:

- Korean tone naturalness.
- Translationese, repeated rhythm, redundant connectors, and stiff AI phrasing.
- Structure-preserving Markdown rewrite checks.
- Content-fidelity checks against the original report.

Out of scope:

- Changing report outline, heading hierarchy, or section order.
- Turning lists into prose or prose into lists as a product behavior.
- Rewriting citations, URLs, source labels, code blocks, tables, numbers, dates,
  or quoted text.
- Designed HTML polish.
- Product integration before the experiment result is accepted.

## Variants

- H0: original Plasma Markdown report.
- H1: conservative tone-only humanize pass based on `im-not-ai` references.
- H2: H1 plus a second audit pass for content fidelity and remaining AI-like
  wording.
- H3: H2 plus conservative loss guards, tested as a follow-up profile after the
  H2 blind preference result.
- H4: selector exploration that tried to revert H2 loss cases back to the
  original wording.
- H5: H2 plus a whole-report tone pass, tested against H2 on full reports and
  changed sections.

H1 and H2 must use a Plasma-specific profile. The upstream `im-not-ai` behavior
is a reference, not a product contract. In particular, Plasma reports must keep
their Markdown structure and evidence-bearing tokens intact.

## Sample Corpus

The initial corpus should cover at least three report shapes:

1. A short recent report, around 10-20 KB.
2. A long final report, around 80 KB or larger.
3. A section or part artifact from a multi-part report.

Raw samples stay in the local archive. The repository may contain only manifests,
redacted summaries, structure signatures, and aggregate findings.

## Decision Rule

Adopt a product follow-up only if a humanized variant improves readability
without any hard structural or fidelity failure.

Hard failures include:

- heading level or heading order changed;
- link target changed, added, or removed;
- code fence count changed;
- table separator count changed;
- non-empty Markdown block count changed;
- source reference, footnote, URL, quote, number, date, or model name changed in
  a way that changes meaning;
- uncertainty was strengthened or weakened without a supporting source.

If the tone improves but the report loses source fidelity, the experiment result
is "not adopted yet" and should lead to a narrower future profile.

## Current Outcome

H1 was executed on two archived samples. It improved Korean tone modestly, but
one full-report candidate dropped a final concluding paragraph.

H2 was then executed on the failing short full-report sample as a repair pass.
It restored the missing paragraph and passed the strengthened structure gate,
including the non-empty block-count check.

The same conservative H2 profile was also executed on the long-form Gemma 4
report sample. The long-form candidate passed the structure gate and an
additional host-side token preservation audit for headings, code fences, tables,
source-bearing lines, inline code, number tokens, and selected technical tokens.

The H2 candidates were then compared against their originals in a blind
preference test over 76 changed text blocks. Three independent judge passes
produced 228 total decisions. Excluding ties, H2 was preferred in 181 of 220
decisions, with an exact one-sided sign-test p-value of `2.32e-23`. This shows a
statistically significant readability preference for H2 within this experiment
set.

This is still an experiment result rather than a product decision. The sample
set is limited, and the judges were model-based rather than human readers. No
product behavior is adopted yet.

H3 was then tested as a conservative loss-reduction profile. It passed the same
structure gates and remained strongly preferred over the original wording, but
it failed against the previous humanized baseline. Across five blind judge
passes, H3 lost to the previous baseline in 144 decisions and won 74, excluding
17 ties. The one-sided sign test for H3 being better than the baseline did not
support adoption (`p = 0.999999`). H3 also failed on the prior H2 loss/tie
attention set.

Decision after H3: keep the H2 statistical signal as the useful result, reject
H3 as a product direction, and do not productize the post-pass yet.

H4 was then tested as a selector path. The idea was to keep H2 where it helped
and revert only cases where H2 was worse than the original. The actual selector
failed: it caught some H2 losses but also reverted many H2 wins, and the
retrospective comparison favored H2 over the H4-selected output. H4 is rejected.

H5 then tested a different follow-up: run a full-report tone pass on the H2
candidate and compare H5 against H2. Five blind judge passes over 2 full-report
cases and 16 changed-section cases produced 90 decisions. H5 was preferred 59
times, H2 8 times, with 23 ties. Excluding ties, H5's preference share was
88.1%, with an exact one-sided sign-test p-value of `5.08e-11`. Full-report
cases were 9 to 1 in favor of H5 across five judges.

Decision after H5: H5 has the strongest follow-up signal so far and should be
carried into product-design discussion as a post-report tone pass. It is not a
planner, source selector, or designed-HTML generator.

See [`analysis-summary.md`](analysis-summary.md).
See also [`h3-judgment.md`](h3-judgment.md) for the follow-up setup and
[`h3-result.md`](h3-result.md) for the completed H3 result.
See [`h5-full-report-result.md`](h5-full-report-result.md) for the full-report
follow-up result.
