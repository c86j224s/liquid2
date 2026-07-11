# MCP Random-seek Decision Memo

Decision status: ready-for-formal-look.

Judged rows currently cover 10 mission ids. The pre-registered
minimum formal look is 10 complete clean blocks, so do not make a product
adoption claim before that threshold.

## R1 vs R0

| metric | candidate | n | mean delta | 95% CI | wins | sign p |
| --- | --- | ---: | ---: | --- | ---: | ---: |
| readability | R1 vs R0 | 10 | 0.0000 | [0.0000, 0.0000] | 0/10 | 1.0000 |
| depth | R1 vs R0 | 10 | -0.1000 | [-0.2960, 0.0960] | 0/10 | 1.0000 |
| breadth | R1 vs R0 | 10 | 0.1000 | [-0.0960, 0.2960] | 1/10 | 1.0000 |
| source_groundedness | R1 vs R0 | 10 | -0.0000 | [-0.0083, 0.0083] | 3/10 | 1.0000 |
| provenance_completeness | R1 vs R0 | 10 | -0.0140 | [-0.0250, -0.0030] | 1/10 | 0.1250 |
| overclaim_rate | R1 vs R0 | 10 | 0.0000 | [-0.0065, 0.0065] | 2/10 | 1.0000 |
| unsupported_conclusion_rate | R1 vs R0 | 10 | 0.0010 | [-0.0058, 0.0078] | 3/10 | 1.0000 |
| unverifiable_conclusion_rate | R1 vs R0 | 10 | -0.0010 | [-0.0198, 0.0178] | 3/10 | 1.0000 |
| internal_id_path_leakage | R1 vs R0 | 10 | 0.0000 | [0.0000, 0.0000] | 0/10 | 1.0000 |
| distinct_original_sources_read | R1 vs R0 | 10 | -1.4000 | [-2.4205, -0.3795] | 0/10 | 0.0625 |
| useful_offset_reads | R1 vs R0 | 10 | 41.6000 | [25.4290, 57.7710] | 9/10 | 0.0039 |
| cited_claim_backing_rate | R1 vs R0 | 0 |  |  |  |  |
| outline_use_count | R1 vs R0 | 10 | 3.4000 | [1.2890, 5.5110] | 6/10 | 0.0312 |
| reference_traversal_count | R1 vs R0 | 0 |  |  |  |  |
| dead_end_reads | R1 vs R0 | 0 |  |  |  |  |
| repeated_reads | R1 vs R0 | 0 |  |  |  |  |
| turns_to_first_useful_source | R1 vs R0 | 0 |  |  |  |  |
| bounded_read_violations | R1 vs R0 | 10 | 0.0000 | [0.0000, 0.0000] | 0/10 | 1.0000 |
| generated_artifact_as_source_incidents | R1 vs R0 | 10 | 0.0000 | [0.0000, 0.0000] | 0/10 | 1.0000 |
