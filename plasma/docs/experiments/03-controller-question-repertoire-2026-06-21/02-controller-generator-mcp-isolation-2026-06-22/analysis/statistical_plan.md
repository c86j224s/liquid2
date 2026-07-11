# Statistical Plan

## Design

The experiment uses paired mission-seed blocks. Each usable block contains all
eight cells: C0G0M0, C0G0M1, C0G1M0, C0G1M1, C1G0M0, C1G0M1, C1G1M0, C1G1M1.

Primary claims use only complete clean blocks. Unpaired cells and contaminated
blocks may be described qualitatively but cannot support primary factor claims.

## Main Effects

- Controller effect: C1 vs C0 averaged over matched G and M cells.
- Generator session effect: G1 vs G0 averaged over matched C and M cells.
- MCP surface effect: M1 vs M0 averaged over matched C and G cells.

## Interactions

- Controller x generator.
- Controller x MCP surface.
- Generator x MCP surface.

## Co-primary Outcomes

- Readability/depth/breadth improvement.
- Grounding/provenance non-inferiority or improvement.

## Risk Thresholds

- Unverifiable conclusion rate must not exceed baseline by more than 3 points.
- Unsupported conclusion rate must not exceed baseline by more than 2 points.
- Overclaim rate must not exceed baseline by more than 2 points.
- Provenance completeness must be at least 90% or no worse than baseline by more
  than 3 points.
- Internal ID/path leakage must be no worse than baseline.

## Looks And Stopping

- No success conclusion before six complete clean blocks.
- First look at six blocks is harm/futility oriented.
- Additional looks occur after every three complete clean blocks.
- Cap at sixteen complete clean blocks.
- Report success, failure, harm, or inconclusive without inventing significance.

## Current Data Status

Nine complete clean judged blocks have been executed and force-rejudged.

The nine-block analysis produced one product-actionable harm finding and several
unresolved factors:

- No main effect produced a supported positive win for controller, generator
  session separation, or added research surface.
- `generator:depth` produced a supported harm signal for `G1` versus `G0`. This
  is enough to reject separate report-generator session as the default report
  path for this experiment's product decision scope.
- Controller and MCP-surface factors remain unresolved. They are not validated,
  but they are not proven ineffective.
- Further work on controller quality or MCP research-surface design should use a
  narrower follow-up hypothesis rather than extending this exact setup by
  default.
