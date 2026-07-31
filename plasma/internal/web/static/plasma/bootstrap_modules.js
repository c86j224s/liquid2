(function bootstrapModules(root) {
  "use strict";
  const Plasma = root.Plasma;
  const $ = Plasma.dom.$;
  const reports = Plasma.reports;
  const mission = Plasma.mission;
  const transport = Plasma.transport;
  const ui = Plasma.ui;

  function configure(c) {
  window.Plasma.conversation.configure({
    requireMission: c.requireMission,
	    reloadMission: mission.reloadMission,
    showError: c.showError,
    syncReportControls: reports.syncReportControls,
    turnControlsBlocked: c.turnControlsBlocked,
    agentControlsBlocked: c.agentControlsBlocked,
    renderReportModelSelection: (executors, executor) => reports.modelSelection.render(executors, executor)
  });
  window.Plasma.conversation.configureRendering({
    renderMarkdown: reports.renderMarkdown,
    empty: ui.empty
  });
  window.Plasma.workflow.configure({
    requireMission: c.requireMission,
	    reloadMission: mission.reloadMission,
    showError: c.showError,
    syncReportControls: reports.syncReportControls,
    workflowControlsBlocked: c.workflowControlsBlocked,
    workflowContinueBlocked: c.workflowContinueBlocked,
    rawInputFallback: () => $("turnText").value.trim(),
    setFormsEnabled: c.setFormsEnabled
  });
  window.Plasma.sources.configure({
    requireMission: c.requireMission,
	    reloadMission: mission.reloadMission,
    showError: c.showError,
    renderDetail: c.renderDetail,
	    renderTabs: Plasma.ui.renderTabs,
    setReportNotice: reports.setReportNotice,
    runBulkSequential: c.runBulkSequential,
	    onDetailButtonClick: Plasma.ui.onDetailButtonClick
  });
  window.Plasma.settings.configure({
	    renderTabs: Plasma.ui.renderTabs
  });

  reports.initRedpenController({
    body: $("detailBody"),
    container: $("detailModal"),
    status: $("reportRedpenStatus"),
    startButton: $("reportRedpenStart"),
    saveButton: $("reportRedpenSave"),
    cancelButton: $("reportRedpenCancel")
  });
  window.Plasma.mission.configure({
    api: transport.api,
    forgetStoredMissionID: mission.forgetStoredMissionID,
    markMissionActivitySeen: Plasma.polling.markMissionActivitySeen,
    recordDetailActivityCursor: window.Plasma.polling.recordDetailActivityCursor,
    rememberMissionID: mission.rememberMissionID,
    renderDetail: c.renderDetail,
    renderMissions: mission.renderMissions,
    renderSelectionCleared: () => {
      $("missionName").textContent = "선택된 미션 없음";
      $("missionObjectiveText").textContent = "미션을 만들거나 선택하세요.";
      $("turnLog").innerHTML = ui.empty("미션을 선택하세요");
    },
    resetTransientState: mission.resetMissionTransientState,
    requireMission: c.requireMission,
    setFormsEnabled: c.setFormsEnabled,
    showError: c.showError
  });
  window.Plasma.polling.configure({
    api: transport.api,
    onPendingPollFailure: () => { $("healthBadge").textContent = "재연결 중"; },
    onPendingPollSuccess: () => { $("healthBadge").textContent = "정상"; },
    refreshSelectedMissionDetail: mission.refreshSelectedMissionDetail,
    renderMissions: mission.renderMissions,
    selectedPending: c.selectedPending
  });

  }

  Plasma.bootstrapModules = { configure };
})(window);
