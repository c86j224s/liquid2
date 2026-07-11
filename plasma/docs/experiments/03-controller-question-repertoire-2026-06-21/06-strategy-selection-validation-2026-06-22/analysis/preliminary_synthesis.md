# Preliminary Synthesis

## Summary

The seed `0002` validation completed for M1, M3, M6, and M5. It compared V2 and V3 only, because earlier runs had already rejected V0 and V1 for this product slice.

The result supports the next development step, but it does not support a single fixed default strategy.

## Results

| Mission | Class | Final-report-only winner | Transcript-quality winner |
| --- | --- | --- | --- |
| M1 | narrow-source | V2 | V2 |
| M3 | broad-topic | V3 | V2 |
| M6 | source-conflict | V2 | V2 |
| M5 | code-analysis | V3 | V2 |

## Interpretation

The earlier seed `0001` suggested a simple rule: V2 for narrow-source work, V3 for broad-topic and source-conflict work. The new seed weakens that simple rule.

M1 stayed stable for V2. The other mission classes either flipped or split by evaluation lens. This makes the product decision clearer: Plasma should not bake in one controller behavior. It should make controller strategy selectable, observable, and replaceable.

## Development Recommendation

Proceed with development of a controller strategy interface.

The first implementation should:

- keep the C1 conversation path read-first and question-only;
- add V2 and V3 as separate strategy implementations;
- record the chosen strategy and reason for each controller turn;
- expose that log for debugging and later product judgment;
- avoid reviving evidence, claim, confidence, source-candidate, or AST machinery in the default path.

Use V2 as the conservative baseline. Keep V3 available as a broadening strategy rather than a global default.

## Code-Analysis Note

The M5 code-analysis run exposed a separate cost issue. Both V2 and V3 spent heavily on the first turn because cold codebase mapping is expensive. That cost should be handled through source-reading ergonomics and continuation behavior, not by choosing one controller variant alone.
