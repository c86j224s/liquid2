(function reportsRedpenInit(root) {
  "use strict";

  const Plasma = root.Plasma;
  const reports = Plasma.reports;
  const mission = Plasma.mission;
  const missionApi = Plasma.transport.missionApi;

  function initRedpenController(elements) {
    reports.redpenController = reports.createRedpenController?.({
      ...elements,
      render: (content, editable) => `<div class="report-modal-body turn-markdown">${reports.renderMarkdown(content, { redpenBlocks: editable })}</div>`,
      enhance: (container) => {
        reports.renderPlasmaMath?.(container);
        reports.renderPlasmaMermaid?.(container);
        reports.enhanceImages?.(container);
        reports.tocController?.refresh();
      },
      load: (sourceArtifactID) => missionApi(mission.captureMissionSelection(), `/artifacts/${encodeURIComponent(sourceArtifactID)}/redpen`),
      save: (sourceArtifactID, content, expectedCurrentArtifactID) => missionApi(mission.captureMissionSelection(), `/artifacts/${encodeURIComponent(sourceArtifactID)}/redpen`, {
        method: "POST",
        body: { content, expected_current_artifact_id: expectedCurrentArtifactID }
      }),
      saved: () => mission.refreshSelectedMissionDetail(),
      error: (error) => {
        if (!mission.isStaleMissionOperation(error)) reports.call("showError", error);
      },
      notify: reports.setReportNotice,
      confirm: (message) => root.confirm(message),
      onDraftChange: (content) => {
        Plasma.state.detailText = String(content ?? "");
      }
    }) || null;
    Plasma.ui.configureDetailHooks({
      beforeLeave: () => {
        const canLeave = reports.redpenController?.beforeLeave() ?? true;
        if (canLeave) reports.tocController?.reset();
        return canLeave;
      },
      copyContent: () => reports.redpenController?.copyContent()
    });
    return reports.redpenController;
  }

  Object.assign(reports, { initRedpenController });
})(window);
