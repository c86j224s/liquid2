package reportpatch

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
)

// PatchSessionSelection is the session lineage selected for a report patch run.
//
// SessionID and PreviousAgentSessionID are the provider session passed to the
// patch agent. ForkSourceAgentSessionID is set only when a new isolated session
// was forked from an earlier report session. The policy fields are stable
// product metadata copied into pending/finalized report patch events.
type PatchSessionSelection struct {
	SessionID                    string
	PreviousAgentSessionID       string
	ForkSourceAgentSessionID     string
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	SessionChainKind             string
}

// SelectSession selects the provider session and lineage metadata for a report
// patch run. A blank requested policy means automatic selection; explicit
// isolated forks fail instead of falling back when the executor cannot fork.
func SelectSession(ctx context.Context, executor agentexec.AgentExecutor, sourceSessionID string, requestedPolicy string) (PatchSessionSelection, error) {
	sourceSessionID = strings.TrimSpace(sourceSessionID)
	if sourceSessionID == "" {
		return PatchSessionSelection{}, fmt.Errorf("%w: report patch requires a previous report session", producterror.ErrInvalidInput)
	}
	requestedPolicy = strings.TrimSpace(requestedPolicy)
	if requestedPolicy != "" {
		policy, err := reportexecution.NormalizeSessionPolicy(requestedPolicy)
		if err != nil {
			return PatchSessionSelection{}, err
		}
		switch policy {
		case reportexecution.SessionPolicySameSession:
			return PatchSessionSelection{
				SessionID:                    sourceSessionID,
				PreviousAgentSessionID:       sourceSessionID,
				ReportSessionPolicy:          reportexecution.SessionPolicySameSession,
				ReportSessionPolicySelection: reportexecution.SessionPolicySelectionExplicitSameSession,
				SessionChainKind:             "same_report_session_patch",
			}, nil
		case reportexecution.SessionPolicyFreshSession:
			return PatchSessionSelection{}, fmt.Errorf("%w: fresh report sessions are automatic-only", producterror.ErrInvalidInput)
		case reportexecution.SessionPolicyIsolatedFork:
			forker, ok := executor.(agentexec.AgentSessionForker)
			if !ok {
				return PatchSessionSelection{}, fmt.Errorf("%w: isolated report patch session requires a forkable executor", producterror.ErrInvalidInput)
			}
			if !agentexec.AgentSessionForkReady(ctx, executor, sourceSessionID) {
				return PatchSessionSelection{}, fmt.Errorf("%w: isolated report patch session is not ready for fork", producterror.ErrInvalidInput)
			}
			fork, err := forker.ForkSession(ctx, sourceSessionID)
			if err != nil {
				return PatchSessionSelection{}, fmt.Errorf("report patch session fork failed: %w", err)
			}
			forkSource := firstNonEmpty(fork.SourceSessionID, sourceSessionID)
			return PatchSessionSelection{
				SessionID:                    fork.SessionID,
				PreviousAgentSessionID:       fork.SessionID,
				ForkSourceAgentSessionID:     forkSource,
				ReportSessionPolicy:          reportexecution.SessionPolicyIsolatedFork,
				ReportSessionPolicySelection: reportexecution.SessionPolicySelectionExplicitIsolatedFork,
				SessionChainKind:             "isolated_fork_report_patch",
			}, nil
		default:
			return PatchSessionSelection{}, fmt.Errorf("%w: unsupported report session policy %q", producterror.ErrInvalidInput, policy)
		}
	}

	forker, canFork := executor.(agentexec.AgentSessionForker)
	if !canFork {
		return PatchSessionSelection{
			SessionID:                    sourceSessionID,
			PreviousAgentSessionID:       sourceSessionID,
			ReportSessionPolicy:          reportexecution.SessionPolicySameSession,
			ReportSessionPolicySelection: reportexecution.SessionPolicySelectionAutoSameSessionNoForker,
			SessionChainKind:             "same_report_session_patch",
		}, nil
	}
	if !agentexec.AgentSessionForkReady(ctx, executor, sourceSessionID) {
		return PatchSessionSelection{
			SessionID:                    sourceSessionID,
			PreviousAgentSessionID:       sourceSessionID,
			ReportSessionPolicy:          reportexecution.SessionPolicySameSession,
			ReportSessionPolicySelection: reportexecution.SessionPolicySelectionAutoSameSessionForkFailed,
			SessionChainKind:             "same_report_session_patch",
		}, nil
	}
	fork, err := forker.ForkSession(ctx, sourceSessionID)
	if err != nil {
		return PatchSessionSelection{
			SessionID:                    sourceSessionID,
			PreviousAgentSessionID:       sourceSessionID,
			ReportSessionPolicy:          reportexecution.SessionPolicySameSession,
			ReportSessionPolicySelection: reportexecution.SessionPolicySelectionAutoSameSessionForkFailed,
			SessionChainKind:             "same_report_session_patch",
		}, nil
	}
	forkSource := firstNonEmpty(fork.SourceSessionID, sourceSessionID)
	return PatchSessionSelection{
		SessionID:                    fork.SessionID,
		PreviousAgentSessionID:       fork.SessionID,
		ForkSourceAgentSessionID:     forkSource,
		ReportSessionPolicy:          reportexecution.SessionPolicyIsolatedFork,
		ReportSessionPolicySelection: reportexecution.SessionPolicySelectionAutoIsolatedFork,
		SessionChainKind:             "isolated_fork_report_patch",
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
