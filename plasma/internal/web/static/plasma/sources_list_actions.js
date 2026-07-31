(function (Plasma) {
  "use strict";

  const sources = Plasma.sources;
  const onDetailButtonClick = (...args) => sources.dependency("onDetailButtonClick")(...args);
  const checkConfluenceSourceUpdate = (...args) => sources.checkConfluenceSourceUpdate(...args);
  const readSource = (...args) => sources.readSource(...args);
  const removeSource = (...args) => sources.removeSource(...args);
  const restoreSource = (...args) => sources.restoreSource(...args);

  async function onSourceListClick(event) {
    if (onDetailButtonClick(event)) return;
    const confluenceUpdateButton = event.target.closest("[data-confluence-source-update]");
    if (confluenceUpdateButton) {
      await checkConfluenceSourceUpdate(confluenceUpdateButton.dataset.confluenceSourceUpdate);
      return;
    }
    const readButton = event.target.closest("[data-source-read]");
    if (readButton) {
      await readSource(readButton.dataset.sourceRead);
      return;
    }
    const removeButton = event.target.closest("[data-source-remove]");
    if (removeButton) {
      await removeSource(removeButton.dataset.sourceRemove);
      return;
    }
    const restoreButton = event.target.closest("[data-source-restore]");
    if (restoreButton) {
      await restoreSource(restoreButton.dataset.sourceRestore);
    }
  }

  Object.assign(sources, {
    onSourceListClick
  });
})(window.Plasma);
