# Media Inspect 실험 결정 메모

작성일: 2026-06-26

## 목적

Plasma가 이미지, 문서 스캔, 그래프 같은 미디어 소스를 다룰 때 기본 읽기 경로를 어떻게 잡을지 검증했다.

핵심 질문은 다음이었다.

- 미디어 소스는 metadata-only로 충분한가?
- 이미지나 문서 스캔을 명시적으로 inspect하는 도구가 필요한가?
- inspect가 필요하다면 항상 실행해야 하는가, 아니면 agent가 필요하다고 판단할 때만 실행해야 하는가?

## 실험 구조

실험은 Codex image attachment를 미래의 `plasma.media.inspect` 도구 대역으로 사용했다. 이 실험은 실제 제품 MCP 도구 구현을 검증한 것이 아니라, 미디어 inspect 표면이 연구 품질과 source/result 경계에 어떤 영향을 주는지 확인하기 위한 것이다.

비교군은 다음과 같다.

- `M0`: metadata-only. source page, metadata, manifest는 읽지만 이미지 픽셀은 보지 못한다.
- `M1`: always inspect. 처음부터 이미지 attachment를 받는다.
- `M2`: document scan OCR/vision inspect. 문서 이미지 스캔을 attachment로 받는다.
- `M1C`: conditional image inspect. 먼저 metadata만 읽고, 필요하다고 판단하면 inspect를 요청한다.
- `M2C`: conditional document OCR/vision inspect. 먼저 metadata와 전사 텍스트를 읽고, 필요하면 문서 scan inspect를 요청한다.

## 코퍼스

사용한 코퍼스는 다음과 같다.

- `C1`: 오다 노부나가 지도/초상 이미지
- `C2`: 오다 노부나가 문서 스캔과 전사 텍스트
- `C3`: p95 latency 선 그래프
- `C4`: error bar가 있는 성능 막대그래프
- `C5`: 처리량과 오류율을 함께 보여주는 이중축 그래프
- `C6`: y축이 잘린 pass-rate 막대그래프

PDF는 포함하지 않았다. PDF는 text PDF, scan PDF, mixed PDF, 논문/보고서 PDF에 따라 필요한 추출 도구가 다르므로 별도 document-source 실험으로 다루는 것이 맞다.

## 실행 결과

최종 분석에는 report run 90개와 judge 비교 72개가 포함된다.

그래프 판단에서 metadata-only는 탈락했다.

- `C3 M0 vs M1`: M1이 6/6 승리, sign-test p=0.0312
- `C3 M0 vs M1C`: M1C가 6/6 승리, sign-test p=0.0312

추가 그래프 코퍼스에서 조건부 inspect는 안정적으로 inspect 필요성을 감지했다.

- `C4-C6 M1C`: 18/18 inspect 요청
- `C3-C6 M1C`: 24/24 inspect 요청
- hard flag 없음
- source ID 누락 없음

Always inspect와 conditional inspect 비교는 다음과 같다.

- `C3 M1 vs M1C`: 3:3
- `C4 M1 vs M1C`: M1C 4/6
- `C5 M1 vs M1C`: M1C 6/6, sign-test p=0.0312
- `C6 M1 vs M1C`: M1C 4/6
- `C4-C6` 합산: M1C 14/18, sign-test p=0.0309
- `C3-C6` 합산: M1C 17/24, sign-test p=0.0639

## 판단

Plasma의 기본 미디어 읽기 경로는 metadata-first가 맞다.

다만 사용자의 질문이 이미지나 그래프의 시각 구조에 의존한다면, agent가 명시적으로 inspect를 요청할 수 있어야 한다. 그래프 벤치마크처럼 trend, crossing point, error bar, dual-axis tradeoff, truncated-axis caution을 판단해야 하는 경우 metadata-only는 충분하지 않다.

Always inspect를 기본값으로 둘 필요는 아직 입증되지 않았다. 조건부 inspect는 그래프 계열에서 inspect 필요성을 놓치지 않았고, judge 기준으로도 추가 그래프 코퍼스에서는 always inspect보다 우세했다. 따라서 제품 기본값은 다음 흐름이 적합하다.

1. 먼저 source metadata, source page, manifest를 읽는다.
2. 현재 질문이 픽셀, scan, chart shape, visual trend, error bar, axis relationship에 의존하는지 판단한다.
3. 필요할 때만 inspect를 요청한다.
4. 원본 이미지, 문서 스캔, 그래프 파일은 source로 유지한다.
5. inspect 관찰은 source가 아니라 tool-produced result로 저장한다.
6. 저장 지식과 보고서에는 관찰 한계와 불확실성을 함께 남긴다.

## 제품화 주의점

조건부 inspect 자체는 유망하지만, 프롬프트와 도구 응답 문구가 정확해야 한다.

실험 중 `M1C` 최종 프롬프트가 `M1`만큼 source ID 인용을 강하게 요구하지 않는 비대칭을 발견했다. 이 비대칭 때문에 일부 조건부 결과에서 source/result 경계가 약해질 수 있었다. 프롬프트를 수정한 뒤에는 `C3-C6`의 M1C 24개 run 모두 source ID를 포함했다.

제품화 시에는 다음 문구와 구조를 고정해야 한다.

- 이미지는 source다.
- inspect 관찰은 result다.
- 보고서가 inspect 관찰을 사용할 때는 원본 source ID와 inspect result를 함께 연결해야 한다.
- 정확한 수치, 통계적 유의성, error bar 의미, axis scale이 source에 없으면 추정으로 남겨야 한다.
- 그래프가 잘린 축이나 이중축을 사용하면 보고서가 시각적 인상을 과장하지 않도록 해야 한다.

## 다음 단계

제품 구현 후보는 `metadata-first + conditional inspect`다.

다음 실험은 PDF가 아니라 별도 document-source 실험으로 설계해야 한다. PDF에서는 페이지 텍스트 추출, 페이지 이미지 렌더링, OCR/vision, 표/그림 추출, 논문 구조 추출이 모두 다른 도구 표면을 요구하기 때문이다.
