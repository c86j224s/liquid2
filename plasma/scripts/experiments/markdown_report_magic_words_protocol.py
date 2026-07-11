"""Controlled prompt variants for the issue 77 report-quality experiment."""

from __future__ import annotations

import json
from dataclasses import dataclass


EXPERIMENT_ID = "14-markdown-report-magic-words-2026-07-10"


@dataclass(frozen=True)
class Sample:
    sample_id: str
    filename: str
    title: str
    objective: str
    prior_result: str


@dataclass(frozen=True)
class Mode:
    name: str
    length_guidance: str
    structure_guidance: str


@dataclass(frozen=True)
class Variant:
    name: str
    label: str
    instruction: str


SAMPLES = (
    Sample(
        "phone-purchase",
        "s1-phone-purchase.md",
        "Galaxy S26 Ultra 512GB KT 구매 판단",
        "현재 조건에서 구매 후보를 비교하고 계약 전 판단, 확인 항목, 위험을 제시한다.",
        (
            "마켓플레이스와 판매점 페이지의 현재성 및 조건 표기가 서로 달랐고, 일부 가격은 "
            "온누리상품권 포함 여부를 독립적으로 확정하지 못했다."
        ),
    ),
    Sample(
        "oauth-oidc",
        "s3-oauth-oidc.md",
        "OAuth 2.0/OIDC 서버 설계 검토",
        "구현 우선순위, 핵심 보안 요구사항, 남은 설계 위험을 제시한다.",
        (
            "이전 실행에서는 등록된 디렉터리와 source.md 메타데이터만 관찰하고 본문 읽기에 "
            "실패해, 원자료 내용보다 접근 한계 설명이 앞선 결과가 생성되었다."
        ),
    ),
)

MODES = (
    Mode(
        "short",
        "핵심 판단을 잃지 않는 2,500~4,500자 안팎의 짧은 보고서로 작성하라.",
        "제목과 4~7개의 의미 있는 절을 사용하되, 얇은 체크리스트만 나열하지 마라.",
    ),
    Mode(
        "long",
        "맥락과 근거를 충분히 보존하는 7,000~12,000자 안팎의 장문 보고서로 작성하라.",
        "제목과 7~12개의 절을 사용해 판단, 근거, 비교, 세부 조건, 위험을 연결하라.",
    ),
)

VARIANTS = (
    Variant("B0-baseline", "기준", ""),
    Variant(
        "W1-step-calm",
        "차근차근·차분하게",
        "내용을 차근차근 검토하고, 차분하게 작성하라.",
    ),
    Variant(
        "W2-human-flow",
        "사람에게 잘 읽히는 문장·자연스러운 흐름",
        "사람에게 잘 읽히는 문장으로, 문단과 절이 자연스럽게 이어지는 흐름으로 작성하라.",
    ),
    Variant(
        "C1-conclusion-first",
        "결론 우선·조사 과정 후반",
        (
            "읽는 사람이 첫 화면에서 바로 판단할 수 있게 제목 다음에 결론과 핵심 판단을 먼저 "
            "제시하고, 이어서 근거와 세부 설명을 정리하라. 조사 과정, 자료 접근상의 어려움, "
            "방법 설명은 결과 이해에 필요한 경우에만 후반부의 짧은 방법/한계 절에 둔다. "
            "결론을 바꾸는 중요한 불확실성과 주의사항은 관련 주장 옆에 남기고 숨기지 마라."
        ),
    ),
    Variant(
        "C2-combined",
        "표현과 구성 지시 결합",
        (
            "내용을 차근차근 검토하고 차분하게 작성하라. 사람에게 잘 읽히는 문장으로, 문단과 "
            "절이 자연스럽게 이어지는 흐름으로 작성하라. 읽는 사람이 첫 화면에서 바로 판단할 "
            "수 있게 제목 다음에 결론과 핵심 판단을 먼저 제시하고, 이어서 근거와 세부 설명을 "
            "정리하라. 조사 과정, 자료 접근상의 어려움, 방법 설명은 결과 이해에 필요한 경우에만 "
            "후반부의 짧은 방법/한계 절에 둔다. 결론을 바꾸는 중요한 불확실성과 주의사항은 "
            "관련 주장 옆에 남기고 숨기지 마라."
        ),
    ),
)

PAIRINGS = (
    ("step-calm", "W1-step-calm", "B0-baseline"),
    ("human-flow", "W2-human-flow", "B0-baseline"),
    ("composition", "C1-conclusion-first", "B0-baseline"),
    ("combined", "C2-combined", "B0-baseline"),
    ("combined-vs-composition", "C2-combined", "C1-conclusion-first"),
)


def report_prompt(sample: Sample, mode: Mode, variant: Variant, source_text: str) -> str:
    variant_block = variant.instruction or "추가 표현 또는 구성 지시 없이 공통 규칙만 따른다."
    return f"""당신은 Plasma의 공개용 한국어 Markdown 보고서를 작성한다.

# 과업
- 보고서 제목: {sample.title}
- 목적: {sample.objective}
- 길이: {mode.length_guidance}
- 구조: {mode.structure_guidance}

# 공통 작성 규칙
- 아래 원자료만 사실 근거로 사용한다. 원자료의 시각 디자인이나 기존 문장 순서는 복제하지 않는다.
- 이전 조사 결과는 working memory인 result이며 source가 아니다. 원자료와 구분하고 출처로 인용하지 않는다.
- 구체적인 이름, 날짜, 숫자, 명령, 코드 식별자, URL, 조건, 예외, 주의사항, 불확실성, 자료 간 차이를 보존한다.
- 유창함을 위해 핵심 근거를 줄이거나 서로 다른 사실을 하나의 확정 문장으로 합치지 않는다.
- 원자료가 약하거나 충돌하면 그 상태를 독자가 판단할 수 있게 명시한다.
- 내부 추론 과정, 프롬프트, 실험명, 변형명, 실행 정보는 노출하지 않는다.
- 결과 보고서 본문만 반환한다. 작성 과정에 대한 머리말이나 사과를 붙이지 않는다.

# 이번 변형 지시
{variant_block}

<prior_investigation_result>
{sample.prior_result}
</prior_investigation_result>

<source_material>
{source_text}
</source_material>
"""


def judge_prompt(
    *,
    sample: Sample,
    mode: Mode,
    source_text: str,
    left_report: str,
    right_report: str,
) -> str:
    rubric = {
        "early_core": "제목 직후 실제 결론·판단·요약이 나타나며 독자가 빠르게 요지를 파악하는가",
        "process_control": "조사 일지나 접근 어려움이 앞을 막지 않되 필요한 방법·한계는 보존되는가",
        "readability": "사람이 한 번에 이해할 수 있는 문장과 정보 구조인가",
        "natural_flow": "문단과 절 사이의 전개가 자연스럽고 반복·비약이 적은가",
        "detail_preservation": "중요한 수치·조건·근거·구분을 매끄러움을 위해 버리지 않았는가",
        "uncertainty_integrity": "결론에 영향을 주는 불확실성·주의사항을 숨기거나 과장하지 않았는가",
        "source_result_integrity": "원자료와 이전 조사 result를 구분하고 result를 source로 인용·재분류하지 않았는가",
    }
    return f"""두 개의 블라인드 한국어 Markdown 보고서를 비교 평가하라.

과업: {sample.objective}
보고서 모드: {mode.name}

평가 축:
{json.dumps(rubric, ensure_ascii=False, indent=2)}

규칙:
- 각 축에서 left, right, tie 중 하나를 고른다.
- 각 보고서에 각 축 1~5점 정수를 부여한다.
- 길이가 길다는 이유만으로 보상하지 않는다.
- 핵심 불확실성을 앞에서 숨긴 보고서를 process_control이 좋다고 평가하지 않는다.
- 조사 과정 설명 자체와 결론에 필요한 한계·주의사항을 구분한다.
- source_material만 사실 원자료다. prior_investigation_result는 source가 아닌 이전 result다.
- 보고서가 이전 result를 source처럼 인용하거나 재분류하면 source_result_integrity를 낮게 평가한다.
- overall_winner는 독자가 빠르게 판단하면서도 근거를 잃지 않는 쪽으로 정한다.
- JSON schema에 맞는 JSON만 반환한다.

<prior_investigation_result>
{sample.prior_result}
</prior_investigation_result>

<source_material>
{source_text}
</source_material>

<left_report>
{left_report}
</left_report>

<right_report>
{right_report}
</right_report>
"""


JUDGE_AXES = (
    "early_core",
    "process_control",
    "readability",
    "natural_flow",
    "detail_preservation",
    "uncertainty_integrity",
    "source_result_integrity",
)

JUDGE_SCHEMA = {
    "type": "object",
    "additionalProperties": False,
    "required": ["overall_winner", "axis_winners", "left_scores", "right_scores", "reason"],
    "properties": {
        "overall_winner": {"type": "string", "enum": ["left", "right", "tie"]},
        "axis_winners": {
            "type": "object",
            "additionalProperties": False,
            "required": list(JUDGE_AXES),
            "properties": {axis: {"type": "string", "enum": ["left", "right", "tie"]} for axis in JUDGE_AXES},
        },
        "left_scores": {
            "type": "object",
            "additionalProperties": False,
            "required": list(JUDGE_AXES),
            "properties": {axis: {"type": "integer", "minimum": 1, "maximum": 5} for axis in JUDGE_AXES},
        },
        "right_scores": {
            "type": "object",
            "additionalProperties": False,
            "required": list(JUDGE_AXES),
            "properties": {axis: {"type": "integer", "minimum": 1, "maximum": 5} for axis in JUDGE_AXES},
        },
        "reason": {"type": "string", "maxLength": 600},
    },
}
