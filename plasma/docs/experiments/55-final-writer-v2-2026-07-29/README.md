# Final Writer V2 Fixed-Input Experiment

Date: 2026-07-29

Status: product adoption approved by explicit user decision; sealed model reading remains mixed/inconclusive

Related issue: #190

## Purpose

This experiment records a bounded, product-faithful comparison between the
current stored v1 final-edit path and the new final-writer v2 path, followed by
the explicit product decision made from that evidence.

The A path is current `reader_style_gate_v1`. The B path is
`assembly_writer_reader_style_gate_v2`. Both paths must consume the same frozen
reviewed Part artifacts for each pair, so the comparison isolates only the final
assembly and final-edit pipeline difference.

The first W6-A execution is retained only as invalid directional evidence:
Sentinel rejected it because the frozen Parts were hand-authored, too small, and
not realistic Korean reviewed Parts from the upstream product path. The corrected
W6-B run fixes that by generating and freezing reviewed Korean Parts through the
actual upstream product path, then discarding those preparation final reports
before the fixed-input A/B terminal comparison.

The corrected W5/W6-B harness now fails closed before W6 can accept evidence.
Each pair must provide the exact preregistered path, pipeline, output path, and
one archive-local SHA-256 receipt for the frozen Part manifest; A and B must
present the same manifest digest, not merely the same path string.

## Current Result State

The corrected W6-B run generated all eight Markdown reports for the four
preregistered pairs on 2026-07-29. Citation, requirement, stage/tool, archive,
payload, and blind-pack checks passed, but a later audit found that the frozen
Part provenance check trusted self-described metadata too far and that the
private blind mapping had been rewritten after the first manual adjudication.
The reports remain reusable; the acceptance result does not.

The first direct reading preferred B in three pairs and found one pair tied:

| Pair | Blind Preference | Resolved Arm |
| --- | --- | --- |
| `wang-anshi-northern-song-exploratory` | Report 2 | B |
| `wang-anshi-northern-song-strict` | Report 2 | B |
| `go-raft-implementation-roadmap-exploratory` | Report 2 | B |
| `go-raft-implementation-roadmap-strict` | Tie | Tie |

B is `assembly_writer_reader_style_gate_v2`; A is `reader_style_gate_v1`.
After the private mapping was revealed, an independent host artifact reread
judged two pairs for B and two pairs tied. It downgraded the Wang exploratory
pair from B to tie because B improved the opening and Part transitions but also
introduced repeated source-interpreter phrasing such as references to what the
source says. The Wang strict and Go Raft exploratory B reports remained better;
the Go Raft strict pair remained tied.

Because the mapping was not durably sealed before that adjudication, the first
reading is retained only as directional evidence. The host reread was performed
after the mapping was known and is likewise diagnostic rather than a blind
acceptance result. Together they show no observed B regression but do not prove
that v2 has superior prose quality.

The superseded unsealed aggregate and its associated mapping and reading packs
have been preserved under the archive-local invalid-evidence directory. The
active `control/reading-results.json` was removed before a fresh private mapping
and blind-pack digest set were sealed.

W6-C replayed the preparation databases, exported ledgers, source snapshots,
source bytes, reviewed Part artifacts, and final report inputs without changing
the eight reports. Two independent model readers then read all four freshly
sealed pairs end to end. Their resolved results conflicted: one reader selected
A twice and B twice, while the other selected A three times and B once. They
agreed only that A was better for the Wang Anshi exploratory pair. Reader
disagreement is not treated as a report tie and neither reader is selected after
the mapping is revealed. The sealed model comparison therefore remains mixed
and inconclusive.

On 2026-07-30, the user explicitly approved product adoption on a narrower
basis: the change separates previously concentrated finalization
responsibilities and creates room for later improvement, while preserving
current quality is sufficient for this adoption. This is a product decision,
not a claim of completed blind human adjudication or proof that v2 already has
superior prose quality.

No prompt-only correction was used. The optional style stage did not run in this
execution because `post_report_humanize` stayed at the product default disabled
setting.

See [decision-memo.md](decision-memo.md) for the concise reading conclusion.

## 한국어 요약

이 문서는 W6 실행과 공개 결론을 기록한다. A는 현재 v1
`reader_style_gate_v1` 경로이고, B는 `assembly_writer_reader_style_gate_v2` 경로다.
각 pair에서 A와 B는 같은 frozen reviewed Part artifact를 입력으로 받아야 하며,
비교 대상은 final assembly와 final-edit pipeline 차이로만 제한한다.

수정된 W5 harness는 W6 evidence를 fail closed로 검증했다. 각 pair는 사전 등록된
경로, pipeline, output path, frozen Part manifest SHA-256 receipt를 정확히 제공해야
하며, A와 B는 같은 path 문자열뿐 아니라 같은 manifest digest를 제시해야 한다.
W6-A 결과는 hand-authored Part 입력 때문에 유효한 채택 근거가 아니며, W6-B에서
실제 upstream product path가 만든 한국어 reviewed Part를 다시 고정했다. 보고서
생성과 기존 hard gate는 완료됐지만, 이후 감사에서 Part provenance 검증이 메타데이터를
과신했고 첫 수동 판정 뒤 private mapping이 다시 기록된 사실이 확인됐다. 첫 판독의
세 pair B 우세·한 pair 동률과 매핑 공개 뒤 재독의 B 우세 두 pair·동률 두 pair는
방향성 기록으로만 보존한다. 기존 보고서를 다시 생성하지 않고 provenance를 재검증한
뒤 새 mapping으로 봉인된 두 모델 직접 독해를 완료했다. 두 판독은 세 pair에서
엇갈렸으므로 이견을 무승부로 바꾸거나 한 판독만 사후 선택하지 않는다. 모델 판독
결과는 mixed/inconclusive로 남긴다. 별도 host 직접 재독에서는 B 우세 두 pair와 동률
두 pair로 비열화가 관찰되지 않았다. 사용자는 즉시 품질 우위를 증명했다는 뜻이 아니라,
몰려 있던 최종화 책임을 분리해 이후 개선 여지를 확보하고 현재 품질을 보존하면 충분하다는
기준으로 제품 채택을 명시적으로 승인했다. 이는 완결된 인간 blind 판독 결과로 기록하지 않는다.

## Pair Matrix

| Pair | Topic | Rigor | A | B |
| --- | --- | --- | --- | --- |
| `wang-anshi-northern-song-exploratory` | Wang Anshi and Northern Song reform memory | Exploratory | current v1 | final-writer v2 |
| `wang-anshi-northern-song-strict` | Wang Anshi and Northern Song reform memory | Strict | current v1 | final-writer v2 |
| `go-raft-implementation-roadmap-exploratory` | Go Raft implementation roadmap | Exploratory | current v1 | final-writer v2 |
| `go-raft-implementation-roadmap-strict` | Go Raft implementation roadmap | Strict | current v1 | final-writer v2 |

## Product Boundary

V1 remains the compatibility baseline: reviewed Parts are assembled into a
reader-source manuscript, then reader edit, optional style edit, and corrective
gate run through their existing MCP partitions.

V2 uses the approved product sequence:

1. deterministic final assembly in product code,
2. final writer through `plasma.report.long_form.final_write.*`,
3. independent reader editor through `plasma.report.long_form.reader_edit.*`,
4. optional style editor through `plasma.report.long_form.style_edit.*`,
5. corrective gate through `plasma.report.long_form.final_edit.*`.

No new writing system, collector, user option, database schema, documentation
surface, or experimental product path is introduced by this preparation.

Stage traces must include the declared order, pipeline, fork source,
`source_artifact`, `canonicalizes`, required events, and the exact enabled MCP
tool set. Provider stages expose the full start/read/patch/submit surface for
their partition. Server-only assembly stages expose no MCP tools.

Acceptance evidence must contain exactly four unique preregistered pair records.
Each record must include an explicit winner (`A`, `B`, or `tie`), explicit
boolean `hard_fail`, explicit boolean `v2_structural_regression`, every
hard-fail count/status field, and explicit `stage_trace_errors`. Missing or
malformed fields reject the evidence instead of defaulting to pass.

## Archive Boundary

The repository stores this public contract, the reusable harness, and harness
tests only. Raw Part inputs, generated reports, private A/B mapping, blind
Markdown reading packs, machine-check payloads, provider traces, session IDs,
and run databases stay outside Git under:

```text
~/research-artifacts/liquid2/plasma/experiments/55-final-writer-v2-2026-07-29/
```

See [protocol.md](protocol.md) for the full preregistered method.

The default blind assignment uses private local randomness. A deterministic
`--blind-seed` is available only for tests or reproducibility checks and must
not be published as the experiment's default mapping source. Once the valid
mapping exists, verification reuses it so reruns cannot rewrite the original
Report 1/Report 2 reading identity.
