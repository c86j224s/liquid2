# Article IL Pilot Harness

Issue: #461

Status: synthetic archive preflight passed; no real Article has been generated.

## Purpose

This wave implements the archive boundary required before the Article IL reading
pilot. It proves that a frozen three-fixture, three-arm matrix can execute once,
retain success and failure, and publish verifiable terminal receipts without
writing Article state into the Plasma product database, ledger, API, or UI.

It does not test Article quality. The synthetic executor writes deterministic
placeholder Markdown and never starts Codex or another provider.

## Contract

`internal/articleexperiment` owns:

- strict protocol, fixture, arm, and run-cell JSON contracts;
- exactly three fixtures (`M1`, `M2`, `M3`) and three arms (`R`, `E`, `A`);
- nine distinct preassigned run IDs with one-to-one fixture/arm binding;
- exact protocol, fixture, input, arm-manifest, and arm-contract SHA-256 checks;
- repository exclusion, symlink resolution, regular-file requirements, relative
  manifest references, and byte ceilings;
- path-free immutable validated input bytes passed to a consumer-supplied executor;
- a 64 MiB protocol input ceiling, 16 inputs per fixture, and output ceilings of
  16 MiB per artifact plus 16 artifacts/64 MiB per run;
- run pending receipts, safe failed terminals, partial-artifact receipts, and an
  atomic terminal manifest;
- a final matrix receipt bound to the one expected `executor_failed` run and all
  nine run-terminal hashes;
- replay validation that rejects tampering, unreceipted files, duplicate runs,
  incomplete runs, and an extra unregistered run directory.

The package imports only the stable product error class from Plasma internals.
An architecture check prevents it from importing Report policy, Web, MCP,
SQLite, connector, source-reader, or provider implementations.

The synthetic harness trusts the local user account and other local processes not
to replace archive parent directories while a run is open. It prevents static
symlink/traversal escapes and detects replacement with `lstat`/open/`fstat` plus
a before/after executor directory-identity check, but it does not provide
directory-descriptor fencing against a hostile concurrent local filesystem
writer. Runs therefore use a dedicated 0700 archive owned by one protocol with
no concurrent writer. Replay accepts a same-inode terminal temp file left after
hard-link publication as crash residue and rejects a different temp file. This
executor errors, including cancellation, become safe failed terminals so a run
identity is never reused. This trust model must be reconsidered before a
real-provider pilot if the archive can be modified concurrently.

## Synthetic Command

The developer command is deliberately synthetic-only:

```text
go run ./cmd/plasma-article-pilot-preflight \
  --synthetic \
  --archive-root /path/to/issue-461/synthetic-preflight-v4 \
  --repository-root /path/to/liquid2 \
  --protocol /path/to/issue-461/synthetic-preflight-v4/protocol.json \
  --protocol-sha256 <sha256> \
  --fail-run preflight-M2-A
```

`--synthetic` and one preassigned forced-failure run are required. Real provider
execution is not available through this command. Existing receipts can be checked
without executing cells:

```text
go run ./cmd/plasma-article-pilot-preflight \
  --verify \
  --archive-root /path/to/issue-461/synthetic-preflight-v4 \
  --repository-root /path/to/liquid2 \
  --protocol /path/to/issue-461/synthetic-preflight-v4/protocol.json \
  --protocol-sha256 <sha256> \
  --matrix-sha256 <sha256>
```

## Durable Preflight Result

The Issue #461 archive contains one synthetic matrix run with:

- protocol SHA-256:
  `a17a906b7bb4841c7b6398752c178d248399326382a7e9a1bad6e4d624d2af5e`
- matrix SHA-256:
  `fb504b6754d75580162ffd71368770d2e6cf097b75383a510246d9ea58b2946b`
- completed cells: 8
- deliberately failed cells: 1 (`M2/A`)
- run directories: 9
- pending manifests: 9
- terminal manifests: 9
- successful placeholder artifacts: 8

An independent replay matched every matrix terminal digest, run identity, and
status. A scan found no repository path, scratch path, raw forced-provider error,
or session identifier in the durable JSON, Markdown, or text artifacts.

The archive path itself is local operational state and is intentionally not
recorded in this public summary. The default local archive policy places it
under the Issue #461 experiment directory outside Git.

## What This Does Not Prove

This result does not prove:

- that R, E, or A can run a real provider;
- that E and A have matched source, author, reader, factual-audit, and repair
  adapters;
- that Article Narrative improves reading;
- that any manuscript is factually safe or worth reading;
- that the product should add Article events, DB tables, routes, or UI.

The next wave may connect bounded real-provider adapters to this harness. It must
preserve the existing immutable matrix and failure-retention contracts before
any real Article pilot is allowed to start.
