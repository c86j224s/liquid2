# Decision Memo

## Decision

Use the C1 controller-led approach as the direction for Plasma research and
report analysis, but do not yet crown a single controller variant.

## Rationale

The repeat showed that question-only steering can change the shape of the final
analysis without adding heavy intermediate structures. V2 and V3 were especially
useful because they moved from local implementation details to product recovery
and user lifecycle questions while still requiring the main agent to map claims
back to code.

The result is aligned with the current Plasma direction:

- Keep the main agent in an existing session.
- Keep prompts thin.
- Use the controller to ask adaptive questions.
- Let the agent read the code and mission material through tools rather than
  stuffing large histories into prompts.
- Treat structured logs as observability, not as mandatory intermediate claims
  that drive product behavior.

## What Not To Do Yet

- Do not rebuild evidence/claim/confidence/AST as required report-generation
  machinery based on this run.
- Do not force every controller to diverge early. The first two turns benefited
  from normal confirmatory narrowing.
- Do not treat this seed as statistically significant. It is one useful repeat,
  not a final benchmark.

## Product Work Unlocked

The experiment produced concrete product follow-ups:

1. Implement atomic Markdown report artifact creation.
2. Improve report failure telemetry and recovery visibility.
3. Decide and test the raw-versus-rendered Markdown preview contract.
4. Fix Korean/download filename behavior.
5. Add browser-level artifact view/download tests.

## Experiment Work Remaining

Seeds 0003 and 0004 remain reserved. They should be run if the next decision
requires stronger evidence about controller variant ranking.
