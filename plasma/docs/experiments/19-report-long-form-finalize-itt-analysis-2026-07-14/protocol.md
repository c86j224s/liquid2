# Experiment 19 Protocol

## Frozen inputs

- Source experiment: `18-report-long-form-finalize-mcp-2026-07-14`, read-only.
- Product represented by the scored reports: `4bc3ac07fab93f31d9447c0a83802f6628bd9623`.
- Current candidate to which the bounded result may transfer: `8a054e6d7d1e50a9ebeb72b6bf6b933303264dc1`.
- Input set: exactly 24 long-form terminal manifests, 23 completed runs, one
  post-start candidate report-plan-binding failure, 11 private mappings, and 11
  judge score files, all locked by path and SHA-256.
- There are no provider, judge, packet, pairing, recovery, or fault actions.

## Analysis

The only action order is `prepare-analysis -> assemble-itt -> analyze-quality`.
The existing provenance adapter reconstructs the 11 scored pairs. The existing
ITT helper assigns every final dimension score 1 to both arms of the incomplete
topic pair, yielding exactly 12 topic pairs. Inputs are not rescored or repaired.

Only the nine final-report dimensions enter the topic-paired
candidate-minus-baseline composite. The one-sided 95% bootstrap lower bound uses
seed 110 and 10,000 draws and must be at least -0.25. Completeness separately
requires a mean difference of at least -0.50 and a candidate low-score-rate
increase no greater than 0.10. All three conditions must pass.

Transfer to the current candidate is conditional on a product diff audit showing
that successful final-report prompting, opening/closing instructions, mechanical
assembly, and public artifact shape did not change between the scored product and
the current candidate. This result does not establish operational reliability.
