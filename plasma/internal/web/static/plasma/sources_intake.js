(function (Plasma) {
  "use strict";

  const sources = Plasma.sources;
  const $ = Plasma.dom.$;
  const missionApi = Plasma.transport.missionApi;
  const captureMissionSelection = Plasma.mission.captureMissionSelection;
  const ownsMissionSelection = Plasma.mission.ownsMissionSelection;
  const isStaleMissionOperation = Plasma.mission.isStaleMissionOperation;
  const requireMission = () => sources.dependency("requireMission")();
  const reloadMission = (...args) => sources.dependency("reloadMission")(...args);
  const showError = (...args) => sources.dependency("showError")(...args);
  const setReportNotice = (...args) => sources.dependency("setReportNotice")(...args);
  const normalizeSourceURL = sources.normalizeSourceURL;
  const addURLSource = (...args) => sources.addURLSource(...args);
  const mediaLocator = (...args) => sources.mediaLocator(...args);

  async function addTextSource(event) {
    event.preventDefault();
    if (!requireMission()) return;
    const owner = captureMissionSelection();
    try {
      await missionApi(owner, "/sources/text", {
        method: "POST",
        body: {
          title: $("sourceTitle").value,
          external_uri: $("sourceURI").value,
          content: $("sourceContent").value
        }
      });
      if (!ownsMissionSelection(owner)) return;
      $("sourceTitle").value = "";
      $("sourceURI").value = "";
      $("sourceContent").value = "";
      await reloadMission(owner.missionId);
    } catch (err) {
      if (!isStaleMissionOperation(err) && ownsMissionSelection(owner)) showError(err);
    }
  }

  async function addUploadSource(event) {
    event.preventDefault();
    if (!requireMission()) return;
    const fileInput = $("sourceUploadFile");
    const file = fileInput.files && fileInput.files[0];
    if (!file) {
      showError(new Error("업로드할 파일을 선택하세요."));
      return;
    }
    const form = new FormData();
    form.append("file", file);
    form.append("title", $("sourceUploadTitle").value.trim());
    const owner = captureMissionSelection();
    try {
      await missionApi(owner, "/sources/upload", {
        method: "POST",
        body: form
      });
      if (!ownsMissionSelection(owner)) return;
      fileInput.value = "";
      $("sourceUploadTitle").value = "";
      setReportNotice("업로드한 파일을 원문 소스로 저장했습니다.");
      await reloadMission(owner.missionId);
    } catch (err) {
      if (!isStaleMissionOperation(err) && ownsMissionSelection(owner)) showError(err);
    }
  }

  async function addURLSourceFromTextForm() {
    if (!requireMission()) return;
    const url = $("sourceURI").value.trim();
    if (!normalizeSourceURL(url)) {
      showError(new Error("원문 URI에 http 또는 https URL을 입력하세요."));
      return;
    }
    const added = await addURLSource(url, $("sourceTitle").value.trim());
    if (!added) return;
    $("sourceTitle").value = "";
    $("sourceURI").value = "";
    $("sourceContent").value = "";
  }

  async function addMediaURLSource(event) {
    event.preventDefault();
    if (!requireMission()) return;
    const owner = captureMissionSelection();
    try {
      const result = await missionApi(owner, "/sources/media_url", {
        method: "POST",
        body: {
          url: $("mediaSourceURL").value,
          title: $("mediaSourceTitle").value,
          license: $("mediaSourceLicense").value,
          attribution: $("mediaSourceAttribution").value
        }
      });
      if (!ownsMissionSelection(owner)) return;
      $("mediaSourceURL").value = "";
      $("mediaSourceTitle").value = "";
      $("mediaSourceLicense").value = "";
      $("mediaSourceAttribution").value = "";
      const snapshot = result.snapshot || result.Snapshot || {};
      const locator = mediaLocator(snapshot);
      if (locator?.media_kind === "image") {
        setReportNotice("이미지 소스를 저장했습니다. 현재 빌드에서는 이미지 내용 분석 없이 메타데이터와 원본만 사용합니다.");
      } else if (locator?.media_kind) {
        setReportNotice("미디어 소스를 라이브 참조로 저장했습니다. 오디오·영상 inspect는 현재 지원하지 않습니다.");
      }
      await reloadMission(owner.missionId);
    } catch (err) {
      if (!isStaleMissionOperation(err) && ownsMissionSelection(owner)) showError(err);
    }
  }

  async function addPDFURLSource(event) {
    event.preventDefault();
    if (!requireMission()) return;
    const owner = captureMissionSelection();
    try {
      await missionApi(owner, "/sources/pdf_url", {
        method: "POST",
        body: {
          url: $("pdfSourceURL").value,
          title: $("pdfSourceTitle").value
        }
      });
      if (!ownsMissionSelection(owner)) return;
      $("pdfSourceURL").value = "";
      $("pdfSourceTitle").value = "";
      setReportNotice("PDF 원본을 소스로 저장했습니다. 읽기 요청은 PDF 텍스트 추출 결과를 반환합니다.");
      await reloadMission(owner.missionId);
    } catch (err) {
      if (!isStaleMissionOperation(err) && ownsMissionSelection(owner)) showError(err);
    }
  }

  Object.assign(sources, {
    addTextSource,
    addUploadSource,
    addURLSourceFromTextForm,
    addMediaURLSource,
    addPDFURLSource
  });
})(window.Plasma);
