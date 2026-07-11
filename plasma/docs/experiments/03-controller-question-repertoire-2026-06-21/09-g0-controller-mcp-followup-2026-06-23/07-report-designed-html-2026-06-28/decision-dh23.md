# DH23 Decision Note

## Decision

Keep `DH23` as the current product candidate for designed interactive HTML report artifacts.

`DH23` uses a thin prompt to ask for a Korean content model, then a deterministic renderer creates a mobile-safe self-contained HTML report. The strongest subject-specific `visual_unit` is promoted into the first viewport as a connected SVG relationship map.

`DH24` was added as a follow-up probe to compress the first viewport further and vary the hero visual grammar by subject. It passed the hard-fail and render gates, but the blind review evidence did not justify replacing `DH23`.

## Why DH23

The reference-grade gap was not solved by prettier cards alone. The major improvement came from making the first viewport explain a real relationship before the reader reaches the body:

- provenance path
- product or runtime flow
- controller or safety boundary
- purchase decision route
- OAuth/OIDC trust path
- stack role map
- historical causal chain

`DH22` proved that promoting the strongest visual unit into the hero improves topic identity. `DH23` improved that further by rendering the promoted unit as a connected relationship map instead of a card strip.

## Evidence

- `DH20`, `DH22`, and `DH23` all completed 24/24 hard-fail passing runs with zero mobile overflow failures.
- `DH22` beat `DH20` in blind screenshot review after answer-key mapping: 24/24 selections for `DH22`.
- `DH22` beat `DH17` in blind screenshot review after answer-key mapping: 23/24 selections for `DH22`.
- `DH23` beat `DH22` in two independent blind screenshot reviews after answer-key mapping: 24/24 selections for `DH23` in both Nimitz and Sentinel reviews.
- `DH24` completed 24/24 hard-fail passing runs with zero render failures, but did not displace `DH23` in blind review after answer-key mapping:
  - Nimitz completed side: `DH23` 16, `DH24` 8.
  - Sentinel completed side: `DH23` 24, `DH24` 0.
  - Combined mapped preference: `DH23` 40, `DH24` 8.

The public blind labels were mixed in those reviews, so the result was not caused by always choosing A or B.

## Important Correction

The existing `DH21` runs are contaminated for ablation purposes. They were intended to be reference-free, but their prompts still included the local reference gallery profile. Do not use those runs as evidence that the reference-free visual-magazine direction succeeded.

The harness now withholds the reference gallery profile for `DH20`, `DH21`, `DH22`, `DH23`, and `DH24`.

## Review Upload

A representative `DH23` review set was copied to the Liquid2 release static root for manual review:

- URL: `<internal-review-url>`
- Server path: `<internal-release-server-path>`
- Contents: 8 representative HTML reports, each with desktop and mobile screenshots.

This upload is a review artifact, not product behavior.

## Product Guidance

Use `DH23` as the first implementation target. Keep `DH20` and `DH22` as fallbacks:

- `DH20`: stable compact report-app baseline.
- `DH22`: readable hero card-strip version if SVG hero maps prove too heavy in production.
- `DH23`: current preferred candidate.

`DH24` remains an informative failed replacement attempt. Its useful lesson is that simply compressing the first viewport and varying visual grammar is not enough; the connected map structure in `DH23` was preferred more consistently.

## Known Limitation

User review identified a material weakness in `DH23`: it solves the need for a
meaningful first-screen visual center, but it overuses one infographic grammar.
The model may label units as timeline, matrix, flow, decision route, causal
field, or architecture map, but the current deterministic renderer still turns
most of them into the same connected node/edge relationship SVG.

That means the artifacts can pass hard-fail gates, preserve density, and win
blind screenshot comparisons while still feeling visually repetitive:

- history becomes a relationship map
- OAuth/OIDC becomes a relationship map
- product purchase comparison becomes a relationship map
- code/product analysis becomes a relationship map
- section-level visual units differ in name more than in actual visual form

This is not primarily a prompt problem. It is a renderer capability problem.
The next improvement should not ask the model for "more diverse infographics"
unless the renderer can actually draw different information grammars.

## Deferred DH25 Hypothesis

Reserve `DH25` for a materially different follow-up, not another compact-hero
or content-first variation. The proposed `DH25` hypothesis is:

> A visual-grammar dispatcher plus multiple deterministic renderers will beat
> `DH23` because the artifact can choose and render the subject's natural
> information structure instead of forcing every topic into a relationship map.

Candidate renderer grammars:

- timeline or causal field for history, biography, and geopolitical causality
- swimlane, trust boundary, or request/response path for OAuth, protocol,
  security, and API design
- role matrix, dependency map, module boundary, or operating loop for codebase,
  stack, tool, and workflow analysis
- decision route, cost ladder, trade-off board, or risk checklist for purchase
  and product choice reports
- evidence chain or uncertainty ladder for source-heavy or conflicting material

Acceptance criteria for `DH25`:

- The content model must choose an explicit `visual_grammar` for each major
  `visual_unit`.
- The renderer must produce different DOM/CSS/SVG structures for different
  grammars; changing a label while still drawing the same node/edge map fails.
- A representative review set must show visible grammar variety across topics
  and inside long reports.
- Blind review should compare `DH25` against `DH23`, but human review must also
  check whether the artifacts stop feeling like one reusable infographic skin.

Do not run `DH25` until the dispatcher and at least three non-map renderers are
implemented in the experiment harness. Otherwise the test will likely repeat
the `DH24` failure mode.

## Product Adoption Preparation

The product should adopt the `DH23` path cautiously:

- Start with the deterministic content-model renderer, not prompt-only HTML
  generation.
- Keep the report artifact self-contained and mobile-safe.
- Preserve source links, caveats, uncertainty notes, and dense section content.
- Treat the renderer as a replaceable adapter. Visual-grammar diversification is
  deferred follow-up work and is not part of the first product cutover.
- Expose the designed HTML artifact as an additional report output beside the
  existing markdown/longform output, not as a replacement.
- Do not claim final reference-grade parity until the visual-grammar diversity
  gap is closed and reviewed.

## Product Cutover Scope

The first product cutover should implement the validated `DH23` shape without
starting a new experiment:

1. A Markdown or longform report artifact already exists.
2. The user requests a designed HTML artifact for that report.
3. Plasma records a pending report-design event that points at the source report
   artifact. The source report remains a result artifact, not a source.
4. An agent turn creates a JSON content model from the report artifact only.
5. Plasma stores the content model as an internal report-rendering artifact or
   event payload for traceability.
6. A deterministic renderer creates a self-contained HTML report artifact from
   that content model.
7. The browser shows the designed HTML artifact beside the existing Markdown
   artifact with view and download actions.

This cutover must not:

- replace the existing Markdown report path
- classify report artifacts as sources
- expose prompt variant names such as `DH23` in the user-facing UI
- implement `DH25` or any new visual-grammar experiment as part of the same
  product change
- claim parity with the `google-io-2026.html` reference until the remaining
  visual-grammar diversity gap has been closed

Do not commit the full generated `runs/` corpus by default. It is large and mostly reproducible experiment output. Commit the harness, this decision note, summary files, and selected lightweight records instead.
