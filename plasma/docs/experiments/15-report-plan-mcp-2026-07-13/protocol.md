# Report Plan MCP Preregistered Protocol

## Status and estimand

This protocol is frozen before the first raw run. The estimand is non-inferiority
of the complete candidate boundary against commit
`15cde729f1dca1b6090711a095fdebc713257c6e`: MCP submission, removal of response
JSON plan synthesis, strict long-form validation, and reference validation.
Transport correctness is not inferred from subjective scores; structural,
fault, recovery, and isolation gates are evaluated separately.

## Product path and isolation

Every run uses a separately built product binary. The baseline is exported with
`git archive`; the candidate is exported from the locked feature commit. Source
and binary SHA-256 values and `version` metadata are recorded. Prepare rejects
a candidate equal to the baseline and writes both expected commits and build
hashes to an immutable gate. Every run must use the exact arm commit and binary
hash in that gate; equal arm commits or binary hashes stop the experiment. The product CLI
creates the mission and attaches the same hashed source bundle to both arms. A
built `plasma serve` binds a unique `127.0.0.1:6200-6299` endpoint, and the public
Web API starts and polls the planned or long-form report. Candidate planning
must traverse the agent-spawned `plasma mcp` stdio process. Internal imports,
test hooks, running development or release binaries, and sibling experiment
artifacts are prohibited.

Each run also owns a second unique loopback port for an inert Liquid2 connector
stub. Its explicit URL is passed to initial serve and recovery, so no command
can fall back to development port 6011 or release port 3011. Both the stub and
Plasma health checks complete before the first product mutation. Planned mode
is fixed to Codex and long-form mode to Claude in both arms. The frozen config
contains exactly `models.codex` and `models.claude`, each a non-blank concrete
identifier for that provider. Planned specs select only the Codex identifier;
long-form specs select only the Claude identifier. Missing, blank, legacy
single-model, equal provider values, or additional provider keys stop prepare
and every run action. Preflight verifies this closed distinct association, not
provider-specific name syntax. Codex and Claude validate their selected model
at the authorized two-worker smoke, and either provider rejecting its model is
a smoke hard-gate failure.
Executor, selected concrete model, effort, connector URL/port, and both process
IDs are recorded.

Each immutable run manifest records topic, replicate, arm, mode, commit and
binary hash, model, effort, source policy, token/time budgets, selected session
policy, database/artifact/work paths, port, namespace, effective child
environment, start boundary, terminal state, and result hash. `HOME`, `TMPDIR`,
`CODEX_HOME`, `CLAUDE_CONFIG_DIR`, and every inherited `XDG_*` key are replaced
with run-local paths or explicitly unset. Raw provider homes, development and
release databases, ports 3001/3002/3011/6001/6002/6011, non-loopback addresses,
duplicate namespaces, and paths outside this experiment root fail closed.
Fault seeding applies the same complete case-local mapping, including every
inherited `XDG_*` key, to its CLI, connector, Web server, provider, and MCP
subprocesses before spawn.

Within a topic/replicate/mode pair, baseline and candidate must have identical
executor, selected model, effort, and all other fixed conditions. Planned and
long-form pairs intentionally use different provider/model locks. Recovery,
calibration, pilot, quality, provenance reconstruction, report start requests,
and terminal manifests retain the selected mode-specific model unchanged.

The 24 reserved topic slots cover independent public-document domains: public
health guidance, transport safety, disaster preparedness, consumer finance,
labor statistics, energy efficiency, climate adaptation, accessibility,
cybersecurity guidance, open-source governance, public procurement, and
education policy, with a second slot per domain. At least 12 licensed or clearly
public-domain bundles must be frozen before quality execution. Each manifest
records source URL, license basis, retrieval date, file hash, and a common goal.
Uncertain licensing or fewer than 12 independent bundles stops the experiment;
source corpora remain archive-local.

## Start and failure rules

The controller prepares the run root, connector stub, product server, ports,
and both health checks before the boundary. Product/agent work starts at the
first CLI mission mutation. Build, directory, environment, port, stub, and
health failures before that boundary are pre-run infrastructure failures. They receive a fresh
run root and port at most twice; a third failure is an infrastructure blocker.
All attempts remain recorded. Every failure after the boundary, including a
provider timeout, is an intention-to-treat result and is never erased or rerun.
A missing plan or report receives score 1 for its applicable composite and
artifact presence 0.

The non-quality smoke is `1 topic x 1 replicate x 2 arms x 2 modes = 4 reports`
with exactly two workers. Candidate submissions and canonical events must each
occur exactly once with matching pending, plan hash, and MCP tool session. A
submission's tool-session producer is not provider provenance. After agent
invocation returns, the canonical event must record the actual validated
provider-session lineage. The preregistered MCP fault matrix and crashes after submission, after
canonical promotion, and after the first long-form section are then exercised.
Any fallback, missing/duplicate canonical, lineage/source/ref/recovery/isolation
violation, contamination, or artifact loss is a hard stop.

Before raw execution, a local no-network acceptance starts both modes from the
Web report API. It uses actual CodexExecutor and ClaudeExecutor config generation,
local provider CLI shims, and the built Plasma MCP stdio tool, requires the
exact `PLAN_SUBMITTED` response, and verifies one submitted event followed by
one canonical event with truthful returned provider lineage.

Fault and recovery stages persist immutable gate objects with an explicit
`passed` value and observed evidence. Pilot, sample-size locking, quality,
packet creation, judging, and analysis require every applicable earlier gate;
a missing or failed gate stops the controller.

## Judge calibration and scoring

Before quality data, separate frozen non-experimental fixtures run through the
same public product path. At least 20 packets are reconstructed only from their
immutable manifests, ledgers, artifact hashes, and collected reports, then
scored twice with the fixed judge model and prompt. Every ordinal dimension must
reach quadratic weighted kappa at least 0.70 and at least 90% agreement within
one point. One rubric revision is allowed before experimental data; a second
failure stops the experiment. Repeats are technical repeats, not independent
judges.

Scores are 1-5, higher is better. The equally weighted plan composite contains
depth, breadth, goal preservation, and investigation discipline. The equally
weighted final-report composite contains source safety, coverage, usefulness,
tone, flow, consistency, non-repetition, heading stability, and completeness.
Artifact presence is a machine gate, not a subjective dimension.

Pairs and packets are rebuilt only from immutable terminal manifests, run
gates, collected artifact and ledger hashes, and the controller-owned private
arm mapping. Each blind A/B packet is scored twice. An absolute two-point dimension
disagreement or a difference above 0.75 on either primary composite triggers a
third arm-blind call. Otherwise dimensions are averaged across two calls; with
a third call, the dimension-wise median is used. Transport, commit, session,
path, and arm identity are removed from packets and the private mapping remains
archive-local.

## Sample size and confirmatory analysis

The pilot uses four independent topics, two report replicates, two arms, and two
modes: 32 reports. Topic is the independent paired unit. Judge repeats and the
two report replicates are averaged within topic/mode/arm before one candidate
minus baseline difference is formed.

While arm identity and signed effects remain blinded, the largest dispersion
among the four primary endpoints is used in
`ceil(((1.645 + 0.842) * s / 0.25)^2)`, rounded up to the next multiple of four,
with minimum 12 and maximum 24 topics. A requirement above 24 is a feasibility
failure. The four pilot topics are included. The sample size is locked once;
there is no later topic block, replicate addition, margin change, or rubric
change.

Calibration, sample-size, judge aggregate, and ITT inputs must have the exact
expected mode/arm/replicate and plan/final dimension sets and trace to this
experiment's immutable files. Arbitrary pair, blinded-endpoint, and ITT config
files are rejected. Analysis fails if any expected run, pair, score, gate,
mapping, hash, or lock is absent or inconsistent.

Planned and long-form modes each have final-report composite first and plan
composite second in a fixed sequence. Each one-sided non-inferiority margin is
`-0.25`. A deterministic 10,000-resample topic-paired percentile bootstrap uses
the fifth percentile as the one-sided 95% lower bound. The second endpoint is
confirmatory only if the first passes, and both must pass. Both mode claims must
pass by intersection-union; strength in one mode cannot offset failure in the
other. Exact sign and paired Wilcoxon tests, with Holm adjustment across the
four endpoints within each sensitivity family, are directional sensitivity
analyses only.

Machine gates require zero missing/duplicate canonicals, zero session,
source-read, reference-scope, recovery, and isolation violations, and 100%
artifact presence across both arms. Candidate gates additionally require zero
final-response fallbacks and zero submission/canonical binding violations.
For each mode, goal-preservation and completeness mean differences must be at
least `-0.50`, and the candidate rate of scores at or below 2 may not exceed the
baseline by more than 10 percentage points.

Only the protocol, redacted conclusions, and small aggregate metrics belong in
Git. Raw prompts, packets, reports, HTML/PDF, logs, sources, ledgers, session
identifiers, mappings, and bootstrap samples remain in the fixed local archive.
