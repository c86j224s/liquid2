# Report Designed HTML Experiment

This experiment keeps completed R7 Markdown reports as input material and tests a separate designed interactive HTML artifact mode. It does not change Plasma product report generation, browser report paths, default MCP production tools, or the VP3 baseline.

Reference HTML files are read only calibration examples. They are summarized as an ambition profile and are not copied, linked as assets, or treated as report sources.

## Commands

```bash
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_designed_html_experiment.py self-test
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_designed_html_experiment.py run --limit 1 --variants DH0,DH1,DH2,DH3 --dry-run
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_designed_html_experiment.py run --limit 1 --variants DH0,DH1,DH2,DH3 --jobs 1
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_designed_html_experiment.py run --missions CQ2,CQ4 --variants DH4,DH5,DH6 --jobs 2
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_designed_html_experiment.py run --missions CQ5,CQ6,CQ7,CQ8 --variants DH7 --jobs 2
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_designed_html_experiment.py run --missions CQ5,CQ6,CQ7,CQ8 --variants DH8 --jobs 2
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_designed_html_experiment.py run --missions CQ5,CQ6,CQ7,CQ8 --variants DH9,DH10 --jobs 2
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_designed_html_experiment.py run --missions CQ5,CQ6,CQ7,CQ8 --variants DH11 --jobs 2
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_designed_html_experiment.py run --missions CQ1,CQ2,CQ3,CQ4,CQ5,CQ6,CQ7,CQ8 --variants DH12,DH13 --jobs 4
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_designed_html_experiment.py run --missions NONE --external-reports /path/to/report.html --variants DH12,DH13 --jobs 2
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_designed_html_experiment.py run --missions CQ5,CQ6,CQ7,CQ8 --variants DH14 --jobs 4
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_designed_html_experiment.py run --missions NONE --external-reports /path/to/report.html --variants DH14 --jobs 2
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_designed_html_experiment.py run --missions CQ5,CQ6,CQ7,CQ8 --external-reports /path/to/report.html --variants DH15 --repeats 3 --jobs 4
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_designed_html_experiment.py run --missions CQ5,CQ6,CQ7,CQ8 --external-reports /path/to/report.html --variants DH16 --repeats 3 --jobs 4
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_designed_html_experiment.py run --missions CQ5,CQ6,CQ7,CQ8 --external-reports /path/to/report.html --variants DH17 --repeats 3 --jobs 4
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_designed_html_experiment.py run --missions CQ5,CQ6,CQ7,CQ8 --external-reports /path/to/report.html --variants DH18 --repeats 3 --jobs 4
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_designed_html_experiment.py run --missions CQ5,CQ6,CQ7,CQ8 --external-reports /path/to/report.html --variants DH19 --repeats 3 --jobs 4
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_designed_html_experiment.py run --missions CQ5,CQ6,CQ7,CQ8 --external-reports /path/to/report.html --variants DH20 --repeats 3 --jobs 4
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_designed_html_experiment.py run --missions CQ5,CQ6,CQ7,CQ8 --external-reports /path/to/report.html --variants DH21,DH22,DH23 --repeats 3 --jobs 4
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_designed_html_experiment.py run --missions CQ5,CQ6,CQ7,CQ8 --external-reports /path/to/report.html --variants DH24 --repeats 3 --jobs 4
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_designed_html_experiment.py summarize
```

## Variant Ladder

- `DH0`: free-form one-shot with minimal component vocabulary.
- `DH1`: one-shot with component vocabulary and summarized reference profile.
- `DH2`: design brief first, then final HTML from the report and brief.
- `DH3`: structural skeleton overfitting probe; diagnostic, not preferred by default.
- `DH4`: Google-I/O-inspired exploratory report app probe; learns the structural ambition without copying reference branding or mobile overflow.
- `DH5`: polished DH4 refinement; keeps the exploratory structure while prioritizing first-glance beauty, cohesive visual system, and share-worthy artifact quality.
- `DH6`: polished dense report-app probe; combines DH4's information richness with DH5's visual finish.
- `DH7`: content-first probe; delays visual polish and forces the first viewport to open with a concrete scene, before/after, failure mode, or decision tension.
- `DH8`: content-first dense infographic probe; combines DH7's concrete entry, DH6's report-app density, and a stricter definition of infographic as visual reasoning rather than card decoration.
- `DH9`: reference-grade tabbed report-app probe; tests whether the gap is mainly information architecture, navigation depth, and repeated information grammar.
- `DH10`: multi-pass reference-grade report-app probe; creates a blueprint, drafts HTML in the same Codex session, then critiques and revises the draft before hard-fail/render checks.
- `DH11`: deterministic component-renderer probe; asks the agent for a rich content model JSON, then renders the HTML through a fixed mobile-safe report-app component library.
- `DH12`: visual-identity renderer probe; keeps the DH11 content-model pipeline but lets the renderer apply a subject-sensitive visual identity such as archive, blueprint, newsroom, cinematic, product, or atlas.
- `DH13`: SVG-diagram renderer probe; extends the content model with visual reasoning units and renders inline SVG diagrams while preserving mobile safety and source/reference gates.
- `DH14`: composition-skeleton renderer probe; keeps DH13's SVG/content-model path and adds a subject-sensitive page skeleton such as scroll narrative, decision dashboard, field guide, or tabbed report app.
- `DH15`: interactive composition renderer probe; keeps DH14's page skeleton diversity while restoring DH13-like navigation controls and fuller section rendering density.
- `DH16`: hero-locked composition probe; keeps DH13's winning first-viewport hierarchy and right-side orientation map while asking the content model for DH15-style composition planning below the fold.
- `DH17`: compact reference-app renderer probe; keeps the deterministic content-model path but replaces the oversized hero with a Google-I/O-calibrated compact top band, KPI strip, sticky topic rail, orientation grid, and faster entry into dense report sections.
- `DH18`: editorial compression renderer probe; keeps DH17's compact reference-app structure but lowers the first viewport, shortens the stat rail, and polishes spacing so the body starts closer to the Google I/O reference rhythm.
- `DH19`: color-story reference-app probe; keeps DH18's compact IA but restores a Google-I/O-like color-led first impression, lighter editorial section rhythm, and less internal-dashboard visual mood.
- `DH20`: DH19 ablation probe; uses the same color-story renderer but withholds the reference gallery profile from the content-model prompt to test whether DH19's apparent quality is reference-hint overfitting.
- `DH21`: reference-free visual-magazine probe; keeps the compact app structure but pushes subject-specific timelines, maps, flows, and trade-off boards harder so the report reads less like a dashboard and more like a designed long-form analysis artifact.
- `DH22`: reference-free visual-lead probe; promotes the strongest subject-specific visual unit into the first viewport so the report's topic identity appears before the generic KPI and section rails.
- `DH23`: reference-free visual-map probe; promotes the strongest subject-specific visual unit into the first viewport as a connected SVG relationship map instead of a card strip.
- `DH24`: reference-free dense visual-app probe; keeps DH23's relationship-first opening but compresses the first viewport and varies the hero visual grammar by subject, such as timeline, swimlane, role matrix, or decision route.

## Hard-Fail Gates

Generated artifacts must be complete Korean HTML documents with embedded CSS, no auto-loaded external assets, no network fetch/import behavior, no experiment/path/model leaks, and preserved source/reference and uncertainty material. Plain `href="https://..."` source links are allowed only as references; scripts, stylesheets, images, fonts, iframes, CSS imports, CSS URL assets, fetch/XHR/WebSocket/EventSource, and script imports fail. External HTML inputs are converted to visible text plus an explicit reference URL list; local paths are not passed into the agent.

## Rendering

The harness tries Playwright first for network-blocked rendering and interaction smoke. If the Python Playwright package is unavailable, it uses local Chrome DevTools Protocol to capture desktop/mobile emulated screenshots and layout metrics. The older Chrome screenshot-only path is used only as a last fallback and is not sufficient for mobile acceptance.

## Results

- Total generated/attempted runs: 370
- Completed hard-fail pass: 360
- Hard-failed: 9
- Runner failed: 0
- Incomplete/partial: 1
- Render passed among completed: 360

| Variant | Runs | Mean bytes | Byte stdev | Mean lines | Mean nav controls | Mean tables | Mean card tokens | Mean source links | Render passed | Mean duration sec |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `DH10` | 4 | 65914.75 | 764.53 | 1535.75 | 9.75 | 2.25 | 27.5 | 0.0 | 4 | 750.67 |
| `DH11` | 24 | 81689.62 | 14040.47 | 269.67 | 8.96 | 25.5 | 70.21 | 0.0 | 24 | 334.36 |
| `DH12` | 12 | 70115.0 | 9918.96 | 306.17 | 8.83 | 19.0 | 73.33 | 2.92 | 12 | 290.58 |
| `DH13` | 28 | 100555.71 | 28966.84 | 303.57 | 8.82 | 16.68 | 68.11 | 3.75 | 28 | 306.61 |
| `DH14` | 24 | 101515.79 | 24330.49 | 324.88 | 8.62 | 15.75 | 7.33 | 5.12 | 24 | 294.19 |
| `DH15` | 24 | 102990.5 | 18119.27 | 376.42 | 8.79 | 14.21 | 7.46 | 5.12 | 24 | 309.04 |
| `DH16` | 24 | 103053.92 | 15969.13 | 303.58 | 8.67 | 18.21 | 66.58 | 4.38 | 24 | 318.88 |
| `DH17` | 24 | 95793.88 | 17543.15 | 286.75 | 8.75 | 14.0 | 80.33 | 4.75 | 24 | 326.46 |
| `DH18` | 24 | 108173.0 | 25035.44 | 385.38 | 8.83 | 18.79 | 82.58 | 5.5 | 24 | 341.01 |
| `DH19` | 24 | 98872.21 | 18452.09 | 547.62 | 8.71 | 13.54 | 87.71 | 5.5 | 24 | 309.41 |
| `DH2` | 7 | 44127.71 | 4857.58 | 826.0 | 37.71 | 2.71 | 31.14 | 0.0 | 7 | 323.82 |
| `DH20` | 24 | 103078.62 | 26088.88 | 546.96 | 8.62 | 14.21 | 86.5 | 5.5 | 24 | 314.29 |
| `DH21` | 24 | 112662.62 | 22627.27 | 709.88 | 8.46 | 12.38 | 90.54 | 5.5 | 24 | 305.43 |
| `DH22` | 24 | 105116.29 | 26035.33 | 614.62 | 8.62 | 15.75 | 83.17 | 5.5 | 24 | 321.68 |
| `DH23` | 24 | 108954.67 | 30222.24 | 649.75 | 8.71 | 16.54 | 83.75 | 5.5 | 24 | 303.97 |
| `DH24` | 24 | 112115.38 | 21839.94 | 790.08 | 8.58 | 15.62 | 83.96 | 5.5 | 24 | 318.46 |
| `DH4` | 2 | 57011.0 | 8343.0 | 1081.0 | 33.0 | 1.0 | 39.5 | 0.0 | 2 | 348.59 |
| `DH5` | 2 | 42953.0 | 2478.0 | 918.5 | 7.5 | 2.0 | 14.0 | 0.0 | 2 | 282.3 |
| `DH6` | 6 | 49666.0 | 3279.09 | 1113.0 | 51.17 | 1.83 | 10.5 | 0.0 | 6 | 320.12 |
| `DH7` | 4 | 40093.0 | 7675.63 | 895.25 | 13.0 | 2.0 | 24.0 | 0.0 | 4 | 244.2 |
| `DH8` | 4 | 46054.0 | 5570.39 | 939.25 | 12.25 | 1.0 | 19.0 | 0.0 | 4 | 320.28 |
| `DH9` | 3 | 54455.0 | 9726.91 | 1027.0 | 24.67 | 7.0 | 18.33 | 0.0 | 3 | 382.35 |

Hard-fail summary:

- `DH0`: 2 hard/runner failures (mobile_horizontal_overflow: 2)
- `DH1`: 3 hard/runner failures (incomplete_status:started: 1, mobile_horizontal_overflow: 2)
- `DH2`: 3 hard/runner failures (mobile_horizontal_overflow: 3, source_reference_section_missing: 1)
- `DH3`: 1 hard/runner failures (mobile_horizontal_overflow: 1)
- `DH9`: 1 hard/runner failures (mobile_horizontal_overflow: 1)

## Review And Pruning

Current decision: DH11 crossed the main structural threshold by splitting agent-authored content models from deterministic mobile-safe rendering. DH12 and DH13 are strong reference-grade candidates on visual identity, source preservation, and SVG diagram axes. DH14 probes the remaining composition-diversity gap: whether different subjects can use different page skeletons instead of the same hero/tab/card structure. DH15 tests whether that variety can coexist with interaction and density. DH16 responds to the blind-review result that DH13's first viewport beat DH15: it locks the first viewport back to DH13's dense hero-map while keeping composition planning below the fold. DH17 tests the next observed gap: the Google I/O reference is more compact and information-architecture-driven than the DH16 oversized hero.

The remaining product question is no longer whether a polished report-app artifact can be generated at all. The harder question is which mode should be exposed: DH12 is lighter and visually adaptive, DH13 is heavier but adds true diagrammatic structure, DH14 tests reference-grade structural variety, DH15 tests whether that variety can coexist with interaction and density, DH16 tests whether the DH13 first-screen contract is the non-negotiable base layer, and DH17 tests whether a compact reference-app information architecture is the missing refinement layer.

Claude generation was sampled once at the branch point as an upper-bound check. That run did not pass the current gates because it wrapped the HTML in explanatory text/code fencing and still produced a clipped mobile navigation. A broader Claude run should wait until the CLI contract is fixed to raw-HTML-only output.

Product adoption still requires a separate implementation plan, but the experiment now supports adopting a deterministic content-model renderer path instead of more prompt-only tuning.

## Experiment Memory

Keep this section as the lightweight experiment ledger. Do not rely on memory or
the raw `runs/` tree when choosing the next variant.

What to keep in git:

- this README, including the variant ladder and aggregate metrics
- `decision-dh23.md`
- harness code and self-tests
- lightweight blind-review summaries and answer-key-mapped scores when present
- selected review upload manifests or links

What not to keep in git by default:

- the full generated `runs/` corpus
- desktop/mobile screenshots for every generated artifact
- private answer keys unless the repository policy explicitly allows them

Important lessons so far:

- Prompt-only HTML generation repeatedly produced attractive but brittle
  artifacts, mobile overflow, missing source sections, or markdown-like pages.
- The strongest structural shift was `DH11`: ask the agent for a content model,
  then use a deterministic renderer.
- `DH12`/`DH13` showed that visual identity and inline SVG diagrams improve
  artifact quality without giving up mobile safety.
- `DH14`/`DH15` tested page skeleton diversity, but the stronger first viewport
  from the earlier diagram path still mattered.
- `DH17` through `DH20` showed that compact reference-app information
  architecture was a real improvement, and `DH20` removed the reference-gallery
  prompt from the content-model path to reduce overfitting risk.
- `DH21` is contaminated for ablation because it still included the reference
  gallery profile. Do not cite it as clean reference-free evidence.
- `DH22` showed that promoting the strongest visual unit into the first
  viewport improved topic identity.
- `DH23` is the current product candidate because the first viewport explains a
  real relationship and beat `DH22` in independent blind screenshot reviews.
- `DH24` did not displace `DH23`; compressing the first viewport and asking for
  varied visual grammar was not enough.

Current known limitation:

`DH23` still overuses one infographic grammar. Even when the model names a
visual as timeline, matrix, flow, decision route, causal field, or architecture
map, the renderer usually draws the same connected node/edge relationship SVG.
This can make different subjects feel like they share one reusable infographic
skin. The next material experiment should be a visual-grammar dispatcher with
multiple real renderers, not another "make the hero shorter" variant.

Reserved next hypothesis:

`DH25` should only run after the harness has a dispatcher and at least three
non-map renderers. Candidate grammars include timeline/causal field, swimlane or
trust boundary, role/dependency matrix, cost ladder, decision route, trade-off
board, and evidence/uncertainty chain. If `DH25` merely renames diagrams while
rendering the same relationship map, it repeats the `DH24` failure and should be
hard-failed.
