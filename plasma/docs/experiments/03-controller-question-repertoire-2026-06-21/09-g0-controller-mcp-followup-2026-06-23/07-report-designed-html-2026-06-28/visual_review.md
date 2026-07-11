# Designed HTML Visual Review

This file tracks visual/manual review notes for generated artifacts. Automated hard-fail status is not a quality score.
Screenshot files are raw run artifacts and are archived outside the public
repository under `research-artifacts/`. This public summary keeps run IDs,
status, and review notes, but does not link to those raw screenshots.

| Run | Mission | Variant | Status | Screenshots | Review note |
| --- | --- | --- | --- | --- | --- |
| `CQ1-AUTO-seed-0001-attempt-1__transcript__R7__DH0__rep01` | `CQ1` | `DH0` | `hard_failed` | archived: desktop/mobile | 초기에는 Chrome headless timeout으로 실패 보존됐지만, 새 CDP probe로 재렌더한 뒤 mobile_horizontal_overflow hard-fail로 정리됐다. 시각 판정상 DH0 후보에서 제외한다. |
| `CQ1-AUTO-seed-0001-attempt-1__transcript__R7__DH0__rep02` | `CQ1` | `DH0` | `hard_failed` | archived: desktop/mobile | Desktop은 문서형 브리핑으로 첫 화면과 탭이 명확하다. Mobile에서는 hero 제목과 본문이 가로로 잘려 mobile 품질 실패로 본다. 출처 표지는 텍스트 marker 중심이고 링크는 없다. |
| `CQ1-AUTO-seed-0001-attempt-1__transcript__R7__DH11__rep01` | `CQ1` | `DH11` | `completed` | archived: desktop/mobile | C1 제품 경계 주제를 dark briefing app으로 안정적으로 변환했다. mission -> session -> steering -> source read -> result -> report artifact의 기본 흐름이 첫 화면에서 바로 보이며, legacy 구조물은 보존 대상으로만 배치된다. DH6/DH8보다 보고서 앱답고 mobile도 안정적이다. |
| `CQ1-AUTO-seed-0001-attempt-1__transcript__R7__DH11__rep02` | `CQ1` | `DH11` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ1-AUTO-seed-0001-attempt-1__transcript__R7__DH11__rep03` | `CQ1` | `DH11` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ1-AUTO-seed-0001-attempt-1__transcript__R7__DH12__rep01` | `CQ1` | `DH12` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ1-AUTO-seed-0001-attempt-1__transcript__R7__DH13__rep01` | `CQ1` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ1-AUTO-seed-0001-attempt-1__transcript__R7__DH1__rep01` | `CQ1` | `DH1` | `started` | - | 중단된 partial run을 보존했다. raw HTML이 없어 시각 판정에서 제외한다. |
| `CQ1-AUTO-seed-0001-attempt-1__transcript__R7__DH1__rep02` | `CQ1` | `DH1` | `hard_failed` | archived: desktop/mobile | Desktop은 오른쪽 단계 패널과 카드 구조가 읽기 좋다. Mobile에서는 심한 가로 overflow와 제목/카드 텍스트 잘림이 있어 mobile 품질 실패로 본다. 영어 product term이 많이 남는다. |
| `CQ1-AUTO-seed-0001-attempt-1__transcript__R7__DH1__rep03` | `CQ1` | `DH1` | `hard_failed` | archived: desktop/mobile | 하드 실패 보존: mobile_horizontal_overflow |
| `CQ1-AUTO-seed-0001-attempt-1__transcript__R7__DH2__rep01` | `CQ1` | `DH2` | `hard_failed` | archived: desktop/mobile | Desktop 시각 완성도는 가장 높지만 source_reference_section_missing으로 hard-fail이 맞다. Mobile에서도 제목과 pill/flow 영역이 가로로 잘린다. broader batch 후보에서 보류한다. |
| `CQ1-AUTO-seed-0001-attempt-1__transcript__R7__DH2__rep02` | `CQ1` | `DH2` | `completed` | archived: desktop/mobile | CDP 모바일 검증까지 통과한 첫 DH2 후보. Desktop은 정보 구조와 출처 패널이 가장 뚜렷하고 interactive artifact 방향성이 보인다. 다만 영어 내부 용어가 많고, 대원수가 기대한 google-io-2026.html급 풍부함에는 아직 못 미친다. |
| `CQ1-AUTO-seed-0001-attempt-1__transcript__R7__DH2__rep03` | `CQ1` | `DH2` | `completed` | archived: desktop/mobile | CDP 모바일 검증까지 통과한 두 번째 DH2 후보. Mobile은 실제 emulation 기준으로 잘리지 않고 읽힌다. Desktop은 정돈됐지만 요약 보드에 가까워, 넓은 배치와 Claude upper-bound 비교 전에 더 풍부한 artifact 방향을 검토해야 한다. |
| `CQ1-AUTO-seed-0001-attempt-1__transcript__R7__DH3__rep01` | `CQ1` | `DH3` | `hard_failed` | archived: desktop/mobile | Desktop은 가장 정돈되어 보이나 skeleton 영향이 강하다. Mobile에서는 hero와 카드 텍스트가 가로로 잘린다. 계획상 diagnostic overfitting probe로만 취급한다. |
| `CQ2-AUTO-seed-0002-attempt-1__transcript__R7__DH11__rep01` | `CQ2` | `DH11` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ2-AUTO-seed-0002-attempt-1__transcript__R7__DH11__rep02` | `CQ2` | `DH11` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ2-AUTO-seed-0002-attempt-1__transcript__R7__DH11__rep03` | `CQ2` | `DH11` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ2-AUTO-seed-0002-attempt-1__transcript__R7__DH12__rep01` | `CQ2` | `DH12` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ2-AUTO-seed-0002-attempt-1__transcript__R7__DH13__rep01` | `CQ2` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ2-AUTO-seed-0002-attempt-1__transcript__R7__DH2__rep01` | `CQ2` | `DH2` | `hard_failed` | archived: desktop/mobile | DH2 확장 batch의 첫 실패 사례. Mobile에서 sticky nav가 4px 정도 viewport를 넘는다. 육안상 큰 붕괴는 아니지만, 현재 mobile hard-fail 기준에서는 실패가 맞다. |
| `CQ2-AUTO-seed-0002-attempt-1__transcript__R7__DH4__rep01` | `CQ2` | `DH4` | `completed` | archived: desktop/mobile | Google I/O reference에서 좋았던 '긴 내용을 탐색형 브리핑 앱으로 바꾸는' 방향을 적용한 첫 통과 사례. DH2에서 실패했던 mobile overflow를 해결했고, 첫 화면이 제품 판단 브리핑처럼 강하게 선다. 카드/기준선/비교 보드가 풍부해졌지만, 탭형 장문 탐색성은 아직 reference보다 약하다. |
| `CQ2-AUTO-seed-0002-attempt-1__transcript__R7__DH5__rep01` | `CQ2` | `DH5` | `completed` | archived: desktop/mobile | DH4보다 더 차분하고 예쁜 product dossier 느낌으로 정돈됐다. Mobile도 안정적이다. 다만 DH4의 풍부한 카드/탐색감 일부가 줄어들어, 예쁘지만 살짝 얌전한 브리핑으로 수렴한다. |
| `CQ2-AUTO-seed-0002-attempt-1__transcript__R7__DH6__rep01` | `CQ2` | `DH6` | `completed` | archived: desktop/mobile | DH4의 정보 앱 구조와 DH5의 polish를 가장 잘 절충한 사례. Desktop은 hero, nav, 세 갈래 경로 맵이 선명하고 Mobile도 안정적이다. DH5보다 덜 얌전하고 DH4보다 덜 거칠다. |
| `CQ3-AUTO-seed-0003-attempt-1__transcript__R7__DH11__rep01` | `CQ3` | `DH11` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ3-AUTO-seed-0003-attempt-1__transcript__R7__DH11__rep02` | `CQ3` | `DH11` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ3-AUTO-seed-0003-attempt-1__transcript__R7__DH11__rep03` | `CQ3` | `DH11` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ3-AUTO-seed-0003-attempt-1__transcript__R7__DH12__rep01` | `CQ3` | `DH12` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ3-AUTO-seed-0003-attempt-1__transcript__R7__DH13__rep01` | `CQ3` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ3-AUTO-seed-0003-attempt-1__transcript__R7__DH2__rep01` | `CQ3` | `DH2` | `hard_failed` | archived: desktop/mobile | DH2 확장 batch의 실제 mobile 실패 사례. 좌측 로고/제목 시작점이 잘리고, 탭 버튼 일부가 viewport 밖으로 밀린다. DH2 채택 전 mobile nav 설계를 더 조여야 한다. |
| `CQ4-AUTO-seed-0004-attempt-1__transcript__R7__DH11__rep01` | `CQ4` | `DH11` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ4-AUTO-seed-0004-attempt-1__transcript__R7__DH11__rep02` | `CQ4` | `DH11` | `completed` | archived: desktop/mobile | 브라우저 턴, Codex 재개 세션, MCP 주입의 실제 경계를 다룬 코드/제품 분석 주제에서도 레이아웃이 무너지지 않는다. 첫 화면의 네 경계 지도와 하단 본문 카드가 읽기 쉽다. 같은 템플릿 얼굴은 남지만, 기술 분석 주제에서 내용이 포스터로 얇아지는 문제는 줄었다. |
| `CQ4-AUTO-seed-0004-attempt-1__transcript__R7__DH11__rep03` | `CQ4` | `DH11` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ4-AUTO-seed-0004-attempt-1__transcript__R7__DH12__rep01` | `CQ4` | `DH12` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ4-AUTO-seed-0004-attempt-1__transcript__R7__DH13__rep01` | `CQ4` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ4-AUTO-seed-0004-attempt-1__transcript__R7__DH2__rep01` | `CQ4` | `DH2` | `completed` | archived: desktop/mobile | CDP mobile 검증 통과. Desktop과 mobile 모두 보고서형 artifact로 읽히며, 카드/요약/섹션 흐름이 깨지지 않는다. |
| `CQ4-AUTO-seed-0004-attempt-1__transcript__R7__DH4__rep01` | `CQ4` | `DH4` | `completed` | archived: desktop/mobile | Google I/O식 정보 앱 방향을 코드 경계 분석 주제에 적용한 통과 사례. 첫 화면의 메트릭 카드와 소스 스냅샷 패널이 읽기 좋고 모바일도 안정적이다. 다만 표가 거의 사라지고 editorial poster 느낌이 강해, 다음 실험에서는 표/타임라인/섹션 탭을 더 균형 있게 강제할 필요가 있다. |
| `CQ4-AUTO-seed-0004-attempt-1__transcript__R7__DH5__rep01` | `CQ4` | `DH5` | `completed` | archived: desktop/mobile | 첫 화면 미감은 DH4보다 좋아졌다. 정보 경계 지도도 깔끔하다. 하지만 전체적으로 미니멀한 executive brief에 가까워졌고, Google I/O reference의 정보 앱적 풍성함은 충분히 살아나지 않았다. |
| `CQ4-AUTO-seed-0004-attempt-1__transcript__R7__DH6__rep01` | `CQ4` | `DH6` | `completed` | archived: desktop/mobile | 이번 polish 계열 중 가장 좋은 시각 결과. 첫 화면이 공유 가능한 기술 브리핑처럼 보이고, source-linked 실행 지도와 경계 표가 명확하다. Mobile도 보기 좋다. 다만 더 넓은 주제에서는 탭/섹션 탐색을 더 많이 살려야 한다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH10__rep01` | `CQ5` | `DH10` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH11__rep01` | `CQ5` | `DH11` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH11__rep02` | `CQ5` | `DH11` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH11__rep03` | `CQ5` | `DH11` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH12__rep01` | `CQ5` | `DH12` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH13__rep01` | `CQ5` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH13__rep02` | `CQ5` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH13__rep03` | `CQ5` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH14__rep01` | `CQ5` | `DH14` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH14__rep02` | `CQ5` | `DH14` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH14__rep03` | `CQ5` | `DH14` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH15__rep01` | `CQ5` | `DH15` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH15__rep02` | `CQ5` | `DH15` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH15__rep03` | `CQ5` | `DH15` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH16__rep01` | `CQ5` | `DH16` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH16__rep02` | `CQ5` | `DH16` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH16__rep03` | `CQ5` | `DH16` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH17__rep01` | `CQ5` | `DH17` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH17__rep02` | `CQ5` | `DH17` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH17__rep03` | `CQ5` | `DH17` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH18__rep01` | `CQ5` | `DH18` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH18__rep02` | `CQ5` | `DH18` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH18__rep03` | `CQ5` | `DH18` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH19__rep01` | `CQ5` | `DH19` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH19__rep02` | `CQ5` | `DH19` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH19__rep03` | `CQ5` | `DH19` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH20__rep01` | `CQ5` | `DH20` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH20__rep02` | `CQ5` | `DH20` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH20__rep03` | `CQ5` | `DH20` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH21__rep01` | `CQ5` | `DH21` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH21__rep02` | `CQ5` | `DH21` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH21__rep03` | `CQ5` | `DH21` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH22__rep01` | `CQ5` | `DH22` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH22__rep02` | `CQ5` | `DH22` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH22__rep03` | `CQ5` | `DH22` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH23__rep01` | `CQ5` | `DH23` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH23__rep02` | `CQ5` | `DH23` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH23__rep03` | `CQ5` | `DH23` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH24__rep01` | `CQ5` | `DH24` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH24__rep02` | `CQ5` | `DH24` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH24__rep03` | `CQ5` | `DH24` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH2__rep01` | `CQ5` | `DH2` | `completed` | archived: desktop/mobile | CDP mobile 검증 통과. 선형 문서보다 artifact 느낌이 강하고, mobile에서도 본문과 controls가 안정적으로 유지된다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH6__rep01` | `CQ5` | `DH6` | `completed` | archived: desktop/mobile | Source, evidence, result, report 구분을 시각화한 통과 사례. 구조 자체는 선명하지만 첫 화면이 개념어와 규칙 중심이라, 실제 사용 장면이나 실패 사례가 약해 내용이 직관적으로 들어오지는 않는다. Mobile은 안정적이다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH7__rep01` | `CQ5` | `DH7` | `completed` | archived: desktop/mobile | Content-first 재생성 후 통과. '요약문을 근거로 써도 되는가'라는 판단 장면으로 시작해 DH6보다 바로 이해된다. 다만 DH6보다 짧고 건조해져, 풍성한 리포트 앱보다는 좋은 첫 장면을 가진 설명 자료에 가깝다. Mobile은 안정적이다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH8__rep01` | `CQ5` | `DH8` | `completed` | archived: desktop/mobile | DH8 최종 보정 후 통과. content-first 진입과 근거 오염 경로 판별은 DH6보다 선명하다. 다만 오른쪽 판별 패널은 아직 카드형 체크리스트에 가까워, 엄밀한 의미의 인포그래픽으로는 약하다. Mobile은 안정적이다. |
| `CQ5-AUTO-seed-0005-attempt-1__transcript__R7__DH9__rep01` | `CQ5` | `DH9` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH10__rep01` | `CQ6` | `DH10` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH11__rep01` | `CQ6` | `DH11` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH11__rep02` | `CQ6` | `DH11` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH11__rep03` | `CQ6` | `DH11` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH12__rep01` | `CQ6` | `DH12` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH13__rep01` | `CQ6` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH13__rep02` | `CQ6` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH13__rep03` | `CQ6` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH14__rep01` | `CQ6` | `DH14` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH14__rep02` | `CQ6` | `DH14` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH14__rep03` | `CQ6` | `DH14` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH15__rep01` | `CQ6` | `DH15` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH15__rep02` | `CQ6` | `DH15` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH15__rep03` | `CQ6` | `DH15` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH16__rep01` | `CQ6` | `DH16` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH16__rep02` | `CQ6` | `DH16` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH16__rep03` | `CQ6` | `DH16` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH17__rep01` | `CQ6` | `DH17` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH17__rep02` | `CQ6` | `DH17` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH17__rep03` | `CQ6` | `DH17` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH18__rep01` | `CQ6` | `DH18` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH18__rep02` | `CQ6` | `DH18` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH18__rep03` | `CQ6` | `DH18` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH19__rep01` | `CQ6` | `DH19` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH19__rep02` | `CQ6` | `DH19` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH19__rep03` | `CQ6` | `DH19` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH20__rep01` | `CQ6` | `DH20` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH20__rep02` | `CQ6` | `DH20` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH20__rep03` | `CQ6` | `DH20` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH21__rep01` | `CQ6` | `DH21` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH21__rep02` | `CQ6` | `DH21` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH21__rep03` | `CQ6` | `DH21` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH22__rep01` | `CQ6` | `DH22` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH22__rep02` | `CQ6` | `DH22` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH22__rep03` | `CQ6` | `DH22` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH23__rep01` | `CQ6` | `DH23` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH23__rep02` | `CQ6` | `DH23` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH23__rep03` | `CQ6` | `DH23` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH24__rep01` | `CQ6` | `DH24` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH24__rep02` | `CQ6` | `DH24` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH24__rep03` | `CQ6` | `DH24` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH2__rep01` | `CQ6` | `DH2` | `completed` | archived: desktop/mobile | CDP mobile 검증 통과. 시각 밀도와 정보량은 괜찮지만, 장기적으로는 도표/미디어 활용을 더 풍부하게 만드는 후속 variant가 필요하다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH6__rep01` | `CQ6` | `DH6` | `completed` | archived: desktop/mobile | UI 없는 Research IDE와 MCP-first 흐름을 다룬 통과 사례. 공통 런타임 맵은 유용하지만 전체적으로 Plasma 내부 용어가 많고 첫 화면이 제품 설명서처럼 보여, 외부 독자에게는 진입 장벽이 높다. Mobile은 안정적이다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH7__rep01` | `CQ6` | `DH7` | `completed` | archived: desktop/mobile | Content-first 재생성 후 가장 설득력 있는 사례. '리포트 버튼 뒤에서 Plasma는 무엇을 읽어야 하나'라는 장면이 바로 잡혀서 UI 없는 MCP-first 흐름이 DH6보다 훨씬 빨리 이해된다. Mobile도 안정적이다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH8__rep01` | `CQ6` | `DH8` | `completed` | archived: desktop/mobile | DH8 최종 보정 후 통과. 리포트 생성 경로를 '이전 유혹'과 'MCP-first 제한 읽기'로 나누어 보여주는 비교형 인포그래픽이 의미를 가진다. DH7보다 밀도도 회복됐다. Mobile은 안정적이다. |
| `CQ6-AUTO-seed-0006-attempt-1__transcript__R7__DH9__rep01` | `CQ6` | `DH9` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH10__rep01` | `CQ7` | `DH10` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH11__rep01` | `CQ7` | `DH11` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH11__rep02` | `CQ7` | `DH11` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH11__rep03` | `CQ7` | `DH11` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH12__rep01` | `CQ7` | `DH12` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH13__rep01` | `CQ7` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH13__rep02` | `CQ7` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH13__rep03` | `CQ7` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH14__rep01` | `CQ7` | `DH14` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH14__rep02` | `CQ7` | `DH14` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH14__rep03` | `CQ7` | `DH14` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH15__rep01` | `CQ7` | `DH15` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH15__rep02` | `CQ7` | `DH15` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH15__rep03` | `CQ7` | `DH15` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH16__rep01` | `CQ7` | `DH16` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH16__rep02` | `CQ7` | `DH16` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH16__rep03` | `CQ7` | `DH16` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH17__rep01` | `CQ7` | `DH17` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH17__rep02` | `CQ7` | `DH17` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH17__rep03` | `CQ7` | `DH17` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH18__rep01` | `CQ7` | `DH18` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH18__rep02` | `CQ7` | `DH18` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH18__rep03` | `CQ7` | `DH18` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH19__rep01` | `CQ7` | `DH19` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH19__rep02` | `CQ7` | `DH19` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH19__rep03` | `CQ7` | `DH19` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH20__rep01` | `CQ7` | `DH20` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH20__rep02` | `CQ7` | `DH20` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH20__rep03` | `CQ7` | `DH20` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH21__rep01` | `CQ7` | `DH21` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH21__rep02` | `CQ7` | `DH21` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH21__rep03` | `CQ7` | `DH21` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH22__rep01` | `CQ7` | `DH22` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH22__rep02` | `CQ7` | `DH22` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH22__rep03` | `CQ7` | `DH22` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH23__rep01` | `CQ7` | `DH23` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH23__rep02` | `CQ7` | `DH23` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH23__rep03` | `CQ7` | `DH23` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH24__rep01` | `CQ7` | `DH24` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH24__rep02` | `CQ7` | `DH24` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH24__rep03` | `CQ7` | `DH24` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH2__rep01` | `CQ7` | `DH2` | `completed` | archived: desktop/mobile | 초기에는 a11y skip link 때문에 false-positive overflow로 잡혔지만, probe 보정 후 통과했다. 실제 mobile screenshot은 깨지지 않았다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH6__rep01` | `CQ7` | `DH6` | `completed` | archived: desktop/mobile | 컨트롤러 질문 전략의 안전 경계를 다룬 통과 사례. 정상 흐름과 금지 흐름을 나누는 게이트 도식은 명확하지만, 큰 여백과 추상 표현 때문에 보고서 본문으로 끌고 들어가는 힘은 약하다. Mobile은 안정적이다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH7__rep01` | `CQ7` | `DH7` | `completed` | archived: desktop/mobile | Content-first 재생성 후 통과. 질문 하나가 근거가 되는 순간을 before/after로 보여줘 DH6보다 제품 경계 문제가 잘 들어온다. 다만 dark comparison panel은 강하지만 나머지 섹션으로 이어지는 풍성함은 아직 부족하다. Mobile은 안정적이다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH8__rep01` | `CQ7` | `DH8` | `completed` | archived: desktop/mobile | DH8 최종 보정 후 통과. 안전한 조향과 금지된 주입을 시각적으로 나누는 구조가 DH6보다 이해하기 쉽다. 다만 첫 화면의 큰 제목과 메트릭 카드가 강해서, 인포그래픽 자체의 중심성은 CQ6/CQ8보다 약하다. Mobile은 안정적이다. |
| `CQ7-AUTO-seed-0007-attempt-1__transcript__R7__DH9__rep01` | `CQ7` | `DH9` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH10__rep01` | `CQ8` | `DH10` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH11__rep01` | `CQ8` | `DH11` | `completed` | archived: desktop/mobile | 대화 기반 연구가 길을 잃었을 때의 세 조정 장치를 가장 안정적인 report-app 형태로 보여준다. hero, 핵심 경로 카드, tab rail, 본문 섹션이 이어져 DH8의 인포그래픽 장점과 DH10의 polish를 더 일관된 런타임 구조로 결합했다. Google I/O reference처럼 완전히 손맛 있는 주제별 디자인은 아니지만, 이번 실험에서 처음으로 reference-grade 방향의 재현 가능한 후보가 됐다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH11__rep02` | `CQ8` | `DH11` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH11__rep03` | `CQ8` | `DH11` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH12__rep01` | `CQ8` | `DH12` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH13__rep01` | `CQ8` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH13__rep02` | `CQ8` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH13__rep03` | `CQ8` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH14__rep01` | `CQ8` | `DH14` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH14__rep02` | `CQ8` | `DH14` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH14__rep03` | `CQ8` | `DH14` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH15__rep01` | `CQ8` | `DH15` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH15__rep02` | `CQ8` | `DH15` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH15__rep03` | `CQ8` | `DH15` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH16__rep01` | `CQ8` | `DH16` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH16__rep02` | `CQ8` | `DH16` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH16__rep03` | `CQ8` | `DH16` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH17__rep01` | `CQ8` | `DH17` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH17__rep02` | `CQ8` | `DH17` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH17__rep03` | `CQ8` | `DH17` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH18__rep01` | `CQ8` | `DH18` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH18__rep02` | `CQ8` | `DH18` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH18__rep03` | `CQ8` | `DH18` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH19__rep01` | `CQ8` | `DH19` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH19__rep02` | `CQ8` | `DH19` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH19__rep03` | `CQ8` | `DH19` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH20__rep01` | `CQ8` | `DH20` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH20__rep02` | `CQ8` | `DH20` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH20__rep03` | `CQ8` | `DH20` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH21__rep01` | `CQ8` | `DH21` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH21__rep02` | `CQ8` | `DH21` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH21__rep03` | `CQ8` | `DH21` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH22__rep01` | `CQ8` | `DH22` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH22__rep02` | `CQ8` | `DH22` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH22__rep03` | `CQ8` | `DH22` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH23__rep01` | `CQ8` | `DH23` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH23__rep02` | `CQ8` | `DH23` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH23__rep03` | `CQ8` | `DH23` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH24__rep01` | `CQ8` | `DH24` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH24__rep02` | `CQ8` | `DH24` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH24__rep03` | `CQ8` | `DH24` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH2__rep01` | `CQ8` | `DH2` | `completed` | archived: desktop/mobile | CDP mobile 검증 통과. 확장 batch 기준으로 DH2가 여러 주제에서 재현 가능하다는 근거를 보탠 사례다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH6__rep01` | `CQ8` | `DH6` | `completed` | archived: desktop/mobile | 대화 기반 연구가 길을 잃었을 때의 조정 장치를 다룬 통과 사례. mission recall, user steering, controller steering의 층위는 잘 보이지만, 여전히 실제 대화 장면이나 before/after가 부족해 내용이 사례로 체감되지는 않는다. Mobile은 안정적이다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH7__rep01` | `CQ8` | `DH7` | `completed` | archived: desktop/mobile | Content-first 재생성 후 통과. '불인 소스만 보라'는 제한에서 답변이 이전 요약과 새 검색으로 흐려지는 장면을 잡아 DH6보다 문제의식이 선명하다. 다만 전체 길이와 장문 탐색감은 줄어 DH6급 밀도와 결합할 필요가 있다. Mobile은 안정적이다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH8__rep01` | `CQ8` | `DH8` | `completed` | archived: desktop/mobile | DH8 최종 보정 후 통과. 길 잃음 진단 삼각형과 source/result 경계 붕괴 중심 박스는 관계를 보여주므로 이번 배치에서 가장 인포그래픽에 가깝다. 첫 화면 진입성과 장문 밀도도 DH7보다 낫다. Mobile은 안정적이다. |
| `CQ8-AUTO-seed-0008-attempt-1__transcript__R7__DH9__rep01` | `CQ8` | `DH9` | `hard_failed` | archived: desktop/mobile | 하드 실패 보존: mobile_horizontal_overflow |
| `EXT-2606-s26-ultra-512-kt-comparison__DH12__rep01` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH12` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH13__rep01` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH13__rep02` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH13__rep03` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH14__rep01` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH14` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH14__rep02` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH14` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH14__rep03` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH14` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH15__rep01` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH15` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH15__rep02` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH15` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH15__rep03` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH15` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH16__rep01` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH16` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH16__rep02` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH16` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH16__rep03` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH16` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH17__rep01` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH17` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH17__rep02` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH17` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH17__rep03` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH17` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH18__rep01` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH18` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH18__rep02` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH18` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH18__rep03` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH18` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH19__rep01` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH19` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH19__rep02` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH19` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH19__rep03` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH19` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH20__rep01` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH20` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH20__rep02` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH20` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH20__rep03` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH20` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH21__rep01` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH21` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH21__rep02` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH21` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH21__rep03` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH21` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH22__rep01` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH22` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH22__rep02` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH22` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH22__rep03` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH22` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH23__rep01` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH23` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH23__rep02` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH23` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH23__rep03` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH23` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH24__rep01` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH24` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH24__rep02` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH24` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-2606-s26-ultra-512-kt-comparison__DH24__rep03` | `EXT-2606-s26-ultra-512-kt-comparison` | `DH24` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH12__rep01` | `EXT-oauth-oidc-design` | `DH12` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH13__rep01` | `EXT-oauth-oidc-design` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH13__rep02` | `EXT-oauth-oidc-design` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH13__rep03` | `EXT-oauth-oidc-design` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH14__rep01` | `EXT-oauth-oidc-design` | `DH14` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH14__rep02` | `EXT-oauth-oidc-design` | `DH14` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH14__rep03` | `EXT-oauth-oidc-design` | `DH14` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH15__rep01` | `EXT-oauth-oidc-design` | `DH15` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH15__rep02` | `EXT-oauth-oidc-design` | `DH15` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH15__rep03` | `EXT-oauth-oidc-design` | `DH15` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH16__rep01` | `EXT-oauth-oidc-design` | `DH16` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH16__rep02` | `EXT-oauth-oidc-design` | `DH16` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH16__rep03` | `EXT-oauth-oidc-design` | `DH16` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH17__rep01` | `EXT-oauth-oidc-design` | `DH17` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH17__rep02` | `EXT-oauth-oidc-design` | `DH17` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH17__rep03` | `EXT-oauth-oidc-design` | `DH17` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH18__rep01` | `EXT-oauth-oidc-design` | `DH18` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH18__rep02` | `EXT-oauth-oidc-design` | `DH18` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH18__rep03` | `EXT-oauth-oidc-design` | `DH18` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH19__rep01` | `EXT-oauth-oidc-design` | `DH19` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH19__rep02` | `EXT-oauth-oidc-design` | `DH19` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH19__rep03` | `EXT-oauth-oidc-design` | `DH19` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH20__rep01` | `EXT-oauth-oidc-design` | `DH20` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH20__rep02` | `EXT-oauth-oidc-design` | `DH20` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH20__rep03` | `EXT-oauth-oidc-design` | `DH20` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH21__rep01` | `EXT-oauth-oidc-design` | `DH21` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH21__rep02` | `EXT-oauth-oidc-design` | `DH21` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH21__rep03` | `EXT-oauth-oidc-design` | `DH21` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH22__rep01` | `EXT-oauth-oidc-design` | `DH22` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH22__rep02` | `EXT-oauth-oidc-design` | `DH22` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH22__rep03` | `EXT-oauth-oidc-design` | `DH22` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH23__rep01` | `EXT-oauth-oidc-design` | `DH23` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH23__rep02` | `EXT-oauth-oidc-design` | `DH23` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH23__rep03` | `EXT-oauth-oidc-design` | `DH23` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH24__rep01` | `EXT-oauth-oidc-design` | `DH24` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH24__rep02` | `EXT-oauth-oidc-design` | `DH24` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-oauth-oidc-design__DH24__rep03` | `EXT-oauth-oidc-design` | `DH24` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH12__rep01` | `EXT-ollama-ui-stack-comparison` | `DH12` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH13__rep01` | `EXT-ollama-ui-stack-comparison` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH13__rep02` | `EXT-ollama-ui-stack-comparison` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH13__rep03` | `EXT-ollama-ui-stack-comparison` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH14__rep01` | `EXT-ollama-ui-stack-comparison` | `DH14` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH14__rep02` | `EXT-ollama-ui-stack-comparison` | `DH14` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH14__rep03` | `EXT-ollama-ui-stack-comparison` | `DH14` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH15__rep01` | `EXT-ollama-ui-stack-comparison` | `DH15` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH15__rep02` | `EXT-ollama-ui-stack-comparison` | `DH15` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH15__rep03` | `EXT-ollama-ui-stack-comparison` | `DH15` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH16__rep01` | `EXT-ollama-ui-stack-comparison` | `DH16` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH16__rep02` | `EXT-ollama-ui-stack-comparison` | `DH16` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH16__rep03` | `EXT-ollama-ui-stack-comparison` | `DH16` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH17__rep01` | `EXT-ollama-ui-stack-comparison` | `DH17` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH17__rep02` | `EXT-ollama-ui-stack-comparison` | `DH17` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH17__rep03` | `EXT-ollama-ui-stack-comparison` | `DH17` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH18__rep01` | `EXT-ollama-ui-stack-comparison` | `DH18` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH18__rep02` | `EXT-ollama-ui-stack-comparison` | `DH18` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH18__rep03` | `EXT-ollama-ui-stack-comparison` | `DH18` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH19__rep01` | `EXT-ollama-ui-stack-comparison` | `DH19` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH19__rep02` | `EXT-ollama-ui-stack-comparison` | `DH19` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH19__rep03` | `EXT-ollama-ui-stack-comparison` | `DH19` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH20__rep01` | `EXT-ollama-ui-stack-comparison` | `DH20` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH20__rep02` | `EXT-ollama-ui-stack-comparison` | `DH20` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH20__rep03` | `EXT-ollama-ui-stack-comparison` | `DH20` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH21__rep01` | `EXT-ollama-ui-stack-comparison` | `DH21` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH21__rep02` | `EXT-ollama-ui-stack-comparison` | `DH21` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH21__rep03` | `EXT-ollama-ui-stack-comparison` | `DH21` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH22__rep01` | `EXT-ollama-ui-stack-comparison` | `DH22` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH22__rep02` | `EXT-ollama-ui-stack-comparison` | `DH22` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH22__rep03` | `EXT-ollama-ui-stack-comparison` | `DH22` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH23__rep01` | `EXT-ollama-ui-stack-comparison` | `DH23` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH23__rep02` | `EXT-ollama-ui-stack-comparison` | `DH23` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH23__rep03` | `EXT-ollama-ui-stack-comparison` | `DH23` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH24__rep01` | `EXT-ollama-ui-stack-comparison` | `DH24` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH24__rep02` | `EXT-ollama-ui-stack-comparison` | `DH24` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-ollama-ui-stack-comparison__DH24__rep03` | `EXT-ollama-ui-stack-comparison` | `DH24` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH12__rep01` | `EXT-sengoku-period-causes` | `DH12` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH13__rep01` | `EXT-sengoku-period-causes` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH13__rep02` | `EXT-sengoku-period-causes` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH13__rep03` | `EXT-sengoku-period-causes` | `DH13` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH14__rep01` | `EXT-sengoku-period-causes` | `DH14` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH14__rep02` | `EXT-sengoku-period-causes` | `DH14` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH14__rep03` | `EXT-sengoku-period-causes` | `DH14` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH15__rep01` | `EXT-sengoku-period-causes` | `DH15` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH15__rep02` | `EXT-sengoku-period-causes` | `DH15` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH15__rep03` | `EXT-sengoku-period-causes` | `DH15` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH16__rep01` | `EXT-sengoku-period-causes` | `DH16` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH16__rep02` | `EXT-sengoku-period-causes` | `DH16` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH16__rep03` | `EXT-sengoku-period-causes` | `DH16` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH17__rep01` | `EXT-sengoku-period-causes` | `DH17` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH17__rep02` | `EXT-sengoku-period-causes` | `DH17` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH17__rep03` | `EXT-sengoku-period-causes` | `DH17` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH18__rep01` | `EXT-sengoku-period-causes` | `DH18` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH18__rep02` | `EXT-sengoku-period-causes` | `DH18` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH18__rep03` | `EXT-sengoku-period-causes` | `DH18` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH19__rep01` | `EXT-sengoku-period-causes` | `DH19` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH19__rep02` | `EXT-sengoku-period-causes` | `DH19` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH19__rep03` | `EXT-sengoku-period-causes` | `DH19` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH20__rep01` | `EXT-sengoku-period-causes` | `DH20` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH20__rep02` | `EXT-sengoku-period-causes` | `DH20` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH20__rep03` | `EXT-sengoku-period-causes` | `DH20` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH21__rep01` | `EXT-sengoku-period-causes` | `DH21` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH21__rep02` | `EXT-sengoku-period-causes` | `DH21` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH21__rep03` | `EXT-sengoku-period-causes` | `DH21` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH22__rep01` | `EXT-sengoku-period-causes` | `DH22` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH22__rep02` | `EXT-sengoku-period-causes` | `DH22` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH22__rep03` | `EXT-sengoku-period-causes` | `DH22` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH23__rep01` | `EXT-sengoku-period-causes` | `DH23` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH23__rep02` | `EXT-sengoku-period-causes` | `DH23` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH23__rep03` | `EXT-sengoku-period-causes` | `DH23` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH24__rep01` | `EXT-sengoku-period-causes` | `DH24` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH24__rep02` | `EXT-sengoku-period-causes` | `DH24` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
| `EXT-sengoku-period-causes__DH24__rep03` | `EXT-sengoku-period-causes` | `DH24` | `completed` | archived: desktop/mobile | 자동 렌더링 산출물이 있으며, 수동 선호/품질 판정은 아직 완료하지 않았다. |
