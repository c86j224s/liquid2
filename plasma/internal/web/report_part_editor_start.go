package web

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func (server *Server) loadCurrentPartEditStart(ctx context.Context, req reportPartEditorRequest, expectedProviderSessionID string) (reporting.PartEditBinding, bool, error) {
	contract, err := server.partEditStartContract(ctx, req, expectedProviderSessionID)
	if err != nil {
		return reporting.PartEditBinding{}, false, err
	}
	return reporting.LoadCurrentPartEditStart(ctx, server.service, contract)
}

func (server *Server) currentPartEditStart(ctx context.Context, req reportPartEditorRequest, expectedProviderSessionID string) (reporting.PartEditBinding, bool, error) {
	return server.loadCurrentPartEditStart(ctx, req, expectedProviderSessionID)
}

func (server *Server) partEditStartContract(ctx context.Context, req reportPartEditorRequest, expectedProviderSessionID string) (reporting.PartEditStartContract, error) {
	sourcePartEventID, err := server.reportPartCreatedEventID(ctx, req.missionID, req.planEventID, req.partIndex+1, req.source.ArtifactID)
	if err != nil {
		return reporting.PartEditStartContract{}, err
	}
	mapHash := ""
	if strings.TrimSpace(req.requirementMapEvent.EventID) != "" {
		mapHash, _, err = reporting.ReportRequirementMapHash(req.requirementMap)
		if err != nil {
			return reporting.PartEditStartContract{}, err
		}
	}
	return reporting.PartEditStartContract{
		MissionID: req.missionID, CurrentPendingEventID: req.pendingEventID, PlanEventID: req.planEventID,
		SourcePartEventID: sourcePartEventID, SourceArtifactID: req.source.ArtifactID, PartIndex: req.partIndex + 1,
		IdempotencyKey:        fmt.Sprintf("report-part-edit:%s:%s:%d", req.pendingEventID, req.planEventID, req.partIndex+1),
		RequirementMapEventID: strings.TrimSpace(req.requirementMapEvent.EventID),
		RequirementMapHash:    mapHash,
		AgentExecutor:         req.executorName, AgentModel: req.agentModel, AgentReasoningEffort: req.agentReasoningEffort,
		AgentSelectionSource: req.agentSelectionSource, MCPMode: req.mcpMode,
		ReportSessionPolicy: req.reportSessionPolicy, ReportSessionPolicySelection: req.reportSessionPolicySelection,
		GenerationGuidanceProfile: req.generationGuidanceProfile, GenerationGuidanceSHA256: req.generationGuidanceSHA256,
		SessionChainKind: req.sessionChainKind, ReportPlanSessionID: req.reportPlanSessionID,
		ForkSourceAgentSessionID:   req.forkSourceAgentSessionID,
		ExpectedProviderSessionID:  expectedProviderSessionID,
		ExcludedProviderSessionIDs: []string{req.reportPlanSessionID},
	}, nil
}

func (server *Server) reportPartCreatedEventID(ctx context.Context, missionID string, planEventID string, partIndex int, artifactID string) (string, error) {
	events, err := server.service.ListEvents(ctx, missionID)
	if err != nil {
		return "", err
	}
	found := ""
	for _, event := range events {
		if event.EventType != "report.part.created" {
			continue
		}
		var payload struct {
			PlanEventID string `json:"plan_event_id"`
			ArtifactID  string `json:"artifact_id"`
			PartIndex   int    `json:"part_index"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if strings.TrimSpace(payload.PlanEventID) != planEventID || strings.TrimSpace(payload.ArtifactID) != artifactID || payload.PartIndex != partIndex {
			continue
		}
		if found != "" {
			return "", fmt.Errorf("%w: source Part event is duplicated", app.ErrConflict)
		}
		found = event.EventID
	}
	if found == "" {
		return "", fmt.Errorf("%w: source Part event is missing", app.ErrConflict)
	}
	return found, nil
}
