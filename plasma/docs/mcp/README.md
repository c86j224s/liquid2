# Plasma MCP Documentation

[Korean](README.ko.md)

This directory is the human-readable repository and Pages surface for Plasma MCP
guidance.

The runtime MCP resources are code-owned copies embedded by
`plasma/internal/mcpdocs`. Those embedded copies are the only files served by
`resources/read`. The English files in this directory must stay byte-for-byte
equal to the embedded runtime copies; package tests fail clearly if either side
drifts. Korean counterparts are provided for humans only and are not added to
the MCP resource catalog.

These documents contain only public-safe static guidance. They must not contain
mission data, source bodies, session identifiers, ledger contents, provider
responses, credentials, private URLs, or runtime state.

## MCP Resource URIs

| Resource URI | English canonical | Korean counterpart |
| --- | --- | --- |
| `plasma://docs/mcp/tools` | [tools.md](tools.md) | [tools.ko.md](tools.ko.md) |
| `plasma://docs/mcp/errors` | [errors.md](errors.md) | [errors.ko.md](errors.ko.md) |
| `plasma://docs/mcp/reporting` | [reporting.md](reporting.md) | [reporting.ko.md](reporting.ko.md) |
| `plasma://docs/mcp/sources` | [sources.md](sources.md) | [sources.ko.md](sources.ko.md) |
| `plasma://docs/mcp/mermaid` | [mermaid.md](mermaid.md) | [mermaid.ko.md](mermaid.ko.md) |

Use `resources/list` to discover the resource set and `resources/read` to fetch
one English Markdown document by URI.
