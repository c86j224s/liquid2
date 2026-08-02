# Plasma Experiment Index

This directory keeps public experiment summaries in chronological order. Numeric
prefixes indicate the order in which the experiments should be read. Raw runs,
private source corpora, prompt packets, screenshots, generated previews, and
large artifacts live outside the repository under `research-artifacts/`.

## Top-Level Sequence

1. [`01-report-generation-ab-2026-06-20.md`](01-report-generation-ab-2026-06-20.md)
   - Early report-generation comparison.
2. [`02-code-analysis-ab-2026-06-20.md`](02-code-analysis-ab-2026-06-20.md)
   - Code-analysis report comparison.
3. [`03-controller-question-repertoire-2026-06-21/`](03-controller-question-repertoire-2026-06-21/README.md)
   - Controller, question strategy, MCP reading, long-form report, and designed
     HTML report experiments.
4. [`04-c1-report-quality-case-2026-06-24.md`](04-c1-report-quality-case-2026-06-24.md)
   - Product-use report-quality case.
5. [`05-media-inspect-2026-06-26/`](05-media-inspect-2026-06-26/README.md)
   - Media inspection and report-embedding experiments.
6. [`06-workflow-step-instruction-2026-06-30/`](06-workflow-step-instruction-2026-06-30/README.md)
   - Workflow step-instruction experiments.
7. [`07-token-diet-measurement-2026-07-01/`](07-token-diet-measurement-2026-07-01/README.md)
   - Token-usage measurement and report-session isolation experiments.
8. [`08-report-humanize-2026-07-04/`](08-report-humanize-2026-07-04/README.md)
   - Korean report tone humanize experiment.
9. [`09-design-skill-rendering-2026-07-05/`](09-design-skill-rendering-2026-07-05/README.md)
   - External design-skill principles translated into Plasma designed HTML
     renderer changes and productization gates.
10. [`10-generation-time-tone-2026-07-07/`](10-generation-time-tone-2026-07-07/README.md)
    - Generation-time Korean report tone experiment and comparison against the
      H5 post-generation tone pass.
11. [`11-product-path-g2-2026-07-07/`](11-product-path-g2-2026-07-07/README.md)
    - Product-path validation of G2 generation guidance with isolated Plasma
      DBs, MCP source reads, H5 comparison, and blind preference judging.
12. [`12-long-form-session-strategy-2026-07-07/`](12-long-form-session-strategy-2026-07-07/README.md)
    - Long-form report session strategy experiment comparing same-session
      report chaining with independent section drafting and C4 heading
      normalization.
13. Experiment 13 is reserved by concurrent work and is not present here yet.
14. [`14-markdown-report-magic-words-2026-07-10/`](14-markdown-report-magic-words-2026-07-10/README.md)
    - Korean report instruction and limitations-placement experiments.
15. [`15-report-plan-mcp-2026-07-13/`](15-report-plan-mcp-2026-07-13/README.md)
    - Stopped product-path smoke: planned passed, while a shared harness option
      invalidated both long-form runs.
16. [`16-report-plan-mcp-focused-2026-07-13/`](16-report-plan-mcp-focused-2026-07-13/README.md)
    - Closed authentication smoke: planned completed, while both long-form
      runs failed before MCP plan submission because Claude authentication failed.
17. [`17-report-plan-mcp-focused-2026-07-14/`](17-report-plan-mcp-focused-2026-07-14/README.md)
    - Completed Codex-only focused comparison: quality non-degradation was
      supported, but one candidate long-form ITT failure and one source-read
      trace audit failure left operational reliability and productization blocked.
18. [`18-report-long-form-finalize-mcp-2026-07-14/`](18-report-long-form-finalize-mcp-2026-07-14/README.md)
    - Stopped Codex-only successor: the corrected smoke passed, but one of 24
      quality runs failed after the ITT boundary and the controller stopped
      before confirmatory statistics, so adoption was rejected.
19. [`19-report-long-form-finalize-itt-analysis-2026-07-14/`](19-report-long-form-finalize-itt-analysis-2026-07-14/README.md)
    - Passed analysis-only successor: the 11 scored pairs plus one preserved ITT
      failure passed final-report noninferiority and completeness guardrails.
20. [`20-report-long-form-finalize-operational-reliability-2026-07-14/`](20-report-long-form-finalize-operational-reliability-2026-07-14/README.md)
    - Passed corrected two-arm smoke and current-candidate-only operational gate
      with 12 of 12 long-form runs satisfying every locked invariant.
21. [`21-report-fanout-2026-07-16/`](21-report-fanout-2026-07-16/README.md)
    - Completed 24-topic A/B comparison of current serial long-form report
      generation against section-level fanout; the candidate was faster on all
      paired topics with no terminal failures and was productized as an
      explicit long-form "fast parallel" option while keeping serial as the
      default.
22. [`22-report-section-contract-2026-07-17/`](22-report-section-contract-2026-07-17/README.md)
    - Completed three-arm reinforcement on whether the existing long-form
      section `purpose` field can carry a more concrete writing contract. The
      idea improved section focus in some readings but produced statistically
      visible shortening, and the coverage-locked arm did not fix it. Later
      section-brief follow-ups found `section_brief` promising but not
      statistically proven as a quality upgrade, while `section_brief_cluster_memory`
      produced a statistically visible length increase. Both follow-up arms were
      kept as explicit long-form writing options, while the default path stayed
      on the existing guidance.
23. [`23-report-visual-aids-2026-07-20/`](23-report-visual-aids-2026-07-20/README.md)
    - Completed product-path visual-aid experiment. The user selected
      `visual_plan`, which adds sparse table/Mermaid intent during planning and
      writing without changing the report schema or forcing visuals, as the
      product default for normal and long-form reports.
24. [`24-report-section-visual-plan-2026-07-20/`](24-report-section-visual-plan-2026-07-20/README.md)
    - Completed product-path follow-up for combining sparse visual-aid planning
      with long-form-only writing options. The focused `section-brief` visual
      candidate looked productizable, while the rich
      `section-brief-cluster-memory` visual candidate stayed mixed because it
      changed the rich-coverage option's length and density more often.
25. [`25-report-visual-type-selection-2026-07-21/`](25-report-visual-type-selection-2026-07-21/README.md)
    - Prepared product-path experiment for testing whether report agents can
      choose visual types by source structure, including dense quantitative
      tables, agent benchmark matrices, protocol lifecycles, and complex
      architecture dependency graphs.
26. [`26-report-assembly-edit-tools-2026-07-21/`](26-report-assembly-edit-tools-2026-07-21/README.md)
    - Prepared product-path experiment for testing whether long-form Part
      assembly should submit only connective tissue through MCP edit tools
      instead of returning the assembly JSON in the agent response.
27. [`27-report-visual-evidence-fit-2026-07-22/`](27-report-visual-evidence-fit-2026-07-22/README.md)
    - Prepared product-path experiment for testing whether reports should use
      Mermaid diagrams, tables, and qualitative charts more readily when the
      source supports a structure, flow, relation, or qualitative contrast
      without exact numeric proof.
28. [`28-report-visual-reading-aid-preference-2026-07-22/`](28-report-visual-reading-aid-preference-2026-07-22/README.md)
    - Completed product-path experiment for testing whether reports should
      prefer compact visual aids over longer explanatory prose when the source
      supports a relationship, sequence, dependency, comparison, or uncertainty
      structure.
29. [`29-report-visual-reader-intent-2026-07-22/`](29-report-visual-reader-intent-2026-07-22/README.md)
    - Completed follow-up product-path experiment for testing whether
      reader-task intent guides visual aids better than direct visual-type
      pressure. The candidate reduced one meta-diagram failure but was too
      conservative overall, so it was not adopted as the product default.
30. [`30-report-visual-clarity-seeking-2026-07-22/`](30-report-visual-clarity-seeking-2026-07-22/README.md)
    - Completed follow-up product-path experiment for testing whether active
      clarity-seeking guidance can improve visual-aid choice without adding
      prohibition-heavy wording. The candidate increased visual-aid count but
      did not improve alignment, so it was not adopted as the product default.
31. [`31-report-visual-affordance-priming-2026-07-22/`](31-report-visual-affordance-priming-2026-07-22/README.md)
    - Completed follow-up product-path experiment for testing whether a light
      source-shape affordance reminder helps report writers apply the existing
      visual-type mapping more consistently. The candidate improved timeline
      activation without regressions, but did not reach strict statistical
      significance and was not adopted as the product default.
32. [`32-report-narrative-contract-2026-07-22/`](32-report-narrative-contract-2026-07-22/README.md)
    - Productized a reader-facing writing contract, bound Section reads for Part
      editors, and a constrained final manuscript editor. An actual serial Web
      run preserved the required details without shortening and reduced
      source-management language. The contract became a common baseline beneath
      the existing Web writing choices rather than a separate visible option.
33. [`33-report-direct-explanation-writing-2026-07-24/`](33-report-direct-explanation-writing-2026-07-24/README.md)
    - Prepared issue #179 experiment redefining report quality around
      source-grounded processed reading artifacts. It compares the current
      narrative-contract baseline, a paragraph-contract supporting candidate,
      and a curiosity-led explanation primary candidate without changing plan
      schema, UI choices, storage, or report assembly.
34. [`34-report-curiosity-natural-voice-2026-07-25/`](34-report-curiosity-natural-voice-2026-07-25/README.md)
    - Completed a small issue #179 follow-up testing whether the curiosity-led
      explanation candidate can keep its reading momentum while reducing
      AI-like signposting, repeated caveats, and mechanical paragraph endings.
      The candidate reduced visible signposting but expanded some reports, so it
      remains a follow-up signal rather than an adoption result.
35. [`35-report-curiosity-tight-voice-2026-07-25/`](35-report-curiosity-tight-voice-2026-07-25/README.md)
    - Completed an issue #179 follow-up testing whether the natural-voice signal
      can be kept while controlling length expansion, repeated caveats, repeated
      examples, and repeated paragraph machinery. The tight candidate clearly
      reduced length, but the result stayed mixed because some outputs still
      exposed report-framing language, title-language drift, or over-compressed
      prose. It was not adopted as a product default.
36. [`36-report-edited-reading-voice-2026-07-25/`](36-report-edited-reading-voice-2026-07-25/README.md)
    - Completed an issue #179 follow-up across three topics and four long-form
      arms. The edited candidate kept more detail than the tight candidate,
      avoided the natural candidate's length expansion, aligned Korean titles,
      and removed direct report self-framing in the samples. It is the leading
      candidate for a broader run, not a product default, because outline-like
      Part and Section narration still remained. The candidate kept the
      existing plan schema and `section_fanout` assembly path.
37. [`37-report-section-direct-reading-2026-07-25/`](37-report-section-direct-reading-2026-07-25/README.md)
    - Completed an issue #179 follow-up after stage attribution found most
      remaining outline narration already present in immutable Section drafts.
      The Section-only candidate won four of six blinded comparisons and
      reduced normalized outline narration in four topics and in aggregate.
      It advances as a controlled experiment baseline, not a product default,
      because Part assembly restored much of the narration and candidate length
      still varied from 0.92x to 1.32x of baseline.
38. [`38-report-part-connective-economy-2026-07-26/`](38-report-part-connective-economy-2026-07-26/README.md)
    - Completed an issue #179 follow-up that kept experiment 37's direct,
      immutable Sections and changed only Part connective writing. The
      candidate reduced Part overhead from 16.2% to 3.2%, reduced newly added
      document-position phrases from 38 to 2, and won all six blinded readings.
      After end-to-end reading and explicit user approval, the cumulative
      candidate became the default for new long-form reports. Section-level
      repetition and evidence-relative length remain tracked in issue #189.
39. [`39-report-subject-direct-synthesis-2026-07-27/`](39-report-subject-direct-synthesis-2026-07-27/README.md)
    - Wave 1 found that the candidate reduced source-as-narrator prose but did
      not control evidence-relative length consistently, so the default was not
      changed. The profile remains available for Wave 2 with Korean original,
      multi-source conflict, market research, and source-sparse archived
      fixtures.
40. [`40-reader-style-gate-2026-07-28.md`](40-reader-style-gate-2026-07-28.md)
    - Bounded fixed-draft test of reader/style/gate sequencing. The candidate
      did not pass as-is, but it identified the reader-orientation prompt
      boundary refined in later #190 work.
53. [`53-reader-orientation-boundary-prompt-2026-07-29.md`](53-reader-orientation-boundary-prompt-2026-07-29.md)
    - Corrected reader prompt boundary accepted after blind review and
      re-reading; established that helpful report-level orientation must not be
      removed as generic meta-signposting.
54. [`54-reader-full-pipeline-acceptance-2026-07-29.md`](54-reader-full-pipeline-acceptance-2026-07-29.md)
    - Candidate-only product-path acceptance showing the corrected reader
      prompt survived Part editing, reader editing, style editing, integrity
      gate, and canonical finalization in one exploratory and one strict run.
55. [`55-final-writer-v2-2026-07-29/`](55-final-writer-v2-2026-07-29/README.md)
    - Generated the corrected W6-B fixed-input product-path comparison of
      current v1 against final-writer v2 using Korean reviewed Parts. Provenance
      replay passed; two sealed model readings remained mixed/inconclusive, and
      a separate host direct reread found two v2 wins and two ties with no
      observed regression. The user approved product adoption to separate
      finalization responsibilities and preserve room for later improvement,
      without claiming immediate quality superiority or completed blind human
      adjudication.
57. [`57-report-natural-voice-selective-acceptance-2026-07-30.md`](57-report-natural-voice-selective-acceptance-2026-07-30.md)
    - Completed the issue #207 tone-and-word-choice experiment on eight sealed
      final manuscripts. All eight selectively assembled candidates passed the
      deterministic guards and won the locked blind reading, while a post-decode
      review found no semantic or citation drift. The sealed prompt advances as
      a productization candidate, not a product default.
58. [`58-report-natural-voice-examples-2026-07-30.md`](58-report-natural-voice-examples-2026-07-30.md)
    - Tested whether six concrete target-voice example sets could amplify the
      safe but subtle experiment 57 correction. The example prompt split the
      eight blind readings 4-4, with four clear wins and four slight losses.
      One of four audited meaning drifts belonged to the example arm; the other
      three belonged to the control. The prompt was not adopted, but the signal
      was retained for a narrower follow-up rather than discarded.
59. [`59-report-natural-voice-contrastive-examples-2026-07-31.md`](59-report-natural-voice-contrastive-examples-2026-07-31.md)
    - Replaced experiment 58's simple example block with category-matched edit,
      preserve, and forbidden contrast cases. The contrastive prompt won two of
      eight blind readings, including one large win, but lost six, with four
      clear losses and two candidate-arm meaning drifts. The approach was not
      adopted; the result shows an unstable example-imitation effect rather
      than proving that examples are categorically ineffective.
60. [`60-report-natural-voice-examples-replication-2026-07-31.md`](60-report-natural-voice-examples-replication-2026-07-31.md)
    - Repeated experiment 58's exact simple-example prompt on eight fresh
      manuscripts. The example arm won three blind readings and lost five, with
      a signed magnitude score of `-2`; it also produced three semantic and
      claim-scope drifts versus one semantic drift in the control. The earlier
      promising signal did not replicate, and product readiness remains
      unevaluated because this experiment did not run the product path.

## Controller Experiment Sequence

The controller experiment directory has its own nested sequence:

1. `01-repeat-2026-06-21`
2. `02-controller-generator-mcp-isolation-2026-06-22`
3. `03-expanded-repeat-judgment-2026-06-22`
4. `04-final-generator-isolation-2026-06-22`
5. `05-mission-class-expansion-2026-06-22`
6. `06-strategy-selection-validation-2026-06-22`
7. `07-v2-v3-transcript-quality-2026-06-22`
8. `08-c1-grounding-validation-2026-06-22`
9. `09-g0-controller-mcp-followup-2026-06-23`
10. `10-question-navigator-2026-06-26`
11. `11-question-navigator-cwd-fixed-2026-06-26`

Inside `09-g0-controller-mcp-followup-2026-06-23`, the nested sequence continues
with report-prompt, controller-quality, MCP random-seek, long-form report,
report-composition, visual-plan, and designed-HTML experiments.
