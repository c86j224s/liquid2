(function (Plasma) {
  "use strict";

  const $ = Plasma.dom.$;
  const conversation = Plasma.conversation = Plasma.conversation || {};

  let turnStepTimer = 0;
  let turnStepIndex = -1;
  let turnHoldActive = false;
  let turnHoldDir = null;
  const TURN_STEP_GAP_MS = 140;

  function updateTurnNavVisibility() {
    const log = $("turnLog");
    const nav = $("turnNav");
    if (!log || !nav) return;
    const scrollable = log.scrollHeight > log.clientHeight + 24;
    nav.classList.toggle("hidden", !scrollable);
  }

  function onTurnNavClick(event) {
    const button = event.target.closest("[data-turn-nav]");
    if (!button) return;
    const dir = button.dataset.turnNav;
    if (dir === "top" || dir === "bottom") {
      turnNavScroll(dir);
      return;
    }
    if (event.detail === 0) turnNavScroll(dir);
  }

  function onTurnNavPointerDown(event) {
    const button = event.target.closest("[data-turn-nav]");
    if (!button) return;
    const dir = button.dataset.turnNav;
    if (dir !== "up" && dir !== "down") return;
    event.preventDefault();
    startTurnStep(dir);
    try {
      button.setPointerCapture?.(event.pointerId);
    } catch {
      /* synthetic pointer */
    }
  }

  function turnOffsets(log, turns) {
    const logTop = log.getBoundingClientRect().top;
    return turns.map((t) => log.scrollTop + (t.getBoundingClientRect().top - logTop));
  }

  function nearestTurnIndex(log, offsets) {
    const cur = log.scrollTop;
    let idx = 0;
    for (let i = 0; i < offsets.length; i += 1) {
      if (offsets[i] <= cur + 2) idx = i;
      else break;
    }
    return idx;
  }

  function smoothScrollThen(log, top, done) {
    let finished = false;
    const finish = () => {
      if (finished) return;
      finished = true;
      log.removeEventListener("scrollend", finish);
      clearTimeout(fallback);
      done();
    };
    log.addEventListener("scrollend", finish, { once: true });
    const fallback = setTimeout(finish, 700);
    log.scrollTo({ top: Math.max(0, Math.round(top)), behavior: "smooth" });
  }

  function turnStepOnce(direction) {
    const log = $("turnLog");
    if (!log || !turnHoldActive) return;
    const turns = [...log.querySelectorAll(".turn")];
    if (!turns.length) {
      stopTurnStep();
      return;
    }
    const offsets = turnOffsets(log, turns);
    if (turnStepIndex < 0) turnStepIndex = nearestTurnIndex(log, offsets);
    const prev = turnStepIndex;
    turnStepIndex = Math.max(0, Math.min(turns.length - 1, turnStepIndex + (direction === "up" ? -1 : 1)));
    if (turnStepIndex === prev) {
      stopTurnStep();
      return;
    }
    smoothScrollThen(log, offsets[turnStepIndex], () => {
      if (!turnHoldActive) return;
      turnStepTimer = setTimeout(() => turnStepOnce(direction), TURN_STEP_GAP_MS);
    });
  }

  function startTurnStep(direction) {
    stopTurnStep();
    const log = $("turnLog");
    if (!log) return;
    turnHoldActive = true;
    turnHoldDir = direction;
    turnStepIndex = nearestTurnIndex(log, turnOffsets(log, [...log.querySelectorAll(".turn")]));
    turnStepOnce(direction);
  }

  function stopTurnStep() {
    turnHoldActive = false;
    turnHoldDir = null;
    if (turnStepTimer) {
      clearTimeout(turnStepTimer);
      turnStepTimer = 0;
    }
    turnStepIndex = -1;
  }

  function turnNavScroll(direction) {
    const log = $("turnLog");
    if (!log) return;
    if (direction === "top") {
      log.scrollTo({ top: 0, behavior: "smooth" });
      return;
    }
    if (direction === "bottom") {
      log.scrollTo({ top: log.scrollHeight, behavior: "smooth" });
      return;
    }
    const turns = [...log.querySelectorAll(".turn")];
    if (!turns.length) return;
    const offsets = turnOffsets(log, turns);
    const cur = log.scrollTop;
    let target;
    if (direction === "up") {
      target = [...offsets].reverse().find((o) => o < cur - 2);
      if (target == null) target = 0;
    } else {
      target = offsets.find((o) => o > cur + 2);
      if (target == null) target = log.scrollHeight;
    }
    log.scrollTo({ top: Math.max(0, Math.round(target)), behavior: "smooth" });
  }

  Object.assign(conversation, {
    updateTurnNavVisibility,
    onTurnNavClick,
    onTurnNavPointerDown,
    turnNavScroll,
    stopTurnStep
  });
})(window.Plasma);
