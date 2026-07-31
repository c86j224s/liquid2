(function (Plasma) {
  "use strict";

  const sources = Plasma.sources;
  const state = Plasma.state;
  const renderConfluenceSpaces = (...args) => sources.renderConfluenceSpaces(...args);
  const renderConfluencePages = (...args) => sources.renderConfluencePages(...args);
  const renderConfluencePreview = (...args) => sources.renderConfluencePreview(...args);
  const renderConfluenceUpdatePanel = (...args) => sources.renderConfluenceUpdatePanel(...args);
  const clearConfluenceSearchResults = (...args) => sources.clearConfluenceSearchResults(...args);

  function clearConfluenceDiscovery() {
    clearConfluenceSearchResults();
    state.confluenceSpaces = [];
    state.confluencePages = [];
    state.confluenceBrowseContext = null;
    state.confluencePreview = null;
    state.confluenceUpdatePreview = null;
    renderConfluenceSpaces([]);
    renderConfluencePages([]);
    renderConfluencePreview(null);
    renderConfluenceUpdatePanel(null);
  }

  Object.assign(sources, {
    clearConfluenceDiscovery
  });
})(window.Plasma);
