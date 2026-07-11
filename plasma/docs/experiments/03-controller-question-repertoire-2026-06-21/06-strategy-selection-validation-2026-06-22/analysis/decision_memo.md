# Decision Memo

## Decision

The additional validation supports development, but it changes the implementation target.

Do not implement one hard-coded controller strategy. Implement controller strategy as a replaceable product interface with observable selection logs.

The evidence now says:

- V2 is the safer conservative baseline.
- V3 remains useful for broadening and repeated reframing.
- The exact selection rule is not stable enough to freeze.
- The product should let Plasma choose, record, and later adjust the strategy.

## Evidence Summary

Previous clean validation on seed `0001` produced:

| Mission | Class | Final-report-only | Transcript-quality |
| --- | --- | --- | --- |
| M1 | narrow-source | V2 | V2 |
| M3 | broad-topic | V3 | V3 |
| M6 | source-conflict | V3 | V3 |

This new seed `0002` produced:

| Mission | Class | Final-report-only | Transcript-quality |
| --- | --- | --- | --- |
| M1 | narrow-source | V2 | V2 |
| M3 | broad-topic | V3 | V2 |
| M6 | source-conflict | V2 | V2 |
| M5 | code-analysis | V3 | V2 |

The important signal is not that V2 beat V3 overall. The important signal is that the winner changes by mission shape and evaluation lens. That directly supports optionization and logging.

## How to Read the Split

M1 is stable. Narrow, concrete recovery work favors V2 because one reframing plus recovery is enough.

M3 is mixed. V3 still produced the stronger final broad-topic report, but V2 produced the better judged transcript in this seed. That means broad exploration benefits from scheduled reframing, but the product should not force repeated divergence when the conversation is already converging.

M6 moved from V3 in the earlier seed to V2 in the new seed. This weakens the old rule that source-conflict always needs V3. Conflict handling should be selected from observed state, not mission label alone.

M5 is mixed. V3 produced a broader architecture answer, while V2 produced a cleaner minimum-change code-analysis conversation. This is useful for implementation: code-analysis should support both a conservative pass and a broad architecture pass.

## Product Direction

Implement:

- a controller strategy interface;
- V2 and V3 as first concrete strategies;
- per-turn strategy selection metadata;
- a visible controller log that explains the selected strategy and question intent;
- a way to override or swap strategy without changing the main research loop.

Do not implement:

- evidence, claim, confidence, source-candidate, or AST report machinery as part of the C1 default conversation loop;
- a universal V3 default;
- a hidden controller that changes steering without inspectable logs.

## Implementation Bias

Start with V2 as the conservative default because it won the stable narrow-source case and all transcript-quality judgments in seed `0002`.

Use V3 as an available broadening strategy when the mission needs repeated lens shifts, when the conversation is stuck, or when the user explicitly wants wider exploration.

Keep the selection rule simple and observable. The point of the first product slice is to make strategy choice inspectable and replaceable, not to pretend the adaptive policy is settled.

## Remaining Risk

The current validation is enough to enter development of the strategy interface. It is not enough to freeze an automatic strategy-selection policy.

Code-analysis also showed a separate cost problem: first-turn repo mapping is expensive for both variants. The next code-analysis validation should include a continuation case where the repo has already been mapped once.
