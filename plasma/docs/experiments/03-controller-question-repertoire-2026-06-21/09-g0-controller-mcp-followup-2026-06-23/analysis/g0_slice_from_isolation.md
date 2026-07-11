# G0 Slice From Isolation

Source files:

- `plasma/docs/experiments/03-controller-question-repertoire-2026-06-21/02-controller-generator-mcp-isolation-2026-06-22/analysis/score_matrix.csv`
- `plasma/docs/experiments/03-controller-question-repertoire-2026-06-21/02-controller-generator-mcp-isolation-2026-06-22/analysis/decision_memo.md`

Recorded product decision: G1 separate report generation is rejected as the
product default because the nine-block isolation experiment found supported
depth harm. This follow-up therefore fixes final report generation to G0.

G0-only controller slice:

- C1-C0, n=18.
- Depth mean +0.1667 with CI [-0.1667, +0.5000].
- Readability was flat.
- Grounding and provenance were flat.
- Finding: unresolved and hypothesis-setting only.

G0-only MCP slice:

- M1-M0, n=18.
- Provenance +0.0139 and unverifiable -0.0050 were directionally favorable.
- Confidence intervals included zero.
- Finding: unresolved and hypothesis-setting only.

This slice is not powered to adopt or reject controller quality or MCP random
seek usefulness after fixing generator choice.
