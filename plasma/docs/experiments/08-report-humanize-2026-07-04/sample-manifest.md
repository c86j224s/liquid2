# Sample Manifest

Raw samples and generated candidates are stored outside Git:

```text
~/research-artifacts/liquid2/plasma/experiments/08-report-humanize-2026-07-04/
```

## Samples

| sample | source class | artifact kind | bytes | use in this run |
|---|---|---:|---:|---|
| `s1-short-ai-report` | `dev-6002` Plasma report artifact | Markdown report | 12,945 | H1 candidate generated and reviewed; H2 repair candidate generated |
| `s2-long-gemma4-report` | `dev-6002` Plasma report artifact | Markdown report | 116,954 | H2 long-form candidate generated and structure-audited |
| `s3-section-gemma4-part03-section02` | `dev-6002` Plasma report artifact | Markdown section artifact | 8,192 | H1 candidate generated and reviewed |

## Archive Files

```text
samples/
  s1-short-ai-report.md
  s2-long-gemma4-report.md
  s3-section-gemma4-part03-section02.md
candidates/
  s1-h1.md
  s1-h2.md
  s1-h3.md
  s1-h5-full.md
  s2-h2.md
  s2-h3.md
  s2-h5-full.md
  s3-h1.md
  s3-h3.md
analysis/
  blind_cases.jsonl
  blind_cases_manifest.json
  blind_judge_j1.jsonl
  blind_judge_j2.jsonl
  blind_judge_j3.jsonl
  blind_preference_summary.json
  blind_preference_summary.md
  h3_blind_cases.jsonl
  h3_blind_cases_manifest.json
  h3_blind_judge_j1.jsonl
  h3_blind_judge_j2.jsonl
  h3_blind_judge_j3.jsonl
  h3_blind_judge_j4.jsonl
  h3_blind_judge_j5.jsonl
  h3_blind_preference_summary.json
  h3_blind_preference_summary.md
  h4_selector_cases.jsonl
  h4_selector_s1.jsonl
  h4_selector_s2.jsonl
  h4_selector_s3.jsonl
  h4_selector_exploration_summary.json
  h4_selector_exploration_summary.md
  h5_full_report_blind_cases.jsonl
  h5_full_report_blind_cases_manifest.json
  h5_full_report_blind_judge_j1.jsonl
  h5_full_report_blind_judge_j2.jsonl
  h5_full_report_blind_judge_j3.jsonl
  h5_full_report_blind_judge_j4.jsonl
  h5_full_report_blind_judge_j5.jsonl
  h5_full_report_blind_preference_summary.json
  h5_full_report_blind_preference_summary.md
  h1-review.md
  s1-h1-agent-last-message.txt
  s1-h2-agent-last-message.txt
  s1-h1-structure.json
  s1-h1-structure-v2.json
  s1-h2-structure.json
  s1-h3-structure.json
  s1-h5-full-structure.json
  s2-h2-structure.json
  s2-h3-structure.json
  s2-h5-full-structure.json
  s3-h1-agent-last-message.txt
  s3-h1-structure.json
  s3-h1-structure-v2.json
  s3-h3-structure.json
```

No raw sample or generated candidate is committed to the repository.
