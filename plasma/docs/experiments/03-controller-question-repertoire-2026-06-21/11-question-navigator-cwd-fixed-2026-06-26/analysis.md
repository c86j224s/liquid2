# NAV investigation-controller experiment analysis

This analysis is machine-scored and investigation-focused. It is not a final product decision by itself.

## Variant summary

| Variant | Runs | Mean score | Median score | Mean sources | Mean tools | Mean coverage | Mean repetition |
|---|---:|---:|---:|---:|---:|---:|---:|
| C0 | 15 | 97.17 | 98.73 | 26.07 | 123.87 | 7.53 | 0.171 |
| NAV | 15 | 93.43 | 97.07 | 25.53 | 112.13 | 7.07 | 0.204 |
| PAL2 | 15 | 96.20 | 98.41 | 25.67 | 98.40 | 7.47 | 0.177 |

## Paired comparisons

### NAV minus C0

- paired blocks: 15
- wins: 3/15
- mean diff: -3.74
- median diff: -2.93
- bootstrap 95% CI for mean diff: [-7.32, -0.53]
- sign-test p-value: 0.0352
- block diffs: M3-0001:1.89, M3-0002:-6.29, M3-0003:-1.14, M3-0004:-11.56, M3-0005:-8.26, M5-0001:-1.38, M5-0002:9.68, M5-0003:-7.89, M5-0004:-20.47, M5-0005:-2.93, W1-0001:-1.86, W1-0002:-5.03, W1-0003:-1.66, W1-0004:3.97, W1-0005:-3.23

### NAV minus PAL2

- paired blocks: 15
- wins: 7/15
- mean diff: -2.77
- median diff: -2.53
- bootstrap 95% CI for mean diff: [-5.86, 0.16]
- sign-test p-value: 1.0000
- block diffs: M3-0001:4.40, M3-0002:1.28, M3-0003:0.71, M3-0004:-10.94, M3-0005:1.23, M5-0001:-7.36, M5-0002:0.43, M5-0003:-2.53, M5-0004:-14.78, M5-0005:0.76, W1-0001:-4.08, W1-0002:-2.54, W1-0003:-7.63, W1-0004:8.04, W1-0005:-8.58

### PAL2 minus C0

- paired blocks: 15
- wins: 5/15
- mean diff: -0.97
- median diff: -2.50
- bootstrap 95% CI for mean diff: [-3.56, 1.73]
- sign-test p-value: 0.3018
- block diffs: M3-0001:-2.51, M3-0002:-7.56, M3-0003:-1.85, M3-0004:-0.62, M3-0005:-9.49, M5-0001:5.98, M5-0002:9.25, M5-0003:-5.36, M5-0004:-5.69, M5-0005:-3.69, W1-0001:2.22, W1-0002:-2.50, W1-0003:5.97, W1-0004:-4.06, W1-0005:5.35

## Reading guide

- Treat the score as a screening metric, not an oracle.
- A useful controller should raise source/path coverage and perspective coverage without increasing repetition.
- If score improves but qualitative transcripts show controller fact-smuggling or over-direction, reject the win.
- If machine metrics are mixed, inspect `controller_messages.md`, `turn*-last.md`, and `tool_trace_summary.json` before productizing.
