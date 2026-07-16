# Plasma

Plasma는 사용자가 대화로 조사를 조향할 수 있는 research workspace입니다.

사용자는 mission을 만들고, source를 붙이고, agent와 대화하고, agent가 도구로 조사하게 한 뒤,
그 작업에서 report artifact를 생성할 수 있습니다. 제품의 중심은 terminal이 아닙니다. 중심 흐름은
대화, source reading, investigation, report generation으로 이어지는 research loop입니다.

Plasma는 Liquid2와 별도 제품입니다. Liquid2는 개인 reference material을 저장합니다. Plasma는 선택된
Liquid2 문서를 connector로 읽을 수 있지만, 자체 database, mission ledger, source, 대화, report를
유지합니다.

## 기본 사용 흐름

1. 주제, 목표, 대략적인 질문으로 mission을 만듭니다.
2. Pasted text, URL, PDF, media URL, Liquid2 document, allowlisted local file/repository를 source로
   추가합니다.
3. Mission 안에서 agent와 대화합니다. 가능하면 agent는 같은 provider session을 이어 쓰므로 여러 턴에
   걸쳐 맥락이 유지됩니다.
4. Source text를 prompt에 모두 붙이는 대신, agent가 MCP tool로 source를 search/read/inspect하게 합니다.
5. Source candidate를 검토한 뒤 mission source로 승인합니다.
6. Report를 artifact로 생성합니다. Markdown이 기본 report format이고, HTML은 rendering/export form입니다.

Automatic investigation은 같은 mission workflow의 확장입니다. 사용자가 자리를 비웠을 때 더 좋은 다음
질문을 던지고 계속 조사하는 것이 목적입니다. 별도의 숨겨진 research product를 만드는 것이 아닙니다.

## 제품 규칙

- Source는 원본 재료입니다. URL, PDF, file, Liquid2 document, media link, local path reference가 여기에
  해당합니다.
- Agent answer는 result입니다. Source를 인용하거나 추천할 수 있지만 source 자체는 아닙니다.
- Report는 mission work에서 조립된 output artifact입니다.
- Plasma는 thin guidance와 MCP/source read를 사용해야 합니다. 매 prompt마다 거대한 ledger나 source pack을
  붙여서 research quality를 해결하려 하면 안 됩니다.
- Browser UI, CLI, MCP tool, agent provider, source reader, report renderer는 같은 product state 위의 교체
  가능한 표면이어야 합니다.
- Plasma research state는 Liquid2 table이 아니라 Plasma database에 속합니다.

## 현재 기능

- Conversation turn, source event, MCP call log, report artifact를 담는 mission ledger.
- 진행 중 작업과 현재 브라우저에서 아직 확인하지 않은 완료 또는 실패 작업을 보여주는 mission 목록 활동 표시.
  확인 상태는 현재 브라우저에만 저장됩니다.
- Mission creation, conversation, source management, source candidate review, automatic investigation,
  report generation을 위한 browser workspace.
- Text와 textual URL source snapshot.
- Metadata-first, chunked read 방식의 PDF source support.
- Image URL snapshot, audio/video URL metadata reference.
- Codebase와 document analysis를 위한 allowlisted local path source.
- Read-only Liquid2 connector boundary.
- API token connection, site/space/page browsing, candidate review, version-pinned snapshot, large page
  range snapshot, update preview/approval을 지원하는 Confluence Cloud source intake.
- Outline, list, grep, read, reference traversal을 위한 MCP research tools.
- 가능한 경우 session resume을 사용하는 Codex-backed, Claude-backed agent turn.
- Markdown report, long-form part/section report, HTML export.
- 기존 artifact를 in-place edit하지 않고 prior report session에서 새 report artifact version을 만드는
  MCP-backed Markdown report patching.

아직 experimental/future work인 영역도 있습니다. Mixed-provider mission, background autonomous worker,
richer media inspection, external publishing adapter, stronger source discovery, more polished designed HTML
report가 여기에 해당합니다.

## 개발 Quick Start

Workspace root에서 Liquid2와 Plasma를 함께 실행합니다.

```sh
./dev-browser.sh start
./dev-browser.sh status
./dev-browser.sh stop
```

Plasma만 실행합니다.

```sh
./dev-browser.sh plasma start
./dev-browser.sh plasma status
./dev-browser.sh plasma logs
./dev-browser.sh plasma stop
```

Plasma development 기본값은 browser port `6002`와
`~/research-artifacts/liquid2/plasma/runtime/dev-6002/` 아래 local SQLite database입니다.
Runtime setting은 environment variable 대신 TOML file로 옮길 수 있습니다. Workspace
[configuration guide](../docs/configuration.md)를 참고하세요.

Release surface를 실행합니다.

```sh
./release-browser.sh plasma start
./release-browser.sh plasma status
./release-browser.sh plasma logs
./release-browser.sh plasma stop
```

Plasma release 기본값은 browser/API port `3002`입니다. 기본 database는 macOS에서
`~/Library/Application Support/Plasma/plasma.db`, WSL2에서
`${XDG_DATA_HOME:-$HOME/.local/share}/plasma/plasma.db`입니다.

## 자주 쓰는 명령

검사를 실행합니다.

```sh
make -C plasma check
```

제품 디렉터리에서 작업합니다.

```sh
cd plasma
make check
make dev-browser-start
make dev-browser-status
make dev-browser-logs
make dev-browser-stop
```

Agent 없이 browser server를 수동 실행합니다.

```sh
cd plasma
go run ./cmd/plasma serve -db /tmp/plasma-ui.db -addr 127.0.0.1:6002
```

Codex agent execution을 켜고 실행합니다.

```sh
cd plasma
go run ./cmd/plasma serve \
  -db /tmp/plasma-ui.db \
  -addr 127.0.0.1:6002 \
  -agent codex
```

Plasma는 새 Codex 세션을 기본적으로 `gpt-5.6-terra`와 `medium` 추론 강도로
시작합니다. 미션 제어에서 GPT-5.6 Sol, Terra, Luna와 해당 모델이 지원하는 추론
강도를 선택한 뒤 새 에이전트 세션을 시작할 수 있습니다. 이때 미션 데이터와 저장된
소스는 유지되고 Codex 세션 연속성만 초기화됩니다. 별도 모델 선택기가 없는 보고서
생성은 이 미션 선택값을 이어받으며, 선택값이 없으면 같은 Terra/medium 기본값을
사용합니다.

Browser에서 Codex와 Claude를 모두 사용할 수 있게 실행합니다.

```sh
cd plasma
go run ./cmd/plasma serve \
  -db /tmp/plasma-ui.db \
  -addr 127.0.0.1:6002 \
  -agent codex,claude \
  -claude-model haiku
```

Browser script는 같은 설정을 environment variable로 노출합니다.

```sh
PLASMA_DEV_BROWSER_AGENT=codex,claude \
PLASMA_DEV_BROWSER_CLAUDE_MODEL=haiku \
  ./dev-browser.sh plasma restart
```

특정 Claude CLI binary를 직접 지정해야 할 때는 `PLASMA_DEV_BROWSER_CLAUDE` 또는
`PLASMA_RELEASE_BROWSER_CLAUDE`를 사용합니다. Claude CLI에 per-turn 예산 제한을 일부러 걸고 싶을 때만
`PLASMA_DEV_BROWSER_CLAUDE_MAX_BUDGET_USD` 또는 `PLASMA_RELEASE_BROWSER_CLAUDE_MAX_BUDGET_USD`를
사용합니다.

## Agent와 Local Source

Agent MCP server는 공유 Plasma research-tool allowlist로 시작됩니다. Codex는 `plasma.sources.read`로
accepted live local path source를 읽을 수 있고, `plasma.sources.tree`, `plasma.sources.grep`으로 source
boundary 내부를 검사할 수 있습니다. 이 도구들은 arbitrary absolute path나 root-wide local path browsing이
아니라 `snapshot_id`와 optional `subpath`를 받습니다.

Claude는 추가로 configured agent work directory 안에서 built-in web tools와 read-only file tools를 사용할
수 있습니다. Shell execution, file edit, task spawning, notebook edit은 비활성화되어 있습니다. Agent work
directory 밖의 자료는 Plasma local source root로 붙여서 mission-bound MCP read로 보이게 해야 합니다.

현재는 한 mission이 하나의 agent provider type만 사용합니다. 첫 provider-backed action이 mission을 그
provider로 lock합니다. Codex로 시작한 mission은 Codex로 이어가고, Claude로 시작한 mission은 Claude로
이어갑니다. Provider 비교는 새 mission에서 하세요.

Local source root를 설정합니다.

```sh
PLASMA_LOCAL_SOURCE_ROOTS=repo=/path/to/repo,docs=/path/to/docs \
  go run ./cmd/plasma serve -db /tmp/plasma-ui.db -addr 127.0.0.1:6002
```

Client는 root를 `root_id`와 relative path로 참조합니다. Plasma는 client absolute path를 거부하고,
configured absolute server root를 Web, CLI, MCP surface로 반환하지 않습니다.

## UI 없는 Research Flow

Plasma는 browser 없이도 유용해야 합니다. Plasma MCP server를 가진 agent는 같은 mission ledger와 source를
검사할 수 있습니다.

```sh
cd plasma
go run ./cmd/plasma mcp \
  -db /tmp/plasma-ui.db \
  -mission-id mis_... \
  -agent-session-id ses_...
```

일반적인 MCP-driven flow는 다음과 같습니다.

- `plasma.research.outline`으로 시작합니다.
- `plasma.research.list` 또는 `plasma.research.grep`으로 후보를 찾습니다.
- `plasma.research.read`로 bounded chunk를 읽습니다.
- 보고 전에 `plasma.research.references`로 관계를 확인합니다.

현재 미션 메타데이터는 CLI 또는 mission-bound idempotent `plasma.mission.update` MCP 도구에서 같은 애플리케이션 계약으로 편집합니다.

```sh
go run ./cmd/plasma missions update mis_... -title "현재 제목" \
  -scope-included "포함할 주제" -scope-excluded "제외할 주제"
```

MCP 수정은 사용자가 명시적으로 요청한 편집에만 사용합니다. Plasma가 내부에서 띄운 조사 에이전트의 기본 도구 목록에는 이 도구를 넣지 않습니다.

요청별 보고서 방향은 선택 사항이며 해당 초안의 약한 편집 축으로만 동작합니다.

```sh
go run ./cmd/plasma reports draft mis_... -wait \
  -agent-model gpt-5.5 -agent-reasoning-effort high \
  -direction-hint "권고 전에 운영 위험을 비교"
```

힌트는 소스나 미션 설정이 아닙니다. Plasma는 이후 보고서 요청으로 힌트를 복사하지 않으며, 대화·말투 보정·보고서 수정·HTML 내보내기 프롬프트에도 힌트를 다시 넣지 않습니다. 다만 이는 프롬프트 전달 경계에 대한 보장이지 제공자 세션 기록을 지운다는 뜻은 아닙니다. 같은 제공자 세션을 의도적으로 이어 쓰는 경로에서는 앞선 보고서 프롬프트가 세션 맥락에 남아 있을 수 있습니다.

기존 Markdown report artifact를 CLI로 patch합니다.

```sh
cd plasma
go run ./cmd/plasma reports patch mis_... \
  -db /tmp/plasma-ui.db \
  -base-artifact art_... \
  -instruction "사이토 도산 관련 조사 내용을 반영해 서술을 보강" \
  -wait
```

Report patching은 해당 patch run에 scoped된 임시 MCP tool surface를 사용합니다. Agent는 저장된 Markdown
report artifact를 도구로 읽고 수정한 뒤 새 report artifact version을 finalize합니다. Base artifact는 그대로
유지됩니다.

## 문서

- [Plasma README Korean](README.ko.md)
- [Documentation Index](docs/README.md)
- [Documentation Index Korean](docs/README.ko.md)
- [Glossary](docs/glossary.md)
- [Glossary Korean](docs/glossary.ko.md)
- [Product Flow](docs/product-flow.md)
- [C1 Default Loop](docs/c1-default-loop.md)
- [C1 Default Loop Korean](docs/c1-default-loop.ko.md)
- [Automatic Investigation](docs/automatic-investigation.md)
- [Product Architecture](docs/product-architecture.md)
- [Product Architecture Korean](docs/product-architecture.ko.md)
- [Media Source Implementation Design](docs/media-source-implementation-design.md)
- [Confluence Cloud Source 연동 기록](docs/confluence-source-integration.md)
- [Confluence live validation checklist](docs/confluence-live-validation-checklist.md)
- [Token Diet Instrumentation](docs/token-diet-instrumentation.md)
- [Evidence Signal Model](docs/evidence-signal-model.md)
- [Evidence Signal Model Korean](docs/evidence-signal-model.ko.md)
- [Experiment Index](docs/experiments/README.md)

### 요청별 보고서 모델 선택

Browser 보고서 제어와 `reports draft`에서 모델과 추론 강도를 비워 두면 같은 executor의 최신 미션 세션 설정을, 없으면 설정된 provider 기본값을 상속합니다. 모델만 지정하면 그 모델이 공개한 기본 추론 강도를 사용합니다. Plasma는 유효 조합을 검증한 뒤 확정값과 `agent_selection_source`를 `report.draft.pending`에 동결하며 stale 복구도 이 값을 재사용합니다.
