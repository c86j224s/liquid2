// Package evidencecheck는 장문 final edit terminal gate stage의 typed 계약을 소유한다.
//
// V3 evidence gate와 V1/V2 corrective gate는 모두 canonical finalization을 쓰는
// terminal gate다. 어떤 gate를 실행할지는 root graph가 FinalTail에서만 결정하며,
// 이 패키지는 prompt, tool allowlist, durable replay/resume 계약을 보존한다.
package evidencecheck
