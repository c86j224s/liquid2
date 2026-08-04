// Package web은 Plasma HTTP server와 browser-facing route adapter를 소유한다.
//
// 이 패키지는 request/response shape, agent process 호출, browser-render 보조
// 작업, 정적 파일 제공을 맡는다. 미션, source, report, workflow의 지속 제품
// 규칙은 app/reporting/workflow 계층으로 위임해야 하며 route handler가 독자적인
// 상태 정책을 만들면 안 된다.
package web
