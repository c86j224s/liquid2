# Experiment 18 Protocol

## Frozen Comparison

- Baseline product: `1b6239805f2dde41f7aaab36d8025812623da5a6`.
- Candidate product: `4bc3ac07fab93f31d9447c0a83802f6628bd9623`.
- Controller: the clean commit containing this protocol and the experiment 18
  controller; `prepare` records its exact SHA before any provider run.
- Executor: Codex only, with one explicit nonblank model and `high` effort.
- Mode: `long_form` only, `same_session`, mission sources only.
- Corpus: the 12 unique long-form topics selected by experiment 17's immutable
  focused schedule. `prepare` checks its fixture/license hashes and makes
  independent regular-file copies in the experiment 18 archive.
- Schedule: one baseline and one candidate per topic, with experiment 17's 6:6
  first-arm counterbalance. There are no planned or Claude cells.

## Execution

The only action order is `prepare -> smoke -> quality -> packets -> judge ->
analyze`. `preflight --dry-run` is non-mutating. Smoke runs one frozen fixture
through both arms with exactly two workers. Quality starts only after smoke and
runs exactly 12 topics by 2 arms, with at most six workers. Every product run
uses the public CLI, Web report API, isolated loopback connector, isolated
database/provider home, and built Plasma MCP stdio path reused from experiment
17. There is no experiment-level fault, recovery, replay, or deterministic-test
subsystem.

Candidate machine evidence requires one canonical report artifact, one or two
successful finalizer traces linked to that artifact event, matching tool-session
provenance, and the safe runtime confirmation that canonical creation and the
exact `REPORT_FINALIZED` sentinel both succeeded. Baseline requires zero
finalizer calls. Both arms must pass the existing source-read, plan-MCP,
provider-session, artifact, and isolation audit.

## Quality Decision

Blind packets and scoring reuse experiment 17's focused rubric, adapter schema,
Codex subprocess judge, private arm mapping, and score aggregation. Only the
nine final-report dimensions enter this decision. The primary endpoint is the
topic-paired candidate-minus-baseline final composite. Noninferiority passes
when its 10,000-draw one-sided 95% bootstrap lower bound is at least `-0.25`.
The completeness guardrail requires mean difference at least `-0.50` and
candidate low-score-rate increase no greater than `0.10`. Exact sign and paired
Wilcoxon results are reported as sensitivity checks.

Adoption requires the frozen product tests, the exact 24-run machine gate,
noninferiority, and the completeness guardrail. Missing runs, artifacts,
lineage, scores, locks, or hashes fail closed. Raw and private evidence stays in
the experiment 18 local archive; experiment 17 is read-only and unchanged.
