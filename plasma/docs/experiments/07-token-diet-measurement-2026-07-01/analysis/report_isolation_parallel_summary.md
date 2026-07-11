# Report Isolation Parallel Ramp Summary

Date: 2026-07-01

This summary covers the report-session isolation ramp that validated the
`isolated_fork` behavior before it was productized.

## Scope

The ramp validates session-chain isolation, runtime isolation, and parallel execution stability.
It is not a final primary product-effect measurement. R0 and R1 are paired fixture blocks:
same scripted mission fixture, source corpus, prompts, report mode, and clean isolated runtime per
variant, but separate mission ids and DBs to avoid cross-variant session contamination.

## Runs

| Stage | Replicates | Jobs | Result | Notes |
| --- | ---: | ---: | --- | --- |
| Preflight | 301 | 1 | Pass | `R1_available`; all required session identities observed. |
| 2-worker smoke | 101-102 | 2 | Pass | 2 completed blocks, 0 failures. |
| 6-worker first attempt | 201-206 | 6 | Fail | 4 completed blocks, 2 failures. Exposed a port-lock race in the harness. |
| 6-worker retry | 301-306 | 6 | Pass | 6 completed blocks, 0 failures. |

The successful 6-worker retry ran from `2026-07-01T15:59:48Z` to
`2026-07-01T16:11:42Z`. The immediately preceding preflight ran from
`2026-07-01T15:56:06Z` to `2026-07-01T15:59:48Z`.

## Session Integrity

All successful R1 runs in replicates 301-306 satisfied the isolation condition:

- `report_session_id` was distinct from `pre_report_research_session_id`.
- `fork_source_agent_session_id` matched `pre_report_research_session_id`.
- Both post-report turns resumed the pre-report research session.
- No required session-chain identity was missing.

## Token Observations

Post-report research turns are the main signal. They measure whether report generation contaminates
the continued research session with the report-generation context.

| Metric | R0 same-session | R1 isolated-fork | Interpretation |
| --- | ---: | ---: | --- |
| Post-report turn 1 mean input tokens | 657,098 | 153,616 | R1 avoids most report-work context on the next research turn. |
| Post-report turn 2 mean input tokens | 897,569 | 250,061 | The gap persists into the second continued research turn. |
| Two post-report turns mean input tokens | 1,554,668 | 403,677 | R1 used about 26% of R0 input tokens. |
| Whole run mean input tokens | 2,311,475 | 1,204,863 | R1 roughly halved total input tokens in this ramp fixture. |
| Whole run mean duration | 325,784 ms | 323,039 ms | Runtime was similar in this fixture; the win is token isolation, not speed. |

Per-replicate post-report input-token ratios for R1 over R0:

| Replicate | R0 post-report input | R1 post-report input | R1/R0 |
| --- | ---: | ---: | ---: |
| 301 | 1,250,139 | 365,592 | 0.292 |
| 302 | 1,731,007 | 340,960 | 0.197 |
| 303 | 1,695,823 | 459,461 | 0.271 |
| 304 | 1,292,249 | 341,259 | 0.264 |
| 305 | 1,574,264 | 511,639 | 0.325 |
| 306 | 1,784,523 | 403,149 | 0.226 |

## Harness Fixes From The Ramp

The first 6-worker attempt exposed a port-lock race. Stale lock cleanup could race with another
worker creating the same lock, letting `FileExistsError` escape and causing failed blocks. The
lock acquisition helper now retries stale-lock creation races before giving up.

The runner also now:

- reruns R1 preflight at the start of report-isolation smoke and parallel phases,
- marks a paired block successful only when both R0 and R1 manifests are `completed`,
- records the ramp as `report_isolation_ramp_smoke` with `primary_comparison_ready=false`,
- keeps R1 explicitly pinned to `isolated_fork` so the comparison does not rely on automatic product selection.

## Product Implication

Report-session isolation is worth productizing as an experiment-backed direction. It directly
targets the observed token-growth problem: report generation should not make later research turns
resume the report-generation session.

Productization should preserve the experiment's session-chain invariant: report generation may run
in a forked report-only session, but later research turns must resume the pre-report research
session. If a provider cannot fork sessions or no pre-report research session exists, the product
must fall back visibly instead of silently pretending isolation happened.
