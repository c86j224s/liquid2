// Package mcp는 Plasma가 에이전트에게 노출하는 MCP transport 표면을 소유한다.
//
// 이 패키지는 tool 이름, 입력 schema, binding 검증, idempotency, bounded draft
// 상태를 관리하고 app/reporting 포트를 호출한다. 지속 제품 상태 전이는
// app과 reporting 계층의 계약을 통해서만 수행하며, MCP tool handler가 독자적인
// 제품 정책의 source of truth가 되면 안 된다.
package mcp
