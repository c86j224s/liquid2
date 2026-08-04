package reporting

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

const (
	FinalEditReaderSourceSchema     = "plasma.final_edit.reader_source.v1"
	finalEditReaderSourceProducerID = "reporting_reader_assembly"
)

var finalEditReaderSourceProducer = app.Producer{Type: "system", ID: finalEditReaderSourceProducerID}

type finalEditReaderSourceIdentity struct {
	Schema          string   `json:"schema"`
	PlanEventID     string   `json:"plan_event_id"`
	PartArtifactIDs []string `json:"part_artifact_ids"`
}

// FinalEditReaderSourceArtifactID는 최종 편집의 reader source artifact ID를 선택한다.
func FinalEditReaderSourceArtifactID(planEventID string, partArtifactIDs []string) string {
	encoded, _ := json.Marshal(finalEditReaderSourceIdentity{
		Schema:          FinalEditReaderSourceSchema,
		PlanEventID:     strings.TrimSpace(planEventID),
		PartArtifactIDs: append([]string(nil), partArtifactIDs...),
	})
	sum := sha256.Sum256(encoded)
	return "art_" + hex.EncodeToString(sum[:])
}

func finalEditReaderSourceRequest(ctx context.Context, store FinalEditStageStore, events []app.LedgerEvent, binding FinalEditStageBinding) (app.CreateRawArtifactRequest, error) {
	assembly, err := finalEditPartAssemblyForBinding(ctx, store, events, binding)
	if err != nil {
		return app.CreateRawArtifactRequest{}, err
	}
	artifactID := FinalEditReaderSourceArtifactID(binding.PlanEventID, assembly.PartArtifactIDs)
	if binding.SourceArtifactID != artifactID {
		return app.CreateRawArtifactRequest{}, fmt.Errorf("%w: reader source artifact id differs from deterministic contract", app.ErrConflict)
	}
	return app.CreateRawArtifactRequest{
		ArtifactID: artifactID,
		MissionID:  binding.MissionID,
		MediaType:  "text/markdown; charset=utf-8",
		Filename:   binding.Filename,
		Producer:   finalEditReaderSourceProducer,
		Content:    []byte(assembly.Markdown),
	}, nil
}

func orderedLongFormPartArtifactsForFinalEdit(ctx context.Context, store FinalEditStageStore, events []app.LedgerEvent, binding FinalEditStageBinding) ([]app.RawArtifact, error) {
	acceptedPending, err := longFormPendingLineage(events, binding.PendingEventID)
	if err != nil {
		return nil, err
	}
	var planParts []finalEditPlannedPart
	var partEditEnabled bool
	planCount := 0
	for _, event := range events {
		if event.EventType != "report.plan.created" {
			continue
		}
		payload := eventPayload(event)
		if event.EventID != binding.PlanEventID {
			if acceptedPending[payloadString(payload, "pending_event_id")] {
				return nil, fmt.Errorf("%w: final edit plan lineage differs from binding", app.ErrConflict)
			}
			continue
		}
		if !acceptedPending[payloadString(payload, "pending_event_id")] || payloadString(payload, "report_mode") != ModeLongForm {
			return nil, fmt.Errorf("%w: final edit plan lineage differs from binding", app.ErrConflict)
		}
		planCount++
		partEditEnabled = payloadBool(payload, "part_edit_enabled")
		var planPayload struct {
			Plan struct {
				Parts []struct {
					Sections []json.RawMessage `json:"sections"`
				} `json:"parts"`
			} `json:"plan"`
		}
		if err := json.Unmarshal(event.Payload, &planPayload); err != nil {
			return nil, fmt.Errorf("%w: final edit plan payload is invalid", app.ErrConflict)
		}
		planParts = make([]finalEditPlannedPart, len(planPayload.Plan.Parts))
		for index, part := range planPayload.Plan.Parts {
			if len(part.Sections) == 0 {
				return nil, fmt.Errorf("%w: final edit plan sections are incomplete", app.ErrConflict)
			}
			planParts[index] = finalEditPlannedPart{SectionCount: len(part.Sections)}
		}
	}
	if planCount != 1 || len(planParts) < 1 {
		return nil, fmt.Errorf("%w: final edit plan parts are incomplete", app.ErrConflict)
	}
	if err := validateFinalEditReaderPlannedSections(ctx, store, events, binding, acceptedPending, planParts); err != nil {
		return nil, err
	}
	partCount := len(planParts)
	partIDs := make([]string, partCount)
	partEventIDs := make([]string, partCount)
	for _, event := range events {
		if event.EventType != "report.part.created" {
			continue
		}
		payload := eventPayload(event)
		if !acceptedPending[payloadString(payload, "pending_event_id")] {
			continue
		}
		if payloadString(payload, "plan_event_id") != binding.PlanEventID {
			return nil, fmt.Errorf("%w: final edit Part plan lineage differs from binding", app.ErrConflict)
		}
		index := jsonInt(payload["part_index"])
		if index < 1 || index > partCount || partIDs[index-1] != "" {
			return nil, fmt.Errorf("%w: duplicate or out-of-range final edit Part lineage", app.ErrConflict)
		}
		partIDs[index-1] = payloadString(payload, "artifact_id")
		partEventIDs[index-1] = event.EventID
	}
	parts := make([]app.RawArtifact, partCount)
	for index, artifactID := range partIDs {
		if artifactID == "" {
			return nil, fmt.Errorf("%w: final edit Part lineage is incomplete", app.ErrConflict)
		}
		if partEditEnabled {
			contract := partEditOutcomeContractFromStageBinding(binding, partEventIDs[index], artifactID, index+1)
			outcomes, err := validPartEditOutcomes(ctx, store, events, acceptedPending, contract)
			if err != nil {
				return nil, err
			}
			if len(outcomes) != 1 {
				return nil, fmt.Errorf("%w: final edit reader source requires exactly one valid reviewed Part edit", app.ErrConflict)
			}
			parts[index] = outcomes[0].Artifact
			continue
		}
		part, err := store.GetRawArtifact(ctx, artifactID)
		if err != nil {
			return nil, err
		}
		parts[index] = part
	}
	for _, part := range parts {
		if part.MissionID != binding.MissionID || part.MediaType != "text/markdown; charset=utf-8" || part.SHA256 != contentSHA256(part.Content) {
			return nil, fmt.Errorf("%w: final edit Part artifact is foreign or not Markdown", app.ErrConflict)
		}
	}
	return parts, nil
}

func partEditOutcomeContractFromStageBinding(binding FinalEditStageBinding, sourcePartEventID string, sourceArtifactID string, partIndex int) PartEditOutcomeContract {
	return PartEditOutcomeContract{
		MissionID: binding.MissionID, CurrentPendingEventID: binding.PendingEventID, PlanEventID: binding.PlanEventID,
		SourcePartEventID: sourcePartEventID, SourceArtifactID: sourceArtifactID, PartIndex: partIndex,
		AgentExecutor: binding.AgentExecutor, AgentModel: binding.AgentModel, AgentReasoningEffort: binding.AgentReasoningEffort,
		AgentSelectionSource: binding.AgentSelectionSource, MCPMode: binding.MCPMode,
		ReportSessionPolicy: binding.ReportSessionPolicy, ReportSessionPolicySelection: binding.ReportSessionPolicySelection,
		GenerationGuidanceProfile: binding.GenerationGuidanceProfile, GenerationGuidanceSHA256: binding.GenerationGuidanceSHA256,
		SessionChainKind: binding.SessionChainKind, ReportPlanSessionID: binding.ReportPlanSessionID,
		ExcludedProviderSessionIDs: []string{binding.ProviderSessionID},
	}
}
