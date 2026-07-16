# Experiment 20 Analysis Summary

The corrected smoke executed the frozen baseline and candidate `8a054e6` once
each through the real serve, Web report API, Codex, MCP, and artifact-download
path. Both cells passed. The baseline made no long-form finalizer call; the
candidate produced one successful finalizer call with matching canonical event
and exact acknowledgement.

After smoke passed, the candidate alone ran once on each of the 12 frozen
long-form topics. All 12 completed and all 12 passed the source-read, plan and
session lineage, finalizer, acknowledgement, canonical artifact, provenance,
download hash, and isolation checks. Every cell recorded one experiment attempt;
there were no replacements, extra replicates, baseline quality cells, or judge
runs.

The operational gate passed at 12 of 12. Raw reports, prompts, provider state,
session identifiers, logs, databases, and ledger payloads remain outside Git.
