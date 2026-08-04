// Package app은 Plasma의 제품 상태 전이와 application service 경계를 소유한다.
//
// HTTP, MCP, CLI, connector, storage adapter는 이 패키지의 요청/결과 타입과
// service 메서드를 통해 제품 규칙에 접근한다. 이 패키지는 장부 이벤트, source,
// report, workflow, connector access 같은 지속 제품 상태의 의미를
// 정의하지만, 구체적인 SQL shape, HTTP route shape, provider 실행 방식은 각
// adapter 패키지에 둔다.
package app
