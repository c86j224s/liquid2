# Article Real Pilot Contract

Issue: #461

Status: the pre-provider contract validator is implemented. Real source fixtures,
the provider adapter, and real Articles do not exist yet.

## Purpose

This wave turns the no-execution boundary between the synthetic preflight and real
Article generation into code. `plasma-article-real-pilot-validate` validates only
a frozen protocol bundle. It does not call a provider or write product database,
ledger, API, or UI state.

## Validated contract

- exactly nine preassigned M1/M2/M3 × R/E/A run cells;
- a repository-owner Issue #461 digest lock requirement;
- Codex, `gpt-5.6-luna`, `xhigh`, fresh ephemeral sessions, ignored user config,
  and exact MCP replacement;
- no ambient Web or filesystem access;
- a dedicated 0700 archive, 0600 files, one writer, and isolated HOME, provider
  config, workspace, and database;
- no retained raw provider log or provider session;
- a 7,000–11,000-character body band, 8–12 material claims, 2–3 caveats, and at
  least one supported connection per fixture;
- exact source-catalog, dossier, claim-inventory, memory-question, and transfer-task
  digests;
- matching E/A provider, stage sequence, common prompts/tools/schema/renderer,
  budget, and artifact allowlist;
- distinct E/A treatment contracts, with `article_narrative` only on A;
- per-cell and matrix provider-call, duration, source-byte, and token budgets with
  fail-closed unknown usage;
- the primary A/E blind lane, contextual R lane, generation order, three anonymous
  pairs, mapping commitment, and rubric/shell/administration digests;
- no reveal before user judgment and factual-audit locks;
- one semantic attempt, no replacement run or matrix resume, INCONCLUSIVE for any
  unusable cell, and INVALID after a protocol change.

R preserves the current Report product topology, so the validator does not claim
that R has the same stage list as E/A. Only E and A must match at byte-level outside
their treatment contract.

## Still not executed

- public-source collection and canonical snapshots;
- source-grounded dossier, truth-ledger, and source-span receipts;
- provider invocation;
- author, reader, auditor, or repair execution;
- Article IL, Markdown, HTML, or PDF generation;
- blind packet generation or reading judgment.

Command success means only `bundle_validated=true`, `issue_lock_verified=false`,
and `provider_ready=false`. A complete fixture/arm bundle's exact protocol,
fixture, and arm digests must be posted to Issue #461, and a separate live
Issue-lock verifier must verify owner, comment URL/ID, creation time, canonical
body hash, and digests before generation begins.
