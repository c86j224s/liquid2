(function reportsTOC(root) {
  "use strict";

  const reports = root.Plasma.reports;

  // The controller derives navigation from the rendered report. It owns no
  // report state and must be prepared explicitly by report preview callers.
  function createReportTOCController({ body, panel, list, toggleButton }) {
    let enabled = false;
    let headings = [];
    const generatedFocusTargets = new Set();

    toggleButton.addEventListener("click", () => setExpanded(panel.hidden));
    list.addEventListener("click", (event) => {
      const button = event.target.closest?.("[data-report-toc-index]");
      if (!button || !list.contains(button)) return;
      navigateTo(Number(button.dataset.reportTocIndex));
    });

    function prepare(nextEnabled) {
      reset();
      enabled = Boolean(nextEnabled);
    }

    function refresh() {
      clearEntries();
      setExpanded(false);
      if (!enabled) return;

      const reportBody = body.querySelector(".report-modal-body.turn-markdown");
      if (!reportBody) return;
      headings = Array.from(reportBody.querySelectorAll("h1, h2, h3, h4"))
        .filter((heading) => normalizedHeadingText(heading));
      toggleButton.hidden = headings.length === 0;
      if (!headings.length) return;

      const document = list.ownerDocument || root.document;
      headings.forEach((heading, index) => {
        const item = document.createElement("li");
        const button = document.createElement("button");
        item.className = "report-toc-item";
        item.dataset.level = heading.tagName.slice(1);
        button.type = "button";
        button.className = "report-toc-link";
        button.dataset.reportTocIndex = String(index);
        button.textContent = normalizedHeadingText(heading);
        item.append(button);
        list.append(item);
      });
    }

    function navigateTo(index) {
      const heading = headings[index];
      if (!heading) return;
      const bodyTop = body.getBoundingClientRect().top;
      const headingTop = heading.getBoundingClientRect().top;
      body.scrollTo({
        top: Math.max(0, body.scrollTop + headingTop - bodyTop - 12),
        behavior: "smooth"
      });
      if (!heading.hasAttribute("tabindex")) {
        heading.setAttribute("tabindex", "-1");
        generatedFocusTargets.add(heading);
      }
      heading.focus({ preventScroll: true });
      setExpanded(false);
    }

    function setExpanded(expanded) {
      const next = Boolean(expanded && !toggleButton.hidden);
      panel.hidden = !next;
      toggleButton.setAttribute("aria-expanded", String(next));
      toggleButton.title = next ? "목차 접기" : "목차 펼치기";
      toggleButton.setAttribute("aria-label", toggleButton.title);
    }

    function clearEntries() {
      generatedFocusTargets.forEach((heading) => heading.removeAttribute("tabindex"));
      generatedFocusTargets.clear();
      headings = [];
      list.replaceChildren();
      toggleButton.hidden = true;
    }

    function reset() {
      enabled = false;
      clearEntries();
      setExpanded(false);
    }

    return { prepare, refresh, reset };
  }

  function normalizedHeadingText(heading) {
    return String(heading?.textContent || "").trim().replace(/\s+/g, " ");
  }

  function initReportTOCController(elements) {
    reports.tocController = createReportTOCController(elements);
    reports.tocController.reset();
    return reports.tocController;
  }

  Object.assign(reports, { createReportTOCController, initReportTOCController });
})(window);
