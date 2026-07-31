(function (Plasma) {
  "use strict";

  const mission = Plasma.mission;

  async function api(path, options = {}) {
    const init = {
      method: options.method || "GET",
      headers: { "Accept": "application/json" }
    };
    if (options.body !== undefined) {
      if (options.body instanceof FormData) {
        init.body = options.body;
      } else {
        init.headers["Content-Type"] = "application/json";
        init.body = JSON.stringify(options.body);
      }
    }
    let response;
    try {
      response = await fetch(path, init);
    } catch (err) {
      const wrapped = new Error(`Network request failed: ${err?.message || String(err)}`);
      wrapped.userMessage = "서버에 연결할 수 없습니다. 잠시 후 다시 시도하거나 Plasma 서버 상태를 확인하세요.";
      wrapped.details = { path, method: init.method, cause: err?.message || String(err) };
      wrapped.isNetworkError = true;
      throw wrapped;
    }
    const text = await response.text();
    let data = {};
    if (text.trim() !== "") {
      try {
        data = JSON.parse(text);
      } catch (err) {
        data = { raw: text };
      }
    }
    if (!response.ok) {
      const message = data.error?.message || response.statusText || "요청 실패";
      const err = new Error(`HTTP ${response.status}: ${message}`);
      err.userMessage = message;
      err.status = response.status;
      err.details = data;
      throw err;
    }
    return data;
  }

  async function missionApi(owner, suffix, options = {}) {
    const result = await api(`/api/missions/${encodeURIComponent(owner.missionId)}${suffix}`, options);
    if (!mission.ownsMissionSelection(owner)) throw new mission.StaleMissionOperationError();
    return result;
  }

  async function missionFetch(owner, suffix, options = {}) {
    const response = await fetch(`/api/missions/${encodeURIComponent(owner.missionId)}${suffix}`, options);
    if (!mission.ownsMissionSelection(owner)) throw new mission.StaleMissionOperationError();
    return response;
  }

  Plasma.transport = {
    api,
    missionApi,
    missionFetch
  };
})(window.Plasma);
