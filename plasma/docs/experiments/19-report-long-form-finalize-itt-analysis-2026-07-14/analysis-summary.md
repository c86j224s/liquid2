# Experiment 19 Analysis Summary

Experiment 18's immutable evidence yielded exactly 12 intention-to-treat topic
pairs: 11 existing scored pairs and one preserved post-start candidate failure.
The existing ITT rule assigned the incomplete pair score 1 on every final-report
dimension for both arms. No provider, judge, packet, or new pairing was run.

Across the nine final-report dimensions, the mean paired candidate-minus-baseline
difference was 0.0417. The preregistered 10,000-draw one-sided 95% bootstrap
lower bound was -0.1065, above the -0.25 noninferiority margin. Completeness mean
difference was 0.0. Both arms had one low completeness score among 12 topics, so
the candidate low-score-rate increase was 0.0. Both completeness guardrails
passed.

The transfer audit from the scored product to candidate `8a054e6` passed for
successful final prompting and boundary instructions, mechanical assembly, and
the public artifact path. The bounded quality gate therefore passed. Raw records,
score files, and the private arm mapping remain outside Git.
