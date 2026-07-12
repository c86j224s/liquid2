# Plasma 제품 아키텍처

이 문서는 Plasma의 제품 경계와 백엔드 경계를 설명합니다. 기준 문서는
[product-architecture.md](product-architecture.md)입니다. 이 파일은 같은 내용을 한국어로 읽기 쉽게
풀어 쓴 동기화 문서입니다.

세부 구현 규칙은 영어 기준 문서와 이 한국어 문서에 함께 유지됩니다. 구현을 판단할 때는 두 문서를 함께
확인하세요.

## 제품 정체성

Plasma는 사용자가 대화로 조사를 조향하고, 그 조사에서 근거 있는 research report를 만들 수 있게 하는
독립 제품입니다. 현재 C1 기본 루프는 mission, 같은 agent session, user/controller steering,
MCP/source read tools, conversation result, report artifact를 중심으로 합니다.

Historical evidence, claim, confidence update, proposal, AST-first report는 legacy ledger machinery입니다.
Plasma는 migration과 experiment work를 위해 table과 read path를 보존합니다. 하지만 이것을 기본 제품
루프로 노출하지 않고, 사용자에게 old/new mode toggle로 보여주지도 않습니다. Source candidate review
record는 bounded workflow run 안에서 user approval prompt로 허용됩니다. 다만 candidate 자체는 source가
아니며, 사용자가 승인하기 전에는 snapshot을 만들지 않습니다.

Browser UI는 Plasma 위에 놓인 client 중 하나일 뿐, 제품의 중심이 아닙니다. Plasma는 MCP를 통해 UI 없는
research IDE처럼도 동작해야 합니다. Agent에게는 짧은 guidance만 제공하고, mission overview, search,
random-seek reading, reference traversal, report drafting은 기존 ledger 위의 도구로 수행해야 합니다.

외부 autonomous research 제품과 관련 논문 조사도 이 방향을 뒷받침합니다. 현대 deep-research system은
거대한 단일 prompt보다 planning, retrieval, tool use, source checking, cited synthesis를 중시합니다.
그렇다고 Plasma가 강한 always-on controller를 기본값으로 둬야 한다는 뜻은 아닙니다. 2026-06-26
C0/PAL2/NAV 실험은 NAV를 기본값에서 제외했고, PAL2도 충분한 결론을 주지 못했습니다. 따라서 controller는
telemetry에 근거한 약한 conditional behavior로 유지합니다.

## 구현 계층의 모양

현재 Go package 구조는 계층을 나누는 방향으로 정리되어 있습니다. 다만 `internal/app`은 아직 잘게
분리된 application service layer라기보다, 여러 use case를 조율하는 넓은 service facade에 가깝습니다.

- `cmd/plasma`, `internal/web`, `internal/mcp`는 외부 entrypoint입니다. CLI, HTTP, MCP 요청을 해석하고,
  product result를 각 transport에 맞게 돌려줍니다.
- `internal/app`은 storage 호출, domain package 호출, provider 실행, Web/CLI/MCP가 공유하는
  compatibility contract를 조율합니다. 나중에는 source, report, workflow, connector, provider service로
  더 나뉠 수 있습니다.
- `workflowruns`, `workflowstate`, `sourceevents`, `sourcecandidates`, `sourceingest`, `reporting`,
  `ledgerstate` 같은 domain/feature package는 product rule, state transition, event payload shape를
  소유합니다.
- `storage/sqlite`는 ledger, raw artifact, source snapshot, projection을 저장합니다. `connectors/*`와
  `sources/*`는 교체 가능한 external access 또는 source-reading 구현입니다.

새 작업은 이 방향을 유지해야 합니다. Transport package는 request를 adapt하고, domain package는 제품
의미를 정의하고, app-level service는 use case를 조율합니다. Storage와 connector는 교체 가능한
implementation으로 남아야 합니다.

## 저장소 경계

Plasma는 자체 database와 domain model을 소유합니다. 다음은 금지합니다.

- Plasma mission state를 Liquid2 document table에 저장
- Liquid2를 직접 SQLite read
- 제품 dependency로 cross-database foreign key나 join 사용
- Liquid2 Go internals를 직접 import

Liquid2는 source connector 또는 external API provider로만 통합할 수 있습니다.

## Mission Ledger

### 명시적 미션 메타데이터 편집

현재 미션 메타데이터는 하나의 `UpdateMissionMetadata` 애플리케이션 서비스에서 편집한다. Web의 `PATCH /api/missions/{id}`, CLI의 `missions update`, 미션에 묶인 멱등 MCP 도구 `plasma.mission.update`는 모두 이 서비스를 호출하는 연결 계층이다. 사용자 편집이 성공하면 입력받은 `title`, `objective`, 전체 `scope`만 담은 `mission.metadata.updated` 이벤트를 하나 추가한다. 각 필드는 장부에서 가장 나중에 기록된 값이 현재 값이 되며, 입력하지 않은 필드는 그대로 유지한다. 빈 `objective`를 명시하면 목표를 지우고, 빈 `scope`를 명시하면 포함·제외 목록을 모두 지운다. 공백뿐인 `title`은 허용하지 않는다.

MCP 수정 도구는 사용자가 직접 제어하는 MCP client에는 제공하지만, Plasma가 내부에서 띄운 조사 에이전트의 기본 도구 목록에서는 제외한다. 에이전트가 사용자 편집을 가장하지 못하게 하여 이벤트의 사용자 소유 의미를 보존한다.

장부가 계속 원본이며 `plasma_missions`는 장부에서 다시 만들 수 있는 현재 상태 캐시다. 명시적 편집은 이전 이벤트를 고치지 않으며 대화 중 방향을 조정하는 `mission.steered`와도 구별된다. `mission.steered`가 기존에 사용하던 작성 주체와 충돌 판단 규칙은 바뀌지 않는다. 메타데이터 편집 이벤트가 없는 기존 장부도 그대로 읽을 수 있다.

Plasma에는 하나의 durable Mission Ledger가 있습니다. User-driven turn, bounded workflow run, MCP tool
call, report request는 모두 같은 ledger 위의 event producer입니다.

- User turn은 사용자의 direction, constraint, question, correction, approval decision을 기록합니다.
- Bounded workflow run은 requested, started, per-step, stop-requested, paused, completed, stopped,
  failed, interrupted event를 기록합니다. 각 workflow step은 별도 mission state를 갖지 않고,
  `workflow_steering` user turn과 agent result가 있는 일반 conversation path를 재사용합니다.
- MCP call은 mission-bound research와 workflow control operation을 bounded trace event로 기록합니다.
- Report request는 pending, artifact-created, failed event를 기록하고 기본 보고서를 Markdown artifact로
  저장합니다.
- Long-running report work는 browser, CLI, export surface가 공유하는 report runner boundary를 통과합니다.
  Runner는 pending/failure event, mode default, in-flight ownership을 소유합니다. 각 surface는 report
  policy를 직접 갖지 않고 executor와 request만 넘깁니다. 재시작 뒤에도 같은 pending event에 runner를
  다시 붙여 pending report draft나 designed HTML export를 이어갈 수 있어야 합니다. 장문 보고서는 기존
  plan, section, part artifact를 재사용한 뒤 계속 진행합니다. 현재 in-flight ownership registry는
  process-local이고, database 하나당 report runner process 하나를 전제로 합니다. 여러 서버 인스턴스가
  같은 Plasma database를 공유하려면 ledger-backed report-run lease가 먼저 필요합니다.
- Source lifecycle과 observation event도 ledger-backed입니다. `source.removed`, `source.restored`는 source
  row나 raw artifact를 삭제하지 않고 projection만 바꿉니다. `source.observed`는 mutable live source를
  read/tree/grep할 때 bounded metadata를 남깁니다.

어떤 producer도 별도의 source of truth를 소유하지 않습니다. Workflow status는 ledger event에서 project한
결과이지, durable mode flag나 별도 workflow table이 아닙니다.

Ledger는 교체 가능한 client와 adapter가 공유하는 기반이기도 합니다. Browser UI, agent provider, search
backend, report renderer는 각자 별도 state를 소유하는 컴포넌트가 아니라, 같은 ledger와 MCP contract 위에서
교체 가능한 컴포넌트여야 합니다.

## Agent Provider Boundary

Agent provider는 같은 mission ledger와 MCP surface 위에 놓인 교체 가능한 adapter입니다. 현재 한 미션의
첫 provider-backed action은 그 미션을 Codex 또는 Claude 같은 provider type 하나로 lock합니다. 이후 같은
미션의 요청은 lock된 provider를 사용해야 하며, 다른 provider로 호출하려 하면 provider 실행 전에
실패해야 합니다.

이렇게 해야 provider session identity, resume behavior, report forking을 이해하기 쉽습니다. 동시에 기존
`agent_executor` event payload를 유지하므로, 나중에 mixed-provider work를 설계할 여지도 남습니다.

Provider lock은 별도 schema field가 아니라 ledger event에서 파생됩니다. Source-only event, source
candidate, non-provider administrative event는 mission을 lock하지 않습니다. Browser, CLI, workflow,
report surface는 모두 같은 provider lookup과 lock validation을 거쳐야 하며, 보조 entrypoint를 통해 provider
switch가 우회되면 안 됩니다.

## Source Mode와 Local Path Connector

Connector와 source는 서로 다른 축입니다. Connector는 Liquid2, Confluence, 나중의 settings-managed local
filesystem root처럼 외부 원천에 접근하는 adapter입니다. Source는 URL, PDF, uploaded file, Liquid2 document,
Confluence page, local path file/directory처럼 Plasma 안에 accepted 또는 staged된 mission research
material입니다. Connector는 source material을 발견하거나 가져올 수 있지만, connector 자체가 source는
아닙니다.

Source registration은 보통 raw artifact를 만들거나 재사용합니다. 사용자가 승인하면 mission source snapshot을
만들고, 이 동작을 mission ledger에 기록합니다. Candidate staging은 approval 전에 raw artifact를 만들 수
있지만, 그 staged artifact는 사용자가 source snapshot으로 promote하기 전까지 candidate-only 상태입니다.

Plasma source snapshot은 Web, CLI, MCP, agent tool에서 같은 model을 공유합니다. 저장되는 retrieval policy는
다음과 같습니다.

- `snapshot_only`: 고정된 source policy입니다. Snapshot은 Plasma가 저장한 raw artifact 하나 이상을
  가리킵니다. Pasted text, browser/CLI file upload, fetched URL content, Liquid2 snapshot의 기본값입니다.
  File upload는 provenance를 위해 `file_upload` connector type을 씁니다. Locator의 `locator_type`은
  `full_document`, `pdf_document`, `media` 같은 content shape를 설명합니다. Locator에는 원본/정제 파일명,
  MIME type, byte size, SHA-256, upload time, content kind를 기록합니다. 같은 미션 안에서 같은 내용의 파일을
  다시 올리면 content SHA 기준으로 기존 raw artifact를 재사용하되, 새 source snapshot/event는 만듭니다.
- `live_reference`: 변할 수 있는 source policy입니다. 첫 구현에서는 `local_path`에 사용합니다. Source는 raw
  artifact body를 저장하지 않고, 빈 artifact list에 content hash가 있는 척하지 않기 위해
  `ContentHash{Algorithm:"none", Value:""}`를 사용합니다.

`local_path` connector는 `root_id`, `relative_path`, `path_kind` 형태의 locator만 저장합니다. 설정된 root의
absolute path는 서버 설정 안에만 있어야 합니다. Source snapshot, Web JSON, MCP response, CLI output, prompt,
report에는 absolute path가 나타나면 안 됩니다. 모든 local path access는 local path engine을 통해
canonicalize되어야 하고, absolute path, traversal, symlink, special file, deny pattern, cap을 검사해야 합니다.
외부로 돌려주는 DTO는 root ID와 relative path만 담아야 합니다.

Agent read는 source-scoped입니다. 사용자가 live local path file/directory를 source snapshot으로 승인한 뒤,
기본 MCP surface는 `snapshot_id`와 optional `subpath`로만 그 source를 읽습니다. 기본 agent surface는 accepted
source boundary 안에서 read, tree, grep을 수행할 수 있습니다. 하지만 root-wide `root_id` browsing이나 임의의
`root_id + relative_path` read를 노출하지 않습니다.

Live local path read, grep, directory tree는 `source.observed` event를 남깁니다. 여기에는 observed time,
root alias, relative path, optional subpath, file kind, size, mtime, bytes를 읽은 경우의 sha256, read range,
truncation/cap state, producer/session provenance, best-effort git metadata가 들어갑니다. 이 event는
observation record입니다. 새 source도 아니고 legacy evidence record도 아닙니다.

Source removal은 기본적으로 soft removal입니다. Removed source는 default list, read, research/reporting,
workflow use에서 숨기지만, `include_removed` 같은 명시적 audit option으로 볼 수 있습니다. 같은 local path
source를 다시 추가할 때는 명시적인 restore가 필요하며, 중복 active row를 만들지 않고 기존 source identity를
다시 활성화합니다. Physical purge나 redaction은 일반 Web/MCP/CLI 동작이 아니라 별도 admin 경계입니다.

Media와 document source도 같은 source snapshot boundary를 따릅니다. Image는 raw artifact로 고정되어
self-contained interactive HTML export에 embedded될 수 있습니다. Audio/video는 기본적으로 metadata 또는
allowlisted provider embed로 남깁니다. PDF URL source는 document snapshot입니다. Plasma는 원본 PDF bytes를
pin하고, page count, extraction support, ingest 시점의 `text_length_known=false` 같은 metadata를 저장하며,
source read tool은 raw PDF bytes 대신 bounded extracted text를 반환합니다. Generated caption, report
rendering, thumbnail, PDF extraction text, alt text는 result 또는 artifact이지 source가 아닙니다.

## MCP Research IDE Surface

MCP-first surface는 좁고 retrieval-oriented해야 합니다.

- `plasma.research.outline`: mission goal, scope, open question, result state, report artifact state를
  포함한 미션 전체 개요.
- `plasma.research.list`: source, evidence, saved knowledge, raw artifact, conversation result, ledger
  event, report artifact를 찾는 discovery 도구. Legacy claim/report-block object kind는 명시적 legacy
  boundary 뒤에 둡니다.
- `plasma.research.read`: 특정 source, evidence item, saved knowledge item, report artifact, raw artifact,
  ledger event를 range support와 함께 읽는 도구. Agent result는 ledger event로 읽으며 source로 재분류하지
  않습니다.
- `plasma.research.grep`: ledger content, pinned source snapshot, live local path source를 shared observation
  engine으로 검색하는 도구. External connector search는 원본 자료를 발견하는 별도 경로일 수 있습니다.
- `plasma.research.references`: source, evidence, saved knowledge, result, report artifact 사이의 graph
  traversal. Legacy claim/report-block reference는 명시적 legacy access 뒤에 둡니다.

Guide, prompt, helper tool은 이 workflow를 설명할 수 있지만 얇아야 합니다. Source body, evidence, saved
knowledge, report data를 거대한 prompt, report-only corpus, prebuilt report pack으로 복제하면 안 됩니다.

Search result와 snippet은 후보일 뿐입니다. 보고서의 문장은 명시적인 source read로 뒷받침되어야 합니다.
Optional evidence layer가 켜져 있다면 원본 source를 가리키는 saved evidence로 뒷받침되어야 합니다. Live
local path material에 의존하는 문장이라면 source ID만 적지 말고 사람이 이해할 수 있는 locator와 observation
metadata를 함께 남겨야 합니다.

Mission-bound MCP call은 관찰 가능한 product event입니다. Plasma는 `mcp.tool.called` ledger event에 tool
name, timing, success state, bounded argument summary, bounded result summary를 기록합니다. 이를 통해
browser와 UI-less client는 agent가 실제로 outline/list/grep/read/references를 사용했는지 확인할 수 있습니다.
이 확인을 위해 source body를 prompt에 복사하거나 별도 report-only corpus를 만들 필요는 없습니다.

## Product Surfaces

구현 slice는 같은 ledger와 MCP contract를 공유해야 합니다.

- research mission 생성
- Liquid2 connector와 media connector 등을 통해 source mount/snapshot
- conversation 또는 MCP client에서 steering directive 수락
- controller steering strategy selection을 observable ledger event로 기록하되, 어떤 controller strategy도
  검증된 기본 제품 controller처럼 취급하지 않기
- bounded workflow run 시작, 검토, 중지
- agent answer와 controller output을 source가 아니라 result로 유지
- large mission recall JSON injection이 아니라 thin guidance와 MCP/source read로 report draft
- planned report mode와 Part/Section long-form report mode를 같은 Markdown artifact model 위에 노출
- F4 report-writing guidance를 기본 Markdown report style로 사용. 이전 조사 결과를 working memory로 쓰고,
  fact, interpretation, weak signal, conflict, reader structure를 조용히 종합한 뒤, prompt, run, session,
  temporary-path 내부 정보를 노출하지 않는 풍부한 보고서를 작성
- 기본 보고서를 Markdown artifact로 저장
- policy가 허용할 때 pinned image를 embedded하여 self-contained interactive HTML로 export하고, audio/video는
  link 또는 allowlisted provider embed로 유지
- designed HTML export를 교체 가능한 deterministic renderer adapter로 노출

현재 designed HTML product slice는 2026-06-28 DH23-style content-model path와 2026-07-05 visual-grammar
update를 따릅니다. 선택된 agent가 Markdown report artifact에서 JSON content model을 만들고, Plasma는 이를
internal rendering artifact로 저장합니다. Renderer는 가장 강한 visual unit을 첫 화면의 compact connected
relationship map으로 승격합니다. 이후 visual unit은 deterministic timeline, evidence-chain, dependency-path,
trade-off matrix, loop, relationship-map renderer로 보냅니다. 결과는 self-contained HTML report artifact입니다.

이 경로가 아직 reference-grade parity에 도달했다는 뜻은 아닙니다. Renderer는 여전히 compact content model에
의존하며, decorative variety보다 source note, caveat, URL, 긴 텍스트 가독성을 우선해야 합니다.

## 현재 Browser Workspace Slice

현재 browser workspace는 Plasma-owned runtime 위의 local testing surface입니다. 할 수 있는 일은 다음과
같습니다.

- mission 생성
- user turn 기록
- 명시적으로 설정된 agent turn 실행
- pasted text, textual URL, PDF, file upload, local path, Liquid2 document, Confluence page 등을 source로 붙이기
- source candidate review
- bounded workflow run 시작/중지
- report-only provider fork session에서 non-one-take report request 실행
- 생성된 Markdown report를 raw artifact로 저장

Browser evidence/proposal/confidence와 AST report 기능은 기본 제품 루프가 아니라 legacy history/experiment
surface입니다.

기본 MCP research surface는 connector search, source candidate proposal, staged unapproved source candidate
chunk read, accepted source chunk read, accepted live local path directory tree/grep를 지원합니다. Staged
candidate read는 conversation/research aid일 뿐입니다. 해당 material이 승인되지 않은 후보임을 분명히
표시해야 하며, normal raw artifact list와 default report input에서는 제외합니다.

기본 surface는 candidate나 local path를 accepted source로 promote하는 source mutation tool을 노출하지 않습니다.
Agent에게 root-wide local path browsing도 노출하지 않습니다. Local path root browsing, attach, source remove,
source restore는 browser/CLI source command나 operator-enabled MCP server 같은 명시적 사용자/운영자 표면에
남깁니다.

같은 미션에 normalized URL이 이미 붙어 있으면 duplicate URL source post는 기존 source snapshot을 재사용합니다.

URL source fetch는 의도적으로 bounded되어 있습니다. Generic URL fetcher는 HTTP/HTTPS textual response만 받고,
proxy use를 끄며, 60초 overall timeout, 45초 response-header timeout, redirect 5회 제한, 64 KiB response-header
cap, 20 MiB body cap을 적용합니다. Loopback, private, link-local, multicast, unspecified, `100.64.0.0/10`
CGNAT 주소로 resolve되는 요청은 거부합니다. Redirect된 요청도 연결 전에 같은 DNS와 address policy를 거칩니다.
PDF URL source는 별도 `pdf_url` path를 사용하고 같은 network safety policy를 재사용합니다. PDF는 최대 100 MiB까지
pin하고, content가 PDF인지 검증하며, raw PDF bytes를 inline으로 반환하지 않고 bounded extracted text chunk를
read tool로 노출합니다.

Agent turn은 가능한 경우 latest agent response의 provider session id를 resume합니다. Plasma는 짧은 mission
reminder와 latest user turn만 보냅니다. 이전 turn history나 source body excerpt를 prompt에 다시 붙이지
않습니다. Source inspection은 tool/connector를 통해 이루어져야 합니다.

보고서 생성도 같은 원칙을 따릅니다. Report writer는 얇은 guidance만 받고, 필요한 정보는 ledger 위에서
MCP read로 찾아야 합니다.

선택 사항인 `direction_hint`는 미션 상태나 근거가 아니라 해당 보고서 요청의 대기 상태에만 속한다. 앞뒤 공백을 제거한 뒤 값이 남아 있을 때만 해당 `report.draft.pending` 이벤트에 저장하므로, 서버가 중단되었다가 다시 시작되어도 같은 요청을 복원할 수 있다. 이 필드가 없는 기존 이벤트는 빈 값으로 읽으며, 이후 보고서 요청으로 값을 복사하지 않는다. 고정 안내문은 힌트를 강제 조건이 아닌 약한 편집 축으로 다루게 한다. Plasma는 원테이크 작성, 계획형 보고서의 계획과 작성, 장문 보고서의 계획과 섹션 작성 프롬프트에만 힌트를 명시적으로 넣는다. 일반 대화와 재개 대화, 미션 알림, 상태 회상, 자율 진행, 파트·전체 조립, 말투 보정, 보고서 수정, 기본·디자인 HTML 내보내기에는 새로운 방향 블록을 넣지 않는다. 이 허용 목록은 애플리케이션이 새 프롬프트를 만드는 방식을 보장할 뿐 제공자 세션 기록을 지우지는 않는다. 같은 제공자 세션을 의도적으로 이어 쓰는 경로에서는 앞선 보고서 프롬프트가 세션 맥락에 남아 있을 수 있다.

`one_take`를 제외한 agent-backed report generation은 가능한 경우 현재 research
provider session을 fork하여 report-only session에서 실행합니다.

기본 보고서 경로는 G2 generation-time guidance를 사용합니다. H5 Korean tone pass는 기본적으로 꺼져 있습니다.
사용자가 humanized Markdown export를 명시적으로 요청한 경우에만, 보고서 생성이 끝난 뒤 실행되는 shared
Markdown transformation으로 동작합니다. H5는 original artifact를 대체하지 않습니다. 또한 planning, source
selection, AST shaping, content-model generation, Designed HTML rendering에도 참여하지 않습니다.

H5 단계는 report session을 resume하고 bounded `plasma.report.patch.*` MCP tool만 노출합니다. Agent는 저장된
Markdown artifact를 slice 단위로 읽고 targeted patch operation을 적용합니다. 전체 Markdown을 prompt에 붙이거나,
완전히 새로 쓴 본문을 반환하지 않습니다. 성공한 H5 결과는 원본 Markdown artifact를 가리키는 별도
`humanized_markdown` export artifact로 저장하고 `humanize_transport: mcp_patch`를 기록합니다.

Agent failure, context cancellation, MCP finalize 누락, fidelity guard failure가 발생하면 원본 Markdown만
유지합니다. 이미 patch artifact가 finalize된 뒤 fidelity guard가 실패하면 Plasma는 `report.patch.rejected`를
기록하고, 그 rejected artifact가 기본 research raw-artifact read/list에 들어가지 않게 숨깁니다. H5가
`NO_H5_CHANGES`를 보고하면 중복 artifact를 만들지 않고 no-change skip으로 기록합니다.

MCP report-composition tool은 nested provider turn을 실행하지 않습니다. 대신 Markdown artifact를 보존하고
H5-ready metadata를 기록합니다. Executor-owning surface는 나중에 같은 pass를 적용할 수 있지만, 아직
humanized artifact가 생긴 것처럼 꾸미면 안 됩니다.

보고서 패치는 기존 Markdown report artifact 위에서 실행되는 provider-backed work입니다. 전체 보고서를 prompt에
붙이거나 base artifact를 in-place mutate하면 안 됩니다. Patch run은 base artifact를 만든 report session을
resume하거나, executor가 지원하면 그 report session을 fork합니다. 이때 `plasma.report.patch.*` MCP tool을
임시로 노출하여 agent가 base Markdown slice를 읽고, exact replace/insert/append operation을 적용하고,
새 Markdown report artifact를 finalize하게 합니다.

Normal conversation turn에는 이 patch tool을 주지 않습니다. Patch artifact는 base artifact id, pending
request id, operation summary, provider session lineage, report-session policy selection을 기록합니다. 그래야
이후 UI/CLI/MCP surface가 이전 report를 source로 재분류하지 않고 version chain을 보여줄 수 있습니다.

Executor가 fork를 지원하지 않거나 미션에 pre-report research session이 없으면 같은 session으로 fallback하고
`report_session_policy_selection`을 기록합니다. 기본 browser path인 `보고서`는 planned Markdown report
artifact를 만듭니다. CLI `reports draft`도 같은 planned default를 사용합니다. `--mode one_take`는 명시적인
same-session compatibility path로 남깁니다.

느린 경로인 `장문 보고서`는 Part/Section plan을 만들고, section을 별도 Markdown artifact로 작성한 뒤, section
body를 보존하면서 part/final artifact를 조립합니다. Final assembly는 C4 experiment의 limited cleanup만 wrapper
boundary에서 적용합니다. Duplicate section heading, numbered self-heading, frame heading, connective heading,
adjacent heading repeat은 정리하되, fenced code와 실제 section body subheading은 보존합니다.

Long-form report event는 `assembly_strategy: c4_normalized_section_headings`를 기록해야 합니다. 그래야 나중에
어떤 assembly rule이 artifact를 만들었는지 디버깅할 수 있습니다. CLI `--mode long_form`은 CLI가 같은 section
runner를 호출할 수 있을 때까지 거부합니다. 단일 Markdown turn으로 흉내 내지 않습니다.

두 보고서 경로 모두 AST repair turn, report version, report block을 피합니다. 나중에 plan review 단계를 넣을 수
있지만, 보고서는 여전히 report artifact로 남아야 하며 source나 legacy AST report version이 되면 안 됩니다. 기본
guidance는 F4 experiment를 이어받습니다. 이전 대화, 조사 답변, controller question은 working memory이지 source가
아닙니다. Writer는 fact, interpretation, weak signal, conflict, reader-facing structure를 내부적으로 정리한 뒤
풍부한 Markdown 보고서를 써야 합니다.

Workflow run도 같은 session rule을 따릅니다. Run은 `workflow.run.requested`에서 시작하고, 최신 provider session을
한 bounded step씩 resume합니다. User-visible agent response를 result로 기록하고, 작은 workflow control marker는
저장 전에 제거한 뒤 terminal status를 mission ledger에 씁니다. Active agent/MCP turn 안에서 workflow start가
요청되면 Plasma는 요청을 기록하고, enclosing turn이 terminal event를 가진 뒤 provider execution을 drain합니다.

MCP workflow start는 현재 user event와 현재 agent executor에 묶여야 합니다. 다른 executor를 요구하는 요청은
queued run을 만들기 전에 거부합니다. In-process runner가 사라지면 projection은 run을 interrupted로 보고하여
사용자가 수동 DB 수정 없이 stop하거나 새 bounded run을 시작할 수 있게 합니다.

Workflow 중 active source가 soft-removed되면 다음 step은 source state를 새로 읽습니다. 그리고 해당 source와
removal event에 대해 `workflow.source.skipped`를 append한 뒤 계속 진행합니다. Runner는 removed source를
조용히 사용하지 않습니다.

CLI와 MCP는 같은 의미 위의 control surface입니다. CLI는 mission create/list/show, turn send, workflow
start/status/stop, Markdown report draft를 같은 SQLite ledger에 수행할 수 있습니다. 첫 slice에서 provider
execution이 필요한 CLI command는 `--wait`가 필요합니다. 별도 CLI background worker가 없기 때문입니다.
MCP workflow tool은 mission-bound이며 workflow event를 append/read할 뿐입니다. `plasma.workflow.start`는
MCP call 내부에서 provider를 실행하지 않고, current user turn과 bound executor에 묶여 host가 terminal
response 뒤에 drain할 수 있게 합니다.

Report drafting도 provider-backed work입니다. Conversation이나 workflow가 terminal state에 도달한 뒤 실행할 수
있지만, 같은 미션의 normal turn이나 workflow run과 겹치면 안 됩니다. Report가 provider session state를
fork하거나 resume하고 durable report artifact를 쓰기 때문입니다.

첫 slice는 이 no-overlap rule을 shared service boundary에서 강제합니다. Normal turn start, report draft start,
agent session reset, workflow run request는 새 pending/request event를 기록하는 conditional ledger append 안에서
active mission work를 다시 확인합니다. SQLite store는 이 conditional append를 immediate transaction locking이
있는 하나의 transaction으로 수행하므로, 서로 다른 Web/CLI process도 process-local lock만 믿지 않고 같은 최종
guard를 공유합니다.

Browser는 agent reply를 vendored `markdown-it`과 DOMPurify로 sanitized Markdown으로 렌더링합니다. 이것은 display
concern일 뿐입니다. Link나 agent text가 source가 되는 것은 아닙니다.

## 보류된 결정

다음 설계 wave에서 결정할 항목은 다음과 같습니다.

- Plasma runtime stack
- database engine과 migration tooling
- API shape와 service boundary
- Liquid2 connector contract
- report canvas와 renderer adapter model
- DH23 이후 designed HTML artifact productization과 visual grammar dispatch
- auth integration strategy with neutral subject identity fields
- unbound MCP mission create/open tools
- cross-process durable queue/lease table
- read-first research surface를 넘어서는 MCP report control tools

## 보고서 모델 선택 경계

Web과 CLI adapter는 원시 요청, 같은 executor의 최신 미션 세션 메타데이터, 설정된 provider 기본값을 수집합니다. reporting package는 우선순위와 capability 검증을 소유합니다. 시작에 성공하면 유효 모델, 추론 강도, `agent_selection_source`를 `report.draft.pending`에 기록하며 새 이벤트 복구는 이 동결값만 역직렬화합니다. 출처가 없는 legacy pending은 기존 resume 경로를 유지합니다. 내구 상태는 ledger payload가 담당하므로 DB migration은 필요하지 않습니다. MCP 보고서 도구나 모델 tier allowlist를 추가하지 않으며 prompt, report mode, session fork, H5, patch, designed HTML, experiment도 바꾸지 않습니다.
