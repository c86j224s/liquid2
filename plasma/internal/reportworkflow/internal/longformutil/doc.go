// Package longformutil은 장문 report stage들이 공유하는 순수 형식 helper를 제공한다.
//
// 이 패키지는 prompt, MCP allowlist, provider 호출, 저장 정책을 소유하지 않는다. stage
// 패키지들이 동일하게 유지해야 하는 JSON 형식, session 검증, 파일명, Markdown count 같은
// 기계적 불변 조건만 둔다.
package longformutil
