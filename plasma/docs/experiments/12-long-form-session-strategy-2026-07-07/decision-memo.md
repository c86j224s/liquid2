# Decision Memo

## Decision

Adopt C4 only as an assembly-stage direction, not as a section-writing or
whole-report rewrite strategy.

## Why

C4 is currently the best long-form assembly candidate:

- it avoids the same-session token explosion pattern;
- it preserves independently drafted section files;
- it fixes duplicate heading and heading-level drift;
- it keeps the product-like cost profile essentially equal to B.

However, it does not prove that C4 makes sections denser. In the final C4
candidate, section bodies are reused from the B-style independent-section run.
The six-sample extension showed five final-length wins out of six, zero heading
drift, and zero adjacent duplicate headings, but not a formal `p < 0.05`
sign-test result for final length.

## Next Step

Use this result to guide product implementation narrowly:

- keep section drafting independent;
- preserve section bodies;
- normalize final headings deterministically;
- avoid whole-report rewrite passes;
- run separate experiments before changing section-drafting prompts, prose
  flow, or tone behavior.

## Product Boundary

If adopted, C4 must not reintroduce whole-report rewriting. The durable
direction is:

1. draft sections independently;
2. preserve section bodies;
3. normalize heading assembly deterministically;
4. assemble the final Markdown from report-part outputs.
