# Controller / Generator / MCP Isolation Experiment

This directory contains the protocol, execution artifacts, blinded judging
packets, and nine-block analysis for the Plasma controller/generator/MCP
isolation experiment dated 2026-06-22.

The experiment preserves a 2x2x2 paired block design:

| Cell | Controller | Generator session | MCP surface |
| --- | --- | --- | --- |
| `C0G0M0` | `C0` | `G0` | `M0` |
| `C0G0M1` | `C0` | `G0` | `M1` |
| `C0G1M0` | `C0` | `G1` | `M0` |
| `C0G1M1` | `C0` | `G1` | `M1` |
| `C1G0M0` | `C1` | `G0` | `M0` |
| `C1G0M1` | `C1` | `G0` | `M1` |
| `C1G1M0` | `C1` | `G1` | `M0` |
| `C1G1M1` | `C1` | `G1` | `M1` |

Factor definitions:

- `C0`: no controller. No `controller.strategy.selected` event is expected.
- `C1`: question-only controller. Controller output is only the next user-style
  steering question plus audit metadata.
- `G0`: final report resumes the investigation provider session.
- `G1`: final report uses a separate report-generator provider session with
  allowed tools only.
- `M0`: source-only surface. Generator can use original source list/read/grep
  and source catalog metadata only.
- `M1`: C1 research surface. Generator can additionally use bounded
  conversation result, tool-trace, and report-artifact state. Final factual
  claims still require original source reads.

Hard-fail controls are experiment harness rules, not Plasma product UX. A
contaminated run is excluded from primary analysis and must be rerun, replaced,
or cause the block to be marked unusable for paired claims.

Current execution status:

- Protocol, run index, source corpora, execution harness, blinded judge packets,
  and analysis outputs are present.
- All 72 planned cells completed cleanly across nine paired blocks.
- All 72 final reports were blindly judged.
- `tools/validate_experiment.py` passes.
- The nine-block analysis is in `analysis/decision_memo.md` and
  `analysis/factor_effects.csv`.
- A follow-up `G0`-only slice is in `analysis/g0_slice_decision_memo.md` and
  `analysis/g0_slice_factor_effects.csv`. It fixes report generation to the
  same investigation session and leaves controller and MCP-surface effects
  unresolved.
- The six-block corrected first look was superseded by the nine-block look after
  adding M7-M9 and force-rejudging all 72 packets.

Nine-block conclusion:

- `C1` question-only controller did not produce a statistically supported
  improvement. It also did not produce a supported harm signal.
- `G1` separate report generation produced a supported depth harm and should not
  be used as the default report path from this experiment.
- `M1` additional generated-context research surface did not produce a
  statistically supported improvement. It also did not produce a supported harm
  signal.
- The result does not invalidate controller-led steering as a product direction.
  It does argue against splitting final report generation into a separate
  default provider session without a stronger design.
- Controller and MCP-surface factors remain unresolved. They should not be
  promoted as defaults from this experiment, but this experiment also does not
  prove them ineffective.
- The `G0`-only slice supports the same boundary: once separate report
  generation is removed from the comparison, controller and MCP-surface effects
  still do not have supported wins or supported harms in this dataset.
