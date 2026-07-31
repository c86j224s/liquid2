(function bootstrapExtrasModule(root) {
  "use strict";
  const Plasma = root.Plasma;
  const $ = Plasma.dom.$;
  const state = Plasma.state;
  const ui = Plasma.ui;

  function init(callbacks) {
  // ── Source-add methods behave like tabs (one panel at a time) ──
  (function initSourceTabs() {
    const group = document.querySelector(".source-add-group");
    if (!group) return;
    const tabs = Array.from(group.querySelectorAll(".source-tab"));
    const panels = tabs
      .map((tab) => document.getElementById(tab.dataset.sourceTab))
      .filter(Boolean);
    let syncing = false;
    function activate(id) {
      if (syncing) return;
      syncing = true;
      for (const panel of panels) panel.open = panel.id === id;
      for (const tab of tabs) {
        const on = tab.dataset.sourceTab === id;
        tab.classList.toggle("active", on);
        tab.setAttribute("aria-selected", on ? "true" : "false");
      }
      syncing = false;
    }
    for (const tab of tabs) {
      tab.addEventListener("click", () => activate(tab.dataset.sourceTab));
    }
    // Programmatic opens (e.g. Confluence flow sets details.open = true) sync tabs.
    for (const panel of panels) {
      panel.addEventListener("toggle", () => {
        if (panel.open) activate(panel.id);
      });
    }
    activate(panels[0] && panels[0].id);
  })();

  // ── Wave 6a: theme toggle ──────────────────────────────────
  (function initTheme() {
    const STORAGE_KEY = "plasma.theme";
    const root = document.documentElement;
    const btn = $("themeToggle");
    const saved = localStorage.getItem(STORAGE_KEY);
    const mql = window.matchMedia("(prefers-color-scheme: light)");

    function applyTheme(theme) {
      root.dataset.theme = theme;
      if (btn) btn.textContent = theme === "light" ? "🌙" : "☀";
    }

    if (saved === "dark" || saved === "light") {
      applyTheme(saved);
    } else {
      applyTheme(mql.matches ? "light" : "dark");
    }

    if (btn) {
      btn.addEventListener("click", () => {
        const next = root.dataset.theme === "light" ? "dark" : "light";
        localStorage.setItem(STORAGE_KEY, next);
        applyTheme(next);
      });
    }

    mql.addEventListener("change", (e) => {
      if (!localStorage.getItem(STORAGE_KEY)) {
        applyTheme(e.matches ? "light" : "dark");
      }
    });
  })();

  // ── Focus mode: fold the header chrome to maximise the conversation ──
  (function initFocusToggle() {
    const STORAGE_KEY = "plasma.chatFocus";
    const btn = $("focusToggle");
    const glyph = btn ? btn.querySelector("span") : null;
    const apply = (on) => {
      document.body.classList.toggle("chat-focus", on);
      if (btn) {
        const label = on ? "상단 정보 펼치기" : "상단 정보 접기";
        btn.classList.toggle("active", on);
        btn.setAttribute("aria-pressed", on ? "true" : "false");
        btn.setAttribute("aria-label", label);
        btn.setAttribute("title", label);
        if (glyph) glyph.textContent = on ? "⌄" : "⌃";
      }
    };
    apply(localStorage.getItem(STORAGE_KEY) === "1");
    if (btn) {
      btn.addEventListener("click", () => {
        const on = !document.body.classList.contains("chat-focus");
        localStorage.setItem(STORAGE_KEY, on ? "1" : "0");
        apply(on);
      });
    }
  })();

  // ── Wave 6b: ⌘/Ctrl+Enter to send ─────────────────────────
  $("turnText").addEventListener("keydown", (e) => {
    if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
      e.preventDefault();
      $("turnForm").dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    }
  });

  // ── Wave 6c: Escape to close modal / toast / mission picker ─
  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      if ($("detailModal") && !$("detailModal").classList.contains("hidden")) {
        ui.hideDetail();
      }
      if ($("errorToast") && !$("errorToast").classList.contains("hidden")) {
        ui.hideError();
      }
      if (document.body.classList.contains("mission-picker-open")) {
        document.body.classList.remove("mission-picker-open");
      }
    }
  });

  // ── Wave 6e: Multi-select bulk approve/reject ──────────────
  $("sourceCandidateList").addEventListener("change", (e) => {
    const cb = e.target.closest("input.item-select[data-select-source-url]");
    if (!cb) return;
    window.Plasma.sources.toggleSourceCandidateSelection(cb);
  });
  $("proposalList").addEventListener("change", (e) => {
    const cb = e.target.closest("input.item-select[data-select-proposal-id]");
    if (!cb) return;
	    Plasma.proposals.toggleProposalSelection(cb);
  });
  $("sourceCandidateSelectAll").addEventListener("click", window.Plasma.sources.selectAllSourceCandidates);
  $("sourceCandidateClearSelection").addEventListener("click", window.Plasma.sources.clearSourceCandidateSelection);
  $("sourceCandidateBulkApprove").addEventListener("click", () => window.Plasma.sources.bulkSourceCandidateAction("approve"));
  $("sourceCandidateBulkReject").addEventListener("click", () => window.Plasma.sources.bulkSourceCandidateAction("reject"));
	  $("proposalSelectAll").addEventListener("click", Plasma.proposals.selectAllProposals);
	  $("proposalClearSelection").addEventListener("click", Plasma.proposals.clearProposalSelection);
	  $("proposalBulkApprove").addEventListener("click", () => Plasma.proposals.bulkProposalAction("approve"));
	  $("proposalBulkReject").addEventListener("click", () => Plasma.proposals.bulkProposalAction("reject"));

  // ── Wave 6d: Mobile mission picker (bottom sheet) ──────────
  (function initMissionPicker() {
    const openBtn = $("missionPickerOpen");
    const closeBtn = $("missionPickerClose");
    const rail = document.querySelector(".rail");
    if (openBtn) {
      openBtn.addEventListener("click", (e) => {
        e.stopPropagation();
        document.body.classList.toggle("mission-picker-open");
      });
    }
    if (closeBtn) {
      closeBtn.addEventListener("click", () => {
        document.body.classList.remove("mission-picker-open");
      });
    }
    // auto-close on mission select
    $("missionList").addEventListener("click", (e) => {
      if (e.target.closest("button.item[data-mission-id]")) {
        document.body.classList.remove("mission-picker-open");
      }
    });
    // click outside the sheet (backdrop) closes
    document.addEventListener("click", (e) => {
      if (!document.body.classList.contains("mission-picker-open")) return;
      if (openBtn && openBtn.contains(e.target)) return;
      if (rail && rail.contains(e.target)) return;
      document.body.classList.remove("mission-picker-open");
    });
  })();


  }

  Plasma.bootstrapExtras = { init };
})(window);
