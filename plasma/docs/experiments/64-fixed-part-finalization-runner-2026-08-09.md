# Fixed Part Finalization Runner

This note documents the issue #111 follow-up runner for replaying a fixed,
reviewed Part fixture through the current product V3 finalization tail.

## Purpose

The runner is a narrow development tool. It verifies archive-local reviewed
Part bytes, seeds a fresh Plasma SQLite database with product-shaped mission,
pending, plan, Part artifact, and ledger events, then calls
`reportworkflow.NewRunner(...).FinalizeLongFormPrefix` with
`FinalTailV3`.

It is not a general experiment platform. It does not add product Web, HTTP,
MCP, schema, prompt, retry, session, provider, judge, scheduler, or corpus
management behavior.

Pure input validation happens before any run directory, SQLite database, or
provider request is created. This includes repository/archive boundary checks,
fixture JSON decoding, Part hashes, writing contract checks, requirements
construction, current product guidance SHA validation, and confirmation that the
fixture guidance profile selects the current V3 final edit pipeline.

## Archive Boundary

Place fixture JSON, Part Markdown files, run databases, reports, ledgers, and
manifests outside the repository under the local archive root, for example:

```text
~/research-artifacts/liquid2/plasma/experiments/64-fixed-part-finalization-runner/
```

Do not commit raw fixtures or run directories. Public docs should contain only
redacted protocols, summaries, and small aggregate receipts.

The command must be launched from inside the repository worktree so the
repository root is explicit. If the repository root cannot be found, the command
fails before creating archive directories. If the archive root does not exist
yet, the runner resolves the nearest existing ancestor through symlinks before
creating it and rejects any path that would land inside the repository.

## Fixture Contract

Fixture JSON uses schema `plasma.reportexperiment.fixed_parts.v1` and must
include:

- fixture ID and source provenance ID/product commit.
- report title, rigor level/label, direction hint, and writing contract.
- an optional `generation_guidance_profile`; its resolved product profile must
  select the V3 final edit pipeline.
- `generation_guidance_sha256`, when present, matching the current product
  guidance selector output.
- `post_report_humanize` set exactly to `enabled` or `disabled`.
- ordered Parts with `index`, `title`, archive-local `path`, and SHA-256.

Part files are read relative to the fixture directory when paths are not
absolute. The runner canonicalizes archive, repository, fixture, and Part paths
through symlink resolution; rejects fixture or Part paths outside the archive
root or inside the repository; and verifies Part order and SHA-256 before
seeding. `direction_hint` may be an empty string because the product treats it
as an optional report direction. Extra JSON tokens after the fixture object,
empty Part files, missing writing contracts, and whitespace-padded
`post_report_humanize` values are rejected before provider execution.

## Command

Build the command as a separate development binary:

```bash
cd plasma
go build -o /tmp/plasma-report-experiment ./cmd/plasma-report-experiment
go build -o /tmp/plasma ./cmd/plasma
```

Run it with explicit archive and provider settings:

```bash
/tmp/plasma-report-experiment \
  -archive-root "$HOME/research-artifacts/liquid2/plasma/experiments/64-fixed-part-finalization-runner" \
  -fixture "$HOME/research-artifacts/liquid2/plasma/experiments/64-fixed-part-finalization-runner/fixtures/example/fixture.json" \
  -run-id example-current \
  -plasma-mcp-binary /tmp/plasma \
  -codex-command codex \
  -codex-model gpt-5 \
  -codex-effort high \
  -codex-timeout 20m
```

The command canonicalizes the Plasma MCP binary and Codex command to absolute
regular executable paths before execution. It hashes those exact executable
paths and passes the hashed Plasma MCP path to the product `CodexExecutor`.
Embedded VCS revision metadata is read from the experiment and Plasma MCP
binaries; if both revisions are known and differ, the command fails before
running the product workflow. Codex is identified by executable path and SHA-256.

Model and reasoning effort are normalized with the product session resolver
before any provider request. The normalized values are reused for seed events,
actual `AgentRequest` values, and the manifest.

The runner uses the product `CodexExecutor` default sanitized environment and
the currently authenticated Codex home. A fresh tools-disabled bootstrap session
is created for each run before seeding. Only that bootstrap request opts in to
`--ignore-user-config`; the V3 finalization requests keep normal product Codex
configuration behavior. The bootstrap session ID is recorded as
`ReportPlanSessionID`; `PreReportResearchSessionID` and
`ForkSourceAgentSessionID` remain empty in the seeded prefix. This is an
intentional finalization-only harness difference; it does not replace the
product planning stage.

## Outputs

Each run creates a new archive directory:

```text
<archive-root>/runs/<run-id>/
  plasma.db
  report.md
  ledger.json
  manifest.json
  provider/work/   # Codex process work directory, not the Codex session store.
```

`manifest.json` records fixture IDs and hashes, source provenance, runner scope
`reportworkflow.finalize_long_form_prefix`, binary hashes and VCS metadata,
Codex executable path/SHA-256, normalized model/effort, the bootstrap prompt hash
and returned provider session ID, seeded session lineage, actual content-free
node observations, actual prompt SHA-256 and tool allowlist receipts from
`AgentRequest`, and final report/ledger hashes. Source provenance is limited to
provenance ID, product commit, and source ID; free-form notes are not recorded.
It does not record prompt bodies, provider raw responses, fixture absolute paths,
or Part absolute paths. The manifest is written only after the successful SQLite
close on the success path, so a close failure leaves the run without a manifest
receipt.

Codex session files are not archived under the run directory. They remain in the
product Codex home selected by the sanitized default environment so real
`ForkSession` behavior matches the product adapter.

## Verified Smoke Run

One authenticated Codex smoke run completed on 2026-08-09 with one reviewed
Part, the product default long-form narrative profile, model `gpt-5.5`, medium
reasoning effort, and post-report humanization enabled.

The actual product observer recorded the following successful one-attempt path:

```text
reportassembly -> finalwrite -> readeredit -> styleedit
               -> semanticcheck -> evidencecheck -> finalstore
```

The run made six provider requests including the tools-disabled bootstrap.
Every request after bootstrap used the current product stage prompt and MCP tool
allowlist. All seven workflow nodes reached a non-failed terminal observation.
The run took about 17 minutes 32 seconds; most of the time was provider latency,
with the evidence gate accounting for about 7 minutes 31 seconds.

The final report preserved the fixed Part's scope boundaries and all three
fixture evidence markers. SQLite `PRAGMA integrity_check` returned `ok`, and the
report and ledger SHA-256 values matched the compact manifest. Raw inputs,
outputs, session identifiers, executable paths, and detailed receipts remain in
the local archive and are not committed.
