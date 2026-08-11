// Package partedit은 조립된 Part를 별도 MCP edit 도구로 검토/저작하는 단계를 소유한다.
//
// 이 패키지는 Part edit/author prompt, tool allowlist, durable start/replay binding 검증,
// provider 요청과 제출 결과 채택을 담당한다. root는 canonical flag에 따라 edit vs author와
// session fork 순서를 결정한다.
package partedit
