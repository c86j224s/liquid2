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
