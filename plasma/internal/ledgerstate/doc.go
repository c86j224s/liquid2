// Package ledgerstate는 원장 이벤트 배열만으로 현재 열려 있는 작업과
// 완료된 작업을 판정한다. 이 패키지는 이벤트를 저장하거나 보정하지
// 않는 읽기 전용 투영 계층이며, 알 수 없는 payload는 상태 판정 불가로
// 취급한다.
package ledgerstate
