(function reportsConstants(root) {
  "use strict";
  const reports = root.Plasma.reports;
  const $ = root.Plasma.dom.$;
const REPORT_RIGOR_LABELS = {
  exploratory: "탐색적",
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

const DEFAULT_REPORT_GENERATION_GUIDANCE = "narrative-contract";
const DEFAULT_LONG_FORM_REPORT_GENERATION_GUIDANCE = "part-connective-economy-voice";
// This includes the active long-form default plus older profiles retained for
// stored events and API compatibility. Only index.html controls visible choices.
const LONG_FORM_ONLY_REPORT_GENERATION_GUIDANCE = new Set([
  "part-connective-economy-voice",
  "part-assembly-edit-tools",
  "section-brief",
  "section-brief-cluster-memory",
  "section-brief-visual-plan",
  "section-brief-cluster-memory-visual-plan",
  "section-brief-narrative-contract",
  "section-brief-cluster-memory-narrative-contract"
]);

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
  const modeDefault = reportMode === "long_form"
    ? DEFAULT_LONG_FORM_REPORT_GENERATION_GUIDANCE
    : DEFAULT_REPORT_GENERATION_GUIDANCE;
  const selected = String($("reportGenerationGuidance")?.value || modeDefault).trim() || modeDefault;
  if (reportMode !== "long_form" && LONG_FORM_ONLY_REPORT_GENERATION_GUIDANCE.has(selected)) {
    return DEFAULT_REPORT_GENERATION_GUIDANCE;
  }
  return selected;
}


const DESIGNED_REPORT_RENDERER_VERSION = "dh31-source-markdown-visuals-20260721";

  Object.assign(reports, {
    REPORT_RIGOR_LABELS, REPORT_MODE_LABELS, REPORT_EXECUTION_STRATEGY_LABELS,
    DEFAULT_REPORT_GENERATION_GUIDANCE, DEFAULT_LONG_FORM_REPORT_GENERATION_GUIDANCE,
    reportGenerationGuidanceLabel, selectedReportGenerationGuidance, DESIGNED_REPORT_RENDERER_VERSION
  });
})(window);
