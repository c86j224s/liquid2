# Confluence Cloud Source 연동 구현 기록

이 문서는 Atlassian Confluence Cloud 문서를 Plasma 미션 source로 등록하기
위한 제품 흐름, 저장 경계, 인증 방식, 업데이트 UX, CLI/Web/API/agent 사용
시나리오를 정리한다.

현재 0.0 제품 경로는 API token 방식만 사용한다. 이전 OAuth 3LO 설계와 일부
내부 helper는 기록과 향후 재검토 대상으로 남아 있지만, Web/CLI 사용자가 새
Confluence 연결을 만들 때는 Atlassian email, API token, site URL을 입력하는
경로가 기본이자 유일한 지원 경로다.

PR #23은 브라우저에서 Confluence page를 검색하고 snapshot으로 저장하는 첫
경로를 만들었지만, usable source intake 기준에는 부족했다. 이 follow-up은
연결 lifecycle, site -> space -> page/children 탐색, 후보 검토, 승인 시점
re-fetch, 큰 page range snapshot, update preview, CLI parity, typed Korean
errors, redaction checks, live validation checklist를 추가한다.

현재 0.0 구현은 전역 Settings의 Confluence API token 연결 등록, 미션별
agent/MCP 검색 허용 grant, spaces/pages/children browse, page 검색, 후보
preview, version-pinned full snapshot, precise plain-text range snapshot, update
check, update preview, new-version snapshot 추가, CLI, Web API, MCP discovery,
브라우저 정적 UI의 일반 사용자 경로를 지원한다.

## 의도한 사용자 흐름

사용자는 Settings에서 재사용 가능한 Confluence Cloud 연결을 등록한다. 미션의
Sources 화면은 이 연결과 Atlassian site를 선택한 뒤, `site -> space ->
page/children` 경로로 page를 탐색하거나 검색한다. Plasma는 browse/search 결과를
source 후보로 보여주고, 후보 검토 화면에서 제목, site, space, URL, page id,
version, updated time, body preview 또는 range option을 보여준다. 사용자가 특정
page version 또는 range를 승인하면 그 시점의 문서를 미션 source로 고정 저장한다.

미션의 agent/MCP Confluence 검색 권한은 별도의 사용자 선택이다. 기본값은 꺼짐이며,
권한을 켜도 source가 자동으로 붙지 않고, source를 붙여도 권한이 자동으로 켜지지
않는다. agent/MCP 검색 결과는 후보 또는 result일 뿐 source가 아니다.

승인 뒤에는 Plasma가 저장한 고정 snapshot이 source다. report와 evidence는 이
고정 page version을 참조한다. 에이전트가 만든 요약, 추천, 후보 목록은 result이며
source가 아니다.

현재 제품 흐름은 다음 순서를 따른다.

1. Settings에서 API token connection을 등록한다.
2. 미션 Sources에서 등록된 connection과 Atlassian site를 선택한다.
3. 선택된 site 안에서 space 목록, space page 목록, page children을 탐색하거나
   검색한다.
4. 제목, space, URL, page id, version metadata를 포함한 page 후보를 보여준다.
5. 사용자가 후보 preview를 연다. preview는 result/UI aid이며 source가 아니다.
6. Plasma가 자체 Confluence connector로 선택한 page를 다시 읽어 preview와 range
   option을 만든다.
7. 사용자가 full page 또는 plain-text range를 source로 승인한다.
8. 승인 시점에 Plasma가 page를 다시 fetch하고 `cloud_id`, `page_id`,
   expected version을 검증한다.
9. 후보를 본 뒤 page version이 바뀌었으면 중단하거나 재확인을 요구한다.
10. 승인된 page 또는 range를 `snapshot_only` source로 저장한다.
11. agent와 report는 Plasma source read 도구로 고정 snapshot을 읽는다.
12. 사용자가 별도 grant를 켠 미션에서만 MCP가 Confluence 후보 검색을 수행한다.

## 구현된 일

- Confluence Cloud source connector를 추가해 page 검색, 읽기, snapshot 생성을
  지원한다.
- Confluence page를 원본 자료 source로 취급한다.
- 승인된 page를 기존 `SourceSnapshot`과 `RawArtifact` 모델 위에 저장한다.
- 여러 Atlassian site를 지원하기 위해 모든 Confluence source identity에
  `cloud_id`를 포함한다.
- 0.0 기본 인증 방식은 API token/email connection으로 둔다.
- OAuth 3LO 경로는 0.0 사용자 표면에서 비활성화한다.
- snapshot payload에는 Confluence `storage` 본문과 추출한 plain text를 저장한다.
- `atlas_doc_format`은 block fidelity가 필요해진 뒤 추가하는 후속 옵션으로 둔다.
- MCP와 agent는 사용자가 미션 grant를 켠 범위에서만 Confluence 후보 검색을 할 수
  있고, 후보를 accepted source로 직접 승격하지 못한다.
- CLI와 Web JSON API에서 검색, snapshot 생성, 업데이트 확인, 승인된 업데이트
  생성을 지원한다.
- 브라우저와 CLI에서 deterministic browse, candidate preview, range snapshot,
  update preview를 지원한다.
- Connection display name 변경, local revoke, hard delete를 지원한다. 이미 승인된
  snapshot은 connection revoke/delete 뒤에도 읽을 수 있어야 한다.
- 미션 connector access grant는 mission ledger event에서 재생한 projection이
  source of truth다. 독립적인 durable grant table은 사용하지 않는다.
- Confluence HTTP 401/403/404/429, cloud mismatch, version drift, too-large page,
  revoked/expired connection은 safe machine code와 한국어 메시지로 응답한다.

## 하지 않을 일

- private Confluence page를 일반 URL source fetch로 가져오지 않는다.
- agent MCP fetch 결과, 요약, snippet을 source snapshot으로 취급하지 않는다.
- 첫 범위에서 Confluence를 `live_reference` source로 만들지 않는다.
- space, page tree, workspace 전체를 기본으로 bulk import하지 않는다.
- 첫 범위에서는 attachment, comment, database, whiteboard, child page tree를
  snapshot하지 않는다.
- source attachment와 agent/MCP 검색 grant를 한 동작으로 묶지 않는다.
- OAuth token, refresh token, API token, Authorization header, cookie, private
  body snippet을 source locator, ledger event payload, MCP trace summary, prompt,
  log에 저장하지 않는다.

## Settings split

Confluence 연동은 세 계층으로 나뉜다.

1. Settings connection registry는 재사용 가능한 Confluence connection을 등록하고
   관리한다. API token 등록, display name 변경, local revoke, hard delete,
   site cache 조회와 refresh는 전역 Settings route만 사용한다.
2. Mission connector access grant는 특정 미션에서 agent/MCP가 Confluence 후보
   검색을 할 수 있는지 결정한다. 기본값은 꺼짐이며, `mission.connector_access.*`
   ledger event를 재생한 projection이 canonical state다.
3. Mission Sources는 사용자가 고른 connection/site/page/version/range를 명시적으로
   승인해 source snapshot을 붙인다. 이 동작은 grant를 켜지 않고, grant 변경도
   source snapshot을 만들거나 삭제하지 않는다.

전역 Settings route는 다음이 primary path다.

```text
GET    /api/settings/connectors/confluence/connections
POST   /api/settings/connectors/confluence/connections
PATCH  /api/settings/connectors/confluence/connections/{connection_id}
DELETE /api/settings/connectors/confluence/connections/{connection_id}
POST   /api/settings/connectors/confluence/connections/{connection_id}/revoke
GET    /api/settings/connectors/confluence/connections/{connection_id}/sites
POST   /api/settings/connectors/confluence/connections/{connection_id}/sites/refresh
POST   /api/settings/connectors/confluence/oauth/start    # 0.0에서는 비활성
GET    /api/settings/connectors/confluence/oauth/callback # 0.0에서는 비활성
```

미션 grant route는 다음이 primary path다.

```text
GET    /api/missions/{mission_id}/connector-access/confluence
PUT    /api/missions/{mission_id}/connector-access/confluence
DELETE /api/missions/{mission_id}/connector-access/confluence
```

기존 mission-scoped connection list/read route는 호환용으로 안전한 metadata만
반환할 수 있다. 기존 mission-scoped connection lifecycle mutation route는 새 UI의
primary path가 아니며, Settings route를 사용하라는 deprecation 응답을 반환한다.
Settings list와 callback payload, mission grant event, MCP trace에는 access token,
refresh token, API token, client secret, Authorization header, raw provider body를
넣지 않는다.

## 현재 코드와의 맞춤

Plasma의 기존 source 모델은 이 연동을 담을 수 있다.

- `SourceSnapshot`은 미션 source의 기준점이다.
- `RawArtifact`는 immutable source bytes와 media type, size, SHA-256, storage
  URI, producer, created time, content를 저장한다.
- `ConnectorRef`는 외부 시스템과 source identity를 나타낸다.
- `SourceSnapshot.Locators`는 connector별 JSON metadata를 담는다.
- `SourceSnapshot.Access.RetrievalPolicy`는 이미 `snapshot_only`를 지원한다.

Liquid2 connector가 가장 가까운 선례다. Liquid2는 read-only connector를 통해
후보를 검색하고, 사용자 승인 경로에서만 source snapshot을 만든다. Confluence도
같은 제품 흐름을 따라야 한다. 다만 Confluence를 Liquid2 전용 interface에 억지로
넣지는 않는다.

첫 Confluence 구현은 Liquid2와 병렬인 app-level connector를 추가한다. 범용
`SourceConnector` refactor는 세 번째 외부 connector가 생겨 실제 공통 형태가
확인된 뒤 검토한다.

## Connector Identity

Confluence Cloud OAuth grant 하나는 여러 Atlassian site를 볼 수 있다. 따라서
page ID만으로 source를 식별하면 site 간 충돌 가능성이 있다. source identity에는
반드시 `cloud_id`가 들어가야 한다.

권장 identity는 다음과 같다.

```text
ConnectorID: confluence
ConnectorType: confluence_cloud
ExternalSourceID: <cloud_id>:<page_id>
ExternalURI: confluence://cloud/<cloud_id>/pages/<page_id>
ExternalVersion: <page version number>
ConnectorVersion: confluence-cloud-http.v1
```

권장 상수는 다음과 같다.

```text
ConfluenceConnectorID = "confluence"
ConfluenceConnectorType = "confluence_cloud"
ConfluenceHTTPConnectorV1 = "confluence-cloud-http.v1"
ConfluenceSnapshotMediaType = "application/vnd.plasma.confluence.snapshot+json"
ConfluenceSnapshotSchemaV1 = "plasma.confluence.snapshot.v1"
```

source locator는 `SourceSnapshot.Locators`에 저장한다.

```json
[
  {
    "locator_type": "confluence_page_body",
    "artifact_id": "art_...",
    "cloud_id": "example-cloud-id",
    "page_id": "67890",
    "content_id": "storage",
    "format": "confluence_storage",
    "start": 0,
    "end": 1234
  }
]
```

site URL, web URL, space, version, provider page metadata는 raw artifact payload의
`page`와 `metadata`에 들어간다. credential이나 private request header는 locator,
artifact metadata, ledger payload에 넣지 않는다.

## Snapshot Payload

`RawArtifact`에는 단순 HTML이 아니라 JSON snapshot payload를 저장한다. 이 payload는
고정된 source body와, 나중에 그 body를 해석하기 위한 metadata를 함께 담는다.

```json
{
  "schema_version": "plasma.confluence.snapshot.v1",
  "connector": {
    "connector_id": "confluence",
    "connector_type": "confluence_cloud",
    "external_source_id": "example-cloud-id:67890",
    "external_uri": "confluence://cloud/example-cloud-id/pages/67890",
    "external_version": "12",
    "connector_version": "confluence-cloud-http.v1"
  },
  "page": {
    "cloud_id": "example-cloud-id",
    "space_id": "12345",
    "space_key": "ENG",
    "page_id": "67890",
    "version": 12,
    "title": "Architecture Notes",
    "web_url": "https://example.atlassian.net/wiki/spaces/ENG/pages/67890",
    "updated_at": "2026-07-03T00:00:00Z"
  },
  "contents": [
    {"content_id": "storage", "role": "source", "format": "confluence_storage", "content": "<p>...</p>"},
    {"content_id": "plain_text", "role": "plain_text", "format": "text", "content": "..."}
  ],
  "metadata": {}
}
```

현재 구현은 기본 1 MiB storage body 크기 제한을 둔다. full page가 너무 크면
조용히 잘라서 저장하지 않고 typed too-large result를 반환한다. 브라우저와 CLI는
plain-text range option을 제공하며, 사용자가 선택한 range만 승인하면
`confluence_page_range` locator를 저장한다. range locator에는 `cloud_id`,
`page_id`, concrete version, `content_id`, `start`, `end`, `format`, `partial`이
포함된다. range preview는 result이며, 승인 전까지 source가 아니다.

## 인증

0.0 기본 인증은 API token connection이다. 사용자는 Settings에서 Atlassian
email, API token, Confluence site URL을 입력해 연결을 만든다. 이 연결은 site URL
에서 내부 `cloud_id`를 파생하고, Confluence REST 호출에는 Basic Auth를 사용한다.

Plasma는 connector credential을 `SourceSnapshot`, `RawArtifact`, locator JSON,
ledger payload, MCP trace summary, prompt, log 밖에 저장해야 한다.

OAuth 3LO는 과거 설계 기록과 일부 내부 helper로 남아 있지만 0.0 사용자 표면에서는
비활성화한다. Web API와 CLI의 OAuth start/exchange 경로는 API token 연결을 쓰라는
명확한 오류를 반환한다.

현재 구현에는 connector-owned credential store가 있다.

- account/user identity
- 접근 가능한 site 목록 cache
- 선택되었거나 요청된 `cloud_id`
- access token 만료 시각
- offline access를 사용할 때 refresh token 상태
- revoke/disconnect 상태

credential은 별도 product state다. source material이 아니며 source snapshot에
포함하지 않는다. `ConfluenceConnection`의 token 필드는 JSON 응답에서 제외된다.

## Atlassian MCP 선택지

Atlassian은 agent와 IDE client를 위한 remote MCP server를 제공한다. 이 경로는
탐색에는 유용하지만, Plasma의 결정론적 source snapshot 경로를 대체하지 않는다.

구분은 다음과 같다.

- Plasma connector: 사용자 승인 뒤 accepted mission source를 만든다.
- Atlassian MCP: agent가 조사 중 Atlassian 자료를 검색하거나 확인하게 한다.

Atlassian MCP를 agent 도구로 붙일 때의 장점은 다음과 같다.

- Plasma에 완전한 browse/search UI가 없어도 발견 작업을 빠르게 시작할 수 있다.
- Atlassian의 MCP permission, audit, domain control을 활용할 수 있다.
- agent가 넓은 질문으로 Atlassian content를 탐색할 수 있다.

한계는 다음과 같다.

- agent의 tool selection과 search phrasing은 비결정적이다.
- MCP search ranking과 snippet은 안정된 source identity가 아니다.
- agent summary와 fetched snippet은 result이지 source가 아니다.
- Plasma는 외부 MCP response를 source of truth로 삼을 수 없다.

따라서 제품 규칙은 이렇다. agent는 Atlassian MCP로 source 후보를 제안할 수 있다.
그러나 accepted source는 반드시 Plasma Confluence connector가 다시 fetch하고
snapshot해야 한다.

## Snapshot Update UX

업데이트는 기존 source snapshot을 덮어쓰면 안 된다. 기존 snapshot은 이미 evidence,
saved knowledge, report text의 근거일 수 있다. 저장된 body를 바꾸면 추적성이
깨진다.

권장 UX는 versioned update다.

1. source card에 pinned Confluence version을 보여준다.
2. 사용자가 명시적으로 `업데이트 확인`을 누를 수 있게 한다.
3. Plasma가 Confluence에서 현재 page metadata를 가져온다.
4. version이 같으면 `업데이트 없음`을 보여준다.
5. version이 바뀌었으면 old/new version number를 보여준다.
6. 사용자가 계속 진행하려 할 때만 새 page body를 가져온다.
7. 새 page body preview는 result이며, source가 아니다.
8. `업데이트된 snapshot 추가`, `이전 snapshot을 superseded로 표시`, `취소`를
   제공한다.
9. audit과 과거 report를 위해 두 snapshot 모두 주소 지정 가능하게 유지한다.

안전한 기본값은 다음과 같다.

- 새 page version에 대해 새 `SourceSnapshot`을 만든다.
- `source.update.available`, `source.updated` 같은 ledger event로 이전 snapshot과
  새 snapshot을 연결한다.
- 사용자가 선택한 view에서만 이전 source를 superseded 상태로 보이게 한다.
- 이전 source를 자동 soft remove하지 않는다.

나중에 stale badge나 scheduled update check를 추가할 수 있다. 그러나 첫 제품
범위에서는 update 적용 자체를 사용자 승인으로 유지한다.

## CLI 흐름

CLI는 agent 없이도 결정론적으로 source 작업을 할 수 있어야 한다. 0.0에서는
API token connection을 만든 뒤 같은 connection으로 탐색, 검색, snapshot을 수행한다.

지원되는 Confluence source 명령은 다음과 같다.

```sh
plasma sources confluence connect-token
plasma sources confluence connections
plasma sources confluence rename-connection <connection_id> --name "Docs"
plasma sources confluence revoke-connection <connection_id>
plasma sources confluence delete-connection <connection_id>
plasma sources confluence sites --connection <id>
plasma sources confluence spaces <mission_id> --connection <id> --cloud-id <cloud_id>
plasma sources confluence pages <mission_id> --connection <id> --cloud-id <cloud_id> --space-id <space_id>
plasma sources confluence children <mission_id> --connection <id> --cloud-id <cloud_id> --page-id <page_id>
plasma sources confluence search <mission_id> --connection <id> --cloud-id <cloud_id> --query "roadmap"
plasma sources confluence preview <mission_id> --connection <id> --cloud-id <cloud_id> --page-id <page_id>
plasma sources confluence snapshot <mission_id> --connection <id> --cloud-id <cloud_id> --page-id <page_id> --version <version>
plasma sources confluence snapshot <mission_id> --connection <id> --cloud-id <cloud_id> --page-id <page_id> --version <version> --range-content-id plain_text --range-start 0 --range-end 4000
plasma sources confluence check-update <mission_id> <source_id> --connection <id>
plasma sources confluence update-preview <mission_id> <source_id> --connection <id>
plasma sources confluence update <mission_id> <source_id> --connection <id> --version <new_version>
```

MCP search remains discovery-oriented and requires an explicit mission grant.
Its Confluence outputs are candidates or results, not accepted sources. Source
promotion still requires browser/API/CLI approval paths that re-fetch through
the Plasma connector.

## Validation Status

Mock and unit validation currently covers connector browse/search/read/update,
large page range handling, Settings web API routes, mission grant ledger
projection, OAuth callback redaction, old snapshot readability after revoke,
CLI command wiring, MCP default-off grant enforcement, source/result
boundaries, and static browser controls.

Live Atlassian tenant validation is **pending live credentials** as of
2026-07-05. The checklist is tracked in
[`confluence-live-validation-checklist.md`](confluence-live-validation-checklist.md).
Mock validation must not be treated as final live usability validation.

PR handoff checklist:

- 새 브라우저 UI는 connection lifecycle에
  `/api/settings/connectors/confluence/*`만 사용한다.
- 미션 Sources는 browse/preview/snapshot/update만 수행하며 grant를 변경하지 않는다.
- 미션 connector access grant는 `mission.connector_access.*` ledger event projection에서
  재생된다.
- grant enable/disable은 source snapshot을 만들거나 지우지 않는다.
- MCP Confluence search는 grant가 없으면 connector factory 호출 전에 거부된다.
- 기존 mission-scoped lifecycle mutation route는 primary path가 아니며 deprecation
  응답으로 제한된다.
- Settings payload, mission grant event, MCP output/trace, source artifact, docs에는
  credential material이나 raw provider response가 들어가지 않는다.

```sh
plasma sources confluence connect-token \
  --connection cnf_docs \
  --email person@example.com \
  --api-token "$ATLASSIAN_API_TOKEN"

plasma sources confluence search <mission_id> \
  --connection cnf_docs \
  --cloud-id <cloud_id> \
  --query "..."

plasma sources confluence snapshot <mission_id> \
  --connection cnf_docs \
  --cloud-id <cloud_id> \
  --page-id <page_id> \
  --version <version>

plasma sources confluence check-update <mission_id> <source_id> \
  --connection cnf_docs

plasma sources confluence update <mission_id> <source_id> \
  --connection cnf_docs \
  --version <new_version>
```

API token 연결에는 site URL도 필요하다. Web Settings UI는 site URL을 같이 저장한다.
CLI에서는 connection에 저장된 site 또는 명령별 `--site-url` 입력을 사용한다.

저장된 snapshot에는 항상 concrete Confluence version이 들어간다.

## Web JSON API 흐름

브라우저 서버의 Confluence JSON API는 connection lifecycle, 미션 grant, source
attachment를 분리한다.

```text
GET    /api/settings/connectors/confluence/connections
POST   /api/settings/connectors/confluence/connections
PATCH  /api/settings/connectors/confluence/connections/{connection_id}
DELETE /api/settings/connectors/confluence/connections/{connection_id}
POST   /api/settings/connectors/confluence/connections/{connection_id}/revoke
GET    /api/settings/connectors/confluence/connections/{connection_id}/sites
POST   /api/settings/connectors/confluence/connections/{connection_id}/sites/refresh
POST   /api/settings/connectors/confluence/oauth/start    # 0.0에서는 비활성
GET    /api/settings/connectors/confluence/oauth/callback # 0.0에서는 비활성

GET    /api/missions/{mission_id}/connector-access/confluence
PUT    /api/missions/{mission_id}/connector-access/confluence
DELETE /api/missions/{mission_id}/connector-access/confluence

GET    /api/missions/{mission_id}/sources/confluence/connections
GET    /api/missions/{mission_id}/sources/confluence/sites?connection_id=cnf_...
POST   /api/missions/{mission_id}/sources/confluence/search
POST   /api/missions/{mission_id}/sources/confluence/snapshot
POST   /api/missions/{mission_id}/sources/confluence/check-update
POST   /api/missions/{mission_id}/sources/confluence/update
```

정적 UI는 Settings 탭에서 Confluence connection registration/lifecycle을 제공하고,
Sources 탭에서는 이미 등록된 connection과 site를 고른 뒤 page를 검색해 선택한
page를 source snapshot으로 저장한다. 미션의 agent/MCP 검색 grant control은 source
attachment control과 분리되어 있다. 저장된 Confluence source card는 Confluence
badge, pinned version, update check action을 보여주며, connection revoke/delete 뒤에도
기존 snapshot 읽기는 유지되어야 한다.

호환용 mission-scoped connection list/read route는 안전한 metadata만 반환한다.
mission-scoped OAuth start/callback, connection create/rename/revoke/delete, site
refresh mutation route는 새 UI의 primary path가 아니며 Settings 경로 안내와 함께
거부된다.

서버 설정은 `PLASMA_CONFLUENCE_OAUTH_CLIENT_ID`,
`PLASMA_CONFLUENCE_OAUTH_CLIENT_SECRET`,
`PLASMA_CONFLUENCE_OAUTH_REDIRECT_URI`, `PLASMA_CONFLUENCE_OAUTH_SCOPES` 또는
동일한 `config.toml` 키(`confluence_oauth_client_id` 등)로 제공할 수 있다.

## Agent와 MCP 흐름

Plasma MCP server는 기존 source search 패턴으로 Confluence 후보 검색을 노출하지만
미션 grant가 없으면 Confluence connector를 만들기 전에 거부한다. grant가 있으면
server가 ledger-derived projection의 `connection_id`, `cloud_id`, optional
`space_key`를 authoritative scope로 사용한다. Tool input의 `connection_id`,
`cloud_id`, `space_key`는 grant와 일치하거나 더 좁은 범위여야 한다.

```json
{
  "mission_id": "mis_...",
  "connectors": ["confluence"],
  "connection_id": "cnf_...",
  "cloud_id": "example-cloud-id",
  "space_key": "ENG",
  "query": "architecture notes"
}
```

결과에는 title, space, URL, page version 같은 후보 metadata를 담는다. private page
body excerpt는 output이나 trace summary에 넣지 않는다.

agent source promotion은 기본값에서 계속 비활성화한다. agent는 visible answer에서
source 후보를 추천할 수 있지만, Web 또는 CLI surface가 accepted snapshot action을
수행해야 한다. MCP는 grant를 직접 만들거나 넓힐 수 없고, source snapshot mutation
tool도 노출하지 않는다.

나중에 운영자용 MCP mutation 경로를 Confluence에 추가하더라도, 반드시 concrete
`cloud_id`, `page_id`, `version`을 요구하고, 쓰기 전에 page version drift를 다시
확인해야 한다.

## 결정론 경계

accepted source 경로는 결정론적으로 설계한다.

- 사용자가 선택한 `cloud_id`
- 사용자가 선택한 `page_id`
- 구체적인 Confluence page version
- Plasma의 deterministic REST fetch
- deterministic snapshot JSON construction
- deterministic SHA-256 content hash
- deterministic ledger event creation

발견 경로는 비결정적일 수 있다.

- agent search phrasing
- search ranking
- source candidate recommendation
- agent summary
- Atlassian MCP tool use

이 경계는 의도적이다. agent는 유용한 원본 자료를 찾는 일을 돕는다. Plasma는
사용자가 승인한 version-pinned source snapshot만 저장한다.

## 구현 상태

### 완료: Connector와 Snapshot Core

- Confluence search candidate와 page document app model을 추가한다.
- `ConfluenceSourceConnector`를 추가한다.
- search와 snapshot 생성을 위한 service method를 추가한다.
- API token basic auth를 받는 Confluence REST client를 추가한다.
- 승인된 page를 `snapshot_only` raw artifact로 저장한다.
- snapshot 생성 전에 version drift guard를 적용한다.
- identity, locator shape, snapshot payload, drift guard 단위 테스트를 추가한다.

### 완료: Web JSON API와 CLI Surface

- Confluence connection, site, page search, snapshot Web JSON route를 추가한다.
- API token connection, site list, search, snapshot, update check, update CLI
  command를 추가한다.
- 여러 Atlassian site는 `cloud_id`와 connection site cache로 구분한다.
- 정적 UI에 API token 연결, site 선택, page 검색, source snapshot 추가,
  source card badge, update check action을 추가한다.

### 완료: MCP와 Agent Discovery

- `plasma.sources.search`가 `confluence`를 지원하게 한다.
- 기본 agent MCP session에서는 source promotion을 계속 비활성화한다.
- private body excerpt가 trace summary에 남지 않는 candidate output을 추가한다.
- Atlassian MCP 결과는 source 후보이지 source가 아니라는 guidance를 추가한다.

### 완료: Update State

- update check event와 projection state를 추가한다.
- metadata update check의 성공과 안전하게 분류된 원격 실패를 source state에 마지막
  확인 시각과 함께 투영한다. 404는 삭제로 단정하지 않고 원본을 찾거나 접근할 수
  없었던 확인 결과로 표시하며, 저장된 snapshot은 그대로 유지한다.
- update를 새 snapshot으로 생성한다.
- superseded/source lineage 표시를 추가한다.
- 자동 source replacement 없이 이전 snapshot을 주소 지정 가능하게 유지한다.
- stale badge와 rendered diff는 아직 정적 UI에 없다. 현재 UI는 update check 결과를
  보여주고, 사용자가 확인하면 새 snapshot을 생성한다.

### 완료: Report Source Context

- 새 `report.draft.pending`은 당시 active Confluence snapshot과 마지막으로 알려진
  update check 상태를 allowlist metadata로 한 번 캡처한다.
- retry와 restart recovery는 최초 pending의 context를 유지하고, 새 보고서만 현재
  상태를 다시 캡처한다.
- 이 정보는 보고서 본문 밖에서 "생성 시점에 사용 가능했던 소스 정보"로 표시한다.
  report prompt, provider request, Markdown, citation에는 삽입하지 않으며 보고서 생성
  시 Confluence update check나 snapshot refresh를 자동 실행하지 않는다.

## 결정된 사항과 남은 범위

- API token/email connection을 0.0 기본 경로로 둔다.
- credential은 mission source가 아닌 connector connection product state다.
- 첫 update UX는 old/new version metadata와 새 snapshot 생성으로 제한한다.
- 큰 page는 기본 1 MiB를 넘으면 거부한다. section/range snapshot은 후속 범위다.
- OAuth 3LO start/callback은 0.0에서 비활성이다. 향후 다시 도입하려면 별도 설계와
  live 검증을 거친다.

## 공식 참고 문서

- Atlassian OAuth 2.0 3LO:
  <https://developer.atlassian.com/cloud/jira/software/oauth-2-3lo-apps/>
- `cloudId` 기반 Atlassian Cloud API 호출:
  <https://developer.atlassian.com/cloud/oauth/getting-started/making-calls-to-api/>
- Confluence OAuth scopes:
  <https://developer.atlassian.com/cloud/confluence/scopes-for-oauth-2-3LO-and-forge-apps/>
- Confluence REST API v2 pages:
  <https://developer.atlassian.com/cloud/confluence/rest/v2/api-group-page/>
- Confluence REST API v2 spaces:
  <https://developer.atlassian.com/cloud/confluence/rest/v2/api-group-space/>
- Confluence CQL search:
  <https://developer.atlassian.com/cloud/confluence/rest/v1/api-group-search/>
- Confluence rate limiting:
  <https://developer.atlassian.com/cloud/confluence/rate-limiting/>
- Atlassian remote MCP server:
  <https://support.atlassian.com/atlassian-rovo-mcp-server/docs/getting-started-with-the-atlassian-remote-mcp-server/>
