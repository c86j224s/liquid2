# G0-Only Slice Decision Memo

This is a follow-up slice of the 9-block controller/generator/MCP isolation experiment.
It fixes report generation to `G0`, meaning the same investigation provider session writes the final report.
This memo does not re-open `G1` as a product default.

## Product Reading

- Separate report generation remains rejected as a default path by the full experiment decision memo.
- Under same-session generation, controller and MCP-surface effects remain unresolved in this data.
- This slice must not be used to claim that controllers are useless or that MCP research surface is unnecessary.
- The next experiment should directly test adaptive controller quality and MCP random-seek surface with `G0` fixed.

## Effects

### Controller C1-C0

| Metric | n | mean diff | 95% CI | sign p | decision |
| --- | ---: | ---: | --- | ---: | --- |
| readability | 18 | 0.0000 | [0.0000, 0.0000] | n/a | inconclusive |
| depth | 18 | 0.1667 | [-0.1667, 0.5000] | 0.5078 | inconclusive |
| breadth | 18 | 0.0556 | [-0.1111, 0.2222] | 1.0000 | inconclusive |
| source_groundedness | 18 | -0.0028 | [-0.0117, 0.0067] | 0.6072 | inconclusive |
| overclaim_rate | 18 | -0.0033 | [-0.0139, 0.0078] | 0.7905 | inconclusive |
| unsupported_rate | 18 | -0.0000 | [-0.0106, 0.0111] | 1.0000 | inconclusive |
| unverifiable_rate | 18 | 0.0028 | [-0.0044, 0.0106] | 0.7905 | inconclusive |
| provenance_completeness | 18 | 0.0017 | [-0.0144, 0.0189] | 1.0000 | inconclusive |
| internal_leakage | 18 | 0.0000 | [0.0000, 0.0000] | n/a | inconclusive |

### MCP Surface M1-M0

| Metric | n | mean diff | 95% CI | sign p | decision |
| --- | ---: | ---: | --- | ---: | --- |
| readability | 18 | 0.0000 | [0.0000, 0.0000] | n/a | inconclusive |
| depth | 18 | 0.0556 | [-0.1667, 0.2778] | 1.0000 | inconclusive |
| breadth | 18 | -0.0556 | [-0.2222, 0.1111] | 1.0000 | inconclusive |
| source_groundedness | 18 | 0.0039 | [-0.0061, 0.0144] | 0.4545 | inconclusive |
| overclaim_rate | 18 | 0.0033 | [-0.0078, 0.0139] | 1.0000 | inconclusive |
| unsupported_rate | 18 | -0.0067 | [-0.0189, 0.0044] | 0.6072 | inconclusive |
| unverifiable_rate | 18 | -0.0050 | [-0.0128, 0.0039] | 0.0923 | inconclusive |
| provenance_completeness | 18 | 0.0139 | [-0.0061, 0.0344] | 0.4240 | inconclusive |
| internal_leakage | 18 | 0.0000 | [0.0000, 0.0000] | n/a | inconclusive |

## Next Step

Run two narrower `G0`-fixed experiments instead of extending this mixed factorial:

1. Controller quality: hold MCP surface fixed and compare no-controller against an adaptive, response-reading controller.
2. MCP random-seek surface: hold controller behavior fixed and compare source-only access against the C1 research surface.
