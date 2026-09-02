(function reportsConstants(root) {
  "use strict";
  const reports = root.Plasma.reports;
const REPORT_RIGOR_LABELS = {
  unverified: "무검증",
  exploratory: "탐색형",
  // Deprecated for new UI selection; retained for stored events.
  balanced: "균형형",
  strict: "검증형"
};

const REPORT_MODE_LABELS = {
  one_take: "원테이크 보고서",
  planned: "보고서",
  long_form: "장문 보고서"
};

const REPORT_EXECUTION_STRATEGY_LABELS = {
  serial: "순차",
  section_fanout: "빠른 병렬"
};

const REPORT_IL_PIPELINE_FAMILY = "report_il_experimental";
const REPORT_UNVERIFIED_PIPELINE_FAMILY = "report_unverified";
const REPORT_IL_ARTIFACT_LABELS = {
  narrative: "원고 설계",
  semantic_il: "문서 구조",
  flow_attestation: "흐름 확인 기록",
  markdown: "Markdown",
  html: "HTML",
  pdf: "PDF",
  image: "보고서 이미지",
  manifest: "생성 기록"
};
const REPORT_IL_STAGE_LABELS = {
  source_packet: "승인 소스 준비",
  il_source_selection: "관련 소스 선별",
  il_editorial_memory: "자료 관계·맥락 정리",
  il_narrative: "전체 원고 작성",
  il_long_form_plan: "장문 구성 설계",
  il_long_form_sections: "장문 섹션 작성",
  il_long_form_parts: "장문 파트 편집",
  il_long_form_final: "장문 전체 원고 편집",
  il_continuity: "자료 관계·맥락 보존 편집",
  il_reader: "독자 관점 최종 편집",
  il_images: "소스 이미지 선택·배치",
  il_document: "출력 구조 컴파일",
  il_flow: "독자 관점 최종 편집 (이전)",
  il_render: "Markdown·HTML·PDF 렌더",
  il_store: "결과물 저장"
};

const DEFAULT_REPORT_GENERATION_GUIDANCE = "narrative-contract";
const DEFAULT_LONG_FORM_REPORT_GENERATION_GUIDANCE = "section-brief-cluster-memory-narrative-contract";
// Labels include active UI choices plus legacy profiles that may appear in
// historical report events. Do not infer that every label is a current selector
// option.
const REPORT_GENERATION_GUIDANCE_LABELS = {
  "narrative-contract": "시각자료 계획",
  "part-connective-economy-voice": "시각자료 계획",
  "visual-plan": "시각자료 계획 (이전)",
  "visual-supplement": "시각자료 보조",
  "part-assembly-edit-tools": "파트 조립 다듬기",
  g2: "기본 글쓰기",
  "section-brief": "섹션 중심 (이전)",
  "section-brief-cluster-memory": "섹션 중심 + 풍부하게 (이전)",
  "section-brief-visual-plan": "섹션 중심 (이전)",
  "section-brief-cluster-memory-visual-plan": "섹션 중심 + 풍부하게 (이전)",
  "section-brief-narrative-contract": "섹션 중심",
  "section-brief-cluster-memory-narrative-contract": "섹션 중심 + 풍부하게",
  none: "없음"
};

function reportGenerationGuidanceLabel(value) {
  const normalized = String(value || DEFAULT_REPORT_GENERATION_GUIDANCE).trim() || DEFAULT_REPORT_GENERATION_GUIDANCE;
  return REPORT_GENERATION_GUIDANCE_LABELS[normalized] || normalized;
}

function selectedReportGenerationGuidance(reportMode) {
  return reportMode === "long_form"
    ? DEFAULT_LONG_FORM_REPORT_GENERATION_GUIDANCE
    : DEFAULT_REPORT_GENERATION_GUIDANCE;
}


const DESIGNED_REPORT_RENDERER_VERSION = "dh31-source-markdown-visuals-20260721";

  Object.assign(reports, {
    REPORT_RIGOR_LABELS, REPORT_MODE_LABELS, REPORT_EXECUTION_STRATEGY_LABELS,
    REPORT_IL_PIPELINE_FAMILY, REPORT_UNVERIFIED_PIPELINE_FAMILY,
    REPORT_IL_ARTIFACT_LABELS, REPORT_IL_STAGE_LABELS,
    DEFAULT_REPORT_GENERATION_GUIDANCE, DEFAULT_LONG_FORM_REPORT_GENERATION_GUIDANCE,
    reportGenerationGuidanceLabel, selectedReportGenerationGuidance, DESIGNED_REPORT_RENDERER_VERSION
  });
})(window);
