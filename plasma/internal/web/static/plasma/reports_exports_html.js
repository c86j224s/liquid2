(function reportsExports(root) {
  "use strict";
  const reports = root.Plasma.reports;
	  const state = root.Plasma.state;
	  const captureMissionSelection = root.Plasma.mission.captureMissionSelection;
	  const ownsMissionSelection = root.Plasma.mission.ownsMissionSelection;
	  const isStaleMissionOperation = root.Plasma.mission.isStaleMissionOperation;
	  const missionApi = root.Plasma.transport.missionApi;
	  const $ = root.Plasma.dom.$;
  const reloadMission = (...args) => reports.call("reloadMission", ...args);
  const showError = (...args) => reports.call("showError", ...args);
async function exportReportArtifactHTML(artifactID, options = {}) {
  if (!state.missionId || !artifactID) return;
  const owner = captureMissionSelection();
  const key = `artifact:${artifactID}`;
  let previewWindow = null;
  if (!options.download) {
    state.selectedReportKey = key;
    reports.renderReportsFromState();
    previewWindow = openReportHTMLPreviewWindow();
    reports.setReportNotice("기본 HTML을 새 탭에서 여는 중입니다.");
  }
  try {
    const result = await missionApi(owner, `/artifacts/${artifactID}/html_export`, {
      method: "POST",
      body: { include_content: Boolean(options.download) }
    });
    const content = result.content || "";
    const artifact = result.artifact || {};
    if (options.download) {
      const filename = artifact.filename || artifact.Filename || `${artifactID}.html`;
      const mediaType = artifact.media_type || artifact.MediaType || "text/html;charset=utf-8";
      reports.downloadContent(filename, mediaType, content);
    } else {
      const htmlArtifactID = artifact.artifact_id || artifact.ArtifactID || "";
      const previewURL = result.preview_url || (htmlArtifactID ? reports.call("missionArtifactPreviewURL", owner.missionId, htmlArtifactID) : "");
      if (!previewURL || !navigateReportHTMLPreviewWindow(previewWindow, previewURL)) {
        reports.setReportNotice("새 탭을 열 수 없습니다. 브라우저의 팝업 차단 설정을 확인해 주세요.", "error");
      } else {
        reports.setReportNotice("기본 HTML을 새 탭에서 열었습니다.");
      }
    }
    await reloadMission();
  } catch (err) {
    if (!options.download) closeReportHTMLPreviewWindow(previewWindow);
    if (isStaleMissionOperation(err) || !ownsMissionSelection(owner)) return;
    showError(err);
  }
}

function openReportHTMLPreviewWindow() {
  const preview = window.open("", "_blank");
  if (!preview) return null;
  try {
    preview.opener = null;
    preview.document.title = "기본 HTML 준비 중";
    preview.document.body.innerHTML = `<main style="font-family:system-ui,sans-serif;max-width:560px;margin:20vh auto;padding:24px;color:#1f2937"><h1 style="font-size:18px;margin:0 0 8px">기본 HTML을 준비 중입니다</h1><p style="margin:0;color:#64748b;line-height:1.6">저장된 HTML artifact를 새 탭에서 여는 중입니다.</p></main>`;
  } catch (_err) {
    // Some browsers restrict writing to a newly opened tab. Navigation below can still work.
  }
  return preview;
}

function navigateReportHTMLPreviewWindow(previewWindow, previewURL) {
  if (!previewURL) return false;
  if (previewWindow && !previewWindow.closed) {
    previewWindow.location.href = previewURL;
    return true;
  }
  return Boolean(window.open(previewURL, "_blank", "noopener"));
}

function closeReportHTMLPreviewWindow(previewWindow) {
  try {
    if (previewWindow && !previewWindow.closed) previewWindow.close();
  } catch (_err) {
    // Best-effort cleanup for a failed export placeholder tab.
  }
}

async function exportReportArtifactDesignedHTML(artifactID, options = {}) {
  if (!state.missionId || !artifactID) return;
  const owner = captureMissionSelection();
  const key = `artifact:${artifactID}`;
  if (!options.download) reports.setReportPreviewLoading(key);
  try {
    const result = await missionApi(owner, `/artifacts/${artifactID}/designed_html_export`, {
      method: "POST",
      body: { agent_executor: $("agentExecutor")?.value || "codex" }
    });
    if (result.status === "pending") {
      reports.setReportNotice("디자인 HTML 리포트를 생성 중입니다. 새로고침해도 진행 상태는 이 미션 장부에서 복구됩니다.");
      if (!options.download && state.reportPreview && state.reportPreview.key === key) reports.clearReportPreview();
      await reloadMission();
      return;
    }
    const content = result.content || "";
    const artifact = result.artifact || {};
    if (options.download) {
      const filename = artifact.filename || artifact.Filename || `${artifactID}-designed.html`;
      const mediaType = artifact.media_type || artifact.MediaType || "text/html;charset=utf-8";
      reports.downloadContent(filename, mediaType, content);
    } else {
      reports.applyReportPreview(key, "html", reports.reportArtifactDesignedHTMLPreviewHeader(artifactID, result), content);
    }
    await reloadMission();
  } catch (err) {
    if (isStaleMissionOperation(err) || !ownsMissionSelection(owner)) return;
    if (!options.download && state.reportPreview && state.reportPreview.key === key) reports.clearReportPreview();
    showError(err);
  }
}

async function exportReportArtifactHumanizedMarkdown(artifactID) {
  if (!state.missionId || !artifactID) return;
  if (state.reportPending) return;
  const owner = captureMissionSelection();
  const key = `artifact:${artifactID}`;
  reports.setReportBusy(true);
  reports.setReportNotice("H5 말투 보정 Markdown artifact를 생성하는 중입니다. 원본 Markdown 리포트는 그대로 유지됩니다.");
  if (state.reportPreview && state.reportPreview.key === key) reports.clearReportPreview();
  try {
    const result = await missionApi(owner, `/artifacts/${artifactID}/humanized_markdown_export`, {
      method: "POST",
      body: { mcp_mode: $("mcpMode")?.value || "auto" }
    });
    if (!ownsMissionSelection(owner)) return;
    reports.setReportNotice(result.pending_event
      ? reports.reportPendingMessage(result.pending_event)
      : "H5 말투 보정 Markdown artifact를 생성하는 중입니다.");
    await reloadMission(owner.missionId);
  } catch (err) {
    if (!ownsMissionSelection(owner)) return;
    reports.setReportNotice(`H5 말투 보정 시작 실패\n\n${err.userMessage || err.message || String(err)}`, "error");
    reports.setReportBusy(false);
    showError(err);
  }
}


  Object.assign(reports, { exportReportArtifactHTML, openReportHTMLPreviewWindow, navigateReportHTMLPreviewWindow, closeReportHTMLPreviewWindow, exportReportArtifactDesignedHTML, exportReportArtifactHumanizedMarkdown });
})(window);
