(function (Plasma) {
  "use strict";

  const sources = Plasma.sources;
  const state = Plasma.state;
  const $ = Plasma.dom.$;
  const updateLocalPathAttachState = (...args) => sources.updateLocalPathAttachState(...args);
  const browseLocalPathTree = (...args) => sources.browseLocalPathTree(...args);

  function localPathParent(p) {
    const clean = String(p || ".").replace(/\/+$/, "");
    if (!clean || clean === ".") return ".";
    const idx = clean.lastIndexOf("/");
    return idx <= 0 ? "." : clean.slice(0, idx);
  }

  function onLocalPathBreadcrumbClick(event) {
    const crumb = event.target.closest("[data-local-path-crumb]");
    if (!crumb) return;
    const path = crumb.dataset.localPathCrumb || ".";
    $("localPathRelativePath").value = path === "." ? "" : path;
    state.localPathSelectedFile = "";
    updateLocalPathAttachState();
    browseLocalPathTree();
  }

  function onLocalPathTreeClick(event) {
    const pick = event.target.closest("[data-local-path-pick]");
    if (!pick || pick.disabled) return;
    const kind = pick.dataset.localPathKind;
    if (kind === "up") {
      const parent = localPathParent(state.localPathCurrentDir || ".");
      $("localPathRelativePath").value = parent === "." ? "" : parent;
      state.localPathSelectedFile = "";
      updateLocalPathAttachState();
      browseLocalPathTree();
      return;
    }
    const rel = pick.dataset.localPathPick || "";
    if (kind === "dir") {
      $("localPathRelativePath").value = rel;
      state.localPathSelectedFile = "";
      updateLocalPathAttachState();
      browseLocalPathTree();
      return;
    }
    // File: select it, auto-fill the title, highlight without refetching.
    state.localPathSelectedFile = rel;
    $("localPathRelativePath").value = rel;
    if (!$("localPathTitle").value.trim()) {
      $("localPathTitle").value = rel.split("/").pop() || rel;
    }
    for (const el of $("localPathTree").querySelectorAll(".local-path-entry")) {
      el.classList.toggle("selected", el.dataset.localPathKind === "file" && el.dataset.localPathPick === rel);
    }
    updateLocalPathAttachState();
  }

  Object.assign(sources, {
    localPathParent,
    onLocalPathBreadcrumbClick,
    onLocalPathTreeClick
  });
})(window.Plasma);
