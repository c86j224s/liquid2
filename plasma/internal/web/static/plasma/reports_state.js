(function reportsState(root) {
  "use strict";
  const reports = root.Plasma.reports;
  const state = root.Plasma.state;
function reportDraftState(events) {
  const completed = reports.completedReportDraftPendingEventIDs(events);
  for (let index = events.length - 1; index >= 0; index -= 1) {
    const event = events[index];
    if (event.EventType !== "report.draft.pending" && event.EventType !== "report.design.pending" && event.EventType !== "report.humanize.pending" && event.EventType !== "report.patch.pending") continue;
    if (!completed.has(event.EventID)) {
      return { state: "pending", event };
    }
  }
  for (let index = events.length - 1; index >= 0; index -= 1) {
    const event = events[index];
    if (event.EventType === "report.draft.failed" || event.EventType === "report.design.failed" || event.EventType === "report.humanize.failed" || event.EventType === "report.patch.failed") {
      return { state: "failed", event };
    }
    if (event.EventType === "report.humanize.skipped") {
      return { state: "skipped", event };
    }
    if (event.EventType === "report.drafted" || event.EventType === "report.artifact.created" || event.EventType === "report.artifact.exported") {
      return { state: "completed", event };
    }
  }
  return { state: "idle", event: null };
}

function completedReportDraftPendingEventIDs(events) {
  const completed = new Set();
  for (const event of events) {
    if (event.EventType === "report.drafted" || event.EventType === "report.artifact.created" || event.EventType === "report.artifact.exported") {
      const pendingEventID = event.Payload?.pending_event_id || event.Payload?.generation?.pending_event_id || "";
      if (pendingEventID) completed.add(pendingEventID);
    }
    if (event.EventType === "report.draft.failed" || event.EventType === "report.design.failed" || event.EventType === "report.humanize.failed" || event.EventType === "report.humanize.skipped" || event.EventType === "report.patch.failed") {
      const pendingEventID = event.Payload?.pending_event_id || "";
      if (pendingEventID) completed.add(pendingEventID);
    }
  }
  return completed;
}

function reportArtifactPayloads() {
  const events = state.detail?.events || [];
  return events
    .filter((event) => event.EventType === "report.artifact.created")
    .map((event) => ({ ...(event.Payload || {}), event_id: event.EventID, created_at: event.CreatedAt }))
    .reverse();
}

function conversationExportPayloads() {
  const events = state.detail?.events || [];
  return events
    .filter((event) => event.EventType === "conversation.exported" && (event.Payload || {}).kind === "conversation_export_markdown")
    .map((event) => ({ ...(event.Payload || {}), event_id: event.EventID, created_at: event.CreatedAt }))
    .reverse();
}

function reportArtifactPlanPayload(artifactPayload) {
  const pendingID = artifactPayload?.pending_event_id || "";
  const artifactID = artifactPayload?.artifact_id || "";
  const planEventID = artifactPayload?.plan_event_id || "";
  const events = state.detail?.events || [];
  for (let i = events.length - 1; i >= 0; i--) {
    const event = events[i];
    if (event.EventType !== "report.plan.created") continue;
    const payload = event.Payload || {};
    if (
      (planEventID && event.EventID === planEventID) ||
      (artifactID && payload.artifact_id === artifactID) ||
      (pendingID && payload.pending_event_id === pendingID)
    ) {
      return { ...payload, event_id: event.EventID, created_at: event.CreatedAt };
    }
  }
  return {};
}

function reportArtifactDesignedExportState(sourceArtifactID) {
  const events = state.detail?.events || [];
  const completedPending = reports.completedReportDraftPendingEventIDs(events);
  let pending = null;
  let failed = null;
  for (let i = events.length - 1; i >= 0; i--) {
    const event = events[i];
    const payload = event.Payload || {};
    if (event.EventType === "report.artifact.exported" &&
      payload.kind === "designed_html_report_artifact" &&
      payload.source_artifact_id === sourceArtifactID &&
      payload.target === "designed_html" &&
      payload.renderer_version === reports.DESIGNED_REPORT_RENDERER_VERSION) {
      return { state: "completed", payload: { ...payload, event_id: event.EventID, created_at: event.CreatedAt } };
    }
    if (!pending &&
      event.EventType === "report.design.pending" &&
      payload.source_artifact_id === sourceArtifactID &&
      !completedPending.has(event.EventID)) {
      pending = { ...payload, event_id: event.EventID, created_at: event.CreatedAt };
    }
    if (!failed &&
      event.EventType === "report.design.failed" &&
      payload.source_artifact_id === sourceArtifactID) {
      failed = { ...payload, event_id: event.EventID, created_at: event.CreatedAt };
    }
  }
  if (pending) return { state: "pending", payload: pending };
  if (failed) return { state: "failed", payload: failed };
  return { state: "idle", payload: {} };
}

function reportArtifactHumanizedExportState(sourceArtifactID) {
  const events = state.detail?.events || [];
  let failed = null;
  for (let i = events.length - 1; i >= 0; i--) {
    const event = events[i];
    const payload = event.Payload || {};
    if (event.EventType === "report.artifact.exported" &&
      payload.kind === "humanized_markdown_report_artifact" &&
      payload.source_artifact_id === sourceArtifactID &&
      payload.target === "humanized_markdown") {
      return { state: "completed", payload: { ...payload, event_id: event.EventID, created_at: event.CreatedAt } };
    }
    if (!failed &&
      event.EventType === "report.humanize.failed" &&
      payload.source_artifact_id === sourceArtifactID &&
      payload.target === "humanized_markdown") {
      failed = { ...payload, event_id: event.EventID, created_at: event.CreatedAt };
    }
    if (!failed &&
      event.EventType === "report.humanize.skipped" &&
      payload.source_artifact_id === sourceArtifactID &&
      payload.target === "humanized_markdown") {
      return { state: "skipped", payload: { ...payload, event_id: event.EventID, created_at: event.CreatedAt } };
    }
    if (!failed &&
      event.EventType === "report.humanize.pending" &&
      payload.source_artifact_id === sourceArtifactID &&
      payload.target === "humanized_markdown") {
      return { state: "pending", payload: { ...payload, event_id: event.EventID, created_at: event.CreatedAt } };
    }
  }
  if (failed) return { state: "failed", payload: failed };
  return { state: "idle", payload: {} };
}

function reportArtifactRedpenState(sourceArtifactID) {
  const events = state.detail?.events || [];
  for (let index = events.length - 1; index >= 0; index -= 1) {
    const event = events[index];
    const payload = event.Payload || {};
    if (event.EventType === "report.redpen.saved" &&
      payload.kind === "redpen_markdown_report_artifact" &&
      payload.source_artifact_id === sourceArtifactID) {
      return { state: "completed", payload: { ...payload, event_id: event.EventID, created_at: event.CreatedAt } };
    }
  }
  return { state: "idle", payload: {} };
}

  Object.assign(reports, { reportDraftState, completedReportDraftPendingEventIDs, reportArtifactPayloads, conversationExportPayloads, reportArtifactPlanPayload, reportArtifactDesignedExportState, reportArtifactHumanizedExportState, reportArtifactRedpenState });
})(window);
