// Package mcptrace는 MCP tool 호출을 장부에 남길 때 쓰는 event payload builder를
// 제공한다.
//
// 이 패키지는 추적 이벤트의 안정적인 JSON 모양만 만든다. tool 실행, 오류 복구,
// 사용자 메시지 구성은 MCP transport나 application service의 책임이다.
package mcptrace
