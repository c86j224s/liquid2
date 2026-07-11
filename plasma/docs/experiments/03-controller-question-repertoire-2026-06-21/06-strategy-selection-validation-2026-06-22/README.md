# Controller Strategy Selection Validation - 2026-06-22

This validation replays V2 and V3 with a new seed and adds one code-analysis mission before product implementation.

Scope:

- M1 narrow-source
- M3 broad-topic
- M6 source-conflict
- M5 code-analysis

Seed: `0002`
Source revision: `5d4cc98`
Source corpora: `<experiment-source-root>`

The run intentionally compares only V2 and V3 because previous validation already rejected V0/V1 as implementation candidates for this slice.

## Result

This run supports implementing controller strategy as a replaceable, logged selection surface rather than promoting one global default.

The new seed did not preserve the earlier clean V2/V3 split exactly:

- M1 remained stable for V2.
- M3 split: final report favored V3, but transcript quality favored V2.
- M6 moved to V2 in both new-seed judging modes.
- M5 code-analysis split: final report favored V3, but transcript quality favored V2.

This is still useful because the product decision is not "V2 or V3 forever." The confirmed direction is that Plasma needs a strategy interface that can choose, log, and later swap question repertoires per mission state.

## Product Interpretation

Implement the next product slice as:

- a controller strategy interface;
- V2 and V3 as separate strategy implementations;
- selection metadata on each controller turn;
- visible logs explaining which strategy was used and why;
- no reintroduction of evidence, claim, confidence, source-candidate, or AST machinery into the C1 default path.

The safest development posture is to keep V2 as the conservative baseline and expose V3 as an explicit or auto-selected broadening strategy for missions that need repeated reframing. The current evidence does not justify hard-coding V3 as the universal default.

## Code-Analysis Caveat

The M5 code-analysis mission surfaced a cost issue independent of winner selection. Both variants spent heavily on first-turn code mapping. That is expected for a cold codebase analysis, but the implementation should treat source-reading ergonomics and incremental continuation as first-class concerns.

The next code-analysis validation should include a continuation case after the first repo map exists.
