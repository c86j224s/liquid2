# Decision Memo

Complete clean judged blocks: 9.

## Plain-Language Reading

No supported positive main effect was found for controller, generator session separation, or added research surface.

Supported harm signals:
- generator:depth: estimate -0.2500, wins=0/6; two_sided_p=0.0312.

Interpretation: this analysis argues against assuming that more generated context improves final reports. It does not reject controller-led steering as a product direction; it says controller quality and generated-context exposure need to be tested separately.

## Product Interpretation

The separate report-generator session (`G1`) should not be adopted as the default report path from this experiment.

`generator:depth` produced a supported harm signal: estimate -0.2500, CI [-0.4167, -0.1111], wins=0/6; two_sided_p=0.0312. This points to depth loss from separating the final report session, not to a provenance-risk increase.

Controller and MCP-surface factors remain unresolved. The data does not support adopting them as product defaults, but it also does not prove them ineffective.

## Main Effects

### controller
- readability: estimate -0.0278, CI [-0.0833, 0.0000], wins=0/1; two_sided_p=1.0000, decision inconclusive.
- depth: estimate 0.1389, CI [-0.0833, 0.4167], wins=4/7; two_sided_p=1.0000, decision inconclusive.
- breadth: estimate -0.0833, CI [-0.2778, 0.0833], wins=2/5; two_sided_p=1.0000, decision inconclusive.
- source_groundedness: estimate -0.0033, CI [-0.0086, 0.0014], wins=2/7; two_sided_p=0.4531, decision inconclusive.
- overclaim_rate: estimate 0.0019, CI [-0.0058, 0.0103], wins=6/9; two_sided_p=0.5078, decision inconclusive.
- unsupported_rate: estimate 0.0039, CI [-0.0017, 0.0083], wins=2/9; two_sided_p=0.1797, decision inconclusive.
- unverifiable_rate: estimate 0.0036, CI [-0.0011, 0.0094], wins=3/6; two_sided_p=1.0000, decision inconclusive.
- provenance_completeness: estimate 0.0025, CI [-0.0092, 0.0144], wins=5/8; two_sided_p=0.7266, decision inconclusive.
- internal_leakage: estimate 0.0000, CI [0.0000, 0.0000], wins=0/0; two_sided_p=1.0000, decision inconclusive.

### generator
- readability: estimate -0.0278, CI [-0.0833, 0.0000], wins=0/1; two_sided_p=1.0000, decision inconclusive.
- depth: estimate -0.2500, CI [-0.4167, -0.1111], wins=0/6; two_sided_p=0.0312, decision harm.
- breadth: estimate -0.2500, CI [-0.4444, -0.0833], wins=0/5; two_sided_p=0.0625, decision inconclusive.
- source_groundedness: estimate -0.0011, CI [-0.0106, 0.0089], wins=4/9; two_sided_p=1.0000, decision inconclusive.
- overclaim_rate: estimate 0.0019, CI [-0.0100, 0.0144], wins=5/8; two_sided_p=0.7266, decision inconclusive.
- unsupported_rate: estimate 0.0000, CI [-0.0108, 0.0119], wins=4/8; two_sided_p=1.0000, decision inconclusive.
- unverifiable_rate: estimate -0.0025, CI [-0.0086, 0.0044], wins=5/8; two_sided_p=0.7266, decision inconclusive.
- provenance_completeness: estimate -0.0086, CI [-0.0167, -0.0006], wins=2/7; two_sided_p=0.4531, decision inconclusive.
- internal_leakage: estimate 0.0000, CI [0.0000, 0.0000], wins=0/0; two_sided_p=1.0000, decision inconclusive.

### mcp_surface
- readability: estimate -0.0278, CI [-0.0833, 0.0000], wins=0/1; two_sided_p=1.0000, decision inconclusive.
- depth: estimate 0.1944, CI [0.0278, 0.3889], wins=5/6; two_sided_p=0.2188, decision inconclusive.
- breadth: estimate -0.0278, CI [-0.1389, 0.0833], wins=2/5; two_sided_p=1.0000, decision inconclusive.
- source_groundedness: estimate 0.0000, CI [-0.0086, 0.0078], wins=5/7; two_sided_p=0.4531, decision inconclusive.
- overclaim_rate: estimate 0.0025, CI [-0.0050, 0.0106], wins=5/9; two_sided_p=1.0000, decision inconclusive.
- unsupported_rate: estimate -0.0028, CI [-0.0111, 0.0061], wins=6/9; two_sided_p=0.5078, decision inconclusive.
- unverifiable_rate: estimate -0.0014, CI [-0.0078, 0.0072], wins=6/8; two_sided_p=0.2891, decision inconclusive.
- provenance_completeness: estimate 0.0142, CI [0.0028, 0.0281], wins=6/7; two_sided_p=0.1250, decision inconclusive.
- internal_leakage: estimate 0.0000, CI [0.0000, 0.0000], wins=0/0; two_sided_p=1.0000, decision inconclusive.
