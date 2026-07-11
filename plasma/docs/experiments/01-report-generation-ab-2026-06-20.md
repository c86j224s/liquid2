# Plasma Report Generation A/B Experiment - 2026-06-20

This note records the small directional experiment that compared two report
generation approaches on the Oda Nobunaga mission.

## Purpose

The experiment tested whether Plasma should continue improving the current
ledger/AST report path or move toward a conversation-first, MCP-tool-centered
report workflow.

## C1 Cutover Decision

The cutover decision is to make the conversation-first MCP reading path the C1
default. The current ledger/AST report path remains historical machinery for
legacy read-only inspection and explicit experiments, not the default product
loop.

The user goal was not a formal benchmark. The goal was to collect enough
evidence to guide the next product direction.

## Mission

- Mission ID: `mis_20260618144638_2de7d6f3`
- Topic: Oda Nobunaga's life
- Existing context: Plasma conversation turns, collected sources, evidence,
  and generated report attempts from the active development database.

Each run used a copied database from `<experiment-db-path>`; the original
development mission database was not mutated by the experiment.

## Variants

### A: Current Plasma Report Path

A used the browser/report API path:

- `POST /api/missions/:id/reports`
- current report plan + AST draft flow
- report ref validation and export to Markdown

### B: Conversation-First MCP Reading Path

B used a fresh Codex execution with:

- copied mission database
- Plasma MCP tools for source and research reads
- exported conversation transcript and source catalog as local inputs
- a thin instruction to write a readable Korean article from the conversation
  and source material

The B path did not ask the agent to treat internal claim, evidence, or source
IDs as public citations.

## Results

| Pair | A | B | Winner |
|---:|---|---|---|
| 1 | success | success | B |
| 2 | failure | success | B |
| 3 | success | success | B |
| 4 | success | success | B |
| 5 | success | success | B |
| 6 | failure | success | B |

Main observations:

- A success rate: 4/6
- B success rate: 6/6
- A successful report average size: 16,812 bytes
- B successful report average size: 30,155 bytes
- A successful reports leaked internal IDs as citations: average 67.8 internal
  IDs per successful report
- B successful reports leaked internal IDs as citations: 0
- A URL citations: average 0
- B URL citations: average 11.3

A failed twice with out-of-scope source snapshot validation errors:

- `src_20260619195801_4995bc66`
- `src_20260619195802_d58cb60a`

## Blind Pairwise Judge

The four pairs where both A and B produced reports were copied to anonymous
`X.md` / `Y.md` files and judged without exposing the variant label.

| Pair | Blind Winner | Actual Winner |
|---:|---|---|
| 1 | X | B |
| 3 | Y | B |
| 4 | X | B |
| 5 | Y | B |

Including A failures in pairs 2 and 6, B won all 6 pairs.

Using a simple paired sign test with 6 independent non-tie pairs:

- B wins: 6/6
- Two-sided p-value: 0.03125
- One-sided p-value for "B is better than A": 0.015625

This is a small but statistically meaningful directional signal.

## Product Interpretation

B was better because it produced an article-like report rather than a ledger
dump. The difference was not only length. B had better narrative flow, clearer
uncertainty handling, human-readable references, and no public leakage of
internal Plasma IDs.

The result argues for:

- keeping MCP as the primary research interface,
- keeping prompts thin,
- letting the agent read source material through tools at report time,
- treating intermediate records as optional research logs, not as mandatory
  public report scaffolding,
- testing code and technical reports with the same method before changing the
  product architecture.

## Follow-Up Product Case

The later C1 product loop produced a stronger Oda Nobunaga report in normal
browser use after a bounded workflow run had deepened the same provider session.
That case is recorded in
[`04-c1-report-quality-case-2026-06-24.md`](04-c1-report-quality-case-2026-06-24.md).

The important update is that the product-quality signal was not only "B beats A"
in an isolated experiment. A normal C1 run also improved when it kept the same
agent session, let the agent read source material through MCP tools, and used
the preceding investigation as report-writing context.

## Next Experiment Direction

The follow-up experiment tested code and technical analysis reports and is
recorded in
[`02-code-analysis-ab-2026-06-20.md`](02-code-analysis-ab-2026-06-20.md).

The original candidate variants were:

- A: current or bulk-context baseline
- B1: conversation-first flow with basic file `list` / `read` / `grep`
- B2: conversation-first flow with basic file tools plus Serena MCP

The test should use a copied Liquid2 repository under `/tmp`, a fresh agent
session that does not know Liquid2 in advance, and a separate user-emulation
driver that steers the analysis using the user's functional prompting style.

The experiment should record:

- transcript,
- tool calls,
- source files actually read,
- final report,
- judge output,
- steering decisions that improved or failed to improve depth and breadth.
