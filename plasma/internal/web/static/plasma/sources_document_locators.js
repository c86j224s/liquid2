(function (Plasma) {
  "use strict";

  const sources = Plasma.sources;
  const formatBytes = Plasma.dom.formatBytes;
  const sourceLocatorType = (...args) => sources.sourceLocatorType(...args);
  const sourceConnectorType = (...args) => sources.sourceConnectorType(...args);
  const uploadedFileContentKind = (...args) => sources.uploadedFileContentKind(...args);
  const uploadedFileMediaType = (...args) => sources.uploadedFileMediaType(...args);
  const uploadedFileFilename = (...args) => sources.uploadedFileFilename(...args);

  function documentLocator(source) {
    const raw = source.Locators || source.locators;
    if (!raw) return null;
    try {
      const parsed = typeof raw === "string" ? JSON.parse(raw) : raw;
      const locators = Array.isArray(parsed) ? parsed : [parsed];
      const locator = locators.find((item) => {
        const type = sourceLocatorType(item);
        if (sourceConnectorType(source) === "file_upload" && type === "full_document") return true;
        if (type !== "file_upload") return false;
        const contentKind = uploadedFileContentKind(item);
        const mediaType = uploadedFileMediaType(item);
        return contentKind === "text" || (!contentKind && mediaType && mediaType !== "application/pdf" && !mediaType.startsWith("image/"));
      });
      if (!locator) return null;
      return {
        filename: uploadedFileFilename(locator),
        mime_type: uploadedFileMediaType(locator),
        byte_size: locator.byte_size || locator.ByteSize || 0,
        content_kind: uploadedFileContentKind(locator)
      };
    } catch (err) {
      return null;
    }
  }

  function pdfLocator(source) {
    const raw = source.Locators || source.locators;
    if (!raw) return null;
    try {
      const parsed = typeof raw === "string" ? JSON.parse(raw) : raw;
      const locators = Array.isArray(parsed) ? parsed : [parsed];
      const locator = locators.find((item) => {
        if (sourceLocatorType(item) === "pdf_document") return true;
        const contentKind = item.content_kind || item.ContentKind || "";
        const mediaType = item.mime_type || item.MIMEType || item.media_type || item.MediaType || "";
        return sourceLocatorType(item) === "file_upload" && (contentKind === "pdf" || mediaType === "application/pdf");
      });
      if (!locator) return null;
      return {
        url: locator.url || locator.URL || "",
        filename: locator.sanitized_filename || locator.SanitizedFilename || locator.original_filename || locator.OriginalFilename || locator.filename || locator.Filename || "",
        mime_type: locator.mime_type || locator.MIMEType || locator.media_type || locator.MediaType || "application/pdf",
        byte_size: locator.byte_size || locator.ByteSize || 0,
        page_count: locator.page_count || locator.PageCount || 0,
        text_length: locator.text_length || locator.TextLength || 0,
        extraction_support: locator.extraction_support || locator.ExtractionSupport || ""
      };
    } catch (err) {
      return null;
    }
  }

  function pdfSourceText(locator) {
    const parts = [locator.mime_type || "application/pdf"];
    if (locator.page_count) parts.push(`${locator.page_count}쪽`);
    if (locator.byte_size) parts.push(formatBytes(locator.byte_size));
    if (locator.extraction_support) parts.push("텍스트 추출");
    const target = locator.url || locator.filename || "";
    return `${parts.join(" · ")}${target ? " / " + target : ""}`;
  }

  function documentSourceText(locator) {
    const parts = [];
    if (locator.mime_type) parts.push(locator.mime_type);
    if (locator.byte_size) parts.push(formatBytes(locator.byte_size));
    return `${parts.join(" · ")}${parts.length && locator.filename ? " / " : ""}${locator.filename || ""}`;
  }

  Object.assign(sources, {
    documentLocator,
    pdfLocator,
    pdfSourceText,
    documentSourceText
  });
})(window.Plasma);
