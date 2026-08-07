(function reportsExports(root) {
  "use strict";
  const reports = root.Plasma.reports;
  const state = root.Plasma.state;
  const captureMissionSelection = root.Plasma.mission.captureMissionSelection;
  const ownsMissionSelection = root.Plasma.mission.ownsMissionSelection;
  const isStaleMissionOperation = root.Plasma.mission.isStaleMissionOperation;
  const api = root.Plasma.transport.api;
  const missionApi = root.Plasma.transport.missionApi;
  const missionFetch = root.Plasma.transport.missionFetch;
  const $ = root.Plasma.dom.$;
  const reloadMission = (...args) => reports.call("reloadMission", ...args);
  const showError = (...args) => reports.call("showError", ...args);
async function exportReport(versionID, target, options = {}) {
  const owner = captureMissionSelection();
  const key = `version:${versionID}`;
  if (!options.download) reports.setReportPreviewLoading(key);
  try {
    const result = await api(`/api/report_versions/${versionID}/export`, {
      method: "POST",
      body: { target }
    });
    if (!ownsMissionSelection(owner)) return;
    reports.assertReportExportMatches(versionID, target, result);
    const content = result.content || JSON.stringify(result, null, 2);
    if (options.download) {
      reports.downloadReportExport(result, target, content);
    } else {
      reports.applyReportPreview(key, target === "markdown" ? "markdown" : "text", reports.reportExportPreviewHeader(versionID, target, result), content, {
        tableOfContents: target === "markdown"
      });
    }
    await reloadMission(owner.missionId);
  } catch (err) {
    if (!ownsMissionSelection(owner)) return;
    if (!options.download && state.reportPreview && state.reportPreview.key === key) reports.clearReportPreview();
    showError(err);
  }
}

async function viewReportArtifact(artifactID) {
  if (!state.missionId || !artifactID) return;
  const owner = captureMissionSelection();
  const key = `artifact:${artifactID}`;
  reports.setReportPreviewLoading(key);
  try {
    const result = await missionApi(owner, `/artifacts/${artifactID}`);
    const content = result.content || "";
    reports.applyReportPreview(key, "markdown", reports.reportArtifactPreviewHeader(artifactID, result), content, {
      sourceArtifactID: artifactID,
      tableOfContents: true
    });
  } catch (err) {
    if (isStaleMissionOperation(err) || !ownsMissionSelection(owner)) return;
    if (state.reportPreview && state.reportPreview.key === key) reports.clearReportPreview();
    showError(err);
  }
}

async function viewReportRedpenWorkcopy(sourceArtifactID) {
  if (!state.missionId || !sourceArtifactID) return;
  const owner = captureMissionSelection();
  const key = `artifact:${sourceArtifactID}`;
  reports.setReportPreviewLoading(key);
  try {
    const result = await missionApi(owner, `/artifacts/${sourceArtifactID}/redpen`);
    if (!result.exists) throw new Error("저장된 빨간펜 작업본이 없습니다.");
    const revision = Number(result.workcopy?.revision || 0);
    reports.applyReportPreview(key, "markdown", `빨간펜 작업본${revision ? ` v${revision}` : ""}`, result.content || "", {
      sourceArtifactID,
      tableOfContents: true
    });
  } catch (err) {
    if (isStaleMissionOperation(err) || !ownsMissionSelection(owner)) return;
    reports.clearReportPreview();
    showError(err);
  }
}

async function downloadReportArtifact(artifactID) {
  if (!state.missionId || !artifactID) return;
  const owner = captureMissionSelection();
  try {
    const response = await missionFetch(owner, `/artifacts/${artifactID}/download`, {
      headers: { "Accept": "text/markdown, text/plain, */*" }
    });
    if (!response.ok) {
      throw await reports.responseError(response);
    }
    const blob = await response.blob();
    const filename = reports.filenameFromContentDisposition(response.headers.get("Content-Disposition")) || `${artifactID}.md`;
    reports.downloadBlob(blob, filename);
  } catch (err) {
    if (!isStaleMissionOperation(err) && ownsMissionSelection(owner)) showError(err);
  }
}

async function downloadReportRedpenWorkcopy(sourceArtifactID) {
  if (!state.missionId || !sourceArtifactID) return;
  const owner = captureMissionSelection();
  try {
    const response = await missionFetch(owner, `/artifacts/${sourceArtifactID}/redpen/download`, {
      headers: { "Accept": "text/markdown, text/plain, */*" }
    });
    if (!response.ok) throw await reports.responseError(response);
    const blob = await response.blob();
    const filename = reports.filenameFromContentDisposition(response.headers.get("Content-Disposition")) || `${sourceArtifactID}-redpen.md`;
    reports.downloadBlob(blob, filename);
  } catch (err) {
    if (!isStaleMissionOperation(err) && ownsMissionSelection(owner)) showError(err);
  }
}

async function createConversationExport() {
  if (!state.missionId) return;
  const owner = captureMissionSelection();
  reports.setReportNotice("대화내역 export를 생성하는 중입니다.");
  try {
    const result = await missionApi(owner, "/conversation_exports", {
      method: "POST",
      body: { title: "대화내역 export" }
    });
    const artifact = result.artifact || {};
    const artifactID = artifact.artifact_id || artifact.ArtifactID || "";
    await reloadMission();
    reports.setReportNotice("대화내역 export를 생성했습니다.");
    if (artifactID) {
      reports.applyReportPreview(`conversation:${artifactID}`, "markdown", reports.conversationExportPreviewHeader(artifactID, result), result.content || "");
    }
  } catch (err) {
    if (isStaleMissionOperation(err) || !ownsMissionSelection(owner)) return;
    reports.setReportNotice(err.userMessage || err.message || String(err), "error");
    showError(err);
  }
}

async function viewConversationExport(artifactID) {
  if (!state.missionId || !artifactID) return;
  const owner = captureMissionSelection();
  const key = `conversation:${artifactID}`;
  reports.setReportPreviewLoading(key);
  try {
    const result = await missionApi(owner, `/artifacts/${artifactID}`);
    reports.applyReportPreview(key, "markdown", reports.conversationExportPreviewHeader(artifactID, result), result.content || "");
  } catch (err) {
    if (isStaleMissionOperation(err) || !ownsMissionSelection(owner)) return;
    if (state.reportPreview && state.reportPreview.key === key) reports.clearReportPreview();
    showError(err);
  }
}


  Object.assign(reports, { exportReport, viewReportArtifact, viewReportRedpenWorkcopy, downloadReportArtifact, downloadReportRedpenWorkcopy, createConversationExport, viewConversationExport });
})(window);
