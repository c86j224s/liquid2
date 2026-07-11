# G0 Controller/MCP Follow-up Experiment

Status: G0 follow-up artifacts exist; the report-prompt fork follow-up was
completed and judged on 2026-06-23.

This experiment follows the G0-fixed boundary after the nine-block isolation
decision. G1 separate report generation is not a product path here. Every
primary run requires the same provider session to perform the investigation and
write the final Markdown report artifact.

The work is split into two independent experiments:

- `02-controller-quality`: C0, V2, V3, and AUTO are compared while the MCP surface
  stays fixed to the source-tool baseline.
- `03-mcp-random-seek`: R0 and R1 are compared while the controller is disabled.

Sources are original materials under `<experiment-source-root>`. Controller questions,
agent answers, tool traces, reports, and judge packets are results or artifacts,
not sources.

`01-report-prompt-followup-2026-06-23` reuses completed investigation artifacts to
compare final-report prompt variants in parallel. Its default path is
transcript/source-copy based; it also supports official Codex `fork` mode for
product-shaped parallel A/B runs. Plain `resume` spot checks are limited to one
prompt variant per source session because `resume` continues the same session.
The 2026-06-23 fork run generated and judged 72 reports. The main signal was
that richer report guidance improves article-like output, but internal label and
path suppression still has to be part of the product prompt.
