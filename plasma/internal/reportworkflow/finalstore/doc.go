// Package finalstore는 reportworkflow direct 경로의 최종 durable 저장과 long-form gate 결과
// 채택을 담당하는 마지막 stage다.
//
// one_take와 planned는 각각 고정된 CommitOneTake, CommitPlanned entrypoint로만 저장 정책을
// 선택한다. long-form V1/V2/V3 gate는 이미 저장된 canonical artifact/event를
// reporting replay 계약으로 다시 읽어 검증·채택하며 terminal event를 새로 쓰지 않는다.
package finalstore
