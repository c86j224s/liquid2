# Generation-Time Tone Analysis Summary

Raw archive:
`~/research-artifacts/liquid2/plasma/experiments/10-generation-time-tone-2026-07-07/`

## Aggregate Results

### Initial Comparison

| Pair | Winner counts | Non-tie n | Exact sign-test note | Hard fails |
|---|---:|---:|---:|---:|
| `G1` vs `B0` | `G1: 10`, `B0: 10` | 20 | no support for `G1` | 0 |
| `G1` vs `H5post` | `G1: 10`, `H5post: 10` | 20 | no support for `G1` | 0 |
| `H5post` vs `B0` | `H5post: 14`, `B0: 1`, `tie: 5` | 15 | `p = 0.00048828125` | 0 |

`G1` was not productizable as-is. It helped some technical samples but compressed
the phone-purchase and history samples.

### G2 Comparison

| Pair | Winner counts | Non-tie n | Exact sign-test note | Hard fails |
|---|---:|---:|---:|---:|
| `G2` vs `B0` | `G2: 18`, `B0: 2` | 20 | `p = 0.00020122528076171875` | 0 |
| `G2` vs `H5post` | `G2: 18`, `H5post: 2` | 20 | `p = 0.00020122528076171875` | 0 |
| `G2` vs `G1` | `G2: 20`, `G1: 0` | 20 | `p = 9.5367431640625e-07` | 0 |

### Axis Breakdown

| Pair | Tone winner counts | Coverage winner counts |
|---|---:|---:|
| `G2` vs `B0` | `B0: 11`, `G2: 7`, `tie: 2` | `G2: 17`, `B0: 2`, `tie: 1` |
| `G2` vs `H5post` | `H5post: 11`, `G2: 7`, `tie: 2` | `G2: 17`, `H5post: 3` |
| `G2` vs `G1` | `G1: 13`, `G2: 2`, `tie: 5` | `G2: 20` |

This is the central finding. `G2` won overall because it preserved more
substance, not because it was the strongest tone pass.

## Size And Coverage Proxy

| Sample | `B0` bytes / blocks | `G1` bytes / blocks | `G2` bytes / blocks | `H5post` bytes / blocks |
|---|---:|---:|---:|---:|
| phone purchase | 10,603 / 46 | 7,463 / 30 | 10,560 / 35 | 10,614 / 46 |
| Sengoku history | 13,045 / 63 | 9,034 / 26 | 16,036 / 42 | 13,152 / 63 |
| OAuth/OIDC | 18,288 / 101 | 26,458 / 127 | 36,981 / 172 | 18,358 / 101 |
| Ollama UI | 17,943 / 62 | 17,071 / 79 | 21,128 / 103 | 18,009 / 62 |

`G2` reduced the unwanted compression seen in `G1`, but it also expanded some
technical reports substantially. Product integration should treat this as a
verbosity-risk area rather than blindly preferring longer output.

## Independent Reviews

Nimitz and Sentinel both converged on the same interpretation:

- `G1` is rejected as a product direction.
- `H5post` remains useful for tone polish.
- `G2` is useful as a generation-stage substance preservation guide.
- The result should not be overclaimed as statistically final product proof,
  because the corpus had four samples and repeated model-judge passes.

Sentinel also noted one blind-hygiene issue: the G2 judge packet directory name
included `g2`. This did not reveal which side was `G2`, but future blind packets
should use variant-neutral paths such as `blind_cases_round2`.

## Product Follow-Up

The next product change should be conservative:

1. Add a compact generation-stage instruction that says the report must not
   improve fluency by dropping concrete conditions, numbers, caveats, source
   distinctions, URLs, code, commands, or procedural details.
2. Keep H5 as a post-generation tone patch.
3. Add or retain guards that detect meaning drift and report over-compression.
4. Run a live-path smoke test for `G2 + H5post` before making it the default.

This should remain a guide, not a hard harness that blocks the agent from
writing a rich report.

