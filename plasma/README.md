# Plasma

Plasma is a steerable research workspace.

It lets a user create a mission, attach sources, talk with an agent, let the
agent investigate through tools, and generate report artifacts from the work.
The center of the product is not a terminal. The center is the research loop:
conversation, source reading, investigation, and report generation.

Plasma is a separate product from Liquid2. Liquid2 stores personal reference
material. Plasma may read selected Liquid2 documents through a connector, but it
keeps its own database, mission ledger, sources, conversations, and reports.

## The Basic Workflow

1. Create a mission with a topic, goal, or rough question.
2. Add sources: pasted text, URLs, PDFs, media URLs, Liquid2 documents, or
   allowlisted local files and repositories.
3. Talk with an agent in the mission. The agent continues the same provider
   session where possible, so the conversation remains useful across turns.
4. Let the agent search, read, and inspect sources through MCP tools instead of
   stuffing all source text into prompts.
5. Review source candidates before accepting them as mission sources.
6. Generate reports as artifacts. Markdown is the primary report format; HTML is
   a rendered/exported form.

Automatic investigation is an extension of the same mission workflow. It should
ask better next questions and keep investigating while the user is away, not
create a separate hidden research product.

## Product Rules

- Sources are original materials: URLs, PDFs, files, Liquid2 documents, media
  links, or local path references.
- Agent answers are results. They may cite or recommend sources, but they are
  not sources themselves.
- Reports are output artifacts assembled from the mission work.
- Plasma should use thin guidance plus MCP/source reads. It should not solve
  research quality by pasting large ledgers or source packs into every prompt.
- Browser UI, CLI, MCP tools, agent providers, source readers, and report
  renderers should remain replaceable over the same product state.
- Plasma research state belongs in the Plasma database, not in Liquid2 tables.

## Current Capabilities

- Mission ledger with conversation turns, source events, MCP call logs, and
  report artifacts.
- Browser workspace for mission creation, conversation, source management,
  source candidate review, automatic investigation, and report generation.
- Source snapshots for text and textual URLs.
- PDF source support with metadata-first and chunked reads.
- Image URL snapshots, plus audio/video URL metadata references.
- Allowlisted local path sources for codebase and document analysis.
- Read-only Liquid2 connector boundary.
- Confluence Cloud source intake with API-token connections, site/space/page
  browsing, candidate review, version-pinned snapshots, range snapshots for
  large pages, and update preview/approval.
- MCP research tools for outline, list, grep, read, and reference traversal.
- Codex-backed and Claude-backed agent turns with session resume when available.
- Markdown reports, long-form part/section reports, and HTML exports.
- MCP-backed Markdown report patching that creates a new report artifact from a
  prior report session instead of editing the old artifact in place.

Some areas are still experimental or future work: mixed-provider missions,
background autonomous workers, richer media inspection, external publishing
adapters, stronger source discovery, and more polished designed HTML reports.

## Quick Start For Development

From the workspace root, run Liquid2 and Plasma together:

```sh
./dev-browser.sh start
./dev-browser.sh status
./dev-browser.sh stop
```

Run only Plasma:

```sh
./dev-browser.sh plasma start
./dev-browser.sh plasma status
./dev-browser.sh plasma logs
./dev-browser.sh plasma stop
```

Plasma development defaults to browser port `6002` and a local SQLite database
under `~/research-artifacts/liquid2/plasma/runtime/dev-6002/`.
Runtime settings can be moved from environment variables into TOML files; see
the workspace [configuration guide](../docs/configuration.md).

Run the local release surface:

```sh
./release-browser.sh plasma start
./release-browser.sh plasma status
./release-browser.sh plasma logs
./release-browser.sh plasma stop
```

Plasma release defaults to browser/API port `3002` and
`~/Library/Application Support/Plasma/plasma.db`.

## Common Commands

Run checks:

```sh
make -C plasma check
```

Work from the product directory:

```sh
cd plasma
make check
make dev-browser-start
make dev-browser-status
make dev-browser-logs
make dev-browser-stop
```

Run the browser server manually without an agent:

```sh
cd plasma
go run ./cmd/plasma serve -db /tmp/plasma-ui.db -addr 127.0.0.1:6002
```

Run it with Codex agent execution:

```sh
cd plasma
go run ./cmd/plasma serve \
  -db /tmp/plasma-ui.db \
  -addr 127.0.0.1:6002 \
  -agent codex
```

Plasma starts new Codex sessions with `gpt-5.6-terra` and `medium` reasoning by
default. The mission controls let users select GPT-5.6 Sol, Terra, or Luna and
the supported reasoning effort before starting a new agent session; mission
data and saved sources remain in place while Codex session continuity resets.
Reports without a separate model selection inherit that mission setting or the
same Terra/medium default.

Run it with both Codex and Claude available in the browser:

```sh
cd plasma
go run ./cmd/plasma serve \
  -db /tmp/plasma-ui.db \
  -addr 127.0.0.1:6002 \
  -agent codex,claude \
  -claude-model haiku
```

The browser scripts expose the same configuration through environment
variables:

```sh
# From the workspace root:
PLASMA_DEV_BROWSER_AGENT=codex,claude \
PLASMA_DEV_BROWSER_CLAUDE_MODEL=haiku \
  ./dev-browser.sh plasma restart
```

Use `PLASMA_DEV_BROWSER_CLAUDE` or `PLASMA_RELEASE_BROWSER_CLAUDE` to point at a
specific Claude CLI binary. Use `PLASMA_DEV_BROWSER_CLAUDE_MAX_BUDGET_USD` or
`PLASMA_RELEASE_BROWSER_CLAUDE_MAX_BUDGET_USD` only when you intentionally want
Claude CLI to enforce a per-turn budget.

Agent MCP servers are started with a shared Plasma research-tool allowlist.
Codex can read accepted live local path sources through `plasma.sources.read`
and can inspect accepted local path directories with source-scoped
`plasma.sources.tree` and `plasma.sources.grep`; these tools take a source
`snapshot_id` and optional `subpath`, not arbitrary absolute paths or root-wide
local path browsing. Claude additionally allows its built-in web tools
(`WebFetch`, `WebSearch`) and read-only file tools (`Read`, `Glob`, `Grep`, and
`LS`) inside the configured agent work directory. Shell execution, file edits,
task spawning, and notebook edits stay disabled. Material outside the agent work
directory should be attached through Plasma local source roots so it remains
visible through mission-bound MCP reads.

For now, each mission uses one agent provider type. The first provider-backed
action locks the mission to that provider, so a mission started with Codex
continues with Codex and a mission started with Claude continues with Claude.
Create a new mission when comparing providers.

Configure local source roots for code or document analysis:

```sh
PLASMA_LOCAL_SOURCE_ROOTS=repo=/path/to/repo,docs=/path/to/docs \
  go run ./cmd/plasma serve -db /tmp/plasma-ui.db -addr 127.0.0.1:6002
```

Clients refer to those roots by `root_id` and relative path. Plasma rejects
absolute client paths and does not return the configured absolute server root
through the web, CLI, or MCP surfaces.

## UI-Less Research Flow

Plasma should also be useful without the browser. An agent with the Plasma MCP
server can inspect the same mission ledger and sources:

```sh
cd plasma
go run ./cmd/plasma mcp \
  -db /tmp/plasma-ui.db \
  -mission-id mis_... \
  -agent-session-id ses_...
```

A typical MCP-driven flow is:

- Start with `plasma.research.outline`.
- Use `plasma.research.list` or `plasma.research.grep` to find candidates.
- Use `plasma.research.read` to inspect bounded chunks.
- Use `plasma.research.references` to check relationships before reporting.

Patch an existing Markdown report artifact from the CLI:

```sh
cd plasma
go run ./cmd/plasma reports patch mis_... \
  -db /tmp/plasma-ui.db \
  -base-artifact art_... \
  -instruction "사이토 도산 관련 조사 내용을 반영해 서술을 보강" \
  -wait
```

Report patching uses a temporary MCP tool surface scoped to that patch run. The
agent reads and edits the stored Markdown report artifact through tools, then
finalizes a new report artifact version. The base artifact is kept unchanged.

## Documentation

- [Plasma README Korean](README.ko.md)
- [Documentation Index](docs/README.md)
- [Documentation Index Korean](docs/README.ko.md)
- [Glossary](docs/glossary.md)
- [Glossary Korean](docs/glossary.ko.md)
- [Product Flow](docs/product-flow.md)
- [C1 Default Loop](docs/c1-default-loop.md)
- [C1 Default Loop Korean](docs/c1-default-loop.ko.md)
- [Automatic Investigation](docs/automatic-investigation.md)
- [Product Architecture](docs/product-architecture.md)
- [Product Architecture Korean](docs/product-architecture.ko.md)
- [Media Source Implementation Design](docs/media-source-implementation-design.md)
- [Confluence Cloud Source 연동 기록](docs/confluence-source-integration.md)
- [Confluence live validation checklist](docs/confluence-live-validation-checklist.md)
- [Token Diet Instrumentation](docs/token-diet-instrumentation.md)
- [Evidence Signal Model](docs/evidence-signal-model.md)
- [Evidence Signal Model Korean](docs/evidence-signal-model.ko.md)
- [Experiment Index](docs/experiments/README.md)
