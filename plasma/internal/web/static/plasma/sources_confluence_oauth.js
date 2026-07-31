(function (Plasma) {
  "use strict";

  const sources = Plasma.sources;
  const settings = Plasma.settings;
  const state = Plasma.state;
  const openSettingsTab = (...args) => settings.openSettingsTab(...args);
  const loadConfluenceConnections = (...args) => sources.loadConfluenceConnections(...args);
  const setConfluenceSettingsStatus = (...args) => settings.setConfluenceSettingsStatus(...args);

  function initConfluenceOAuthListener() {
    if (state.confluenceOAuthListenerReady) return;
    state.confluenceOAuthListenerReady = true;
    const onOAuthMessage = async (payload) => {
      if (!payload || (payload.type !== "plasma.confluence.settings.oauth" && payload.type !== "plasma.confluence.oauth")) return;
      if (payload.mission_id && payload.mission_id !== state.missionId) return;
      if (payload.ok) {
        openSettingsTab();
        await loadConfluenceConnections(payload.connection_id || "");
        setConfluenceSettingsStatus("Confluence 연결이 완료되었습니다. 미션 소스에서 연결과 사이트를 선택해 페이지를 소스로 승인할 수 있습니다.");
        return;
      }
      setConfluenceSettingsStatus(payload.message || "Confluence 연결이 완료되지 않았습니다.", "warn");
    };
    window.addEventListener("message", (event) => {
      if (event.origin !== window.location.origin) return;
      onOAuthMessage(event.data);
    });
    if ("BroadcastChannel" in window) {
      const channel = new BroadcastChannel("plasma.confluence.oauth");
      channel.addEventListener("message", (event) => onOAuthMessage(event.data));
      state.confluenceOAuthChannel = channel;
      const settingsChannel = new BroadcastChannel("plasma.confluence.settings.oauth");
      settingsChannel.addEventListener("message", (event) => onOAuthMessage(event.data));
      state.confluenceSettingsOAuthChannel = settingsChannel;
    }
  }

  Object.assign(sources, {
    initConfluenceOAuthListener
  });
})(window.Plasma);
