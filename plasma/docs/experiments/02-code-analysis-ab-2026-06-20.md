# Plasma Code Analysis A/B Experiment - 2026-06-20

This note records the first code-analysis experiment that followed the
conversation-first report generation test.

## Purpose

The experiment tested whether a fresh agent can analyze an unfamiliar codebase
well enough to produce a useful technical report when it is steered through a
research conversation, rather than receiving a large prebuilt prompt or report
pack.

## C1 Cutover Decision

The cutover decision is to keep adaptive controller steering as steering, not as
a separate product center or source producer. Controller outputs guide the next
user-style turn and remain results, not sources, evidence, or claims.

The specific question was whether Plasma should keep investing in a thin
MCP-centered research interface where the agent chooses what to inspect, and
where a separate controller can steer breadth and depth between turns.

## Target

- Repository: Liquid2
- Source root for each run: copied from `git archive HEAD liquid2` into an
  isolated `<experiment-source-root>/...` directory.
- Logs and generated outputs: stored outside the copied source roots under
  `<experiment-run-root>/...`.

The experiment intentionally avoided writing intermediate analysis artifacts
into the analyzed repository copy. This was necessary to prevent the agent from
reading its own prior output as if it were source code.

## Variants

| Variant | Shape | Tooling |
|---|---|---|
| A | Single-shot baseline | Basic shell/file inspection |
| B1 | Static steering | Basic shell/file inspection |
| B2 | Static steering | Basic shell/file inspection plus Serena MCP |
| C1 | Adaptive controller steering | Basic shell/file inspection |
| C2 | Adaptive controller steering | Basic shell/file inspection plus Serena MCP |

"Adaptive controller steering" means a separate controller read the latest
agent answer and chose the next user-style steering prompt. It did not replace
the analysis agent. It acted like a user who asks the next narrowing or
broadening question after seeing the answer.

## Contamination Controls

- Each variant/run used a fresh source copy.
- The agent process started with its working directory set to that source copy.
- Resume calls were also executed from the same source copy.
- Logs, prompts, controller outputs, and reports were written outside the source
  copy.
- Runs recorded before/after source audits and log scope audits.
- Serena fallback was removed. If Serena could not attach to the intended
  source root, the run was expected to fail rather than silently fall back.

No run was excluded by the final judge for source-scope contamination. Serena
runs did create `.serena` metadata in the copied source root; this was recorded
as tool-state contamination and should be prevented in future Serena-specific
experiments by forcing Serena metadata outside the analyzed repository.

## Pilot Result

The pilot judge ranked the variants:

1. C2
2. C1
3. B1
4. B2
5. A

The pilot showed that adaptive steering improved depth and breadth. It did not
prove that Serena caused the improvement, because C2 combined Serena with the
adaptive controller.

## Repeat Result

The repeat pass ran:

- A: one additional run
- B1: one additional run
- C1: one additional run
- C2: two additional runs

The repeat judge chose C1 as the best operational default. C2 produced one of
the strongest individual reports, but it varied more across runs in length,
format, and evidence style. C1 was more stable and produced reports that were
immediately useful for engineering follow-up.

Main conclusions:

- Keep A as a baseline. It is cheaper and sometimes surprisingly strong, so it
  is useful for measuring whether steering is paying for itself.
- Deprioritize B1 as the main product direction. Static steering can work, but
  it costs more turns without adapting to the actual answer.
- Adopt C1 as the current default research experiment shape: one analysis
  session, thin instructions, source inspection through tools, and adaptive
  controller turns.
- Split Serena into a separate tool experiment. Its current value was not
  isolated, and its metadata behavior needs cleanup before it becomes a default
  code-analysis tool.

## Product Interpretation

The useful direction is not to build larger prompts or larger precomputed
report bundles. The useful direction is to give the agent a small set of
high-leverage research tools and steer it based on what it actually finds.

For code analysis, the first turn is naturally expensive because the agent must
map an unfamiliar repository. A future experiment should test a warm follow-up
pass in the same session after the first analysis. That better matches the
intended product behavior: a user first asks for a broad analysis, then later
asks the same mission to inspect a narrower issue, risk, or implementation
path.

## Next Experiment Direction

Run a warm follow-up experiment:

- Use the same analysis session after the cold-start report.
- Ask a narrower follow-up question instead of generating a full second report.
- Prevent full rewrites unless the prior report was materially wrong.
- Measure whether the follow-up pass improves:
  - missing path discovery,
  - correction of wrong assumptions,
  - precise file/symbol references,
  - practical implementation recommendations,
  - cost relative to a new cold-start run.

Serena should be tested separately after:

- metadata is kept outside the analyzed repository,
- Go/Dart language support is known to work,
- the same controller and budget are used for both Serena and non-Serena runs.
