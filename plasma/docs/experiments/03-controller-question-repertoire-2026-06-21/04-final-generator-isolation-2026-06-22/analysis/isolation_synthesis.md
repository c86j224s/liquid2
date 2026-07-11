# Isolation Synthesis

## Question

Were the observed controller variant differences caused by the question-answer
trajectory, or could they have been created only by the final report generation
step?

## Method

The isolation used the existing seed 0002 repeat run and removed the original
final reports from the input.

1. Four blind intermediate bundles were created from turn1-turn4 answers only.
   The bundles excluded original `final-report.md` files and controller decision
   metadata.
2. Fresh neutral Codex sessions generated new final reports from each blind
   intermediate bundle.
3. A blind evaluator scored the intermediate bundles directly.
4. A second blind evaluator scored the neutral regenerated final reports.

## Result

The difference was visible before final report generation.

Intermediate-only evaluation:

- K1, mapped to V2, was strongest on reliability and recovery. It focused on
  atomicity, app-service boundaries, orphan artifacts, and missing failure
  signals.
- K3, mapped to V3, was strongest on user lifecycle and UX. It focused on raw
  Markdown preview, Korean filenames, card metadata, and artifact lifecycle.
- K4, mapped to V1, was strongest on product/test triage. It separated product
  problems, test-only gaps, and design decisions.
- K2, mapped to V0, was strongest as a balanced implementation map.

Neutral final regeneration preserved the same shape:

- K1 remained the strongest reliability/recovery report.
- K3 remained the clearest UX/lifecycle report.
- K4 remained highly actionable and UX/test-oriented.
- K2 remained balanced but less distinctive.

## Conclusion

Within seed 0002, the variant separation is not merely an artifact of the
original final report generator. The differences were already present in the
intermediate answers and were preserved when a fresh neutral final generator
rewrote the reports from those intermediate answers.

The precise wording and emphasis can still be amplified by the final generator,
but the final generator is not the sole cause of the observed separation.

## Remaining Limits

- This is still one seed. It supports the causal interpretation for seed 0002,
  not a statistical ranking across all missions.
- The blind evaluators are still LLM evaluators, so they are useful for
  comparative signal but not a substitute for user judgment.
- The blind bundles removed controller decision metadata, but the intermediate
  answers themselves naturally reveal what each controller emphasized. That is
  the intended signal, not contamination.

## Product Implication

The controller-led approach remains worth pursuing. The most useful behavior is
not "make every final report longer"; it is "shape the intermediate analysis so
that the final report has better material to synthesize."

For product work, this strengthens the case for:

1. Keeping the agent session and MCP-centered research loop.
2. Adding a controller that asks short adaptive questions.
3. Logging controller questions and intermediate answers as the primary research
   trace.
4. Treating final report generation as a synthesis step, not as the place where
   all insight is supposed to appear from nowhere.
