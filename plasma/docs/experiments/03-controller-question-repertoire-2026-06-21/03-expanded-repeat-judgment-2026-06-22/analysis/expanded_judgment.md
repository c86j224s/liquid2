# Expanded Judgment

## Question

After running more seeds, should Plasma choose one controller variant as the
default research controller?

## Evidence

The expanded run now includes:

- seed 0002: original repeat plus final-generation isolation,
- seed 0003: full repeat with blind intermediate and final evaluations,
- seed 0004: full repeat with the same fixed decision protocol and blind
  intermediate/final evaluations.

All seed 0003 and 0004 source copies were made from product commit `a4aff7d` to
avoid contaminating the analysis with earlier experiment conclusions.

## Findings

The earlier signal did not collapse, but it also did not become a single
variant ranking.

V2-style reliability focus repeatedly helped. It surfaced:

- non-atomic raw artifact plus `report.artifact.created` writes,
- orphan raw artifact risk,
- duplicate `(mission_id, sha256)` retry risk,
- thin `report.draft.failed` telemetry,
- missing tool/session linkage for failed report generation.

V3-style lifecycle divergence repeatedly helped. It surfaced:

- raw Markdown preview versus rendered report expectation,
- Korean title collapse to `source.md`,
- missing filename/media metadata on artifact cards,
- weak later rediscovery through mission list ordering,
- missing report generation cancel affordance.

V0/V1-style confirmatory narrowing also remained useful. It sometimes won blind
evaluation because it produced a clearer implementation map or a cleaner
product/test/design-decision triage.

## Judgment

Do not pick a fixed winner such as "always V2" or "always V3".

The right product direction is an adaptive controller:

1. Start with confirmatory narrowing until the implementation or research map is
   precise enough.
2. If the analysis over-focuses on one mechanism, switch to product recovery,
   observability, or user lifecycle questions.
3. If the analysis is broad but shallow, return to focused confirmation.
4. Final report generation should synthesize the already-shaped intermediate
   material; it should not be expected to invent the missing insight by itself.

## Practical Controller Policy

For Plasma C1, the controller should keep a small repertoire:

- `confirm_boundary`: narrow code/data boundaries and object identity.
- `confirm_lifecycle`: trace pending, success, failure, refresh, and recovery.
- `reliability_switch`: reframe a local persistence issue as user-visible loss,
  recovery, and observability.
- `ux_lifecycle_switch`: reframe implementation paths as user states and
  affordances.
- `triage_split`: separate product bug, test-only gap, and design decision.

The controller should choose among these based on the intermediate answer, not
based on a fixed variant label.

## Product Work That Stayed Stable Across Seeds

The same product work remains high priority:

1. Make C1 Markdown artifact creation atomic.
2. Improve failed report telemetry with stage, duration, agent session, tool
   session, and log/result excerpt where available.
3. Decide raw versus rendered Markdown view.
4. Fix Korean/download filename policy.
5. Add browser-level tests for report artifact cards, view/download actions,
   filename handling, preview semantics, and report failure state.
6. Revisit report generation cancel/retry and mission rediscovery behavior.

## Limit

The evaluators are still LLM evaluators. The result is strong enough for product
direction, not strong enough to declare a statistically superior controller
variant.
