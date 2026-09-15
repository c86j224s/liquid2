(function reportsEvents(root) {
  "use strict";
  const reports = root.Plasma.reports;

  function onConversationExportClick(event) {
    if (reports.call("onDetailButtonClick", event)) return true;
    if (event.target.closest("[data-conversation-export-create]")) {
      reports.createConversationExport();
      return true;
    }
    const button = event.target.closest("[data-conversation-export-id][data-action]");
    if (!button) return false;
    const artifactID = button.dataset.conversationExportId;
    if (button.dataset.action === "download") reports.downloadReportArtifact(artifactID);
    else reports.viewConversationExport(artifactID);
    return true;
  }

  function onReportListClick(event) {
    if (reports.call("onDetailButtonClick", event)) return;
    const planButton = event.target.closest("[data-report-plan-event-id][data-action]");
    if (planButton) {
      reports.showReportPlanEvent(planButton.dataset.reportPlanEventId);
      return;
    }
    const artifactButton = event.target.closest("[data-report-artifact-id][data-action]");
    if (artifactButton) {
      const artifactID = artifactButton.dataset.reportArtifactId;
      const action = artifactButton.dataset.action;
      if (action === "download-artifact") reports.downloadReportArtifact(artifactID);
      else if (action === "view-text-artifact") {
        const bundleCard = event.target.closest("[data-report-il-bundle-key]");
        reports.viewReportTextArtifact(
          artifactID,
          artifactButton.dataset.reportArtifactLabel || "Artifact",
          bundleCard?.dataset.reportIlBundleKey || `artifact:${artifactID}`
        );
      }
      else if (action === "view-stored-html-artifact") reports.viewStoredReportHTMLArtifact(artifactID);
      else if (action === "view-redpen-artifact") reports.viewReportRedpenWorkcopy(artifactID);
      else if (action === "download-redpen-artifact") reports.downloadReportRedpenWorkcopy(artifactID);
      else if (action === "view-html-artifact") reports.exportReportArtifactHTML(artifactID);
      else if (action === "download-html-artifact") reports.exportReportArtifactHTML(artifactID, { download: true });
      else if (action === "patch-artifact") reports.patchReportArtifact(artifactID, artifactButton.dataset.reportTitle || "");
      else if (action === "delete-report-artifact") reports.deleteReportArtifact(artifactID);
      else if (action === "view-designed-html-artifact" || action === "start-designed-html-artifact") reports.exportReportArtifactDesignedHTML(artifactID);
      else if (action === "download-designed-html-artifact") reports.exportReportArtifactDesignedHTML(artifactID, { download: true });
      else reports.viewReportArtifact(artifactID);
      return;
    }
    const button = event.target.closest("[data-report-version-id][data-action]");
    if (button) {
      const versionID = button.dataset.reportVersionId;
      const action = button.dataset.action;
      if (action === "ast") reports.viewReportAST(versionID);
      else if (action === "plan") reports.showReportPlan(versionID);
      else if (action === "mcp-trace") reports.showMCPTrace(versionID);
      else if (action.startsWith("download-")) reports.exportReport(versionID, action.slice("download-".length), { download: true });
      else reports.exportReport(versionID, action);
      return;
    }
    if (event.target.closest("a") || event.target.closest("summary")) return;
    const card = event.target.closest("[data-report-key]");
    if (card) reports.selectReport(card.dataset.reportKey);
  }

  Object.assign(reports, { onConversationExportClick, onReportListClick });
})(window);
