// Package readeredit는 장문 final edit reader stage의 typed 계약을 소유한다.
//
// V1과 V2/V3 prompt 차이는 canonical plan event의 final_edit_pipeline에서만 읽는다.
// 이 패키지는 다른 stage를 import하지 않고, root가 넘긴 source artifact와 session
// fork 입력으로 durable replay와 provider 실행을 수행한다.
package readeredit
