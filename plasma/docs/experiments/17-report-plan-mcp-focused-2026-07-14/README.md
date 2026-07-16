# Focused Report Plan MCP Experiment - 2026-07-14

This successor keeps issue 110's product candidate unchanged and corrects the
provider mismatch found in the completed #16 authentication smoke. The
controller and product candidate commits are locked separately. Planned and
long-form reports both use the locked Codex model with high reasoning effort.

The experiment answers two questions:

1. Can both report modes submit and promote their plans through the real product
   CLI, Web API, Codex provider, and MCP paths?
2. Does the MCP plan path preserve report quality relative to the baseline in
   planned and long-form modes?

After the passed smoke, the focused quality phase runs each of the 12 frozen
quality topics once for planned/long-form and baseline/candidate: 48 product
runs and up to 24 blinded completed mode/topic pairs. The frozen source bundles are short, so
this measures relative non-degradation against the paired baseline, not
absolute long-form report quality.

The execution gate records matrix completeness separately from all-success.
Started failures remain intention-to-treat records; only pre-run infrastructure
failures block the phase. When a pair is incomplete, it is not judged and the
analysis uses the documented deterministic low-score treatment.

Existing binding, idempotency, retry, and recovery rules remain deterministic
product-test concerns. This experiment tests only the product paths and quality
non-degradation. Raw runs stay outside the repository under the experiment
archive policy.

## Result

The frozen quality phase started all 48 real product runs: 12 topics across
planned and long-form modes, with baseline and candidate arms. Forty-seven runs
completed. One candidate long-form run was an intention-to-treat (ITT) failure.
One completed baseline planned run also failed the machine audit because its
source-read trace was missing. The resulting matrix had 23 blind eligible
pairs.

All four pre-specified quality endpoints met the non-inferiority criterion. The
candidate-minus-baseline mean difference and lower confidence bound were:

| Endpoint | Mean difference | Lower confidence bound |
| --- | ---: | ---: |
| Planned final report | +0.0787 | -0.0741 |
| Planned plan | +0.2083 | -0.0521 |
| Long-form final report | +0.0556 | -0.1250 |
| Long-form plan | +0.0729 | -0.0208 |

The guardrail checks and the planned/long-form mode claims were true. However,
the candidate long-form ITT failure and the missing baseline source-read trace
made `machine_gate` false; consequently, `overall_claim` was also false.

The bounded quality conclusion is that this frozen comparison supported
non-degradation for the measured endpoints. Positive mean differences do not
prove superiority, and the short frozen sources do not support an absolute
long-form quality claim. Separately, the operational reliability and
productization conclusion failed and remains blocked under the frozen protocol.
Raw artifacts remain in the local archive under the experiment archive policy.
