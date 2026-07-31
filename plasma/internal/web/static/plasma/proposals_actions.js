(function (Plasma) {
  "use strict";

  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const mission = Plasma.mission;
  const missionApi = Plasma.transport.missionApi;
  const proposals = Plasma.proposals;

  async function proposeEvidence(event) {
    event.preventDefault();
    if (!proposals.call("requireMission")) return;
    const summary = $("candidateSummary").value.trim();
    if (!summary) return;
    const sourceValue = $("candidateSource").value;
    if (!sourceValue) {
      proposals.call("showError", new Error("근거 후보를 제안하려면 먼저 소스를 추가하고 선택해야 합니다."));
      return;
    }
    const [snapshotID, artifactID] = sourceValue.split("|");
    const owner = mission.captureMissionSelection();
    try {
      await missionApi(owner, "/candidates/evidence", {
        method: "POST",
        body: { summary, evidence_type: $("candidateEvidenceType").value || "observation", snapshot_id: snapshotID, artifact_id: artifactID }
      });
      if (!mission.ownsMissionSelection(owner)) return;
      $("candidateSummary").value = "";
      await mission.reloadMission(owner.missionId);
    } catch (err) {
      if (!mission.isStaleMissionOperation(err) && mission.ownsMissionSelection(owner)) proposals.call("showError", err);
    }
  }

  async function decideProposal(proposalID, action) {
    if (!proposals.call("requireMission")) return;
    const owner = mission.captureMissionSelection();
    try {
      await missionApi(owner, `/proposals/${proposalID}/${action}`, { method: "POST", body: {} });
      await mission.reloadMission(owner.missionId);
    } catch (err) {
      if (!mission.isStaleMissionOperation(err) && mission.ownsMissionSelection(owner)) proposals.call("showError", err);
    }
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
      return `<option value="${Plasma.dom.escapeAttr(snapshotID + "|" + artifactID)}">${Plasma.dom.escapeHTML(title)} / ${Plasma.dom.escapeHTML(snapshotID)}</option>`;
    }).join("");
  }

  function onProposalListClick(event) {
    if (Plasma.ui.onDetailButtonClick(event)) return;
    const button = event.target.closest("[data-proposal-id][data-action]");
    if (button) decideProposal(button.dataset.proposalId, button.dataset.action);
  }

  async function bulkProposalAction(action) {
    if (!proposals.call("requireMission")) return;
    const owner = mission.captureMissionSelection();
    const ids = [...state.selectedProposals];
    if (ids.length === 0) return;
    const errors = await proposals.call("runBulkSequential", ids, (id) => {
      if (!mission.ownsMissionSelection(owner)) throw new mission.StaleMissionOperationError();
      return missionApi(owner, `/proposals/${id}/${action}`, { method: "POST", body: {} });
    });
    if (!mission.ownsMissionSelection(owner)) return;
    state.selectedProposals.clear();
    await mission.reloadMission(owner.missionId);
    if (errors.length > 0) {
      const sample = errors.slice(0, 3).map((e) => e?.message || String(e)).join("; ");
      proposals.call("showError", new Error(`검토 후보 ${ids.length}개 중 ${errors.length}개 처리 실패: ${sample}`));
    }
  }

  Object.assign(proposals, { bulkProposalAction, decideProposal, onProposalListClick, proposeEvidence, renderCandidateSourceOptions });
})(window.Plasma);
