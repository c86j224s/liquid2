// Package partassembly는 작성된 Sections를 Part Markdown으로 보존 조립하는 단계를 소유한다.
//
// 이 패키지는 Part connective prompt, 선택적 MCP assembly edit tools binding/replay, provider
// 요청, Part artifact와 part.created event 저장을 담당한다. Part 순서와 session fork는 root가
// 결정한다.
package partassembly
