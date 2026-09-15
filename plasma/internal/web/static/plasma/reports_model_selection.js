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

  function selectionPairs() {
    return [
      ["articleAgentModel", "articleAgentReasoningEffort"],
      ["reportAgentModel", "reportAgentReasoningEffort"]
    ];
  }

  function render(statuses, executor) {
    const status = configuredStatus(statuses, executor);
    for (const [modelID, effortID] of selectionPairs()) {
      const modelSelect = document.getElementById(modelID);
      if (!modelSelect || !document.getElementById(effortID)) continue;
      const previous = modelSelect.value;
      modelSelect.innerHTML = '<option value="">미션 설정 상속</option>';
      (status?.models || []).forEach((model) => modelSelect.add(new Option(model.label || model.name, model.name)));
      modelSelect.value = Array.from(modelSelect.options).some((option) => option.value === previous) ? previous : "";
      refreshEfforts(status, modelID, effortID);
    }
  }

  function refreshEfforts(status, modelID = "reportAgentModel", effortID = "reportAgentReasoningEffort") {
    const modelSelect = document.getElementById(modelID);
    const effortSelect = document.getElementById(effortID);
    if (!modelSelect || !effortSelect) return;
    const previous = effortSelect.value;
    const model = (status?.models || []).find((candidate) => candidate.name === modelSelect.value);
    effortSelect.innerHTML = `<option value="">${model ? "모델 기본값" : "미션 설정 상속"}</option>`;
    (model?.reasoning_efforts || []).forEach((effort) => effortSelect.add(new Option(effort, effort)));
    effortSelect.value = Array.from(effortSelect.options).some((option) => option.value === previous) ? previous : "";
  }

  root.Plasma.reports.modelSelection = { configuredStatus, payload, render, refreshEfforts };
})(typeof window === "undefined" ? globalThis : window);
