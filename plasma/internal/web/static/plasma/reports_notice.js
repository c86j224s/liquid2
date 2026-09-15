(function reportsNotice(root) {
  "use strict";
  const reports = root.Plasma.reports;
  const state = root.Plasma.state;
  const $ = root.Plasma.dom.$;
  const timeShort = root.Plasma.dom.timeShort;
  const setReportBusy = (...args) => reports.setReportBusy(...args);
  const shouldHideDraftPendingNotice = (...args) => reports.shouldHideDraftPendingNotice(...args);
  const reportGenerationGuidanceLabel = (...args) => reports.reportGenerationGuidanceLabel(...args);
  const REPORT_RIGOR_LABELS = reports.REPORT_RIGOR_LABELS;
  const REPORT_MODE_LABELS = reports.REPORT_MODE_LABELS;
  const REPORT_EXECUTION_STRATEGY_LABELS = reports.REPORT_EXECUTION_STRATEGY_LABELS;
  function setReportNotice(text, kind) {
    const el = $("reportNotice");
    if (!el) return;
    const message = String(text || "").trim();
    if (!message) {
      el.classList.add("hidden");
      el.classList.remove("error");
      el.textContent = "";
      return;
    }
    el.textContent = message;
    el.classList.remove("hidden");
    el.classList.toggle("error", kind === "error");
  }

function renderReportDraftStatus(status, wasPending) {
  if (status.state === "pending") {
    setReportBusy(true);
    if (shouldHideDraftPendingNotice(status)) {
      setReportNotice("");
      return;
    }
    setReportNotice(reportPendingMessage(status.event));
    return;
  }
  setReportBusy(false);
  if (status.state === "failed") {
    const payload = status.event?.Payload || {};
    if (status.event?.EventType === "report.humanize.failed") {
      const prefix = payload.canceled === true
        ? "H5 말투 보정이 취소되었습니다."
        : "H5 말투 보정이 완료되지 않았습니다.";
      const preserved = payload.preserved_original_markdown === true
        ? "\n\n원본 Markdown 리포트는 유지되었습니다."
        : "";
      setReportNotice(`${prefix}${reportTimingDetails(status.event)}\n\n${payload.text || payload.error || "원본 리포트를 유지합니다."}${preserved}`, payload.canceled === true ? undefined : "error");
    } else if (status.event?.EventType === "report.patch.failed") {
      const prefix = payload.canceled === true
        ? "리포트 MCP 패치가 취소되었습니다."
        : "리포트 MCP 패치가 완료되지 않았습니다.";
      setReportNotice(`${prefix}${reportTimingDetails(status.event)}\n\n${payload.error || payload.text || "패치 실패 사유 없음"}\n\n원본 Markdown 리포트는 유지되었습니다.`, payload.canceled === true ? undefined : "error");
    } else if (payload.canceled === true) {
      setReportNotice(`리포트 생성이 취소되었습니다.${reportTimingDetails(status.event)}\n\n${payload.text || "사용자가 리포트 생성을 취소했습니다."}`);
    } else if (String(payload.pipeline_family || eventByID(payload.pending_event_id)?.Payload?.pipeline_family || "").trim() === reports.REPORT_UNVERIFIED_PIPELINE_FAMILY) {
      setReportNotice(`무검증 보고서 생성 실패${reportTimingDetails(status.event)}\n\n${payload.safe_error_message || payload.error || payload.text || "모델 호출 또는 결과 저장이 완료되지 않았습니다."}\n\n원고 교정과 자동 재시도 없이 종료되었습니다.`, "error");
    } else if (payload.failed_stage_kind && (reports.REPORT_IL_STAGE_LABELS || {})[payload.failed_stage_kind]) {
      const stage = reports.REPORT_IL_STAGE_LABELS[payload.failed_stage_kind];
      setReportNotice(`IL 보고서 · ${stage} 단계 실패${reportTimingDetails(status.event)}\n\n${payload.safe_error_message || payload.text || "리포트 생성 단계가 실패했습니다."}\n\n다른 생성 방식으로 바꾸거나 자동 재시도하지 않았습니다.`, "error");
    } else {
      setReportNotice(`리포트 초안 생성 실패${reportTimingDetails(status.event)}\n\n${payload.error || payload.text || "실패 사유 없음"}`, "error");
    }
    return;
  }
  if (status.state === "skipped" && wasPending) {
    const payload = status.event?.Payload || {};
    setReportNotice(`H5 말투 보정 결과가 원본과 같아 별도 artifact를 만들지 않았습니다.${reportTimingDetails(status.event)}\n\n${payload.text || "원본 Markdown 리포트는 유지되었습니다."}`);
    return;
  }
  if (status.state === "completed" && wasPending) {
    // New artifact is now the newest card — select it so its preview opens.
    state.selectedReportKey = "";
    state.reportPreview = null;
    const article = status.event?.Payload?.output_kind === "article" || status.event?.Payload?.kind === "article_artifact";
    setReportNotice(article
      ? `글 생성이 완료되었습니다.${reportTimingDetails(status.event)}\n\n최신 글 카드에서 읽어보세요.`
      : `Markdown 리포트 artifact 생성이 완료되었습니다.${reportTimingDetails(status.event)}\n\n최신 리포트 카드에서 미리보기를 확인하세요.`);
  }
}

function reportPendingMessage(event) {
  const payload = event?.Payload || {};
  if (event?.EventType === "report.humanize.pending") {
    const title = payload.title ? `\n대상: ${payload.title}` : "";
    const eventID = event.EventID ? `\n대기 이벤트: ${event.EventID}` : "";
    return [
      "H5 말투 보정 Markdown artifact를 생성하는 중입니다.",
      "원본 Markdown 리포트는 이미 저장되어 있으며, 보정이 실패하거나 취소되어도 원본은 유지됩니다."
    ].join("\n") + title + reportTimingDetails(event) + eventID;
  }
  if (event?.EventType === "report.patch.pending") {
    const title = payload.title ? `\n대상: ${payload.title}` : "";
    const base = payload.base_artifact_id ? `\n기준 artifact: ${payload.base_artifact_id}` : "";
    const eventID = event.EventID ? `\n대기 이벤트: ${event.EventID}` : "";
    return [
      "MCP 패치 방식으로 리포트 새 버전을 만드는 중입니다.",
      "보고서 본문은 프롬프트에 통째로 넣지 않고, 보고서 세션에서 MCP 도구로 필요한 범위만 읽고 수정합니다.",
      "완료되면 아래 리포트 목록에 새 Markdown artifact가 추가됩니다."
    ].join("\n") + title + base + reportTimingDetails(event) + eventID;
  }
  const title = payload.title ? `\n대상: ${payload.title}` : "";
  if (payload.output_kind === "article") {
    const intent = payload.article_intent || {};
    const eventID = event?.EventID ? `\n대기 이벤트: ${event.EventID}` : "";
    const longForm = payload.report_mode === "long_form";
    return [
      longForm ? "단편 책자 분량의 장문 글을 만드는 중입니다." : "독자를 위한 글을 만드는 중입니다.",
      `독자: ${intent.audience || "지정 없음"}`,
      `읽고 나서 얻을 것: ${intent.reader_promise || "지정 없음"}`,
      `특히 살릴 내용: ${intent.emphasis || "지정 없음"}`,
      longForm ? "기존 장문 IL 단계와 전체 원고 편집·Markdown·HTML·PDF 저장을 재사용합니다." : "기존 소스 읽기·실행·저장 경로를 사용해 한 명의 작성자가 전체 글을 씁니다."
    ].join("\n") + title + reportTimingDetails(event) + eventID;
  }
  const experimental = payload.pipeline_family === reports.REPORT_IL_PIPELINE_FAMILY;
  const unverified = payload.pipeline_family === reports.REPORT_UNVERIFIED_PIPELINE_FAMILY;
  if (experimental) {
    const direction = String(payload.direction_hint || "").trim();
    const longForm = payload.report_mode === "long_form";
    const profile = String(payload.rigor_level || "strict").trim();
    const profileLine = profile === "unverified"
      ? "무검증: 자료를 읽고 원고를 한 번 작성하며, 별도 근거 교정 단계는 생략합니다."
      : profile === "exploratory"
        ? "탐색형: 자료 관계를 정리해 원고를 작성하고 독자 관점의 경량 편집을 거칩니다."
        : "검증형: 자료 관계·근거를 확인하고 독자 편집과 최종 관계 교정을 거칩니다.";
    const eventID = event?.EventID ? `\n대기 이벤트: ${event.EventID}` : "";
    return [
      longForm ? "장문 IL 보고서 생성 요청을 보냈습니다." : "IL 보고서 생성 요청을 보냈습니다.",
      longForm
        ? "장문 작성: 하나의 긴 원고를 완성하고 전체를 다시 읽은 뒤 같은 원고를 모든 형식으로 투영합니다."
        : "일반 작성: 하나의 완성 원고를 작성한 뒤 같은 원고를 모든 형식으로 투영합니다.",
      profileLine,
      "선택한 검증 방식과 관계없이 같은 원고로 Markdown·HTML·PDF와 관련 이미지를 함께 만듭니다.",
      `모델: ${String(payload.agent_model || "Codex 기본값")} · 추론: ${String(payload.agent_reasoning_effort || "모델 기본값")} · 새 세션`,
      "완료되면 출력 형식별 보기와 받기가 하나의 보고서 카드에 표시됩니다."
    ].join("\n") + title + `\n방향: ${direction || "지정 없음"}` + reportTimingDetails(event) + eventID;
  }
  if (unverified) {
    const direction = String(payload.direction_hint || "").trim();
    const eventID = event?.EventID ? `\n대기 이벤트: ${event.EventID}` : "";
    return [
      "무검증 보고서 생성 요청을 보냈습니다.",
      "주제·방향과 연결 자료를 바탕으로 모델이 Markdown을 한 번 작성합니다.",
      "근거 확인과 별도 원고 교정 없이 반환된 내용을 그대로 저장합니다.",
      "모델: gpt-5.6-luna · 추론: xhigh · 새 세션",
      "완료되면 Markdown 보고서가 아래 목록에 표시됩니다."
    ].join("\n") + title + `\n방향: ${direction || "지정 없음"}` + reportTimingDetails(event) + eventID;
  }
  const rigor = payload.rigor_label || REPORT_RIGOR_LABELS[payload.rigor_level] || "";
  const rigorLine = rigor ? `\n엄격도: ${rigor}` : "";
  const mode = payload.report_mode || "planned";
  const modeLabel = payload.report_mode_label || REPORT_MODE_LABELS[mode] || "보고서";
  const modeLine = `\n방식: ${modeLabel}`;
  const strategy = String(payload.execution_strategy || "serial").trim() || "serial";
  const strategyLine = mode === "long_form" ? `\n장문 작성: ${REPORT_EXECUTION_STRATEGY_LABELS[strategy] || strategy}` : "";
  const guidance = String(payload.generation_guidance_profile || "g2").trim() || "g2";
  const guidanceLine = `\n글쓰기: ${reportGenerationGuidanceLabel(guidance)}`;
  const model = String(payload.agent_model || "").trim();
  const effort = String(payload.agent_reasoning_effort || "").trim();
  const modelLine = `\n모델: ${model || "미션 설정 상속"}`;
  const effortLine = `\n추론: ${effort || (model ? "모델 기본값" : "미션 설정 상속")}`;
  const direction = String(payload.direction_hint || "").trim();
  const directionLine = `\n방향: ${direction || "지정 없음"}`;
  const workLine = mode === "long_form"
    ? "에이전트가 계획을 만든 뒤 섹션별로 작성하고, 섹션 본문을 보존한 채 파트와 최종 Markdown artifact를 조립하는 중입니다."
    : "에이전트가 생성 계획을 만든 뒤 MCP 읽기 도구로 필요한 소스를 확인하고 Markdown artifact를 작성하는 중입니다.";
  const eventID = event?.EventID ? `\n대기 이벤트: ${event.EventID}` : "";
  const timing = reportTimingDetails(event);
  return [
    "리포트 초안 생성 요청을 보냈습니다.",
    workLine,
    "완료되면 아래 리포트 목록에 새 artifact 기록이 추가됩니다."
  ].join("\n") + title + modeLine + strategyLine + guidanceLine + rigorLine + modelLine + effortLine + directionLine + timing + eventID;
}

function reportTimingDetails(event) {
  if (!event) return "";
  const payload = event.Payload || {};
  const pendingID = payload.pending_event_id || payload.generation?.pending_event_id || "";
  const pendingEvent = pendingID ? eventByID(pendingID) : null;
  const lines = [];
  if (pendingEvent?.CreatedAt) {
    lines.push(`시작: ${timeShort(pendingEvent.CreatedAt)}`);
  } else if (event.EventType === "report.draft.pending" || event.EventType === "report.design.pending" || event.EventType === "report.humanize.pending" || event.EventType === "report.patch.pending") {
    lines.push(`시작: ${timeShort(event.CreatedAt)}`);
  }
  if (event.EventType !== "report.draft.pending" && event.EventType !== "report.design.pending" && event.EventType !== "report.humanize.pending" && event.EventType !== "report.patch.pending" && event.CreatedAt) {
    lines.push(`종료: ${timeShort(event.CreatedAt)}`);
  }
  const durationMS = Number(payload.duration_ms || payload.generation?.duration_ms || 0);
  if (durationMS > 0) lines.push(`소요: ${durationLabel(durationMS)}`);
  return lines.length ? `\n${lines.join("\n")}` : "";
}

function eventByID(eventID) {
  if (!eventID) return null;
  return (state.detail?.events || []).find((event) => event.EventID === eventID) || null;
}

function durationLabel(ms) {
  const seconds = Math.max(0, Math.round(Number(ms || 0) / 1000));
  if (seconds < 60) return `${seconds}초`;
  const minutes = Math.floor(seconds / 60);
  const rest = seconds % 60;
  if (minutes < 60) return rest ? `${minutes}분 ${rest}초` : `${minutes}분`;
  const hours = Math.floor(minutes / 60);
  const minuteRest = minutes % 60;
  return minuteRest ? `${hours}시간 ${minuteRest}분` : `${hours}시간`;
}


  Object.assign(reports, { setReportNotice, renderReportDraftStatus, reportPendingMessage, reportTimingDetails, eventByID, durationLabel });
})(window);
