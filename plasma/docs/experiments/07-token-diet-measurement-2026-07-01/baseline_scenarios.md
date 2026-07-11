# Baseline Scenarios

## Minimum Baseline

| Scenario | Repeats | Notes |
| --- | ---: | --- |
| `web_turn_new` | 3 | Fresh mission first turn. |
| `web_turn_resumed` | 3 | Same mission's second turn. |
| `web_workflow_current` | 3 | Current workflow mode. |
| `web_report_planned` | 3 | Planned Markdown report. |
| `mcp_tool_research_read` | 3 | Direct MCP tool-only read. |

## Implemented First

`harness_smoke` validates isolated server startup, per-run DB/profile/output, manifest creation,
and redacted ledger export without running an agent.

`minimum-baseline` currently automates `web_turn_new`, `web_turn_resumed`,
`web_workflow_current`, and `web_report_planned` through isolated HTTP/API paths.
`mcp_tool_research_read` is emitted as `unavailable` until a direct MCP wrapper exists.

`web_report_isolated` is emitted as `unavailable` unless R1 preflight proves the complete
report session chain. The harness must not substitute a fresh session for true fork support.
