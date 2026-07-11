# Qualitative Synthesis

## What Ran

The pilot ran mission M5 once for each planned controller variant. Each run used
a fresh Codex thread, a read-only source copy under `/tmp`, one controller
decision, a second main-agent turn, and a final Markdown report request.

The mission asked the agent to analyze Plasma C1 browser conversation flow and
Markdown report artifact handling from source inspection only.

## What The Pilot Proved

The harness can run a source-inspection mission, resume the same agent session,
record controller questions as separate artifacts, and collect final Markdown
reports per variant.

The code-analysis task also surfaced a real product issue. All four variants
converged on the same conclusion: direct C1 Markdown report generation stores a
raw artifact and records `report.artifact.created`, but the browser report panel
does not expose the stored Markdown body for viewing or download. Legacy AST
report export has download actions, but that is a separate path.

## What The Pilot Did Not Prove

This run does not establish a winning controller strategy. No stagnation trigger
fired after the first turn, so V1 and V2 did not exercise their distinctive
behavior. V3 did not reach the scheduled divergent-question interval. The four
variants therefore behaved almost identically.

The result should not be used to choose V0, V1, V2, or V3. It should be treated
as a successful smoke test of the experiment harness and a useful code-analysis
finding.

## Product Finding

The next product fix suggested by the reports is a browser-safe C1 raw artifact
read/download boundary:

- mission-scoped artifact read route,
- mission ownership validation,
- content type and filename from the raw artifact record,
- "view Markdown" and "download Markdown" actions on `report.artifact.created`
  cards,
- no reuse of legacy AST report version export for direct Markdown artifacts.

## Experiment Lessons

The first code-analysis turn is expensive. For future codebase missions, the
experiment should either give the agent a narrower source map or use a stronger
random-seek code navigation tool.

The controller should run for more than one decision if the goal is to compare
question strategies. A two-turn pilot is enough to prove the harness, but not
enough to evaluate creative-switch behavior.

One artifact-generation issue occurred while preparing controller decision
files: an unquoted shell heredoc expanded Markdown backticks while writing a
local artifact. The generated decision files were repaired before commit, and
the product/source copies were not modified. Future experiment tooling should
avoid shell interpolation for Markdown artifacts.
