# Report Plan MCP Experiment - 2026-07-13

This experiment is closed as an invalid smoke for the combined planned and
long-form claim. Its raw records remain immutable in the local archive.

Prepare succeeded and the two-worker product-path smoke ran four reports. Both
planned reports completed, including the candidate MCP submission with no
fallback or binding violation. Both long-form reports failed before report
creation because the harness sent `agent_reasoning_effort=high` to Claude,
which does not support that option. Development and release state remained
unchanged.

```text
~/research-artifacts/liquid2/plasma/experiments/15-report-plan-mcp-2026-07-13/
```

Because mission creation was the preregistered start boundary, the failed
long-form runs are retained as intention-to-treat failures and are not rerun or
combined with later results. This experiment makes no quality or combined-mode
adoption claim. The focused successor is experiment 16.

See [`protocol.md`](protocol.md) for the original frozen design. Raw sources,
reports, prompts, ledgers, provider state, and session identifiers are not
committed.
