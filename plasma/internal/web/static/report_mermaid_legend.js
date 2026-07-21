(function (global) {
  "use strict";

  const maxLegendItems = 24;
  const maxLegendLabelLength = 80;

  function plasmaMermaidLineLegendLabels(source) {
    const text = String(source || "");
    if (!/^\s*xychart-beta\b/im.test(text)) return [];
    const labels = [];
    const seen = new Set();
    text.split(/\r?\n/).forEach((line) => {
      const label = mermaidLineLabel(line);
      if (!label || seen.has(label)) return;
      seen.add(label);
      labels.push(label);
    });
    return labels.slice(0, maxLegendItems);
  }

  function mermaidLineLabel(line) {
    const rest = String(line || "").trim().replace(/^line\b/i, "").trim();
    if (!rest || rest === String(line || "").trim()) return "";
    let label = "";
    if (rest[0] === '"' || rest[0] === "'") {
      label = readQuotedLabel(rest);
    } else {
      const valuesAt = rest.indexOf("[");
      if (valuesAt <= 0) return "";
      label = rest.slice(0, valuesAt).trim();
    }
    return normalizeLegendLabel(label);
  }

  function readQuotedLabel(text) {
    const quote = text[0];
    let value = "";
    for (let index = 1; index < text.length; index += 1) {
      const char = text[index];
      if (char === "\\" && index + 1 < text.length) {
        value += text[index + 1];
        index += 1;
        continue;
      }
      if (char === quote) return value;
      value += char;
    }
    return "";
  }

  function normalizeLegendLabel(label) {
    return String(label || "").replace(/\s+/g, " ").trim().slice(0, maxLegendLabelLength);
  }

  function applyPlasmaMermaidLineLegend(figure, source) {
    if (!figure || !global.document) return;
    figure.querySelector(".plasma-mermaid-line-legend")?.remove();
    const labels = plasmaMermaidLineLegendLabels(source);
    if (!labels.length) return;
    const diagram = figure.querySelector(".plasma-mermaid-diagram");
    const svg = figure.querySelector(".plasma-mermaid-diagram svg");
    if (!diagram) return;
    const series = bindLineSeries(svg, labels.length);
    const legend = global.document.createElement("div");
    legend.className = "plasma-mermaid-line-legend";
    legend.setAttribute("role", "group");
    legend.setAttribute("aria-label", "Mermaid 라인 차트 범례");
    labels.forEach((label, index) => {
      const item = global.document.createElement("button");
      item.type = "button";
      item.className = "plasma-mermaid-line-legend-item";
      item.setAttribute("aria-label", `${label} 라인 강조`);
      item.setAttribute("aria-pressed", "false");
      item.style.setProperty("--plasma-mermaid-series-index", String(index + 1));
      const marker = global.document.createElement("span");
      marker.className = "plasma-mermaid-line-legend-marker";
      marker.setAttribute("aria-hidden", "true");
      const color = series[index]?.color || "";
      if (color) marker.style.backgroundColor = color;
      const text = global.document.createElement("span");
      text.className = "plasma-mermaid-line-legend-text";
      text.textContent = label;
      item.append(marker, text);
      bindHighlight(item, figure, index, series);
      legend.appendChild(item);
    });
    diagram.appendChild(legend);
  }

  function bindLineSeries(svg, count) {
    if (!svg) return [];
    const series = [];
    for (let index = 0; index < count; index += 1) {
      const groups = Array.from(svg.querySelectorAll(`.line-plot-${index}`));
      groups.forEach((group) => {
        group.classList.add("plasma-mermaid-line-series");
        group.dataset.plasmaMermaidSeriesIndex = String(index);
        if (!group.getAttribute("tabindex")) group.setAttribute("tabindex", "0");
        bindHighlight(group, svg.closest(".plasma-mermaid-card") || svg, index, series);
      });
      series.push({ groups, color: lineSeriesColor(groups) });
    }
    return series;
  }

  function lineSeriesColor(groups) {
    for (const group of groups) {
      const shape = group.querySelector("path, polyline, line") || group;
      const color = colorValue(shape, "stroke") || colorValue(group, "stroke") || colorValue(shape, "fill");
      if (color) return color;
    }
    return "";
  }

  function colorValue(node, property) {
    if (!node) return "";
    const value = node.getAttribute(property) || node.style?.[property] || global.getComputedStyle?.(node)?.[property] || "";
    if (!value || value === "none" || value === "transparent" || value === "rgba(0, 0, 0, 0)") return "";
    return value;
  }

  function bindHighlight(target, figure, index, series) {
    if (!target || !figure) return;
    const activate = () => setActiveSeries(figure, series, index);
    const clear = () => clearActiveSeries(figure, series);
    target.addEventListener("mouseenter", activate);
    target.addEventListener("focus", activate);
    target.addEventListener("pointerdown", activate);
    target.addEventListener("mouseleave", clear);
    target.addEventListener("blur", clear);
  }

  function setActiveSeries(figure, series, activeIndex) {
    figure.dataset.plasmaMermaidActiveSeries = String(activeIndex);
    series.forEach((entry, index) => {
      entry.groups.forEach((group) => {
        group.classList.toggle("plasma-mermaid-line-series--active", index === activeIndex);
        group.classList.toggle("plasma-mermaid-line-series--dimmed", index !== activeIndex);
      });
    });
    figure.querySelectorAll(".plasma-mermaid-line-legend-item").forEach((item, index) => {
      item.classList.toggle("plasma-mermaid-line-legend-item--active", index === activeIndex);
      item.setAttribute("aria-pressed", index === activeIndex ? "true" : "false");
    });
  }

  function clearActiveSeries(figure, series) {
    delete figure.dataset.plasmaMermaidActiveSeries;
    series.forEach((entry) => {
      entry.groups.forEach((group) => {
        group.classList.remove("plasma-mermaid-line-series--active", "plasma-mermaid-line-series--dimmed");
      });
    });
    figure.querySelectorAll(".plasma-mermaid-line-legend-item").forEach((item) => {
      item.classList.remove("plasma-mermaid-line-legend-item--active");
      item.setAttribute("aria-pressed", "false");
    });
  }

  global.plasmaMermaidLineLegendLabels = plasmaMermaidLineLegendLabels;
  global.applyPlasmaMermaidLineLegend = applyPlasmaMermaidLineLegend;
})(window);
