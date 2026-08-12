# Plasma MCP Tool Guide

URI: `plasma://docs/mcp/tools`

This guide explains how to use the Plasma MCP tool surface without changing the
product boundaries. The current input schemas and output envelopes are the ones
returned by `tools/list`.

## Basic Calling Rules

- Tool names, input schemas, and output envelopes are stable wire contracts. Do
  not rename tools or send a reduced shape that differs from `tools/list`.
- On a mission-bound server, keep tool arguments inside the server's mission and
  session binding.
- Before calling a mutating tool, confirm whether the user request, runner
  binding, and idempotency key are required.
- A tool result's `content` is Plasma's safe result representation. Do not append
  provider responses or local runtime state as if they were new sources.
- Search results, grep matches, and connector results are candidates. They are
  not sources, evidence, or saved knowledge until they are read and judged.
- `plasma.research.grep` is case-insensitive literal substring search. The
  entire query must be contiguous; split separate concepts into separate short
  searches. All non-overlapping matches found within each retrieved candidate
  are returned through the existing cursor and limit pagination.

## Default Research Flow

1. Use `plasma.research.outline` to understand the mission ledger structure and
   retain its `last_sequence`.
2. In a resumed provider session, use `plasma.research.changes` with the last
   confirmed sequence when mission changes need checking. Retain the returned
   `current_sequence`; re-read the outline when `resync_required` is true.
3. Use `plasma.research.list` or `plasma.research.grep` to narrow candidates.
4. Use `plasma.research.read` to inspect the needed object or bounded source
   chunk.
5. Use `plasma.research.references` when source, artifact, observation, or
   ledger-event relationships matter.
6. If original material is worth user review, use
   `plasma.sources.candidates.propose`.

## Source Tools

- `plasma.sources.list`: list active source snapshots for a mission.
- `plasma.sources.read`: read bounded content from an accepted source snapshot or
  live local path reference.
- `plasma.sources.tree`, `plasma.sources.grep`: observe a tree or snippets
  inside an accepted live local path source.
- `plasma.sources.search`: find original-material candidates through mounted
  read-only connectors.
- `plasma.sources.candidates.propose`, `plasma.sources.candidates.read`: record
  and read source candidates before user approval.
- `plasma.local_path.roots`, `plasma.local_path.tree`: browse allowlisted local
  path roots by root id and relative path.

Operator-only source mutation tools are visible only when explicitly enabled for
the server.

## Workflow Tools

`plasma.workflow.start`, `plasma.workflow.status`, and `plasma.workflow.stop`
request, inspect, or stop bounded mission workflow runs. Start queues work for
the current user turn and runner binding; it does not call the provider inside
the MCP tool call.

## Report Tools

Report tools are exposed only on runner-created stage-specific MCP servers. Plan
submit, requirement mapping, part assembly, part edit, long-form finalization,
final edit stages, and patch tools each have different responsibilities. Do not
use one stage's tool to stand in for another stage's policy.

## Mermaid Tool

Call `plasma.mermaid.validate` before showing diagram source to the user.
`ok: true` means the static preflight passed; it is not a browser-render
guarantee.
