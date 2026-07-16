package reporting

import (
	"sort"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

const reportSourceContextSchemaVersion = "plasma.report_source_context.v1"

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

func buildReportSourceContext(sources []app.SourceSnapshot, capturedAt time.Time) reportSourceContext {
	context := reportSourceContext{
		SchemaVersion:     reportSourceContextSchemaVersion,
		CapturedAt:        capturedAt.UTC().Format(time.RFC3339Nano),
		ConfluenceSources: make([]reportConfluenceSourceContext, 0),
	}
	for _, source := range sources {
		if source.Connector.ConnectorType != app.ConfluenceConnectorType || source.State.Removed || source.State.Superseded {
			continue
		}
		context.ConfluenceSources = append(context.ConfluenceSources, reportConfluenceSourceContext{
			SnapshotID:       strings.TrimSpace(source.SnapshotID),
			Title:            strings.TrimSpace(source.Title),
			ConnectorType:    app.ConfluenceConnectorType,
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

func buildReportConfluenceCheckContext(update *app.ConfluenceUpdateState) reportConfluenceCheckContext {
	if update == nil {
		return reportConfluenceCheckContext{Status: "not_checked"}
	}
	check := reportConfluenceCheckContext{
		Status:    strings.TrimSpace(update.Status),
		CheckedAt: reportSourceTime(update.CheckedAt),
	}
	switch check.Status {
	case app.ConfluenceUpdateStatusCurrent, app.ConfluenceUpdateStatusAvailable:
		check.LatestVersion = update.LatestVersion
	case app.ConfluenceUpdateStatusFailed:
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
	case app.ConfluenceErrorCategoryAuth:
		return code == app.ConfluenceErrorCodeUnauthorized || code == app.ConfluenceErrorCodeTokenExpired || code == app.ConfluenceErrorCodeRevoked
	case app.ConfluenceErrorCategoryPermission:
		return code == app.ConfluenceErrorCodeForbidden
	case app.ConfluenceErrorCategoryNotFound:
		return code == app.ConfluenceErrorCodeNotFound
	case app.ConfluenceErrorCategoryRateLimited:
		return code == app.ConfluenceErrorCodeRateLimited
	case app.ConfluenceErrorCategoryUpstream:
		return code == app.ConfluenceErrorCodeUpstream
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
