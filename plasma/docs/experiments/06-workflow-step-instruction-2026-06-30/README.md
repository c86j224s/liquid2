# Plasma Workflow Step-Instruction Experiment - 2026-06-30

This experiment compares `S0-current` and `S1-layered` before any product default change.

The primary endpoint is investigation process quality, not final report prose quality.
The experiment preserves the product language distinction between source, evidence, result,
saved knowledge, and report. Agent outputs, run goals, step instructions, scorer packets,
and final results are generated results and are not placed in the source corpus.
Codex apps/MCP resources are disabled for investigation runs; the only valid source-reading
surface is local shell reads against the fixed source corpus snapshot.

## Comparison Arms

- `S0-current`: neutral continuation close to the current investigation loop. It keeps a short
  mission reminder and current progress request, but does not expose an explicit
  `user_instruction_raw` + `run_goal` + `step_instruction` block structure.
- `S1-layered`: every investigation prompt includes `user_instruction_raw`, `run_goal`, and
  `step_instruction`. The user raw instruction is highest priority, the run goal is only a
  working interpretation, and the step instruction is the lowest-priority action request.

## Directory Layout

- `source_corpus/`: fixed source snapshots for fixtures.
- `runs/`: run outputs, transcripts, prompts, tool traces, and generated final results.
- `scorer_packets/`: variant-label-hidden packets used by the process scorer.
- `analysis/`: pre-registered statistical plan, score matrices, audits, and decision memos.
- `run_goal_generation/`: generated run-goal draft outputs and metadata.

## Execution Status

The pilot gate must pass before primary runs are interpreted. Pilot rows are never merged into
the primary score matrix.

After the design review on 2026-07-01, primary execution is additionally gated by:

- a dedicated `smoke` phase, separate from `primary`, for the 2-parallel concurrency check;
- scorer masking that prevents S1 prompt scaffolding labels from directly contributing to
  keyword-based process scores;
- per-run `sandbox-exec` isolation with a runtime source-copy, temporary `CODEX_HOME`, and
  user-home/live-DB read denial;
- Codex apps/MCP disabled for fixed-corpus investigation runs;
- review of `analysis/scorer_mask_audit.csv` after pilot/smoke scoring.

Smoke rows are harness and contamination evidence only. They are not primary statistical evidence.

## Primary Result

Primary execution completed with 120 scored runs and 60 clean paired blocks. All primary runs
finished with `completed` status; hard failures and contamination failures were both zero.

The primary decision is to reject `S1-layered` as a product default based on this experiment.
`S1-layered` produced 27 wins, 31 losses, and 2 ties against `S0-current`. The mean paired delta
was `+0.2980`, with a paired bootstrap 95% CI of `[-1.4704, 2.1685]` and sign-test p-value
`0.6940`. This does not provide statistically useful evidence that the three-layer prompt
structure improves the investigation process overall.

The strongest positive slice was `narrow-directed` (`+4.9814` mean delta), while
`ambiguous-intent`, `code-or-architecture`, and `source-conflict` were negative. This supports
keeping the implementation as an optional mode for continued product use and qualitative review,
not adopting it as the default autonomous investigation instruction structure.

During primary execution, an earlier contamination checker version falsely treated the search
string `func New` as the forbidden `nc` command. The checker was changed to detect forbidden
network tools by shell command token rather than substring. The affected interrupted runs were
force-regenerated before final scoring.
