(function bootstrapModule(root) {
  "use strict";
  const Plasma = root.Plasma;
  const $ = Plasma.dom.$;
  const state = Plasma.state;
  const reports = Plasma.reports;
  const ui = Plasma.ui;
  const mission = Plasma.mission;
  const conversation = Plasma.conversation;
  const workflow = Plasma.workflow;

  function start(callbacks) {
    const c = callbacks || {};
    document.addEventListener("DOMContentLoaded", () => {

	  Plasma.bootstrapModules.configure(c);
	  Plasma.bootstrap.initMissionRailToggle();
	  mission.initMissionMetadataEditor?.();
	  $("refreshMissions").addEventListener("click", mission.loadMissions);
	  $("includeArchivedMissions").addEventListener("change", mission.onIncludeArchivedMissionsChange);
	  $("missionArchiveButton").addEventListener("click", () => mission.changeMissionLifecycle("archive"));
	  $("missionRestoreButton").addEventListener("click", () => mission.changeMissionLifecycle("restore"));
	  $("missionHardDeleteButton").addEventListener("click", mission.hardDeleteMission);
	  $("tabBar").addEventListener("click", ui.onTabBarClick);
	  $("missionForm").addEventListener("submit", mission.createMission);
  $("turnForm").addEventListener("submit", conversation.sendTurn);
  $("workflowInstruction").addEventListener("input", workflow.onWorkflowRawInput);
  $("workflowStepInstructionMode").addEventListener("change", workflow.updateWorkflowStepInstructionMode);
  $("draftWorkflowGoalButton").addEventListener("click", workflow.draftWorkflowGoal);
  $("startWorkflowButton").addEventListener("click", workflow.startWorkflow);
  $("stopWorkflowButton").addEventListener("click", workflow.stopWorkflow);
  $("workflowRunList").addEventListener("click", workflow.onWorkflowRunListClick);
  $("openSourceCandidatesButton").addEventListener("click", window.Plasma.sources.openSourceCandidatesTab);
  $("resetAgentSessionButton").addEventListener("click", conversation.resetAgentSession);
  $("agentExecutor").addEventListener("change", conversation.onAgentExecutorChange);
  $("agentModel").addEventListener("change", conversation.onAgentModelChange);
  $("agentReasoningEffort").addEventListener("change", conversation.onAgentReasoningEffortChange);
  $("controllerStrategy").addEventListener("change", conversation.renderAgentControlsSummary);
  $("confluenceAccessConnectionSelect").addEventListener("change", window.Plasma.sources.renderConfluenceAccessControls);
  $("confluenceAccessSiteSelect").addEventListener("change", window.Plasma.sources.renderConfluenceAccessControls);
  $("confluenceAccessEnable").addEventListener("click", window.Plasma.sources.enableConfluenceAccess);
  $("confluenceAccessDisable").addEventListener("click", window.Plasma.sources.disableConfluenceAccess);
  $("modelDefaultsForm").addEventListener("submit", window.Plasma.settings.saveModelDefaults);
  $("modelDefaultsRefresh").addEventListener("click", () => window.Plasma.settings.loadModelDefaults());
  $("missionUsageRefresh").addEventListener("click", () => window.Plasma.settings.loadMissionUsage());
  $("workflowGoalDefaultModel").addEventListener("change", window.Plasma.settings.renderModelDefaultEfforts);
  $("sourceForm").addEventListener("submit", window.Plasma.sources.addTextSource);
  $("sourceUploadForm").addEventListener("submit", window.Plasma.sources.addUploadSource);
  $("sourceFetchURLButton").addEventListener("click", window.Plasma.sources.addURLSourceFromTextForm);
  $("mediaSourceForm").addEventListener("submit", window.Plasma.sources.addMediaURLSource);
  $("pdfSourceForm").addEventListener("submit", window.Plasma.sources.addPDFURLSource);
  $("localPathForm").addEventListener("submit", window.Plasma.sources.attachLocalPathSource);
  $("localPathTreeButton").addEventListener("click", () => window.Plasma.sources.browseLocalPathTree());
  $("localPathTree").addEventListener("click", window.Plasma.sources.onLocalPathTreeClick);
  $("localPathBreadcrumb").addEventListener("click", window.Plasma.sources.onLocalPathBreadcrumbClick);
  $("localPathRoot").addEventListener("change", () => {
    $("localPathRelativePath").value = "";
    state.localPathSelectedFile = "";
    window.Plasma.sources.browseLocalPathTree();
  });
  $("localPathRelativePath").addEventListener("input", window.Plasma.sources.updateLocalPathAttachState);
  $("localPathSourceDetails").addEventListener("toggle", (event) => {
    if (event.target.open && state.missionId && state.localPathRoots.length) {
      window.Plasma.sources.browseLocalPathTree();
    }
  });
  $("confluenceSourceDetails").addEventListener("toggle", (event) => {
    if (event.target.open && state.missionId) window.Plasma.sources.loadConfluenceConnections();
  });
  $("confluenceSettingsDetails").addEventListener("toggle", (event) => {
    if (event.target.open) window.Plasma.sources.loadConfluenceConnections();
  });
  $("modelDefaultsDetails").addEventListener("toggle", (event) => {
    if (event.target.open) window.Plasma.settings.loadModelDefaults();
  });
  $("missionUsageDetails").addEventListener("toggle", (event) => {
    if (event.target.open) window.Plasma.settings.loadMissionUsage();
  });
  $("confluenceRefreshConnections").addEventListener("click", () => window.Plasma.sources.loadConfluenceConnections());
  $("confluenceConnectionSelect").addEventListener("change", () => {
    window.Plasma.sources.clearConfluenceDiscovery();
    window.Plasma.sources.renderConfluenceControls();
  });
  $("openConfluenceSettings").addEventListener("click", window.Plasma.settings.openSettingsTab);
  $("confluenceSettingsRefreshConnections").addEventListener("click", () => window.Plasma.sources.loadConfluenceConnections());
  $("confluenceSettingsConnections").addEventListener("click", window.Plasma.settings.onConfluenceSettingsCardClick);
  $("confluenceSettingsAPIForm").addEventListener("submit", window.Plasma.settings.connectConfluenceAPIToken);
  $("confluenceOneClickStart").addEventListener("click", () => window.Plasma.sources.runConfluenceOneClickFlow());
  $("confluenceSiteSelect").addEventListener("change", window.Plasma.sources.clearConfluenceDiscovery);
  $("confluenceLoadSpaces").addEventListener("click", () => window.Plasma.sources.loadConfluenceSpaces());
  $("confluenceLoadMoreSpaces").addEventListener("click", window.Plasma.sources.loadMoreConfluenceSpaces);
  $("confluenceLoadMorePages").addEventListener("click", window.Plasma.sources.loadMoreConfluencePages);
  $("confluenceSpaces").addEventListener("click", window.Plasma.sources.onConfluenceSpacesClick);
  $("confluencePages").addEventListener("click", window.Plasma.sources.onConfluencePagesClick);
  $("confluenceURLForm").addEventListener("submit", window.Plasma.sources.addConfluenceURLSource);
  $("confluenceApproveFullSnapshot").addEventListener("click", () => window.Plasma.sources.approveConfluenceSnapshot(false));
  $("confluenceApproveRangeSnapshot").addEventListener("click", () => window.Plasma.sources.approveConfluenceSnapshot(true));
  $("confluenceUpdatePreviewButton").addEventListener("click", window.Plasma.sources.previewConfluenceUpdate);
  $("confluenceApproveUpdate").addEventListener("click", window.Plasma.sources.approveConfluenceUpdate);
  $("confluenceSearchForm").addEventListener("submit", window.Plasma.sources.searchConfluence);
  $("confluenceResults").addEventListener("click", window.Plasma.sources.onConfluenceResultsClick);
  $("includeRemovedSources").addEventListener("change", window.Plasma.sources.toggleRemovedSources);
  $("liquid2Form").addEventListener("submit", window.Plasma.sources.searchLiquid2);
	  $("candidateForm").addEventListener("submit", Plasma.proposals.proposeEvidence);
  $("draftQuickReport").addEventListener("click", () => reports.draftReport("planned"));
  $("draftLongReport").addEventListener("click", () => reports.draftReport("long_form"));
	$("reportAgentModel").addEventListener("change", () => {
		const status = reports.modelSelection.configuredStatus(state.detail?.agent_executors || [], $("agentExecutor").value);
		reports.modelSelection.refreshEfforts(status);
	});
  $("cancelReportButton").addEventListener("click", reports.cancelReport);
  document.addEventListener("click", (event) => {
    const action = event.target.closest("[data-active-work-action]")?.dataset.activeWorkAction;
    if (action) c.performActiveWorkAction(action);
	    if (event.target.closest("[data-retry-mission-load]")) mission.reloadMission();
  });
  $("copyError").addEventListener("click", ui.copyError);
  $("closeError").addEventListener("click", ui.hideError);
  $("copyDetail").addEventListener("click", ui.copyDetail);
  $("closeDetail").addEventListener("click", ui.hideDetail);
  $("detailBody").addEventListener("scroll", ui.updateDetailScrollRatio);
  window.addEventListener("resize", ui.updateDetailScrollRatio);
  $("detailModal").addEventListener("click", ui.onDetailModalClick);
	  $("missionList").addEventListener("click", mission.onMissionListClick);
	  $("missionRecallButton").addEventListener("click", mission.showMissionRecall);
  $("turnLog").addEventListener("click", (event) => {
    const button = event.target.closest("[data-copy-text]");
    if (button) ui.copyText(button.dataset.copyText).catch(ui.showError);
  });
  $("turnLog").addEventListener("scroll", conversation.updateTurnNavVisibility, { passive: true });
  $("turnNav").addEventListener("click", conversation.onTurnNavClick);
  $("turnNav").addEventListener("pointerdown", conversation.onTurnNavPointerDown);
  window.addEventListener("pointerup", conversation.stopTurnStep);
  window.addEventListener("pointercancel", conversation.stopTurnStep);
  workflow.updateWorkflowStepInstructionMode();
  $("liquid2Results").addEventListener("click", window.Plasma.sources.onLiquid2ResultsClick);
  $("sourceCandidateList").addEventListener("click", window.Plasma.sources.onSourceCandidateListClick);
  $("rejectedSourceCandidateList").addEventListener("click", window.Plasma.sources.onRejectedSourceCandidateListClick);
  $("sourceList").addEventListener("click", window.Plasma.sources.onSourceListClick);
	  $("proposalList").addEventListener("click", Plasma.proposals.onProposalListClick);
	  $("savedList").addEventListener("click", ui.onDetailButtonClick);
	  $("claimConfidenceList").addEventListener("click", Plasma.knowledge.onDetailButtonClick);
	  $("savedClaimList").addEventListener("click", Plasma.knowledge.onDetailButtonClick);
  $("reportList").addEventListener("click", reports.onReportListClick);
	  $("ledgerList").addEventListener("click", ui.onDetailButtonClick);
  c.setFormsEnabled(false);
  // Deep-link the initial tab via URL hash (e.g. #reports), when valid.
  const initialTab = decodeURIComponent((location.hash || "").replace(/^#/, "")).trim();
  if (initialTab && document.querySelector(`[data-tab="${CSS.escape(initialTab)}"]`)) {
    state.activeTab = initialTab;
  }
	  ui.renderTabs();

  Plasma.bootstrapExtras.init(c);
	  Plasma.bootstrap.boot();

    });
  }

  Plasma.bootstrap = { start };
})(window);
