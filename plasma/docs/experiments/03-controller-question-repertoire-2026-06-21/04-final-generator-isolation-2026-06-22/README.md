# Final Generator Isolation - 2026-06-22

## Purpose

This experiment checks whether the observed V2/V3 advantage came from the
controller-shaped intermediate analysis or merely from the final report
generation step.

## Design

The experiment uses the completed seed 0002 repeat run, but excludes the
original `final-report.md` files from the input.

Two checks are performed:

1. Neutral final regeneration: a fresh neutral Codex session receives only
   turn1-turn4 intermediate answers for one blind bundle and writes a new final
   report with the same prompt.
2. Blind intermediate evaluation: another neutral session receives K1-K4
   intermediate bundles without variant names or controller decision metadata
   and evaluates whether the differences are already visible before final
   report generation.

## Blind Mapping

The blind mapping is stored for reproducibility, but it is not included in the
evaluation prompt:

- K1: M5-V2-seed-0002
- K2: M5-V0-seed-0002
- K3: M5-V3-seed-0002
- K4: M5-V1-seed-0002

## Interpretation Rule

If the blind intermediate evaluation sees the same V2/V3-shaped strengths before
the original final reports are shown, and neutral final regeneration preserves
those strengths, then the separation is not merely a final-report generation
artifact. It still remains a seed-0002 result, not a statistical proof.

## Result

The isolation checks passed for seed 0002. The intermediate-only blind evaluator
already saw distinct strengths in K1/K3/K4 before any original final report was
shown, and the neutral regenerated final reports preserved the same shape.

Within seed 0002, the observed separation cannot be explained solely as an
artifact of the original final report generator. Broader controller ranking still
requires more seeds.
