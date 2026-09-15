# Article Real Pilot Contract

Issue: #461

상태: provider 실행 전 계약 validator 구현. 실제 source fixture, provider adapter, 실제 Article은 아직 없다.

## 목적

이 wave는 synthetic preflight와 실제 Article 생성 사이의 실행 금지 경계를 코드로 만든다. `plasma-article-real-pilot-validate`는 frozen protocol bundle만 검증하며 provider, product DB, ledger, API, UI를 호출하지 않는다.

## 검증하는 계약

- 정확히 M1/M2/M3 × R/E/A의 9개 사전 배정 run
- repository-owner Issue #461 digest lock 요구
- Codex `gpt-5.6-luna` `xhigh`, fresh ephemeral session, ignored user config, exact MCP replacement
- ambient Web/file access 금지
- 0700 전용 archive, 0600 파일, single writer, 격리 HOME/provider config/workspace/DB
- raw provider log와 provider session 보존 금지
- fixture별 7,000–11,000자 body band, 8–12 material claims, 2–3 caveats, 최소 1개 supported connection
- source catalog, dossier, claim inventory, memory question, transfer task exact digest
- E/A의 provider, stage sequence, common prompts/tools/schema/renderer, budget, artifact allowlist 일치
- E/A treatment contract는 서로 다르고 A만 `article_narrative`
- cell/matrix provider-call, duration, source-byte, token budget과 unknown usage fail-closed
- A/E primary blind lane, R contextual lane, generation order, 3개 anonymous pair, mapping commitment, rubric/shell/administration digest
- user judgment와 factual audit lock 이전 mapping reveal 금지
- semantic attempt 1회, replacement run/matrix resume 금지, unusable cell은 INCONCLUSIVE, protocol change는 INVALID

R은 기존 Report product topology를 보존하므로 E/A stage list와 동일하다고 주장하지 않는다. E/A만 treatment 외 실행 계약이 byte-level로 matched되어야 한다.

## 아직 실행하지 않는 것

- 공개 source 수집과 canonical snapshot
- source-grounded dossier/truth ledger/span receipt 생성
- provider invocation
- author/reader/auditor/repair execution
- Article IL, Markdown, HTML, PDF 생성
- blind packet 생성과 독서 평가

이 command의 성공은 `bundle_validated=true`, `issue_lock_verified=false`, `provider_ready=false`만 의미한다. 완전한 fixture/arm bundle의 exact protocol/fixture/arm digest를 Issue #461에 사전 게시하고, 별도의 live Issue-lock verifier가 owner, comment URL/ID, 생성 시각, canonical body hash, digest를 검증해야 실제 generation을 시작할 수 있다.
