(function (root) {
  function configuredStatus(statuses, executor) {
    return (statuses || []).find((status) => status.name === executor && status.configured) ||
      (statuses || []).find((status) => status.name === executor) || null;
  }

  function payload(model, effort) {
    return {
      agent_model: String(model || "").trim(),
      agent_reasoning_effort: String(effort || "").trim()
    };
  }

  function render(statuses, executor) {
    const modelSelect = document.getElementById("reportAgentModel");
    const effortSelect = document.getElementById("reportAgentReasoningEffort");
    if (!modelSelect || !effortSelect) return;
    const status = configuredStatus(statuses, executor);
    const previous = modelSelect.value;
    modelSelect.innerHTML = '<option value="">미션 설정 상속</option>';
    (status?.models || []).forEach((model) => modelSelect.add(new Option(model.label || model.name, model.name)));
    modelSelect.value = Array.from(modelSelect.options).some((option) => option.value === previous) ? previous : "";
    refreshEfforts(status);
  }

  function refreshEfforts(status) {
    const modelSelect = document.getElementById("reportAgentModel");
    const effortSelect = document.getElementById("reportAgentReasoningEffort");
    if (!modelSelect || !effortSelect) return;
    const previous = effortSelect.value;
    const model = (status?.models || []).find((candidate) => candidate.name === modelSelect.value);
    effortSelect.innerHTML = `<option value="">${model ? "모델 기본값" : "미션 설정 상속"}</option>`;
    (model?.reasoning_efforts || []).forEach((effort) => effortSelect.add(new Option(effort, effort)));
    effortSelect.value = Array.from(effortSelect.options).some((option) => option.value === previous) ? previous : "";
  }

  root.ReportModelSelection = { configuredStatus, payload, render, refreshEfforts };
})(typeof window === "undefined" ? globalThis : window);
