(function reportsExports(root) {
  "use strict";
  const reports = root.Plasma.reports;
  const state = root.Plasma.state;
  const captureMissionSelection = root.Plasma.mission.captureMissionSelection;
  const ownsMissionSelection = root.Plasma.mission.ownsMissionSelection;
  const api = root.Plasma.transport.api;
  const $ = root.Plasma.dom.$;
  const showError = (...args) => reports.call("showError", ...args);
async function responseError(response) {
  const text = await response.text();
  let data = {};
  if (text.trim() !== "") {
    try {
      data = JSON.parse(text);
    } catch (err) {
      data = { raw: text };
    }
  }
  const message = data.error?.message || response.statusText || "요청 실패";
  const err = new Error(`HTTP ${response.status}: ${message}`);
  err.userMessage = message;
  err.status = response.status;
  err.details = data;
  return err;
}

function assertReportExportMatches(versionID, target, result) {
  const payload = result.event?.Payload || result.event?.payload || {};
  const returnedVersionID = payload.report_version_id || "";
  const returnedTarget = payload.target || "";
  if (returnedVersionID && returnedVersionID !== versionID) {
    throw new Error(`리포트 export 버전 불일치: 요청 ${versionID}, 응답 ${returnedVersionID}`);
  }
  if (returnedTarget && returnedTarget !== target) {
    throw new Error(`리포트 export 형식 불일치: 요청 ${target}, 응답 ${returnedTarget}`);
  }
}

function reportArtifactPreviewHeader(artifactID, result) {
  const artifact = result.artifact || {};
  const filename = artifact.filename || artifact.Filename || "";
  return [
    `리포트 artifact: ${artifactID}`,
    filename ? `파일명: ${filename}` : "",
    artifact.media_type ? `형식: ${artifact.media_type}` : ""
  ].filter(Boolean).join("\n");
}

function conversationExportPreviewHeader(artifactID, result) {
  const artifact = result.artifact || {};
  const filename = artifact.filename || artifact.Filename || "";
  return [
    `대화내역 export: ${artifactID}`,
    filename ? `파일명: ${filename}` : "",
    artifact.media_type ? `형식: ${artifact.media_type}` : ""
  ].filter(Boolean).join("\n");
}

function reportArtifactHTMLPreviewHeader(artifactID, result) {
  const artifact = result.artifact || {};
  const filename = artifact.filename || artifact.Filename || "";
  return [
    `HTML export: ${artifact.artifact_id || artifact.ArtifactID || ""}`,
    `원본 Markdown: ${artifactID}`,
    filename ? `파일명: ${filename}` : "",
    "self-contained interactive HTML"
  ].filter(Boolean).join("\n");
}

function reportArtifactDesignedHTMLPreviewHeader(artifactID, result) {
  const artifact = result.artifact || {};
  const filename = artifact.filename || artifact.Filename || "";
  return [
    `디자인 HTML: ${artifact.artifact_id || artifact.ArtifactID || ""}`,
    `원본 Markdown: ${artifactID}`,
    filename ? `파일명: ${filename}` : "",
    "self-contained designed interactive HTML"
  ].filter(Boolean).join("\n");
}

function reportExportPreviewHeader(versionID, target, result) {
  const artifact = result.artifact || result.Artifact || {};
  const filename = artifact.Filename || artifact.filename || "";
  return [
    `리포트 버전: ${versionID}`,
    `형식: ${target}`,
    filename ? `파일명: ${filename}` : ""
  ].filter(Boolean).join("\n");
}

function downloadReportExport(result, target, content) {
  const artifact = result.artifact || result.Artifact || {};
  const filename = artifact.Filename || artifact.filename || `${target}-report.txt`;
  const mediaType = artifact.MediaType || artifact.media_type || reports.exportMediaType(target);
  reports.downloadContent(filename, mediaType, content);
}

function downloadContent(filename, mediaType, content) {
  const blob = new Blob([content], { type: mediaType });
  reports.downloadBlob(blob, filename);
}

function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

function filenameFromContentDisposition(value) {
  if (!value) return "";
  const match = value.match(/filename\*=UTF-8''([^;]+)|filename="?([^";]+)"?/i);
  const encoded = match?.[1] || match?.[2] || "";
  if (!encoded) return "";
  try {
    return decodeURIComponent(encoded);
  } catch (err) {
    return encoded;
  }
}

function exportMediaType(target) {
  switch (target) {
    case "html":
      return "text/html;charset=utf-8";
    case "json_ast":
      return "application/json";
    case "markdown":
      return "text/markdown;charset=utf-8";
    default:
      return "text/plain;charset=utf-8";
  }
}

async function viewReportAST(versionID) {
  const owner = captureMissionSelection();
  const key = `version:${versionID}`;
  reports.setReportPreviewLoading(key);
  try {
    const result = await api(`/api/report_versions/${versionID}/ast`);
    if (!ownsMissionSelection(owner)) return;
    reports.applyReportPreview(key, "text", "JSON AST", JSON.stringify(result, null, 2));
  } catch (err) {
    if (!ownsMissionSelection(owner)) return;
    if (state.reportPreview && state.reportPreview.key === key) reports.clearReportPreview();
    showError(err);
  }
}

  Object.assign(reports, { responseError, assertReportExportMatches, reportArtifactPreviewHeader, conversationExportPreviewHeader, reportArtifactHTMLPreviewHeader, reportArtifactDesignedHTMLPreviewHeader, reportExportPreviewHeader, downloadReportExport, downloadContent, downloadBlob, filenameFromContentDisposition, exportMediaType, viewReportAST });
})(window);
