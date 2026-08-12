# Plasma MCP Tool 안내

URI: `plasma://docs/mcp/tools`

이 문서는 Plasma MCP tool surface를 제품 경계를 바꾸지 않고 사용하는 방법을 설명합니다. 현재 input schema와 output envelope은 `tools/list`가 반환하는 tool definition을 기준으로 합니다.

## 기본 호출 규칙

- Tool 이름, input schema, output envelope은 안정적인 wire contract입니다. Tool 이름을 바꾸거나 `tools/list`와 다른 축약 shape로 호출하지 않습니다.
- Mission-bound server에서는 tool argument를 server의 mission과 session binding 안에 둡니다.
- Mutating tool을 호출하기 전에 사용자 요청, runner binding, idempotency key가 필요한지 확인합니다.
- Tool result의 `content`는 Plasma가 안전하게 만든 result 표현입니다. Provider 응답이나 local runtime state를 새 source처럼 덧붙이지 않습니다.
- Search result, grep match, connector result는 candidate입니다. 읽고 판단하기 전까지 source, evidence, saved knowledge가 아닙니다.
- `plasma.research.grep`은 case-insensitive literal substring search입니다. 전체 query가 contiguous하게 나타나야 하며, 서로 다른 concept은 별도의 짧은 search로 나눕니다. 검색으로 가져온 각 candidate 안에서 발견된 비중첩 match는 모두 기존 cursor와 limit pagination으로 반환됩니다.

## 기본 조사 흐름

1. `plasma.research.outline`으로 mission ledger 구조를 파악하고 `last_sequence`를 기억합니다.
2. 기존 provider session을 재개한 뒤 mission 변경 확인이 필요하면 마지막으로 확인한 sequence로 `plasma.research.changes`를 호출합니다. 반환된 `current_sequence`를 기억하고, `resync_required`가 true이면 outline을 다시 읽습니다.
3. `plasma.research.list` 또는 `plasma.research.grep`으로 candidate를 좁힙니다.
4. `plasma.research.read`로 필요한 object 또는 bounded source chunk를 읽습니다.
5. Source, artifact, observation, ledger-event 관계가 중요하면 `plasma.research.references`를 사용합니다.
6. 사용자 검토 가치가 있는 original material은 `plasma.sources.candidates.propose`로 기록합니다.

## Source Tool

- `plasma.sources.list`: mission의 active source snapshot을 나열합니다.
- `plasma.sources.read`: accepted source snapshot 또는 live local path reference에서 bounded content를 읽습니다.
- `plasma.sources.tree`, `plasma.sources.grep`: accepted live local path source 안의 tree 또는 snippet을 관찰합니다.
- `plasma.sources.search`: mounted read-only connector에서 original-material candidate를 찾습니다.
- `plasma.sources.candidates.propose`, `plasma.sources.candidates.read`: 사용자 승인 전 source candidate를 기록하고 읽습니다.
- `plasma.local_path.roots`, `plasma.local_path.tree`: allowlisted local path root를 root id와 relative path로 탐색합니다.

Operator-only source mutation tool은 server에서 명시적으로 enabled된 경우에만 보입니다.

## Workflow Tool

`plasma.workflow.start`, `plasma.workflow.status`, `plasma.workflow.stop`은 bounded mission workflow run을 요청, 확인, 중지합니다. Start는 current user turn과 runner binding에 맞춰 work를 queue합니다. MCP tool call 안에서 provider를 호출하지 않습니다.

## Report Tool

Report tool은 runner가 만든 stage-specific MCP server에만 노출됩니다. Plan submit, requirement mapping, part assembly, part edit, long-form finalization, final edit stage, patch tool은 각각 책임이 다릅니다. 한 stage의 tool로 다른 stage의 policy를 대신하지 않습니다.

## Mermaid Tool

Diagram source를 사용자에게 보여주기 전에 `plasma.mermaid.validate`를 호출합니다. `ok: true`는 static preflight를 통과했다는 뜻이며 browser-render 보장은 아닙니다.
