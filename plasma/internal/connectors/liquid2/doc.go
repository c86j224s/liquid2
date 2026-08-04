// Package liquid2는 Liquid2 문서를 Plasma source connector 포트에 맞게 읽는
// HTTP adapter다.
//
// 이 패키지는 Liquid2 API 응답을 Plasma의 source 후보와 snapshot 입력으로
// 변환한다. Liquid2 문서가 미션 source로 승인되는 시점과 장부 이벤트 기록은 이
// 패키지 밖의 app service가 결정한다.
package liquid2
