# Expanded Repeat Judgment - 2026-06-22

## Purpose

This run extends the seed 0002 controller repeat with seeds 0003 and 0004. The
goal is to decide whether the earlier V2/V3 signal generalizes, or whether it
was a seed-specific result.

## Design

- Source copies use product commit `a4aff7d`, not current HEAD, to avoid
  contaminating the agents with prior experiment documents.
- Seeds 0003 and 0004 each run V0, V1, V2, and V3.
- Each run has turn1, three controller decisions, turn2-turn4, and a final
  report.
- Seed 0004 uses a fixed repeated decision protocol to reduce controller
  discretion during the repeat.
- Blind evaluators score intermediate answers and final reports without seeing
  variant names.

## Result

The added seeds do not support declaring a single winning variant. They do
support a narrower product decision:

- Focused reliability questioning is useful and repeatedly surfaces atomicity,
  orphan artifact, duplicate SHA, and failure telemetry issues.
- Lifecycle/UX divergence is useful and repeatedly surfaces raw preview, Korean
  filename, artifact card metadata, mission rediscovery, and cancel affordance
  issues.
- Baseline/confirmatory narrowing remains valuable. It can produce the strongest
  implementation map and sometimes wins blind evaluation.

The controller should therefore be adaptive, not a fixed "always diverge" or
"always creative-switch" variant.
