(function appCompositionRoot(root) {
  "use strict";

  const Plasma = root.Plasma;
  const state = Plasma.state;
  const $ = Plasma.dom.$;

  function requireMission() {
    if (state.missionId) return true;
    Plasma.ui.showError(new Error("먼저 미션을 선택하거나 만들어야 합니다."));
    return false;
  }

  function renderDetail() {
    const detail = state.detail;
    if (!detail) return;
    const events = detail.events || [];
    const reportDraft = Plasma.reports.reportDraftState(events);
    const workflowRuns = detail.workflow_runs || [];
    const activeItems = detail.active_work?.items || [];
    const wasReportPending = state.reportPending;
    state.turnPending = activeItems.some((item) => item.kind === "agent_turn");
    state.reportPending = activeItems.some((item) => item.kind === "report_generation");
    state.workflowPending = activeItems.some((item) => item.kind === "workflow_run");
    setFormsEnabled(Boolean(detail));
    Plasma.sources.renderLocalPathControls();
    Plasma.sources.renderConfluenceControls();
    Plasma.sources.renderConfluenceResults(state.confluenceSearchResults);
    Plasma.conversation.renderAgentOptions(detail.agent_executors || []);
    Plasma.conversation.renderAgentModelOptions(events);
    Plasma.conversation.renderAgentReasoningEffortOptions(events);
    Plasma.reports.modelSelection.render(detail.agent_executors || [], $("agentExecutor").value);
    Plasma.conversation.renderAgentSessionStatus(events);
    Plasma.workflow.renderWorkflowControls(workflowRuns);
    $("missionName").textContent = detail.projection.title || detail.projection.mission_id;
    $("missionObjectiveText").textContent = detail.projection.objective || "목표 없음";
    Plasma.mission.renderMissionLifecycleControls();
    Plasma.mission.renderMissionMetadataEditor?.(detail.projection);
    const sources = detail.sources || [];
    const records = detail.records || {};
    const proposals = records.proposals || [];
    const savedEvidence = Plasma.knowledge.approvedEvidence(proposals, records.evidence || []);
    const savedClaims = Plasma.knowledge.approvedClaims(proposals, records.claims || []);
    $("sourceCount").textContent = `소스 ${sources.length}개`;
    if ($("includeRemovedSources")) $("includeRemovedSources").checked = state.showRemovedSources;
    $("candidateCount").textContent = `후보 ${proposals.filter((proposal) => proposal.state === "pending_review").length}개`;
    $("savedCount").textContent = `저장 ${savedEvidence.length + savedClaims.length}개`;
    Plasma.conversation.renderTurns(events);
    Plasma.sources.renderSources(sources);
    Plasma.sources.renderSourceCandidates(events, sources);
    Plasma.proposals.renderCandidateSourceOptions(sources);
    Plasma.proposals.renderProposals(proposals, records);
    Plasma.knowledge.renderSavedEvidence(savedEvidence);
    Plasma.knowledge.renderClaimConfidenceChanges(records.claim_confidence || [], records.claims || []);
    Plasma.knowledge.renderSavedClaims(savedClaims, records.claim_confidence || []);
    Plasma.reports.renderReports(detail.report_versions || []);
    Plasma.reports.renderReportDraftStatus(reportDraft, wasReportPending);
    Plasma.conversation.renderActiveWork(detail.active_work || {});
    Plasma.ledger.renderLedger(events);
    Plasma.polling.schedulePendingPoll();
  }

  function setFormsEnabled(enabled) {
    const lifecycleBlocked = Plasma.mission.missionLifecycleWriteBlocked();
    for (const id of ["turnText", "agentExecutor", "agentModel", "agentReasoningEffort", "mcpMode", "controllerStrategy", "resetAgentSessionButton", "confluenceAccessConnectionSelect", "confluenceAccessSiteSelect", "confluenceAccessSpaceKey", "confluenceAccessEnable", "confluenceAccessDisable", "missionHardDeleteButton", "workflowInstruction", "workflowStepInstructionMode", "draftWorkflowGoalButton", "workflowRunGoal", "workflowStepInstruction", "startWorkflowButton", "stopWorkflowButton", "sourceTitle", "sourceURI", "sourceContent", "sourceUploadFile", "sourceUploadTitle", "sourceFetchURLButton", "mediaSourceURL", "mediaSourceTitle", "mediaSourceLicense", "mediaSourceAttribution", "pdfSourceURL", "pdfSourceTitle", "localPathRoot", "localPathRelativePath", "localPathTitle", "localPathRestore", "localPathTreeButton", "localPathAttachButton", "confluenceConnectionSelect", "confluenceRefreshConnections", "openConfluenceSettings", "confluenceOneClickStart", "confluenceSiteSelect", "confluencePageURL", "confluenceAddURLButton", "confluenceLoadSpaces", "confluenceLoadMoreSpaces", "confluenceLoadMorePages", "confluenceQuery", "confluenceSpaceKey", "confluenceLimit", "confluenceRangeSelect", "confluenceUpdateRangeSelect", "liquid2Query", "candidateSource", "candidateEvidenceType", "candidateSummary", "reportRigor", "reportAgentModel", "reportAgentReasoningEffort", "reportLongFormExecutionStrategy", "reportGenerationGuidance", "draftQuickReport", "draftLongReport", "cancelReportButton"]) {
      Plasma.ui.setElementDisabled(id, !enabled || lifecycleBlocked ||
        (id === "agentExecutor" && Boolean(Plasma.conversation.lockedAgentExecutor())) ||
        (id === "agentReasoningEffort" && Plasma.conversation.agentReasoningEffortSelectionDisabled(false)) ||
        (state.reportPending && ["turnText", "agentExecutor", "agentModel", "agentReasoningEffort", "mcpMode", "controllerStrategy", "resetAgentSessionButton", "workflowInstruction", "workflowStepInstructionMode", "draftWorkflowGoalButton", "workflowRunGoal", "workflowStepInstruction", "startWorkflowButton", "draftQuickReport", "draftLongReport", "reportRigor", "reportAgentModel", "reportAgentReasoningEffort", "reportLongFormExecutionStrategy", "reportGenerationGuidance"].includes(id)) ||
        (state.workflowGoalDraftPending && ["turnText", "workflowInstruction", "workflowStepInstructionMode", "draftWorkflowGoalButton", "workflowRunGoal", "workflowStepInstruction", "startWorkflowButton"].includes(id)) ||
        (state.turnPending && ["turnText", "agentExecutor", "agentModel", "agentReasoningEffort", "mcpMode", "controllerStrategy", "resetAgentSessionButton", "reportAgentModel", "reportAgentReasoningEffort", "reportLongFormExecutionStrategy", "reportGenerationGuidance", "draftQuickReport", "draftLongReport"].includes(id)) ||
        (state.workflowPending && ["turnText", "agentExecutor", "agentModel", "agentReasoningEffort", "mcpMode", "controllerStrategy", "resetAgentSessionButton", "workflowInstruction", "workflowStepInstructionMode", "draftWorkflowGoalButton", "workflowRunGoal", "workflowStepInstruction", "startWorkflowButton", "reportAgentModel", "reportAgentReasoningEffort", "reportLongFormExecutionStrategy", "reportGenerationGuidance", "draftQuickReport", "draftLongReport"].includes(id)));
    }
    for (const form of ["turnForm", "sourceForm", "sourceUploadForm", "mediaSourceForm", "pdfSourceForm", "localPathForm", "confluenceURLForm", "confluenceSearchForm", "liquid2Form", "candidateForm"]) {
      Plasma.ui.setFormButtonsDisabled(form, (button) => !enabled || lifecycleBlocked || ((state.turnPending || state.workflowPending || state.reportPending) && button.id === "sendTurnButton"));
    }
    if (enabled) Plasma.conversation.setTurnBusy(state.turnPending);
    Plasma.reports.setReportBusy(state.reportPending);
    Plasma.workflow.setWorkflowBusy(state.workflowPending);
  }

  function activeWorkBlocksControl(control) {
    return (state.detail?.active_work?.blocked_controls || []).some((item) => item.control === control);
  }

  function turnControlsBlocked(busy) {
    return busy || activeWorkBlocksControl("turn_submit") || state.workflowGoalDraftPending || Plasma.mission.missionLifecycleWriteBlocked() || !state.detail;
  }

  function agentControlsBlocked() {
    return state.turnPending || state.workflowPending || state.workflowGoalDraftPending || state.reportPending || !state.detail;
  }

  function workflowControlsBlocked() {
    return activeWorkBlocksControl("workflow_start") || Plasma.mission.missionLifecycleWriteBlocked();
  }

  function workflowContinueBlocked() {
    return state.workflowPending || state.reportPending || state.workflowGoalDraftPending;
  }

  async function performActiveWorkAction(action) {
    if (action === "cancel_turn") return Plasma.conversation.cancelTurn();
    if (action === "cancel_report") return Plasma.reports.cancelReport();
    if (action === "view_workflow") {
      $("workflowControlDetails").open = true;
      $("workflowControlDetails").scrollIntoView({ behavior: "smooth", block: "nearest" });
    }
  }

  async function runBulkSequential(items, runOne) {
    // SQLite serializes writes; running these in parallel causes SQLITE_BUSY.
    const errors = [];
    for (const item of items) {
      try {
        await runOne(item);
      } catch (err) {
        errors.push(err);
      }
    }
    return errors;
  }

  function selectedPending() {
    return state.turnPending || state.reportPending || state.workflowPending;
  }

  Plasma.ui.configureTabs({
    loadConfluenceConnections: Plasma.sources.loadConfluenceConnections,
    loadModelDefaults: Plasma.settings.loadModelDefaults
  });
  Plasma.mission.configure({
    beforeSelectionChange: (currentMissionId, nextMissionId) => {
      if (!currentMissionId || !nextMissionId || currentMissionId === nextMissionId) return true;
      return Plasma.reports.redpenController?.beforeLeave() ?? true;
    },
    afterSelectionApplied: async (owner) => {
      await Plasma.sources.loadConfluenceConnections("", owner);
      if (!Plasma.mission.ownsDetailRequest(owner)) return;
      await Plasma.sources.loadConfluenceAccess(owner);
    }
  });
  Plasma.reports.configure({
    activeWorkBlocksControl,
    missionArtifactPreviewURL: Plasma.mission.missionArtifactPreviewURL,
    missionLifecycleWriteBlocked: Plasma.mission.missionLifecycleWriteBlocked,
    onDetailButtonClick: Plasma.ui.onDetailButtonClick,
    reloadMission: Plasma.mission.reloadMission,
    requireMission,
    selectedAgentModel: Plasma.conversation.selectedAgentModel,
    selectedAgentReasoningEffort: Plasma.conversation.selectedAgentReasoningEffort,
    showError: Plasma.ui.showError
  });
  Plasma.proposals.configure({ requireMission, runBulkSequential, showError: Plasma.ui.showError });
  Plasma.bootstrap.start({
    activeWorkBlocksControl,
    agentControlsBlocked,
    performActiveWorkAction,
    renderDetail,
    requireMission,
    runBulkSequential,
    selectedPending,
    setFormsEnabled,
    showError: Plasma.ui.showError,
    turnControlsBlocked,
    workflowContinueBlocked,
    workflowControlsBlocked
  });
})(window);
