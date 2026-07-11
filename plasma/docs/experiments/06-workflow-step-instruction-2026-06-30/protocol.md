# Workflow Step-Instruction Experiment Protocol

Pre-registration timestamp: 2026-06-30T09:07:48Z

## Purpose

Measure whether explicit layering of `user_instruction_raw`, `run_goal`, and
`step_instruction` improves autonomous investigation depth, breadth, and raw-goal preservation
without narrowing broad user requests.

## Precedence Rule

1. `user_instruction_raw`: user's original instruction; highest priority.
2. `run_goal`: generated working interpretation; can help orientation but cannot replace or
   narrow the raw instruction.
3. `step_instruction`: current investigation action request; cannot inject facts, conclusions,
   citations, report prose, or override the raw instruction.

## Variants

- `S0-current`: short mission reminder and neutral progress request. No explicit three-layer
  prompt block. The raw user request is still available as the mission reminder; it is not
  removed to disadvantage the baseline.
- `S1-layered`: the three named blocks are present on every investigation turn in fixed order.

Prohibited arm: a prompt that omits `user_instruction_raw` and gives only `run_goal` plus
`step_instruction`.

## Fixtures

Primary design: 12 fixtures x 5 seeds x 2 variants = 120 runs.

Fixture classes:
- broad-open: 3
- narrow-directed: 2
- ambiguous-intent: 2
- source-conflict: 2
- code-or-architecture: 2
- thin-source: 1

Pilot design: fixtures `F01, F04, F06, F08` x seeds `9001, 9002` x
2 variants = 16 runs. Pilot validates harness/logging/scoring and is excluded from primary
analysis.

Smoke design: fixture `F01` x seed `9901` x 2 variants = 2 runs. Smoke is executed with
`--phase smoke --jobs 2` after pilot gate passes and after scorer construct-validity review.
Smoke validates concurrent execution, path isolation, cache behavior, Codex app/MCP isolation,
local shell source reads, and contamination auditing. Smoke rows are excluded from primary
statistical claims.

## Model And Config

- Investigation model: `gpt-5.4-mini`
- Run-goal draft model: `gpt-5.4-mini`
- Provider surface: `codex exec`
- Sandbox: Codex internal sandbox is disabled so shell commands can run inside the outer
  OS-level per-run `sandbox-exec` profile without nested macOS sandbox failure.
- Codex apps/MCP resources: disabled with `--disable apps`.
- Web browsing: disabled by prompt, fixture source policy, and unavailable through Codex apps.
- Temperature/config: Codex CLI defaults; identical for both primary variants.
- Timeout per model call: 1200 seconds.

## Budget

Each run receives 6 investigation turns plus one generated final result. The final result is
only a run-closure artifact and not the primary endpoint. Tool budget is enforced by audit and
manifest rather than product runtime throttling:

- tool calls: 80
- source/artifact reads: 30
- external search: 0 in this fixed-corpus execution
- wall-clock budget: 45 minutes

## Step Instruction Rule

Step instructions are deterministic experiment-only action requests. They are short, fact-free,
and ask for investigation movement only. In broad-open fixtures, they explicitly preserve open
possibilities instead of narrowing the task.

## Isolation Rule

Each run starts a fresh provider session in `source_corpus/<fixture>`. Resume calls use the same
fixture directory as working directory. Runs may not read parent directories, sibling fixture
directories, run outputs, scorer packets, prompt-generation outputs, or `<experiment-db-path>`.

Execution copies the fixture source corpus into a run-specific runtime directory under `/tmp` and
runs Codex through `sandbox-exec`. The runtime uses a temporary `CODEX_HOME` containing only the
auth material needed for the experiment session. The sandbox denies reads from the real user home,
the live Plasma DB path, local-source roots, and the product server work directory. Codex writes
`--output-last-message` inside the runtime directory; the harness copies that result back into the
run artifact directory after the call returns.

Each investigation turn must inspect at least one concrete source file through local shell
file-read commands. A run with no local source-read command is a hard failure, not a soft
quality penalty. Any MCP tool call is marked as contamination because it can bypass the fixed
source corpus. Shell commands that imply external network/source access, such as `curl`,
`wget`, `gh`, `git clone`, or DNS/network probes, are also marked as contamination.

Primary parallel execution is allowed only after pilot contamination checks report no shared
mutable state contamination.

## Scoring

The process scorer uses transcript/tool-trace behavior, not final report prose polish.
S1-specific prompt scaffolding labels are masked before keyword-based process scoring so that
the score does not reward echoing `user_instruction_raw`, `run_goal`, `step_instruction`, or
precedence wording. Raw unmasked scorer values are retained beside masked values for audit.

Scorer packet filenames use `masked_run_id`, not the original `run_id`, so packet filenames do not
reveal `S0-current` or `S1-layered` to downstream process review.

- depth: 30
- breadth: 25
- goal preservation: 30
- investigation discipline: 15

Broad-open narrowing harm is recorded separately and remains a decision-blocking gate.

## Statistical Plan

Primary analysis unit is a paired block: one fixture, one seed, both variants. The primary
effect is `S1-layered - S0-current` process score. Analysis reports paired bootstrap 95% CI,
paired sign test, and fixture-class sensitivity. If primary cannot reach 60 clean paired blocks,
the decision memo must mark the result partial/inconclusive and include interpretability limits.
