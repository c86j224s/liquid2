# Plasma Artifact Archive

Plasma experiments generate large and sometimes sensitive local artifacts:
runtime databases, screenshots, generated HTML files, prompt packets, session
ids, raw ledgers, and temporary source copies. These files are useful for
iteration, but they are not source code and should not be committed to this
repository by default.

## Default Root

The local archive root is:

```text
~/research-artifacts/liquid2/plasma
```

The development browser now defaults to:

```text
~/research-artifacts/liquid2/plasma/runtime/dev-6002/plasma-ui-user.db
```

Local source roots are intentionally not configured by default. On a machine
that has copied source corpora, opt in with `PLASMA_LOCAL_SOURCE_ROOTS` or the
product-specific browser script override:

```text
fleet=~/research-artifacts/liquid2/plasma/local-sources/fleet-harness
```

## Repository Boundary

Keep these in Git:

- product code and tests
- reusable experiment harness code
- experiment protocols, summaries, decision memos, and small fixture manifests
- redacted aggregate metrics that are needed to understand a product decision

Keep these outside Git under the archive root:

- runtime databases and SQLite WAL/SHM files
- generated report outputs, screenshots, thumbnails, and browser captures
- raw run directories, prompt packets, judge packets, scorer packets, and logs
- copied external repositories or local source snapshots
- session ids, unredacted ledgers, and local agent state

## Current Archive Layout

```text
~/research-artifacts/liquid2/plasma/
  experiments/
    03-controller-question-repertoire-2026-06-21/
    media-collection-2026-06-26/
    05-media-inspect-2026-06-26/
    media-self-contained-2026-06-26/
    06-workflow-step-instruction-2026-06-30/
    07-token-diet-measurement-2026-07-01/
    09-design-skill-rendering-2026-07-05/
  local-sources/
    fleet-harness/
  runtime/
    dev-6002/
  tmp-review/
```

`tmp-review/` is a holding area for artifacts recovered from `/tmp` whose
long-term value is not yet decided. Promote useful summaries back into docs,
move raw artifacts into the relevant experiment archive, or delete them after
review.

## Operational Notes

Use `plasma/scripts/dev-browser.sh` to start, stop, and inspect the 6002
development browser. The script owns the default database path, but local
source roots are machine-local allowlists and must be configured explicitly
when needed.

If an experiment needs committed fixtures, commit the smallest redacted fixture
that reproduces the behavior. Do not commit whole run directories just because
they are convenient to inspect locally.
