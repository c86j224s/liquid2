(function (Plasma) {
  "use strict";

  const state = Plasma.state;
  const deps = {};

  function configureTabs(values = {}) {
    Object.assign(deps, values);
  }

  function renderTabs() {
    for (const tab of document.querySelectorAll("[data-tab]")) {
      tab.classList.toggle("active", tab.dataset.tab === state.activeTab);
    }
    for (const panel of document.querySelectorAll("[data-tab-panel]")) {
      panel.classList.toggle("active", panel.dataset.tabPanel === state.activeTab);
    }
  }

  function onTabBarClick(event) {
    const tab = event.target.closest("[data-tab]");
    if (!tab) return;
    state.activeTab = tab.dataset.tab;
    renderTabs();
    if (state.activeTab === "settings") {
      deps.loadMissionUsage?.();
      deps.loadModelDefaults?.();
      deps.loadConfluenceConnections?.();
    }
  }

  Object.assign(Plasma.ui, { configureTabs, onTabBarClick, renderTabs });
})(window.Plasma);
