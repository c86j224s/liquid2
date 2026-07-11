# Controller-quality Decision Memo

Decision status: ready-for-formal-look.

Judged rows currently cover 8 mission ids. The pre-registered
minimum formal look is 8 complete clean blocks, so do not make a product
adoption claim before that threshold.

## AUTO vs C0

| metric | candidate | n | mean delta | 95% CI | wins | sign p |
| --- | --- | ---: | ---: | --- | ---: | ---: |
| readability | AUTO vs C0 | 8 | 0.0000 | [0.0000, 0.0000] | 0/8 | 1.0000 |
| depth | AUTO vs C0 | 8 | 0.0000 | [0.0000, 0.0000] | 0/8 | 1.0000 |
| breadth | AUTO vs C0 | 8 | 0.0000 | [-0.3704, 0.3704] | 1/8 | 1.0000 |
| source_groundedness | AUTO vs C0 | 8 | -0.0013 | [-0.0106, 0.0081] | 2/8 | 0.4531 |
| provenance_completeness | AUTO vs C0 | 8 | 0.0000 | [-0.0143, 0.0143] | 4/8 | 0.6875 |
| overclaim_rate | AUTO vs C0 | 8 | 0.0025 | [-0.0078, 0.0128] | 5/8 | 0.4531 |
| unsupported_conclusion_rate | AUTO vs C0 | 8 | 0.0025 | [-0.0056, 0.0106] | 4/8 | 1.0000 |
| unverifiable_conclusion_rate | AUTO vs C0 | 8 | 0.0000 | [-0.0111, 0.0111] | 4/8 | 1.0000 |
| internal_id_path_leakage | AUTO vs C0 | 8 | 0.0000 | [0.0000, 0.0000] | 0/8 | 1.0000 |


## V2 vs C0

| metric | candidate | n | mean delta | 95% CI | wins | sign p |
| --- | --- | ---: | ---: | --- | ---: | ---: |
| readability | V2 vs C0 | 8 | 0.0000 | [0.0000, 0.0000] | 0/8 | 1.0000 |
| depth | V2 vs C0 | 8 | 0.0000 | [0.0000, 0.0000] | 0/8 | 1.0000 |
| breadth | V2 vs C0 | 8 | 0.0000 | [0.0000, 0.0000] | 0/8 | 1.0000 |
| source_groundedness | V2 vs C0 | 8 | -0.0150 | [-0.0261, -0.0039] | 1/8 | 0.1250 |
| provenance_completeness | V2 vs C0 | 8 | 0.0025 | [-0.0264, 0.0314] | 4/8 | 1.0000 |
| overclaim_rate | V2 vs C0 | 8 | -0.0100 | [-0.0234, 0.0034] | 2/8 | 0.4531 |
| unsupported_conclusion_rate | V2 vs C0 | 8 | -0.0112 | [-0.0299, 0.0074] | 2/8 | 1.0000 |
| unverifiable_conclusion_rate | V2 vs C0 | 8 | -0.0138 | [-0.0290, 0.0015] | 1/8 | 0.2188 |
| internal_id_path_leakage | V2 vs C0 | 8 | 0.0000 | [0.0000, 0.0000] | 0/8 | 1.0000 |


## V3 vs C0

| metric | candidate | n | mean delta | 95% CI | wins | sign p |
| --- | --- | ---: | ---: | --- | ---: | ---: |
| readability | V3 vs C0 | 8 | 0.0000 | [0.0000, 0.0000] | 0/8 | 1.0000 |
| depth | V3 vs C0 | 8 | 0.0000 | [0.0000, 0.0000] | 0/8 | 1.0000 |
| breadth | V3 vs C0 | 8 | 0.1250 | [-0.1200, 0.3700] | 1/8 | 1.0000 |
| source_groundedness | V3 vs C0 | 8 | -0.0125 | [-0.0206, -0.0044] | 0/8 | 0.0625 |
| provenance_completeness | V3 vs C0 | 8 | -0.0050 | [-0.0293, 0.0193] | 3/8 | 1.0000 |
| overclaim_rate | V3 vs C0 | 8 | -0.0175 | [-0.0363, 0.0013] | 2/8 | 0.4531 |
| unsupported_conclusion_rate | V3 vs C0 | 8 | -0.0188 | [-0.0355, -0.0020] | 1/8 | 0.1250 |
| unverifiable_conclusion_rate | V3 vs C0 | 8 | -0.0150 | [-0.0320, 0.0020] | 1/8 | 0.2188 |
| internal_id_path_leakage | V3 vs C0 | 8 | 0.0000 | [0.0000, 0.0000] | 0/8 | 1.0000 |


## Controller Question Diagnostics

- V2 controller_question_quality: n=8, mean=5.000
- V2 controller_followup_effect: n=8, mean=5.000
- V2 controller_non_leading: n=8, mean=0.866
- V2 controller_fact_free: n=8, mean=0.879
- V3 controller_question_quality: n=8, mean=5.000
- V3 controller_followup_effect: n=8, mean=5.000
- V3 controller_non_leading: n=8, mean=0.810
- V3 controller_fact_free: n=8, mean=0.839
- AUTO controller_question_quality: n=8, mean=4.750
- AUTO controller_followup_effect: n=8, mean=4.625
- AUTO controller_non_leading: n=8, mean=0.871
- AUTO controller_fact_free: n=8, mean=0.906
