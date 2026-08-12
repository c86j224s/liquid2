(function (Plasma) {
  "use strict";

  const state = Plasma.state;
  const { $, escapeHTML, escapeAttr, shortID, timeShort } = Plasma.dom;
  const conversation = Plasma.conversation = Plasma.conversation || {};
  const callbacks = conversation._renderCallbacks || {
    renderMarkdown: (value) => Plasma.dom.escapeHTML(value),
    empty: (value) => Plasma.ui.empty(value)
  };

  function configureRendering(options = {}) {
    Object.assign(callbacks, options);
    conversation._renderCallbacks = callbacks;
  }

  function renderTurns(events) {
    conversation.reconcileLiveTurnForDurableEvents?.(events);
    conversation.syncLiveTurnSubscription?.(events);
    const turns = events.filter((event) =>
      event.EventType === "turn.user" ||
      event.EventType === "controller.strategy.selected" ||
      event.EventType === "turn.agent.response" ||
      event.EventType === "turn.agent.pending" ||
      event.EventType === "turn.agent.compacted" ||
      event.EventType === "agent.session.reset"
    );
    const userEventIDs = new Set(
      turns
        .filter((event) => event.EventType === "turn.user" && event.EventID)
        .map((event) => String(event.EventID))
    );
    const steeringEventsByUserEventID = new Map();
    turns.forEach((event) => {
      if (event.EventType !== "controller.strategy.selected") return;
      const payload = event.Payload || {};
      const userEventID = payload.user_event_id ? String(payload.user_event_id) : "";
      if (!userEventID || !userEventIDs.has(userEventID)) return;
      const eventsForUser = steeringEventsByUserEventID.get(userEventID) || [];
      eventsForUser.push(event);
      steeringEventsByUserEventID.set(userEventID, eventsForUser);
    });
    const renderSteeringMeta = (event) => {
      const payload = event.Payload || {};
      const label = payload.strategy_label || payload.strategy_id || "조향 전략";
      return `
        <div class="turn-steering-meta">
          <span class="turn-steering-label">자동 조향</span>
          <span class="turn-steering-text"><strong>${escapeHTML(label)}</strong></span>
        </div>
      `;
    };
    const completed = conversation.completedUserEventIDs(events);
    const html = turns.map((event) => {
      const payload = event.Payload || {};
      const isUser = event.EventType === "turn.user";
      const isPending = event.EventType === "turn.agent.pending";
      const isSessionReset = event.EventType === "agent.session.reset";
      if (isPending && completed.has(payload.user_event_id || "")) return "";
      if (isSessionReset) return renderSessionResetTurn(event, payload);
      if (event.EventType === "turn.agent.compacted") return renderContextCompactionTurn(event, payload);
      if (event.EventType === "controller.strategy.selected") {
        const userEventID = payload.user_event_id ? String(payload.user_event_id) : "";
        if (userEventID && userEventIDs.has(userEventID)) return "";
        return renderStandaloneSteeringTurn(event, payload);
      }
      return renderConversationTurn(event, payload, isUser, isPending, steeringEventsByUserEventID, renderSteeringMeta);
    }).filter(Boolean);
    if (state.pendingTurn && state.pendingTurn.missionId === state.missionId) {
      html.push(`
        <div class="turn user pending">
          <div class="turn-label">사용자 / ${escapeHTML(timeShort(state.pendingTurn.createdAt))}</div>
          <div class="turn-text">${escapeHTML(state.pendingTurn.text)}</div>
        </div>
      `);
    }
    if (state.turnPending && state.pendingTurn && state.pendingTurn.missionId === state.missionId) {
      const live = conversation.liveTurnSnapshot?.(state.pendingTurn.userEventID);
      html.push(`
        <div class="turn pending">
          <div class="turn-label">에이전트</div>
          ${renderPendingAgentBody(live)}
        </div>
      `);
    }
    const log = $("turnLog");
    const missionChanged = state.turnScrollMission !== state.missionId;
    const nearBottom = log.scrollHeight - log.scrollTop - log.clientHeight < 80;
    log.innerHTML = html.length ? html.join("") : callbacks.empty("아직 대화가 없습니다.");
    window.Plasma?.reports?.renderPlasmaMath?.(log);
    window.Plasma?.reports?.renderPlasmaMermaid?.(log);
    window.Plasma?.reports?.enhanceImages?.(log);
    if (missionChanged || nearBottom) log.scrollTop = log.scrollHeight;
    state.turnScrollMission = state.missionId;
    conversation.updateTurnNavVisibility();
  }

  function contextWindowPercent(usedTokens, windowTokens) {
    const used = Number(usedTokens);
    const windowSize = Number(windowTokens);
    if (!Number.isFinite(used) || !Number.isFinite(windowSize) || used < 0 || windowSize <= 0) return "";
    return `${((used / windowSize) * 100).toFixed(1)}%`;
  }

  function renderContextCompactionTurn(event, payload) {
    const compactedWindow = payload.agent_usage?.context_window || {};
    const before = contextWindowPercent(payload.context_used_tokens, payload.context_window_tokens);
    const after = contextWindowPercent(compactedWindow.used_tokens, compactedWindow.window_tokens || payload.context_window_tokens);
    const rangeBadge = before && after
      ? `<span class="badge muted">${escapeHTML(before)} → ${escapeHTML(after)}</span>`
      : "";
    const action = payload.manual === true ? "압축 완료" : "자동 압축";
    const detail = before && after
      ? `에이전트 세션의 작업 맥락을 ${before}에서 ${after}로 줄였습니다.`
      : "에이전트 세션의 작업 맥락을 압축했습니다.";
    return `
      <div class="turn session-event compaction-event">
        <div class="turn-label">컨텍스트 / ${escapeHTML(timeShort(event.CreatedAt))} <span class="badge session-new">${action}</span> ${rangeBadge}</div>
        <div class="turn-text">${escapeHTML(detail)} 같은 세션에서 작업을 이어갑니다.</div>
      </div>
    `;
  }

  function renderSessionResetTurn(event, payload) {
    const executor = payload.agent_executor || "agent";
    const model = payload.agent_model || "";
    const reasoningEffort = payload.agent_reasoning_effort || "";
    const previousID = payload.previous_agent_session_id || "";
    const previousBadge = previousID ? `
      <span class="badge muted" title="${escapeAttr(previousID)}">이전 ${escapeHTML(shortID(previousID))}</span>
      <button type="button" class="mini-copy" data-copy-text="${escapeAttr(previousID)}">이전 ID 복사</button>
    ` : "";
    const modelBadge = model ? `<span class="badge muted" title="${escapeAttr(model)}">모델 ${escapeHTML(model)}</span>` : "";
    const effortBadge = reasoningEffort ? `<span class="badge muted">추론 ${escapeHTML(reasoningEffort)}</span>` : "";
    const resetText = previousID
      ? `${executor} 새 세션이 준비되었습니다. 다음 메시지는 이전 세션을 resume하지 않습니다.`
      : `${executor} 새 세션이 준비되었습니다. 다음 메시지는 새 세션으로 시작됩니다.`;
    return `
      <div class="turn session-event">
        <div class="turn-label">세션 / ${escapeHTML(timeShort(event.CreatedAt))} <span class="badge session-new">${escapeHTML(executor)} 새 세션 준비</span> ${modelBadge} ${effortBadge} ${previousBadge}</div>
        <div class="turn-text">${escapeHTML(resetText)}</div>
      </div>
    `;
  }

  function renderStandaloneSteeringTurn(event, payload) {
    const label = payload.strategy_label || payload.strategy_id || "조향 전략";
    const reason = payload.reason || "";
    const strategyID = payload.strategy_id ? `<span class="badge muted">${escapeHTML(payload.strategy_id)}</span>` : "";
    return `
      <div class="turn controller">
        <div class="turn-label">조향 / ${escapeHTML(timeShort(event.CreatedAt))} ${strategyID}</div>
        <div class="turn-text"><strong>${escapeHTML(label)}</strong>${reason ? ` · ${escapeHTML(reason)}` : ""}</div>
      </div>
    `;
  }

  function renderConversationTurn(event, payload, isUser, isPending, steeringEventsByUserEventID, renderSteeringMeta) {
    const text = payload.text || JSON.stringify(payload);
    const executor = payload.agent_executor || (isUser ? "" : "agent");
    const isWorkflowTurn = payload.workflow_run_id || payload.kind === "workflow_steering";
    const terminalBadge = payload.kind === "agent_error"
      ? `<span class="badge danger">응답 실패</span>`
      : payload.kind === "agent_canceled"
        ? `<span class="badge muted">응답 취소</span>`
        : "";
    const sessionBadge = renderSessionBadge(payload, isUser, isPending, executor);
    const live = isPending ? conversation.liveTurnSnapshot?.(payload.user_event_id) : null;
    const body = isPending
      ? renderPendingAgentBody(live)
      : isUser
        ? `<div class="turn-text">${escapeHTML(text)}</div>${(steeringEventsByUserEventID.get(String(event.EventID || "")) || []).map(renderSteeringMeta).join("")}`
        : `<div class="turn-text turn-markdown">${callbacks.renderMarkdown(text)}</div>`;
    const copyButton = isPending
      ? ""
      : `<button type="button" class="mini-copy turn-copy" data-copy-text="${escapeAttr(text)}" title="이 메시지 복사">복사</button>`;
    return `
      <div class="turn ${isUser ? "user" : "agent"} ${isPending ? "pending" : ""}">
        <div class="turn-label">${isWorkflowTurn ? "워크플로우" : (isUser ? "사용자" : "에이전트")} / ${escapeHTML(timeShort(event.CreatedAt))} ${terminalBadge}${sessionBadge}${copyButton}</div>
        ${body}
      </div>
    `;
  }

  function renderPendingAgentBody(live) {
    if (live?.state === "answer" && live.preview) {
      const status = live?.status || "응답 작성 중...";
      return `
        <div class="turn-text turn-markdown live-turn-preview">${callbacks.renderMarkdown(live.preview)}</div>
        <div class="turn-text live-turn-status" role="status" aria-live="polite" aria-atomic="true"><span class="spinner"></span> <span class="live-turn-status-text">${escapeHTML(status)}</span></div>
      `;
    }
    const status = live?.status || "에이전트 응답을 기다리는 중...";
    return `<div class="turn-text live-turn-status" role="status" aria-live="polite" aria-atomic="true"><span class="spinner"></span> <span class="live-turn-status-text">${escapeHTML(status)}</span></div>`;
  }

  function renderSessionBadge(payload, isUser, isPending, executor) {
    const sessionID = payload.agent_session_id || "";
    const isNewSession = !isUser && !isPending && payload.kind === "agent_response" && payload.resumed === false && sessionID;
    return sessionID ? `
      <span class="badge ${isNewSession ? "session-new" : "muted"}" title="${escapeAttr(sessionID)}">${escapeHTML(executor)} ${isNewSession ? "새 세션 " : ""}${escapeHTML(shortID(sessionID))}</span>
      <button type="button" class="mini-copy" data-copy-text="${escapeAttr(sessionID)}">세션 ID 복사</button>
    ` : (executor ? `<span class="badge muted">${escapeHTML(executor)}</span>` : "");
  }

  Object.assign(conversation, {
    configureRendering,
    renderPendingAgentBody,
    renderTurns
  });
})(window.Plasma);
