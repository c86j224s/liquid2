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

async function renameConfluenceConnection() {
  return renameConfluenceSettingsConnection();
}

async function revokeConfluenceConnection() {
  return revokeConfluenceSettingsConnection();
}

async function deleteConfluenceConnection() {
  return deleteConfluenceSettingsConnection();
}
