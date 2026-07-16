# Plasma Product Flow

이 문서는 Plasma 구현을 다시 시작하기 전에 고정하는 제품 흐름 문서다.
목적은 사용자가 실제로 어떤 순서로 Plasma를 쓰는지, 그리고 소스, 결과,
저장 지식, 리포트를 어떻게 구분할지 명확히 하는 것이다.

## C1 Default Loop

현재 기본 제품 루프는 C1이다. 사용자는 미션을 만들고 같은 에이전트 세션을
이어가며, 사용자 또는 controller가 다음 질문을 조정하고, 에이전트는
MCP/source read 도구로 원본 자료를 읽고 답변한다. 답변은 result로 저장되고,
리포트는 Plasma가 소유한 artifact로 저장된다.

공유 workflow run은 이 C1 루프 안에서만 동작한다. 사용자가 같은 미션에서
명시적으로 제한된 자율 진행을 시작하면, Plasma는 새 미션이나 별도 제품 모드를
만들지 않고 미션 장부에 요청, 시작, step, 종료 이벤트를 남긴다. 사용자는 실행
중 상태를 보고 멈출 수 있고, 완료나 중지 뒤에는 같은 에이전트 세션으로 일반
대화를 이어가거나 리포트를 요청한다.

기본 루프는 evidence, claim, confidence update, proposal bundle, AST-first report
machinery를 새로 만들지 않는다. 일반 대화와 bounded workflow run은 새 원자료를
찾았을 때 사용자 승인을 위한 source candidate를 남길 수 있다. 이때 후보는
에이전트가 명시적인 URL과 채택 의견을 함께 제시한 경우에만 검토 대상으로 남긴다.
source candidate는 소스가 아니며, 사용자가 승인하기 전에는 스냅샷을 만들지 않는다.
컨트롤러 전략 선택 이벤트 자체는 source candidate를 만들지 않는다. 기존 데이터와
코드는 과거 기록 조회나 명시적 개발자 실험 경계 뒤에 보존하지만, 사용자가 old/new
제품 모드를 선택하는 토글로 노출하지 않는다.

URL source candidate는 가능하면 제안 직후 원문 가져오기를 시도한다. 이때
`source.candidate.staging_started`가 먼저 남고, 성공하면
`source.candidate.staged`, 실패하면 `source.candidate.staging_failed`가 남는다.
staged 후보 artifact는 승인된 소스가 아니며, 기본 소스 목록과 리포트 입력에서
제외된다. 에이전트는 전용 candidate read 도구로만 이 본문을 읽을 수 있고, 그
응답에는 미승인 후보라는 경계가 함께 들어가야 한다.

2026-06-26 C0/PAL2/NAV 실험은 강한 상시 controller를 제품 기본값으로 채택할
근거를 만들지 못했다. NAV는 C0-like baseline보다 나빴고 PAL2는 결론이 약했다.
따라서 현재 제품 흐름은 같은 에이전트 세션을 이어가는 C0-like 기본 흐름을 유지한다.
controller는 별도 연구자나 작성자가 아니라, 정체, 반복, 지나친 협소화, 미션 이탈이
확인될 때만 약하게 steering하는 후보 기능으로 남긴다.

### 리포트 시도와 복구

장문 리포트 화면은 장부에 실제로 남은 계획, 섹션, 파트 조립, 최종 조립과 artifact
이벤트만으로 파이프라인을 보여준다. 상태는 대기, 진행 중, 완료, 실패, 건너뜀뿐이며
퍼센트나 완료 시간 추정은 표시하지 않는다. 실패한 시도는 닫힌 기록으로 남는다. 사용자는
새 pending 시도를 만들어 실패 지점부터 재시도하거나 처음부터 다시 생성할 수 있다.
전자는 같은 계보에서 검증된 완료 artifact와 호환되는 세션만 재사용하고, 후자는 조상
출력을 실행 입력으로 사용하지 않는다. 다른 미션이나 다른 계보의 artifact와 세션은
재사용하지 않는다.

## Product Purpose

Plasma는 사용자가 대화로 리서치 방향을 조정하고, 그 과정에서 나온 유용한
정보를 미션에 맞게 저장한 뒤, 필요할 때 근거 있는 리포트로 내보내는 제품이다.

Liquid2는 참고자료를 보관하는 개인 지식 저장소이고, Plasma는 그 자료와 외부
자료를 가져와 대화형 리서치를 진행하는 별도 제품이다. Plasma는 Liquid2 DB를
자기 저장소처럼 직접 소유하지 않는다. Liquid2는 Plasma가 조사할 수 있는 여러
소스 중 하나다.

브라우저 UI는 Plasma를 쓰는 하나의 클라이언트다. 제품의 중심은 브라우저 화면
자체가 아니라, 미션 장부와 그 장부를 읽고 쓰는 도구 계약이다. 따라서 Plasma는
짧은 MCP 안내와 에이전트가 제어하는 읽기/탐색/초안 작성 도구만으로도 UI 없이
사용할 수 있어야 한다.

### 미션 전환 표시 상태

미션 A에서 B로 실제 선택이 바뀌면 브라우저는 A의 request-local pending, 미리보기,
목록 action과 modal을 즉시 비우고 B의 loading 상태를 보여준다. B 상세 로드 실패는
재시도 action을 가진 안전한 disabled 상태로 남는다. 같은 미션의 refresh는 이 화면을
비우지 않고 최신 detail 응답만 적용한다. durable `active_work`는 장부와 workflow
projection에서 온 차단 사유와 가능한 action을 모두 표시하며, request-local pending은
응답 전의 즉시 비활성화에만 사용한다. 이전 선택에서 늦게 도착한 응답은 현재 미션의
notice, modal, 목록, busy 상태를 바꾸지 않는다.

## Core Terms

### Mission

미션은 사용자가 조사하려는 주제와 목표다. 미션에는 최초 목표, 현재 범위,
열린 질문, 연결된 소스, 저장된 지식, 생성된 리포트가 붙는다.

미션은 사용자가 대화 중 방향을 바꾸면 갱신될 수 있다. 이때 원본 이벤트는
장부에 남기고, 현재 미션 상태는 그 이벤트들을 반영한 최신 보기로 관리한다.

사용자는 현재 제목, 목표, 포함/제외 범위를 Web 편집기, `missions update`, `plasma.mission.update`로 명시적으로 고칠 수도 있다. 이 동작은 대화 조향과 구별되며 `mission.metadata.updated` 이벤트를 추가한다. 공급한 필드만 장부 순서대로 최신 값이 되고 이전 이벤트는 수정하지 않는다.

### Source

소스는 원본 자료다. 예시는 Liquid2 문서, URL, PDF, 로컬 파일, 외부 저장소,
웹 페이지, 이미지, 오디오/비디오 원천 또는 메타데이터 참조다.

소스는 사용자가 직접 추가하거나, 연결된 커넥터에서 찾은 원본 자료를 사용자가
선택해 붙인다. 에이전트가 만든 요약, 비교, 답변, 중간 결론은 소스가 아니다.

현재 구현의 소스 모드는 두 가지다. `snapshot_only`는 붙여넣은 텍스트, 브라우저나
CLI에서 업로드한 파일, URL fetch, PDF URL, Liquid2 문서처럼 원문 내용을 Plasma raw artifact에 저장하는 고정
스냅샷이다. `live_reference`는 서버에 설정된 allowlisted local path root 아래의
파일이나 디렉터리를 `root_id`와 `relative_path`로만 가리키는 라이브 참조다. 라이브
참조는 원문을 artifact로 복사하지 않고, 읽을 때마다 현재 상태를 관찰한다. 브라우저,
CLI, MCP, 에이전트 프롬프트는 임의 absolute path를 받거나 보여주지 않는다.

파일 업로드 소스는 `file_upload` connector type을 쓰는 `snapshot_only` source다.
업로드된 파일은 에이전트 결과가 아니라 사용자가 붙인 원본 자료이며, raw artifact와
source snapshot, `source.snapshotted` 이벤트로 저장된다. `file_upload`는
connector와 이벤트 provenance에만 남기고, locator의 `locator_type`은 실제 콘텐츠
구조인 `full_document`, `pdf_document`, `media` 중 하나를 쓴다. locator에는 원래
파일명, sanitized filename, MIME type, byte size, SHA-256, upload timestamp,
content kind를 남긴다. 같은 미션에서 같은 SHA-256 파일을 다시 업로드하면 raw
artifact를 새로 만들지 않고 기존 artifact를 가리키는 새 snapshot/event를 만든다.

PDF URL source는 `pdf_url` connector로 저장된다. Plasma는 원본 PDF bytes를
artifact로 고정하고, page count, byte size, SHA-256, 추출 텍스트 길이 같은
metadata를 locator에 남긴다. PDF를 읽을 때는 원본 PDF bytes를 프롬프트나 MCP
응답에 싣지 않고, bounded extracted text와 extraction metadata를 반환한다. 서버
local path root 안의 `.pdf` 파일도 live reference source로 붙일 수 있으며, 읽을
때 `source.observed` 이벤트와 함께 PDF text extraction 결과를 반환한다.

미디어 소스는 같은 source snapshot 경계를 따른다. 첫 구현 설계는 이미지 bytes를
고정 raw artifact로 저장하고, 오디오/비디오는 기본적으로 외부 원천의
metadata/live reference로 남기는 방향이다. 리포트에 들어가는 캡션, alt text,
self-contained HTML 렌더링은 source가 아니라 result 또는 artifact다.
업로드된 PNG/JPEG/GIF는 raw artifact로 보존하지만 Web, CLI, MCP read surface는 binary
bytes를 출력하지 않고 metadata-only 응답을 반환한다. 텍스트 계열 업로드는 bounded
UTF-8 chunk로 읽고, PDF 업로드는 원본 PDF bytes 대신 추출 텍스트 chunk와 extraction
metadata를 반환한다.

검색 결과나 에이전트가 언급한 URL은 아직 미션 소스가 아니다. C1 기본 루프에서는
그 자료가 왜 필요한지 result로 설명하고, 에이전트가 명시적인 source candidate와
채택 의견을 남긴 경우에만 검토 대상으로 보여준다. 사용자가 원본 자료를 source로
붙일 때 Plasma가 스냅샷을 만든다. 단순 URL 언급은 소스 후보가 아니다.

### Observation

관찰은 라이브 소스를 읽거나 grep하거나 directory tree로 확인한 특정 시점의 기록이다.
관찰은 source가 아니며, source의 현재 상태를 설명하는 ledger event다. 라이브 local
path 관찰은 `source.observed` 이벤트로 남고, `observation_event_id`, `observed_at`,
`root_id`, `relative_path`, size, mtime, sha256, read range, truncation, 가능한 git
metadata를 포함한다. 리포트나 답변이 mutable local path 자료에 의존하면 source ID만
말하지 않고 이 관찰 메타데이터를 함께 인용해야 한다.

### Evidence

근거는 소스 안의 특정 부분이다. 예시는 문서의 한 문단, URL의 특정 인용 구간,
PDF의 특정 페이지, 저장소의 특정 파일과 줄이다.

저장 지식은 가능한 한 근거를 참조해야 한다. 근거를 아직 만들 수 없으면
사용자 주장이나 임시 메모로 구분해서 남긴다.

근거는 확정 사실만 의미하지 않는다. 해석, 평가, 커뮤니티 반응, 루머,
논쟁 축, 시장 신호, 코드 예제, 수식, 벤치마크, 열린 질문도 미션에 유용하면
근거 후보가 될 수 있다. 단, 이런 신호는 사실처럼 승격하지 않고, 신호의 종류,
출처 품질, 신뢰도, 한계, 리포트 활용 가치를 함께 보여줘야 한다. 세부 기준은
[Evidence Signal Model](evidence-signal-model.md)에 고정한다.

### Result

결과는 에이전트가 만든 출력이다. 예시는 답변, 요약, 비교표, 중간 결론,
질문 목록, 초안이다.

결과는 다시 소스로 분류하지 않는다. 결과가 가치 있으면 저장 지식으로 승격하고,
그 결과가 어떤 소스와 근거를 바탕으로 만들어졌는지 연결한다.

### Saved Knowledge

저장 지식은 미션 장부에 의도적으로 남긴 주장, 메모, 결정, 질문, 정리 내용이다.
대화의 모든 턴이 자동으로 저장 지식이 되는 것은 아니다. Plasma는 저장 지식으로
남길 만한 내용을 제안할 수 있고, 사용자는 승인하거나 수정할 수 있다.

### Legacy Claim Confidence

주장 신뢰도는 legacy ledger loop에서 저장된 주장에 대한 현재 판단이다. 신뢰도는 `unknown`, `low`,
`medium`, `high` 같은 작은 단계로 표시하고, 수치 점수나 확률처럼 보이게 만들지
않는다.

이 흐름은 C1 기본 루프가 새로 만드는 제품 경로가 아니다. 기존 기록을 읽거나
명시적인 개발자 실험을 검증할 때만 이 이벤트를 다룬다.

주장의 최초 신뢰도는 주장 후보가 만들어질 때 붙는다. 이후 새 근거가 들어오거나
기존 근거의 의미가 바뀌면 `claim.confidence.updated` 장부 이벤트로 변경 이력을
남긴다. 이 이벤트는 주장을 승인하거나 기각하지 않는다. 사용자는 신뢰도 변화를
보고 판단할 수 있지만, 신뢰도 갱신 자체가 별도의 승인 허들이 되면 안 된다.

화면에서는 신뢰도 변화가 접힌 요약으로 보여야 한다. 기본 목록에는 주장, 현재
신뢰도, 상승/하락 방향, 짧은 사유만 보이고, 변경 이력, 열린 위험, 연결된 근거
ID는 상세 보기에서만 펼친다.

### Report

리포트는 미션 대화, 소스 참조, 저장된 미션 자료를 바탕으로 만든 산출물이다.
현재 기본 경로에서는 같은 에이전트 세션이 얇은 작성 지침과 MCP 읽기 도구를
사용해 Markdown을 만들고, Plasma는 그 Markdown을 report artifact로 저장한다.
채택된 기본 작성 지침은 F4 실험 결과를 따른다. 이전 대화, 조사 답변, controller
질문은 작업 기억으로만 쓰고 source로 인용하지 않는다. 에이전트는 내부적으로
원자료 기반 사실, 해석과 함의, 약한 신호, 충돌과 열린 질문, 독자에게 자연스러운
구조를 정리한 뒤 풍부한 Markdown 리포트를 작성한다. 실험명, 프롬프트명, tool
session id, 임시 경로 같은 내부 실행 정보는 리포트에 노출하지 않는다.

리포트 생성은 요청 수명에 묶인 일회성 작업이 아니다. 요청 시
`report.draft.pending` 이벤트를 먼저 남기고, 장문 리포트는
`report.plan.created`, `report.section.created`, `report.part.created`,
`report.artifact.created` 이벤트와 Markdown artifact를 단계별 진행 상태로 쓴다.
Web planned와 long-form은 계획 에이전트가 `plasma.report.plan.submit`으로
`report.plan.submitted` provenance를 먼저 남긴다. 이 제출의 `session_id`와 producer는 MCP 도구 세션이며
공급자 세션 provenance가 아니다. 실행기는 정확한 완료 sentinel과 반환된 실제 공급자
세션 계보를 검증한 뒤 현재 도구 세션의 제출 하나만 `report.plan.created`로 원자 승격하고,
검증된 공급자 세션은 이 정식 이벤트에 기록한다.
제출만 남은 시도는 진행 상태를 전진시키지 않는다.
서버가 재시작되거나 worker 메모리 상태를 잃어도 열린 pending을 발견하면 같은
pending event에 worker를 다시 붙이고, 이미 생성된 plan/section/part artifact를
재사용해 남은 단계부터 이어간다. 이미 완료된 섹션은 다시 쓰지 않는다.

보고서 pending 이벤트는 가시성을 위한 durable at-least-once 경계다. worker의 terminal
기록이 일시적으로 실패하면 pending은 열린 채 보수적으로 표시한다. 이 상태에서 새
generation worker를 시작하지 않는다. 별도 terminal-write-pending outbox는 후속 작업이며 현재 구현하지 않는다.
장문 리포트의 최종 조립은 C4 실험 결과를 따라 섹션 본문을 다시 쓰지 않는다.
대신 조립 경계에서만 중복 섹션 제목, 번호가 붙은 자기 제목, 프레임/전환 heading,
인접 반복 heading을 정규화한다. 코드 블록과 실제 섹션 소제목은 보존하며, 생성
이벤트에는 `assembly_strategy: c4_normalized_section_headings`를 남겨 어떤 조립
규칙이 적용되었는지 추적할 수 있게 한다.

기본 Markdown report artifact가 저장된 뒤에는 Web과 CLI report runner가 같은 H5
기반 한국어 말투 보정 pass를 후처리로 실행할 수 있다. 이 pass는 planner, source
selector, content model, AST, Designed HTML 경로가 아니며, 전체 Markdown을
프롬프트에 붙여 다시 쓰게 하지 않는다. 대신 report session에
`plasma.report.patch.*` MCP 도구만 열어 저장된 Markdown artifact를 bounded read로
읽고 작은 patch operation으로 보정한다. 원본 Markdown artifact를 덮어쓰지 않는다.
보정본이 구조와 충실도 guard를 통과하면 `report.artifact.exported` 이벤트로 원본
`source_artifact_id`를 가리키는 별도 Markdown artifact를 저장하고
`humanize_transport: mcp_patch`를 남긴다. 보정할 안전한 변경이 없으면 에이전트는
`NO_H5_CHANGES`를 반환하고, Plasma는 `report.humanize.skipped`로 닫아 중복 artifact를
만들지 않는다. 보정 실패, 컨텍스트 취소, MCP finalize 누락, guard 실패가 발생하면
`report.humanize.failed` 이벤트만 남기고 원본 Markdown artifact를 그대로 유지한다.
guard 실패 전에 patch artifact가 이미 finalize되었다면 Plasma는 `report.patch.rejected`
이벤트를 남기고, 그 rejected artifact를 기본 연구 raw artifact 목록과 읽기 표면에서
제외해 이후 에이전트 작업이 실패한 중간 산출물을 다시 소비하지 않게 한다.
MCP report-composition 도구는 provider 실행 주체가 아니므로 H5 pass를 직접 실행하지
않는다. 대신 finalize 시 `experiment.report.humanize.ready`를 기록해, 해당 Markdown
artifact가 같은 H5 pass의 대상임을 명시한다.

AST-first 리포트 버전, repair turn, report block은 legacy history와 명시적 실험
경계에 남긴다. 기본 리포트 요청은 이 객체들을 만들지 않는다.

Designed HTML은 기존 report material을 보기 좋게 렌더링한 추가 report artifact다.
소스가 아니고, legacy AST report도 아니다. 제품 경로는 2026-06-28 designed HTML
실험에서 가장 강했던 DH23 계열을 바탕으로 하되, 2026-07-05 visual-grammar
업데이트를 포함한다. 이 경로는 에이전트가 content model을 만들고, Plasma의
deterministic renderer가 모바일 안전한 self-contained HTML을 만든다. 제품 renderer는
content model의 가장 강한 visual unit을 첫 화면의 연결형 관계도로 승격해서,
사용자가 본문을 읽기 전에 보고서의 핵심 관계를 먼저 볼 수 있게 한다. 이후의 visual
unit은 timeline, evidence chain, dependency path, trade-off matrix, loop, relationship
map 중 정보 구조에 맞는 문법으로 나뉘어 렌더링된다. 이 다양성은 출처, 한계, URL,
긴 텍스트 가독성을 보존하는 범위 안에서만 적용한다. 미션에 승인된 image source가
있으면 Designed HTML content model은 원본 bytes가 아니라 `image_1` 같은 안전한
참조 ID만 사용해 관련 섹션 안에 이미지를 배치할 수 있다. 실제 image bytes와 출처
metadata는 기존 source snapshot/raw artifact 경계에 남고, HTML 안의 이미지 배치와
캡션은 report artifact의 표현 결과다.

## Primary User Flow

1. 사용자가 미션을 만든다.

   최초 입력에는 주제, 목표, 원하는 산출물이 들어간다. 예를 들면 "HTTPS DNS와
   DNS 레코드 타입을 알고 싶다"가 미션의 시작점이 된다.

2. 사용자가 소스를 붙인다.

   사용자는 Liquid2 문서, URL, 파일, 외부 저장소 같은 원본 자료를 미션에
   연결한다. 아직 소스가 없으면 빈 상태로 대화를 시작할 수 있지만, Plasma는
   사용자에게 허락된 소스 범위 안에서 조사 중 필요한 원본 자료 위치를 result로
   설명해야 한다.
   새 자료 탐색이 필요하지만 허락되지 않은 상태라면 먼저 사용자에게 물어본다.

   현재 브라우저 워크스페이스에서는 사용자가 텍스트를 붙여 소스 스냅샷을 만들
   수 있고, URL을 직접 추가하면 Plasma가 해당 HTTP/HTTPS 원문을 가져와 URL 소스
   스냅샷으로 저장한다. URL은 원본 자료인 소스이고, 에이전트가 그 URL을 언급한
   답변은 결과다.

   서버 운영자가 local source root를 허용 목록으로 설정한 경우, 사용자는 root ID와
   상대 경로만으로 로컬 파일이나 디렉터리를 live reference 소스로 붙일 수 있다.
   이 경로는 임의 파일 매니저가 아니라 미션 소스 선택 흐름이다. 같은 active local
   path source를 다시 붙이면 기존 source를 돌려주고, 같은 source가 soft removed
   상태이면 명시적인 restore 선택이 필요하다.

3. 사용자가 대화로 방향을 조정한다.

   사용자는 질문을 던지거나, 범위를 좁히거나, 다른 관점을 요구하거나, 중간
   리포트를 요청한다. 이 대화형 연구 흐름은 브라우저 화면, MCP 클라이언트,
   다른 에이전트 실행면에서 모두 성립해야 한다. 브라우저 UI는 한 클라이언트이고,
   브라우저 터미널은 기본 사용 흐름이 아니다.

4. 백엔드가 같은 에이전트 세션을 이어서 한 턴을 실행한다.

   각 사용자 입력마다 백엔드는 해당 미션에 연결된 에이전트 세션을 재개한다.
   이 세션은 `codex --resume` 같은 방식으로 이어갈 수 있는 에이전트의 실제
   대화 세션 식별자를 저장해야 한다.

   현재 Codex 실행 경로는 직전 에이전트 응답에 남은 세션 ID가 있으면 다음 턴에
   그 ID를 resume 값으로 전달한다. 이전 대화 맥락은 에이전트 제공자 세션이
   담당한다. Plasma가 매 턴 전달하는 것은 짧은 미션 상기와 최신 사용자 입력이며,
   이전 턴 원문, 이전 에이전트 결과 전체, 소스 본문 발췌를 다시 넣지 않는다.

   턴을 시작할 때 백엔드는 에이전트에게 다음 정보만 직접 준다.

   - 현재 미션 목표와 범위
   - 이번 턴에서 지켜야 할 제약
   - 필요한 경우 저장된 소스를 도구로 조회하라는 지침

   연결된 소스의 원문은 Plasma 저장소에 남아 있어야 한다. 에이전트가 소스 내용을
   확인해야 할 때는 사용 가능한 도구나 커넥터로 조회해야 하며, Plasma가 매 턴
   소스 본문을 프롬프트에 복사해서 전달하지 않는다.

5. 에이전트가 답변하고 필요한 후속 기록을 제안한다.

   에이전트는 사용자에게 답을 주고, 필요하면 저장 지식으로 승격할 만한 결과,
   열린 질문, 결정, 원본 자료 추가 필요성을 제안한다. 답변 자체는 result이며,
   그 안의 URL이나 요약은 자동으로 source나 evidence가 되지 않는다.

6. Plasma가 미션 장부에 저장한다.

   저장은 이벤트로 남기고, 화면에는 최신 상태를 보여준다. 사용자가 승인해야 할
   항목은 승인 대기 상태로 둔다. 승인 없이 확정 상태가 되는 것은 작업 진행
   기록처럼 되돌리기 쉬운 기록으로 제한한다.

   잘못 붙인 source는 기본적으로 soft remove/archive 한다. 제거는 source를 active
   목록, 기본 read, reporting, workflow 사용에서 숨기지만, source row, raw artifact,
   관찰 이벤트, audit history를 물리적으로 삭제하지 않는다. 필요한 경우
   `include_removed`나 UI audit toggle로 확인하고, 같은 source identity를 restore해
   다시 active 상태로 만들 수 있다. 물리 purge/redaction은 일반 Web/MCP/CLI 기능이
   아니라 별도 admin 결정이 필요한 후속 범위다.

7. 사용자가 제한된 자율 진행을 시작한다.

   사용자가 브라우저의 자율 진행 시작 제어 또는 `plasma workflow start` CLI로
   명시적으로 시작하면 Plasma는 `workflow.run.requested`를 미션 장부에 남긴다. 기본
   연구 에이전트와의 일반 채팅은 새 workflow를 시작하지 않는다. `max_steps`는 기본값
   적용 뒤 1..20만 허용하고, `max_duration_ms`는 0..86400000만 허용한다. 0은 전체 실행
   시간 제한이 없음을 뜻하며, 양수만 기존의 전체 실행 시간 예산을 적용한다. 각
   step의 agent 실행은 최초 호출, 자동 압축, 재시도가 공유하는 하나의 25분 마감
   시간을 가진다. 각 step은 일반
   대화 턴과 같은 provider session을 재개하고, `workflow_steering` 사용자 턴,
   에이전트 응답, step 완료 이벤트를 같은 장부에 남긴다.

   Web, CLI, MCP는 같은 workflow run을 보는 세 가지 제어면이다. Web은 시작/상태/
   중지 UI를 제공하고, CLI는 `plasma workflow start/status/stop`으로 같은 장부를
   조작한다. 첫 구현에서 CLI의 provider 실행 명령은 별도 background worker가 없기
   때문에 `--wait`으로 같은 프로세스 안에서 실행해야 한다. MCP는 mission-bound
   `plasma.workflow.start/status/stop` 도구로 요청과 중지만 기록한다. Plasma가 기본으로
   spawn한 연구 에이전트에는 status와 stop만 노출하고 start는 노출하지 않는다. 사용자가
   명시적으로 구성한 MCP 클라이언트는 `-enabled-tool plasma.workflow.start`로 start를
   계속 노출할 수 있다. MCP 도구
   호출 안에서 provider를 즉시 재귀 실행하지 않으며, host가 현재 사용자 턴이
   끝난 뒤 이어 실행할 수 있도록 현재 `turn.user` 이벤트와 현재 `agent_executor`
   binding에 묶어 기록한다. MCP 요청이 현재 executor와 다른 executor를 지정하면
   queued run을 만들지 않고 거절한다.

8. workflow run이 중지되거나 완료된다.

   사용자가 중지하면 `workflow.run.stop_requested`가 남고 runner는 다음 step 전에
   이를 관찰해 멈춘다. 아직 runner가 시작하지 않은 queued run은 stop 요청 시 바로
   `workflow.run.stopped`로 닫힌다. 사용자 중지 없이도 최대 step 수, 설정된 전체 실행 시간,
   provider 실패, 같은 세션 검증 실패, 에이전트가 더 할 일이 없다고 선언한 경우
   종료된다. 실행 중이던 프로세스가 사라지면 projection은 상태를 `interrupted`로
   보여준다.
   첫 구현은 새 queue/lease table 없이 장부 이벤트와 in-process runner만 사용하므로,
   사용자는 보이는 상태를 보고 다시 시작하거나 중지를 요청해 회복한다.

   workflow 실행 중 active source가 제거되면 진행 중인 안전한 read를 억지로 중단하지
   않는다. 대신 runner는 다음 step을 시작하기 전에 active source projection을 다시
   보고, 시작 이후 제거된 source에 대해 `workflow.source.skipped` 이벤트를 남긴다.
   이후 step은 제거된 source를 기본 read/reporting 대상으로 쓰지 않고 나머지 active
   source나 사용 가능한 조사 경로로 계속 진행한다.

9. 사용자가 같은 미션에서 대화를 재개한다.

   workflow run이 terminal 상태가 되면 사용자는 같은 미션과 같은 provider session에서
   다시 일반 질문을 보낼 수 있다. queued, running, stopping 상태의 workflow나
   pending report draft가 있는 동안 일반 turn이나 중복 workflow start는 충돌로
   거절되며, 사용자는 먼저 중지하거나 완료를 기다린다.

   이 충돌 검사는 브라우저 프로세스 안의 잠금에만 의존하지 않는다. 일반 turn,
   report draft, agent session reset, workflow start는 기록 직전에 공유 service가
   같은 SQLite 원장 트랜잭션 안에서 active work를 다시 확인하고, 통과한 경우에만
   pending/request 이벤트를 append한다. 따라서 Web과 CLI가 같은 DB를 쓰더라도
   같은 미션의 provider session을 동시에 시작하지 않도록 한 번 더 막는다.

10. 사용자가 리포트를 만든다.

   사용자는 이번 리포트에 적용할 방향 힌트를 선택적으로 입력할 수 있다. 힌트는 근거나 강제 범위가 아니라 약한 편집 축이며, 해당 요청의 계획과 본문 작성 프롬프트에만 명시적으로 넣는다. 요청이 접수되면 브라우저 입력값을 비우고, 접수에 실패하면 재시도를 위해 유지한다. 이후 대화, 자율 진행, 상태 회상, 말투 보정, 보고서 수정, HTML 내보내기, 다음 리포트 요청에는 힌트를 새로 넣거나 복사하지 않는다. 다만 같은 제공자 세션을 이어 쓰는 경우에는 앞선 보고서 프롬프트가 세션 기록에 남아 있을 수 있다.

   사용자는 active agent 작업이 없을 때 현재 미션 자료를 바탕으로 리포트를 요청할 수 있다. 리포트는
   대화의 원문 전체를 프롬프트에 다시 붙여 넣는 방식이 아니라, 에이전트가 미션 대화,
   소스 참조, 저장된 미션 자료를 필요한 만큼 읽어 만들어진다.
   리포트 작성은 큰 미션 리콜 JSON이나 사전 조립된 리포트 전용 자료 묶음을
   프롬프트에 넣는 방식에서 벗어나야 한다. 앞으로의 방향은 짧은 작성 지침과
   MCP 읽기 도구로 기존 장부의 소스, 근거, 저장 지식, 결과, artifact를 필요한 만큼
   조회하는 것이다. 원테이크를 제외한 리포트 생성은 가능하면 기존 조사 세션을 fork한
   보고서 전용 세션에서 Markdown artifact를 작성하고, 이후 일반 대화는 원래 조사 세션을 이어간다.
   fork 가능한 executor나 기존 조사 세션이 없으면 같은 세션으로 작성하되,
   `report_session_policy_selection`에 그 이유를 남긴다. 리포트 작성 지침은 얇은 요약을 요구하지 않고, 원자료가
   허용하는 범위에서 맥락, 비교, 결과, 긴장을 포함한 읽을 만한 글을 요구한다.
   다만 약한 신호와 추정은 명확히 표시한다. report draft가 pending인 동안에는 일반 turn과 workflow start를
   막아 같은 provider session을 병렬로 건드리지 않는다. completed workflow run이 자동으로
   리포트를 만들지는 않으며, 사용자나 에이전트가 명시적으로 요청해야 한다.
   workflow run이 단계 제한이나 설정된 시간 제한에 닿았지만 에이전트가 다음 조사가 필요하다고
   판단한 경우에는 completed가 아니라 paused로 남고, 다음 조사 지시를 보존한다.
   live local path source에 근거한 문장은 relative path 같은 사람이 읽을 수 있는
   locator와 `observation_event_id`, `observed_at`, sha256, git 상태 같은 관찰 메타데이터를
   함께 참조해야 한다. source ID만으로 mutable source 상태를 인용하지 않는다.

## Mission Recall

미션 리콜은 사용자를 위한 안내 문구가 아니라, 에이전트가 길을 잃지 않도록 매
턴 전달되는 짧은 상기 정보다.

미션 리콜에는 다음이 들어간다.

- 미션의 현재 목표
- 포함 범위와 제외 범위
- 열린 질문
- 이번 턴에서 지켜야 할 제약

미션 리콜에는 이전 대화 전문, 에이전트 결과 전문, 소스 본문, evidence 본문을
넣지 않는다. 소스와 근거는 Plasma 저장소에 남기고, 에이전트가 필요할 때 사용
가능한 도구나 커넥터로 조회한다.

짧은 MCP 안내, 시스템 프롬프트, 또는 안내용 도구는 에이전트가 어떤 순서로
장부를 조회해야 하는지 알려줄 수 있다. 그러나 그 안내면이 소스, 근거, 주장,
legacy 리포트 블록 데이터를 크게 들고 다니면 안 된다. 이름이 안내, 리콜, 팩, 캐시,
프리빌드 자료 중 무엇이든, 장부 데이터를 프롬프트에 다시 포장해 넣는 방식은
Plasma의 기본 흐름이 아니다.

## MCP Research IDE Surface

UI 없는 MCP 사용면은 짧은 안내와 다섯 가지 조회 도구로 구성한다.

- `plasma.research.outline`: 미션 목표, 범위, 열린 질문, 저장 지식, 리포트 진행
  상태를 한눈에 보는 전체 개요다.
- `plasma.research.list`: 소스, 근거, 저장 지식, 질문, 원장 이벤트, raw artifact,
  report artifact 같은 장부 항목을 조건별로 찾는 목록 조회다. legacy claim/report
  block 조회는 명시적인 legacy 경계 뒤에 둔다.
- `plasma.research.read`: 특정 소스, 근거, 저장 지식, report artifact, 원장 이벤트를
  읽는다. 긴 원문과 장문 payload는 범위를 지정해 읽을 수 있어야 하며, 이 부분
  읽기가 임의 위치 탐색의 기본이다. 에이전트 결과는 별도 소스가 아니라
  `ledger_event`로 조회된다.
- `plasma.research.grep`: 기존 장부와 연결된 소스에서 후보 문자열이나 패턴을
  찾는다. 검색 결과와 스니펫은 후보일 뿐이며, 리포트 주장은 저장 근거 또는
  명시적인 소스 읽기로 다시 고정해야 한다.
- `plasma.research.references`: 소스, 근거, 저장 지식, 결과, report artifact 사이의
  참조 그래프를 따라간다. legacy claim/report block 참조는 명시적인 legacy 조회다.

에이전트는 먼저 `plasma.research.outline`으로 전체 미션을 파악하고,
`plasma.research.list`와 `plasma.research.grep`으로 읽을 후보를 좁힌 뒤,
`plasma.research.read`로 필요한 원문을 확인하고,
`plasma.research.references`로 소스-근거-저장 지식-리포트 연결을 검증해야 한다.
이 조회면은 리포트 전용 중복 코퍼스나 미리 만든 리포트 팩을 만들기 위한 우회로가
아니다. 같은 장부와 같은 MCP 계약 위에서 브라우저 UI, 에이전트 제공자, 검색
백엔드, 리포트 렌더러를 서로 교체할 수 있어야 한다.

미션에 묶인 MCP 도구 호출은 `mcp.tool.called` 이벤트로 남긴다. 이 로그는
사용자에게 숨기는 내부 하네스가 아니라, 에이전트가 어떤 도구를 어떤 순서로
사용했고 어떤 호출이 실패했는지 확인하는 디버깅 표면이다. 다만 다른 세션 ID로
위장한 mutating 호출처럼 보안 경계에서 거절된 호출은 저장소에 도달하지 않게
한다.

## Agent Session Model

Plasma가 관리해야 하는 세션은 브라우저 터미널 세션이 아니라 에이전트 대화
세션이다. 미션은 하나 이상의 에이전트 세션을 가질 수 있고, 현재 브라우저
작업공간은 에이전트 종류별 기본 세션을 관리한다.

세션 기록에는 최소한 다음 정보가 필요하다.

- 미션 ID
- 에이전트 종류
- 에이전트가 resume할 수 있는 실제 세션 ID
- 마지막 실행 시각
- 현재 상태

브라우저가 닫혀도 에이전트 세션 기록은 사라지면 안 된다. 다음 사용자 입력은
같은 에이전트 세션을 재개해야 한다. 터미널이 필요하다면 나중에 운영자용 콘솔로
추가할 수 있지만, 기본 제품 흐름은 대화 입력과 에이전트 턴 실행이다.

기존 provider 세션이 컨텍스트 한계 등으로 더 이상 재개되지 않으면 Plasma는 새
세션으로 자동 전환하지 않는다. 사용자가 명시적으로 새 세션 시작을 요청했을 때만
`agent.session.reset` 이벤트를 남기고, 다음 턴부터 같은 미션의 새 provider 세션을
시작한다. 이 동작은 Plasma 미션 장부, 소스, 근거, 저장 지식을 삭제하지 않는다.

## Source Handling

소스는 Plasma 리서치의 시작점이다. 소스 추가 UI는 미션 생성 이후 바로 접근할
수 있어야 한다.

초기 소스 종류는 다음을 우선한다.

- Liquid2 문서 선택
- URL 추가
- 파일 추가
- 텍스트 스니펫 추가

Liquid2 연결은 읽기 전용 커넥터로 시작한다. 사용자가 Liquid2에서 자료를 직접
선택할 수 있어야 하고, 에이전트도 필요하다고 판단하면 Liquid2 자료를 검색하고
참조할 수 있어야 한다. Liquid2는 중요한 소스 경로지만 유일한 경로가 아니다.

에이전트가 조사할 수 있는 범위는 사용자의 지시로 정한다.

- 이미 미션에 붙은 소스 안에서 조사하라고 한 경우, 에이전트는 그 소스 안에서
  필요한 부분을 읽고 result로 답한다.
- 사용자가 처음부터 조사를 요청했거나 에이전트가 새 자료가 필요하다고 판단한
  경우, 에이전트는 붙어 있는 소스, Liquid2, 웹 검색 등 사용 가능한 경로를
  순서대로 시도하고 자료를 찾는다. 별도의 검색 사전 승인은 요구하지 않는다.
  찾은 자료는 source candidate로 제안할 수 있지만 자동 저장되지는 않으며, 사용자가
  승인할 때만 source로 붙는다.
- 특정 경로가 실패하면, 예를 들어 Liquid2 커넥터가 응답하지 않으면, 에이전트는
  그 실패를 보고하고 가능한 다른 경로로 조사를 계속한다.
- 사용자가 새 자료 탐색을 명시적으로 제한했는데 에이전트가 새 자료가 필요하다고
  판단하면, 먼저 사용자에게 조사 범위 변경을 요청한다.

에이전트가 찾아온 자료는 곧바로 미션 소스가 되지 않는다. C1 기본 루프에서는
에이전트가 자료의 위치와 쓰임을 result에 설명하고, 일반 대화와 bounded workflow
run 모두 명시적인 source candidate와 채택 의견을 검토 영역에 남길 수 있다. 사용자는
그 원본 자료를 붙일지 결정한다. 단순 URL 언급이나 "원문을 확인하라"는 일반 문구는
소스도 소스 후보도 아니다. source candidate 승인/거절 장부는 이 검토 결정을 남기는
표면이며, 승인 전 후보를 소스로 취급하지 않는다.

승인 전 후보를 읽을 수 있게 하는 것은 에이전트의 조사 보조 기능이다. 이는 후보가
중복인지, 실제로 쓸모 있는 원문인지, 사용자가 승인할 가치가 있는지 먼저 점검하기
위한 경로이며, 후보 artifact를 정식 source snapshot으로 격상하는 동작은 아니다.
사용자가 승인하면 staged artifact를 재사용할 수 있지만, 그때도 별도의
`source.snapshotted` 이벤트와 source snapshot 관계가 만들어져야 한다.

현재 브라우저 워크스페이스의 URL 스냅샷은 absolute `http` 또는 `https` URL만
받는다. URL fragment는 정규화 과정에서 제거하고, 응답은 전체 60초 타임아웃과
45초 응답 헤더 타임아웃, 최대 5회 리다이렉트, 64 KiB 응답 헤더 제한, 20 MiB 본문
제한 안에서 가져온다. 프록시 설정은 사용하지 않는다. DNS 해석 뒤 loopback,
private, link-local, multicast, unspecified, `100.64.0.0/10` CGNAT
주소로 가는 URL은 거절하고, 리다이렉트된 요청도 같은 주소 검사를 통과해야 한다.
HTML, plain text, JSON, XML 같은 텍스트성 응답만 소스 스냅샷으로 저장한다.

같은 미션에 같은 정규화 URL을 다시 추가하면 Plasma는 URL을 다시 가져오지 않고
기존 URL 소스 스냅샷을 반환한다. 이 동작은 중복 소스를 만들지 않기 위한 것이며,
기존 소스의 원문 내용을 새로 갱신하는 기능은 아니다.

에이전트 답변은 브라우저에서 Markdown으로 보여줄 수 있지만, 그것은 표시 방식일
뿐이다. 답변은 결과로 남고, Markdown 안의 링크가 자동으로 소스가 되지는 않는다.
링크는 사용자가 직접 소스로 붙일 수 있는 원본 자료 위치로 남는다.

## Automatic Investigation

자동조사는 현재 구현에서는 bounded workflow run이다. 사용자가 같은 미션에서
명시적으로 시작하고, Plasma가 제한된 step과 시간 안에서 같은 provider session을
이어 한 번씩 진행한다. 각 step의 에이전트 답변은 result이며, workflow 요약도
source가 아니다.

자동조사는 별도 미션이 아니고 영구 자동 모드도 아니다. 사용자가 대화로 연구하다가
workflow run을 시작하고, 중지되거나 완료되면 같은 미션에서 다시 대화를 이어간다.
실행 상태는 브라우저 UI, CLI, mission-bound MCP 도구가 모두 같은 projection으로
본다.

초기 권한은 다음으로 제한한다.

- 다음에 할 작은 조사 step을 정한다.
- 연결된 소스와 장부를 read-first MCP 도구로 확인한다.
- 필요한 외부 자료 경로가 있으면 result로 설명한다.
- 사용자가 검토할 수 있는 중간 결론이나 다음 질문을 남긴다.
- 더 진행할지, 멈출지, 다음 지시가 필요한지 작은 control decision을 남긴다.

자동조사는 새 원본 자료를 실제 미션 소스로 격상하거나, 주장을 확정하거나, 리포트를
자동 확정하지 않는다. 그런 변경은 사용자 승인이나 명시적 요청 후에만 일어난다.
사용자 승인 없이 외부에 게시하거나, Liquid2 원본 자료를 변경하지 않는다.

agent/MCP turn 안에서 workflow start가 요청되면, Plasma는 요청 이벤트만 남기고
현재 provider turn이 끝난 뒤 실행한다. 이 deferred-start 규칙은 같은 provider
session을 재귀적으로 재개하지 않기 위한 것이다. report draft는 이번 slice에서
deferred MCP 작업이 아니며, active turn, workflow run, report draft가 없을 때만
시작된다.

자동조사의 시작 조건, 새 소스 탐색 권한, bounded loop, 중지와 회복 동작, 후속
대규모 병렬 조사 확장 방향은 [Automatic Investigation](automatic-investigation.md)에
정리한다.

## Explicit Non-Goals For The Recovery Branch

이번 복구 흐름에서는 다음을 하지 않는다.

- 브라우저 터미널을 Plasma의 기본 화면으로 만들지 않는다.
- 터미널 attach/reattach를 미션 대화의 핵심 모델로 삼지 않는다.
- 에이전트 결과를 소스로 재분류하지 않는다.
- 브라우저 UI를 Plasma 제품 자체와 동일시하지 않는다.
- 소스 후보를 사용자 승인 없이 미션 소스로 격상하지 않는다.
- staged source candidate artifact를 일반 raw artifact나 기본 리포트 입력으로
  노출하지 않는다.
- 거절된 소스 후보를 같은 미션의 검토 목록에 계속 노출하지 않는다.
- 검색 결과나 스니펫을 저장 근거 또는 명시적 소스 읽기 없이 리포트 주장으로
  사용하지 않는다.
- 리포트 전용 중복 코퍼스, 사전 빌드 리포트 팩, 대형 프롬프트 주입을 다른 이름으로
  되살리지 않는다.
- 하네스로 결과 품질을 강하게 통제하려 하지 않는다.
- Liquid2 DB를 Plasma의 내부 저장소처럼 직접 결합하지 않는다.

## Implementation Direction

다음 구현 방향은 같은 장부 위에 UI와 MCP 사용면을 함께 올리는 것이다.

1. 미션과 소스를 만들고 볼 수 있는 최소 브라우저 UI를 유지한다.
2. MCP 사용면에서 `plasma.research.outline`, `plasma.research.list`,
   `plasma.research.read`, `plasma.research.grep`,
   `plasma.research.references` 조회 모델을 안정적인 계약으로 유지한다.
3. 사용자 입력을 받아 같은 에이전트 세션으로 한 턴을 실행한다.
4. 턴마다 짧은 미션 상기와 도구 사용 지침만 전달한다. 소스 본문, 근거 본문,
   대형 리콜 JSON은 직접 주입하지 않는다.
5. 에이전트 응답과 저장 지식 제안을 분리해서 보여준다.
6. 사용자가 승인하거나 명시한 내용만 저장 지식으로 장부에 남긴다.
7. 제한된 workflow run을 Web, CLI, MCP에서 같은 장부 projection으로 시작, 조회,
   중지할 수 있게 한다.
8. workflow가 끝나면 같은 미션과 provider session에서 일반 대화를 이어가며, 리포트
   요청은 가능하면 fork된 보고서 전용 세션에서 처리한다.
9. 저장 지식과 근거, 필요한 명시적 소스 읽기를 바탕으로 Markdown 리포트 artifact를
   생성하고, 보고서 작성 세션이 일반 조사 세션을 오염시키지 않게 한다.

이 방향이 고정된 뒤에만 과거 PR의 커밋을 선별해서 가져온다.

## 보고서 요청별 모델 선택

사용자는 리포트 탭에서 이번 요청의 모델과 추론 강도를 선택하거나 빈 선택으로 미션 세션/provider 설정을 상속한 뒤 보고서 또는 장문 보고서를 시작한다. 빈 모델은 최신 같은 executor 세션 모델, provider 기본 모델 순으로 정해지고, 명시 모델과 빈 추론 강도는 해당 모델 기본값을 뜻한다. 서버는 조합을 pending 기록 전에 검증하고 유효 모델·추론 강도·선택 출처를 한 번 동결한다. stale 복구는 이후 세션 설정이 달라져도 pending 값을 그대로 사용한다.
