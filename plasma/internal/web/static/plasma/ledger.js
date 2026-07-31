(function (Plasma) {
  "use strict";

  const $ = Plasma.dom.$;
  const escapeHTML = Plasma.dom.escapeHTML;
  const escapeAttr = Plasma.dom.escapeAttr;
  const shortID = Plasma.dom.shortID;
  const empty = Plasma.ui.empty;
  const updateCountChip = Plasma.ui.updateCountChip;
  const ledger = Plasma.ledger || (Plasma.ledger = {});

  function renderLedger(events) {
    updateCountChip("ledgerCount", events.length);
    updateCountChip("ledgerTabCount", events.length);
    const recent = events.slice(-150).reverse();
    $("ledgerList").innerHTML = recent.length ? recent.map((event) => `
    <button type="button" class="ledger-row" data-detail-title="장부 이벤트 상세" data-detail-json="${escapeAttr(JSON.stringify(event))}" title="${escapeAttr(event.EventID || "")}">
      <span class="ledger-seq">#${escapeHTML(String(event.Sequence ?? ""))}</span>
      <span class="ledger-type">${escapeHTML(ledgerEventLabel(event))}</span>
      <span class="ledger-time">${escapeHTML(ledgerTime(event.CreatedAt))}</span>
      <span class="ledger-id">${escapeHTML(shortID(event.EventID || ""))}</span>
    </button>
  `).join("") : empty("장부 이벤트 없음");
  }

  function ledgerEventLabel(event) {
    if (!event || event.EventType !== "mcp.tool.called") return event?.EventType || "";
    const payload = event.Payload || {};
    const toolName = payload.tool_name || "unknown";
    const status = payload.success === false ? "실패" : "성공";
    return `MCP 호출 · ${toolName} · ${status}`;
  }

  function ledgerTime(value) {
    if (!value) return "";
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return "";
    const pad = (n) => String(n).padStart(2, "0");
    return `${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
  }

  Object.assign(ledger, { ledgerEventLabel, ledgerTime, renderLedger });
  Plasma.ledger = ledger;
})(window.Plasma);
