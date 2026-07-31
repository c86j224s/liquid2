(function (Plasma) {
  "use strict";

  const sources = Plasma.sources;
  const $ = Plasma.dom.$;
  const escapeHTML = Plasma.dom.escapeHTML;
  const escapeAttr = Plasma.dom.escapeAttr;
  const empty = Plasma.ui.empty;
  const missionApi = Plasma.transport.missionApi;
  const captureMissionSelection = Plasma.mission.captureMissionSelection;
  const ownsMissionSelection = Plasma.mission.ownsMissionSelection;
  const isStaleMissionOperation = Plasma.mission.isStaleMissionOperation;
  const requireMission = () => sources.dependency("requireMission")();
  const reloadMission = (...args) => sources.dependency("reloadMission")(...args);
  const showError = (...args) => sources.dependency("showError")(...args);
  const onDetailButtonClick = (...args) => sources.dependency("onDetailButtonClick")(...args);

  async function searchLiquid2(event) {
    event.preventDefault();
    if (!requireMission()) return;
    const owner = captureMissionSelection();
    try {
      const result = await missionApi(owner, "/sources/liquid2/search", {
        method: "POST",
        body: { query: $("liquid2Query").value, limit: 8 }
      });
      if (ownsMissionSelection(owner)) renderLiquid2Results(result.Candidates || result.candidates || []);
    } catch (err) {
      if (!isStaleMissionOperation(err) && ownsMissionSelection(owner)) renderLiquid2Error(err.message);
    }
  }

  async function attachLiquid2(externalSourceID) {
    if (!requireMission()) return;
    const owner = captureMissionSelection();
    try {
      await missionApi(owner, "/sources/liquid2/snapshot", {
        method: "POST",
        body: {
          external_source_id: externalSourceID,
          reason: "Plasma 작업공간에서 선택함"
        }
      });
      if (!ownsMissionSelection(owner)) return;
      $("liquid2Results").innerHTML = "";
      await reloadMission(owner.missionId);
    } catch (err) {
      if (!isStaleMissionOperation(err) && ownsMissionSelection(owner)) showError(err);
    }
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

  function onLiquid2ResultsClick(event) {
    if (onDetailButtonClick(event)) return;
    const button = event.target.closest("[data-liquid2-source-id]");
    if (button) attachLiquid2(button.dataset.liquid2SourceId);
  }

  Object.assign(sources, {
    searchLiquid2,
    attachLiquid2,
    renderLiquid2Results,
    renderLiquid2Error,
    onLiquid2ResultsClick
  });
})(window.Plasma);
