const state = {
  missions: [],
  missionId: "",
  detail: null,
  lastError: "",
  turnPending: false,
  reportPending: false,
  workflowPending: false,
  workflowGoalDraftPending: false,
  workflowGoalDraftRaw: "",
  pendingTurn: null,
  pollTimer: 0,
  pollInFlight: false,
  sourceCandidateBusy: new Set(),
  selectedSourceCandidates: new Set(),
  selectedProposals: new Set(),
  confluenceConnections: [],
  confluenceSearchResults: [],
  confluenceSearchContext: null,
  confluenceSpaces: [],
  confluencePages: [],
  confluenceBrowseContext: null,
  confluencePreview: null,
  confluenceUpdatePreview: null,
  confluenceOAuthURL: "",
  confluenceOAuthConfigured: false,
  confluenceBusy: false,
  confluenceAccess: null,
  localPathRoots: [],
  localPathSelectedFile: "",
  localPathCurrentDir: ".",
  showRemovedSources: false,
  activeTab: "conversation",
  detailText: "",
  selectedReportKey: "",
  reportPreview: null,
  turnScrollMission: "",
  agentModelTouched: false,
  agentModelExecutor: "",
  agentReasoningEffortTouched: false,
  agentReasoningEffortExecutor: ""
};

const EVIDENCE_TYPE_LABELS = {
  fact: "사실",
  observation: "관찰",
  quote: "인용",
  statistic: "통계",
  table_row: "표 항목",
  interpretation: "해석/평가",
  reaction: "반응",
  rumor: "루머/미확정설",
  controversy: "논쟁 축",
  market_signal: "시장 신호",
  code: "코드",
  formula: "수식",
  benchmark: "벤치마크",
  open_question: "열린 질문",
  user_assertion: "사용자 진술"
};

const REPORT_RIGOR_LABELS = {
  exploratory: "탐색적",
  balanced: "균형형",
  strict: "검증형"
};

const REPORT_MODE_LABELS = {
  one_take: "원테이크 보고서",
  planned: "보고서",
  long_form: "장문 보고서"
};

const AGENT_MODEL_OPTIONS = {
  claude: [
    { value: "", label: "기본값" },
    { value: "haiku", label: "Claude Haiku (haiku)" },
    { value: "sonnet", label: "Claude Sonnet (sonnet)" },
    { value: "opus", label: "Claude Opus (opus)" }
  ]
};

const AGENT_REASONING_EFFORT_OPTIONS = {
  claude: [
    { value: "", label: "지정 불가" }
  ]
};

const DESIGNED_REPORT_RENDERER_VERSION = "dh26-inline-images-20260706";

const $ = (id) => document.getElementById(id);
const MISSION_STORAGE_KEY = "plasma.activeMissionId";

const markdownRenderer = window.markdownit ? window.markdownit({
  html: false,
  linkify: true,
  breaks: true,
  typographer: false
}) : null;

if (markdownRenderer) {
  const defaultLinkOpen = markdownRenderer.renderer.rules.link_open ||
    ((tokens, idx, options, env, self) => self.renderToken(tokens, idx, options));
  markdownRenderer.renderer.rules.link_open = (tokens, idx, options, env, self) => {
    const token = tokens[idx];
    const targetIndex = token.attrIndex("target");
    if (targetIndex < 0) {
      token.attrPush(["target", "_blank"]);
    } else {
      token.attrs[targetIndex][1] = "_blank";
    }
    const relIndex = token.attrIndex("rel");
    if (relIndex < 0) {
      token.attrPush(["rel", "noopener noreferrer"]);
    } else {
      token.attrs[relIndex][1] = "noopener noreferrer";
    }
    return defaultLinkOpen(tokens, idx, options, env, self);
  };
}

document.addEventListener("DOMContentLoaded", () => {
  $("refreshMissions").addEventListener("click", loadMissions);
  $("tabBar").addEventListener("click", onTabBarClick);
  $("missionForm").addEventListener("submit", createMission);
  $("turnForm").addEventListener("submit", sendTurn);
  $("cancelTurnButton").addEventListener("click", cancelTurn);
  $("workflowInstruction").addEventListener("input", onWorkflowRawInput);
  $("turnText").addEventListener("input", onWorkflowRawInput);
  $("workflowStepInstructionMode").addEventListener("change", updateWorkflowStepInstructionMode);
  $("draftWorkflowGoalButton").addEventListener("click", draftWorkflowGoal);
  $("startWorkflowButton").addEventListener("click", startWorkflow);
  $("stopWorkflowButton").addEventListener("click", stopWorkflow);
  $("workflowRunList").addEventListener("click", onWorkflowRunListClick);
  $("openSourceCandidatesButton").addEventListener("click", openSourceCandidatesTab);
  $("resetAgentSessionButton").addEventListener("click", resetAgentSession);
  $("agentExecutor").addEventListener("change", onAgentExecutorChange);
  $("agentModel").addEventListener("change", onAgentModelChange);
  $("agentReasoningEffort").addEventListener("change", onAgentReasoningEffortChange);
  $("controllerStrategy").addEventListener("change", renderAgentControlsSummary);
  $("confluenceAccessConnectionSelect").addEventListener("change", renderConfluenceAccessControls);
  $("confluenceAccessSiteSelect").addEventListener("change", renderConfluenceAccessControls);
  $("confluenceAccessEnable").addEventListener("click", enableConfluenceAccess);
  $("confluenceAccessDisable").addEventListener("click", disableConfluenceAccess);
  $("sourceForm").addEventListener("submit", addTextSource);
  $("sourceUploadForm").addEventListener("submit", addUploadSource);
  $("sourceFetchURLButton").addEventListener("click", addURLSourceFromTextForm);
  $("mediaSourceForm").addEventListener("submit", addMediaURLSource);
  $("pdfSourceForm").addEventListener("submit", addPDFURLSource);
  $("localPathForm").addEventListener("submit", attachLocalPathSource);
  $("localPathTreeButton").addEventListener("click", () => browseLocalPathTree());
  $("localPathTree").addEventListener("click", onLocalPathTreeClick);
  $("localPathBreadcrumb").addEventListener("click", onLocalPathBreadcrumbClick);
  $("localPathRoot").addEventListener("change", () => {
    $("localPathRelativePath").value = "";
    state.localPathSelectedFile = "";
    browseLocalPathTree();
  });
  $("localPathRelativePath").addEventListener("input", updateLocalPathAttachState);
  $("localPathSourceDetails").addEventListener("toggle", (event) => {
    if (event.target.open && state.missionId && state.localPathRoots.length) {
      browseLocalPathTree();
    }
  });
  $("confluenceSourceDetails").addEventListener("toggle", (event) => {
    if (event.target.open && state.missionId) loadConfluenceConnections();
  });
  $("confluenceSettingsDetails").addEventListener("toggle", (event) => {
    if (event.target.open) loadConfluenceConnections();
  });
  $("confluenceRefreshConnections").addEventListener("click", () => loadConfluenceConnections());
  $("confluenceConnectionSelect").addEventListener("change", () => {
    clearConfluenceDiscovery();
    renderConfluenceControls();
  });
  $("openConfluenceSettings").addEventListener("click", openSettingsTab);
  $("confluenceSettingsRefreshConnections").addEventListener("click", () => loadConfluenceConnections());
  $("confluenceSettingsConnections").addEventListener("click", onConfluenceSettingsCardClick);
  $("confluenceSettingsAPIForm").addEventListener("submit", connectConfluenceAPIToken);
  $("confluenceOneClickStart").addEventListener("click", () => runConfluenceOneClickFlow());
  $("confluenceSiteSelect").addEventListener("change", clearConfluenceDiscovery);
  $("confluenceLoadSpaces").addEventListener("click", () => loadConfluenceSpaces());
  $("confluenceLoadMoreSpaces").addEventListener("click", loadMoreConfluenceSpaces);
  $("confluenceLoadMorePages").addEventListener("click", loadMoreConfluencePages);
  $("confluenceSpaces").addEventListener("click", onConfluenceSpacesClick);
  $("confluencePages").addEventListener("click", onConfluencePagesClick);
  $("confluenceURLForm").addEventListener("submit", addConfluenceURLSource);
  $("confluenceApproveFullSnapshot").addEventListener("click", () => approveConfluenceSnapshot(false));
  $("confluenceApproveRangeSnapshot").addEventListener("click", () => approveConfluenceSnapshot(true));
  $("confluenceUpdatePreviewButton").addEventListener("click", previewConfluenceUpdate);
  $("confluenceApproveUpdate").addEventListener("click", approveConfluenceUpdate);
  $("confluenceSearchForm").addEventListener("submit", searchConfluence);
  $("confluenceResults").addEventListener("click", onConfluenceResultsClick);
  $("includeRemovedSources").addEventListener("change", toggleRemovedSources);
  $("liquid2Form").addEventListener("submit", searchLiquid2);
  $("candidateForm").addEventListener("submit", proposeEvidence);
  $("draftQuickReport").addEventListener("click", () => draftReport("planned"));
  $("draftLongReport").addEventListener("click", () => draftReport("long_form"));
  $("cancelReportButton").addEventListener("click", cancelReport);
  $("copyError").addEventListener("click", copyError);
  $("closeError").addEventListener("click", hideError);
  $("copyDetail").addEventListener("click", copyDetail);
  $("closeDetail").addEventListener("click", hideDetail);
  $("detailModal").addEventListener("click", onDetailModalClick);
  $("missionList").addEventListener("click", onMissionListClick);
  $("missionRecallButton").addEventListener("click", showMissionRecall);
  $("turnLog").addEventListener("click", onTurnLogClick);
  $("turnLog").addEventListener("scroll", updateTurnNavVisibility, { passive: true });
  $("turnNav").addEventListener("click", onTurnNavClick);
  $("turnNav").addEventListener("pointerdown", onTurnNavPointerDown);
  window.addEventListener("pointerup", stopTurnStep);
  window.addEventListener("pointercancel", stopTurnStep);
  updateWorkflowStepInstructionMode();
  $("liquid2Results").addEventListener("click", onLiquid2ResultsClick);
  $("sourceCandidateList").addEventListener("click", onSourceCandidateListClick);
  $("rejectedSourceCandidateList").addEventListener("click", onRejectedSourceCandidateListClick);
  $("sourceList").addEventListener("click", onSourceListClick);
  $("proposalList").addEventListener("click", onProposalListClick);
  $("savedList").addEventListener("click", onDetailButtonClick);
  $("claimConfidenceList").addEventListener("click", onDetailButtonClick);
  $("savedClaimList").addEventListener("click", onDetailButtonClick);
  $("reportList").addEventListener("click", onReportListClick);
  $("ledgerList").addEventListener("click", onDetailButtonClick);
  setFormsEnabled(false);
  // Deep-link the initial tab via URL hash (e.g. #reports), when valid.
  const initialTab = decodeURIComponent((location.hash || "").replace(/^#/, "")).trim();
  if (initialTab && document.querySelector(`[data-tab="${CSS.escape(initialTab)}"]`)) {
    state.activeTab = initialTab;
  }
  renderTabs();

  // ── Source-add methods behave like tabs (one panel at a time) ──
  (function initSourceTabs() {
    const group = document.querySelector(".source-add-group");
    if (!group) return;
    const tabs = Array.from(group.querySelectorAll(".source-tab"));
    const panels = tabs
      .map((tab) => document.getElementById(tab.dataset.sourceTab))
      .filter(Boolean);
    let syncing = false;
    function activate(id) {
      if (syncing) return;
      syncing = true;
      for (const panel of panels) panel.open = panel.id === id;
      for (const tab of tabs) {
        const on = tab.dataset.sourceTab === id;
        tab.classList.toggle("active", on);
        tab.setAttribute("aria-selected", on ? "true" : "false");
      }
      syncing = false;
    }
    for (const tab of tabs) {
      tab.addEventListener("click", () => activate(tab.dataset.sourceTab));
    }
    // Programmatic opens (e.g. Confluence flow sets details.open = true) sync tabs.
    for (const panel of panels) {
      panel.addEventListener("toggle", () => {
        if (panel.open) activate(panel.id);
      });
    }
    activate(panels[0] && panels[0].id);
  })();

  // ── Wave 6a: theme toggle ──────────────────────────────────
  (function initTheme() {
    const STORAGE_KEY = "plasma.theme";
    const root = document.documentElement;
    const btn = $("themeToggle");
    const saved = localStorage.getItem(STORAGE_KEY);
    const mql = window.matchMedia("(prefers-color-scheme: light)");

    function applyTheme(theme) {
      root.dataset.theme = theme;
      if (btn) btn.textContent = theme === "light" ? "🌙" : "☀";
    }

    if (saved === "dark" || saved === "light") {
      applyTheme(saved);
    } else {
      applyTheme(mql.matches ? "light" : "dark");
    }

    if (btn) {
      btn.addEventListener("click", () => {
        const next = root.dataset.theme === "light" ? "dark" : "light";
        localStorage.setItem(STORAGE_KEY, next);
        applyTheme(next);
      });
    }

    mql.addEventListener("change", (e) => {
      if (!localStorage.getItem(STORAGE_KEY)) {
        applyTheme(e.matches ? "light" : "dark");
      }
    });
  })();

  // ── Focus mode: fold the header chrome to maximise the conversation ──
  (function initFocusToggle() {
    const STORAGE_KEY = "plasma.chatFocus";
    const btn = $("focusToggle");
    const apply = (on) => {
      document.body.classList.toggle("chat-focus", on);
      if (btn) {
        btn.classList.toggle("active", on);
        btn.setAttribute("aria-pressed", on ? "true" : "false");
        btn.textContent = on ? "⤡" : "⤢";
      }
    };
    apply(localStorage.getItem(STORAGE_KEY) === "1");
    if (btn) {
      btn.addEventListener("click", () => {
        const on = !document.body.classList.contains("chat-focus");
        localStorage.setItem(STORAGE_KEY, on ? "1" : "0");
        apply(on);
      });
    }
  })();

  // ── Wave 6b: ⌘/Ctrl+Enter to send ─────────────────────────
  $("turnText").addEventListener("keydown", (e) => {
    if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
      e.preventDefault();
      $("turnForm").dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    }
  });

  // ── Wave 6c: Escape to close modal / toast / mission picker ─
  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      if ($("detailModal") && !$("detailModal").classList.contains("hidden")) {
        hideDetail();
      }
      if ($("errorToast") && !$("errorToast").classList.contains("hidden")) {
        hideError();
      }
      if (document.body.classList.contains("mission-picker-open")) {
        document.body.classList.remove("mission-picker-open");
      }
    }
  });

  // ── Wave 6e: Multi-select bulk approve/reject ──────────────
  $("sourceCandidateList").addEventListener("change", (e) => {
    const cb = e.target.closest("input.item-select[data-select-source-url]");
    if (!cb) return;
    toggleSourceCandidateSelection(cb);
  });
  $("proposalList").addEventListener("change", (e) => {
    const cb = e.target.closest("input.item-select[data-select-proposal-id]");
    if (!cb) return;
    toggleProposalSelection(cb);
  });
  $("sourceCandidateSelectAll").addEventListener("click", selectAllSourceCandidates);
  $("sourceCandidateClearSelection").addEventListener("click", clearSourceCandidateSelection);
  $("sourceCandidateBulkApprove").addEventListener("click", () => bulkSourceCandidateAction("approve"));
  $("sourceCandidateBulkReject").addEventListener("click", () => bulkSourceCandidateAction("reject"));
  $("proposalSelectAll").addEventListener("click", selectAllProposals);
  $("proposalClearSelection").addEventListener("click", clearProposalSelection);
  $("proposalBulkApprove").addEventListener("click", () => bulkProposalAction("approve"));
  $("proposalBulkReject").addEventListener("click", () => bulkProposalAction("reject"));

  // ── Wave 6d: Mobile mission picker (bottom sheet) ──────────
  (function initMissionPicker() {
    const openBtn = $("missionPickerOpen");
    const closeBtn = $("missionPickerClose");
    const rail = document.querySelector(".rail");
    if (openBtn) {
      openBtn.addEventListener("click", (e) => {
        e.stopPropagation();
        document.body.classList.toggle("mission-picker-open");
      });
    }
    if (closeBtn) {
      closeBtn.addEventListener("click", () => {
        document.body.classList.remove("mission-picker-open");
      });
    }
    // auto-close on mission select
    $("missionList").addEventListener("click", (e) => {
      if (e.target.closest("button.item[data-mission-id]")) {
        document.body.classList.remove("mission-picker-open");
      }
    });
    // click outside the sheet (backdrop) closes
    document.addEventListener("click", (e) => {
      if (!document.body.classList.contains("mission-picker-open")) return;
      if (openBtn && openBtn.contains(e.target)) return;
      if (rail && rail.contains(e.target)) return;
      document.body.classList.remove("mission-picker-open");
    });
  })();

  boot();
});

async function boot() {
  try {
    const health = await api("/api/health");
    $("healthBadge").textContent = health.Status || "정상";
    await loadRuntimeInfo();
  } catch (err) {
    showError(err);
    $("healthBadge").textContent = "오프라인";
  }
  await loadLocalPathRoots();
  await loadMissions();
}

async function loadRuntimeInfo() {
  const runtime = await api("/api/runtime");
  const label = (runtime.environment_label || "").trim();
  const badge = $("environmentBadge");
  if (!badge) return;
  if (!label) {
    badge.classList.add("hidden");
    badge.textContent = "";
    return;
  }
  badge.textContent = label;
  badge.classList.remove("hidden");
}

async function api(path, options = {}) {
  const init = {
    method: options.method || "GET",
    headers: { "Accept": "application/json" }
  };
  if (options.body !== undefined) {
    if (options.body instanceof FormData) {
      init.body = options.body;
    } else {
      init.headers["Content-Type"] = "application/json";
      init.body = JSON.stringify(options.body);
    }
  }
  let response;
  try {
    response = await fetch(path, init);
  } catch (err) {
    const wrapped = new Error(`Network request failed: ${err?.message || String(err)}`);
    wrapped.userMessage = "서버에 연결할 수 없습니다. 잠시 후 다시 시도하거나 Plasma 서버 상태를 확인하세요.";
    wrapped.details = { path, method: init.method, cause: err?.message || String(err) };
    wrapped.isNetworkError = true;
    throw wrapped;
  }
  const text = await response.text();
  let data = {};
  if (text.trim() !== "") {
    try {
      data = JSON.parse(text);
    } catch (err) {
      data = { raw: text };
    }
  }
  if (!response.ok) {
    const message = data.error?.message || response.statusText || "요청 실패";
    const err = new Error(`HTTP ${response.status}: ${message}`);
    err.userMessage = message;
    err.status = response.status;
    err.details = data;
    throw err;
  }
  return data;
}

async function loadMissions() {
  try {
    const data = await api("/api/missions");
    state.missions = data.missions || [];
    renderMissions();
    const missionIDs = new Set(state.missions.map((mission) => mission.MissionID).filter(Boolean));
    const savedMissionID = localStorage.getItem(MISSION_STORAGE_KEY) || "";
    const nextMissionID = missionIDs.has(state.missionId) ? state.missionId :
      (missionIDs.has(savedMissionID) ? savedMissionID : state.missions[0]?.MissionID);
    if (nextMissionID) await selectMission(nextMissionID);
  } catch (err) {
    showError(err);
  }
}

async function createMission(event) {
  event.preventDefault();
  const body = {
    title: $("missionTitle").value,
    objective: $("missionObjective").value,
    scope: { included: [], excluded: [] }
  };
  try {
    const detail = await api("/api/missions", { method: "POST", body });
    $("missionTitle").value = "";
    $("missionObjective").value = "";
    state.missionId = detail.projection.mission_id;
    localStorage.setItem(MISSION_STORAGE_KEY, state.missionId);
    await loadMissions();
    state.detail = detail;
    renderDetail();
  } catch (err) {
    showError(err);
  }
}

async function selectMission(missionId) {
  if (!missionId) return;
  const previousMissionID = state.missionId;
  if (previousMissionID !== missionId) {
    state.confluenceSearchResults = [];
    state.confluenceSearchContext = null;
    state.confluenceOAuthURL = "";
  }
  state.missionId = missionId;
  clearPendingPoll();
  try {
    state.detail = await api(`/api/missions/${encodeURIComponent(missionId)}`);
    localStorage.setItem(MISSION_STORAGE_KEY, missionId);
    renderDetail();
    renderMissions();
    await loadConfluenceConnections();
    await loadConfluenceAccess();
  } catch (err) {
    state.missionId = previousMissionID;
    renderMissions();
    showError(err);
  }
}

async function reloadMission() {
  if (!state.missionId) return;
  await selectMission(state.missionId);
}

async function sendTurn(event) {
  event.preventDefault();
  if (!requireMission()) return;
  const text = $("turnText").value.trim();
  if (!text) return;
  const missionId = state.missionId;
  state.turnPending = true;
  state.pendingTurn = {
    missionId,
    text,
    agentExecutor: $("agentExecutor").value,
    mcpMode: $("mcpMode").value,
    controllerStrategy: $("controllerStrategy").value,
    createdAt: new Date().toISOString()
  };
  setTurnBusy(true);
  $("turnText").value = "";
  if (state.detail) renderTurns(state.detail.events || []);
  try {
    await api(`/api/missions/${missionId}/turns`, {
      method: "POST",
      body: {
        text,
        agent_executor: $("agentExecutor").value,
        mcp_mode: $("mcpMode").value,
        controller_strategy: $("controllerStrategy").value
      }
    });
    state.pendingTurn = null;
    if (state.missionId === missionId) {
      await reloadMission();
    }
  } catch (err) {
    showError(err);
    state.turnPending = false;
    state.pendingTurn = null;
    setTurnBusy(false);
    if (state.missionId === missionId) {
      $("turnText").value = text;
      if (state.detail) renderTurns(state.detail.events || []);
    }
  }
}

async function cancelTurn() {
  if (!requireMission()) return;
  try {
    await api(`/api/missions/${state.missionId}/turns/cancel`, {
      method: "POST",
      body: {}
    });
    state.pendingTurn = null;
    await reloadMission();
  } catch (err) {
    showError(err);
  }
}

async function startWorkflow() {
  if (!requireMission()) return;
  if (state.workflowPending || state.workflowGoalDraftPending) return;
  const stepInstructionMode = workflowStepInstructionMode();
  const userInstructionRaw = workflowRawInputValue();
  const runGoal = $("workflowRunGoal").value.trim() || userInstructionRaw;
  const instruction = $("workflowStepInstruction").value.trim() || runGoal || userInstructionRaw;
  if (!instruction) {
    showError(new Error("워크플로우 지시문을 입력해야 합니다."));
    return;
  }
  if (state.workflowGoalDraftRaw && state.workflowGoalDraftRaw !== userInstructionRaw && ($("workflowRunGoal").value.trim() || $("workflowStepInstruction").value.trim())) {
    showError(new Error("요청 원문이 목표 초안 생성 이후 변경되었습니다. 목표 초안을 다시 생성하거나 목표/첫 스텝을 비워 직접 시작하세요."));
    return;
  }
  const body = {
    step_instruction_mode: stepInstructionMode,
    instruction,
    agent_executor: $("agentExecutor").value,
    mcp_mode: $("mcpMode").value,
    max_steps: 10,
    max_duration_ms: 1500000,
    stop_condition: "사용자 정지, 최대 단계, 최대 시간, 에이전트 완료 선언 또는 오류"
  };
  body.user_instruction_raw = userInstructionRaw;
  body.run_goal = runGoal;
  state.workflowPending = true;
  setWorkflowBusy(true);
  try {
    await api(`/api/missions/${state.missionId}/workflows`, {
      method: "POST",
      body
    });
    $("workflowInstruction").value = "";
    $("workflowRunGoal").value = "";
    $("workflowStepInstruction").value = "";
    state.workflowGoalDraftRaw = "";
    await reloadMission();
  } catch (err) {
    state.workflowPending = false;
    setWorkflowBusy(false);
    showError(err);
  }
}

function workflowRawInputValue() {
  return $("workflowInstruction").value.trim() || $("turnText").value.trim();
}

function onWorkflowRawInput() {
  const raw = workflowRawInputValue();
  if (state.workflowGoalDraftRaw && raw !== state.workflowGoalDraftRaw) {
    state.workflowGoalDraftRaw = "";
    $("workflowRunGoal").value = "";
    $("workflowStepInstruction").value = "";
  }
}

function workflowStepInstructionMode() {
  return "layered";
}

function updateWorkflowStepInstructionMode() {
  const layered = workflowStepInstructionMode() === "layered";
  $("workflowLayeredFields").classList.toggle("hidden", !layered);
  $("draftWorkflowGoalButton").textContent = state.workflowGoalDraftPending ? "초안 생성 중" : "목표 초안 생성";
  setWorkflowBusy(state.workflowPending);
}

async function draftWorkflowGoal() {
  if (!requireMission()) return;
  if (state.workflowGoalDraftPending || state.workflowPending) return;
  const userInstructionRaw = workflowRawInputValue();
  if (!userInstructionRaw) {
    showError(new Error("자율진행 요청 원문을 입력해야 합니다."));
    return;
  }
  state.workflowGoalDraftPending = true;
  setWorkflowBusy(false);
  const button = $("draftWorkflowGoalButton");
  button.textContent = "초안 생성 중";
  try {
    const response = await api(`/api/missions/${state.missionId}/workflows/goal_draft`, {
      method: "POST",
      body: {
        user_instruction_raw: userInstructionRaw,
        agent_executor: $("agentExecutor").value
      }
    });
    const draft = response.workflow_goal_draft || {};
    const currentRaw = workflowRawInputValue();
    if (currentRaw !== userInstructionRaw) return;
    $("workflowInstruction").value = draft.user_instruction_raw || userInstructionRaw;
    $("workflowRunGoal").value = draft.run_goal || "";
    $("workflowStepInstruction").value = draft.step_instruction || draft.run_goal || "";
    state.workflowGoalDraftRaw = draft.user_instruction_raw || userInstructionRaw;
  } catch (err) {
    showError(err);
  } finally {
    state.workflowGoalDraftPending = false;
    button.textContent = "목표 초안 생성";
    setFormsEnabled(true);
  }
}

async function stopWorkflow() {
  if (!requireMission()) return;
  const run = currentWorkflowRun(state.detail?.workflow_runs || []);
  if (!run?.workflow_run_id) return;
  try {
    await api(`/api/missions/${state.missionId}/workflows/${encodeURIComponent(run.workflow_run_id)}/stop`, {
      method: "POST",
      body: {}
    });
    await reloadMission();
  } catch (err) {
    showError(err);
  }
}

async function continueWorkflowRun(workflowRunID) {
  if (!requireMission()) return;
  if (state.workflowPending || state.reportPending || state.workflowGoalDraftPending) return;
  const run = findWorkflowRun(workflowRunID);
  if (!run) {
    showError(new Error("이어갈 자율진행 기록을 찾을 수 없습니다."));
    return;
  }
  if (!workflowCanContinue(run)) {
    showError(new Error("완료되었거나 진행 중인 자율진행은 이어갈 수 없습니다."));
    return;
  }
  const instruction = workflowContinuationInstruction(run);
  if (!instruction) {
    showError(new Error("이어갈 다음 지시를 찾을 수 없습니다."));
    return;
  }
  const mode = "layered";
  const body = {
    step_instruction_mode: mode,
    instruction,
    agent_executor: run.agent_executor || $("agentExecutor").value,
    mcp_mode: run.mcp_mode || $("mcpMode").value,
    max_steps: Number(run.max_steps || 10),
    max_duration_ms: Number(run.max_duration_ms || 1500000),
    stop_condition: run.stop_condition || "사용자 정지, 최대 단계, 최대 시간, 에이전트 완료 선언 또는 오류",
    continue_from_workflow_run_id: run.workflow_run_id || ""
  };
  body.user_instruction_raw = run.user_instruction_raw || run.instruction || instruction;
  body.run_goal = run.run_goal || run.user_instruction_raw || instruction;
  state.workflowPending = true;
  setWorkflowBusy(true);
  try {
    await api(`/api/missions/${state.missionId}/workflows`, {
      method: "POST",
      body
    });
    await reloadMission();
  } catch (err) {
    state.workflowPending = false;
    setWorkflowBusy(false);
    showError(err);
  }
}

function onWorkflowRunListClick(event) {
  const continueButton = event.target.closest("[data-continue-workflow-id]");
  if (!continueButton) return;
  continueWorkflowRun(continueButton.dataset.continueWorkflowId || "");
}

function findWorkflowRun(workflowRunID) {
  return (state.detail?.workflow_runs || []).find((run) => run.workflow_run_id === workflowRunID) || null;
}

function workflowCanContinue(run) {
  return ["paused", "failed", "interrupted", "stopped"].includes(run?.status);
}

function workflowContinuationInstruction(run) {
  const steps = run?.steps || [];
  const lastStep = steps.length ? steps[steps.length - 1] : null;
  return [
    run?.continuation_instruction,
    lastStep?.next_instruction,
    lastStep?.instruction,
    run?.instruction
  ].map((value) => String(value || "").trim()).find(Boolean) || "";
}

async function resetAgentSession() {
  if (!requireMission()) return;
  const executor = $("agentExecutor").value;
  const selectedModel = selectedAgentModel();
  const selectedReasoningEffort = selectedAgentReasoningEffort();
  const model = state.agentModelTouched ? selectedModel : "";
  const reasoningEffort = state.agentModelTouched || state.agentReasoningEffortTouched ? selectedReasoningEffort : "";
  const modelText = selectedModel ? ` 모델 ${selectedModel}로` : "";
  const effortText = selectedReasoningEffort ? `, 추론 강도 ${selectedReasoningEffort}` : "";
  if (!window.confirm(`${executor}${modelText}${effortText} 세션을 새로 시작할까요? Plasma 미션과 저장된 소스는 유지됩니다.`)) return;
  try {
    await api(`/api/missions/${state.missionId}/agent_sessions/reset`, {
      method: "POST",
      body: { agent_executor: executor, agent_model: model, agent_reasoning_effort: reasoningEffort }
    });
    state.agentModelTouched = false;
    state.agentReasoningEffortTouched = false;
    await reloadMission();
  } catch (err) {
    showError(err);
  }
}

async function addTextSource(event) {
  event.preventDefault();
  if (!requireMission()) return;
  try {
    await api(`/api/missions/${state.missionId}/sources/text`, {
      method: "POST",
      body: {
        title: $("sourceTitle").value,
        external_uri: $("sourceURI").value,
        content: $("sourceContent").value
      }
    });
    $("sourceTitle").value = "";
    $("sourceURI").value = "";
    $("sourceContent").value = "";
    await reloadMission();
  } catch (err) {
    showError(err);
  }
}

async function addUploadSource(event) {
  event.preventDefault();
  if (!requireMission()) return;
  const fileInput = $("sourceUploadFile");
  const file = fileInput.files && fileInput.files[0];
  if (!file) {
    showError(new Error("업로드할 파일을 선택하세요."));
    return;
  }
  const form = new FormData();
  form.append("file", file);
  form.append("title", $("sourceUploadTitle").value.trim());
  try {
    await api(`/api/missions/${state.missionId}/sources/upload`, {
      method: "POST",
      body: form
    });
    fileInput.value = "";
    $("sourceUploadTitle").value = "";
    setReportNotice("업로드한 파일을 원문 소스로 저장했습니다.");
    await reloadMission();
  } catch (err) {
    showError(err);
  }
}

async function addURLSourceFromTextForm() {
  if (!requireMission()) return;
  const url = $("sourceURI").value.trim();
  if (!normalizeSourceURL(url)) {
    showError(new Error("원문 URI에 http 또는 https URL을 입력하세요."));
    return;
  }
  const added = await addURLSource(url, $("sourceTitle").value.trim());
  if (!added) return;
  $("sourceTitle").value = "";
  $("sourceURI").value = "";
  $("sourceContent").value = "";
}

async function addMediaURLSource(event) {
  event.preventDefault();
  if (!requireMission()) return;
  try {
    const result = await api(`/api/missions/${state.missionId}/sources/media_url`, {
      method: "POST",
      body: {
        url: $("mediaSourceURL").value,
        title: $("mediaSourceTitle").value,
        license: $("mediaSourceLicense").value,
        attribution: $("mediaSourceAttribution").value
      }
    });
    $("mediaSourceURL").value = "";
    $("mediaSourceTitle").value = "";
    $("mediaSourceLicense").value = "";
    $("mediaSourceAttribution").value = "";
    const snapshot = result.snapshot || result.Snapshot || {};
    const locator = mediaLocator(snapshot);
    if (locator?.media_kind === "image") {
      setReportNotice("이미지 소스를 저장했습니다. 현재 빌드에서는 이미지 내용 분석 없이 메타데이터와 원본만 사용합니다.");
    } else if (locator?.media_kind) {
      setReportNotice("미디어 소스를 라이브 참조로 저장했습니다. 오디오·영상 inspect는 현재 지원하지 않습니다.");
    }
    await reloadMission();
  } catch (err) {
    showError(err);
  }
}

async function addPDFURLSource(event) {
  event.preventDefault();
  if (!requireMission()) return;
  try {
    await api(`/api/missions/${state.missionId}/sources/pdf_url`, {
      method: "POST",
      body: {
        url: $("pdfSourceURL").value,
        title: $("pdfSourceTitle").value
      }
    });
    $("pdfSourceURL").value = "";
    $("pdfSourceTitle").value = "";
    setReportNotice("PDF 원본을 소스로 저장했습니다. 읽기 요청은 PDF 텍스트 추출 결과를 반환합니다.");
    await reloadMission();
  } catch (err) {
    showError(err);
  }
}

async function loadLocalPathRoots() {
  try {
    const result = await api("/api/local_path/roots");
    state.localPathRoots = result.roots || result.Roots || [];
  } catch (err) {
    state.localPathRoots = [];
  }
  renderLocalPathControls();
}

function renderLocalPathControls() {
  const select = $("localPathRoot");
  if (!select) return;
  if (!state.localPathRoots.length) {
    select.innerHTML = `<option value="">설정된 local source root 없음</option>`;
    select.disabled = true;
    $("localPathTreeButton").disabled = true;
    $("localPathAttachButton").disabled = true;
    $("localPathBreadcrumb").innerHTML = "";
    $("localPathTree").innerHTML = empty("서버에 allowlisted root가 설정되어 있지 않습니다.");
    return;
  }
  const enabled = Boolean(state.missionId);
  select.disabled = !enabled;
  $("localPathTreeButton").disabled = !enabled;
  select.innerHTML = state.localPathRoots.map((root) => {
    const rootID = root.root_id || root.RootID || "";
    const alias = root.alias || root.Alias || rootID;
    return `<option value="${escapeAttr(rootID)}">${escapeHTML(alias)}</option>`;
  }).join("");
  if (!$("localPathTree").innerHTML.trim()) {
    $("localPathTree").innerHTML = empty(enabled ? "‘새로고침’을 누르면 파일을 탐색할 수 있습니다." : "먼저 미션을 선택하세요.");
  }
  updateLocalPathAttachState();
}

function localPathParent(p) {
  const clean = String(p || ".").replace(/\/+$/, "");
  if (!clean || clean === ".") return ".";
  const idx = clean.lastIndexOf("/");
  return idx <= 0 ? "." : clean.slice(0, idx);
}

function updateLocalPathAttachState() {
  const btn = $("localPathAttachButton");
  if (!btn) return;
  const ready = Boolean(state.missionId) && state.localPathRoots.length &&
    Boolean(($("localPathRelativePath").value || "").trim() || state.localPathSelectedFile);
  btn.disabled = !ready;
}

async function browseLocalPathTree() {
  if (!requireMission()) return;
  try {
    const result = await api(`/api/missions/${state.missionId}/sources/local_path/tree`, {
      method: "POST",
      body: {
        root_id: $("localPathRoot").value,
        relative_path: $("localPathRelativePath").value,
        depth: 1,
        limit: 200
      }
    });
    renderLocalPathTree(result.tree || result.Tree);
  } catch (err) {
    showError(err);
  }
}

function renderLocalPathBreadcrumb(current) {
  const bc = $("localPathBreadcrumb");
  if (!bc) return;
  const parts = current && current !== "." ? current.split("/").filter(Boolean) : [];
  const crumbs = [`<span class="local-path-crumb${parts.length ? "" : " current"}" data-local-path-crumb=".">root</span>`];
  let acc = "";
  parts.forEach((seg, i) => {
    acc = acc ? `${acc}/${seg}` : seg;
    crumbs.push(`<span class="local-path-sep">/</span>`);
    crumbs.push(`<span class="local-path-crumb${i === parts.length - 1 ? " current" : ""}" data-local-path-crumb="${escapeAttr(acc)}">${escapeHTML(seg)}</span>`);
  });
  bc.innerHTML = crumbs.join("");
}

function renderLocalPathTree(tree) {
  const container = $("localPathTree");
  if (!container) return;
  const entries = tree?.entries || tree?.Entries || [];
  const truncated = tree?.truncated || tree?.Truncated;
  const current = tree?.relative_path || tree?.RelativePath || ".";
  state.localPathCurrentDir = current || ".";
  renderLocalPathBreadcrumb(state.localPathCurrentDir);
  const atRoot = !current || current === "." || current === "/";
  const up = atRoot ? "" : `
    <button type="button" class="local-path-entry dir" data-local-path-pick=".." data-local-path-kind="up">
      <span class="lp-icon">⬆</span><span class="lp-name">상위 폴더</span>
    </button>`;
  const rows = entries.map((entry) => {
    const rel = entry.relative_path || entry.RelativePath || "";
    const kind = String(entry.path_kind || entry.PathKind || "").toLowerCase();
    const isDir = kind.includes("dir");
    const denied = entry.denied || entry.Denied;
    const name = entry.name || entry.Name || rel;
    const selected = !isDir && rel === state.localPathSelectedFile;
    return `
      <button type="button" class="local-path-entry ${isDir ? "dir" : "file"}${denied ? " denied" : ""}${selected ? " selected" : ""}"
        ${denied ? "disabled" : ""} data-local-path-pick="${escapeAttr(rel)}" data-local-path-kind="${isDir ? "dir" : "file"}" title="${escapeAttr(rel)}">
        <span class="lp-icon">${isDir ? "📁" : "📄"}</span>
        <span class="lp-name">${escapeHTML(name)}</span>
        <span class="lp-meta">${denied ? "접근 불가" : (isDir ? "폴더" : "파일")}</span>
      </button>`;
  }).join("");
  container.innerHTML = `${up}${entries.length ? rows : empty("표시할 항목 없음")}${truncated ? `<div class="local-path-note">일부 항목만 표시됩니다. 하위 폴더로 좁혀보세요.</div>` : ""}`;
}

function onLocalPathBreadcrumbClick(event) {
  const crumb = event.target.closest("[data-local-path-crumb]");
  if (!crumb) return;
  const path = crumb.dataset.localPathCrumb || ".";
  $("localPathRelativePath").value = path === "." ? "" : path;
  state.localPathSelectedFile = "";
  updateLocalPathAttachState();
  browseLocalPathTree();
}

async function attachLocalPathSource(event) {
  event.preventDefault();
  if (!requireMission()) return;
  try {
    await api(`/api/missions/${state.missionId}/sources/local_path`, {
      method: "POST",
      body: {
        root_id: $("localPathRoot").value,
        relative_path: $("localPathRelativePath").value,
        title: $("localPathTitle").value,
        restore: $("localPathRestore").checked
      }
    });
    $("localPathTitle").value = "";
    $("localPathRestore").checked = false;
    state.localPathSelectedFile = "";
    updateLocalPathAttachState();
    await reloadMission();
  } catch (err) {
    showError(err);
  }
}

async function toggleRemovedSources(event) {
  state.showRemovedSources = Boolean(event.target.checked);
  await refreshSourcesOnly();
}

async function refreshSourcesOnly() {
  if (!state.missionId || !state.detail) return;
  try {
    const query = state.showRemovedSources ? "?include_removed=true" : "";
    const result = await api(`/api/missions/${state.missionId}/sources${query}`);
    state.detail.sources = result.sources || result.Sources || [];
    renderDetail();
  } catch (err) {
    showError(err);
  }
}

async function removeSource(snapshotID) {
  if (!requireMission()) return;
  if (!window.confirm("이 소스를 active 사용과 리포트에서 제외할까요? 저장된 artifact나 로컬 파일은 삭제하지 않습니다.")) return;
  try {
    await api(`/api/missions/${state.missionId}/sources/${encodeURIComponent(snapshotID)}/remove`, {
      method: "POST",
      body: { reason: "Removed from Plasma UI" }
    });
    await reloadMission();
    if (state.showRemovedSources) await refreshSourcesOnly();
  } catch (err) {
    showError(err);
  }
}

async function restoreSource(snapshotID) {
  if (!requireMission()) return;
  try {
    await api(`/api/missions/${state.missionId}/sources/${encodeURIComponent(snapshotID)}/restore`, {
      method: "POST",
      body: {}
    });
    await reloadMission();
    if (state.showRemovedSources) await refreshSourcesOnly();
  } catch (err) {
    showError(err);
  }
}

async function readSource(snapshotID) {
  if (!requireMission()) return;
  try {
    const result = await api(`/api/missions/${state.missionId}/sources/${encodeURIComponent(snapshotID)}/read?max_bytes=20000`);
    showDetail("소스 읽기", result);
    await reloadMission();
  } catch (err) {
    showError(err);
  }
}

async function addURLSource(url, title = "") {
  if (!requireMission()) return;
  const key = normalizeSourceURL(url) || url;
  if (state.sourceCandidateBusy.has(key)) return false;
  state.sourceCandidateBusy.add(key);
  refreshSourceCandidates();
  try {
    const route = sourceRouteForURL(url);
    try {
      await api(`/api/missions/${state.missionId}/sources/${route}`, {
        method: "POST",
        body: { url, title }
      });
    } catch (err) {
      if (route !== "url" || !looksLikePDFSourceError(err)) throw err;
      await api(`/api/missions/${state.missionId}/sources/pdf_url`, {
        method: "POST",
        body: { url, title }
      });
    }
    await reloadMission();
    return true;
  } catch (err) {
    showError(err);
    return false;
  } finally {
    state.sourceCandidateBusy.delete(key);
    refreshSourceCandidates();
  }
}

function looksLikePDFSourceError(err) {
  const message = `${err?.userMessage || ""} ${err?.message || ""}`.toLowerCase();
  return message.includes("application/pdf") || message.includes("pdf source") || message.includes("pdf");
}

function sourceRouteForURL(value) {
  if (looksLikeConfluenceURL(value)) return "confluence/url";
  if (looksLikePDFURL(value)) return "pdf_url";
  if (looksLikeMediaURL(value)) return "media_url";
  return "url";
}

function looksLikeConfluenceURL(value) {
  try {
    const url = new URL(value);
    const host = url.hostname.toLowerCase();
    if (host !== "atlassian.net" && !host.endsWith(".atlassian.net")) return false;
    const path = url.pathname.replace(/\/+$/, "");
    return path === "" || path === "/wiki" || path.startsWith("/wiki/");
  } catch (err) {
    return false;
  }
}

function looksLikePDFURL(value) {
  try {
    const url = new URL(value);
    return /\.pdf$/i.test(url.pathname);
  } catch (err) {
    return false;
  }
}

function looksLikeMediaURL(value) {
  try {
    const url = new URL(value);
    const path = url.pathname.toLowerCase();
    return /\.(png|jpe?g|gif|mp3|m4a|wav|ogg|mp4|mov|webm|m4v)$/.test(path);
  } catch (err) {
    return false;
  }
}

async function rejectSourceCandidate(url, reason = null) {
  if (!requireMission()) return;
  const key = normalizeSourceURL(url) || url;
  if (state.sourceCandidateBusy.has(key)) return;
  const rejectionReason = reason === null
    ? window.prompt("기각 사유를 입력하세요. 비워두면 기본 사유로 기록됩니다.", "")
    : reason;
  if (rejectionReason === null) return;
  state.sourceCandidateBusy.add(key);
  refreshSourceCandidates();
  try {
    await api(`/api/missions/${state.missionId}/candidates/sources/reject`, {
      method: "POST",
      body: { url, reason: rejectionReason.trim() }
    });
    await reloadMission();
  } catch (err) {
    showError(err);
  } finally {
    state.sourceCandidateBusy.delete(key);
    refreshSourceCandidates();
  }
}

async function restoreSourceCandidate(url) {
  if (!requireMission()) return;
  const key = normalizeSourceURL(url) || url;
  if (state.sourceCandidateBusy.has(key)) return;
  state.sourceCandidateBusy.add(key);
  refreshSourceCandidates();
  try {
    await api(`/api/missions/${state.missionId}/candidates/sources/restore`, {
      method: "POST",
      body: { url }
    });
    await reloadMission();
  } catch (err) {
    showError(err);
  } finally {
    state.sourceCandidateBusy.delete(key);
    refreshSourceCandidates();
  }
}

async function searchLiquid2(event) {
  event.preventDefault();
  if (!requireMission()) return;
  try {
    const result = await api(`/api/missions/${state.missionId}/sources/liquid2/search`, {
      method: "POST",
      body: { query: $("liquid2Query").value, limit: 8 }
    });
    renderLiquid2Results(result.Candidates || result.candidates || []);
  } catch (err) {
    renderLiquid2Error(err.message);
  }
}

async function attachLiquid2(externalSourceID) {
  if (!requireMission()) return;
  try {
    await api(`/api/missions/${state.missionId}/sources/liquid2/snapshot`, {
      method: "POST",
      body: {
        external_source_id: externalSourceID,
        reason: "Plasma 작업공간에서 선택함"
      }
    });
    $("liquid2Results").innerHTML = "";
    await reloadMission();
  } catch (err) {
    showError(err);
  }
}

async function proposeEvidence(event) {
  event.preventDefault();
  if (!requireMission()) return;
  const summary = $("candidateSummary").value.trim();
  if (!summary) return;
  const sourceValue = $("candidateSource").value;
  if (!sourceValue) {
    showError(new Error("근거 후보를 제안하려면 먼저 소스를 추가하고 선택해야 합니다."));
    return;
  }
  const [snapshotID, artifactID] = sourceValue.split("|");
  try {
    await api(`/api/missions/${state.missionId}/candidates/evidence`, {
      method: "POST",
      body: {
        summary,
        evidence_type: $("candidateEvidenceType").value || "observation",
        snapshot_id: snapshotID,
        artifact_id: artifactID
      }
    });
    $("candidateSummary").value = "";
    await reloadMission();
  } catch (err) {
    showError(err);
  }
}

async function decideProposal(proposalID, action) {
  if (!requireMission()) return;
  try {
    await api(`/api/missions/${state.missionId}/proposals/${proposalID}/${action}`, {
      method: "POST",
      body: {}
    });
    await reloadMission();
  } catch (err) {
    showError(err);
  }
}

async function draftReport(reportMode = "one_take") {
  if (!requireMission()) return;
  if (state.reportPending) return;
  const title = `${state.detail?.projection?.title || "미션"} 리포트`;
  setReportBusy(true);
  setReportNotice(reportPendingMessage({ Payload: { title, report_mode: reportMode, rigor_level: $("reportRigor").value || "balanced" } }));
  let result;
  try {
    result = await api(`/api/missions/${state.missionId}/reports`, {
      method: "POST",
      body: {
        title,
        agent_executor: $("agentExecutor").value,
        mcp_mode: $("mcpMode").value,
        rigor_level: $("reportRigor").value || "balanced",
        report_mode: reportMode
      }
    });
  } catch (err) {
    setReportNotice(`리포트 초안 생성 실패\n\n${err.userMessage || err.message || String(err)}`, "error");
    setReportBusy(false);
    showError(err);
    return;
  }
  setReportNotice(result.pending_event
    ? reportPendingMessage(result.pending_event)
    : reportPendingMessage({ Payload: { title, report_mode: reportMode, rigor_level: $("reportRigor").value || "balanced" } }));
  try {
    await reloadMission();
  } catch (err) {
    showError(err);
    schedulePendingPoll();
  }
}

async function patchReportArtifact(artifactID, currentTitle = "") {
  if (!requireMission()) return;
  if (state.reportPending) return;
  artifactID = (artifactID || "").trim();
  if (!artifactID) return;
  const instruction = window.prompt("이 리포트를 어떻게 수정할까요? 보고서 세션에서 MCP 패치로 새 버전을 만듭니다.", "");
  if (!instruction || !instruction.trim()) return;
  const titleBase = (currentTitle || state.detail?.projection?.title || "리포트").trim();
  const title = `${titleBase} 수정본`;
  const selectedModel = selectedAgentModel();
  const selectedReasoningEffort = selectedAgentReasoningEffort();
  const agentModel = state.agentModelTouched ? selectedModel : "";
  const agentReasoningEffort = state.agentModelTouched || state.agentReasoningEffortTouched ? selectedReasoningEffort : "";
  setReportBusy(true);
  setReportNotice(reportPendingMessage({
    EventType: "report.patch.pending",
    Payload: { title, base_artifact_id: artifactID, instruction: instruction.trim() }
  }));
  let result;
  try {
    result = await api(`/api/missions/${state.missionId}/reports/patch`, {
      method: "POST",
      body: {
        base_artifact_id: artifactID,
        instruction: instruction.trim(),
        title,
        agent_executor: $("agentExecutor").value,
        agent_model: agentModel,
        agent_reasoning_effort: agentReasoningEffort,
        mcp_mode: $("mcpMode").value
      }
    });
  } catch (err) {
    setReportNotice(`리포트 MCP 패치 시작 실패\n\n${err.userMessage || err.message || String(err)}`, "error");
    setReportBusy(false);
    showError(err);
    return;
  }
  setReportNotice(result.pending_event
    ? reportPendingMessage(result.pending_event)
    : reportPendingMessage({ EventType: "report.patch.pending", Payload: { title, base_artifact_id: artifactID } }));
  try {
    await reloadMission();
  } catch (err) {
    showError(err);
    schedulePendingPoll();
  }
}

async function cancelReport() {
  if (!requireMission()) return;
  if (!state.reportPending) return;
  try {
    await api(`/api/missions/${state.missionId}/reports/cancel`, {
      method: "POST",
      body: {}
    });
    setReportNotice("리포트 생성 취소를 요청했습니다. 장부에 취소 이벤트가 기록되면 다시 생성할 수 있습니다.");
    await reloadMission();
  } catch (err) {
    showError(err);
  }
}

async function exportReport(versionID, target, options = {}) {
  const key = `version:${versionID}`;
  if (!options.download) setReportPreviewLoading(key);
  try {
    const result = await api(`/api/report_versions/${versionID}/export`, {
      method: "POST",
      body: { target }
    });
    assertReportExportMatches(versionID, target, result);
    const content = result.content || JSON.stringify(result, null, 2);
    if (options.download) {
      downloadReportExport(result, target, content);
    } else {
      applyReportPreview(key, target === "markdown" ? "markdown" : "text", reportExportPreviewHeader(versionID, target, result), content);
    }
    await reloadMission();
  } catch (err) {
    if (!options.download && state.reportPreview && state.reportPreview.key === key) clearReportPreview();
    showError(err);
  }
}

async function viewReportArtifact(artifactID) {
  if (!state.missionId || !artifactID) return;
  const key = `artifact:${artifactID}`;
  setReportPreviewLoading(key);
  try {
    const result = await api(`/api/missions/${state.missionId}/artifacts/${artifactID}`);
    const content = result.content || "";
    applyReportPreview(key, "markdown", reportArtifactPreviewHeader(artifactID, result), content);
  } catch (err) {
    if (state.reportPreview && state.reportPreview.key === key) clearReportPreview();
    showError(err);
  }
}

async function downloadReportArtifact(artifactID) {
  if (!state.missionId || !artifactID) return;
  try {
    const response = await fetch(`/api/missions/${state.missionId}/artifacts/${artifactID}/download`, {
      headers: { "Accept": "text/markdown, text/plain, */*" }
    });
    if (!response.ok) {
      throw await responseError(response);
    }
    const blob = await response.blob();
    const filename = filenameFromContentDisposition(response.headers.get("Content-Disposition")) || `${artifactID}.md`;
    downloadBlob(blob, filename);
  } catch (err) {
    showError(err);
  }
}

async function exportReportArtifactHTML(artifactID, options = {}) {
  if (!state.missionId || !artifactID) return;
  const key = `artifact:${artifactID}`;
  if (!options.download) setReportPreviewLoading(key);
  try {
    const result = await api(`/api/missions/${state.missionId}/artifacts/${artifactID}/html_export`, {
      method: "POST",
      body: {}
    });
    const content = result.content || "";
    const artifact = result.artifact || {};
    if (options.download) {
      const filename = artifact.filename || artifact.Filename || `${artifactID}.html`;
      const mediaType = artifact.media_type || artifact.MediaType || "text/html;charset=utf-8";
      downloadContent(filename, mediaType, content);
    } else {
      applyReportPreview(key, "html", reportArtifactHTMLPreviewHeader(artifactID, result), content);
    }
    await reloadMission();
  } catch (err) {
    if (!options.download && state.reportPreview && state.reportPreview.key === key) clearReportPreview();
    showError(err);
  }
}

async function exportReportArtifactDesignedHTML(artifactID, options = {}) {
  if (!state.missionId || !artifactID) return;
  const key = `artifact:${artifactID}`;
  if (!options.download) setReportPreviewLoading(key);
  try {
    const result = await api(`/api/missions/${state.missionId}/artifacts/${artifactID}/designed_html_export`, {
      method: "POST",
      body: { agent_executor: $("agentExecutor")?.value || "codex" }
    });
    if (result.status === "pending") {
      setReportNotice("디자인 HTML 리포트를 생성 중입니다. 새로고침해도 진행 상태는 이 미션 장부에서 복구됩니다.");
      if (!options.download && state.reportPreview && state.reportPreview.key === key) clearReportPreview();
      await reloadMission();
      return;
    }
    const content = result.content || "";
    const artifact = result.artifact || {};
    if (options.download) {
      const filename = artifact.filename || artifact.Filename || `${artifactID}-designed.html`;
      const mediaType = artifact.media_type || artifact.MediaType || "text/html;charset=utf-8";
      downloadContent(filename, mediaType, content);
    } else {
      applyReportPreview(key, "html", reportArtifactDesignedHTMLPreviewHeader(artifactID, result), content);
    }
    await reloadMission();
  } catch (err) {
    if (!options.download && state.reportPreview && state.reportPreview.key === key) clearReportPreview();
    showError(err);
  }
}

async function exportReportArtifactHumanizedMarkdown(artifactID) {
  if (!state.missionId || !artifactID) return;
  if (state.reportPending) return;
  const key = `artifact:${artifactID}`;
  setReportBusy(true);
  setReportNotice("H5 말투 보정 Markdown artifact를 생성하는 중입니다. 원본 Markdown 리포트는 그대로 유지됩니다.");
  if (state.reportPreview && state.reportPreview.key === key) clearReportPreview();
  try {
    const result = await api(`/api/missions/${state.missionId}/artifacts/${artifactID}/humanized_markdown_export`, {
      method: "POST",
      body: { mcp_mode: $("mcpMode")?.value || "auto" }
    });
    setReportNotice(result.pending_event
      ? reportPendingMessage(result.pending_event)
      : "H5 말투 보정 Markdown artifact를 생성하는 중입니다.");
    await reloadMission();
  } catch (err) {
    setReportNotice(`H5 말투 보정 시작 실패\n\n${err.userMessage || err.message || String(err)}`, "error");
    setReportBusy(false);
    showError(err);
  }
}

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
  const mediaType = artifact.MediaType || artifact.media_type || exportMediaType(target);
  downloadContent(filename, mediaType, content);
}

function downloadContent(filename, mediaType, content) {
  const blob = new Blob([content], { type: mediaType });
  downloadBlob(blob, filename);
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
  const key = `version:${versionID}`;
  setReportPreviewLoading(key);
  try {
    const result = await api(`/api/report_versions/${versionID}/ast`);
    applyReportPreview(key, "text", "JSON AST", JSON.stringify(result, null, 2));
  } catch (err) {
    if (state.reportPreview && state.reportPreview.key === key) clearReportPreview();
    showError(err);
  }
}

function setSectionEmpty(el, isEmpty) {
  if (!el) return;
  el.classList.toggle("collapsed-empty", isEmpty);
  el.classList.toggle("hidden", isEmpty);
}

function updateCountChip(id, n) {
  const el = $(id);
  if (!el) return;
  el.textContent = n > 0 ? String(n) : "";
}

function renderAgentControlsSummary() {
  const summaryText = $("agentControlsSummaryText");
  if (!summaryText) return;
  const executor = $("agentExecutor")?.value || "codex";
  const strategy = $("controllerStrategy")?.value || "auto";
  const locked = lockedAgentExecutor();
  const statusEl = $("agentSessionStatus");
  const statusText = statusEl ? statusEl.textContent.trim() : "";
  const selectedModel = selectedAgentModel();
  const model = state.agentModelTouched ? selectedModel : (selectedModel || currentAgentModel(state.detail?.events || [], executor));
  const selectedReasoningEffort = selectedAgentReasoningEffort();
  const status = agentExecutorStatus(executor);
  const reasoningEffort = state.agentReasoningEffortTouched
    ? selectedReasoningEffort
    : (selectedReasoningEffort || currentAgentReasoningEffort(state.detail?.events || [], executor) || status?.default_reasoning_effort || "");
  const strategyText = strategy === "auto" ? "자동 조향" : strategy.toUpperCase();
  const lockText = locked ? "미션 에이전트 고정" : "에이전트 선택 가능";
  const modelText = model ? `모델 ${model}` : "기본 모델";
  const effortText = status?.reasoning_effort_supported === false
    ? "추론 지정 불가"
    : `추론 ${reasoningEffort || "medium"}`;
  summaryText.textContent = statusText
    ? `${executor} · ${modelText} · ${effortText} · ${strategyText} · ${lockText} · ${statusText}`
    : `${executor} · ${modelText} · ${effortText} · ${strategyText} · ${lockText}`;
}

function renderMissions() {
  const list = $("missionList");
  const n = state.missions.length;
  updateCountChip("missionListCount", n);
  if (n === 0) {
    list.innerHTML = empty("미션 없음");
    return;
  }
  list.innerHTML = state.missions.map((mission) => `
    <button class="item secondary ${mission.MissionID === state.missionId ? "active" : ""}"
      type="button" data-mission-id="${escapeAttr(mission.MissionID)}" title="${escapeAttr(mission.Title || mission.MissionID)}">
      <div class="item-title">${escapeHTML(mission.Title || mission.MissionID)}</div>
      <div class="item-meta" title="${escapeAttr(mission.MissionID)}">${escapeHTML(mission.MissionID)}</div>
    </button>
  `).join("");
}

function renderDetail() {
  const detail = state.detail;
  if (!detail) return;
  const events = detail.events || [];
  const reportDraft = reportDraftState(events);
  const workflowRuns = detail.workflow_runs || [];
  const wasReportPending = state.reportPending;
  state.turnPending = hasOpenPendingTurn(events);
  state.reportPending = reportDraft.state === "pending";
  state.workflowPending = workflowRuns.some((run) => ["queued", "running", "stopping"].includes(run.status));
  setFormsEnabled(Boolean(detail));
  renderLocalPathControls();
  renderConfluenceControls();
  renderConfluenceResults(state.confluenceSearchResults);
  renderAgentOptions(detail.agent_executors || []);
  renderAgentModelOptions(events);
  renderAgentReasoningEffortOptions(events);
  renderAgentSessionStatus(events);
  renderWorkflowControls(workflowRuns);
  $("missionName").textContent = detail.projection.title || detail.projection.mission_id;
  $("missionObjectiveText").textContent = detail.projection.objective || "목표 없음";

  const sources = detail.sources || [];
  const records = detail.records || {};
  const proposals = records.proposals || [];
  const savedEvidence = approvedEvidence(proposals, records.evidence || []);
  const savedClaims = approvedClaims(proposals, records.claims || []);
  $("sourceCount").textContent = `소스 ${sources.length}개`;
  if ($("includeRemovedSources")) $("includeRemovedSources").checked = state.showRemovedSources;
  $("candidateCount").textContent = `후보 ${proposals.filter((p) => p.state === "pending_review").length}개`;
  $("savedCount").textContent = `저장 ${savedEvidence.length + savedClaims.length}개`;

  renderTurns(events);
  renderSources(sources);
  renderSourceCandidates(events, sources);
  renderCandidateSourceOptions(sources);
  renderProposals(proposals, records);
  renderSavedEvidence(savedEvidence);
  renderClaimConfidenceChanges(records.claim_confidence || [], records.claims || []);
  renderSavedClaims(savedClaims, records.claim_confidence || []);
  renderReports(detail.report_versions || []);
  renderReportDraftStatus(reportDraft, wasReportPending);
  renderLedger(events);
  schedulePendingPoll();
}

function renderAgentOptions(statuses) {
  const select = $("agentExecutor");
  if (!select) return;
  const current = select.value || "codex";
  const locked = lockedAgentExecutor();
  const known = statuses.length ? statuses : [
    { name: "codex", label: "Codex", configured: true },
    { name: "claude", label: "Claude", configured: false }
  ];
  select.innerHTML = known.map((status) => {
    const lockedOut = locked && status.name !== locked;
    const disabled = status.configured && !lockedOut ? "" : "disabled";
    const lockLabel = locked && status.name === locked ? " · 이 미션에서 사용 중" : "";
    const label = `${status.label || status.name}${status.configured ? "" : " 준비 중"}${lockLabel}`;
    return `<option value="${escapeAttr(status.name)}" ${disabled}>${escapeHTML(label)}</option>`;
  }).join("");
  if (locked && [...select.options].some((option) => option.value === locked)) {
    select.value = locked;
    return;
  }
  if ([...select.options].some((option) => option.value === current && !option.disabled)) {
    select.value = current;
    return;
  }
  const firstAvailable = [...select.options].find((option) => !option.disabled);
  if (firstAvailable) {
    select.value = firstAvailable.value;
  }
}

function onAgentExecutorChange() {
  state.agentModelTouched = false;
  state.agentReasoningEffortTouched = false;
  renderAgentModelOptions(state.detail?.events || []);
  renderAgentReasoningEffortOptions(state.detail?.events || []);
  const effortSelect = $("agentReasoningEffort");
  if (effortSelect) {
    const blocked = state.turnPending || state.workflowPending || state.workflowGoalDraftPending || state.reportPending || !state.detail;
    effortSelect.disabled = agentReasoningEffortSelectionDisabled(blocked);
  }
  renderAgentSessionStatus(state.detail?.events || []);
  renderAgentControlsSummary();
}

function onAgentModelChange() {
  state.agentModelTouched = true;
  renderAgentReasoningEffortOptions(state.detail?.events || [], true);
  renderAgentControlsSummary();
}

function onAgentReasoningEffortChange() {
  state.agentReasoningEffortTouched = true;
  renderAgentControlsSummary();
}

function renderAgentModelOptions(events) {
  const select = $("agentModel");
  if (!select) return;
  const executor = $("agentExecutor")?.value || "codex";
  if (state.agentModelExecutor !== executor) {
    state.agentModelTouched = false;
    state.agentModelExecutor = executor;
  }
  const status = agentExecutorStatus(executor);
  const options = agentModelOptions(executor, status);
  const saved = currentAgentModel(events || [], executor);
  const preferred = state.agentModelTouched ? select.value : saved;
  select.innerHTML = options.map((option) =>
    `<option value="${escapeAttr(option.value)}">${escapeHTML(agentModelOptionLabel(option, status))}</option>`
  ).join("");
  if (preferred && !options.some((option) => option.value === preferred)) {
    select.insertAdjacentHTML(
      "beforeend",
      `<option value="${escapeAttr(preferred)}">${escapeHTML(`저장된 모델: ${preferred}`)}</option>`
    );
  }
  select.value = preferred || "";
}

function agentModelOptions(executor, status) {
  if (executor === "codex") {
    const catalog = Array.isArray(status?.models) ? status.models : [];
    return [{ value: "", label: "기본값" }, ...catalog.map((model) => ({
      value: String(model.name || "").trim(),
      label: String(model.label || model.name || "").trim()
    })).filter((model) => model.value)];
  }
  return AGENT_MODEL_OPTIONS[executor] || [{ value: "", label: "기본값" }];
}

function agentModelOptionLabel(option, status) {
  if (option.value !== "") return option.label;
  const defaultText = agentDefaultModelText(status);
  return defaultText ? `기본값 (${defaultText})` : option.label;
}

function agentDefaultModelText(status) {
  if (!status) return "";
  const label = String(status.default_model_label || "").trim();
  const version = String(status.default_model_version || status.default_model || "").trim();
  if (label && version && label !== version) return `${label}, ${version}`;
  return label || version;
}

function renderAgentReasoningEffortOptions(events, modelChanged = false) {
  const select = $("agentReasoningEffort");
  if (!select) return;
  const executor = $("agentExecutor")?.value || "codex";
  if (state.agentReasoningEffortExecutor !== executor) {
    state.agentReasoningEffortTouched = false;
    state.agentReasoningEffortExecutor = executor;
  }
  const status = agentExecutorStatus(executor);
  const options = agentReasoningEffortOptions(executor, status);
  const saved = currentAgentReasoningEffort(events || [], executor);
  const supported = agentReasoningEffortSupported(executor, status);
  const defaultEffort = String(status?.default_reasoning_effort || "medium").trim();
  const defaultSelection = options.some((option) => option.value === defaultEffort) ? defaultEffort : (options[0]?.value || "");
  const preferred = supported
    ? (modelChanged ? defaultSelection : (state.agentReasoningEffortTouched ? select.value : (saved || defaultSelection)))
    : "";
  select.innerHTML = options.map((option) =>
    `<option value="${escapeAttr(option.value)}">${escapeHTML(option.label)}</option>`
  ).join("");
  if (!modelChanged && preferred && !options.some((option) => option.value === preferred)) {
    select.insertAdjacentHTML(
      "beforeend",
      `<option value="${escapeAttr(preferred)}">${escapeHTML(`저장된 추론 강도: ${preferred}`)}</option>`
    );
  }
  select.value = preferred;
  select.title = supported ? "" : (status?.reasoning_effort_note || "이 에이전트는 추론 강도 지정을 지원하지 않습니다.");
}

function agentReasoningEffortOptions(executor, status) {
  if (executor === "codex") {
    const model = selectedAgentModel() || String(status?.default_model || "").trim();
    const catalog = Array.isArray(status?.models) ? status.models : [];
    const selected = catalog.find((item) => String(item.name || "").trim() === model);
    const efforts = Array.isArray(selected?.reasoning_efforts) ? selected.reasoning_efforts : ["low", "medium", "high", "xhigh"];
    return efforts.map((effort) => ({ value: String(effort), label: String(effort).replace(/^./, (letter) => letter.toUpperCase()) }));
  }
  return AGENT_REASONING_EFFORT_OPTIONS[executor] || [{ value: "", label: "지정 불가" }];
}

function agentExecutorStatus(executor) {
  const statuses = state.detail?.agent_executors || [];
  return statuses.find((status) => status.name === executor) || null;
}

function lockedAgentExecutor() {
  return (state.detail?.locked_agent_executor || "").trim();
}

function agentExecutorSelectionDisabled(baseDisabled) {
  return baseDisabled || Boolean(lockedAgentExecutor());
}

function agentReasoningEffortSelectionDisabled(baseDisabled) {
  const executor = $("agentExecutor")?.value || "codex";
  return baseDisabled || !agentReasoningEffortSupported(executor, agentExecutorStatus(executor));
}

function agentReasoningEffortSupported(executor, status) {
  if (status && typeof status.reasoning_effort_supported === "boolean") {
    return status.reasoning_effort_supported;
  }
  return executor === "codex";
}

function selectedAgentModel() {
  return ($("agentModel")?.value || "").trim();
}

function selectedAgentReasoningEffort() {
  const executor = $("agentExecutor")?.value || "codex";
  const status = agentExecutorStatus(executor);
  if (!agentReasoningEffortSupported(executor, status)) return "";
  return ($("agentReasoningEffort")?.value || status?.default_reasoning_effort || "medium").trim() || "medium";
}

function currentAgentModel(events, executor) {
  for (let index = events.length - 1; index >= 0; index--) {
    const event = events[index];
    const payload = event.Payload || {};
    if (!agentEventMatchesExecutor(payload.agent_executor || "", executor)) continue;
    if (event.EventType === "agent.session.reset") {
      return (payload.agent_model || "").trim();
    }
    if (event.EventType === "turn.agent.response" && payload.kind === "agent_response" && payload.agent_model) {
      return String(payload.agent_model || "").trim();
    }
  }
  return "";
}

function currentAgentReasoningEffort(events, executor) {
  for (let index = events.length - 1; index >= 0; index--) {
    const event = events[index];
    const payload = event.Payload || {};
    if (!agentEventMatchesExecutor(payload.agent_executor || "", executor)) continue;
    if (event.EventType === "agent.session.reset") {
      return (payload.agent_reasoning_effort || "").trim();
    }
    if (event.EventType === "turn.agent.response" && payload.kind === "agent_response" && payload.agent_reasoning_effort) {
      return String(payload.agent_reasoning_effort || "").trim();
    }
  }
  return "";
}

function renderTurns(events) {
  const turns = events.filter((event) =>
    event.EventType === "turn.user" ||
    event.EventType === "controller.strategy.selected" ||
    event.EventType === "turn.agent.response" ||
    event.EventType === "turn.agent.pending" ||
    event.EventType === "agent.session.reset"
  );
  const completed = completedUserEventIDs(events);
  const html = turns.map((event) => {
    const payload = event.Payload || {};
    const isUser = event.EventType === "turn.user";
    const isPending = event.EventType === "turn.agent.pending";
    const isSessionReset = event.EventType === "agent.session.reset";
    if (isPending && completed.has(payload.user_event_id || "")) {
      return "";
    }
    if (isSessionReset) {
      const executor = payload.agent_executor || "agent";
      const model = payload.agent_model || "";
      const reasoningEffort = payload.agent_reasoning_effort || "";
      const previousID = payload.previous_agent_session_id || "";
      const previousBadge = previousID ? `
        <span class="badge muted" title="${escapeAttr(previousID)}">이전 ${escapeHTML(shortID(previousID))}</span>
        <button type="button" class="mini-copy" data-copy-text="${escapeAttr(previousID)}">이전 ID 복사</button>
      ` : "";
      const modelBadge = model ? `<span class="badge muted" title="${escapeAttr(model)}">모델 ${escapeHTML(model)}</span>` : "";
      const effortBadge = reasoningEffort ? `<span class="badge muted">추론 ${escapeHTML(reasoningEffort)}</span>` : "";
      const resetText = previousID
        ? `${executor} 새 세션이 준비되었습니다. 다음 메시지는 이전 세션을 resume하지 않습니다.`
        : `${executor} 새 세션이 준비되었습니다. 다음 메시지는 새 세션으로 시작됩니다.`;
      return `
        <div class="turn session-event">
          <div class="turn-label">세션 / ${escapeHTML(timeShort(event.CreatedAt))} <span class="badge session-new">${escapeHTML(executor)} 새 세션 준비</span> ${modelBadge} ${effortBadge} ${previousBadge}</div>
          <div class="turn-text">${escapeHTML(resetText)}</div>
        </div>
      `;
    }
    if (event.EventType === "controller.strategy.selected") {
      const label = payload.strategy_label || payload.strategy_id || "조향 전략";
      const reason = payload.reason || "";
      const strategyID = payload.strategy_id ? `<span class="badge muted">${escapeHTML(payload.strategy_id)}</span>` : "";
      return `
        <div class="turn controller">
          <div class="turn-label">조향 / ${escapeHTML(timeShort(event.CreatedAt))} ${strategyID}</div>
          <div class="turn-text"><strong>${escapeHTML(label)}</strong>${reason ? ` · ${escapeHTML(reason)}` : ""}</div>
        </div>
      `;
    }
    const text = payload.text || JSON.stringify(payload);
    const executor = payload.agent_executor || (isUser ? "" : "agent");
    const isWorkflowTurn = payload.workflow_run_id || payload.kind === "workflow_steering";
    const sessionID = payload.agent_session_id || "";
    const isNewSession = !isUser && !isPending && payload.kind === "agent_response" && payload.resumed === false && sessionID;
    const sessionBadge = sessionID ? `
      <span class="badge ${isNewSession ? "session-new" : "muted"}" title="${escapeAttr(sessionID)}">${escapeHTML(executor)} ${isNewSession ? "새 세션 " : ""}${escapeHTML(shortID(sessionID))}</span>
      <button type="button" class="mini-copy" data-copy-text="${escapeAttr(sessionID)}">세션 ID 복사</button>
    ` : (executor ? `<span class="badge muted">${escapeHTML(executor)}</span>` : "");
    const body = isPending
      ? `<div class="turn-text"><span class="spinner"></span> ${escapeHTML("에이전트 응답을 기다리는 중...")}</div>`
      : isUser
        ? `<div class="turn-text">${escapeHTML(text)}</div>`
        : `<div class="turn-text turn-markdown">${renderMarkdown(text)}</div>`;
    const copyButton = isPending
      ? ""
      : `<button type="button" class="mini-copy turn-copy" data-copy-text="${escapeAttr(text)}" title="이 메시지 복사">복사</button>`;
    return `
      <div class="turn ${isUser ? "user" : "agent"} ${isPending ? "pending" : ""}">
        <div class="turn-label">${isWorkflowTurn ? "워크플로우" : (isUser ? "사용자" : "에이전트")} / ${escapeHTML(timeShort(event.CreatedAt))} ${sessionBadge}${copyButton}</div>
        ${body}
      </div>
    `;
  }).filter(Boolean);
  if (state.pendingTurn && state.pendingTurn.missionId === state.missionId) {
    html.push(`
      <div class="turn user pending">
        <div class="turn-label">사용자 / ${escapeHTML(timeShort(state.pendingTurn.createdAt))}</div>
        <div class="turn-text">${escapeHTML(state.pendingTurn.text)}</div>
      </div>
    `);
  }
  if (state.turnPending && state.pendingTurn && state.pendingTurn.missionId === state.missionId) {
    html.push(`
      <div class="turn pending">
        <div class="turn-label">에이전트</div>
        <div class="turn-text"><span class="spinner"></span> 에이전트 응답을 기다리는 중...</div>
      </div>
    `);
  }
  const log = $("turnLog");
  // Only auto-scroll when the user is already near the bottom (or just switched
  // missions); otherwise polling re-renders would yank them away while reading.
  const missionChanged = state.turnScrollMission !== state.missionId;
  const nearBottom = log.scrollHeight - log.scrollTop - log.clientHeight < 80;
  log.innerHTML = html.length ? html.join("") : empty("아직 대화가 없습니다.");
  if (missionChanged || nearBottom) log.scrollTop = log.scrollHeight;
  state.turnScrollMission = state.missionId;
  updateTurnNavVisibility();
}

function updateTurnNavVisibility() {
  const log = $("turnLog");
  const nav = $("turnNav");
  if (!log || !nav) return;
  const scrollable = log.scrollHeight > log.clientHeight + 24;
  nav.classList.toggle("hidden", !scrollable);
}

let turnStepTimer = 0;
let turnStepIndex = -1;
let turnHoldActive = false;
let turnHoldDir = null;
const TURN_STEP_GAP_MS = 140;

function onTurnNavClick(event) {
  const button = event.target.closest("[data-turn-nav]");
  if (!button) return;
  const dir = button.dataset.turnNav;
  if (dir === "top" || dir === "bottom") { turnNavScroll(dir); return; }
  // up/down via mouse/touch are handled by press-and-hold (pointerdown);
  // only keyboard-activated clicks (detail === 0) take a single step here.
  if (event.detail === 0) turnNavScroll(dir);
}

function onTurnNavPointerDown(event) {
  const button = event.target.closest("[data-turn-nav]");
  if (!button) return;
  const dir = button.dataset.turnNav;
  if (dir !== "up" && dir !== "down") return;
  event.preventDefault();
  startTurnStep(dir);
  try { button.setPointerCapture?.(event.pointerId); } catch { /* synthetic pointer */ }
}

function turnOffsets(log, turns) {
  const logTop = log.getBoundingClientRect().top;
  return turns.map((t) => log.scrollTop + (t.getBoundingClientRect().top - logTop));
}

function nearestTurnIndex(log, offsets) {
  const cur = log.scrollTop;
  let idx = 0;
  for (let i = 0; i < offsets.length; i += 1) {
    if (offsets[i] <= cur + 2) idx = i; else break;
  }
  return idx;
}

// Smoothly scroll to a position, then run `done` once it settles (with a
// fallback in case `scrollend` is unsupported or no scroll actually happens).
function smoothScrollThen(log, top, done) {
  let finished = false;
  const finish = () => {
    if (finished) return;
    finished = true;
    log.removeEventListener("scrollend", finish);
    clearTimeout(fallback);
    done();
  };
  log.addEventListener("scrollend", finish, { once: true });
  const fallback = setTimeout(finish, 700);
  log.scrollTo({ top: Math.max(0, Math.round(top)), behavior: "smooth" });
}

// One smooth step to the next/previous turn. While held, the next step starts
// only after this one finishes (a brief pause between), giving a stepped glide.
function turnStepOnce(direction) {
  const log = $("turnLog");
  if (!log || !turnHoldActive) return;
  const turns = [...log.querySelectorAll(".turn")];
  if (!turns.length) { stopTurnStep(); return; }
  const offsets = turnOffsets(log, turns);
  if (turnStepIndex < 0) turnStepIndex = nearestTurnIndex(log, offsets);
  const prev = turnStepIndex;
  turnStepIndex = Math.max(0, Math.min(turns.length - 1, turnStepIndex + (direction === "up" ? -1 : 1)));
  if (turnStepIndex === prev) { stopTurnStep(); return; } // reached an edge
  smoothScrollThen(log, offsets[turnStepIndex], () => {
    if (!turnHoldActive) return;
    turnStepTimer = setTimeout(() => turnStepOnce(direction), TURN_STEP_GAP_MS);
  });
}

function startTurnStep(direction) {
  stopTurnStep();
  const log = $("turnLog");
  if (!log) return;
  turnHoldActive = true;
  turnHoldDir = direction;
  turnStepIndex = nearestTurnIndex(log, turnOffsets(log, [...log.querySelectorAll(".turn")]));
  turnStepOnce(direction); // first step immediately (also handles a quick tap)
}

function stopTurnStep() {
  turnHoldActive = false;
  turnHoldDir = null;
  if (turnStepTimer) {
    clearTimeout(turnStepTimer);
    turnStepTimer = 0;
  }
  turnStepIndex = -1;
}

// Single smooth step (keyboard) or absolute jump (top/bottom).
function turnNavScroll(direction) {
  const log = $("turnLog");
  if (!log) return;
  if (direction === "top") { log.scrollTo({ top: 0, behavior: "smooth" }); return; }
  if (direction === "bottom") { log.scrollTo({ top: log.scrollHeight, behavior: "smooth" }); return; }
  const turns = [...log.querySelectorAll(".turn")];
  if (!turns.length) return;
  const offsets = turnOffsets(log, turns);
  const cur = log.scrollTop;
  let target;
  if (direction === "up") {
    target = [...offsets].reverse().find((o) => o < cur - 2);
    if (target == null) target = 0;
  } else {
    target = offsets.find((o) => o > cur + 2);
    if (target == null) target = log.scrollHeight;
  }
  log.scrollTo({ top: Math.max(0, Math.round(target)), behavior: "smooth" });
}

function renderAgentSessionStatus(events) {
  const el = $("agentSessionStatus");
  if (!el) return;
  const executor = $("agentExecutor").value || "codex";
  const model = currentAgentModel(events, executor);
  const modelText = model ? ` · 모델 ${model}` : "";
  const reasoningEffort = currentAgentReasoningEffort(events, executor);
  const effortText = reasoningEffort ? ` · 추론 ${reasoningEffort}` : "";
  el.classList.remove("ready", "live");
  el.removeAttribute("title");
  for (let index = events.length - 1; index >= 0; index--) {
    const event = events[index];
    const payload = event.Payload || {};
    if (!agentEventMatchesExecutor(payload.agent_executor || "", executor)) continue;
    if (event.EventType === "agent.session.reset") {
      const previousID = payload.previous_agent_session_id || "";
      el.textContent = previousID
        ? `${executor} 새 세션 준비됨${modelText}${effortText} · 이전 ${shortID(previousID)}`
        : `${executor} 새 세션 준비됨${modelText}${effortText}`;
      if (previousID || model || reasoningEffort) el.title = [model ? `모델: ${model}` : "", reasoningEffort ? `추론 강도: ${reasoningEffort}` : "", previousID ? `이전 세션: ${previousID}` : ""].filter(Boolean).join("\n");
      el.classList.add("ready");
      renderAgentControlsSummary();
      return;
    }
    if (event.EventType === "turn.agent.response" && payload.kind === "agent_response" && payload.agent_session_id) {
      const isNew = payload.resumed === false;
      el.textContent = `${executor} ${isNew ? "새 세션" : "현재 세션"}${modelText}${effortText} · ${shortID(payload.agent_session_id)}`;
      el.title = [model ? `모델: ${model}` : "", reasoningEffort ? `추론 강도: ${reasoningEffort}` : "", `현재 세션: ${payload.agent_session_id}`].filter(Boolean).join("\n");
      el.classList.add("live");
      renderAgentControlsSummary();
      return;
    }
  }
  el.textContent = `${executor} 세션 없음`;
  renderAgentControlsSummary();
}

function renderWorkflowControls(runs) {
  const badge = $("workflowStatusBadge");
  const list = $("workflowRunList");
  const active = currentWorkflowRun(runs);
  const latest = active || runs[runs.length - 1] || null;
  if (badge) {
    badge.className = `badge ${latest ? workflowStatusClass(latest.status) : "muted"}`;
    badge.textContent = latest ? workflowStatusLabel(latest.status) : "대기 없음";
    badge.title = latest?.status_text || "";
  }
  if (list) {
    const recent = runs.slice(-50).reverse();
    list.innerHTML = recent.length ? recent.map(renderWorkflowRun).join("") : "";
  }
  const countChip = $("workflowRunCount");
  if (countChip) countChip.textContent = runs.length ? String(runs.length) : "";
  setWorkflowBusy(Boolean(active));
}

function renderWorkflowRun(run) {
  const steps = run.steps || [];
  const mode = run.step_instruction_mode === "layered" ? "layered" : "current";
  const modeLabel = mode === "layered" ? "3층 지시" : "이전 방식 기록";
  const stepsHTML = steps.length
    ? steps.slice(-50).map((step) => {
        const decision = step.decision ? `<span class="wf-step-tag">${escapeHTML(workflowDecisionLabel(step.decision))}</span>` : "";
        const reason = step.reason ? `<span class="wf-step-reason">${escapeHTML(step.reason)}</span>` : "";
        return `
          <div class="wf-step ${workflowStepDotClass(step.status)}">
            <span class="wf-step-dot"></span>
            <span class="wf-step-id">${escapeHTML(step.workflow_step_id || "step")}</span>
            <span class="wf-step-status">${escapeHTML(workflowStatusLabel(step.status))}</span>
            ${decision}${reason}
          </div>`;
      }).join("")
    : `<div class="wf-step wf-dot-muted"><span class="wf-step-dot"></span><span class="wf-step-status">단계 기록 없음</span></div>`;
  const field = (key, value) => value
    ? `<div class="wf-field"><span class="wf-k">${key}</span><span class="wf-v">${escapeHTML(value)}</span></div>`
    : "";
  const metaHTML = [
    field("원문", mode === "layered" ? run.user_instruction_raw : ""),
    field("목표", mode === "layered" ? run.run_goal : ""),
    field("이어받음", run.continue_from_workflow_run_id ? shortID(run.continue_from_workflow_run_id) : ""),
    field("정지 사유", run.stop_reason),
    field("다음 조사", run.continuation_instruction),
  ].join("");
  const continueDisabled = state.workflowPending || state.reportPending || state.workflowGoalDraftPending ? "disabled" : "";
  const continueAction = workflowCanContinue(run)
    ? `<button type="button" class="secondary" data-continue-workflow-id="${escapeAttr(run.workflow_run_id || "")}" ${continueDisabled}>이어서 진행</button>`
    : "";
  return `
    <div class="workflow-run">
      <div class="workflow-run-head">
        <span class="wf-run-id">${escapeHTML(shortID(run.workflow_run_id || ""))}</span>
        <span class="badge ${workflowStatusClass(run.status)}">${escapeHTML(workflowStatusLabel(run.status))}</span>
        <span class="badge muted">${escapeHTML(modeLabel)}</span>
        <span class="wf-run-count">완료 ${Number(run.completed_step_count || 0)}단계${run.latest_event_id ? ` · ${escapeHTML(shortID(run.latest_event_id))}` : ""}</span>
        ${continueAction}
      </div>
      ${metaHTML ? `<div class="workflow-run-meta">${metaHTML}</div>` : ""}
      <div class="workflow-steps">${stepsHTML}</div>
    </div>
  `;
}

function workflowStepDotClass(status) {
  switch (status) {
    case "completed": return "wf-dot-done";
    case "running":
    case "stopping": return "wf-dot-active";
    case "queued":
    case "paused": return "wf-dot-wait";
    case "failed":
    case "interrupted": return "wf-dot-fail";
    case "stopped": return "wf-dot-stopped";
    default: return "wf-dot-muted";
  }
}

function currentWorkflowRun(runs) {
  for (let index = runs.length - 1; index >= 0; index -= 1) {
    if (["queued", "running", "stopping"].includes(runs[index].status)) return runs[index];
  }
  return null;
}

function workflowStatusLabel(status) {
  switch (status) {
    case "queued": return "대기";
    case "running": return "진행 중";
    case "stopping": return "정지 중";
    case "completed": return "완료";
    case "paused": return "추가 진행 필요";
    case "stopped": return "정지됨";
    case "failed": return "실패";
    case "interrupted": return "중단";
    default: return status || "상태 없음";
  }
}

function workflowStatusClass(status) {
  switch (status) {
    case "running": return "session-new";
    case "queued":
    case "stopping": return "warn";
    case "paused": return "warn";
    case "failed":
    case "interrupted": return "danger";
    default: return "muted";
  }
}

function workflowDecisionLabel(decision) {
  switch (decision) {
    case "continue": return "계속";
    case "stop": return "완료 선언";
    default: return decision || "";
  }
}

function renderProposalExtractionStatus(status) {
  if (!status || typeof status !== "object") return "";
  if (status.error) {
    return `<div class="turn-note warn">후보 생성 실패: ${escapeHTML(status.error)}</div>`;
  }
  if (status.created_proposals) {
    return `<div class="turn-note">소스 기반 후보가 검토 대기 목록에 추가되었습니다.</div>`;
  }
  if (status.attempted) {
    return `<div class="turn-note muted">소스 기반으로 저장할 후보를 찾지 못했습니다.</div>`;
  }
  switch (status.reason) {
    case "no_sources":
      return `<div class="turn-note muted">저장된 소스가 없어 근거 후보를 만들지 않았습니다.</div>`;
    case "explicit_mode":
      return `<div class="turn-note muted">명시 요청 모드라 자동 후보 생성을 건너뛰었습니다.</div>`;
    case "no_agent_session":
      return `<div class="turn-note warn">에이전트 세션 ID가 없어 후보 생성을 건너뛰었습니다.</div>`;
    default:
      return "";
  }
}

function agentEventMatchesExecutor(eventExecutor, executor) {
  const eventName = String(eventExecutor || "").trim();
  const executorName = String(executor || "codex").trim();
  if (!eventName) return executorName === "codex";
  return eventName === executorName;
}

function hasOpenPendingTurn(events) {
  const completed = completedUserEventIDs(events);
  return events.some((event) => {
    if (event.EventType !== "turn.agent.pending") return false;
    const userEventID = event.Payload?.user_event_id || "";
    return userEventID && !completed.has(userEventID);
  });
}

function completedUserEventIDs(events) {
  const completed = new Set();
  for (const event of events) {
    if (event.EventType !== "turn.agent.response") continue;
    const userEventID = event.Payload?.user_event_id || "";
    if (userEventID) completed.add(userEventID);
  }
  return completed;
}

function reportDraftState(events) {
  const completed = completedReportDraftPendingEventIDs(events);
  for (let index = events.length - 1; index >= 0; index -= 1) {
    const event = events[index];
    if (event.EventType !== "report.draft.pending" && event.EventType !== "report.design.pending" && event.EventType !== "report.humanize.pending" && event.EventType !== "report.patch.pending") continue;
    if (!completed.has(event.EventID)) {
      return { state: "pending", event };
    }
  }
  for (let index = events.length - 1; index >= 0; index -= 1) {
    const event = events[index];
    if (event.EventType === "report.draft.failed" || event.EventType === "report.design.failed" || event.EventType === "report.humanize.failed" || event.EventType === "report.patch.failed") {
      return { state: "failed", event };
    }
    if (event.EventType === "report.humanize.skipped") {
      return { state: "skipped", event };
    }
    if (event.EventType === "report.drafted" || event.EventType === "report.artifact.created" || event.EventType === "report.artifact.exported") {
      return { state: "completed", event };
    }
  }
  return { state: "idle", event: null };
}

function completedReportDraftPendingEventIDs(events) {
  const completed = new Set();
  for (const event of events) {
    if (event.EventType === "report.drafted" || event.EventType === "report.artifact.created" || event.EventType === "report.artifact.exported") {
      const pendingEventID = event.Payload?.pending_event_id || event.Payload?.generation?.pending_event_id || "";
      if (pendingEventID) completed.add(pendingEventID);
    }
    if (event.EventType === "report.draft.failed" || event.EventType === "report.design.failed" || event.EventType === "report.humanize.failed" || event.EventType === "report.humanize.skipped" || event.EventType === "report.patch.failed") {
      const pendingEventID = event.Payload?.pending_event_id || "";
      if (pendingEventID) completed.add(pendingEventID);
    }
  }
  return completed;
}

function renderSources(sources) {
  const n = sources.length;
  updateCountChip("sourceListCount", n);
  const savedHeader = document.querySelector(".saved-source-header");
  const savedDivider = document.querySelector(".saved-source-divider");
  setSectionEmpty(savedHeader, n === 0);
  setSectionEmpty(savedDivider, n === 0);
  $("sourceList").innerHTML = n ? sources.map((source) => {
    const snapshotID = source.SnapshotID || source.snapshot_id || "";
    const connector = source.Connector || source.connector || {};
    const access = source.Access || source.access || {};
    const sourceState = source.State || source.state || {};
    const removed = Boolean(sourceState.removed || sourceState.Removed || sourceState.state === "removed" || sourceState.State === "removed");
    const retrievalPolicy = source.Access?.RetrievalPolicy || access.retrieval_policy || source.retrieval_policy || "snapshot_only";
    const modeLabel = retrievalPolicy === "live_reference" ? "라이브 참조" : "스냅샷";
    const locator = localPathLocator(source);
    const media = mediaLocator(source);
    const pdf = pdfLocator(source);
    const document = documentLocator(source);
    const confluence = confluenceSourceInfo(source);
    const mediaLabel = media ? mediaSourceLabel(media) : "";
    const pdfLabel = pdf ? "PDF" : "";
    const documentLabel = document && !pdf && !media ? "문서" : "";
    const confluenceLabel = confluence ? "Confluence" : "";
    let locatorText = connector.ExternalURI || connector.ExternalSourceID || "";
    if (locator) locatorText = `${locator.root_id}/${locator.relative_path || "."}`;
    if (document && !pdf && !media) locatorText = documentSourceText(document);
    if (pdf) locatorText = pdfSourceText(pdf);
    if (media) locatorText = mediaSourceText(media);
    if (confluence) locatorText = confluenceSourceText(confluence);
    const detailPayload = sourceDetailPayload(source, confluence);
    return `
      <div class="item ${removed ? "source-removed" : ""}">
        <div class="item-title">
          ${escapeHTML(source.Title || source.title || snapshotID)}
          <span class="badge muted">${escapeHTML(modeLabel)}</span>
          ${mediaLabel ? `<span class="badge">${escapeHTML(mediaLabel)}</span>` : ""}
          ${pdfLabel ? `<span class="badge">${escapeHTML(pdfLabel)}</span>` : ""}
          ${documentLabel ? `<span class="badge">${escapeHTML(documentLabel)}</span>` : ""}
          ${confluenceLabel ? `<span class="badge">${escapeHTML(confluenceLabel)}</span>` : ""}
          ${confluence?.version ? `<span class="badge muted">v${escapeHTML(confluence.version)}</span>` : ""}
          ${removed ? `<span class="badge warn">제거됨</span>` : ""}
        </div>
        <div class="item-meta">${escapeHTML(snapshotID)} / ${escapeHTML(connector.ConnectorID || connector.connector_id || "source")}</div>
        <div class="item-meta">${escapeHTML(locatorText)}</div>
        <div class="item-actions">
          <button type="button" class="secondary" data-detail-title="소스 상세" data-detail-json="${escapeAttr(JSON.stringify(detailPayload))}">자세히</button>
          ${removed ? `
            <button type="button" data-source-restore="${escapeAttr(snapshotID)}">복원</button>
          ` : `
            <button type="button" class="secondary" data-source-read="${escapeAttr(snapshotID)}">읽기</button>
            ${confluence ? `<button type="button" class="secondary" data-confluence-source-update="${escapeAttr(snapshotID)}">업데이트 확인</button>` : ""}
            <button type="button" class="danger" data-source-remove="${escapeAttr(snapshotID)}">제거</button>
          `}
        </div>
      </div>
    `;
  }).join("") : empty("저장된 소스 없음");
  setSectionEmpty($("sourceList"), n === 0);
}

function sourceDetailPayload(source, confluence) {
  if (!confluence) return source;
  const connector = source.Connector || source.connector || {};
  const access = source.Access || source.access || {};
  const sourceState = source.State || source.state || {};
  const displayURI = confluenceDisplayableExternalURI(confluence.external_uri);
  const detail = {
    type: "confluence_source",
    snapshot_id: source.SnapshotID || source.snapshot_id || "",
    title: source.Title || source.title || "",
    connector_id: connector.ConnectorID || connector.connector_id || "",
    connector_version: connector.ConnectorVersion || connector.connector_version || "",
    site_url: confluence.site_url || "",
    page_id: confluence.page_id || "",
    version: confluence.version || "",
    retrieval_policy: access.RetrievalPolicy || access.retrieval_policy || source.retrieval_policy || "",
    state: sourceState.state || sourceState.State || (sourceState.removed || sourceState.Removed ? "removed" : "active")
  };
  if (displayURI) detail.external_uri = displayURI;
  return detail;
}

function localPathLocator(source) {
  const raw = source.Locators || source.locators;
  if (!raw) return null;
  try {
    const parsed = typeof raw === "string" ? JSON.parse(raw) : raw;
    const locators = Array.isArray(parsed) ? parsed : [parsed];
    return locators.find((locator) => sourceLocatorType(locator) === "local_path") || null;
  } catch (err) {
    return null;
  }
}

function mediaLocator(source) {
  const raw = source.Locators || source.locators;
  if (!raw) return null;
  try {
    const parsed = typeof raw === "string" ? JSON.parse(raw) : raw;
    const locators = Array.isArray(parsed) ? parsed : [parsed];
    const locator = locators.find((item) => {
      if (sourceLocatorType(item) === "media") return true;
      if (sourceConnectorType(source) !== "file_upload" || sourceLocatorType(item) !== "file_upload") return false;
      return uploadedFileContentKind(item) === "image" || uploadedFileMediaType(item).startsWith("image/");
    });
    if (!locator) return null;
    return {
      media_kind: locator.media_kind || locator.MediaKind || (uploadedFileContentKind(locator) === "image" ? "image" : ""),
      filename: uploadedFileFilename(locator),
      mime_type: locator.mime_type || locator.MIMEType || locator.media_type || locator.MediaType || "",
      byte_size: locator.byte_size || locator.ByteSize || 0,
      width: locator.width || locator.Width || 0,
      height: locator.height || locator.Height || 0,
      canonical_url: locator.canonical_url || locator.CanonicalURL || "",
      direct_media_url: locator.direct_media_url || locator.DirectMediaURL || "",
      license: locator.license || locator.License || "",
      attribution: locator.attribution || locator.Attribution || "",
      inspection_support: locator.inspection_support || locator.InspectionSupport || ""
    };
  } catch (err) {
    return null;
  }
}

function documentLocator(source) {
  const raw = source.Locators || source.locators;
  if (!raw) return null;
  try {
    const parsed = typeof raw === "string" ? JSON.parse(raw) : raw;
    const locators = Array.isArray(parsed) ? parsed : [parsed];
    const locator = locators.find((item) => {
      const type = sourceLocatorType(item);
      if (sourceConnectorType(source) === "file_upload" && type === "full_document") return true;
      if (type !== "file_upload") return false;
      const contentKind = uploadedFileContentKind(item);
      const mediaType = uploadedFileMediaType(item);
      return contentKind === "text" || (!contentKind && mediaType && mediaType !== "application/pdf" && !mediaType.startsWith("image/"));
    });
    if (!locator) return null;
    return {
      filename: uploadedFileFilename(locator),
      mime_type: uploadedFileMediaType(locator),
      byte_size: locator.byte_size || locator.ByteSize || 0,
      content_kind: uploadedFileContentKind(locator)
    };
  } catch (err) {
    return null;
  }
}

function pdfLocator(source) {
  const raw = source.Locators || source.locators;
  if (!raw) return null;
  try {
    const parsed = typeof raw === "string" ? JSON.parse(raw) : raw;
    const locators = Array.isArray(parsed) ? parsed : [parsed];
    const locator = locators.find((item) => {
      if (sourceLocatorType(item) === "pdf_document") return true;
      const contentKind = item.content_kind || item.ContentKind || "";
      const mediaType = item.mime_type || item.MIMEType || item.media_type || item.MediaType || "";
      return sourceLocatorType(item) === "file_upload" && (contentKind === "pdf" || mediaType === "application/pdf");
    });
    if (!locator) return null;
    return {
      url: locator.url || locator.URL || "",
      filename: locator.sanitized_filename || locator.SanitizedFilename || locator.original_filename || locator.OriginalFilename || locator.filename || locator.Filename || "",
      mime_type: locator.mime_type || locator.MIMEType || locator.media_type || locator.MediaType || "application/pdf",
      byte_size: locator.byte_size || locator.ByteSize || 0,
      page_count: locator.page_count || locator.PageCount || 0,
      text_length: locator.text_length || locator.TextLength || 0,
      extraction_support: locator.extraction_support || locator.ExtractionSupport || ""
    };
  } catch (err) {
    return null;
  }
}

function sourceLocatorType(locator) {
  return locator.locator_type || locator.LocatorType || locator.kind || locator.Kind || "";
}

function sourceConnectorType(source) {
  const connector = source.Connector || source.connector || {};
  return connector.ConnectorType || connector.connector_type || "";
}

function uploadedFileContentKind(locator) {
  return locator.content_kind || locator.ContentKind || "";
}

function uploadedFileMediaType(locator) {
  return locator.mime_type || locator.MIMEType || locator.media_type || locator.MediaType || "";
}

function uploadedFileFilename(locator) {
  return locator.sanitized_filename || locator.SanitizedFilename || locator.original_filename || locator.OriginalFilename || locator.filename || locator.Filename || "";
}

function confluenceSourceInfo(source) {
  const connector = source.Connector || source.connector || {};
  const connectorID = connector.ConnectorID || connector.connector_id || "";
  const connectorType = connector.ConnectorType || connector.connector_type || "";
  if (connectorID !== "confluence" && connectorType !== "confluence_cloud") return null;
  const externalID = connector.ExternalSourceID || connector.external_source_id || "";
  const externalURI = connector.ExternalURI || connector.external_uri || "";
  const parts = String(externalID || "").split(":");
  const raw = source.Locators || source.locators;
  let locator = null;
  try {
    const parsed = raw ? (typeof raw === "string" ? JSON.parse(raw) : raw) : [];
    const locators = Array.isArray(parsed) ? parsed : [parsed];
    locator = locators.find((item) => {
      const locatorType = sourceLocatorType(item);
      return locatorType === "confluence_page_body" || locatorType === "confluence_page_range" || item.partial || item.Partial;
    }) || null;
  } catch (err) {
    locator = null;
  }
  return {
    cloud_id: locator?.cloud_id || locator?.CloudID || (parts.length >= 2 ? parts[0] : ""),
    site_url: locator?.site_url || locator?.SiteURL || "",
    page_id: locator?.page_id || locator?.PageID || (parts.length >= 2 ? parts.slice(1).join(":") : externalID),
    external_uri: externalURI,
    version: connector.ExternalVersion || connector.external_version || ""
  };
}

function mediaSourceLabel(locator) {
  switch (locator.media_kind) {
    case "image": return "이미지";
    case "audio": return "오디오";
    case "video": return "영상";
    default: return "미디어";
  }
}

function pdfSourceText(locator) {
  const parts = [locator.mime_type || "application/pdf"];
  if (locator.page_count) parts.push(`${locator.page_count}쪽`);
  if (locator.byte_size) parts.push(formatBytes(locator.byte_size));
  if (locator.extraction_support) parts.push("텍스트 추출");
  const target = locator.url || locator.filename || "";
  return `${parts.join(" · ")}${target ? " / " + target : ""}`;
}

function mediaSourceText(locator) {
  const parts = [];
  if (locator.mime_type) parts.push(locator.mime_type);
  if (locator.width && locator.height) parts.push(`${locator.width}×${locator.height}`);
  if (locator.byte_size) parts.push(formatBytes(locator.byte_size));
  if (locator.inspection_support === "inspect_unsupported") parts.push("inspect 미지원");
  if (locator.inspection_support === "metadata_only_until_vision_engine_configured") parts.push("이미지 분석 미설정");
  const url = locator.canonical_url || locator.direct_media_url || locator.filename || "";
  return `${parts.join(" · ")}${parts.length && url ? " / " : ""}${url}`;
}

function documentSourceText(locator) {
  const parts = [];
  if (locator.mime_type) parts.push(locator.mime_type);
  if (locator.byte_size) parts.push(formatBytes(locator.byte_size));
  return `${parts.join(" · ")}${parts.length && locator.filename ? " / " : ""}${locator.filename || ""}`;
}

function confluenceSourceText(info) {
  const parts = [];
  const externalURI = confluenceDisplayableExternalURI(info.external_uri);
  const siteHost = confluenceExternalURIHost(info.site_url || externalURI);
  if (siteHost) parts.push(`site ${siteHost}`);
  if (info.page_id) parts.push(`page ${info.page_id}`);
  if (info.version) parts.push(`v${info.version}`);
  if (externalURI) parts.push(externalURI);
  return parts.join(" / ");
}

function confluenceDisplayableExternalURI(uri) {
  try {
    const parsed = new URL(uri);
    return parsed.protocol === "https:" || parsed.protocol === "http:" ? uri : "";
  } catch (err) {
    return "";
  }
}

function confluenceExternalURIHost(uri) {
  try {
    return new URL(uri).host;
  } catch (err) {
    return "";
  }
}

function formatBytes(value) {
  const bytes = Number(value || 0);
  if (!Number.isFinite(bytes) || bytes <= 0) return "";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KiB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MiB`;
}

function renderSourceCandidates(events, sources) {
  const existing = acceptedSourceCandidateKeys(sources);
  const decisions = sourceCandidateDecisions(events);
  const candidates = sourceCandidatesFromEvents(events).filter((candidate) => {
    const normalized = normalizeSourceURL(candidate.url);
    return normalized && !sourceCandidateAccepted(existing, normalized) && decisions.get(normalized)?.state !== "rejected";
  });
  updateCountChip("sourceCandidateCount", candidates.length);
  updateSourceCandidateIndicators(candidates.length);
  const candidateHeader = document.querySelector(".source-candidate-header");
  const candidateDivider = document.querySelector(".source-candidate-divider");
  setSectionEmpty(candidateHeader, candidates.length === 0);
  setSectionEmpty(candidateDivider, candidates.length === 0);
  setSectionEmpty($("sourceCandidateList"), candidates.length === 0);
  $("sourceCandidateList").innerHTML = candidates.length ? candidates.map((candidate) => {
    const normalized = normalizeSourceURL(candidate.url);
    const busy = state.sourceCandidateBusy.has(normalized);
    const selected = state.selectedSourceCandidates.has(normalized);
    return `
      <div class="item${selected ? " selected" : ""}">
        <input type="checkbox" class="item-select" data-select-source-url="${escapeAttr(normalized)}" data-source-candidate-title="${escapeAttr(candidate.title || "")}" aria-label="후보 선택" ${selected ? "checked" : ""}>
        <div class="item-title">${escapeHTML(candidate.title || candidate.url)}</div>
        <div class="item-meta"><a href="${escapeAttr(candidate.url)}" target="_blank" rel="noopener noreferrer">${escapeHTML(candidate.url)}</a></div>
        ${candidate.staging ? `<div class="item-meta">${sourceCandidateStagingLabel(candidate.staging)}</div>` : ""}
        <div class="item-meta source-candidate-reason"><strong>채택 의견</strong> ${escapeHTML(candidate.reason)}</div>
        <div class="item-actions">
          <button type="button" class="secondary" data-detail-title="소스 후보 상세" data-detail-json="${escapeAttr(JSON.stringify(candidate))}">자세히</button>
          <button type="button" data-add-source-url="${escapeAttr(candidate.url)}" data-source-candidate-title="${escapeAttr(candidate.title || "")}" ${busy ? "disabled" : ""}>${busy ? "처리 중" : "소스로 추가"}</button>
          <button type="button" class="danger" data-reject-source-url="${escapeAttr(candidate.url)}" ${busy ? "disabled" : ""}>기각</button>
        </div>
      </div>
    `;
  }).join("") : empty("채택 의견이 있는 소스 후보 없음");
  pruneSelectedSourceCandidates(candidates);
  updateSourceCandidateBulkBar();
  const rejected = sourceCandidatesFromEvents(events).filter((candidate) => {
    const normalized = normalizeSourceURL(candidate.url);
    return normalized && !sourceCandidateAccepted(existing, normalized) && decisions.get(normalized)?.state === "rejected";
  });
  updateCountChip("rejectedSourceCandidateCount", rejected.length);
  setSectionEmpty($("rejectedSourceCandidateDetails"), rejected.length === 0);
  $("rejectedSourceCandidateList").innerHTML = rejected.length ? rejected.map((candidate) => {
    const decision = decisions.get(normalizeSourceURL(candidate.url)) || {};
    const busy = state.sourceCandidateBusy.has(normalizeSourceURL(candidate.url));
    return `
      <div class="item">
        <div class="item-title">${escapeHTML(candidate.title || candidate.url)}</div>
        <div class="item-meta"><a href="${escapeAttr(candidate.url)}" target="_blank" rel="noopener noreferrer">${escapeHTML(candidate.url)}</a></div>
        ${candidate.staging ? `<div class="item-meta">${sourceCandidateStagingLabel(candidate.staging)}</div>` : ""}
        <div class="item-meta">${escapeHTML(decision.reason || "사용자가 기각했습니다.")}</div>
        <div class="item-actions">
          <button type="button" class="secondary" data-detail-title="기각한 소스 후보 상세" data-detail-json="${escapeAttr(JSON.stringify({ candidate, decision }))}">자세히</button>
          <button type="button" data-restore-source-url="${escapeAttr(candidate.url)}" ${busy ? "disabled" : ""}>다시 검토</button>
        </div>
      </div>
    `;
  }).join("") : empty("기각한 후보 없음");
}

function acceptedSourceCandidateKeys(sources) {
  const keys = new Set();
  for (const source of sources || []) {
    const connector = source.Connector || source.connector || {};
    for (const value of [connector.ExternalURI, connector.external_uri, connector.ExternalSourceID, connector.external_source_id]) {
      const normalized = normalizeSourceURL(value);
      if (normalized) keys.add(normalized);
    }
    for (const locator of sourceLocators(source)) {
      const key = confluenceSourceKey(locator.site_url || locator.SiteURL || "", locator.page_id || locator.PageID || "");
      if (key) keys.add(key);
    }
  }
  return keys;
}

function sourceCandidateAccepted(existingKeys, normalizedURL) {
  if (!normalizedURL) return false;
  return existingKeys.has(normalizedURL) || existingKeys.has(confluenceCandidateKeyFromURL(normalizedURL));
}

function sourceLocators(source) {
  const raw = source?.Locators ?? source?.locators;
  if (!raw) return [];
  if (Array.isArray(raw)) return raw;
  if (typeof raw === "string") {
    try {
      const parsed = JSON.parse(raw);
      return Array.isArray(parsed) ? parsed : [];
    } catch (err) {
      return [];
    }
  }
  return [];
}

function confluenceCandidateKeyFromURL(rawURL) {
  try {
    const url = new URL(rawURL);
    const segments = url.pathname.split("/").filter(Boolean);
    const index = segments.findIndex((segment) => segment.toLowerCase() === "pages");
    if (index < 0 || index + 1 >= segments.length) return "";
    return confluenceSourceKey(`${url.protocol}//${url.host}`, decodeURIComponent(segments[index + 1]));
  } catch (err) {
    return "";
  }
}

function confluenceSourceKey(siteURL, pageID) {
  try {
    const site = new URL(siteURL);
    const id = String(pageID || "").trim();
    if (!id) return "";
    return `confluence:${site.hostname.toLowerCase()}:${id}`;
  } catch (err) {
    return "";
  }
}

function updateSourceCandidateIndicators(count) {
  updateCountChip("sourceTabCandidateCount", count);
  const metric = $("candidateCount");
  if (metric) {
    metric.textContent = `소스 후보 ${count}개`;
    metric.classList.toggle("collapsed-empty", count === 0);
    metric.classList.remove("hidden");
  }
  const notice = $("sourceCandidateNotice");
  if (notice) notice.classList.toggle("hidden", count === 0);
  const noticeCount = $("sourceCandidateNoticeCount");
  if (noticeCount) noticeCount.textContent = String(count);
}

function openSourceCandidatesTab() {
  state.activeTab = "sources";
  location.hash = "sources";
  renderTabs();
  $("sourceCandidateList")?.scrollIntoView({ block: "start", behavior: "smooth" });
}

function sourceCandidatesFromEvents(events) {
  const byURL = new Map();
  const staging = sourceCandidateStagingByProposal(events);
  for (const event of events) {
    if (event.EventType !== "source.candidate.proposed") continue;
    const payload = event.Payload || {};
    for (const candidate of payload.candidates || []) {
      const url = normalizeSourceURL(candidate.url || candidate.URL || "");
      const reason = candidate.reason || candidate.Reason || "";
      if (!url || !String(reason).trim()) continue;
      const sequence = Number(event.Sequence || 0);
      const existing = byURL.get(url);
      if (existing && existing.sequence > sequence) continue;
      byURL.set(url, {
        url,
        title: candidate.title || candidate.Title || "",
        reason,
        eventID: event.EventID,
        sequence,
        userEventID: payload.user_event_id || "",
        agentEventID: payload.agent_event_id || "",
        staging: staging.get(url) || staging.get(event.EventID) || null
      });
    }
  }
  return [...byURL.values()];
}

function sourceCandidateStagingByProposal(events) {
  const byKey = new Map();
  for (const event of events) {
    if (!["source.candidate.staging_started", "source.candidate.staged", "source.candidate.staging_failed"].includes(event.EventType)) continue;
    const payload = event.Payload || {};
    const proposalEventID = payload.proposal_event_id || "";
    const url = normalizeSourceURL(payload.url || payload.URL || "");
    const sequence = Number(event.Sequence || 0);
    const state = event.EventType === "source.candidate.staged"
      ? "staged"
      : event.EventType === "source.candidate.staging_failed"
        ? "failed"
        : "fetching";
    const record = {
      state,
      eventID: event.EventID,
      sequence,
      artifactID: payload.artifact_id || "",
      message: payload.message || ""
    };
    for (const key of [proposalEventID, url].filter(Boolean)) {
      const existing = byKey.get(key);
      if (!existing || Number(existing.sequence || 0) <= sequence) {
        byKey.set(key, record);
      }
    }
  }
  return byKey;
}

function sourceCandidateStagingLabel(staging) {
  if (!staging) return "";
  if (staging.state === "staged") {
    return `<strong>본문 상태</strong> 미승인 후보 본문 준비됨`;
  }
  if (staging.state === "failed") {
    return `<strong>본문 상태</strong> 가져오기 실패${staging.message ? ` · ${escapeHTML(staging.message)}` : ""}`;
  }
  return `<strong>본문 상태</strong> 가져오는 중`;
}

function sourceCandidateDecisions(events) {
  const decisions = new Map();
  for (const event of events) {
    if (event.EventType !== "source.candidate.rejected" && event.EventType !== "source.candidate.restored") continue;
    const url = normalizeSourceURL(event.Payload?.url || event.Payload?.URL || "");
    if (!url) continue;
    decisions.set(url, {
      state: event.EventType === "source.candidate.rejected" ? "rejected" : "restored",
      reason: event.Payload?.reason || "",
      eventID: event.EventID,
      sequence: event.Sequence
    });
  }
  return decisions;
}

function sourceCandidateTitleForURL(url) {
  const normalized = normalizeSourceURL(url);
  if (!normalized) return "";
  const candidates = sourceCandidatesFromEvents(state.detail?.events || []);
  const match = candidates.find((candidate) => normalizeSourceURL(candidate.url) === normalized);
  return match?.title || "";
}

function refreshSourceCandidates() {
  if (!state.detail) return;
  renderSourceCandidates(state.detail.events || [], state.detail.sources || []);
}

function renderLiquid2Results(candidates) {
  $("liquid2Results").innerHTML = candidates.length ? candidates.map((candidate) => {
    const connector = candidate.Connector || candidate.connector || {};
    const sourceID = connector.ExternalSourceID || connector.external_source_id || "";
    const summary = candidate.Summary || candidate.summary || "검색 결과 요약 없음";
    return `
      <div class="item">
        <div class="item-title">${escapeHTML(candidate.Title || candidate.title || sourceID)}</div>
        <div class="item-meta">${escapeHTML(sourceID)}</div>
        <div class="item-meta"><strong>검색 요약</strong> ${escapeHTML(summary)}</div>
        <div class="item-actions">
          <button type="button" class="secondary" data-detail-title="Liquid2 검색 결과" data-detail-json="${escapeAttr(JSON.stringify(candidate))}">자세히</button>
          <button type="button" data-liquid2-source-id="${escapeAttr(sourceID)}">소스로 가져오기</button>
        </div>
      </div>
    `;
  }).join("") : empty("Liquid2 검색 결과 없음");
}

function renderLiquid2Error(message) {
  $("liquid2Results").innerHTML = `<div class="item"><div class="item-title">Liquid2 연결 실패</div><div class="item-meta">${escapeHTML(message)}</div></div>`;
}

function renderProposals(proposals, records) {
  const evidenceByID = new Map((records.evidence || []).map((record) => [record.evidence_id, record]));
  const claimByID = new Map((records.claims || []).map((record) => [record.claim_id, record]));
  const questionByID = new Map((records.questions || []).map((record) => [record.question_id, record]));
  const optionByID = new Map((records.options || []).map((record) => [record.option_id, record]));
  const pending = proposals.filter((proposal) => proposal.state === "pending_review");
  updateCountChip("proposalListCount", pending.length);
  $("proposalList").innerHTML = pending.length ? pending.map((proposal) => {
    const refs = proposal.object_refs || [];
    const objects = refs.map((ref) => proposalObjectDetail(ref, evidenceByID, claimByID, questionByID, optionByID));
    const text = objects.map((object) => object.label).join("\n");
    const title = proposal.title || proposalKindLabel(refs) || proposal.proposal_id;
    const objectMeta = objects.map((object) => object.meta).filter(Boolean).join(" · ");
    const detail = { proposal, objects };
    const selected = state.selectedProposals.has(proposal.proposal_id);
    const actions = `
      <div class="item-actions">
        <button type="button" class="secondary" data-detail-title="후보 상세" data-detail-json="${escapeAttr(JSON.stringify(detail))}">자세히</button>
        <button type="button" data-proposal-id="${escapeAttr(proposal.proposal_id)}" data-action="approve">승인</button>
        <button type="button" class="danger" data-proposal-id="${escapeAttr(proposal.proposal_id)}" data-action="reject">기각</button>
      </div>
    `;
    return `
      <div class="item${selected ? " selected" : ""}">
        <input type="checkbox" class="item-select" data-select-proposal-id="${escapeAttr(proposal.proposal_id)}" aria-label="후보 선택" ${selected ? "checked" : ""}>
        <div class="item-title">${escapeHTML(title)}</div>
        <div class="item-meta">${escapeHTML(proposalKindLabel(refs) || "저장 후보")} / ${escapeHTML(proposal.proposal_id)}${objectMeta ? ` / ${escapeHTML(objectMeta)}` : ""}</div>
        <div>${escapeHTML(text)}</div>
        ${actions}
      </div>
    `;
  }).join("") : empty("지금은 검토 대기 후보가 없습니다.");
  pruneSelectedProposals(pending);
  updateProposalBulkBar();
}

function proposalObjectDetail(ref, evidenceByID, claimByID, questionByID, optionByID) {
  if (ref.object_kind === "evidence_record" && evidenceByID.has(ref.object_id)) {
    const record = evidenceByID.get(ref.object_id);
    const typeLabel = evidenceTypeLabel(record.evidence_type);
    return {
      ref,
      record,
      label: `근거: ${record.summary || ref.object_id}`,
      meta: `근거 신호: ${typeLabel}`
    };
  }
  if (ref.object_kind === "claim_record" && claimByID.has(ref.object_id)) {
    const record = claimByID.get(ref.object_id);
    return {
      ref,
      record,
      label: `주장: ${record.text || ref.object_id}`,
      meta: ""
    };
  }
  if (ref.object_kind === "question_record" && questionByID.has(ref.object_id)) {
    const record = questionByID.get(ref.object_id);
    return {
      ref,
      record,
      label: `질문: ${record.text || ref.object_id}`,
      meta: ""
    };
  }
  if (ref.object_kind === "option_record" && optionByID.has(ref.object_id)) {
    const record = optionByID.get(ref.object_id);
    return {
      ref,
      record,
      label: `선택지: ${record.title || ref.object_id}`,
      meta: ""
    };
  }
  return {
    ref,
    record: null,
    label: `${ref.object_kind}: ${ref.object_id}`,
    meta: ""
  };
}

function evidenceTypeLabel(type) {
  return EVIDENCE_TYPE_LABELS[type] || type || "근거";
}

function proposalKindLabel(refs) {
  const kinds = new Set((refs || []).map((ref) => ref.object_kind));
  if (kinds.size > 1) return "복합 후보";
  if (kinds.has("evidence_record")) return "근거 후보";
  if (kinds.has("claim_record")) return "주장 후보";
  if (kinds.has("question_record")) return "질문 후보";
  if (kinds.has("option_record")) return "선택지 후보";
  return "저장 후보";
}

function renderSavedEvidence(saved) {
  const n = saved.length;
  updateCountChip("savedCountSummary", n);
  setSectionEmpty($("savedListDetails"), n === 0);
  $("savedList").innerHTML = n ? saved.map((record) => `
    <div class="item">
      <div>${escapeHTML(record.summary)}</div>
      <div class="item-meta">${escapeHTML(evidenceTypeLabel(record.evidence_type))} / ${escapeHTML(record.evidence_id)}</div>
      <div class="item-actions">
        <button type="button" class="secondary" data-detail-title="승인된 근거 상세" data-detail-json="${escapeAttr(JSON.stringify(record))}">자세히</button>
      </div>
    </div>
  `).join("") : empty("승인된 근거 없음");
}

function renderClaimConfidenceChanges(confidenceViews, claims) {
  const claimByID = new Map((claims || []).map((record) => [record.claim_id, record]));
  const changed = (confidenceViews || []).filter((view) => (view.history || []).length > 0);
  updateCountChip("claimConfidenceCount", changed.length);
  setSectionEmpty($("claimConfidenceDetails"), changed.length === 0);
  $("claimConfidenceList").innerHTML = changed.length ? changed.map((view) => {
    const claim = claimByID.get(view.claim_id) || {};
    const current = view.current_confidence || {};
    return `
      <div class="item confidence-item">
        <div class="confidence-line">
          ${confidenceBadge(view)}
          <span class="item-title">${escapeHTML(claim.text || view.claim_id)}</span>
        </div>
        <div class="item-meta clamp-line">${escapeHTML(current.rationale || "신뢰도 변경 사유 없음")}</div>
        <div class="item-actions">
          <button type="button" class="secondary" data-confidence-claim-id="${escapeAttr(view.claim_id)}">자세히</button>
        </div>
      </div>
    `;
  }).join("") : empty("신뢰도 변화 없음");
}

function renderSavedClaims(saved, confidenceViews = []) {
  const n = saved.length;
  updateCountChip("savedClaimListCount", n);
  setSectionEmpty($("savedClaimListDetails"), n === 0);
  const confidenceByClaim = new Map((confidenceViews || []).map((view) => [view.claim_id, view]));
  $("savedClaimList").innerHTML = n ? saved.map((record) => {
    const confidence = confidenceByClaim.get(record.claim_id) || initialConfidenceView(record);
    const rationale = confidence.current_confidence?.rationale || record.confidence?.rationale || "";
    return `
    <div class="item">
      <div class="item-title">${escapeHTML(record.text)}</div>
      <div class="confidence-line">
        ${confidenceBadge(confidence)}
        <span class="item-meta">${escapeHTML(record.claim_type || "claim")} / ${escapeHTML(record.claim_id)}</span>
      </div>
      ${rationale ? `<div class="item-meta clamp-line">${escapeHTML(rationale)}</div>` : ""}
      <div class="item-meta">근거 ${escapeHTML((record.supporting_evidence_ids || []).join(", ") || "없음")}</div>
      <div class="item-actions">
        <button type="button" class="secondary" data-detail-title="승인된 주장 상세" data-detail-json="${escapeAttr(JSON.stringify(record))}">자세히</button>
        <button type="button" class="secondary" data-confidence-claim-id="${escapeAttr(record.claim_id)}">신뢰도</button>
      </div>
    </div>
  `;
  }).join("") : empty("승인된 주장 없음");
}

function initialConfidenceView(record) {
  return {
    claim_id: record.claim_id,
    initial_confidence: record.confidence || { level: "unknown" },
    current_confidence: record.confidence || { level: "unknown" },
    direction: "initial",
    history: []
  };
}

function confidenceBadge(view) {
  const current = view.current_confidence || {};
  const level = current.level || "unknown";
  const direction = view.direction || "initial";
  const needsVerification = current.needs_verification ? " · 확인 필요" : "";
  return `<span class="badge confidence ${escapeAttr(level)}">${escapeHTML(confidenceLabel(level))}${escapeHTML(directionGlyph(direction))}${escapeHTML(needsVerification)}</span>`;
}

function confidenceLabel(level) {
  switch (level) {
    case "high":
      return "신뢰 높음";
    case "medium":
      return "신뢰 보통";
    case "low":
      return "신뢰 낮음";
    default:
      return "신뢰 미정";
  }
}

function directionGlyph(direction) {
  switch (direction) {
    case "up":
      return " ↑";
    case "down":
      return " ↓";
    case "unchanged":
      return " ·";
    default:
      return "";
  }
}

// Collapse a set of secondary report actions (tools / downloads) behind a
// small dropdown so the card shows a tidy row of primary "보기" actions.
function reportActionMenu(label, itemsHTML) {
  if (!itemsHTML || !itemsHTML.trim()) return "";
  return `<details class="report-menu"><summary>${escapeHTML(label)}</summary><div class="report-menu-items">${itemsHTML}</div></details>`;
}

function renderReports(versions) {
  const artifactReports = reportArtifactPayloads();
  const reports = versions.map((version, index) => reportViewModel(version, index));
  updateCountChip("reportListCount", artifactReports.length + reports.length);
  updateCountChip("reportTabCount", artifactReports.length + reports.length);

  const artifactCards = artifactReports.map((payload, index) => ({
    key: `artifact:${payload.artifact_id || `idx${index}`}`,
    isLatest: index === 0,
    payload
  }));
  const legacyCards = reports.map((report) => ({ key: `version:${report.versionID}`, report }));
  const allKeys = [...artifactCards.map((c) => c.key), ...legacyCards.map((c) => c.key)];

  // Default to the newest card; drop a stale selection/preview if its card is gone.
  if (!state.selectedReportKey || !allKeys.includes(state.selectedReportKey)) {
    state.selectedReportKey = allKeys[0] || "";
  }
  if (state.reportPreview && !allKeys.includes(state.reportPreview.key)) {
    state.reportPreview = null;
  }
  const selectedKey = state.selectedReportKey;

  const sections = [];
  if (artifactCards.length) {
    sections.push(`
      <div class="list-section-label">Markdown artifact</div>
      ${artifactCards.map(({ key, isLatest, payload }) => {
        const plan = reportArtifactPlanPayload(payload);
        const planData = plan.plan || {};
        const sectionCount = Array.isArray(planData.sections) ? planData.sections.length : 0;
        const modeLabel = payload.report_mode_label || REPORT_MODE_LABELS[payload.report_mode] || "보고서";
        const planLabel = planData.summary
          ? `${sectionCount}개 섹션 / ${planData.summary}`
          : (payload.report_mode === "one_take" ? "원테이크 생성: 별도 계획 없음" : "기록된 생성 계획 없음");
        const planButton = plan.event_id
          ? `<button type="button" class="secondary" data-report-plan-event-id="${escapeAttr(plan.event_id)}" data-action="plan">생성 계획</button>`
          : "";
        const trace = mcpTraceSummary(payload.tool_session_id || payload.plan_tool_session_id || "");
        const designed = reportArtifactDesignedExportState(payload.artifact_id || "");
        const humanized = reportArtifactHumanizedExportState(payload.artifact_id || "");
        const humanizedLabel = humanized.state === "completed"
          ? "생성 완료"
          : humanized.state === "failed"
            ? "생성 실패"
            : humanized.state === "pending"
              ? "생성 중"
              : humanized.state === "skipped"
                ? "변경 없음"
              : "선택 실행 가능";
        const humanizedActions = humanized.state === "completed"
          ? `
                <button type="button" data-report-artifact-id="${escapeAttr(humanized.payload.artifact_id || "")}" data-action="view-artifact">보정 Markdown 보기</button>
                <button type="button" class="secondary" data-report-artifact-id="${escapeAttr(humanized.payload.artifact_id || "")}" data-action="download-artifact">보정 MD 받기</button>
              `
          : humanized.state === "pending"
            ? `<button type="button" class="secondary" disabled>말투 보정 중</button>`
            : `<button type="button" data-report-artifact-id="${escapeAttr(payload.artifact_id || "")}" data-action="start-humanized-markdown-artifact" ${state.reportPending ? "disabled" : ""}>${humanized.state === "failed" ? "H5 말투 보정 다시 생성" : "H5 말투 보정 생성"}</button>`;
        const humanizedFailureLine = humanized.state === "failed"
          ? `<div class="report-plan-line report-error-line"><span class="badge warn">실패 사유</span><span>${escapeHTML(humanized.payload.error || humanized.payload.text || "실패 사유 없음")}</span></div>`
          : "";
        const designedLabel = designed.state === "completed"
          ? "생성 완료"
          : designed.state === "pending"
            ? "생성 중"
            : designed.state === "failed"
              ? "생성 실패"
              : "아직 없음";
        const designedActions = designed.state === "completed"
          ? `
                <button type="button" data-report-artifact-id="${escapeAttr(payload.artifact_id || "")}" data-action="view-designed-html-artifact">디자인 HTML 보기</button>
                <button type="button" class="secondary" data-report-artifact-id="${escapeAttr(payload.artifact_id || "")}" data-action="download-designed-html-artifact">디자인 HTML 받기</button>
              `
          : designed.state === "pending"
            ? `<button type="button" class="secondary" disabled>디자인 HTML 생성 중</button>`
            : `<button type="button" data-report-artifact-id="${escapeAttr(payload.artifact_id || "")}" data-action="start-designed-html-artifact">${designed.state === "failed" ? "디자인 HTML 다시 생성" : "디자인 HTML 생성"}</button>`;
        return `
          <div class="item report-card ${isLatest ? "active" : ""} ${key === selectedKey ? "selected" : ""}" data-report-key="${escapeAttr(key)}">
            <div class="item-title report-title-line report-card-toggle">
              <span>${escapeHTML(payload.title || "Markdown report")}</span>
              <span class="chip-row report-chip-row">
                ${isLatest ? `<span class="badge session-new">최신</span>` : `<span class="badge muted">이전</span>`}
                <span class="badge">${escapeHTML(modeLabel)}</span>
                <span class="badge muted">Markdown artifact</span>
              </span>
            </div>
            <div class="report-card-body">
              <div class="item-meta clamp-line" title="${escapeAttr(payload.artifact_id || "")}">${escapeHTML(payload.artifact_id || "")}</div>
              <div class="item-meta">${escapeHTML(payload.text || "리포트 artifact가 생성되었습니다.")}</div>
              <div class="report-plan-line">
                <span class="badge muted">생성 계획</span>
                <span>${escapeHTML(planLabel)}</span>
              </div>
              <div class="report-plan-line">
                <span class="badge muted">디자인 HTML</span>
                <span>${escapeHTML(designedLabel)}</span>
              </div>
              <div class="report-plan-line">
                <span class="badge muted">H5 말투 보정</span>
                <span>${escapeHTML(humanizedLabel)}</span>
              </div>
              ${humanizedFailureLine}
              <div class="report-trace">
                <div class="report-trace-head">
                  <span class="badge muted">MCP 추적</span>
                  <span>${escapeHTML(trace.total ? "도구 호출 기록 있음" : "기록된 MCP 호출 없음")}</span>
                </div>
                ${renderTraceBars(trace)}
              </div>
              <div class="item-actions">
                <button type="button" data-report-artifact-id="${escapeAttr(payload.artifact_id || "")}" data-action="view-artifact">Markdown 보기</button>
                <button type="button" data-report-artifact-id="${escapeAttr(payload.artifact_id || "")}" data-action="view-html-artifact">기본 HTML 보기</button>
                <button type="button" class="secondary" data-report-artifact-id="${escapeAttr(payload.artifact_id || "")}" data-report-title="${escapeAttr(payload.title || "Markdown report")}" data-action="patch-artifact" ${state.reportPending ? "disabled" : ""}>MCP 패치</button>
                ${humanizedActions}
                ${designedActions}
                ${reportActionMenu("도구 ▾", `<button type="button" class="secondary" data-detail-title="리포트 artifact 상세" data-detail-json="${escapeAttr(JSON.stringify(payload))}">자세히</button>${planButton}`)}
                ${reportActionMenu("받기 ▾", `<button type="button" class="secondary" data-report-artifact-id="${escapeAttr(payload.artifact_id || "")}" data-action="download-artifact">MD 받기</button><button type="button" class="secondary" data-report-artifact-id="${escapeAttr(payload.artifact_id || "")}" data-action="download-html-artifact">기본 HTML 받기</button>`)}
              </div>
              ${reportPreviewInlineHTML(key)}
            </div>
          </div>
        `;
      }).join("")}
    `);
  }
  if (legacyCards.length) {
    sections.push(`
      <div class="list-section-label">Legacy AST report</div>
      ${legacyCards.map(({ key, report }) => `
    <div class="item report-card ${report.isLatest ? "active" : ""} ${key === selectedKey ? "selected" : ""}" data-report-key="${escapeAttr(key)}" data-report-card-version-id="${escapeAttr(report.versionID)}">
      <div class="item-title report-title-line report-card-toggle">
        <span>${escapeHTML(report.title)}</span>
        <span class="chip-row report-chip-row">
          ${report.isLatest ? `<span class="badge session-new">최신</span>` : `<span class="badge muted">이전</span>`}
          ${report.rigorLabel ? `<span class="badge">${escapeHTML(report.rigorLabel)}</span>` : ""}
          <span class="badge muted">${escapeHTML(report.stateLabel)}</span>
        </span>
      </div>
      <div class="report-card-body">
        <div class="item-meta clamp-line" title="${escapeAttr(report.versionID)}">${escapeHTML(report.createdLabel)} / ${escapeHTML(report.versionID)}</div>
        <div class="item-meta">${escapeHTML(report.exportLabel)}</div>
        <div class="report-plan-line">
          <span class="badge muted">생성 계획</span>
          <span>${escapeHTML(report.planLabel)}</span>
        </div>
        <div class="report-trace">
          <div class="report-trace-head">
            <span class="badge muted">MCP 추적</span>
            <span>${escapeHTML(report.traceLabel)}</span>
          </div>
          ${renderTraceBars(report.trace)}
        </div>
        <div class="item-actions">
          <button type="button" data-report-version-id="${escapeAttr(report.versionID)}" data-action="markdown">Markdown 보기</button>
          <button type="button" data-report-version-id="${escapeAttr(report.versionID)}" data-action="html">HTML 보기</button>
          <button type="button" data-report-version-id="${escapeAttr(report.versionID)}" data-action="json_ast">JSON 보기</button>
          ${reportActionMenu("도구 ▾", `<button type="button" class="secondary" data-detail-title="리포트 버전 상세" data-detail-json="${escapeAttr(JSON.stringify(report.version))}">자세히</button><button type="button" class="secondary" data-report-version-id="${escapeAttr(report.versionID)}" data-action="plan">생성 계획</button><button type="button" class="secondary" data-report-version-id="${escapeAttr(report.versionID)}" data-action="mcp-trace">MCP 추적</button><button type="button" class="secondary" data-report-version-id="${escapeAttr(report.versionID)}" data-action="ast">AST 보기</button>`)}
          ${reportActionMenu("받기 ▾", `<button type="button" class="secondary" data-report-version-id="${escapeAttr(report.versionID)}" data-action="download-markdown">MD 받기</button><button type="button" class="secondary" data-report-version-id="${escapeAttr(report.versionID)}" data-action="download-html">HTML 받기</button><button type="button" class="secondary" data-report-version-id="${escapeAttr(report.versionID)}" data-action="download-json_ast">JSON 받기</button>`)}
        </div>
        ${reportPreviewInlineHTML(key)}
      </div>
    </div>
  `).join("")}
    `);
  }
  $("reportList").innerHTML = sections.length ? sections.join("") : empty("리포트 artifact 없음");
}

function renderReportsFromState() {
  renderReports(state.detail?.report_versions || []);
}

function reportPreviewInlineHTML(key) {
  if (key === state.selectedReportKey) {
    return `<div class="report-card-preview report-card-preview-hint">‘Markdown 보기’ 등을 누르면 전체 화면 팝업으로 열립니다.</div>`;
  }
  return "";
}

// Report viewing opens a large, screen-fitting modal (the in-card preview was
// too small). markdown → rendered, html → sandboxed iframe, else → <pre>.
function openReportModal(header, kind, content) {
  $("detailTitle").textContent = header || "리포트 보기";
  state.detailText = String(content ?? "");
  let body;
  if (kind === "markdown") {
    body = `<div class="report-modal-body turn-markdown">${renderMarkdown(content)}</div>`;
  } else if (kind === "html") {
    body = `<div class="report-modal-frame"><iframe title="HTML 리포트" sandbox="allow-scripts" srcdoc="${escapeAttr(content)}"></iframe></div>`;
  } else {
    body = `<pre class="report-modal-pre">${escapeHTML(content)}</pre>`;
  }
  $("detailBody").innerHTML = body;
  openDetailModal(true);
}

function openReportModalLoading(header) {
  $("detailTitle").textContent = header || "리포트 불러오는 중";
  $("detailBody").innerHTML = `<div class="report-preview-loading"><span class="spinner"></span>불러오는 중…</div>`;
  openDetailModal(true);
}

function openDetailModal(wide) {
  const card = $("detailModal").querySelector(".modal-card");
  if (card) card.classList.toggle("modal-card--wide", Boolean(wide));
  $("detailModal").classList.remove("hidden");
}

function setReportPreviewLoading(key) {
  state.selectedReportKey = key;
  renderReportsFromState();
  openReportModalLoading("리포트 불러오는 중");
}

function applyReportPreview(key, kind, header, content) {
  state.selectedReportKey = key;
  renderReportsFromState();
  openReportModal(header, kind, content);
}

function clearReportPreview() {
  hideDetail();
}

function selectReport(key) {
  if (!key) return;
  // Accordion select only — the 보기 buttons open the content modal explicitly.
  state.selectedReportKey = key;
  renderReportsFromState();
}

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

function reportArtifactPayloads() {
  const events = state.detail?.events || [];
  return events
    .filter((event) => event.EventType === "report.artifact.created")
    .map((event) => ({ ...(event.Payload || {}), event_id: event.EventID, created_at: event.CreatedAt }))
    .reverse();
}

function reportArtifactPlanPayload(artifactPayload) {
  const pendingID = artifactPayload?.pending_event_id || "";
  const artifactID = artifactPayload?.artifact_id || "";
  const planEventID = artifactPayload?.plan_event_id || "";
  const events = state.detail?.events || [];
  for (let i = events.length - 1; i >= 0; i--) {
    const event = events[i];
    if (event.EventType !== "report.plan.created") continue;
    const payload = event.Payload || {};
    if (
      (planEventID && event.EventID === planEventID) ||
      (artifactID && payload.artifact_id === artifactID) ||
      (pendingID && payload.pending_event_id === pendingID)
    ) {
      return { ...payload, event_id: event.EventID, created_at: event.CreatedAt };
    }
  }
  return {};
}

function reportArtifactDesignedExportState(sourceArtifactID) {
  const events = state.detail?.events || [];
  const completedPending = completedReportDraftPendingEventIDs(events);
  let pending = null;
  let failed = null;
  for (let i = events.length - 1; i >= 0; i--) {
    const event = events[i];
    const payload = event.Payload || {};
    if (event.EventType === "report.artifact.exported" &&
      payload.kind === "designed_html_report_artifact" &&
      payload.source_artifact_id === sourceArtifactID &&
      payload.target === "designed_html" &&
      payload.renderer_version === DESIGNED_REPORT_RENDERER_VERSION) {
      return { state: "completed", payload: { ...payload, event_id: event.EventID, created_at: event.CreatedAt } };
    }
    if (!pending &&
      event.EventType === "report.design.pending" &&
      payload.source_artifact_id === sourceArtifactID &&
      !completedPending.has(event.EventID)) {
      pending = { ...payload, event_id: event.EventID, created_at: event.CreatedAt };
    }
    if (!failed &&
      event.EventType === "report.design.failed" &&
      payload.source_artifact_id === sourceArtifactID) {
      failed = { ...payload, event_id: event.EventID, created_at: event.CreatedAt };
    }
  }
  if (pending) return { state: "pending", payload: pending };
  if (failed) return { state: "failed", payload: failed };
  return { state: "idle", payload: {} };
}

function reportArtifactHumanizedExportState(sourceArtifactID) {
  const events = state.detail?.events || [];
  let failed = null;
  for (let i = events.length - 1; i >= 0; i--) {
    const event = events[i];
    const payload = event.Payload || {};
    if (event.EventType === "report.artifact.exported" &&
      payload.kind === "humanized_markdown_report_artifact" &&
      payload.source_artifact_id === sourceArtifactID &&
      payload.target === "humanized_markdown") {
      return { state: "completed", payload: { ...payload, event_id: event.EventID, created_at: event.CreatedAt } };
    }
    if (!failed &&
      event.EventType === "report.humanize.failed" &&
      payload.source_artifact_id === sourceArtifactID &&
      payload.target === "humanized_markdown") {
      failed = { ...payload, event_id: event.EventID, created_at: event.CreatedAt };
    }
    if (!failed &&
      event.EventType === "report.humanize.skipped" &&
      payload.source_artifact_id === sourceArtifactID &&
      payload.target === "humanized_markdown") {
      return { state: "skipped", payload: { ...payload, event_id: event.EventID, created_at: event.CreatedAt } };
    }
    if (!failed &&
      event.EventType === "report.humanize.pending" &&
      payload.source_artifact_id === sourceArtifactID &&
      payload.target === "humanized_markdown") {
      return { state: "pending", payload: { ...payload, event_id: event.EventID, created_at: event.CreatedAt } };
    }
  }
  if (failed) return { state: "failed", payload: failed };
  return { state: "idle", payload: {} };
}

function reportViewModel(version, index) {
  const versionID = version.report_version_id || "";
  const drafted = reportDraftedPayload(versionID);
  const generation = drafted.generation || {};
  const plan = reportPlanPayload(versionID);
  const trace = mcpTraceSummary(generation.tool_session_id || plan.tool_session_id || "");
  const exports = reportExportPayloads(versionID);
  const title = reportTitle(version) || "리포트";
  const createdLabel = version.created_at ? timeShort(version.created_at) : "생성 시각 없음";
  const exportTargets = exports.map((item) => exportTargetLabel(item.target)).filter(Boolean);
  const planData = plan.plan || {};
  const sectionCount = Array.isArray(planData.sections) ? planData.sections.length : 0;
  return {
    version,
    versionID,
    title,
    isLatest: index === 0,
    createdLabel,
    stateLabel: reportStateLabel(version.state),
    rigorLabel: generation.rigor_label || REPORT_RIGOR_LABELS[generation.rigor_level] || "",
    exportLabel: exportTargets.length ? `내보냄: ${exportTargets.join(", ")}` : "아직 내보낸 파일 없음",
    plan,
    trace,
    planLabel: planData.summary ? `${sectionCount}개 섹션 / ${planData.summary}` : "기록된 생성 계획 없음",
    traceLabel: trace.total ? `총 ${trace.total}회, 오류 ${trace.errors}회` : "기록된 MCP 호출 없음"
  };
}

function reportDraftedPayload(versionID) {
  const events = state.detail?.events || [];
  for (let i = events.length - 1; i >= 0; i--) {
    const event = events[i];
    if (event.EventType !== "report.drafted") continue;
    const payload = event.Payload || {};
    if (payload.report_version_id === versionID) return payload;
  }
  return {};
}

function reportPlanPayload(versionID) {
  const events = state.detail?.events || [];
  for (let i = events.length - 1; i >= 0; i--) {
    const event = events[i];
    if (event.EventType !== "report.plan.created") continue;
    const payload = event.Payload || {};
    if (payload.report_version_id === versionID) return payload;
  }
  return {};
}

function reportPlanPayloadByEventID(eventID) {
  const events = state.detail?.events || [];
  for (const event of events) {
    if (event.EventType !== "report.plan.created" || event.EventID !== eventID) continue;
    return { ...(event.Payload || {}), event_id: event.EventID, created_at: event.CreatedAt };
  }
  return {};
}

function mcpTraceEvents(toolSessionID) {
  if (!toolSessionID) return [];
  const events = state.detail?.events || [];
  return events.filter((event) => {
    if (event.EventType !== "mcp.tool.called") return false;
    const payload = event.Payload || {};
    return payload.tool_session_id === toolSessionID || payload.agent_session_id === toolSessionID;
  });
}

function mcpTraceSummary(toolSessionID) {
  const events = mcpTraceEvents(toolSessionID);
  const tools = new Map();
  let errors = 0;
  let totalDuration = 0;
  for (const event of events) {
    const payload = event.Payload || {};
    const name = payload.tool_name || "unknown";
    tools.set(name, (tools.get(name) || 0) + 1);
    if (payload.success === false) errors += 1;
    totalDuration += Number(payload.duration_ms || 0);
  }
  const toolCounts = [...tools.entries()]
    .map(([name, count]) => ({ name, count }))
    .sort((a, b) => b.count - a.count || a.name.localeCompare(b.name));
  return { toolSessionID, events, total: events.length, errors, totalDuration, toolCounts };
}

function renderTraceBars(trace) {
  if (!trace.total) return `<div class="trace-empty">추적 데이터 없음</div>`;
  const max = Math.max(...trace.toolCounts.map((item) => item.count), 1);
  return `
    <div class="trace-bars">
      ${trace.toolCounts.slice(0, 5).map((item) => `
        <div class="trace-bar" title="${escapeAttr(item.name)} ${item.count}회">
          <span>${escapeHTML(toolShortName(item.name))}</span>
          <i><b style="width:${Math.max(8, Math.round((item.count / max) * 100))}%"></b></i>
          <em>${item.count}</em>
        </div>
      `).join("")}
    </div>
  `;
}

function reportExportPayloads(versionID) {
  const events = state.detail?.events || [];
  return events
    .filter((event) => event.EventType === "report.exported")
    .map((event) => event.Payload || {})
    .filter((payload) => payload.report_version_id === versionID);
}

function reportTitle(version) {
  const report = (state.detail?.reports || []).find((item) => item.report_id === version.report_id);
  return report?.title || version.report_id || version.report_version_id || "";
}

function reportStateLabel(stateValue) {
  switch (stateValue) {
    case "draft":
      return "초안";
    case "export_candidate":
      return "내보내기 가능";
    default:
      return stateValue || "상태 미정";
  }
}

function exportTargetLabel(target) {
  switch (target) {
    case "markdown":
      return "MD";
    case "html":
      return "HTML";
    case "json_ast":
      return "JSON";
    default:
      return target || "";
  }
}

function toolShortName(name) {
  return String(name || "unknown")
    .replace(/^plasma\./, "")
    .replace(/^research\./, "research ")
    .replace(/^sources\./, "sources ")
    .replace(/\./g, " ");
}

function showReportPlan(versionID) {
  const plan = reportPlanPayload(versionID);
  showReportPlanPayload(plan);
}

function showReportPlanEvent(eventID) {
  const plan = reportPlanPayloadByEventID(eventID);
  showReportPlanPayload(plan);
}

function showReportPlanPayload(plan) {
  const data = plan.plan || {};
  state.detailText = JSON.stringify(plan || {}, null, 2);
  $("detailTitle").textContent = "리포트 생성 계획";
  if (!Object.keys(plan).length) {
    $("detailBody").innerHTML = `<p class="detail-meta">이 리포트 버전에는 저장된 생성 계획이 없습니다.</p>`;
    $("detailModal").classList.remove("hidden");
    return;
  }
  const sections = Array.isArray(data.sections) ? data.sections : [];
  $("detailBody").innerHTML = `
    <section class="detail-section">
      <h3>계획 요약</h3>
      <p>${escapeHTML(data.summary || "요약 없음")}</p>
      <div class="detail-meta">${escapeHTML(plan.agent_executor || "")} / ${escapeHTML(plan.tool_session_id || "")} / ${escapeHTML(plan.duration_ms || 0)}ms</div>
    </section>
    <section class="detail-section">
      <h3>섹션 계획</h3>
      ${sections.length ? sections.map((section) => `
        <div class="trace-entry">
          <div class="trace-entry-head">
            <strong>${escapeHTML(section.title || "제목 없음")}</strong>
          </div>
          <p>${escapeHTML(section.purpose || "목적 없음")}</p>
          ${detailChips(refValues(section.target_refs || {}))}
        </div>
      `).join("") : `<p class="detail-meta">섹션 계획 없음</p>`}
    </section>
    <section class="detail-grid">
      <div class="detail-box">
        <h3>활용 메모</h3>
        ${detailList(data.coverage_notes || [], "기록된 활용 메모 없음")}
      </div>
      <div class="detail-box">
        <h3>누락/한계</h3>
        ${detailList(data.planned_omissions || [], "기록된 누락 사항 없음")}
      </div>
    </section>
  `;
  $("detailModal").classList.remove("hidden");
}

function showMCPTrace(versionID) {
  const drafted = reportDraftedPayload(versionID);
  const plan = reportPlanPayload(versionID);
  const toolSessionID = drafted.generation?.tool_session_id || plan.tool_session_id || "";
  const trace = mcpTraceSummary(toolSessionID);
  state.detailText = JSON.stringify(trace.events, null, 2);
  $("detailTitle").textContent = "MCP 호출 추적";
  if (!trace.total) {
    $("detailBody").innerHTML = `<p class="detail-meta">이 리포트 버전에 연결된 MCP 호출 기록이 없습니다.</p>`;
    $("detailModal").classList.remove("hidden");
    return;
  }
  $("detailBody").innerHTML = `
    <section class="detail-section">
      <h3>요약</h3>
      <div class="chip-row">
        <span class="badge">총 ${trace.total}회</span>
        <span class="badge ${trace.errors ? "warn" : "muted"}">오류 ${trace.errors}회</span>
        <span class="badge muted">${Math.round(trace.totalDuration)}ms</span>
        <span class="badge muted">${escapeHTML(toolSessionID)}</span>
      </div>
      ${renderTraceBars(trace)}
    </section>
    <section class="detail-section">
      <h3>호출 목록</h3>
      <div class="trace-list">
        ${trace.events.map(renderTraceEvent).join("")}
      </div>
    </section>
  `;
  $("detailModal").classList.remove("hidden");
}

function renderTraceEvent(event) {
  const payload = event.Payload || {};
  const result = payload.result || {};
  const args = payload.arguments || {};
  const ok = payload.success !== false;
  return `
    <div class="trace-entry ${ok ? "" : "failed"}">
      <div class="trace-entry-head">
        <strong>${escapeHTML(toolShortName(payload.tool_name || "unknown"))}</strong>
        <span class="badge ${ok ? "muted" : "warn"}">${ok ? "성공" : "실패"}</span>
      </div>
      <div class="detail-meta">#${escapeHTML(event.Sequence || "")} / ${escapeHTML(timeShort(event.CreatedAt))} / ${escapeHTML(payload.duration_ms || 0)}ms</div>
      <div class="trace-args">${escapeHTML(traceArgSummary(args))}</div>
      ${result.error ? `<div class="trace-error">${escapeHTML(result.error.message || "오류 메시지 없음")}</div>` : ""}
    </div>
  `;
}

function traceArgSummary(args) {
  const parts = [];
  for (const key of ["object_kind", "object_id", "query", "snapshot_id", "artifact_id", "claim_id", "evidence_id", "limit", "offset", "max_bytes"]) {
    if (args[key] === undefined || args[key] === null || args[key] === "") continue;
    parts.push(`${key}=${JSON.stringify(args[key])}`);
  }
  return parts.length ? parts.join(" / ") : "핵심 인자 없음";
}

function refValues(refs) {
  const values = [];
  for (const key of ["claim_ids", "evidence_ids", "snapshot_ids", "question_ids", "option_ids"]) {
    for (const value of refs[key] || []) {
      if (value && !values.includes(value)) values.push(value);
    }
  }
  return values;
}

function detailList(items, emptyText) {
  if (!Array.isArray(items) || !items.length) return `<p class="detail-meta">${escapeHTML(emptyText)}</p>`;
  return `<ul>${items.map((item) => `<li>${escapeHTML(item)}</li>`).join("")}</ul>`;
}

function renderReportDraftStatus(status, wasPending) {
  if (status.state === "pending") {
    setReportBusy(true);
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
      setReportNotice(`${prefix}${reportTimingDetails(status.event)}\n\n${payload.error || payload.text || "패치 실패 사유 없음"}`, payload.canceled === true ? undefined : "error");
    } else if (payload.canceled === true) {
      setReportNotice(`리포트 생성이 취소되었습니다.${reportTimingDetails(status.event)}\n\n${payload.text || "사용자가 리포트 생성을 취소했습니다."}`);
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
    setReportNotice(`Markdown 리포트 artifact 생성이 완료되었습니다.${reportTimingDetails(status.event)}\n\n최신 리포트 카드에서 미리보기를 확인하세요.`);
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
  const rigor = payload.rigor_label || REPORT_RIGOR_LABELS[payload.rigor_level] || "";
  const rigorLine = rigor ? `\n엄격도: ${rigor}` : "";
  const mode = payload.report_mode || "planned";
  const modeLabel = payload.report_mode_label || REPORT_MODE_LABELS[mode] || "보고서";
  const modeLine = `\n방식: ${modeLabel}`;
  const workLine = mode === "long_form"
    ? "에이전트가 계획을 만든 뒤 섹션별로 작성하고, 섹션 본문을 보존한 채 파트와 최종 Markdown artifact를 조립하는 중입니다."
    : "에이전트가 생성 계획을 만든 뒤 MCP 읽기 도구로 필요한 소스를 확인하고 Markdown artifact를 작성하는 중입니다.";
  const eventID = event?.EventID ? `\n대기 이벤트: ${event.EventID}` : "";
  const timing = reportTimingDetails(event);
  return [
    "리포트 초안 생성 요청을 보냈습니다.",
    workLine,
    "완료되면 아래 리포트 목록에 새 artifact 기록이 추가됩니다."
  ].join("\n") + title + modeLine + rigorLine + timing + eventID;
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

function renderCandidateSourceOptions(sources) {
  const select = $("candidateSource");
  const candidates = (sources || []).filter((source) => {
    const sourceState = source.State || source.state || {};
    const removed = Boolean(sourceState.removed || sourceState.Removed || sourceState.state === "removed" || sourceState.State === "removed");
    return !removed && (source.ArtifactIDs || source.artifact_ids || []).length > 0;
  });
  if (!candidates.length) {
    select.innerHTML = `<option value="">먼저 소스를 추가하세요</option>`;
    select.disabled = true;
    return;
  }
  select.disabled = false;
  select.innerHTML = candidates.map((source) => {
    const snapshotID = source.SnapshotID;
    const artifactID = (source.ArtifactIDs || [])[0] || "";
    const title = source.Title || snapshotID;
    return `<option value="${escapeAttr(snapshotID + "|" + artifactID)}">${escapeHTML(title)} / ${escapeHTML(snapshotID)}</option>`;
  }).join("");
}

function renderLedger(events) {
  updateCountChip("ledgerCount", events.length);
  updateCountChip("ledgerTabCount", events.length);
  const recent = events.slice(-150).reverse();
  $("ledgerList").innerHTML = recent.length ? recent.map((event) => `
    <button type="button" class="ledger-row" data-detail-title="장부 이벤트 상세" data-detail-json="${escapeAttr(JSON.stringify(event))}" title="${escapeAttr(event.EventID || "")}">
      <span class="ledger-seq">#${escapeHTML(String(event.Sequence ?? ""))}</span>
      <span class="ledger-type">${escapeHTML(ledgerEventLabel(event))}</span>
      <span class="ledger-time">${escapeHTML(ledgerTime(event.CreatedAt))}</span>
      <span class="ledger-id">${escapeHTML(shortID(event.EventID || ""))}</span>
    </button>
  `).join("") : empty("장부 이벤트 없음");
}

function ledgerEventLabel(event) {
  if (!event || event.EventType !== "mcp.tool.called") return event?.EventType || "";
  const payload = event.Payload || {};
  const toolName = payload.tool_name || "unknown";
  const status = payload.success === false ? "실패" : "성공";
  return `MCP 호출 · ${toolName} · ${status}`;
}

function ledgerTime(value) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  const pad = (n) => String(n).padStart(2, "0");
  return `${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function approvedEvidence(proposals, evidence) {
  const ids = new Set();
  for (const proposal of proposals) {
    if (proposal.state !== "approved" && proposal.state !== "partially_approved") continue;
    for (const ref of proposal.object_refs || []) {
      if (ref.object_kind === "evidence_record") ids.add(ref.object_id);
    }
  }
  return evidence.filter((record) => record.state === "approved" || ids.has(record.evidence_id));
}

function approvedClaims(proposals, claims) {
  const ids = new Set();
  for (const proposal of proposals) {
    if (proposal.state !== "approved" && proposal.state !== "partially_approved") continue;
    for (const ref of proposal.object_refs || []) {
      if (ref.object_kind === "claim_record") ids.add(ref.object_id);
    }
  }
  return claims.filter((record) => record.state === "approved" || ids.has(record.claim_id));
}

function setFormsEnabled(enabled) {
  for (const id of ["turnText", "agentExecutor", "agentModel", "agentReasoningEffort", "mcpMode", "controllerStrategy", "resetAgentSessionButton", "confluenceAccessConnectionSelect", "confluenceAccessSiteSelect", "confluenceAccessSpaceKey", "confluenceAccessEnable", "confluenceAccessDisable", "workflowInstruction", "workflowStepInstructionMode", "draftWorkflowGoalButton", "workflowRunGoal", "workflowStepInstruction", "startWorkflowButton", "stopWorkflowButton", "sourceTitle", "sourceURI", "sourceContent", "sourceUploadFile", "sourceUploadTitle", "sourceFetchURLButton", "mediaSourceURL", "mediaSourceTitle", "mediaSourceLicense", "mediaSourceAttribution", "pdfSourceURL", "pdfSourceTitle", "localPathRoot", "localPathRelativePath", "localPathTitle", "localPathRestore", "localPathTreeButton", "localPathAttachButton", "confluenceConnectionSelect", "confluenceRefreshConnections", "openConfluenceSettings", "confluenceOneClickStart", "confluenceSiteSelect", "confluencePageURL", "confluenceAddURLButton", "confluenceLoadSpaces", "confluenceLoadMoreSpaces", "confluenceLoadMorePages", "confluenceQuery", "confluenceSpaceKey", "confluenceLimit", "confluenceRangeSelect", "confluenceUpdateRangeSelect", "liquid2Query", "candidateSource", "candidateEvidenceType", "candidateSummary", "reportRigor", "draftQuickReport", "draftLongReport", "cancelReportButton"]) {
    const el = $(id);
    if (el) {
      el.disabled = !enabled ||
        (id === "agentExecutor" && Boolean(lockedAgentExecutor())) ||
        (id === "agentReasoningEffort" && agentReasoningEffortSelectionDisabled(false)) ||
        (state.reportPending && ["turnText", "agentExecutor", "agentModel", "agentReasoningEffort", "mcpMode", "controllerStrategy", "resetAgentSessionButton", "workflowInstruction", "workflowStepInstructionMode", "draftWorkflowGoalButton", "workflowRunGoal", "workflowStepInstruction", "startWorkflowButton", "draftQuickReport", "draftLongReport", "reportRigor"].includes(id)) ||
        (state.workflowGoalDraftPending && ["turnText", "workflowInstruction", "workflowStepInstructionMode", "draftWorkflowGoalButton", "workflowRunGoal", "workflowStepInstruction", "startWorkflowButton"].includes(id)) ||
        (state.turnPending && ["turnText", "agentExecutor", "agentModel", "agentReasoningEffort", "mcpMode", "controllerStrategy", "resetAgentSessionButton"].includes(id)) ||
        (state.workflowPending && ["turnText", "agentExecutor", "agentModel", "agentReasoningEffort", "mcpMode", "controllerStrategy", "resetAgentSessionButton", "workflowInstruction", "workflowStepInstructionMode", "draftWorkflowGoalButton", "workflowRunGoal", "workflowStepInstruction", "startWorkflowButton"].includes(id));
    }
  }
  for (const form of ["turnForm", "sourceForm", "sourceUploadForm", "mediaSourceForm", "pdfSourceForm", "localPathForm", "confluenceURLForm", "confluenceSearchForm", "liquid2Form", "candidateForm"]) {
    for (const button of $(form).querySelectorAll("button")) {
      button.disabled = !enabled || ((state.turnPending || state.workflowPending || state.reportPending) && button.id === "sendTurnButton");
    }
  }
  if (enabled) setTurnBusy(state.turnPending);
  setReportBusy(state.reportPending);
  setWorkflowBusy(state.workflowPending);
}

function setTurnBusy(busy) {
  $("turnStatus").classList.toggle("hidden", !busy);
  $("cancelTurnButton").classList.toggle("hidden", !busy);
  const blocked = busy || state.workflowPending || state.workflowGoalDraftPending || state.reportPending || !state.detail;
  $("turnText").disabled = blocked;
  $("agentExecutor").disabled = agentExecutorSelectionDisabled(blocked);
  $("agentModel").disabled = blocked;
  $("agentReasoningEffort").disabled = agentReasoningEffortSelectionDisabled(blocked);
  $("mcpMode").disabled = blocked;
  $("controllerStrategy").disabled = blocked;
  $("resetAgentSessionButton").disabled = blocked;
  $("sendTurnButton").disabled = blocked;
  $("sendTurnButton").textContent = busy ? "대기 중" : "보내기";
}

function setReportBusy(busy) {
  state.reportPending = busy;
  $("reportStatus").classList.toggle("hidden", !busy);
  $("reportRigor").disabled = busy || !state.detail;
  $("draftQuickReport").disabled = busy || !state.detail;
  $("draftLongReport").disabled = busy || !state.detail;
  $("cancelReportButton").disabled = !busy || !state.detail;
  $("cancelReportButton").classList.toggle("hidden", !busy);
  $("draftQuickReport").textContent = busy ? "생성 중" : "보고서";
  $("draftLongReport").textContent = busy ? "생성 중" : "장문 보고서";
}

function setWorkflowBusy(busy) {
  state.workflowPending = busy;
  const draftBusy = state.workflowGoalDraftPending;
  const layered = workflowStepInstructionMode() === "layered";
  $("workflowInstruction").disabled = busy || draftBusy || state.reportPending || !state.detail;
  $("workflowStepInstructionMode").disabled = busy || draftBusy || state.reportPending || !state.detail;
  $("draftWorkflowGoalButton").disabled = !layered || busy || draftBusy || state.reportPending || !state.detail;
  $("workflowRunGoal").disabled = !layered || busy || draftBusy || state.reportPending || !state.detail;
  $("workflowStepInstruction").disabled = !layered || busy || draftBusy || state.reportPending || !state.detail;
  $("startWorkflowButton").disabled = busy || draftBusy || state.reportPending || !state.detail;
  $("stopWorkflowButton").disabled = !busy || !state.detail;
  $("startWorkflowButton").textContent = busy ? "진행 중" : "시작";
  $("draftWorkflowGoalButton").textContent = draftBusy ? "초안 생성 중" : "목표 초안 생성";
}

function schedulePendingPoll() {
  clearPendingPoll();
  if ((!state.turnPending && !state.reportPending && !state.workflowPending) || !state.missionId) return;
  state.pollTimer = window.setTimeout(async () => {
    if (state.pollInFlight || (!state.turnPending && !state.reportPending && !state.workflowPending) || !state.missionId) return;
    state.pollInFlight = true;
    try {
      await reloadMission();
      $("healthBadge").textContent = "정상";
    } catch (err) {
      console.warn("pending poll failed", err);
      $("healthBadge").textContent = "재연결 중";
    } finally {
      state.pollInFlight = false;
      if (state.turnPending || state.reportPending || state.workflowPending) schedulePendingPoll();
    }
  }, 2000);
}

function clearPendingPoll() {
  if (!state.pollTimer) return;
  window.clearTimeout(state.pollTimer);
  state.pollTimer = 0;
}

function renderTabs() {
  for (const tab of document.querySelectorAll("[data-tab]")) {
    tab.classList.toggle("active", tab.dataset.tab === state.activeTab);
  }
  for (const panel of document.querySelectorAll("[data-tab-panel]")) {
    panel.classList.toggle("active", panel.dataset.tabPanel === state.activeTab);
  }
}

function onTabBarClick(event) {
  const tab = event.target.closest("[data-tab]");
  if (!tab) return;
  state.activeTab = tab.dataset.tab;
  renderTabs();
  if (state.activeTab === "settings") loadConfluenceConnections();
}

function onMissionListClick(event) {
  const button = event.target.closest("[data-mission-id]");
  if (button) selectMission(button.dataset.missionId);
}

function onLiquid2ResultsClick(event) {
  if (onDetailButtonClick(event)) return;
  const button = event.target.closest("[data-liquid2-source-id]");
  if (button) attachLiquid2(button.dataset.liquid2SourceId);
}

function onConfluenceResultsClick(event) {
  if (onDetailButtonClick(event)) return;
  const button = event.target.closest("[data-confluence-candidate-index]");
  if (button) previewConfluenceCandidate(button.dataset.confluenceCandidateIndex);
}

function onLocalPathTreeClick(event) {
  const pick = event.target.closest("[data-local-path-pick]");
  if (!pick || pick.disabled) return;
  const kind = pick.dataset.localPathKind;
  if (kind === "up") {
    const parent = localPathParent(state.localPathCurrentDir || ".");
    $("localPathRelativePath").value = parent === "." ? "" : parent;
    state.localPathSelectedFile = "";
    updateLocalPathAttachState();
    browseLocalPathTree();
    return;
  }
  const rel = pick.dataset.localPathPick || "";
  if (kind === "dir") {
    $("localPathRelativePath").value = rel;
    state.localPathSelectedFile = "";
    updateLocalPathAttachState();
    browseLocalPathTree();
    return;
  }
  // File: select it, auto-fill the title, highlight without refetching.
  state.localPathSelectedFile = rel;
  $("localPathRelativePath").value = rel;
  if (!$("localPathTitle").value.trim()) {
    $("localPathTitle").value = rel.split("/").pop() || rel;
  }
  for (const el of $("localPathTree").querySelectorAll(".local-path-entry")) {
    el.classList.toggle("selected", el.dataset.localPathKind === "file" && el.dataset.localPathPick === rel);
  }
  updateLocalPathAttachState();
}

async function onSourceListClick(event) {
  if (onDetailButtonClick(event)) return;
  const confluenceUpdateButton = event.target.closest("[data-confluence-source-update]");
  if (confluenceUpdateButton) {
    await checkConfluenceSourceUpdate(confluenceUpdateButton.dataset.confluenceSourceUpdate);
    return;
  }
  const readButton = event.target.closest("[data-source-read]");
  if (readButton) {
    await readSource(readButton.dataset.sourceRead);
    return;
  }
  const removeButton = event.target.closest("[data-source-remove]");
  if (removeButton) {
    await removeSource(removeButton.dataset.sourceRemove);
    return;
  }
  const restoreButton = event.target.closest("[data-source-restore]");
  if (restoreButton) {
    await restoreSource(restoreButton.dataset.sourceRestore);
  }
}

async function onSourceCandidateListClick(event) {
  if (onDetailButtonClick(event)) return;
  const addButton = event.target.closest("[data-add-source-url]");
  if (addButton) {
    await addURLSource(addButton.dataset.addSourceUrl, addButton.dataset.sourceCandidateTitle || "");
    return;
  }
  const rejectButton = event.target.closest("[data-reject-source-url]");
  if (rejectButton) {
    await rejectSourceCandidate(rejectButton.dataset.rejectSourceUrl);
  }
}

async function onRejectedSourceCandidateListClick(event) {
  if (onDetailButtonClick(event)) return;
  const restoreButton = event.target.closest("[data-restore-source-url]");
  if (restoreButton) {
    await restoreSourceCandidate(restoreButton.dataset.restoreSourceUrl);
  }
}

function onProposalListClick(event) {
  if (onDetailButtonClick(event)) return;
  const button = event.target.closest("[data-proposal-id][data-action]");
  if (button) decideProposal(button.dataset.proposalId, button.dataset.action);
}

function onReportListClick(event) {
  if (onDetailButtonClick(event)) return;
  const planButton = event.target.closest("[data-report-plan-event-id][data-action]");
  if (planButton) {
    showReportPlanEvent(planButton.dataset.reportPlanEventId);
    return;
  }
  const artifactButton = event.target.closest("[data-report-artifact-id][data-action]");
  if (artifactButton) {
    const artifactID = artifactButton.dataset.reportArtifactId;
    if (artifactButton.dataset.action === "download-artifact") {
      downloadReportArtifact(artifactID);
    } else if (artifactButton.dataset.action === "view-html-artifact") {
      exportReportArtifactHTML(artifactID);
    } else if (artifactButton.dataset.action === "download-html-artifact") {
      exportReportArtifactHTML(artifactID, { download: true });
    } else if (artifactButton.dataset.action === "patch-artifact") {
      patchReportArtifact(artifactID, artifactButton.dataset.reportTitle || "");
    } else if (artifactButton.dataset.action === "start-humanized-markdown-artifact") {
      exportReportArtifactHumanizedMarkdown(artifactID);
    } else if (artifactButton.dataset.action === "view-designed-html-artifact" || artifactButton.dataset.action === "start-designed-html-artifact") {
      exportReportArtifactDesignedHTML(artifactID);
    } else if (artifactButton.dataset.action === "download-designed-html-artifact") {
      exportReportArtifactDesignedHTML(artifactID, { download: true });
    } else {
      viewReportArtifact(artifactID);
    }
    return;
  }
  const button = event.target.closest("[data-report-version-id][data-action]");
  if (button) {
    const versionID = button.dataset.reportVersionId;
    const action = button.dataset.action;
    if (action === "ast") {
      viewReportAST(versionID);
    } else if (action === "plan") {
      showReportPlan(versionID);
    } else if (action === "mcp-trace") {
      showMCPTrace(versionID);
    } else if (action.startsWith("download-")) {
      exportReport(versionID, action.slice("download-".length), { download: true });
    } else {
      exportReport(versionID, action);
    }
    return;
  }
  // Click on a card header/body (not a button or link) → select it (accordion).
  if (event.target.closest("a")) return;
  // A 도구/받기 menu summary toggles its own <details>; don't treat it as a
  // card selection (which would re-render and immediately close the menu).
  if (event.target.closest("summary")) return;
  const card = event.target.closest("[data-report-key]");
  if (card) selectReport(card.dataset.reportKey);
}

function onTurnLogClick(event) {
  const button = event.target.closest("[data-copy-text]");
  if (!button) return;
  copyText(button.dataset.copyText).catch(showError);
}

function onDetailButtonClick(event) {
  const confidenceButton = event.target.closest("[data-confidence-claim-id]");
  if (confidenceButton) {
    showClaimConfidenceDetail(confidenceButton.dataset.confidenceClaimId);
    return true;
  }
  const button = event.target.closest("[data-detail-json]");
  if (!button) return false;
  const title = button.dataset.detailTitle || "상세 보기";
  try {
    showDetail(title, JSON.parse(button.dataset.detailJson));
  } catch (err) {
    showDetail(title, button.dataset.detailJson || "");
  }
  return true;
}

function requireMission() {
  if (state.missionId) return true;
  showError(new Error("먼저 미션을 선택하거나 만들어야 합니다."));
  return false;
}

function showError(err) {
  state.lastError = err.userMessage || err.message || String(err);
  if (!err.userMessage && err.stack) {
    state.lastError = err.stack;
  }
  if (err.details && !err.userMessage) {
    state.lastError += "\n\n" + JSON.stringify(err.details, null, 2);
  }
  $("errorText").textContent = state.lastError;
  // Transient connection failures (e.g. on browser wake / session recovery, or
  // a brief server restart) are surfaced by the health badge, not a toast.
  if (err && err.isNetworkError) {
    const badge = $("healthBadge");
    if (badge) badge.textContent = "연결 끊김";
    return;
  }
  $("errorToast").classList.remove("hidden");
}

async function copyError() {
  try {
    await copyText(state.lastError || $("errorText").textContent);
  } catch (err) {
    $("errorText").textContent += "\n\nclipboard copy failed: " + err.message;
  }
}

function showMissionRecall() {
  if (!state.detail) {
    showError(new Error("먼저 미션을 선택하거나 만들어야 합니다."));
    return;
  }
  showDetail("현재 미션 상태", state.detail.recall || {});
}

function showDetail(title, value) {
  $("detailTitle").textContent = title;
  const text = typeof value === "string" ? value : JSON.stringify(value, null, 2);
  state.detailText = text;
  $("detailBody").innerHTML = `<pre>${escapeHTML(text)}</pre>`;
  $("detailModal").classList.remove("hidden");
}

function showClaimConfidenceDetail(claimID) {
  const records = state.detail?.records || {};
  const claim = (records.claims || []).find((record) => record.claim_id === claimID);
  const view = (records.claim_confidence || []).find((item) => item.claim_id === claimID) || initialConfidenceView(claim || { claim_id: claimID });
  const current = view.current_confidence || {};
  const initial = view.initial_confidence || {};
  const risks = current.open_risks || [];
  const history = view.history || [];
  const latestHistory = history.length ? history[history.length - 1] : null;
  const data = { claim, confidence: view };
  state.detailText = JSON.stringify(data, null, 2);
  $("detailTitle").textContent = "주장 신뢰도 상세";
  $("detailBody").innerHTML = `
    <section class="detail-section">
      <h3>주장</h3>
      <p>${escapeHTML(claim?.text || claimID)}</p>
      <div class="detail-meta">${escapeHTML(claimID)}</div>
    </section>
    <section class="detail-grid">
      <div class="detail-box">
        <h3>현재 신뢰도</h3>
        ${confidenceBadge(view)}
        <p>${escapeHTML(current.rationale || "현재 신뢰도 판단 사유가 없습니다.")}</p>
      </div>
      <div class="detail-box">
        <h3>초기 신뢰도</h3>
        <span class="badge confidence ${escapeAttr(initial.level || "unknown")}">${escapeHTML(confidenceLabel(initial.level || "unknown"))}</span>
        <p>${escapeHTML(initial.rationale || "초기 신뢰도 판단 사유가 없습니다.")}</p>
      </div>
    </section>
    <section class="detail-section">
      <h3>판단 근거</h3>
      ${detailChips(latestHistory?.basis_evidence_ids || [])}
    </section>
    <section class="detail-section">
      <h3>열린 위험</h3>
      ${risks.length ? `<ul>${risks.map((risk) => `<li>${escapeHTML(risk)}</li>`).join("")}</ul>` : `<p class="detail-meta">열린 위험 없음</p>`}
    </section>
    <section class="detail-section">
      <h3>변경 이력</h3>
      ${history.length ? history.map(renderConfidenceHistoryEntry).join("") : `<p class="detail-meta">아직 변경 이력이 없습니다.</p>`}
    </section>
  `;
  $("detailModal").classList.remove("hidden");
}

function renderConfidenceHistoryEntry(entry) {
  return `
    <div class="history-entry">
      <div class="confidence-line">
        <span class="badge confidence ${escapeAttr(entry.level || "unknown")}">${escapeHTML(confidenceLabel(entry.level || "unknown"))}${escapeHTML(directionGlyph(entry.direction))}</span>
        <span class="item-meta">${escapeHTML(timeShort(entry.created_at))} / ${escapeHTML(entry.origin || "unknown")} / ${escapeHTML(entry.event_id || "")}</span>
      </div>
      <div>${escapeHTML(entry.rationale || "변경 사유 없음")}</div>
      ${detailChips(entry.basis_evidence_ids || [])}
    </div>
  `;
}

function detailChips(values) {
  if (!values.length) return `<p class="detail-meta">연결된 근거 없음</p>`;
  return `<div class="chip-row">${values.map((value) => `<span class="badge muted">${escapeHTML(value)}</span>`).join("")}</div>`;
}

async function copyDetail() {
  try {
    await copyText(state.detailText || $("detailBody").textContent);
  } catch (err) {
    showError(err);
  }
}

function hideDetail() {
  $("detailModal").classList.add("hidden");
  const card = $("detailModal").querySelector(".modal-card");
  if (card) card.classList.remove("modal-card--wide");
}

function onDetailModalClick(event) {
  if (event.target.id === "detailModal") hideDetail();
}

function hideError() {
  $("errorToast").classList.add("hidden");
}

function empty(text) {
  return `<div class="item"><div class="item-meta">${escapeHTML(text)}</div></div>`;
}

function shortID(value) {
  const text = String(value || "");
  if (text.length <= 12) return text;
  return `${text.slice(0, 8)}…${text.slice(-4)}`;
}

function timeShort(value) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

function escapeHTML(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function escapeAttr(value) {
  return escapeHTML(value).replaceAll("`", "&#096;");
}

function renderMarkdown(value) {
  const text = String(value ?? "");
  if (!markdownRenderer || !window.DOMPurify) {
    return escapeHTML(text);
  }
  const rendered = markdownRenderer.render(text);
  return window.DOMPurify.sanitize(rendered, {
    USE_PROFILES: { html: true },
    ADD_ATTR: ["target", "rel"]
  });
}

function normalizeSourceURL(value) {
  const text = String(value || "").trim();
  if (!text) return "";
  try {
    const url = new URL(text);
    if (url.protocol !== "http:" && url.protocol !== "https:") return "";
    url.protocol = url.protocol.toLowerCase();
    url.hostname = url.hostname.toLowerCase();
    url.hash = "";
    return url.toString();
  } catch (err) {
    return "";
  }
}

async function copyText(value) {
  const text = String(value ?? "");
  // Only use the async Clipboard API in a secure context; over plain HTTP
  // (e.g. a non-loopback dev server) it is unavailable or rejects, so fall back
  // to the execCommand path instead of failing.
  if (window.isSecureContext && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text);
      return;
    } catch {
      /* fall through to the textarea fallback */
    }
  }
  const textarea = document.createElement("textarea");
  textarea.value = text;
  textarea.setAttribute("readonly", "");
  textarea.style.position = "fixed";
  textarea.style.left = "-9999px";
  document.body.appendChild(textarea);
  textarea.select();
  const copied = document.execCommand("copy");
  document.body.removeChild(textarea);
  if (!copied) {
    throw new Error("clipboard API is not available");
  }
}

// ── Multi-select bulk actions (source candidates + evidence proposals) ──
function pruneSelectedSourceCandidates(candidates) {
  const valid = new Set();
  for (const c of candidates) {
    const k = normalizeSourceURL(c.url);
    if (k) valid.add(k);
  }
  for (const url of [...state.selectedSourceCandidates]) {
    if (!valid.has(url)) state.selectedSourceCandidates.delete(url);
  }
}

function pruneSelectedProposals(pending) {
  const valid = new Set(pending.map((p) => p.proposal_id));
  for (const id of [...state.selectedProposals]) {
    if (!valid.has(id)) state.selectedProposals.delete(id);
  }
}

function updateSourceCandidateBulkBar() {
  const bar = $("sourceCandidateBulk");
  if (!bar) return;
  const n = state.selectedSourceCandidates.size;
  const countEl = $("sourceCandidateBulkCount");
  if (countEl) countEl.textContent = String(n);
  bar.classList.toggle("hidden", n === 0);
}

function updateProposalBulkBar() {
  const bar = $("proposalBulk");
  if (!bar) return;
  const n = state.selectedProposals.size;
  const countEl = $("proposalBulkCount");
  if (countEl) countEl.textContent = String(n);
  bar.classList.toggle("hidden", n === 0);
}

function toggleSourceCandidateSelection(checkbox) {
  const url = checkbox?.dataset?.selectSourceUrl;
  if (!url) return;
  if (checkbox.checked) state.selectedSourceCandidates.add(url);
  else state.selectedSourceCandidates.delete(url);
  if (checkbox.parentElement) {
    checkbox.parentElement.classList.toggle("selected", checkbox.checked);
  }
  updateSourceCandidateBulkBar();
}

function toggleProposalSelection(checkbox) {
  const id = checkbox?.dataset?.selectProposalId;
  if (!id) return;
  if (checkbox.checked) state.selectedProposals.add(id);
  else state.selectedProposals.delete(id);
  if (checkbox.parentElement) {
    checkbox.parentElement.classList.toggle("selected", checkbox.checked);
  }
  updateProposalBulkBar();
}

function selectAllSourceCandidates() {
  $("sourceCandidateList")
    .querySelectorAll("input.item-select[data-select-source-url]")
    .forEach((cb) => {
      cb.checked = true;
      state.selectedSourceCandidates.add(cb.dataset.selectSourceUrl);
      if (cb.parentElement) cb.parentElement.classList.add("selected");
    });
  updateSourceCandidateBulkBar();
}

function selectAllProposals() {
  $("proposalList")
    .querySelectorAll("input.item-select[data-select-proposal-id]")
    .forEach((cb) => {
      cb.checked = true;
      state.selectedProposals.add(cb.dataset.selectProposalId);
      if (cb.parentElement) cb.parentElement.classList.add("selected");
    });
  updateProposalBulkBar();
}

function clearSourceCandidateSelection() {
  state.selectedSourceCandidates.clear();
  $("sourceCandidateList")
    .querySelectorAll("input.item-select:checked")
    .forEach((cb) => {
      cb.checked = false;
      if (cb.parentElement) cb.parentElement.classList.remove("selected");
    });
  updateSourceCandidateBulkBar();
}

function clearProposalSelection() {
  state.selectedProposals.clear();
  $("proposalList")
    .querySelectorAll("input.item-select:checked")
    .forEach((cb) => {
      cb.checked = false;
      if (cb.parentElement) cb.parentElement.classList.remove("selected");
    });
  updateProposalBulkBar();
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

async function bulkSourceCandidateAction(action) {
  if (!requireMission()) return;
  const urls = [...state.selectedSourceCandidates];
  if (urls.length === 0) return;
  const rejectionReason = action === "reject"
    ? window.prompt("선택한 후보에 남길 기각 사유를 입력하세요. 비워두면 기본 사유로 기록됩니다.", "")
    : "";
  if (rejectionReason === null) return;
  const errors = action === "approve"
    ? await runBulkSequential(urls, async (url) => {
        const added = await addURLSource(url, sourceCandidateTitleForURL(url));
        if (!added) throw new Error(`소스 추가 실패: ${url}`);
      })
    : await runBulkSequential(urls, (url) =>
        api(`/api/missions/${state.missionId}/candidates/sources/reject`, {
          method: "POST",
          body: { url, reason: rejectionReason.trim() }
        })
      );
  state.selectedSourceCandidates.clear();
  await reloadMission();
  if (errors.length > 0) {
    const sample = errors.slice(0, 3).map((e) => e?.message || String(e)).join("; ");
    showError(new Error(`소스 후보 ${urls.length}개 중 ${errors.length}개 처리 실패: ${sample}`));
  }
}

async function bulkProposalAction(action) {
  if (!requireMission()) return;
  const ids = [...state.selectedProposals];
  if (ids.length === 0) return;
  const errors = await runBulkSequential(ids, (id) =>
    api(`/api/missions/${state.missionId}/proposals/${id}/${action}`, {
      method: "POST",
      body: {}
    })
  );
  state.selectedProposals.clear();
  await reloadMission();
  if (errors.length > 0) {
    const sample = errors.slice(0, 3).map((e) => e?.message || String(e)).join("; ");
    showError(new Error(`검토 후보 ${ids.length}개 중 ${errors.length}개 처리 실패: ${sample}`));
  }
}
