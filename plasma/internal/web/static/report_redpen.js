(function (global) {
  "use strict";

  function createReportRedpenController(options) {
    const { body, container, status, startButton, saveButton, cancelButton } = options;
    let view = null;
    let active = false;
    let busy = false;
    let inlineEditor = null;
    let selectedBlock = null;
    let draft = "";
    let saved = "";
    let currentArtifactID = "";
    let revision = 0;

    startButton.addEventListener("click", handleStartClick);
    saveButton.addEventListener("click", save);
    cancelButton.addEventListener("click", cancelMode);
    body.addEventListener("click", selectBlockOnClick);
    body.addEventListener("pointerdown", clearSelectedBlockOnNewGesture);
    document.addEventListener("selectionchange", captureSelection);
    global.addEventListener("beforeunload", (event) => {
      if (!hasUnsavedChanges()) return;
      event.preventDefault();
      event.returnValue = "";
    });
    updateToolbar();

    function open(nextView) {
      reset();
      view = nextView?.sourceArtifactID ? { ...nextView, content: String(nextView.content ?? "") } : null;
      draft = view?.content || "";
      saved = draft;
      render(false);
      updateToolbar();
    }

    async function start() {
      if (!view || busy) return;
      setBusy(true);
      try {
        const result = await options.load(view.sourceArtifactID);
        const workcopy = result.workcopy || {};
        const artifact = workcopy.artifact || {};
        draft = result.exists ? String(result.content ?? "") : view.content;
        saved = draft;
        currentArtifactID = result.exists ? String(artifact.artifact_id || "") : "";
        revision = Number(workcopy.revision || 0);
        active = true;
        render(true);
        options.onDraftChange?.(draft);
        if (result.exists) options.notify?.(`빨간펜 작업본 v${revision}을 불러왔습니다.`);
      } catch (error) {
        options.error(error);
      } finally {
        setBusy(false);
      }
    }

    function captureSelection() {
      if (!active || inlineEditor) return;
      const selection = global.getSelection?.();
      if (!selection || selection.isCollapsed) return;
      if (!body.contains(selection.anchorNode) || !body.contains(selection.focusNode)) return;
      setSelectedBlock(global.ReportRedpenMarkdown.selectionBlock(body, selection));
      selection.removeAllRanges();
    }

    function selectBlockOnClick(event) {
      if (!active || inlineEditor) return;
      const block = global.ReportRedpenMarkdown.blockForTarget(body, event.target);
      if (!block) return;
      event.preventDefault();
      setSelectedBlock(block);
      global.getSelection?.()?.removeAllRanges?.();
    }

    function clearSelectedBlockOnNewGesture(event) {
      if (!active || inlineEditor || !selectedBlock) return;
      if (!selectedBlock.element.contains(event.target)) setSelectedBlock(null);
    }

    function setSelectedBlock(block) {
      selectedBlock?.element?.classList.remove("report-redpen-block-selected");
      selectedBlock = block;
      selectedBlock?.element?.classList.add("report-redpen-block-selected");
      updateToolbar();
    }

    function handleStartClick() {
      if (inlineEditor) return;
      if (active) return editSelectedBlock();
      return start();
    }

    function editSelectedBlock() {
      if (!selectedBlock || inlineEditor) {
        options.error(new Error("고칠 문단, 제목 또는 단순 목록 항목을 먼저 선택하세요."));
        return;
      }
      const { element, startLine, endLine } = selectedBlock;
      const raw = global.ReportRedpenMarkdown.rawBlock(draft, startLine, endLine);
      inlineEditor = global.ReportRedpenMarkdown.createInlineEditor(element, raw, {
        apply(value) {
          draft = global.ReportRedpenMarkdown.replaceBlock(draft, startLine, endLine, value);
          inlineEditor = null;
          setSelectedBlock(null);
          render(true);
          options.onDraftChange?.(draft);
        },
        cancel() {
          inlineEditor = null;
          setSelectedBlock(null);
          render(true);
        }
      });
      updateToolbar();
    }

    async function save() {
      if (!view || busy) return;
      if (inlineEditor) {
        options.error(new Error("현재 블록 수정을 먼저 반영하거나 취소하세요."));
        return;
      }
      setBusy(true);
      try {
        const result = await options.save(view.sourceArtifactID, draft, currentArtifactID);
        const workcopy = result.workcopy || {};
        currentArtifactID = String(workcopy.artifact?.artifact_id || currentArtifactID);
        revision = Number(workcopy.revision || revision);
        saved = draft;
        options.onDraftChange?.(draft);
        await options.saved?.(result);
        options.notify?.(result.changed ? `빨간펜 작업본 v${revision}을 저장했습니다.` : "저장된 작업본과 내용이 같습니다.");
      } catch (error) {
        options.error(error);
      } finally {
        setBusy(false);
      }
    }

    function cancelMode() {
      if (!active || !confirmDiscard()) return;
      active = false;
      inlineEditor = null;
      setSelectedBlock(null);
      draft = view.content;
      saved = draft;
      currentArtifactID = "";
      revision = 0;
      render(false);
      options.onDraftChange?.(draft);
      updateToolbar();
    }

    function render(editable) {
      if (!view) return;
      body.innerHTML = options.render(editable ? draft : view.content, editable);
      body.classList.toggle("report-redpen-active", editable);
      container?.classList.toggle("report-redpen-mode", editable);
      options.enhance?.(body);
      setSelectedBlock(null);
    }

    function beforeLeave() {
      if (!confirmDiscard()) return false;
      reset();
      return true;
    }

    function confirmDiscard() {
      return !hasUnsavedChanges() || options.confirm("저장하지 않은 빨간펜 수정이 있습니다. 수정 내용을 버릴까요?");
    }

    function hasUnsavedChanges() {
      return active && draft !== saved;
    }

    function setBusy(value) {
      busy = Boolean(value);
      updateToolbar();
    }

    function updateToolbar() {
      startButton.classList.toggle("hidden", !view);
      saveButton.classList.toggle("hidden", !active);
      cancelButton.classList.toggle("hidden", !active);
      status.classList.toggle("hidden", !active);
      startButton.disabled = busy;
      saveButton.disabled = busy || Boolean(inlineEditor) || !draft;
      cancelButton.disabled = busy;
      startButton.title = active ? "선택한 블록 편집" : "빨간펜 편집 시작";
      startButton.setAttribute("aria-label", startButton.title);
      startButton.setAttribute("aria-pressed", String(active));
      saveButton.title = revision ? `빨간펜 작업본 저장 (현재 v${revision})` : "빨간펜 작업본 저장";
      status.textContent = busy
        ? "처리 중"
        : inlineEditor
          ? "블록 편집 중"
          : hasUnsavedChanges()
            ? "저장 안 됨"
            : revision
              ? `v${revision} 저장됨`
              : "빨간펜 켜짐";
    }

    function reset() {
      view = null;
      active = false;
      busy = false;
      inlineEditor = null;
      setSelectedBlock(null);
      draft = "";
      saved = "";
      currentArtifactID = "";
      revision = 0;
      body.classList.remove("report-redpen-active");
      container?.classList.remove("report-redpen-mode");
      updateToolbar();
    }

    return { open, reset, beforeLeave, copyContent: () => active ? draft : view?.content || "", hasUnsavedChanges };
  }

  global.createReportRedpenController = createReportRedpenController;
})(window);
