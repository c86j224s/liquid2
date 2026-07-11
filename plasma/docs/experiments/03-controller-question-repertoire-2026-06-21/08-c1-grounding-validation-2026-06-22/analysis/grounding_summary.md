# Plasma C1 Grounding Validation Summary

Audited runs: 48 / 48 discovered

## Variant Means

| Variant | n | grounding | overclaim | unsupported | unverifiable | provenance |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| AUTO | 12 | 0.796 | 0.085 | 0.006 | 0.077 | 0.714 |
| NC | 12 | 0.910 | 0.043 | 0.000 | 0.000 | 0.839 |
| V2 | 12 | 0.803 | 0.068 | 0.000 | 0.112 | 0.709 |
| V3 | 12 | 0.797 | 0.074 | 0.006 | 0.113 | 0.657 |

## Paired Tests

- V2_minus_NC on `grounding_score`: n=12, mean_diff=-0.107, ci95=[-0.173, -0.048], sign_p=0.0386, sufficient: paired signal passes current threshold
- V3_minus_NC on `grounding_score`: n=12, mean_diff=-0.113, ci95=[-0.231, -0.009], sign_p=0.7744, insufficient: 95% CI includes zero or sign test p >= 0.05
- AUTO_minus_NC on `grounding_score`: n=12, mean_diff=-0.114, ci95=[-0.203, -0.035], sign_p=0.0654, insufficient: 95% CI includes zero or sign test p >= 0.05
- V3_minus_V2 on `grounding_score`: n=12, mean_diff=-0.006, ci95=[-0.117, 0.088], sign_p=0.3877, insufficient: 95% CI includes zero or sign test p >= 0.05
- AUTO_minus_V2 on `grounding_score`: n=12, mean_diff=-0.007, ci95=[-0.116, 0.099], sign_p=1.0000, insufficient: 95% CI includes zero or sign test p >= 0.05
- AUTO_minus_V3 on `grounding_score`: n=12, mean_diff=-0.001, ci95=[-0.084, 0.067], sign_p=1.0000, insufficient: 95% CI includes zero or sign test p >= 0.05
- V2_minus_NC on `overclaim_rate`: n=12, mean_diff=0.025, ci95=[-0.036, 0.105], sign_p=0.4531, insufficient: 95% CI includes zero or sign test p >= 0.05
- V3_minus_NC on `overclaim_rate`: n=12, mean_diff=0.031, ci95=[-0.042, 0.118], sign_p=1.0000, insufficient: 95% CI includes zero or sign test p >= 0.05
- AUTO_minus_NC on `overclaim_rate`: n=12, mean_diff=0.042, ci95=[-0.023, 0.114], sign_p=0.2891, insufficient: 95% CI includes zero or sign test p >= 0.05
- V3_minus_V2 on `overclaim_rate`: n=12, mean_diff=0.006, ci95=[-0.107, 0.117], sign_p=0.7266, insufficient: 95% CI includes zero or sign test p >= 0.05
- AUTO_minus_V2 on `overclaim_rate`: n=12, mean_diff=0.017, ci95=[-0.083, 0.112], sign_p=1.0000, insufficient: 95% CI includes zero or sign test p >= 0.05
- AUTO_minus_V3 on `overclaim_rate`: n=12, mean_diff=0.011, ci95=[-0.018, 0.045], sign_p=1.0000, insufficient: 95% CI includes zero or sign test p >= 0.05
- V2_minus_NC on `unsupported_rate`: n=12, mean_diff=0.000, ci95=[0.000, 0.000], sign_p=n/a, insufficient: no nonzero paired signal
- V3_minus_NC on `unsupported_rate`: n=12, mean_diff=0.006, ci95=[0.000, 0.017], sign_p=1.0000, insufficient: 95% CI includes zero or sign test p >= 0.05
- AUTO_minus_NC on `unsupported_rate`: n=12, mean_diff=0.006, ci95=[0.000, 0.017], sign_p=1.0000, insufficient: 95% CI includes zero or sign test p >= 0.05
- V3_minus_V2 on `unsupported_rate`: n=12, mean_diff=0.006, ci95=[0.000, 0.017], sign_p=1.0000, insufficient: 95% CI includes zero or sign test p >= 0.05
- AUTO_minus_V2 on `unsupported_rate`: n=12, mean_diff=0.006, ci95=[0.000, 0.017], sign_p=1.0000, insufficient: 95% CI includes zero or sign test p >= 0.05
- AUTO_minus_V3 on `unsupported_rate`: n=12, mean_diff=0.000, ci95=[0.000, 0.000], sign_p=1.0000, insufficient: 95% CI includes zero or sign test p >= 0.05
- V2_minus_NC on `unverifiable_rate`: n=12, mean_diff=0.112, ci95=[0.072, 0.151], sign_p=0.0020, sufficient: paired signal passes current threshold
- V3_minus_NC on `unverifiable_rate`: n=12, mean_diff=0.113, ci95=[0.056, 0.173], sign_p=0.0078, sufficient: paired signal passes current threshold
- AUTO_minus_NC on `unverifiable_rate`: n=12, mean_diff=0.077, ci95=[0.028, 0.139], sign_p=0.0312, sufficient: paired signal passes current threshold
- V3_minus_V2 on `unverifiable_rate`: n=12, mean_diff=0.001, ci95=[-0.057, 0.057], sign_p=1.0000, insufficient: 95% CI includes zero or sign test p >= 0.05
- AUTO_minus_V2 on `unverifiable_rate`: n=12, mean_diff=-0.035, ci95=[-0.106, 0.041], sign_p=0.5078, insufficient: 95% CI includes zero or sign test p >= 0.05
- AUTO_minus_V3 on `unverifiable_rate`: n=12, mean_diff=-0.035, ci95=[-0.101, 0.037], sign_p=0.3438, insufficient: 95% CI includes zero or sign test p >= 0.05
- V2_minus_NC on `provenance_completeness_rate`: n=12, mean_diff=-0.130, ci95=[-0.205, -0.048], sign_p=0.1460, insufficient: 95% CI includes zero or sign test p >= 0.05
- V3_minus_NC on `provenance_completeness_rate`: n=12, mean_diff=-0.182, ci95=[-0.286, -0.073], sign_p=0.0386, sufficient: paired signal passes current threshold
- AUTO_minus_NC on `provenance_completeness_rate`: n=12, mean_diff=-0.125, ci95=[-0.215, -0.029], sign_p=0.1460, insufficient: 95% CI includes zero or sign test p >= 0.05
- V3_minus_V2 on `provenance_completeness_rate`: n=12, mean_diff=-0.051, ci95=[-0.147, 0.049], sign_p=1.0000, insufficient: 95% CI includes zero or sign test p >= 0.05
- AUTO_minus_V2 on `provenance_completeness_rate`: n=12, mean_diff=0.006, ci95=[-0.079, 0.094], sign_p=1.0000, insufficient: 95% CI includes zero or sign test p >= 0.05
- AUTO_minus_V3 on `provenance_completeness_rate`: n=12, mean_diff=0.057, ci95=[-0.020, 0.136], sign_p=0.3877, insufficient: 95% CI includes zero or sign test p >= 0.05

## Current Stopping Rule

A comparison is treated as statistically sufficient only when it has at least 12 paired cells,
a 95% bootstrap CI that excludes zero, and an exact sign-test p value below 0.05.
If these conditions are not met, the next action is to add more paired runs or narrow the
claim to descriptive evidence.
