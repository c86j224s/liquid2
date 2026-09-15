package web

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportpatch"
	"strings"
)

func selectReportPatchSession(ctx context.Context, executor AgentExecutor, sourceSessionID string, requestedPolicy string) (reportPatchSessionSelection, error) {
	return SelectReportPatchSession(ctx, executor, sourceSessionID, requestedPolicy)
}

// SelectReportPatchSession는 patch 요청에 사용할 report session과 fork 출처를 선택한다.
func SelectReportPatchSession(ctx context.Context, executor AgentExecutor, sourceSessionID string, requestedPolicy string) (ReportPatchSessionSelection, error) {
	return reportpatch.SelectSession(ctx, executor, sourceSessionID, requestedPolicy)
}

func reportPatchMCPTools() []string {
	return reportpatch.MCPTools()
}

func normalizeReportMode(mode string) (string, error) {
	return reportexecution.NormalizeMode(mode)
}

func normalizeReportExecutionStrategy(strategy string, reportMode string) (string, error) {
	strategy = strings.TrimSpace(strings.ToLower(strategy))
	if strategy == "" || strategy == reportExecutionStrategySerial {
		return reportExecutionStrategySerial, nil
	}
	if strategy != reportExecutionStrategySectionFanout {
		return "", fmt.Errorf("%w: unsupported report execution strategy", producterror.ErrInvalidInput)
	}
	if reportMode != reportModeLongForm {
		return "", fmt.Errorf("%w: section fanout is only supported for long-form reports", producterror.ErrInvalidInput)
	}
	return strategy, nil
}

func storedReportExecutionStrategy(strategy string) string {
	if strings.TrimSpace(strings.ToLower(strategy)) == reportExecutionStrategySerial {
		return ""
	}
	return strings.TrimSpace(strings.ToLower(strategy))
}

func reportEventString(event ledger.Event, key string) string {
	var payload map[string]any
	if json.Unmarshal(event.Payload, &payload) != nil {
		return ""
	}
	value, _ := payload[key].(string)
	return strings.TrimSpace(value)
}

func normalizeReportSessionPolicy(policy string) (string, error) {
	return reportexecution.NormalizeSessionPolicy(policy)
}

func (server *Server) selectReportSessionPolicy(ctx context.Context, missionID string, executorName string, reportMode string, requestedPolicy string, executor AgentExecutor) (string, string, error) {
	requestedPolicy = strings.TrimSpace(requestedPolicy)
	if requestedPolicy == "" {
		return reportexecution.SelectSessionPolicy(reportexecution.SessionPolicySelectionInput{
			ReportMode: reportMode,
		})
	}
	requestedCanonical, err := reportexecution.NormalizeSessionPolicy(requestedPolicy)
	if err != nil {
		return "", "", err
	}
	if requestedCanonical != reportexecution.SessionPolicyIsolatedFork {
		return reportexecution.SelectSessionPolicy(reportexecution.SessionPolicySelectionInput{
			RequestedPolicy: requestedPolicy,
			ReportMode:      reportMode,
		})
	}
	_, canFork := executor.(AgentSessionForker)
	_, canCheckFork := executor.(AgentSessionForkReadiness)
	preReportSessionID := ""
	forkReady := false
	if canFork {
		preReportSessionID = strings.TrimSpace(server.latestAgentSessionID(ctx, missionID, executorName))
		forkReady = canCheckFork && AgentSessionForkReady(ctx, executor, preReportSessionID)
	}
	return reportexecution.SelectSessionPolicy(reportexecution.SessionPolicySelectionInput{
		RequestedPolicy:             requestedPolicy,
		ReportMode:                  reportMode,
		CanForkSession:              canFork,
		HasPreReportResearchSession: preReportSessionID != "",
		ForkReady:                   forkReady,
	})
}

func (server *Server) validateReportSessionPolicy(ctx context.Context, missionID string, executorName string, reportMode string, policy string, executor AgentExecutor, requireReady bool) error {
	if executor == nil {
		return fmt.Errorf("%w: report generation requires an agent executor", producterror.ErrInvalidInput)
	}
	_, canFork := executor.(AgentSessionForker)
	_, canCheckFork := executor.(AgentSessionForkReadiness)
	preReportSessionID := strings.TrimSpace(server.latestAgentSessionID(ctx, missionID, executorName))
	return reportexecution.ValidateSessionPolicy(policy, reportMode, canFork, !requireReady || preReportSessionID != "", !requireReady || (canCheckFork && AgentSessionForkReady(ctx, executor, preReportSessionID)))
}

func reportModeLabel(mode string) string {
	return reportexecution.ModeLabel(mode)
}

func normalizeReportRigorProfile(level string) (reportRigorProfile, error) {
	normalized := strings.TrimSpace(level)
	if normalized == "" {
		normalized = defaultReportRigorLevel
	}
	switch normalized {
	case "loose":
		normalized = "exploratory"
	case "normal":
		normalized = "balanced"
	case "rigorous":
		normalized = "strict"
	}
	profile, ok := reportRigorProfiles[normalized]
	if !ok {
		return reportRigorProfile{}, fmt.Errorf("%w: unsupported report rigor level", producterror.ErrInvalidInput)
	}
	return profile, nil
}
