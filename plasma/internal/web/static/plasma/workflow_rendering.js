(function (Plasma) {
  "use strict";

  const { $, escapeHTML, escapeAttr, shortID } = Plasma.dom;
  const workflow = Plasma.workflow = Plasma.workflow || {};

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
    workflow.setWorkflowBusy(Boolean(active));
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
    const continueDisabled = workflow._callbacks.workflowContinueBlocked() ? "disabled" : "";
    const continueAction = workflow.workflowCanContinue(run)
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

  Object.assign(workflow, {
    renderWorkflowControls,
    renderWorkflowRun,
    workflowStepDotClass,
    currentWorkflowRun,
    workflowStatusLabel,
    workflowStatusClass,
    workflowDecisionLabel
  });
})(window.Plasma);
