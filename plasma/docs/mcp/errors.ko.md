# Plasma MCP 오류 안내

URI: `plasma://docs/mcp/errors`

이 문서는 Plasma MCP client가 오류를 해석하고 다음 행동을 고르는 기준입니다.

## 오류 층

Plasma MCP에는 두 오류 층이 있습니다.

- JSON-RPC protocol error: 요청 envelope, method, params shape 자체가 잘못된 경우입니다. 예를 들어 알 수 없는 method는 method-not-found 오류로 끝납니다.
- Tool execution error: tool 이름과 JSON-RPC 요청은 유효하지만 Plasma 제품 규칙, binding, 입력 값, connector 상태 때문에 실행이 실패한 경우입니다.

Tool execution error는 `tools/call` 결과의 `isError: true`와 Plasma tool response envelope 안의 `error` 필드로 전달됩니다. 이 envelope은 안정적인 tool contract입니다. Client는 `error.error_kind`, `error.message`, `error.retryable`, `error.related_object_ids`를 확인해야 합니다.

## 주요 error_kind

- `validation`: 입력 값, mission binding, session binding, 허용 범위가 맞지 않습니다. 재시도하기 전에 입력을 고칩니다.
- `approval_required`: 사용자가 승인해야 하는 상태 전이입니다. Agent가 우회하지 않습니다.
- `conflict`: 같은 idempotency key 또는 draft 상태에 서로 다른 변경이 충돌했습니다. 현재 상태를 다시 읽고 새 요청을 만듭니다.
- `binding`: 이 MCP server instance에 허용되지 않은 보고서 단계나 session 경계입니다. Runner가 제공한 binding 값을 바꾸지 않습니다.
- `internal`: server 내부 실패입니다. `retryable`이 false이면 같은 요청 반복으로 해결될 가능성이 낮습니다.

## Resource URI 오류

`resources/read`는 invalid resource URI 입력과 unknown resource를 구분합니다.

- Invalid resource URI 입력은 JSON-RPC `-32602`를 반환합니다. Blank URI는 invalid params입니다. `"not a uri"` 같은 malformed URI도 invalid URI 입력입니다.
- Unknown이지만 well-formed resource는 JSON-RPC `-32002`와 `resource not found`를 반환합니다. 예를 들어 `plasma://docs/mcp/unknown`은 well-formed이지만 Plasma static resource catalog에 없습니다.
- 이 예시는 공개 저장소에 안전한 값만 사용합니다. Mission, session, source, runtime 식별자를 포함하지 않습니다.

`resources/list`는 static non-paginated catalog입니다. Params가 없거나 `null`, `{}`, empty cursor이면 허용합니다. Malformed params 또는 non-empty cursor는 JSON-RPC `-32602`와 `invalid params`를 반환합니다.

## 재시도 기준

`retryable: true`는 agent가 입력을 보완하거나 일시 조건을 기다린 뒤 재시도할 수 있다는 뜻입니다. 성공을 보장하지 않습니다. `retryable: false`는 같은 인자를 그대로 반복하지 말라는 뜻에 가깝습니다.

Idempotency key가 있는 mutating tool은 같은 의미의 재시도에 같은 key를 유지합니다. 서로 다른 변경을 같은 key로 보내면 conflict가 날 수 있습니다.

## 공개 안전성

오류 메시지는 사용자와 agent에게 보여도 안전해야 합니다. Credentials, cookie, private key, provider 응답, 민감한 URL, source 본문 전체를 문서나 보고서에 붙여 넣지 않습니다. 오류를 보고할 때는 안전한 `error_kind`, 요약 메시지, 관련 object id를 사용합니다.
