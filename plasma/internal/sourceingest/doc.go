// Package sourceingest는 외부 입력을 Plasma의 source snapshot과 원장 이벤트로
// 변환한다. 이 패키지는 소스 승인 여부를 결정하지 않고, 이미 선택된 URL,
// PDF, 텍스트, 미디어 입력을 저장 계층이 이해하는 artifact/snapshot/event
// 계약으로 조립한다.
package sourceingest
