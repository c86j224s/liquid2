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
`final_edit_pipeline: assembly_writer_reader_style_validation_evidence_gate_v3`를 저장한다. 서버는 먼저
검토된 Part들에서 agent session 없이 deterministic `final_assembly` artifact를 만든다.
그 다음 final writer가 전용 `plasma.report.long_form.final_write.*` tool로 최종 원고
작성 범위를 다루고, 별도 reader editor가 writer artifact를 검토한다. 기존
`post_report_humanize` 설정이 켜져 있으면 optional pre-canonical style edit이
실행된 뒤 read-only `style_semantic_validation`이 의미 보존 여부를 판정한다.
마지막 `evidence_gate`는 report-to-evidence 연결만 판단하고, 서버가 bound artifact를
정확히 하나의 canonical `report.artifact.created` event로 확정한다.

Web 진행 화면은 이 순서를 `최종 조립`, `최종 작성`, `독자 편집`, `말투 편집`,
`말투 의미 검증`, `근거 연결 검증`으로 표시한다. `말투 편집`과 `말투 의미 검증`은
`post_report_humanize`가 enabled인 실행에만 나타나며, 꺼진 실행은 reader edit에서
바로 `evidence_gate`로 간다.

`final_edit_pipeline: reader_style_gate_v1`가 저장된 계획은 replay와 중단 작업 복구를
위해 v1 reader/style/gate 경로를 유지한다. 이 경로에서는 서버가 검토된 Part artifact를
하나의 immutable reader-source Markdown artifact로 조립한 뒤 reader edit, optional
style edit, corrective gate만 실행하고 final writer를 두지 않는다.
`final_edit_pipeline` 필드가 없는 저장된 legacy 계획은 이전 장문 finalization 의미를
유지한다.

계획형 보고서와 CLI 보고서 동작은 이 명령을 사용하지 않는다.

## Part 조립과 Part 편집 도구

브라우저는 더 이상 보고서 글쓰기 선택지를 노출하지 않는다. 새 장문 Web 요청은
`section-brief-cluster-memory-narrative-contract` rich section-centered composite
profile을 기본값으로 쓰고, 장문이 아닌 요청은 `narrative-contract`를 기본값으로 쓴다.
독자 중심 작성 계약은 공통 기준이며, 저장된 event와 direct API 호출을 새 의미로
재해석하지 않기 위해 older profile 값만 계속 허용한다.

활성 장문 기본값에서 계획자는 독자의 반응이나 의무적인 놀라움을 설계하지 않는다.
원자료가 실제로 뒷받침하는 기제, 비교, 인과 순서, 묶음, 긴장, 평가 질문으로 각
Section의 목적과 전개를 정한다. Section 작성자는 그 계획을 해설하지 않고 주제를
직접 쓴다. 각 문장은 주장, 사실, 기제, 구분, 결과, 한계 중 하나를 진전시켜야 하며,
구체적인 내용을 추상적으로 다시 포장하는 문장, 근거 없는 장식적 대조, 이미 끝난
문단의 결론 반복을 덧붙이지 않는다. 문단 길이와 맺음을 한 형식으로 맞추거나
접속사를 금칙어로 다루지도 않는다.

Section 작성자는 먼저 title과 purpose가 약속한 주요 설명과 catalog metadata,
판본·provenance 비교, 전승 주의, source 비교 표 같은 보조 작업을 구분한다. 자료비평,
서지, 전승, 소장 이력이 명시적으로 Section의 주요 주제인 경우에만 보조 작업이
본문의 중심이 될 수 있다. 현재 근거가 보조 작업만 뒷받침한다면 작성자는 계획된
설명을 source tour로 대체하지 않고 evidence gap을 반환한다. 최종 재시도에서도 이
근거 기준을 낮추지 않는다.

Section 작성자의 유효한 결과는 Markdown Section draft 또는 정확한 control response
`SECTION_EVIDENCE_GAP` 둘뿐이다. Gap은 fixed reason code
`inadequate_section_evidence`, 현재 pending/plan ID, 1-based Part/Section 좌표,
attempt number, provider/tool session lineage, duration, 표준 `agent_usage`만 담은
`report.section.evidence_gap`을 기록한다. Section artifact나
`report.section.created` event는 만들지 않으며, free-form diagnosis나 source content는
저장하지 않는다. 실행기는 같은 provider session과 tool-session binding에서 해당
Section만 한 번 재시도한다. Attempt 2에서는 기존 Section title/purpose 안에서 마지막
replacement search 또는 bounded scope reduction을 수행한 뒤 Markdown이나 정확한 gap
token을 반환한다.

Attempt 2에서도 gap으로 끝난 좌표가 있으면 실행기는 해당 retry lineage에서 계획
보정을 정확히 한 번만 허용한다. 원래 report-plan session의 계획자는 읽기 전용
research/source 도구로 실패 좌표를 함께 검토하고, 같은 좌표의 title, purpose,
`target_refs`만 자료가 뒷받침하는 설명 과제로 교체하거나 정확히
`SECTION_PLAN_UNREPAIRABLE`을 반환한다. Part/Section 삭제·병합·이동, 성공한 Section
변경, 사용자 요구사항 재배정은 허용하지 않는다. 교체 `target_refs`는 event 기록 전에
기존 미션 범위 참조 검증을 통과해야 하며, 실패하면 보정 결과를 기록하지 않는다. 원 canonical plan event는 바꾸지
않고 결과를 `report.plan.section_repair.completed`에 `applied` 또는 `unrepairable`로
기록한다. `applied`이면 성공한 Section artifact는 그대로 두고 교체 좌표만 새로운
attempt 1→2 예산으로 작성한다. `unrepairable`이거나 교체 뒤에도 attempt 2 gap이면
명시적으로 실패하며 같은 lineage에서 계획자를 다시 호출하지 않는다.

활성 장문 기본값은 같은 Part assembly MCP 인계를 사용한다. Section을 읽는 Part
assembler는 현재 Part에 바인딩된 immutable Section을 모두 bounded read한다. Intro,
transition, closing은 기본 산출물이 아니다. Section 관계를 이해하는 데 꼭 필요할
때만 최소한으로 추가한 뒤 `PART_ASSEMBLY_SUBMITTED` sentinel을 반환한다. 이
assembler는 post-assembly Part editor가 아니다.

Post-assembly Part editor는 `part_edit_enabled`가 true일 때만 실행된다. 이 editor는
source Part artifact 하나에만 바인딩되고 `plasma.report.part_edit.*` tool만 받는다.
source, research, Section, 다른 Part, final-edit tool은 읽을 수 없다. 제출된
outcome은 source Part event, source artifact, edited artifact, provider session,
profile metadata를 기록한다. No-op 제출도 내구 completion이다. source Part는
immutable input으로 보존되고, 중복 artifact를 만들지 않고 source artifact에 outcome을
바인딩한다. 활성 기본값의 Part editor는 인접 Section마다 앞 Section의 마지막 실질
문단, 연결문, 다음 Section의 첫 실질 문단을 함께 읽는다. 같은 기제나 결론, 읽기
지시가 반복되거나 단순한 관계가 불필요하게 추상화됐을 때만 한쪽을 최소 수정하며,
Section 내부의 일반 문체와 리듬은 다시 쓰지 않는다.

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
`final_edit_pipeline`만 기준으로 결정한다. 새 v3 계획은 deterministic final assembly,
report-plan provider session에서 fork한 final writer, 같은 report-plan session의
독립 sibling reader, optional `style_edit`, read-only `style_semantic_validation`,
read-only `evidence_gate` 순서로 실행된다. 저장된 v1/v2 계획은 legacy replay/recovery
compatibility path이며 기존 reader/style/corrective-gate 의미를 유지한다. 저장된 legacy
profile은 `plasma.report.long_form.finalize`를 계속 사용하며 staged final-edit 단계를
실행하지 않는다.

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

v3 `style_semantic_validation`은 read-only 비교와 verdict submit tool만 받는다.
Verdict는 `accepted_equivalent`와 `rejected_revert_to_reader`뿐이며, agent는 prose,
patch, final paragraph ordinal, manuscript Markdown, `repaired_by_gate`를 제출할 수
없다. 서버는 durable reader/style paragraph lineage에서 resolved Markdown을 만들고,
문단 수, 순서, delimiter, protected Markdown invariant를 증명할 수 없으면 닫힌다.

v3 `evidence_gate`는 approved read tool과 read-only evidence-gate tool만 받는다.
Evidence read surface는 deterministic report-owned Markdown block passage와 서버가
계산한 `statement_sha256`을 짝지어 제공하므로 provider가 hash를 계산하지 않는다.
Finding은 `statement_sha256`, `classification`, 승인된 `evidence_ids`만 포함할 수
있다. repair action, patch, replacement prose, manuscript Markdown, semantic
acceptance, operation count는 제출할 수 없다. Reporting layer는 bound
`SourceArtifactID`를 다시 읽고 lineage/SHA를 검증한 뒤 그 정확한 source content에 없는 hash를
거부하고, raw passage나 미승인 ref 없이 connection judgment를 저장하며, byte-identical source
content를 `operation_count=0`으로 canonicalize한다. Evidence judgment는 canonicalization을
막거나 자동 repair를 일으키지 않는다.

Runner는 매 evidence-gate 시도마다 하나의 `draft_id`와 하나의 bound tool session을
지정한다. Agent는 그 tool session을 `session_id`로 사용하고 offset 0에서 packet 읽기를
시작한 뒤, 같은 draft에서 반환된 `next_offset`만 따라 `truncated=false`까지 읽고 한 번만
submit한다. 다른 draft나 session, 잘못된 offset, 완료 전 submit은 계속 거부하지만,
오류 결과는 활성 draft, bound session, 다음 offset, packet 완료 상태와 다음 행동을
돌려주므로 새 draft를 만들지 않고 같은 검증을 이어갈 수 있다.

저장된 v1/v2 corrective gate event는 기존 의미대로 decode와 replay를 유지한다.

에이전트는 artifact ID, 파일명, 제목, 보고서 모드, 파트와 섹션 순서, 공급자
provenance, 모델 설정, binding identity를 선택할 수 없다. Stage artifact ID나
canonical event ID도 선택할 수 없다. Reader와 style 단계는 source/research를 읽을
수 없고 Section, source Part, edited Part, reader-source, prior-stage artifact를
mutate할 수 없다. Gate만 canonical finalization event를 만들 수 있다. Legacy
`plasma.report.long_form.finalize`는 저장 profile compatibility를 위한 closed
opening/closing input에만 묶여 남으며, 저장된 계획에 active pipeline 필드가 없을 때
사용된다. 이 값은 서버가 binding하며 commit 전에 mission ledger와 raw artifact에
다시 대조한다.

v3 staged pipeline에서 writer, reader, style, style semantic validation, evidence gate
submission은 durable intermediate event일 뿐이다. Evidence gate submission 뒤 같은
gate/finalization 경계에서 정확히 하나의 canonical event를 만들거나, 복구가 저장된
gate submission에서 이어 받아 완료된 provider를 다시 실행하지 않고 한 번만
canonicalize한다. Evidence gate는 alias나 repair artifact를 만들지 않고 이전 durable
artifact를 canonical로 채택한다. 같은 binding과 content SHA는 기존 결과를 replay한다.
식별자, provenance, 파트 순서, idempotency key, pipeline marker, stage lineage,
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

Section evidence-gap attempt는 현재 pending event, plan event, 1-based Part/Section
좌표에 scope된다. 복구가 attempt 1을 찾고 created Section이 없으면 같은 provider
session과 tool-session binding에서 attempt 2로 이어간다. Attempt 2를 찾고 created
Section이 없으면 provider 호출 없이 Section failure를 재구성한다. Gap 뒤에 created
Section이 있으면 복구는 해당 Section을 완료로 본다. 명시적인 새 report retry pending은
아직 계획 보정 결과가 없는 좌표에 fresh two-attempt Section budget을 받는다. 보정
완료 event가 있으면 `resume_failed`는 canonical plan과 같은 좌표의 amendment를 합쳐
유효 계획을 복원하고, 보정 전 gap만 교체 좌표의 새 예산에서 제외한다. `unrepairable`
결과도 복원하므로 재시작이나 retry가 계획 보정을 두 번째로 실행하지 않는다.

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
넣거나 대화 ledger event에 기록하지 않는다. 대신 canonical 제출을 확인하고 공급자
usage를 확보한 뒤에는 본문을 담지 않는 `report.agent_usage.recorded` event가 canonical
event ID와 session lineage, 표준 `agent_usage`만 기록한다. redacted 운영 로그에는 반환 세션의
존재 여부와 bound 세션 일치 여부, token 집계, duration만 남으며, 반환 세션 ID나
공급자 usage 상세를 canonical 상태에 기록하지 않는다.

공용 `mcp.tool.called` payload는 변경하지 않는다. tool name, success, created event
ID를 canonical 보고서 provenance와 결합해 경로를 검증할 수 있으며 opening,
closing, prompt, 전체 보고서 본문은 trace 요약에 기록하지 않는다.
