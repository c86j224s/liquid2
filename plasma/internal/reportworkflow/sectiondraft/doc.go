// Package sectiondraft는 장문 report Section Markdown 작성 단계를 소유한다.
//
// 이 패키지는 Section prompt, read-only MCP allowlist, provider 요청, Section artifact와
// section.created event 저장을 담당한다. serial/fanout 순서, worker limit, session fork는
// root reportworkflow runner가 결정한다.
package sectiondraft
