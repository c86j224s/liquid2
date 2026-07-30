# 장문 보고서 최종화

Web 장문 보고서는 기존 계획, 섹션 작성, 파트 조립, 세션 정책, H5, 디자인
HTML 흐름을 그대로 사용한다. 기본 실행 전략은 순차 작성이다. 별도 장문 전용
"빠른 병렬" 선택지를 고르면 canonical 계획 세션에서 섹션 작성을 fanout한 뒤,
다시 같은 파트 조립과 최종화 계약으로 돌아온다.

두 전략 모두 Part edit과 staged final-edit 인계가 같다. 활성 narrative-contract
profile은 writing guidance가 쓰는 같은 profile contract에 따라 canonical 계획
event에 `part_edit_enabled: true`를 저장한다. 필드가 없는 저장된 legacy 계획은
false로 해석하고, 새 non-narrative profile 계획만 false를 저장한다. 정적
projection은 durable progress에 Part edit 단계가 있을 때만 그 단계를 렌더링하며,
legacy 계획에 새 단계를 합성하지 않는다. Part-connective narrative profile을 쓰는
명시적 `section_fanout` 요청은 같은 canonical 계획 event에
`part_planning_enabled: true`도 저장한다. 별도 capability event는 없다. 복구,
progress, 정적 projection은 이 저장된 계획 payload에서만 Part planning 여부를
도출하며, 필드가 없는 legacy와 serial 작업은 false로 남는다.

Part edit이 켜지면 post-assembly Part editor가 immutable source Part artifact
하나를 읽고 patch한 뒤, 새 edited Part artifact 또는 source artifact를 재사용하는
unchanged completion을 제출한다. 새 planned narrative 장문 보고서는
`final_edit_pipeline: assembly_writer_reader_style_gate_v2`를 저장한다. 서버는 먼저
검토된 Part들에서 agent session 없이 deterministic `final_assembly` artifact를 만든다.
그 다음 final writer가 전용 `plasma.report.long_form.final_write.*` tool로 최종 원고
작성 범위를 다루고, 별도 reader editor가 writer artifact를 검토한다. 기존
`post_report_humanize` 설정이 켜져 있으면 optional pre-canonical style edit이
실행되며, corrective provenance gate가 source와 requirement 경계를 확인한 뒤
정확히 하나의 canonical `report.artifact.created` event를 만든다.

Web 진행 화면은 이 순서를 `최종 조립`, `최종 작성`, `독자 편집`, `말투 편집`,
`근거·요구 교정`으로 표시한다. `말투 편집`은 `post_report_humanize`가 enabled인
실행에만 나타나며, 꺼진 실행에서는 나머지 단계 순서를 바꾸지 않고 생략된다.

`final_edit_pipeline: reader_style_gate_v1`가 저장된 계획은 replay와 중단 작업 복구를
위해 v1 reader/style/gate 경로를 유지한다. 이 경로에서는 서버가 검토된 Part artifact를
하나의 immutable reader-source Markdown artifact로 조립한 뒤 reader edit, optional
style edit, corrective gate만 실행하고 final writer를 두지 않는다.
`final_edit_pipeline` 필드가 없는 저장된 legacy 계획은 이전 장문 finalization 의미를
유지한다.

계획형 보고서와 CLI 보고서 동작은 이 명령을 사용하지 않는다.

## Part 조립과 Part 편집 도구

브라우저에는 시각자료 계획, 섹션 중심, 섹션 중심 + 풍부한 cluster memory라는 세
가지 글쓰기 선택지가 남는다. 독자 중심 작성 계약은 세 선택지 아래의 공통 기준이며
네 번째 선택지가 아니다. 내부적으로 새 요청은 서로 다른 composite profile 값을
사용하므로 저장된 legacy profile 값을 새 의미로 재해석하지 않는다.

세 선택지는 같은 Part assembly MCP 인계를 사용한다. Section을 읽는 Part assembler는
현재 Part에 바인딩된 immutable Section을 모두 bounded read한 뒤 intro, transition,
closing을 쓰고 `PART_ASSEMBLY_SUBMITTED` sentinel을 반환한다. 이 assembler는
post-assembly Part editor가 아니다.

Post-assembly Part editor는 `part_edit_enabled`가 true일 때만 실행된다. 이 editor는
source Part artifact 하나에만 바인딩되고 `plasma.report.part_edit.*` tool만 받는다.
source, research, Section, 다른 Part, final-edit tool은 읽을 수 없다. 제출된
outcome은 source Part event, source artifact, edited artifact, provider session,
profile metadata를 기록한다. No-op 제출도 내구 completion이다. source Part는
immutable input으로 보존되고, 중복 artifact를 만들지 않고 source artifact에 outcome을
바인딩한다.

공용 reporting start 계약은 provider-owned Part-edit draft를 로드하기 전에 정확히
하나의 canonical `report.part_edit.started` event를 쓴다. Web pre-start와 direct
MCP `plasma.report.part_edit.start` 호출은 모두 같은 `StartPartEdit` transaction을
사용하므로, Web pre-start 뒤 MCP replay는 두 번째 start를 만들지 않고 저장된 binding을
반환한다. Start payload는 `report.part.edited`와 같은 field 이름으로 정규화된 전체
binding을 저장한다. 여기에는 예정 edited artifact, filename, tool session, provider
session, previous provider session, requirement-map binding, model selection, session
policy, guidance profile, session chain, report-plan session, fork source가 포함된다.

## 실행 전략

`serial`은 기본 장문 전략이다. 계획, 각 섹션, 각 파트, 최종화를 기존 보고서
세션 순서대로 이어 간다.

`section_fanout`은 명시적으로 선택하는 브라우저 장문 옵션이다. 먼저 기존
`plasma.report.plan.submit` 경계로 canonical 계획을 만든다. 그 뒤 보고서 계획
공급자 세션을 fork해 섹션 작업자들이 독립적으로 작성한다. 각 섹션은 여전히
기존 섹션 프롬프트와 bounded source tool을 쓴다. 브라우저 실행기는 동시에 최대
8개의 섹션 작업자를 실행한다. 파트 조립은 해당 파트의 섹션 artifact가 모두
끝난 뒤 시작하며, 섹션 본문을 보존한다. W4가 켜진 `section_fanout`에서는 섹션
작성 전에 Part마다 정확히 하나의 내구 Part 계획 event를 만든다. 섹션 writer와
Part assembler는 그 Part-owner provider session에서 fork되고, 최종 Part author는
기계적 조립 뒤 같은 Part-owner session을 resume해 기존 closed
`plasma.report.part_edit.*` tool로 작업한다. 최종화 경로는 저장된
`final_edit_pipeline`만 기준으로 결정한다. 새 v2 계획은 deterministic final assembly,
report-plan provider session에서 fork한 final writer, 같은 report-plan session의
독립 sibling reader, reader provider session에서 fork한 optional style editor, 같은
report-plan session의 sibling corrective gate 순서로 실행된다. 저장된 v1 계획은
final writer나 final-assembly progress 없이 reader/style/gate를 계속 실행한다. 저장된
legacy profile은 `plasma.report.long_form.finalize`를 계속 사용하며 staged final-edit
단계를 실행하지 않는다.

선택한 전략은 `report.draft.pending`의 `execution_strategy`에 저장되어 재시작과
stale 복구가 같은 경로를 사용한다. 값이 없거나 `serial`이면 기존 순차 동작이다.
`section_fanout`은 계획형, 원테이크, CLI, H5, patch, 디자인 HTML 요청에는 사용할
수 없다.

## 공개 도구 계약

활성 Part editor tool은 완전한 숨은 실행기 binding과 명시적 도구 활성화가 있는
전용 Part edit session에서만 노출된다.

- `plasma.report.part_edit.start`는 source Part 하나에 대한 bounded Part-edit draft를 만든다.
- `plasma.report.part_edit.read`는 source Part에서 bounded UTF-8 slice를 반환한다.
- `plasma.report.part_edit.patch`는 Part-edit draft에만 bounded exact edit을 적용한다.
- `plasma.report.part_edit.submit`은 edited outcome 또는 unchanged completion을 canonical Part-edit transaction으로 commit한다.

v2의 deterministic final assembly는 서버 소유 제품 단계다.
`report.final_assembly.created`를 producer `reporting_final_assembly`, schema
`plasma.final_assembly.v1`로 만들며 agent session과 MCP tool이 없다. Final writer tool은
그 assembly에 바인딩된 writer-stage session에서만 노출된다.

- `plasma.report.long_form.final_write.start`
- `plasma.report.long_form.final_write.read`
- `plasma.report.long_form.final_write.patch`
- `plasma.report.long_form.final_write.submit`

Writer는 whole-report opening, conclusion, Part transition, 전역 connective logic을
추가하거나 조정할 수 있고, unique fact, number, condition, citation, uncertainty,
owner requirement를 잃지 않는 범위에서 cross-Part duplicate paragraph를 병합하거나
옮길 수 있다. Research, external fact 추가, 전체 Part/Section reorder, 고정 Part order
변경은 할 수 없다.

활성 reader edit tool은 완전한 숨은 실행기 binding과 명시적 도구 활성화가 있는
reader-stage session에서만 노출된다.

- `plasma.report.long_form.reader_edit.start`는 검토된 Part에서 조립한 immutable reader-source Markdown을 바탕으로 bounded reader draft를 만든다.
- `plasma.report.long_form.reader_edit.read`는 bounded UTF-8 slice를 반환한다.
- `plasma.report.long_form.reader_edit.patch`는 bounded exact replace, insert-after, append operation을 적용한다.
- `plasma.report.long_form.reader_edit.submit`은 edited outcome 또는 unchanged completion을 durable stage submission으로 commit하며 canonicalize하지 않는다.

저장된 계획의 정규화된 `post_report_humanize`가 enabled이면 optional pre-canonical
style tool도 같은 stage-scoped 계약으로만 노출된다.

- `plasma.report.long_form.style_edit.start`
- `plasma.report.long_form.style_edit.read`
- `plasma.report.long_form.style_edit.patch`
- `plasma.report.long_form.style_edit.submit`

Corrective gate는 기존 final-edit tool 이름을 재사용하지만, 완전한 gate binding과
matching final binding이 모두 있는 gate-stage session에서만 노출된다.

- `plasma.report.long_form.final_edit.start`
- `plasma.report.long_form.final_edit.read`
- `plasma.report.long_form.final_edit.patch`
- `plasma.report.long_form.final_edit.submit`

Gate는 support가 불명확한 claim을 확인하기 위해 approved read tool을 사용할 수 있다.
Gate는 무조건 보고서를 줄이거나 검열하는 단계가 아니다. source/evidence 경계 위반,
owner-bound requirement 위반, 지원되지 않는 claim처럼 승인된 repair action이 필요한
문제만 교정한다. Gate finding은 서버가 계산한 statement hash, classification, repair
action, approved evidence ID만 저장한다. Raw statement text는 transient tool input일
뿐 저장하지 않는다.

에이전트는 artifact ID, 파일명, 제목, 보고서 모드, 파트와 섹션 순서, 공급자
provenance, 모델 설정, binding identity를 선택할 수 없다. Stage artifact ID나
canonical event ID도 선택할 수 없다. Reader와 style 단계는 source/research를 읽을
수 없고 Section, source Part, edited Part, reader-source, prior-stage artifact를
mutate할 수 없다. Gate만 canonical finalization event를 만들 수 있다. Legacy
`plasma.report.long_form.finalize`는 저장 profile compatibility를 위한 closed
opening/closing input에만 묶여 남으며, 저장된 계획에 active pipeline 필드가 없을 때
사용된다. 이 값은 서버가 binding하며 commit 전에 mission ledger와 raw artifact에
다시 대조한다.

Staged pipeline에서 writer, reader, style submission은 durable intermediate event일
뿐이다. Corrective gate submission 뒤 같은 gate/finalization 경계에서 정확히 하나의
canonical event를 만들거나, 복구가 저장된 gate submission에서 이어 받아 완료된
provider를 다시 실행하지 않고 한 번만 canonicalize한다. No-op gate는 alias artifact를
만들지 않고 이전 durable artifact를 canonical로 채택한다. 변경된 gate만 예정된 final
artifact를 만든다. 같은 binding과 content SHA는 기존 결과를 replay한다. 식별자,
provenance, 파트 순서, idempotency key, pipeline marker, stage lineage,
approved-evidence state, 내용이 다르면 재시작이나 동시 호출 뒤에도 conflict다. 이
조건부 트랜잭션은 현재 ledger 상태를 기준으로 함께 판정하므로, pending 보고서의
terminal event와 최종 canonical artifact/event 생성은 경합할 수 없다.

## 완료와 재시도

Writer/reader/style의 `FINAL_EDIT_STAGE_SUBMITTED`와 gate의 `REPORT_FINALIZED`는
정상 acknowledgement 문자열이지만 완료 여부의 권위 상태는 아니다. 실행기는 각 provider
호출 뒤 durable state를 다시 읽는다. Matching writer/reader/style submission이 있으면
반환된 문자열과 관계없이 채택하고, gate는 durable submission과 canonical report artifact
event가 모두 있을 때 완료된다. Gate submission만 있고 canonical event가 없으면 복구가
저장된 submission에서 canonicalization을 완료하며 provider를 다시 실행하지 않는다.
필요한 durable state가 없을 때만 각 stage에 기술 재시도를 한 번 허용한다. 두 호출은
같은 durable binding, tool session, idempotency key, artifact identity,
provider-session chain을 재사용하며 계획, 섹션, Part, 완료된 final-edit stage는
반복하지 않는다.

`resume_failed`는 실패한 시도의 조상 chain에서 검증된 계획, 섹션, Part, Part-edit
outcome만 재사용한다. 실패한 시도가 Part assembly까지 도달했지만 Part edit
completion 전 실패했다면, resume된 Part editor는 accepted ancestor Part에
바인딩된다. 이전 Part edit이 no-op으로 끝났다면 downstream finalization chain은
artifact ID가 immutable source Part와 같아도 durable edited outcome lineage를
사용한다. 실패한 시도를 다시 열거나 바꾸지 않는다. `restart`는 새 lineage에서 시작하며 조상 Part
출력을 재사용하지 않는다.

계획 event에 `part_planning_enabled`가 저장되어 있으면, 복구는 Part plan이 아직
하나도 기록되기 전 crash였더라도 Part-owner 경로를 계속해야 한다. 기존 Part plan은
현재 pending report, report plan session, Part index, owner session, fork source,
저장된 envelope/provenance 제약이 모두 맞을 때만 받아들인다. Retry 요청이 다른
brief를 들고 와도 replay는 저장된 canonical brief를 검증하고 반환하며, retry의 새
brief와 비교하지 않는다. 누락, 중복, malformed, wrong-Part, wrong-plan, wrong-session,
stale Part plan은 거부한다. `resume_failed`는 accepted ancestor Part plan을 재사용할 수
있지만 `restart`는 재사용하면 안 된다. Part-plan terminal companion은 `part-plan-N`
failure ID와 `report.part_plan.failed`를 쓴다.

열려 있는 Part-edit start 복구는 exact-current-pending에만 적용된다. W3 separate Part
editor와 W4 final Part author는 provider fork, tool session, artifact ID, filename을
새로 만들기 전에 현재 pending에 대해 정확히 하나의 유효한 `report.part_edit.started`
event가 있는지 확인하고, 있으면 그 저장된 binding을 그대로 채택한다.
`FinalizePartEdit`는 정확히 하나의 matching valid start가 없는 edited outcome을 모두
거부한다. 새 `resume_failed` pending이나 `restart`는 조상 partial start를 재사용하면
안 되며, 완료된 accepted outcome은 기존 accepted-lineage replay 규칙을 유지한다.

첫 응답이 정확히 `front_matter`와 `closing` 문자열만 가진 legacy 객체이고 루트
trailing comma가 정확히 하나일 때만 재시도 힌트를 만들 수 있다. scanner는 그
쉼표 하나만 제거한다. 정상 JSON, fence, 앞뒤 설명, 추가 값, 알 수 없거나 중복된
필드, 중첩 trailing comma, 잘린 입력은 거부한다. 복구된 글은 두 번째 공급자
호출을 위한 비내구 참고값일 뿐이며 Web 코드가 artifact나 event를 만드는 데
사용하지 않는다.

명령이 commit됐지만 exact sentinel이 없으면 실행기는 acknowledgement 문자열만을
이유로 재실행하지 않고 durable submission이나 canonical event를 채택한다. 첫 호출 뒤
matching durable state가 없을 때만 두 번째 호출이 한 번 허용된 기술 재시도가 된다.
Stage submission이 생기기
전에 provider가 두 번 실패하면 실행기는 기존 `report.final.failed` companion과
해당 pending attempt의 terminal `report.draft.failed`를 기록한다. 이미 완료된 중간
artifact는 보존되며, 두 번째 실패 뒤에는 canonical completion을 차단한다.

## Provenance와 관측

공개 도구의 `producer`는 기존 MCP tool-session 관례를 따른다. 최종 artifact와
canonical event producer는 서버가 binding한 실제 보고서 공급자 세션을 쓴다.
canonical payload는 기존 보고서 metadata를 보존하고 final tool session을 별도로
기록한다. 도구 호출 뒤에야 알 수 있는 공급자 usage를 canonical event에 만들어
넣거나 대화 ledger event에 기록하지 않는다. redacted 운영 로그에는 반환 세션의
존재 여부와 bound 세션 일치 여부, token 집계, duration만 남으며, 반환 세션 ID나
공급자 usage 상세를 canonical 상태에 기록하지 않는다.

공용 `mcp.tool.called` payload는 변경하지 않는다. tool name, success, created event
ID를 canonical 보고서 provenance와 결합해 경로를 검증할 수 있으며 opening,
closing, prompt, 전체 보고서 본문은 trace 요약에 기록하지 않는다.
