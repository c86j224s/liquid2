# Experiment 18: Long-Form Finalization MCP

This issue #110 successor isolates one product change: the final long-form
handoff moves from assistant-returned JSON to
`plasma.report.long_form.finalize`. Both arms already use the report-plan MCP
boundary, so plan generation, section and part generation, H5, designed HTML,
and public report artifacts remain paired conditions.

The experiment uses the existing experiment 17 subprocess, isolation, blind
packet, Codex judge, and statistics helpers through one thin controller. Raw
runs, provider state, databases, logs, mappings, and scores remain in the local
experiment 18 archive. See [protocol.md](protocol.md) for the frozen design.

The corrected smoke passed and exactly 24 quality runs were executed. One
candidate run was retained as a post-boundary failure, so the machine gate
failed. The controller then stopped before confirmatory statistics because its
analysis path omitted the existing intention-to-treat assembly call. See
[analysis-summary.md](analysis-summary.md) and
[decision-memo.md](decision-memo.md). The experiment is closed without adoption.
