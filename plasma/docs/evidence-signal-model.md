# Plasma Evidence Signal Model

Status: this is a future/legacy design note, not the active C1 default product
loop. The current default loop is source-centered and does not create evidence,
claims, confidence updates, or proposal records. If this model is revived, it
must act as a reference, index, and traceability layer over sources rather than
a gate that decides whether investigation or report generation may proceed.

This document records the product checkpoint for expanding Plasma's evidence
model. The direction is to avoid blocking useful information while making the
evidence hierarchy rich enough for the user and report writer to judge it.

## Principle

Plasma should not reject useful research material merely because it is not a
confirmed fact. Rumors, reactions, interpretations, community discussion, code
examples, formulas, benchmarks, and conflicting claims can all be useful
research signals.

The control point is not whether the signal may enter the mission. The control
point is how clearly Plasma labels the signal, connects it to sources, and
shows its confidence, limits, and report-use value.

Agent results are not sources. A result may identify or summarize a signal, but
the saved evidence must still point back to an original source snapshot, user
assertion, code location, formula source, or other explicit provenance.
One source may yield many evidence records, and those records should be editable,
removable, and supersedable without changing the identity of the source itself.

## Signal Kinds

Evidence should be able to represent more than strict facts:

- `fact`: official or source-stated factual information, such as dates,
  specs, cast names, API behavior, or release metadata.
- `interpretation`: an analyst, critic, author, or agent interpretation that
  is useful but not itself a verified fact.
- `reaction`: community, market, press, or audience response.
- `rumor`: unconfirmed report, leak, speculation, or circulating claim.
- `controversy`: a recurring disagreement, backlash axis, or contested framing.
- `market_signal`: presales, traffic, view counts, adoption signals, rankings,
  or other demand/attention indicators.
- `code`: source code, example code, tests, snippets, API usage patterns, or
  implementation details.
- `formula`: mathematical expression, model, algorithmic equation, or derived
  calculation.
- `benchmark`: measured performance result, comparison, experiment, or
  reproducibility note.
- `open_question`: an explicit gap, unresolved contradiction, or missing
  verification target.

These kinds do not decide approval by themselves. They inform how the evidence
is displayed, how it affects claim confidence, and how report generation should
word the material.

## Confidence And Usefulness

Plasma should separate two judgments:

- Confidence: how reliable the signal is as a statement about the world.
- Report value: how useful the signal is for explaining the mission, even when
  it is weak, contested, speculative, or only representative of discourse.

A rumor may have low confidence but high report value if the mission is about
public reaction. A code example may have high confidence as "this repository
uses the API this way" but lower portability if version, runtime, or license
constraints are unclear.

Reports should preserve this distinction. They may include weak or conflicting
signals, but they must not flatten them into confirmed facts.

Example wording:

> The casting is not confirmed by a primary source, but the claim is important
> as a reaction signal because several secondary sources discuss it as a
> controversy axis.

## Rigor Levels

Strictness is a report-generation control, not a research-domain taxonomy. It
does not decide whether useful material may enter the mission. It changes how
evidence is used, weighted, and worded when Plasma writes a report.

Initial levels:

- `exploratory`: collect and use a broad set of signals. Rumors, reactions,
  interpretations, controversies, market signals, code examples, formulas,
  benchmarks, and open questions may enrich the report when clearly labeled.
  Weak material must remain visible as weak material.
- `balanced`: anchor the main narrative on source-backed facts and medium/high
  confidence claims. Use weak or interpretive signals as context, competing
  accounts, unresolved questions, or explanatory color when they materially
  improve understanding.
- `strict`: base major conclusions on source-backed facts and medium/high
  confidence evidence. Weak signals may still appear, but only as explicitly
  labeled uncertainty, background discourse, risk, or coverage gaps.

The level changes collection breadth only indirectly through the report writer's
use of saved evidence. It must not silently discard useful signals or turn
approval into a higher user hurdle.

## Code Evidence

Code evidence needs metadata beyond text:

- language and framework
- source type, such as official docs, GitHub repository, issue, PR, Stack
  Overflow, blog, or local code
- repository, commit, path, line range, URL, or package version
- role, such as API example, production implementation, test, workaround,
  anti-pattern, or migration note
- execution status: runnable, partial snippet, pseudocode, or illustrative
- runtime and dependency constraints
- license or copyright risk
- portability caveats

Reports should cite code as evidence for patterns and constraints, not as
universal truth. Long code should not be copied into reports unless the license
and quoting limits allow it; prefer short excerpts, links, and explanation.

## Formula Evidence

Formula evidence needs enough context to be useful:

- original formula in LaTeX, MathML, or plain text
- variable definitions
- units
- assumptions and boundary conditions
- derivation source
- example calculation, when available
- applicability limits
- confidence and verification status

Reports should not render a formula without its assumptions. If a formula is
used only as an explanatory model, label it as such.

## Report Behavior

Report generation should become richer, not looser.

It should:

- include confirmed facts, interpretations, reactions, rumors, and conflicts
  when they are useful to the mission
- label signal kind and confidence in the prose or in collapsible metadata
- show competing accounts as competing accounts instead of forcing a single
  resolution too early
- include missing-information and coverage-gap sections by default when gaps
  matter
- surface high-value weak signals without promoting them to facts
- keep Markdown report artifacts as the C1 default product output. AST report
  records remain legacy or explicit experiment machinery until a later product
  decision replaces the default artifact model

For wiki-like topics, Plasma should benchmark predictable coverage patterns
such as fact sheets and section scaffolds. It should not become a generic wiki.
Its advantage is evidence lineage, confidence, uncertainty, corrections, and
mission-specific synthesis.

## Product Guardrails

- Do not block useful information merely because it is weak.
- Do not auto-promote weak signals into saved facts.
- Do not turn source candidates, evidence, claims, and reports into one flat
  bucket.
- Do not make confidence updates an approval hurdle.
- Do not hide uncertainty to make reports look cleaner.
- Do not make the user approve every search step. Let the agent search when the
  mission implies investigation; when optional source, evidence, claim, and
  report promotion points exist, keep approval at those promotion boundaries
  rather than at every search step.
- Do not let a fact sheet become a free-text surface detached from evidence.

## Implementation Slices

Suggested implementation order:

1. Add signal-kind and source-quality vocabulary to evidence and proposal
   surfaces without changing approval rules.
2. Add report rigor levels to mission/report generation context.
3. Teach report drafting to use signal kinds and confidence when wording
   facts, interpretations, reactions, rumors, conflicts, code, and formulas.
4. Add a fact sheet and coverage map as projections over approved evidence and
   claims.
5. Add report freshness signals when new approved evidence changes a section's
   confidence or coverage.
