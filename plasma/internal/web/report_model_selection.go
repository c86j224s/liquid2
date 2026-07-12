package web

import (
	"context"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func (server *Server) resolveReportModelSelection(ctx context.Context, missionID string, req reportDraftRequest) (reporting.ModelSelection, error) {
	var status agentExecutorStatus
	for _, candidate := range server.agentStatuses() {
		if candidate.Name == req.AgentExecutor {
			status = candidate
			break
		}
	}
	selection, err := reporting.ResolveModelSelection(reporting.ModelSelectionInput{
		Executor:                 req.AgentExecutor,
		RequestModel:             req.AgentModel,
		RequestReasoningEffort:   req.AgentReasoningEffort,
		SessionModel:             server.latestAgentSessionModel(ctx, missionID, req.AgentExecutor),
		SessionReasoningEffort:   server.latestAgentReasoningEffort(ctx, missionID, req.AgentExecutor),
		ProviderModel:            status.DefaultModel,
		ProviderReasoningEffort:  status.DefaultReasoningEffort,
		ReasoningEffortSupported: status.ReasoningEffortSupported,
	})
	if err != nil {
		return reporting.ModelSelection{}, fmt.Errorf("%w: %v", app.ErrInvalidInput, err)
	}
	return selection, nil
}
