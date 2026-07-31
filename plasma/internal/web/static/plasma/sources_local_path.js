(function (Plasma) {
  "use strict";

  const sources = Plasma.sources;
  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const escapeHTML = Plasma.dom.escapeHTML;
  const escapeAttr = Plasma.dom.escapeAttr;
  const empty = Plasma.ui.empty;
  const api = Plasma.transport.api;
  const missionApi = Plasma.transport.missionApi;
  const captureMissionSelection = Plasma.mission.captureMissionSelection;
  const ownsMissionSelection = Plasma.mission.ownsMissionSelection;
  const isStaleMissionOperation = Plasma.mission.isStaleMissionOperation;
  const requireMission = () => sources.dependency("requireMission")();
  const reloadMission = (...args) => sources.dependency("reloadMission")(...args);
  const showError = (...args) => sources.dependency("showError")(...args);

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

  function updateLocalPathAttachState() {
    const btn = $("localPathAttachButton");
    if (!btn) return;
    const ready = Boolean(state.missionId) && state.localPathRoots.length &&
    Boolean(($("localPathRelativePath").value || "").trim() || state.localPathSelectedFile);
    btn.disabled = !ready;
  }

  async function browseLocalPathTree() {
    if (!requireMission()) return;
    const owner = captureMissionSelection();
    try {
      const result = await missionApi(owner, "/sources/local_path/tree", {
        method: "POST",
        body: {
          root_id: $("localPathRoot").value,
          relative_path: $("localPathRelativePath").value,
          depth: 1,
          limit: 200
        }
      });
      if (ownsMissionSelection(owner)) renderLocalPathTree(result.tree || result.Tree);
    } catch (err) {
      if (!isStaleMissionOperation(err) && ownsMissionSelection(owner)) showError(err);
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

  async function attachLocalPathSource(event) {
    event.preventDefault();
    if (!requireMission()) return;
    const owner = captureMissionSelection();
    try {
      await missionApi(owner, "/sources/local_path", {
        method: "POST",
        body: {
          root_id: $("localPathRoot").value,
          relative_path: $("localPathRelativePath").value,
          title: $("localPathTitle").value,
          restore: $("localPathRestore").checked
        }
      });
      if (!ownsMissionSelection(owner)) return;
      $("localPathTitle").value = "";
      $("localPathRestore").checked = false;
      state.localPathSelectedFile = "";
      updateLocalPathAttachState();
      await reloadMission(owner.missionId);
    } catch (err) {
      if (!isStaleMissionOperation(err) && ownsMissionSelection(owner)) showError(err);
    }
  }

  Object.assign(sources, {
    loadLocalPathRoots,
    renderLocalPathControls,
    updateLocalPathAttachState,
    browseLocalPathTree,
    renderLocalPathBreadcrumb,
    renderLocalPathTree,
    attachLocalPathSource
  });
})(window.Plasma);
