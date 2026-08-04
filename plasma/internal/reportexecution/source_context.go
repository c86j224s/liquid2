package reportexecution

import (
	"sort"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/source"
)

const (
	reportSourceContextSchemaVersion = "plasma.report_source_context.v1"
	reportConfluenceConnectorType    = "confluence_cloud"

	reportConfluenceErrorCategoryAuth        = "confluence_auth"
	reportConfluenceErrorCategoryPermission  = "confluence_permission"
	reportConfluenceErrorCategoryNotFound    = "confluence_not_found"
	reportConfluenceErrorCategoryRateLimited = "confluence_rate_limited"
	reportConfluenceErrorCategoryUpstream    = "confluence_upstream"

	reportConfluenceErrorCodeUnauthorized = "confluence_unauthorized"
	reportConfluenceErrorCodeTokenExpired = "confluence_token_expired"
	reportConfluenceErrorCodeRevoked      = "confluence_connection_revoked"
	reportConfluenceErrorCodeForbidden    = "confluence_forbidden"
	reportConfluenceErrorCodeNotFound     = "confluence_not_found"
	reportConfluenceErrorCodeRateLimited  = "confluence_rate_limited"
	reportConfluenceErrorCodeUpstream     = "confluence_upstream_error"
)

type reportSourceContext struct {
	SchemaVersion     string                          `json:"schema_version"`
	CapturedAt        string                          `json:"captured_at"`
	ConfluenceSources []reportConfluenceSourceContext `json:"confluence_sources"`
}

type reportConfluenceSourceContext struct {
	SnapshotID       string                       `json:"snapshot_id"`
	Title            string                       `json:"title"`
	ConnectorType    string                       `json:"connector_type"`
	SnapshotVersion  string                       `json:"snapshot_version,omitempty"`
	SnapshotCaptured string                       `json:"snapshot_captured_at,omitempty"`
	ExternalUpdated  string                       `json:"external_updated_at,omitempty"`
	LastCheck        reportConfluenceCheckContext `json:"last_check"`
}

type reportConfluenceCheckContext struct {
	Status        string `json:"status"`
	CheckedAt     string `json:"checked_at,omitempty"`
	LatestVersion int    `json:"latest_version,omitempty"`
	ErrorCategory string `json:"error_category,omitempty"`
	ErrorCode     string `json:"error_code,omitempty"`
}

func buildReportSourceContext(sources []source.Snapshot, capturedAt time.Time) reportSourceContext {
	context := reportSourceContext{
		SchemaVersion:     reportSourceContextSchemaVersion,
		CapturedAt:        capturedAt.UTC().Format(time.RFC3339Nano),
		ConfluenceSources: make([]reportConfluenceSourceContext, 0),
	}
	for _, source := range sources {
		if source.Connector.ConnectorType != reportConfluenceConnectorType || source.State.Removed || source.State.Superseded {
			continue
		}
		context.ConfluenceSources = append(context.ConfluenceSources, reportConfluenceSourceContext{
			SnapshotID:       strings.TrimSpace(source.SnapshotID),
			Title:            strings.TrimSpace(source.Title),
			ConnectorType:    reportConfluenceConnectorType,
			SnapshotVersion:  strings.TrimSpace(source.Connector.ExternalVersion),
			SnapshotCaptured: reportSourceTime(source.CapturedAt),
			ExternalUpdated:  reportSourceTime(source.ExternalUpdatedAt),
			LastCheck:        buildReportConfluenceCheckContext(source.State.ConfluenceUpdate),
		})
	}
	sort.Slice(context.ConfluenceSources, func(i, j int) bool {
		return context.ConfluenceSources[i].SnapshotID < context.ConfluenceSources[j].SnapshotID
	})
	return context
}

func buildReportConfluenceCheckContext(update *source.ConfluenceUpdateState) reportConfluenceCheckContext {
	if update == nil {
		return reportConfluenceCheckContext{Status: "not_checked"}
	}
	check := reportConfluenceCheckContext{
		Status:    strings.TrimSpace(update.Status),
		CheckedAt: reportSourceTime(update.CheckedAt),
	}
	switch check.Status {
	case source.ConfluenceUpdateStatusCurrent, source.ConfluenceUpdateStatusAvailable:
		check.LatestVersion = update.LatestVersion
	case source.ConfluenceUpdateStatusFailed:
		if reportConfluenceSafeError(update.ErrorCategory, update.ErrorCode) {
			check.ErrorCategory = strings.TrimSpace(update.ErrorCategory)
			check.ErrorCode = strings.TrimSpace(update.ErrorCode)
		}
	default:
		check.Status = "not_checked"
		check.CheckedAt = ""
	}
	return check
}

func reportConfluenceSafeError(category string, code string) bool {
	category = strings.TrimSpace(category)
	code = strings.TrimSpace(code)
	switch category {
	case reportConfluenceErrorCategoryAuth:
		return code == reportConfluenceErrorCodeUnauthorized || code == reportConfluenceErrorCodeTokenExpired || code == reportConfluenceErrorCodeRevoked
	case reportConfluenceErrorCategoryPermission:
		return code == reportConfluenceErrorCodeForbidden
	case reportConfluenceErrorCategoryNotFound:
		return code == reportConfluenceErrorCodeNotFound
	case reportConfluenceErrorCategoryRateLimited:
		return code == reportConfluenceErrorCodeRateLimited
	case reportConfluenceErrorCategoryUpstream:
		return code == reportConfluenceErrorCodeUpstream
	default:
		return false
	}
}

func reportSourceTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
