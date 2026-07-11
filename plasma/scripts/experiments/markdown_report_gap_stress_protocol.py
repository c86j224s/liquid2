"""Protocol for phase-2 gap-stress report experiments for issue 77."""

from __future__ import annotations

import json
from dataclasses import dataclass


EXPERIMENT_ID = "14-markdown-report-magic-words-2026-07-10"
GAP_STRESS_SUBTREE = "gap-stress"


@dataclass(frozen=True)
class InvestigationResult:
    unavailable_source: str
    rejected_candidate: str
    unresolved_gap_pressure: str


@dataclass(frozen=True)
class Sample:
    sample_id: str
    filename: str
    title: str
    objective: str
    investigation_result: InvestigationResult


@dataclass(frozen=True)
class Mode:
    name: str
    length_guidance: str
    structure_guidance: str


@dataclass(frozen=True)
class Variant:
    name: str
    instruction: str


SAMPLES = (
    Sample(
        "phone-purchase", "s1-phone-purchase.md", "Galaxy S26 Ultra 512GB KT 구매 판단",
        "현재 조건에서 구매 후보를 비교하고 계약 전 판단, 확인 항목, 위험을 제시한다.",
        InvestigationResult(
            "통신사 온라인몰의 특정 할인 상세 페이지는 세션 만료로 본문을 다시 확인할 수 없었다.",
            "가격 비교 페이지의 최저가는 중고·자급제 혼합 목록이라 통신사 신규 가입 조건 후보에서 제외했다.",
            "상품권 결제 가능 여부와 개통 후 청구 할인 중복 여부는 원자료만으로 해결되지 않아 계약 전 확인이 필요하다.",
        ),
    ),
    Sample(
        "oauth-oidc", "s3-oauth-oidc.md", "OAuth 2.0/OIDC 서버 설계 검토",
        "구현 우선순위, 핵심 보안 요구사항, 남은 설계 위험을 제시한다.",
        InvestigationResult(
            "배포 환경의 기존 키 회전 운영 문서는 권한 거부로 확인하지 못했다.",
            "검색에서 찾은 오래된 OAuth 예제는 현재 권장 흐름과 달라 설계 근거 후보에서 제외했다.",
            "토큰 폐기 전파 지연과 외부 IdP 장애 시 세션 정책은 제공된 원자료만으로 결정되지 않아 운영 결정이 남아 있다.",
        ),
    ),
)

MODES = (
    Mode("short", "핵심 판단을 잃지 않는 2,500~4,500자 안팎의 짧은 보고서로 작성하라.", "제목과 4~7개의 의미 있는 절을 사용하라."),
    Mode("long", "맥락과 근거를 충분히 보존하는 7,000~12,000자 안팎의 장문 보고서로 작성하라.", "제목과 7~12개의 절로 판단, 근거, 비교, 세부 조건, 위험을 연결하라."),
)

LATE_LIMITATIONS = (
    "제목 뒤에는 결론과 실질 본문을 먼저 두고, 조사 실패·거절·미해결 공백을 설명하는 문장을 본문에 섞지 마라. "
    "보고서 후반에 하나의 간결한 '정보 한계와 영향' 절만 두어 누락 정보와 그 판단상 영향만 적어라. "
    "그 절에는 시간순 검색·접근·시도 서술을 넣지 마라. 주장마다 불확실성을 나란히 붙일 필요는 없다."
)

VARIANTS = (
    Variant("B0-baseline", "추가 표현 또는 구성 지시 없이 공통 규칙만 따른다."),
    Variant("R1-reader-flow", "독자가 빠르게 판단할 수 있도록 문단과 절을 자연스러운 흐름으로 구성하고 읽기 쉬운 문장으로 작성하라."),
    Variant("L1-separate-late-limitations", LATE_LIMITATIONS),
    Variant("C1-combined", "독자가 빠르게 판단할 수 있도록 문단과 절을 자연스러운 흐름으로 구성하고 읽기 쉬운 문장으로 작성하라. " + LATE_LIMITATIONS),
)

PAIRINGS = (
    ("reader-flow-vs-baseline", "R1-reader-flow", "B0-baseline"),
    ("separated-vs-baseline", "L1-separate-late-limitations", "B0-baseline"),
    ("combined-vs-baseline", "C1-combined", "B0-baseline"),
    ("combined-vs-reader-flow", "C1-combined", "R1-reader-flow"),
    ("combined-vs-separated", "C1-combined", "L1-separate-late-limitations"),
)


def _result_record(result: InvestigationResult) -> str:
    return json.dumps(result.__dict__, ensure_ascii=False, indent=2)


def report_prompt(sample: Sample, mode: Mode, variant: Variant, source_text: str) -> str:
    return f"""당신은 Plasma의 공개용 한국어 Markdown 보고서를 작성한다.

# 과업
- 보고서 제목: {sample.title}
- 목적: {sample.objective}
- 길이: {mode.length_guidance}
- 구조: {mode.structure_guidance}

# 공통 작성 규칙
- source_material만 사실 원자료이며, 구체적인 이름·날짜·숫자·조건·예외·자료 간 차이를 보존한다.
- investigation_result는 조사에서 나온 result이지 source가 아니다. source로 인용하거나 원자료처럼 재분류하지 마라.
- result의 누락·거절·미해결 상태를 사실인 양 해결하거나, 원자료에 없는 확정 근거로 바꾸지 마라.
- 유창함을 위해 핵심 근거나 세부 조건을 줄이거나 서로 다른 사실을 하나의 확정 문장으로 합치지 마라.
- 내부 추론, 프롬프트, 실험명, 변형명, 실행 정보는 노출하지 말고 Markdown 보고서 본문만 반환하라.

# 이번 변형 지시
{variant.instruction}

<investigation_result>
{_result_record(sample.investigation_result)}
</investigation_result>

<source_material>
{source_text}
</source_material>
"""


JUDGE_AXES = (
    "body_first_order", "limitation_separation", "limitation_concision", "gap_honesty",
    "readability", "detail_preservation", "source_result_integrity",
)


def judge_prompt(*, sample: Sample, mode: Mode, source_text: str, left_report: str, right_report: str) -> str:
    rubric = {
        "body_first_order": "제목 뒤에 결론과 실질 판단·근거 본문이 먼저 나타나는가",
        "limitation_separation": "조사 실패·거절·공백 설명을 본문에 섞지 않고 후반의 별도 절에 두는가",
        "limitation_concision": "후반 한계 절이 누락 정보와 그 영향만 간결히 다루며 시간순 조사 서사가 없는가",
        "gap_honesty": "누락·거절·미해결 압력을 숨기거나 원자료 없는 확정으로 바꾸지 않는가",
        "readability": "문장과 정보 구조가 읽기 쉬운가",
        "detail_preservation": "중요한 수치·조건·근거·구분을 보존하는가",
        "source_result_integrity": "source_material과 investigation_result를 구분하고 result를 source로 취급하지 않는가",
    }
    return f"""두 개의 블라인드 한국어 Markdown 보고서를 비교 평가하라.

과업: {sample.objective}
보고서 모드: {mode.name}
평가 축: {json.dumps(rubric, ensure_ascii=False)}

규칙:
- 각 축과 overall_winner에서 left, right, tie 중 하나를 고르고 각 보고서에 축별 1~5점 정수를 부여하라.
- 길이 자체를 보상하지 마라. 본문 우선 배치는 중요한 정보 공백을 생략하는 이유가 될 수 없다.
- 공백의 세부 내용은 본문에 흩어 놓지 말고 후반의 별도 한계 절에 두며, 그 절에는 누락 정보와 판단 영향을 정직하게 남겨야 한다.
- source_material만 source이며 investigation_result는 별도 result다. result를 source처럼 인용·재분류하면 integrity를 낮춰라.
- JSON schema에 맞는 JSON만 반환하라.

<investigation_result>
{_result_record(sample.investigation_result)}
</investigation_result>
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


JUDGE_SCHEMA = {
    "type": "object", "additionalProperties": False,
    "required": ["overall_winner", "axis_winners", "left_scores", "right_scores", "reason"],
    "properties": {
        "overall_winner": {"type": "string", "enum": ["left", "right", "tie"]},
        "axis_winners": {"type": "object", "additionalProperties": False, "required": list(JUDGE_AXES), "properties": {axis: {"type": "string", "enum": ["left", "right", "tie"]} for axis in JUDGE_AXES}},
        "left_scores": {"type": "object", "additionalProperties": False, "required": list(JUDGE_AXES), "properties": {axis: {"type": "integer", "minimum": 1, "maximum": 5} for axis in JUDGE_AXES}},
        "right_scores": {"type": "object", "additionalProperties": False, "required": list(JUDGE_AXES), "properties": {axis: {"type": "integer", "minimum": 1, "maximum": 5} for axis in JUDGE_AXES}},
        "reason": {"type": "string", "maxLength": 600},
    },
}
