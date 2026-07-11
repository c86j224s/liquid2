# Contamination Audit

## Isolation

- Each variant used a separate source copy under
  `<experiment-source-root>`.
- Source copies were created from commit `a4aff7d` after the C1 Markdown artifact
  browser fix.
- Experiment outputs were written only under this repeat directory and per-run
  directories.
- Agents were instructed not to modify files in the analyzed source copy.

## Known Limits

- The four variants all analyzed the same product slice, so findings can
  converge naturally.
- The host agent selected steering prompts after reading prior outputs. This is
  intentional for the controller experiment, but means the controller is not a
  blind evaluator.
- Only seed 0002 was executed in this repeat. Seeds 0003 and 0004 are reserved
  but not run here.
- The event logs preserve full Codex JSONL traces. This audit does not claim the
  traces are normalized across variants beyond the common prompt structure.

## Observed Separation

- V0 remained a straightforward code-path audit.
- V1 stayed confirmatory and classified product gaps versus test gaps.
- V2 deliberately changed perspective after over-focusing on atomicity.
- V3 followed the scheduled divergence rule and mapped the user artifact
  lifecycle.

No experiment output was copied back into the analyzed source copies.
