# Agent Call Schema

The analysis unit is one agent call.

Required normalized fields:

- mission id and event id
- surface and event type
- prompt metrics
- provider usage when available
- previous and returned agent session ids
- duration and failure state
- MCP I/O join state

Direct MCP tool-only runs do not automatically expose provider token usage. They record Plasma-side
MCP I/O and mark provider usage as unavailable unless an explicit external wrapper captures it.
