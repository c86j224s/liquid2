# Plasma Token-Diet Measurement Experiment - 2026-07-01

This experiment measures Plasma token usage before any product token-diet change.

The first phase is measurement only. It does not shrink prompts, reduce source read caps,
change model/effort, change compaction policy, enable report fork by default, or claim MCP
savings.

## Status

- Plan: `.fleet/plans/plasma-token-diet-measurement-experiment.md`
- Harness: `plasma/scripts/experiments/token-diet-measurement/run_experiment.py`
- Current implemented phases: isolated no-agent harness smoke, minimum baseline HTTP/API runner,
  report fork preflight, report-isolation smoke gate, and report-isolation parallel ramp.
- Live 6002 server: not used
- Phase 1 closeout: complete as of 2026-07-02. The phase 1 outcome is
  instrumentation plus report-session isolation, not a general token-reduction
  solution.

## Layout

- `runs/`: per-run manifests and redacted artifacts.
- `analysis/`: generated summaries after baseline runs.
- `local-redacted/`: gitignored opt-in excerpts for local debugging only.

Committed `ledger_events.jsonl` files are normalized/redacted exports, not raw ledger dumps.

## Current Minimum Baseline Command

```bash
python3 plasma/scripts/experiments/token-diet-measurement/run_experiment.py --phase minimum-baseline --scenario minimum --agent codex --repeats 1 --force
```

Increase `--repeats` to 3 after the first minimum-baseline pass is clean.

## Report Isolation Preflight

```bash
python3 plasma/scripts/experiments/token-diet-measurement/run_experiment.py --phase report-fork-preflight --force
python3 plasma/scripts/experiments/token-diet-measurement/run_experiment.py --phase report-isolation-smoke --force
```

R1 is available only when the pre-report research session id, report session id,
fork source session id, and post-report research session id are all observable, and
post-report research resumes the pre-report session. If any identity is missing, the
harness records `R1_unavailable` and does not run a fresh-session handoff as a substitute.

`isolated_fork` is now a product capability. Non-one-take report requests use it
automatically when a fork-capable executor and pre-report research session are available;
the harness still sends the policy explicitly to keep R0/R1 comparisons pinned.

## Report Isolation Parallel Ramp

```bash
python3 plasma/scripts/experiments/token-diet-measurement/run_experiment.py --phase report-isolation-parallel --agent codex --repeats 2 --jobs 2 --replicate-id 101
python3 plasma/scripts/experiments/token-diet-measurement/run_experiment.py --phase report-isolation-parallel --agent codex --repeats 6 --jobs 6 --replicate-id 201
```

The parallel ramp runs R0 and R1 as paired fixture blocks. Each variant uses the same scripted
mission fixture, source corpus, prompts, report mode, and clean isolated runtime, but it gets a
separate mission id and DB so the variants cannot contaminate each other's session history.

Treat this ramp as isolation and throughput evidence first. It is not, by itself, a final primary
product-effect measurement until a separate paired-analysis gate confirms the comparison design.

Latest ramp summary: `analysis/report_isolation_parallel_summary.md`.

## Phase 1 Closeout Notes

The product now uses isolated report forks for non-one-take report generation
when the executor can fork from the active research session. The manual Gemma4
product measurement on the live 6002 server confirmed the same shape that the
harness was designed to protect:

- research/autonomous investigation session: `019f2096...`
- long-form Markdown report fork: `b5c9e863...`
- designed HTML export session: `019f2129...`
- post-report research turn: resumed `019f2096...`

The measurement supports the phase 1 claim that report generation no longer
inflates later research turns with report planning, section drafting, part
assembly, or framing context. It does not prove a reduction in the cost of report
generation itself.

Phase 2 should be opened separately. The next target is research-session growth:
MCP savings, ledger-reconstructed fresh sessions, conversation memory policy,
compaction, and budget/stop-condition visibility.
