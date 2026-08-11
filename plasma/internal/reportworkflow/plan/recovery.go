package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

type planPayload struct {
	PendingEventID               string          `json:"pending_event_id"`
	ReportMode                   string          `json:"report_mode"`
	ArtifactID                   string          `json:"artifact_id"`
	AgentSessionID               string          `json:"agent_session_id"`
	PreviousAgentSessionID       string          `json:"previous_agent_session_id"`
	ToolSessionID                string          `json:"tool_session_id"`
	ReportSessionPolicy          string          `json:"report_session_policy"`
	ReportSessionPolicySelection string          `json:"report_session_policy_selection"`
	SessionChainKind             string          `json:"session_chain_kind"`
	PreReportResearchSessionID   string          `json:"pre_report_research_session_id"`
	ReportPlanSessionID          string          `json:"report_plan_session_id"`
	ForkSourceSessionID          string          `json:"fork_source_agent_session_id"`
	Plan                         json.RawMessage `json:"plan"`
}

type recoveredLineage struct {
	reportSessionPolicy          string
	reportSessionPolicySelection string
	sessionChainKind             string
	preReportResearchSessionID   string
	forkSourceSessionID          string
}

// RecoverMarkdown는 같은 retry lineage에 이미 기록된 canonical planned plan을 복원한다.
func (runner Runner) RecoverMarkdown(ctx context.Context, missionID string, pendingEventID string) (Output, bool, error) {
	events, err := runner.Service.ListEvents(ctx, missionID)
	if err != nil {
		return Output{}, false, err
	}
	lineage, err := RecoveryLineage(events, pendingEventID)
	if err != nil {
		return Output{}, false, err
	}
	var recovered Output
	for _, attemptID := range lineage {
		for _, event := range events {
			if event.EventType != "report.plan.created" {
				continue
			}
			applied, err := applyRecoveredPlan(attemptID, event, &recovered)
			if err != nil {
				return Output{}, false, err
			}
			if applied {
				recovered.Event = event
			}
		}
	}
	return recovered, recovered.ReportPlanSessionID != "", nil
}

func applyRecoveredPlan(pendingEventID string, event ledger.Event, recovered *Output) (bool, error) {
	var payload planPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return false, nil
	}
	if strings.TrimSpace(payload.PendingEventID) != pendingEventID || strings.TrimSpace(payload.ReportMode) != reportexecution.ModePlanned {
		return false, nil
	}
	if recovered.ReportPlanSessionID != "" {
		return false, fmt.Errorf("%w: multiple recovered planned report plans match one pending event", producterror.ErrConflict)
	}
	if len(payload.Plan) == 0 || string(payload.Plan) == "null" {
		return false, fmt.Errorf("%w: recovered planned report plan is missing", producterror.ErrConflict)
	}
	var rawPlan reporting.ReportPlan
	if err := json.Unmarshal(payload.Plan, &rawPlan); err != nil {
		return false, fmt.Errorf("%w: invalid recovered planned report plan: %v", producterror.ErrConflict, err)
	}
	normalized, err := reporting.NormalizeReportPlan(rawPlan)
	if err != nil {
		return false, fmt.Errorf("%w: invalid recovered planned report plan: %v", producterror.ErrConflict, err)
	}
	artifactID := strings.TrimSpace(payload.ArtifactID)
	reportPlanSessionID := firstNonEmpty(payload.ReportPlanSessionID, payload.AgentSessionID)
	planToolSessionID := strings.TrimSpace(payload.ToolSessionID)
	if artifactID == "" || reportPlanSessionID == "" || planToolSessionID == "" {
		return false, fmt.Errorf("%w: recovered planned report plan is incomplete", producterror.ErrConflict)
	}
	lineage, err := validateRecoveredLineage(payload, reportPlanSessionID)
	if err != nil {
		return false, err
	}
	recovered.Plan = normalized
	recovered.ArtifactID = artifactID
	recovered.PlanToolSessionID = planToolSessionID
	recovered.ReportPlanSessionID = reportPlanSessionID
	recovered.ReportSessionPolicy = lineage.reportSessionPolicy
	recovered.ReportSessionPolicySelection = lineage.reportSessionPolicySelection
	recovered.SessionChainKind = lineage.sessionChainKind
	recovered.PreReportResearchSessionID = lineage.preReportResearchSessionID
	recovered.ForkSourceSessionID = lineage.forkSourceSessionID
	recovered.Recovered = true
	return true, nil
}

func validateRecoveredLineage(payload planPayload, reportPlanSessionID string) (recoveredLineage, error) {
	agentSessionID := strings.TrimSpace(payload.AgentSessionID)
	storedPlanSessionID := strings.TrimSpace(payload.ReportPlanSessionID)
	if agentSessionID != "" && storedPlanSessionID != "" && agentSessionID != storedPlanSessionID {
		return recoveredLineage{}, fmt.Errorf("%w: recovered planned report plan has conflicting session ids", producterror.ErrConflict)
	}
	policy := reportexecution.SessionPolicySameSession
	if strings.TrimSpace(payload.ReportSessionPolicy) != "" {
		normalized, err := reportexecution.NormalizeSessionPolicy(payload.ReportSessionPolicy)
		if err != nil {
			return recoveredLineage{}, fmt.Errorf("%w: recovered planned report plan has invalid session policy", producterror.ErrConflict)
		}
		policy = normalized
	}
	selection := strings.TrimSpace(payload.ReportSessionPolicySelection)
	chainKind := strings.TrimSpace(payload.SessionChainKind)
	previousSessionID := strings.TrimSpace(payload.PreviousAgentSessionID)
	preReportSessionID := strings.TrimSpace(payload.PreReportResearchSessionID)
	forkSourceSessionID := strings.TrimSpace(payload.ForkSourceSessionID)
	switch policy {
	case reportexecution.SessionPolicyFreshSession:
		if selection != "" && selection != reportexecution.SessionPolicySelectionAutoFreshSession {
			return recoveredLineage{}, fmt.Errorf("%w: recovered fresh planned report has invalid session policy selection", producterror.ErrConflict)
		}
		if chainKind != "" && chainKind != "fresh_session_report" {
			return recoveredLineage{}, fmt.Errorf("%w: recovered fresh planned report has invalid session chain kind", producterror.ErrConflict)
		}
		if previousSessionID != "" || forkSourceSessionID != "" {
			return recoveredLineage{}, fmt.Errorf("%w: recovered fresh planned report has conflicting lineage", producterror.ErrConflict)
		}
	case reportexecution.SessionPolicyIsolatedFork:
		if selection != "" && selection != reportexecution.SessionPolicySelectionAutoIsolatedFork && selection != reportexecution.SessionPolicySelectionExplicitIsolatedFork {
			return recoveredLineage{}, fmt.Errorf("%w: recovered isolated planned report has invalid session policy selection", producterror.ErrConflict)
		}
		if chainKind != "" && chainKind != "isolated_fork_report" {
			return recoveredLineage{}, fmt.Errorf("%w: recovered isolated planned report has invalid session chain kind", producterror.ErrConflict)
		}
		if preReportSessionID == "" || forkSourceSessionID == "" || preReportSessionID != forkSourceSessionID {
			return recoveredLineage{}, fmt.Errorf("%w: recovered isolated planned report has conflicting fork lineage", producterror.ErrConflict)
		}
		if previousSessionID != "" && previousSessionID != reportPlanSessionID {
			return recoveredLineage{}, fmt.Errorf("%w: recovered isolated planned report has conflicting previous session", producterror.ErrConflict)
		}
	case reportexecution.SessionPolicySameSession:
		if selection != "" && !sameSessionSelection(selection) {
			return recoveredLineage{}, fmt.Errorf("%w: recovered same-session planned report has invalid session policy selection", producterror.ErrConflict)
		}
		if selection == reportexecution.SessionPolicySelectionAutoSameSessionNoSession && (previousSessionID != "" || preReportSessionID != "") {
			return recoveredLineage{}, fmt.Errorf("%w: recovered same-session planned report unexpectedly has a pre-report session", producterror.ErrConflict)
		}
		if chainKind != "" && chainKind != "same_session_report" {
			return recoveredLineage{}, fmt.Errorf("%w: recovered same-session planned report has invalid session chain kind", producterror.ErrConflict)
		}
		if forkSourceSessionID != "" {
			return recoveredLineage{}, fmt.Errorf("%w: recovered same-session planned report has conflicting fork source", producterror.ErrConflict)
		}
		if previousSessionID != "" && previousSessionID != reportPlanSessionID {
			return recoveredLineage{}, fmt.Errorf("%w: recovered same-session planned report has conflicting previous session", producterror.ErrConflict)
		}
		if preReportSessionID != "" && preReportSessionID != reportPlanSessionID {
			return recoveredLineage{}, fmt.Errorf("%w: recovered same-session planned report has conflicting pre-report session", producterror.ErrConflict)
		}
	default:
		return recoveredLineage{}, fmt.Errorf("%w: recovered planned report plan has invalid session policy", producterror.ErrConflict)
	}
	return recoveredLineage{
		reportSessionPolicy:          policy,
		reportSessionPolicySelection: selection,
		sessionChainKind:             firstNonEmpty(chainKind, chainKindForPolicy(policy)),
		preReportResearchSessionID:   firstNonEmpty(preReportSessionID, previousSessionID),
		forkSourceSessionID:          forkSourceSessionID,
	}, nil
}

func sameSessionSelection(selection string) bool {
	switch strings.TrimSpace(selection) {
	case reportexecution.SessionPolicySelectionExplicitSameSession,
		reportexecution.SessionPolicySelectionAutoSameSessionNoSession,
		reportexecution.SessionPolicySelectionAutoSameSessionNoForker,
		reportexecution.SessionPolicySelectionAutoSameSessionForkFailed:
		return true
	default:
		return false
	}
}
