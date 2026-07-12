# Plasma C1 기본 루프

이 문서는 현재 Plasma의 기본 제품 흐름을 설명합니다. `C1`은 사용자에게 보여 주는 제품명이 아니라,
지금의 기본 동작 방식을 가리키는 내부 코드명입니다.

기본 흐름은 다음 순서로 움직입니다.

1. 사용자가 미션을 만들거나 엽니다.
2. 그 미션에서는 같은 agent provider session을 이어 씁니다.
3. 사용자 또는 사용자처럼 동작하는 controller가 다음 턴의 방향을 잡습니다.
4. 어떤 controller steering strategy가 선택되었고 왜 선택되었는지 장부에 남깁니다.
5. Agent는 MCP/source read 도구로 원본 자료를 읽습니다.
6. Agent의 답변은 conversation result로 저장합니다.
7. 보고서는 source가 아니라 Plasma가 소유한 artifact로 저장합니다.
8. 사용자는 report artifact를 보거나 다운로드합니다. 이 과정에서 legacy AST report로 변환하지 않습니다.

## 보고서 생성

보고서는 같은 artifact model 위에서 두 가지 방식으로 만들 수 있습니다.

- `보고서`: 계획을 세운 뒤 Markdown artifact를 만듭니다. Provider executor가 session fork를 지원하고
  미션에 기존 research session이 있으면, 보고서 생성은 report-only fork session에서 실행합니다. 이렇게
  해야 보고서를 만든 뒤에도 원래 research session으로 조사를 계속 이어갈 수 있습니다. Fork가 불가능하면
  같은 session으로 fallback하고, 그 이유를 기록합니다.
- `장문 보고서`: 더 느리지만 긴 보고서에 맞춘 Part/Section 경로입니다. 먼저 사람이 볼 수 있는 생성
  계획을 만들고, 각 section을 따로 작성한 뒤, section body를 보존하면서 part/final Markdown artifact를
  조립합니다.

나중에 plan review 단계를 추가할 수는 있습니다. 그래도 최종 결과는 report artifact여야 합니다. Source나
legacy AST report version으로 바뀌면 안 됩니다.

## Source와 Result 경계

Source는 원본 자료입니다. pasted text, fetched URL, file, PDF, image, audio/video metadata reference,
Liquid2 document, connector material이 여기에 해당합니다.

Agent answer, controller output, rendered media caption, report는 result 또는 artifact입니다. 이것들은
source를 참조할 수 있지만 source로 재분류하지 않습니다.

## Source Read Policy

Source read에는 두 가지 mode가 있습니다.

- `snapshot_only`: Plasma가 고정해 둔 artifact를 읽습니다. URL, PDF, uploaded file, Liquid2 snapshot
  같은 자료가 여기에 해당합니다.
- `live_reference`: 변할 수 있는 원본 자료를 읽습니다. 현재는 allowlisted local path source에 사용합니다.

Accepted source snapshot은 `snapshot_id`와 선택적인 `subpath`로 읽습니다. Agent에게 임의의 absolute
path나 root-wide browsing 권한을 주지 않습니다.

Local path read, grep, tree는 `source.observed` event를 남깁니다. 이것은 observation metadata이지 legacy
evidence/claim record가 아니며, 파일을 snapshot으로 복사하지도 않습니다.

PDF URL source는 pinned `snapshot_only` artifact입니다. Read tool은 raw PDF bytes가 아니라 bounded
extracted text와 extraction metadata를 반환합니다.

반면 local path의 `.pdf` 파일은 live reference로 남습니다. 이 경우 파일을 읽을 때 `source.observed` event를
남기며, PDF URL source처럼 pinned snapshot으로 바꾸지 않습니다.

## Workflow Run

Bounded workflow run은 별도 제품 모드가 아니라 같은 C1 루프 안에 있습니다. 각 step은 controller-like
`workflow_steering` user turn을 기록하고, 같은 provider session을 이어 쓰며, agent response를 result로
저장합니다.

Workflow event는 진행 상태와 stop condition을 설명합니다. Workflow summary는 source가 아닙니다.

정상 turn과 workflow step은 agent가 유용한 새 원본 자료를 발견했을 때 source candidate review record를
만들 수 있습니다. Source candidate는 사용자가 승인하기 전까지 source가 아닙니다.

## Source Candidate Staging

URL source candidate는 가능하면 staging합니다.

1. `source.candidate.staging_started`를 기록합니다.
2. 성공하면 `source.candidate.staged`와 candidate-only raw artifact를 기록합니다.
3. 실패하면 `source.candidate.staging_failed`를 기록합니다.

Staged artifact는 승인 전 후보입니다. Agent는 중복 제안이나 낮은 가치의 제안을 피하기 위해 dedicated
candidate-read MCP tool로 읽을 수 있습니다. 하지만 default source list, normal raw artifact read, report
generation에는 포함하지 않습니다. 사용자가 승인해야 source snapshot이 생깁니다.

## Controller Strategy

Controller strategy selection은 관찰 가능한 steering event입니다. 짧은 guidance를 agent prompt에 더할 수는
있지만, strategy event 자체가 source, evidence, claim, confidence update, proposal bundle, report artifact를
만들면 안 됩니다.

2026-06-26 C0/PAL2/NAV 실험은 더 강한 always-on controller를 기본값으로 검증하지 못했습니다. NAV는 C0-like
baseline보다 나빴고, PAL2는 결론이 충분하지 않았습니다. 따라서 현재 제품 규칙은 보수적으로 잡습니다. 기본
turn은 같은 session의 C0 흐름에 가깝게 두고, controller behavior는 stuck, repetitive, too narrow, drifting
상태에서만 약한 conditional steering으로 사용합니다.

## Legacy Records

현재 기본 루프는 evidence, claim, confidence update, proposal bundle, AST report object를 만들지 않습니다.
기존 historical record와 legacy code는 read-only inspection, migration check, explicit developer experiment를
위해 보존합니다. 사용자에게 old/new mode toggle로 노출하지 않습니다.

나중에 evidence나 claim record를 되살리더라도, 이것들은 source-backed signal을 찾고 비교하고 설명하는 데
도움을 주어야 합니다. Source reading, investigation, report generation을 막는 gate가 되어서는 안 됩니다.

## MCP Path

기본 MCP path는 read-first입니다.

- `plasma.research.outline`
- `plasma.research.list`
- `plasma.research.grep`
- `plasma.research.read`
- `plasma.research.references`
- source read/search tools

Accepted live local path directory source에 대해서는 `plasma.sources.read`, `plasma.sources.tree`,
`plasma.sources.grep`이 source boundary 내부의 `subpath`를 사용할 수 있습니다.

이 도구들은 large mission prompt pack, source body stuffing, report-only corpus, root-wide local filesystem
browsing으로 대체하면 안 됩니다.

## Removed Sources

Soft-removed source는 default source list, read, reporting, workflow use에서 제외합니다. Audit history에는
남고, 명시적인 `include_removed` control이나 restore로만 다룹니다. Removal은 physical purge/redaction이
아닙니다.

Workflow 중 active source가 제거되면 다음 step은 `workflow.source.skipped`를 기록하고, 그 source를 조용히
사용하지 않습니다.

## Media-Aware Reports

Pinned image bytes는 policy가 허용할 때 self-contained interactive HTML export에 포함할 수 있습니다. 그래도
HTML은 source가 아니라 report artifact입니다.

Audio/video는 기본적으로 link 또는 allowlisted provider embed로 남깁니다. Markdown은 원본 media URL과
attribution을 보존해야 합니다.

## Designed HTML

Designed HTML export는 기존 report material 위에 얹는 추가 report artifact view입니다. 새 source type이
아니고 legacy AST report도 아닙니다.

현재 제품 slice는 DH23 실험과 visual-grammar update를 따릅니다. Agent가 JSON content model을 만들고,
deterministic renderer가 이를 self-contained HTML로 승격합니다. 첫 화면은 generic card strip이 아니라
report의 핵심 관계를 보여주는 compact connected relationship map을 우선합니다.

이것은 최종 visual system이 아닙니다. Renderer는 계속 교체 가능해야 합니다. Source/caveat, URL, 긴 텍스트
가독성을 decorative variety보다 우선해야 합니다.

## Long-Running Work

Report draft와 designed HTML generation은 browser-owned goroutine이 아니라 shared report runner boundary를
사용합니다. Pending work는 durable ledger state로 남아야 하며, in-memory worker가 사라졌다는 이유만으로 실패
처리하면 안 됩니다.

Active agent/MCP turn 안에서 workflow request가 들어오면 provider를 즉시 재귀 실행하지 않고 요청을 ledger에
기록합니다. 현재 provider turn이 terminal event를 가진 뒤 같은 provider session으로 이어서 실행합니다.
Report draft도 provider-backed work이므로, 같은 mission에 normal turn, workflow run, report draft가 active
상태라면 새 report draft를 시작하지 않습니다.

각 report draft는 pending 기록 전에 모델과 추론 강도를 요청 명시값, 같은 executor의 최신 미션 세션, 설정된 provider 기본값 순으로 한 번만 정합니다. 모델만 명시하면 해당 모델의 기본 추론 강도를 사용합니다. 유효하지 않은 조합은 pending이나 provider 작업을 만들지 않습니다. 복구는 동결된 pending 값을 사용하고 출처 없는 legacy pending만 별도 호환 경로를 유지하며 report prompt, mode, fork 동작, H5, patch, designed HTML은 변경하지 않습니다.
