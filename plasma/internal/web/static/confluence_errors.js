function confluenceErrorDetails(err) {
  return err?.details?.error || err?.error || {};
}

function confluenceRetryMessage(retryAfter) {
  const seconds = Number.parseInt(String(retryAfter || ""), 10);
  return Number.isFinite(seconds) && seconds > 0 ? `약 ${seconds}초 후` : "잠시 후";
}

function confluenceErrorMessage(err) {
  const details = confluenceErrorDetails(err);
  const category = String(details.category || err?.category || "");
  const code = String(details.code || err?.code || "");
  const status = Number(details.status || err?.status || 0);
  const retryAfter = details.retry_after || err?.retry_after;

  if (code === "confluence_token_expired") {
    return "Confluence 연결 인증이 만료되었습니다. 설정에서 연결을 다시 추가한 뒤 사이트를 다시 선택하세요.";
  }
  if (code === "confluence_connection_revoked") {
    return "Confluence 연결이 해제되었습니다. 설정에서 연결을 다시 추가한 뒤 사이트를 다시 선택하세요.";
  }
  if (code === "confluence_unauthorized" || category === "confluence_auth" || status === 401) {
    return "Confluence 인증에 실패했습니다. 사이트 URL, Atlassian 계정 이메일, API token을 확인하고 필요하면 새 token을 만든 뒤 다시 연결하세요.";
  }
  if (code === "confluence_forbidden" || category === "confluence_permission" || status === 403) {
    return "이 연결에는 요청한 사이트, 공간 또는 페이지를 볼 권한이 없습니다. Confluence에서 접근 권한을 확인한 뒤 다시 시도하세요.";
  }
  if (code === "confluence_not_found" || category === "confluence_not_found" || status === 404) {
    return "요청한 사이트, 공간 또는 페이지를 찾을 수 없습니다. 선택한 사이트와 페이지 주소를 확인한 뒤 다시 시도하세요.";
  }
  if (code === "confluence_rate_limited" || category === "confluence_rate_limited" || status === 429) {
    return `Confluence 요청이 제한되었습니다. ${confluenceRetryMessage(retryAfter)} 다시 시도하세요.`;
  }
  if (code === "confluence_version_changed") {
    return "페이지가 다른 버전으로 변경되었습니다. 최신 페이지를 다시 미리보기한 뒤 새 스냅샷으로 승인하세요.";
  }
  if (code === "confluence_cloud_mismatch" || code === "confluence_page_mismatch") {
    return "선택한 사이트와 페이지가 일치하지 않습니다. 해당 페이지가 있는 사이트를 선택한 뒤 다시 시도하세요.";
  }
  if (code === "confluence_page_too_large") {
    return "페이지 전체가 너무 큽니다. 필요한 범위를 선택해 새 스냅샷으로 승인하세요.";
  }
  if (code === "confluence_upstream_error" || category === "confluence_upstream" || err?.isNetworkError || status >= 500) {
    return "Confluence 또는 네트워크 연결을 확인할 수 없습니다. 잠시 후 다시 시도하세요.";
  }
  return "Confluence 요청을 완료하지 못했습니다. 연결, 사이트, 페이지를 확인한 뒤 다시 시도하세요.";
}

function showConfluenceError(err) {
  showError({ userMessage: confluenceErrorMessage(err) });
}
