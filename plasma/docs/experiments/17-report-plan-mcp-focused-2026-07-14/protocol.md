# Focused Protocol

## Fixed comparison

- Baseline product commit: `15cde729f1dca1b6090711a095fdebc713257c6e`
- Candidate product commit: `1b6239805f2dde41f7aaab36d8025812623da5a6`
- The controller commit is recorded separately and must have a clean worktree.
- Both arms use the same source, objective, Codex provider, locked model,
  session policy, and high reasoning effort within each report-mode pair.
- Planned and long-form modes use the same locked Codex model with high reasoning
  effort. The only treatment variable is baseline JSON plan return versus
  candidate MCP plan submission.

## Execution

1. Prepare reproducible baseline and candidate binaries in a new archive.
2. Run one isolated two-worker smoke containing baseline and candidate reports
   for both planned and long-form modes.
3. Stop if either mode fails its product-path, artifact, plan-submission,
   canonical-promotion, source-scope, or session-lineage checks.
4. If smoke passes, run all 12 frozen quality topics exactly once for each
   planned/long-form and baseline/candidate cell: 48 product runs with one to
   six workers and no additional replicates. A seed-derived execution schedule
   counterbalances baseline-first and candidate-first ordering six-to-six in
   each mode, then freezes the schedule hash before any quality run starts.
5. Reconstruct and blind up to 24 completed mode/topic pairs. The private arm
   mapping remains outside the judge input; pairs with a started failed arm are
   retained for ITT analysis but not judged.
6. Score plan and final-report dimensions with the configured Codex judge. The
   adapter runs ephemerally and read-only in a per-call temporary cwd with a
   minimal environment. It instructs Codex not to use tools and rejects any
   observed tool event; it does not claim that the CLI technically disables all
   tools. The adapter accepts one blind packet on stdin and emits only the A/B
   numeric score schema.
7. Analyze non-degradation separately for planned and long-form modes using the
   existing ITT and non-inferiority calculation. One mode cannot compensate for
   degradation in the other.

Binding, idempotency, malformed calls, storage failure, and crash recovery are
verified by the product's deterministic Go and integration tests. They are not
experiment stages. The earlier #16 Claude-auth smoke artifacts remain preserved
in their raw archive and are excluded from this successor run.

The frozen quality sources are approximately 600-3300 bytes. This phase tests
relative non-degradation against paired baselines; it does not establish
absolute long-form report quality.

Before the 48 quality runs start, a separate quality protocol lock records the
clean controller commit without changing the already frozen product binaries or
the completed smoke gate. The run configuration supplies `focused_judge` with exactly a non-blank
`model` and `effort: "high"`. Before packet creation, the protocol lock freezes
the controller commit, seed, execution schedule, judge settings, [`focused rubric`](focused-rubric.md) hash, adapter
hash, and response-schema hash. Packet creation then freezes public packet and private-mapping hashes
without exposing the mapping to the judge. Every judge attempt
is retained in a new immutable `attempt-N` directory; only a completion gate
written after all eligible packets score selects the attempt used for analysis.

Started product failures remain in the 48-cell matrix as ITT failures and
receive deterministic low scores during analysis. Packets are made only for
completed baseline/candidate pairs, so an otherwise completed arm without an
eligible pair is conservatively assigned the same low score. Pre-run
infrastructure failures still block the focused-quality gate.

Raw sources, reports, prompts, ledgers, provider state, session identifiers,
and blind mappings remain outside Git under the experiment archive policy.
