# Partial Results Memo

Status: infrastructure-limited partial execution.

## What Ran

- Pre-registration artifacts were created before scored runs.
- 12 fixtures and fixed source corpus snapshots were created.
- 12 run_goal drafts were generated with `gpt-5.4-mini`.
- Pilot execution started with `pilot-F01-S1-layered-seed-9001-attempt-1`.
- Only turn1 completed; turn2 was interrupted.

## Why Primary Did Not Run

The first completed turn consumed:

- input tokens: 355242
- cached input tokens: 235776
- output tokens: 24066
- reasoning output tokens: 18969

The planned pilot requires 16 runs, each with 6 investigation turns plus a final result. The primary design requires 120 runs. This observed cost profile makes completion infeasible in the current execution window without changing the pre-registered design.

## Statistical Interpretability

- There are 0 complete pilot paired blocks.
- There are 0 primary paired blocks.
- No process score matrix was produced.
- No bootstrap CI, sign test, or fixture-class sensitivity result is interpretable.
- No product-default decision can be made.

## Required Redirection

To continue this experiment, the Admiral should redesign or explicitly approve one of these changes before new scored runs:

- reduce per-turn source output by forcing bounded file reads and shorter answer caps;
- reduce investigation turns per run;
- use a smaller/faster execution model;
- run the full 16-run pilot asynchronously outside this turn;
- redesign the scorer to use shorter transcript packets without changing the primary endpoint.

Until such redesign is approved, the registered pilot gate remains failed and primary execution remains blocked.
