// Package confluence는 Atlassian Confluence HTTP API를 Plasma source connector
// 포트에 맞게 감싸는 adapter다.
//
// 이 패키지는 인증 헤더 구성, Confluence API 호출, 페이지 본문을 source snapshot
// 후보 형태로 변환하는 일을 맡는다. 미션별 접근 권한, source 승인 여부, 장부 기록
// 같은 제품 상태 결정은 app 계층에서 처리한다.
package confluence
