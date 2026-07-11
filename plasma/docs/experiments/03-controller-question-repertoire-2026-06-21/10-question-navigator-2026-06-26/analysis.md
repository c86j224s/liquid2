# NAV investigation-controller experiment analysis

Superseded: this 9-block run is kept as an audit artifact only. It was
replaced by `../11-question-navigator-cwd-fixed-2026-06-26/`, which fixed
working-directory contamination in resumed turns and expanded the experiment to
45 valid runs. Do not use this file for current product decisions.

This analysis is machine-scored and investigation-focused. It is not a final product decision by itself.

## Variant summary

| Variant | Runs | Mean score | Median score | Mean sources | Mean tools | Mean coverage | Mean repetition |
|---|---:|---:|---:|---:|---:|---:|---:|
| C0 | 9 | 117.59 | 118.85 | 75.22 | 263.33 | 7.22 | 0.163 |
| NAV | 9 | 115.93 | 114.33 | 66.00 | 199.78 | 7.11 | 0.177 |
| PAL2 | 9 | 113.08 | 115.24 | 58.56 | 178.44 | 7.11 | 0.140 |

## Paired comparisons

### NAV minus C0

- paired blocks: 9
- wins: 4/9
- mean diff: -1.66
- median diff: -3.57
- bootstrap 95% CI for mean diff: [-7.67, 4.50]
- sign-test p-value: 1.0000
- block diffs: M3-0001:-13.57, M3-0002:13.63, M3-0003:-14.73, M5-0001:-6.74, M5-0002:-3.57, M5-0003:-5.47, W1-0001:8.30, W1-0002:7.02, W1-0003:0.14

### NAV minus PAL2

- paired blocks: 9
- wins: 4/9
- mean diff: 2.85
- median diff: -2.16
- bootstrap 95% CI for mean diff: [-5.61, 12.61]
- sign-test p-value: 1.0000
- block diffs: M3-0001:-16.28, M3-0002:6.91, M3-0003:-6.97, M5-0001:-3.19, M5-0002:-2.16, M5-0003:-6.98, W1-0001:19.29, W1-0002:1.44, W1-0003:33.60

### PAL2 minus C0

- paired blocks: 9
- wins: 4/9
- mean diff: -4.52
- median diff: -1.41
- bootstrap 95% CI for mean diff: [-12.55, 2.06]
- sign-test p-value: 1.0000
- block diffs: M3-0001:2.71, M3-0002:6.71, M3-0003:-7.76, M5-0001:-3.55, M5-0002:-1.41, M5-0003:1.51, W1-0001:-10.99, W1-0002:5.58, W1-0003:-33.46

## Reading guide

- Treat the score as a screening metric, not an oracle.
- A useful controller should raise source/path coverage and perspective coverage without increasing repetition.
- If score improves but qualitative transcripts show controller fact-smuggling or over-direction, reject the win.
- If machine metrics are mixed, inspect `controller_messages.md`, `turn*-last.md`, and `tool_trace_summary.json` before productizing.
