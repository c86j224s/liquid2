function codexStatusForModelDefaults() {
  const statuses = state.modelDefaultAgentExecutors || [];
  return statuses.find((status) => status.name === "codex") || {
    name: "codex",
    label: "Codex",
    configured: true,
    default_model: "gpt-5.5",
    default_model_label: "GPT-5.5",
    default_model_version: "gpt-5.5",
    default_reasoning_effort: "medium",
    reasoning_effort_supported: true,
    models: []
  };
}

function modelDefaultLabel(status) {
  const label = String(status?.default_model_label || "").trim();
  const version = String(status?.default_model_version || status?.default_model || "").trim();
  if (label && version && label !== version) return `${label}, ${version}`;
  return label || version || "provider 기본값";
}

function workflowGoalSelectedModel() {
  return ($("workflowGoalDefaultModel")?.value || "").trim();
}

function workflowGoalEffortOptions(status) {
  const model = workflowGoalSelectedModel() || String(status?.default_model || "").trim();
  const catalog = Array.isArray(status?.models) ? status.models : [];
  const selected = catalog.find((item) => String(item.name || "").trim() === model);
  const efforts = Array.isArray(selected?.reasoning_efforts) && selected.reasoning_efforts.length
    ? selected.reasoning_efforts
    : ["low", "medium", "high", "xhigh"];
  return efforts.map((effort) => String(effort).trim()).filter(Boolean);
}

function renderModelDefaultEfforts() {
  const select = $("workflowGoalDefaultReasoningEffort");
  if (!select) return;
  const status = codexStatusForModelDefaults();
  const saved = String(state.modelDefaults?.model_defaults?.workflow_goal_reasoning_effort || "").trim();
  const current = select.value || saved || "";
  const defaultEffort = String(status.default_reasoning_effort || "medium").trim();
  const options = workflowGoalEffortOptions(status);
  select.innerHTML = [
    `<option value="">기본값 (${escapeHTML(defaultEffort)})</option>`,
    ...options.map((effort) => `<option value="${escapeAttr(effort)}">${escapeHTML(effort.replace(/^./, (letter) => letter.toUpperCase()))}</option>`)
  ].join("");
  select.value = options.includes(current) ? current : "";
}

function renderModelDefaultsSettings() {
  const status = codexStatusForModelDefaults();
  const defaults = state.modelDefaults?.model_defaults || {};
  const effective = state.modelDefaults?.effective || {};
  const modelSelect = $("workflowGoalDefaultModel");
  if (!modelSelect) return;
  const catalog = Array.isArray(status.models) ? status.models : [];
  modelSelect.innerHTML = [
    `<option value="">기본값 (${escapeHTML(modelDefaultLabel(status))})</option>`,
    ...catalog.map((model) => {
      const name = String(model.name || "").trim();
      const label = String(model.label || name).trim();
      return name ? `<option value="${escapeAttr(name)}">${escapeHTML(label)} (${escapeHTML(name)})</option>` : "";
    }).filter(Boolean)
  ].join("");
  const savedModel = String(defaults.workflow_goal_model || "").trim();
  if (savedModel && !catalog.some((model) => String(model.name || "").trim() === savedModel)) {
    modelSelect.insertAdjacentHTML("beforeend", `<option value="${escapeAttr(savedModel)}">저장된 모델: ${escapeHTML(savedModel)}</option>`);
  }
  modelSelect.value = savedModel;
  renderModelDefaultEfforts();
  $("workflowGoalDefaultReasoningEffort").value = String(defaults.workflow_goal_reasoning_effort || "").trim();

  const effectiveModel = String(effective.workflow_goal_model || "").trim() || modelDefaultLabel(status);
  const effectiveEffort = String(effective.workflow_goal_reasoning_effort || "").trim() || String(status.default_reasoning_effort || "medium").trim();
  const source = state.modelDefaults?.display?.workflow_goal_source || "provider_default";
  const sourceLabel = source === "settings" ? "저장값" : (source === "server_config" ? "서버 설정" : "provider 기본값");
  const statusText = `자율진행 조향 모델: ${sourceLabel} · ${effectiveModel} / ${effectiveEffort}`;
  $("modelDefaultsStatus").textContent = statusText;
  $("agentSessionDefaultModel").textContent = `${modelDefaultLabel(status)} / ${status.default_reasoning_effort || "medium"}`;
  $("reportDefaultModel").textContent = "생성 요청별 선택";
  for (const id of ["workflowGoalDefaultModel", "workflowGoalDefaultReasoningEffort", "modelDefaultsRefresh", "modelDefaultsSave"]) {
    const el = $(id);
    if (el) el.disabled = state.modelDefaultsBusy;
  }
}

async function loadModelDefaults() {
  if (state.modelDefaultsBusy) return;
  state.modelDefaultsBusy = true;
  renderModelDefaultsSettings();
  try {
    const response = await api("/api/settings/model-defaults");
    state.modelDefaults = response;
    state.modelDefaultAgentExecutors = response.agent_executors || [];
  } catch (err) {
    showError(err);
  } finally {
    state.modelDefaultsBusy = false;
    renderModelDefaultsSettings();
  }
}

async function saveModelDefaults(event) {
  event.preventDefault();
  if (state.modelDefaultsBusy) return;
  const payload = {
    workflow_goal_model: workflowGoalSelectedModel(),
    workflow_goal_reasoning_effort: ($("workflowGoalDefaultReasoningEffort")?.value || "").trim()
  };
  state.modelDefaultsBusy = true;
  renderModelDefaultsSettings();
  try {
    const response = await api("/api/settings/model-defaults", {
      method: "PATCH",
      body: payload
    });
    state.modelDefaults = response;
    state.modelDefaultAgentExecutors = response.agent_executors || [];
  } catch (err) {
    showError(err);
  } finally {
    state.modelDefaultsBusy = false;
    renderModelDefaultsSettings();
  }
}
