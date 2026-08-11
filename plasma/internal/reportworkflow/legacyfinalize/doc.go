// Package legacyfinalize는 legacy 장문 finalization tail의 typed stage 계약을 소유한다.
//
// 이 패키지는 기존 report.long_form.finalize/final_edit MCP prompt, 도구 allowlist,
// 두 번 시도, REPORT_FINALIZED sentinel, hint recovery, canonical replay를 보존한다.
// H5 humanize 실행과 graph 선택은 root reportworkflow runner가 소유한다.
package legacyfinalize
