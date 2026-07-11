# Plasma 문서 안내

이 디렉터리는 Plasma를 이해하고 작업하기 위한 문서 입구입니다. Plasma는 제품이면서,
아키텍처이면서, 운영 표면이면서, 여러 실험을 거쳐 방향을 다듬어 온 연구 프로젝트입니다.

앞으로 문서는 다음 원칙으로 관리합니다.

- 기준 문서는 영어로 작성합니다.
- 같은 의미를 담은 한국어 문서는 같은 위치에 `*.ko.md` 이름으로 둡니다.
- `C1`, `G2`, `H5` 같은 실험 코드명은 숨기지 않고, 처음 읽는 사람이 이해할 수 있게 설명합니다.
- 현재 제품 규칙, 과거 설계, 실험 기록, 미래 아이디어가 섞이지 않게 구분합니다.

> 정리 중 메모:
> Plasma 문서는 제품을 발견하고 고쳐 가는 과정에서 함께 작성되었습니다. 그래서 일부 오래된 문서는
> 한국어가 먼저 쓰였거나, 제품 결정과 실험 기록이 한 문서 안에 섞여 있습니다. 이것을 최종 문서
> 구조로 보지 마세요. #67에서는 현재 문서 구조와 한영 동기화 원칙을 정리했습니다. 이후의 세부 정리는
> 구체적인 범위가 생길 때 별도 이슈로 다룹니다.

## 처음 읽을 문서

Plasma를 처음 이해하려면 아래 순서로 읽는 것이 좋습니다.

1. [Plasma README](../README.md) /
   [Plasma README 한국어](../README.ko.md) - 제품 개요와 개발 명령을 설명합니다.
2. [Glossary](glossary.md) / [용어집](glossary.ko.md) - 제품 용어와 실험 코드명을 설명합니다.
3. [C1 Default Loop](c1-default-loop.md) /
   [C1 기본 루프](c1-default-loop.ko.md) - 현재 기본 제품 흐름을 설명합니다.
4. [Product Architecture](product-architecture.md) /
   [제품 아키텍처](product-architecture.ko.md) - 유지해야 할 제품 경계와 백엔드 경계를 설명합니다.

## 현재 제품 규칙

현재 Plasma가 따라야 하는 동작은 주로 아래 문서에 정리되어 있습니다.

- [C1 Default Loop](c1-default-loop.md)
- [Product Architecture](product-architecture.md)
- [Product Flow](product-flow.md) - 제품 흐름의 변천을 보존하는 한국어 중심 연결 문서입니다. 별도 정리
  범위가 생기면 새 이슈에서 다룹니다.
- [Automatic Investigation](automatic-investigation.md) - 현재 흐름과 legacy 설명을 함께 보존하는 한국어 중심
  연결 문서입니다. 별도 정리 범위가 생기면 새 이슈에서 다룹니다.
- [Media And Document Source Implementation Design](media-source-implementation-design.md)
- [Token Diet Instrumentation](token-diet-instrumentation.md)

## Source와 Connector

소스 등록과 외부 원천 연동은 아래 문서에서 다룹니다.

- [Confluence Source Integration](confluence-source-integration.md)
- [Confluence Live Validation Checklist](confluence-live-validation-checklist.md)
- [Media And Document Source Implementation Design](media-source-implementation-design.md)

용어는 다음처럼 구분합니다.

- connector는 외부 원천에 접근하는 어댑터입니다.
- source는 미션에 붙은 연구 재료입니다.
- raw artifact는 저장된 본문, 추출 텍스트, 파일 메타데이터 같은 저장물입니다.
- source snapshot은 사용자가 승인한 미션 단위 source 기록입니다.

## Legacy와 Future Design Note

아래 문서는 배경을 이해하는 데 유용하지만, 현재 기본 제품 흐름을 설명하는 문서는 아닙니다.

- [Legacy Ledger Loop](legacy-ledger-loop.md)
- [Evidence Signal Model](evidence-signal-model.md) /
  [Evidence Signal Model 한국어](evidence-signal-model.ko.md)

현재 C1 기본 루프는 evidence와 claim record를 만들지 않습니다. 나중에 이 계층을 되살리더라도,
source 탐색, 인용, 불확실성 추적, 추적성을 돕는 참고 계층이어야 합니다. 조사나 보고서 생성을
허가하거나 막는 gate가 되어서는 안 됩니다.

## 실험 기록

실험 요약은 [experiments/](experiments/README.md)에 둡니다. 이 디렉터리에는 사람이 읽을 수 있는
protocol, decision memo, 작게 정리한 metric만 둡니다. raw run payload, 스크린샷, 생성 HTML,
비공개 corpus는 저장소 밖 artifact archive 정책에 따라 관리합니다.

중요한 코드명 계열은 다음과 같습니다.

- `C1`: 현재 기본 제품 루프
- `C0`, `PAL2`, `NAV`: controller strategy 실험
- `G2`, `H5`: 보고서 말투와 humanization 실험
- `DH23`: designed HTML 보고서 렌더링 실험
- `C4`: 장문 보고서 조립 실험

제품 문서에서 이 코드명을 사용할 때는 먼저 [용어집](glossary.ko.md)을 확인하세요.

## 운영 문서

런타임과 로컬 산출물 경계는 아래 문서에 있습니다.

- [Plasma Artifact Archive](artifact-archive.md)
- 저장소 루트의 [Configuration](../../docs/configuration.md)
- `plasma/README.md`의 개발 명령

## 문서 관리 규칙

- 조밀한 설계 설명 앞에는 독자가 먼저 방향을 잡을 수 있는 짧은 소개를 둡니다.
- 현재 동작, historical note, future idea는 시각적으로 구분합니다.
- 파일을 이동하면 이 디렉터리와 `plasma/README.md`의 링크를 함께 고칩니다.
- 실험 코드명을 쓰면 그 자리에서 설명하거나 glossary로 연결합니다.
- 한국어 대응 문서는 영어 기준 문서와 같은 의미를 담아야 합니다. 단순 요약본으로 만들지 않습니다.
