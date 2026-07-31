(function (Plasma) {
  "use strict";

  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const mission = Plasma.mission;
  const transport = Plasma.transport;
  const ui = Plasma.ui;
  const workflow = Plasma.workflow = Plasma.workflow || {};

  function workflowRawInputValue() {
    return $("workflowInstruction").value.trim() || workflow._callbacks.rawInputFallback();
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
    if (!workflow._callbacks.requireMission()) return;
    if (state.workflowGoalDraftPending || state.workflowPending) return;
    const userInstructionRaw = workflowRawInputValue();
    if (!userInstructionRaw) {
      workflow._callbacks.showError(new Error("자율진행 요청 원문을 입력해야 합니다."));
      return;
    }
    const owner = mission.captureMissionSelection();
    state.workflowGoalDraftPending = true;
    setWorkflowBusy(false);
    workflow._callbacks.syncReportControls();
    const button = $("draftWorkflowGoalButton");
    button.textContent = "초안 생성 중";
    try {
      const response = await transport.missionApi(owner, "/workflows/goal_draft", {
        method: "POST",
        body: {
          user_instruction_raw: userInstructionRaw,
          agent_executor: $("agentExecutor").value
        }
      });
      if (!mission.ownsMissionSelection(owner)) return;
      const draft = response.workflow_goal_draft || {};
      const currentRaw = workflowRawInputValue();
      if (currentRaw !== userInstructionRaw) return;
      $("workflowInstruction").value = draft.user_instruction_raw || userInstructionRaw;
      $("workflowRunGoal").value = draft.run_goal || "";
      $("workflowStepInstruction").value = draft.step_instruction || draft.run_goal || "";
      state.workflowGoalDraftRaw = draft.user_instruction_raw || userInstructionRaw;
    } catch (err) {
      if (mission.ownsMissionSelection(owner)) workflow._callbacks.showError(err);
    } finally {
      if (!mission.ownsMissionSelection(owner)) return;
      state.workflowGoalDraftPending = false;
      button.textContent = "목표 초안 생성";
      workflow._callbacks.setFormsEnabled(true);
      workflow._callbacks.syncReportControls();
    }
  }

  function setWorkflowBusy(busy) {
    state.workflowPending = busy;
    const draftBusy = state.workflowGoalDraftPending;
    const layered = workflowStepInstructionMode() === "layered";
    const blocked = workflow._callbacks.workflowControlsBlocked();
    ui.setElementDisabled("workflowInstruction", busy || draftBusy || blocked || !state.detail);
    ui.setElementDisabled("workflowStepInstructionMode", busy || draftBusy || blocked || !state.detail);
    ui.setElementDisabled("draftWorkflowGoalButton", !layered || busy || draftBusy || blocked || !state.detail);
    ui.setElementDisabled("workflowRunGoal", !layered || busy || draftBusy || blocked || !state.detail);
    ui.setElementDisabled("workflowStepInstruction", !layered || busy || draftBusy || blocked || !state.detail);
    ui.setElementDisabled("startWorkflowButton", busy || draftBusy || blocked || !state.detail);
    ui.setElementDisabled("stopWorkflowButton", !busy || !state.detail);
    ui.setButtonText("startWorkflowButton", busy ? "진행 중" : "시작");
    ui.setButtonText("draftWorkflowGoalButton", draftBusy ? "초안 생성 중" : "목표 초안 생성");
  }

  Object.assign(workflow, {
    workflowRawInputValue,
    onWorkflowRawInput,
    workflowStepInstructionMode,
    updateWorkflowStepInstructionMode,
    draftWorkflowGoal,
    setWorkflowBusy
  });
})(window.Plasma);
