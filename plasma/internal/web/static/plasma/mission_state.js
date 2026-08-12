(function (Plasma) {
  "use strict";

  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const empty = Plasma.ui.empty;

  function resetMissionTransientState() {
    Plasma.polling.clearPendingPoll();
    state.detail = null;
    state.turnPending = false;
    state.reportPending = false;
    state.workflowPending = false;
    state.workflowGoalDraftPending = false;
    state.missionHardDeletePending = false;
    state.workflowGoalDraftRaw = "";
    state.pendingTurn = null;
    Plasma.conversation?.clearLiveTurn?.();
    state.sourceCandidateBusy.clear();
    state.selectedSourceCandidates.clear();
    state.selectedProposals.clear();
    state.selectedReportKey = "";
    state.reportPreview = null;
    state.reportDeletePreview = null;
    state.missionUsage = null;
    state.missionUsageMissionId = "";
    state.missionUsageLoading = false;
    state.missionUsageError = "";
    state.confluenceSearchResults = [];
    state.confluenceSearchContext = null;
    state.confluenceSpaces = [];
    state.confluencePages = [];
    state.confluenceBrowseContext = null;
    state.confluencePreview = null;
    state.confluenceUpdatePreview = null;
    state.confluenceAccess = null;
    state.confluenceBusy = false;
    state.confluenceOAuthURL = "";
    hideBulkBars();
    Plasma.reports?.setReportNotice?.("");
    Plasma.conversation.renderActiveWork({ items: [], blocks: [] });
    Plasma.mission.call("setFormsEnabled", false);
    Plasma.mission.renderMissionLifecycleControls();
    Plasma.mission.renderMissionMetadataEditor?.(null, true);
    Plasma.ui.hideDetail?.();
    clearDetailLists();
    Plasma.mission.renderMissionLoading();
    Plasma.sources.resetConfluenceMissionUI();
    Plasma.settings?.renderMissionUsage?.();
  }

  function hideBulkBars() {
    for (const id of ["sourceCandidateBulk", "proposalBulk"]) {
      const el = $(id);
      if (el) el.classList.add("hidden");
    }
    for (const id of ["sourceCandidateBulkCount", "proposalBulkCount"]) {
      const el = $(id);
      if (el) el.textContent = "0";
    }
  }

  function clearDetailLists() {
    for (const id of ["workflowRunList", "sourceList", "sourceCandidateList", "rejectedSourceCandidateList", "proposalList", "savedList", "claimConfidenceList", "savedClaimList", "reportList", "ledgerList"]) {
      const el = $(id);
      if (el) el.innerHTML = empty("미션 불러오는 중");
    }
    for (const id of ["reportListCount", "sourceListCount", "proposalListCount", "savedClaimListCount", "ledgerCount", "workflowRunCount"]) {
      const el = $(id);
      if (el) el.textContent = "";
    }
  }

  Object.assign(Plasma.mission, { resetMissionTransientState });
})(window.Plasma);
