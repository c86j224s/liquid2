// Package plan은 Plasma 보고서 계획 생성 단계를 소유한다.
//
// 이 패키지는 planned 보고서의 prompt, MCP allowlist, fresh/fork/same session 시작 계약,
// canonical plan 제출 lifecycle, durable recovery를 다룬다. 본문 작성, long-form downstream
// stage, HTTP 요청 정규화는 이 패키지의 책임이 아니다.
package plan
