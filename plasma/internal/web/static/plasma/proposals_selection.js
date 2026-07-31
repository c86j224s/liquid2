(function (Plasma) {
  "use strict";

  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const proposals = Plasma.proposals;

  function pruneSelectedProposals(pending) {
    const valid = new Set(pending.map((proposal) => proposal.proposal_id));
    for (const id of [...state.selectedProposals]) {
      if (!valid.has(id)) state.selectedProposals.delete(id);
    }
  }

  function updateProposalBulkBar() {
    const bar = $("proposalBulk");
    if (!bar) return;
    const n = state.selectedProposals.size;
    const countEl = $("proposalBulkCount");
    if (countEl) countEl.textContent = String(n);
    bar.classList.toggle("hidden", n === 0);
  }

  function toggleProposalSelection(checkbox) {
    const id = checkbox?.dataset?.selectProposalId;
    if (!id) return;
    if (checkbox.checked) state.selectedProposals.add(id);
    else state.selectedProposals.delete(id);
    if (checkbox.parentElement) checkbox.parentElement.classList.toggle("selected", checkbox.checked);
    updateProposalBulkBar();
  }

  function selectAllProposals() {
    $("proposalList").querySelectorAll("input.item-select[data-select-proposal-id]").forEach((cb) => {
      cb.checked = true;
      state.selectedProposals.add(cb.dataset.selectProposalId);
      if (cb.parentElement) cb.parentElement.classList.add("selected");
    });
    updateProposalBulkBar();
  }

  function clearProposalSelection() {
    state.selectedProposals.clear();
    $("proposalList").querySelectorAll("input.item-select:checked").forEach((cb) => {
      cb.checked = false;
      if (cb.parentElement) cb.parentElement.classList.remove("selected");
    });
    updateProposalBulkBar();
  }

  Object.assign(proposals, {
    clearProposalSelection,
    pruneSelectedProposals,
    selectAllProposals,
    toggleProposalSelection,
    updateProposalBulkBar
  });
})(window.Plasma);
