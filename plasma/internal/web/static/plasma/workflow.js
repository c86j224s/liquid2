(function (Plasma) {
  "use strict";

  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const mission = Plasma.mission;
  const transport = Plasma.transport;
  const workflow = Plasma.workflow = Plasma.workflow || {};
  const callbacks = workflow._callbacks || {
    requireMission: () => Boolean(state.missionId),
    reloadMission: (missionId) => mission.refreshSelectedMissionDetail({ ...mission.captureMissionSelection(), missionId: missionId || state.missionId }),
    showError: (err) => console.error(err),
    syncReportControls: () => {},
    workflowControlsBlocked: () => false,
    workflowContinueBlocked: () => false,
    setFormsEnabled: () => {}
  };

  const WORKFLOW_DEFAULT_MAX_STEPS = 20;
  const WORKFLOW_DEFAULT_MAX_DURATION_MS = 0;

  function configure(options = {}) {
    Object.assign(callbacks, options);
    workflow._callbacks = callbacks;
  }

  async function startWorkflow() {
    if (!callbacks.requireMission()) return;
    if (state.workflowPending || state.workflowGoalDraftPending) return;
    const owner = mission.captureMissionSelection();
    const userInstructionRaw = workflow.workflowRawInputValue();
    const runGoal = $("workflowRunGoal").value.trim() || userInstructionRaw;
    const instruction = $("workflowStepInstruction").value.trim() || runGoal || userInstructionRaw;
    if (!instruction) {
      callbacks.showError(new Error("워크플로우 지시문을 입력해야 합니다."));
      return;
    }
    if (state.workflowGoalDraftRaw && state.workflowGoalDraftRaw !== userInstructionRaw && ($("workflowRunGoal").value.trim() || $("workflowStepInstruction").value.trim())) {
      callbacks.showError(new Error("요청 원문이 목표 초안 생성 이후 변경되었습니다. 목표 초안을 다시 생성하거나 목표/첫 스텝을 비워 직접 시작하세요."));
      return;
    }
    state.workflowPending = true;
    workflow.setWorkflowBusy(true);
    callbacks.syncReportControls();
    try {
      await transport.missionApi(owner, "/workflows", { method: "POST", body: workflowStartBody(userInstructionRaw, runGoal, instruction) });
      if (!mission.ownsMissionSelection(owner)) return;
      $("workflowInstruction").value = "";
      $("workflowRunGoal").value = "";
      $("workflowStepInstruction").value = "";
      state.workflowGoalDraftRaw = "";
      await callbacks.reloadMission(owner.missionId);
    } catch (err) {
      if (!mission.ownsMissionSelection(owner)) return;
      state.workflowPending = false;
      workflow.setWorkflowBusy(false);
      callbacks.syncReportControls();
      callbacks.showError(err);
    }
  }

  function workflowStartBody(userInstructionRaw, runGoal, instruction) {
    return {
      step_instruction_mode: workflow.workflowStepInstructionMode(),
      instruction,
      agent_executor: $("agentExecutor").value,
      mcp_mode: $("mcpMode").value,
      max_steps: WORKFLOW_DEFAULT_MAX_STEPS,
      max_duration_ms: WORKFLOW_DEFAULT_MAX_DURATION_MS,
      stop_condition: "사용자 정지, 최대 단계, 에이전트 완료 선언 또는 오류",
      user_instruction_raw: userInstructionRaw,
      run_goal: runGoal
    };
  }

  async function stopWorkflow() {
    if (!callbacks.requireMission()) return;
    const owner = mission.captureMissionSelection();
    const run = workflow.currentWorkflowRun(state.detail?.workflow_runs || []);
    if (!run?.workflow_run_id) return;
    try {
      await transport.missionApi(owner, `/workflows/${encodeURIComponent(run.workflow_run_id)}/stop`, {
        method: "POST",
        body: {}
      });
      if (!mission.ownsMissionSelection(owner)) return;
      await callbacks.reloadMission(owner.missionId);
    } catch (err) {
      if (mission.ownsMissionSelection(owner)) callbacks.showError(err);
    }
  }

  async function continueWorkflowRun(workflowRunID) {
    if (!callbacks.requireMission()) return;
    if (callbacks.workflowContinueBlocked()) return;
    const owner = mission.captureMissionSelection();
    const run = findWorkflowRun(workflowRunID);
    if (!run) {
      callbacks.showError(new Error("이어갈 자율진행 기록을 찾을 수 없습니다."));
      return;
    }
    if (!workflowCanContinue(run)) {
      callbacks.showError(new Error("완료되었거나 진행 중인 자율진행은 이어갈 수 없습니다."));
      return;
    }
    const instruction = workflowContinuationInstruction(run);
    if (!instruction) {
      callbacks.showError(new Error("이어갈 다음 지시를 찾을 수 없습니다."));
      return;
    }
    state.workflowPending = true;
    workflow.setWorkflowBusy(true);
    callbacks.syncReportControls();
    try {
      await transport.missionApi(owner, "/workflows", { method: "POST", body: workflowContinuationBody(run, instruction) });
      if (!mission.ownsMissionSelection(owner)) return;
      await callbacks.reloadMission(owner.missionId);
    } catch (err) {
      if (!mission.ownsMissionSelection(owner)) return;
      state.workflowPending = false;
      workflow.setWorkflowBusy(false);
      callbacks.syncReportControls();
      callbacks.showError(err);
    }
  }

  function workflowContinuationBody(run, instruction) {
    return {
      step_instruction_mode: "layered",
      instruction,
      agent_executor: run.agent_executor || $("agentExecutor").value,
      mcp_mode: run.mcp_mode || $("mcpMode").value,
      max_steps: Number(run.max_steps ?? WORKFLOW_DEFAULT_MAX_STEPS),
      max_duration_ms: Number(run.max_duration_ms ?? WORKFLOW_DEFAULT_MAX_DURATION_MS),
      stop_condition: run.stop_condition || "사용자 정지, 최대 단계, 에이전트 완료 선언 또는 오류",
      continue_from_workflow_run_id: run.workflow_run_id || "",
      user_instruction_raw: run.user_instruction_raw || run.instruction || instruction,
      run_goal: run.run_goal || run.user_instruction_raw || instruction
    };
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

  Object.assign(workflow, {
    _callbacks: callbacks,
    configure,
    startWorkflow,
    stopWorkflow,
    continueWorkflowRun,
    onWorkflowRunListClick,
    findWorkflowRun,
    workflowCanContinue,
    workflowContinuationInstruction
  });
})(window.Plasma);
