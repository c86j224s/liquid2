# Decision

## Confirmed for Seed 0002

The observed V2/V3-style advantage is not solely a final-report generation
artifact.

This is now supported by two checks:

- The blind intermediate evaluator saw the differences before seeing any final
  reports.
- A fresh neutral final generator preserved those differences when rewriting
  reports from intermediate answers only.

## Not Yet Claimed

This does not prove that V2 or V3 always wins. It only confirms that, in the
seed 0002 C1 code-analysis mission, the meaningful difference existed before the
final report stage.

## Working Decision

Continue with a controller-led Plasma research loop. Design the controller to
shape the investigation through short adaptive questions, then let a separate
final synthesis step write the report from the better intermediate material.

The next product wave should focus on the concrete issues surfaced by both the
repeat and the isolation:

1. Atomic Markdown report artifact creation.
2. Better report failure telemetry.
3. Explicit raw-versus-rendered Markdown view contract.
4. Korean/download filename handling.
5. Browser-level tests for artifact cards and view/download behavior.
