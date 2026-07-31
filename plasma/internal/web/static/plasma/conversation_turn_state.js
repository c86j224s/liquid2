(function (Plasma) {
  "use strict";

  const conversation = Plasma.conversation = Plasma.conversation || {};

  function hasOpenPendingTurn(events) {
    const completed = completedUserEventIDs(events);
    return events.some((event) => {
      if (event.EventType !== "turn.agent.pending") return false;
      const userEventID = event.Payload?.user_event_id || "";
      return userEventID && !completed.has(userEventID);
    });
  }

  function completedUserEventIDs(events) {
    const completed = new Set();
    for (const event of events) {
      if (event.EventType !== "turn.agent.response") continue;
      const userEventID = event.Payload?.user_event_id || "";
      if (userEventID) completed.add(userEventID);
    }
    return completed;
  }

  Object.assign(conversation, {
    hasOpenPendingTurn,
    completedUserEventIDs
  });
})(window.Plasma);
