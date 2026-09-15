package reportdocument

import (
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"strings"
)

// NormalizeScope validates explicit scope before app applies projection defaults.
func NormalizeScope(scope ReportEvidenceScope) (ReportEvidenceScope, error) {
	var err error
	scope.EvidenceIDs, err = normalizeIDList("evd_", scope.EvidenceIDs)
	if err != nil {
		return ReportEvidenceScope{}, err
	}
	scope.ClaimIDs, err = normalizeIDList("clm_", scope.ClaimIDs)
	if err != nil {
		return ReportEvidenceScope{}, err
	}
	scope.QuestionIDs, err = normalizeIDList("qst_", scope.QuestionIDs)
	if err != nil {
		return ReportEvidenceScope{}, err
	}
	scope.OptionIDs, err = normalizeIDList("opt_", scope.OptionIDs)
	if err != nil {
		return ReportEvidenceScope{}, err
	}
	if scope.IncludeProposed {
		return ReportEvidenceScope{}, fmt.Errorf("%w: report drafts cannot include proposed records", producterror.ErrInvalidInput)
	}
	if len(scope.QuestionIDs)+len(scope.OptionIDs) > 0 {
		return ReportEvidenceScope{}, fmt.Errorf("%w: report drafts do not support question or option records until approval semantics are defined", producterror.ErrInvalidInput)
	}
	scope.AcceptedOnly = true
	scope.IncludeProposed = false
	return scope, nil
}
func RequirePromotionEvent(event ledger.Event, reportVersionID string) error {
	if !isApprovalProducer(event.Producer) {
		return fmt.Errorf("%w: report promotion requires user or steering_chat event", producterror.ErrInvalidInput)
	}
	if event.EventType != "report.promoted" {
		return fmt.Errorf("%w: report promotion requires report.promoted event", producterror.ErrInvalidInput)
	}
	var payload struct {
		ReportVersionID string `json:"report_version_id"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("%w: report.promoted payload is invalid", producterror.ErrInvalidInput)
	}
	if strings.TrimSpace(payload.ReportVersionID) != reportVersionID {
		return fmt.Errorf("%w: report.promoted event does not reference report version", producterror.ErrInvalidInput)
	}
	return nil
}

var allowedReportFormatIntents = map[string]bool{
	"briefing":    true,
	"full_report": true,
	"outline":     true,
}

var allowedReportExportTargets = map[string]bool{
	ReportExportTargetMarkdown: true,
	ReportExportTargetJSONAST:  true,
	ReportExportTargetHTML:     true,
}

func normalizeIDList(prefix string, ids []string) ([]string, error) {
	normalized := make([]string, 0, len(ids))
	seen := map[string]struct{}{}
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		if err := validateID(prefix, trimmed); err != nil {
			return nil, err
		}
		if _, ok := seen[trimmed]; ok {
			return nil, fmt.Errorf("%w: duplicate id", producterror.ErrInvalidInput)
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized, nil
}

func isApprovalProducer(producer ledger.Producer) bool {
	return producer.Type == "user" || producer.Type == "steering_chat"
}

func FormatIntentAllowed(value string) bool { return allowedReportFormatIntents[value] }
func ExportTargetAllowed(value string) bool { return allowedReportExportTargets[value] }
