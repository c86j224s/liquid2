// Package reportassembly는 final edit pipeline이 읽을 deterministic 장문 조립 source를
// typed stage로 보장한다.
//
// V2/V3 writer source는 reporting.EnsureFinalEditAssembly의 durable replay 계약을
// 감싸고, V1 reader source는 기존 deterministic ID만 typed output으로 고정한다.
// provider 실행, prompt, final tail 선택은 소유하지 않는다.
package reportassembly
