(function reportsPipelineGraph(root) {
  "use strict";
  const reports = root.Plasma.reports;
  const core = reports.pipelineCore;
  function visualNode(node, x, y, width, fixed) {
    const label = core.escapeHTML(core.nodeLabel(node));
    const state = core.escapeHTML(node.state || "unknown");
    const timing = core.escapeHTML(core.nodeTiming(node));
    const fixedClass = fixed ? " pipeline-visual-node-plan" : "";
    return `<g class="pipeline-visual-node${fixedClass} state-${state}" data-pipeline-node-width="${width}" transform="translate(${x} ${y})">
      ${timing ? `<title${core.liveTimingAttributes(node, core.nodeLabel(node))}>${core.escapeHTML(`${core.nodeLabel(node)} ${timing}`)}</title>` : ""}
      <circle class="pipeline-visual-dot" r="5"></circle>
      <text class="pipeline-visual-label" y="27" text-anchor="middle">${label}</text>
      ${timing ? `<text class="pipeline-visual-time" y="44" text-anchor="middle"${core.liveTimingAttributes(node)}>${timing}</text>` : ""}
    </g>`;
  }
  function connector(x1, x2) {
    return `<path class="pipeline-connector" d="M ${x1 + 7} 62 H ${x2 - 9}" marker-end="url(#pipeline-arrow)"></path>`;
  }
  function pathConnector(x1, y1, x2, y2) {
    const mid = Math.round((x1 + x2) / 2);
    return `<path class="pipeline-connector" d="M ${x1 + 7} ${y1} C ${mid} ${y1}, ${mid} ${y2}, ${x2 - 9} ${y2}" marker-end="url(#pipeline-arrow)"></path>`;
  }
  function visualNodeWidth(node) {
    const labelWidth = core.nodeLabel(node).length * 12 + 24;
    const timing = core.nodeTiming(node);
    const timingWidth = timing ? timing.length * 10 + 24 : 0;
    return Math.max(144, labelWidth, timingWidth);
  }
  function progressGraph(plan, phases, closing, revealing, fanout) {
    if (fanout) return fanoutProgressGraph(plan, phases, closing, revealing);
    const output = [];
    const nodeGap = 32;
    const graphPadding = 32;
    let nextX = graphPadding;
    let previous;
    const addNode = (node) => {
      const width = visualNodeWidth(node);
      const x = previous ? previous.x + previous.width / 2 + nodeGap + width / 2 : nextX + width / 2;
      if (previous) output.push(connector(previous.x, x));
      output.push(visualNode(node, x, 62, width, !previous));
      previous = { x, width };
      nextX = x + width / 2 + graphPadding;
      return previous;
    };
    if (plan) addNode(plan);
    phases.forEach((phase) => {
      let first;
      phase.nodes.forEach((node) => {
        const layout = addNode(node);
        if (!first) first = layout;
      });
      const phaseStart = first.x - first.width / 2 - 14;
      const phaseWidth = previous.x + previous.width / 2 + 14 - phaseStart;
      output.unshift(`<g class="pipeline-visual-phase"><rect x="${phaseStart}" y="16" width="${phaseWidth}" height="82" rx="4"></rect><text class="pipeline-phase-label" x="${phaseStart + 10}" y="34">${core.escapeHTML(phase.label)}</text></g>`);
    });
    closing.forEach(addNode);
    const width = Math.max(760, nextX);
    const transition = revealing ? " pipeline-graph-revealing" : "";
    const hasRequirements = phases.some((phase) => phase.nodes.some((node) => node.kind === "requirements"));
    const hasPartPlanning = phases.some((phase) => phase.nodes.some((node) => node.kind === "part_plan"));
    const hasPartAuthor = phases.some((phase) => phase.nodes.some((node) => node.kind === "part_author"));
    const middle = [hasRequirements ? "요구 연결" : "", hasPartPlanning ? "파트 계획" : "", "섹션 작성", "파트 조립", hasPartAuthor ? "파트 최종 작성" : ""].filter(Boolean).join(", ");
    const aria = `계획, ${middle}, 최종화, 산출물 순서의 리포트 생성 진행 상황`;
    return `<svg class="pipeline-graph${transition}" style="--pipeline-width: ${width}px; --pipeline-height: 136px" viewBox="0 0 ${width} 136" role="img" aria-label="${aria}"><defs>${arrowMarker()}</defs>${output.join("")}</svg>`;
  }
  function arrowMarker() {
    return `<marker id="pipeline-arrow" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="6" markerHeight="6" orient="auto"><path d="M 0 0 L 8 4 L 0 8 z"></path></marker>`;
  }
  function groupByPart(nodes) {
    const groups = new Map();
    nodes.forEach((node) => {
      const part = Number.isInteger(node.part_index) && node.part_index > 0 ? node.part_index : 1;
      if (!groups.has(part)) groups.set(part, []);
      groups.get(part).push(node);
    });
    groups.forEach((items) => items.sort((a, b) => (a.section_index || 0) - (b.section_index || 0)));
    return [...groups.entries()].sort(([a], [b]) => a - b);
  }
  function maxNodeWidth(nodes) {
    return nodes.reduce((width, node) => Math.max(width, visualNodeWidth(node)), 144);
  }
  function fanoutProgressGraph(plan, phases, closing, revealing) {
    const requirements = phases.flatMap((phase) => phase.nodes).filter((node) => node.kind === "requirements");
    const partPlans = phases.flatMap((phase) => phase.nodes).filter((node) => node.kind === "part_plan");
    const sections = phases.flatMap((phase) => phase.nodes).filter((node) => node.kind === "section");
    const parts = phases.flatMap((phase) => phase.nodes).filter((node) => node.kind === "part");
    const partFinals = phases.flatMap((phase) => phase.nodes).filter((node) => node.kind === "part_edit" || node.kind === "part_author");
    if (!sections.length || !parts.length) return progressGraph(plan, phases, closing, revealing, false);
    const output = [];
    const padding = 36;
    const nodeGap = 42;
    const branchGap = 76;
    const rowGap = 84;
    const rows = groupByPart(sections);
    const rowCount = Math.max(1, rows.length);
    const lastRowY = 62 + (rowCount - 1) * rowGap;
    const centerY = 62 + (lastRowY - 62) / 2;
    const height = Math.max(136, Math.round(lastRowY + 74));
    const maxSectionsPerPart = Math.max(...rows.map(([, items]) => items.length));
    const partPlanWidth = partPlans.length ? maxNodeWidth(partPlans) : 0;
    const sectionWidth = maxNodeWidth(sections);
    const partWidth = maxNodeWidth(parts);
    const partFinalWidth = partFinals.length ? maxNodeWidth(partFinals) : 0;
    const prefixNodes = [plan, ...requirements].filter(Boolean);
    const prefixLayouts = [];
    let prefixRight = padding;
    prefixNodes.forEach((node, index) => {
      const width = visualNodeWidth(node);
      const x = prefixRight + width / 2;
      prefixLayouts.push({ node, x, width, fixed: index === 0 });
      prefixRight = x + width / 2 + nodeGap;
    });
    const branchSource = prefixLayouts[prefixLayouts.length - 1];
    const partPlanX = partPlans.length ? prefixRight - nodeGap + branchGap + partPlanWidth / 2 : 0;
    const firstSectionX = (partPlans.length ? partPlanX + partPlanWidth / 2 : prefixRight - nodeGap) + branchGap + sectionWidth / 2;
    const sectionStep = sectionWidth + nodeGap;
    const partX = firstSectionX + Math.max(0, maxSectionsPerPart - 1) * sectionStep + sectionWidth / 2 + branchGap + partWidth / 2;
    const partFinalX = partFinals.length ? partX + partWidth / 2 + branchGap + partFinalWidth / 2 : 0;
    const closingNodes = closing.map((node) => ({ node, width: visualNodeWidth(node) }));
    let closingX = (partFinals.length ? partFinalX + partFinalWidth / 2 : partX + partWidth / 2) + branchGap;
    const partPlanByIndex = new Map(partPlans.map((node) => [node.part_index || 1, node]));
    const partByIndex = new Map(parts.map((node) => [node.part_index || 1, node]));
    const partFinalByIndex = new Map(partFinals.map((node) => [node.part_index || 1, node]));
    const sectionPhaseStart = firstSectionX - sectionWidth / 2 - 16;
    const sectionPhaseEnd = firstSectionX + Math.max(0, maxSectionsPerPart - 1) * sectionStep + sectionWidth / 2 + 16;
    if (partPlans.length) {
      output.push(`<g class="pipeline-visual-phase pipeline-visual-phase-fanout"><rect x="${partPlanX - partPlanWidth / 2 - 16}" y="16" width="${partPlanWidth + 32}" height="${height - 38}" rx="4"></rect><text class="pipeline-phase-label" x="${partPlanX - partPlanWidth / 2 - 6}" y="34">파트 계획</text></g>`);
    }
    output.push(`<g class="pipeline-visual-phase pipeline-visual-phase-fanout"><rect x="${sectionPhaseStart}" y="16" width="${sectionPhaseEnd - sectionPhaseStart}" height="${height - 38}" rx="4"></rect><text class="pipeline-phase-label" x="${sectionPhaseStart + 10}" y="34">섹션 작성</text></g>`);
    output.push(`<g class="pipeline-visual-phase pipeline-visual-phase-fanout"><rect x="${partX - partWidth / 2 - 16}" y="16" width="${partWidth + 32}" height="${height - 38}" rx="4"></rect><text class="pipeline-phase-label" x="${partX - partWidth / 2 - 6}" y="34">파트 조립</text></g>`);
    if (partFinals.length) {
      const label = partFinals.some((node) => node.kind === "part_author") ? "파트 최종 작성" : "파트 편집";
      output.push(`<g class="pipeline-visual-phase pipeline-visual-phase-fanout"><rect x="${partFinalX - partFinalWidth / 2 - 16}" y="16" width="${partFinalWidth + 32}" height="${height - 38}" rx="4"></rect><text class="pipeline-phase-label" x="${partFinalX - partFinalWidth / 2 - 6}" y="34">${label}</text></g>`);
    }
    prefixLayouts.forEach((layout, index) => {
      if (index > 0) output.push(pathConnector(prefixLayouts[index - 1].x, centerY, layout.x, centerY));
      output.push(visualNode(layout.node, layout.x, centerY, layout.width, layout.fixed));
    });
    const partLayouts = [];
    const branchTerminalLayouts = [];
    rows.forEach(([partIndex, items], rowIndex) => {
      const y = 62 + rowIndex * rowGap;
      const partPlanNode = partPlanByIndex.get(partIndex);
      let rowSource = branchSource ? { x: branchSource.x, y: centerY } : null;
      if (partPlanNode) {
        if (branchSource) output.push(pathConnector(branchSource.x, centerY, partPlanX, y));
        output.push(visualNode(partPlanNode, partPlanX, y, partPlanWidth, false));
        rowSource = { x: partPlanX, y };
      }
      items.forEach((node, index) => {
        const x = firstSectionX + index * sectionStep;
        if (rowSource) output.push(pathConnector(rowSource.x, rowSource.y, x, y));
        output.push(visualNode(node, x, y, sectionWidth, false));
      });
      const partNode = partByIndex.get(partIndex);
      if (!partNode || !items.length) return;
      items.forEach((_node, index) => output.push(pathConnector(firstSectionX + index * sectionStep, y, partX, y)));
      output.push(visualNode(partNode, partX, y, partWidth, false));
      partLayouts.push({ x: partX, y, node: partNode });
      const partFinalNode = partFinalByIndex.get(partIndex);
      if (partFinalNode) {
        output.push(pathConnector(partX, y, partFinalX, y));
        output.push(visualNode(partFinalNode, partFinalX, y, partFinalWidth, false));
        branchTerminalLayouts.push({ x: partFinalX, y, node: partFinalNode });
      } else {
        branchTerminalLayouts.push({ x: partX, y, node: partNode });
      }
    });
    let previousClosing = null;
    closingNodes.forEach(({ node, width }, index) => {
      const x = closingX + width / 2;
      const y = centerY;
      if (index === 0) {
        branchTerminalLayouts.forEach((layout) => output.push(pathConnector(layout.x, layout.y, x, y)));
      } else if (previousClosing) {
        output.push(pathConnector(previousClosing.x, previousClosing.y, x, y));
      }
      output.push(visualNode(node, x, y, width, false));
      previousClosing = { x, y, width };
      closingX = x + width / 2 + nodeGap;
    });
    const width = Math.max(760, Math.round((previousClosing ? previousClosing.x + previousClosing.width / 2 : partX + partWidth / 2) + padding));
    const transition = revealing ? " pipeline-graph-revealing" : "";
    const aria = partFinals.length ?
      `계획 뒤 ${partPlans.length ? "파트 계획과 " : ""}여러 섹션 작성으로 갈라지고 파트 조립과 ${partFinals.some((node) => node.kind === "part_author") ? "파트 최종 작성" : "파트 편집"}을 거쳐 최종화되는 병렬 리포트 생성 진행 상황` :
      requirements.length ?
        `계획과 요구 연결 뒤 ${partPlans.length ? "파트 계획과 " : ""}여러 섹션 작성으로 갈라지고 파트 조립으로 합쳐지는 병렬 리포트 생성 진행 상황` :
        `계획에서 ${partPlans.length ? "파트 계획과 " : ""}여러 섹션 작성으로 갈라지고 파트 조립으로 합쳐지는 병렬 리포트 생성 진행 상황`;
    return `<svg class="pipeline-graph pipeline-graph-fanout${transition}" style="--pipeline-width: ${width}px; --pipeline-height: ${height}px" viewBox="0 0 ${width} ${height}" role="img" aria-label="${aria}"><defs>${arrowMarker()}</defs>${output.join("")}</svg>`;
  }
  reports.pipelineGraph = { progressGraph };
})(window);
