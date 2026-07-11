# Baseline Usage Summary

Minimum baseline collection completed for 3 replicates on 2026-07-01. Live 6002 was not used. Direct MCP tool-only runs are marked unavailable because no external provider-usage wrapper exists yet.

Token averages are separated by agent-call and run aggregate because multi-call scenarios such as resumed turns and planned reports otherwise look artificially cheap.

| Scenario | Runs | Unavailable runs | Agent calls | Avg per-agent-call input tokens | Avg per-agent-call cached input tokens | Avg per-agent-call total tokens | Avg per-run total tokens | Max per-run total tokens |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| mcp_tool_research_read | 3 | 3 | 0 | 0.0 | 0.0 | 0.0 | 0.0 | 0 |
| web_report_planned | 3 | 0 | 9 | 292088.1 | 207658.7 | 299102.2 | 897306.7 | 999789 |
| web_turn_new | 3 | 0 | 3 | 82887.0 | 51712.0 | 84237.7 | 84237.7 | 93749 |
| web_turn_resumed | 3 | 0 | 6 | 106196.8 | 75200.0 | 107761.0 | 215522.0 | 217614 |
| web_workflow_current | 3 | 0 | 3 | 134679.7 | 87466.7 | 138034.0 | 138034.0 | 160294 |
