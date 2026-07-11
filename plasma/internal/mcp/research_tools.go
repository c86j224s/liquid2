package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/researchproposal"
)

func (server *Server) callResearchOutline(ctx context.Context, call ToolCall) ToolResult {
	var input researchOutlineInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.enforceBoundMission(missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.enforceLegacyResearchRead(input.Legacy); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	var outline app.ResearchIDEOutline
	var err error
	if input.Legacy {
		legacy, ok := server.service.(legacyResearchReader)
		if !ok {
			return errorResult(call.Name, missionID, "validation", "legacy research reader is not available", false, nil)
		}
		outline, err = legacy.OutlineMissionLegacy(ctx, missionID)
	} else {
		outline, err = server.service.OutlineMission(ctx, missionID)
	}
	if err != nil {
		return errorFromErr(call.Name, missionID, err, nil)
	}
	return ToolResult{ToolName: call.Name, MissionID: missionID, Content: outline}
}

func (server *Server) callResearchList(ctx context.Context, call ToolCall) ToolResult {
	var input researchListInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.enforceBoundMission(missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.enforceLegacyResearchRead(input.Legacy); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	var page app.ResearchIDEPage
	var err error
	if input.Legacy {
		legacy, ok := server.service.(legacyResearchReader)
		if !ok {
			return errorResult(call.Name, missionID, "validation", "legacy research reader is not available", false, nil)
		}
		page, err = legacy.ListMissionObjectsLegacy(ctx, missionID, input.ObjectKind, input.Limit, input.Cursor)
	} else {
		page, err = server.service.ListMissionObjects(ctx, missionID, input.ObjectKind, input.Limit, input.Cursor)
	}
	if err != nil {
		return errorFromErr(call.Name, missionID, err, nil)
	}
	return ToolResult{ToolName: call.Name, MissionID: missionID, Content: page}
}

func (server *Server) callResearchRead(ctx context.Context, call ToolCall) ToolResult {
	var input researchReadInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.enforceBoundMission(missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.enforceLegacyResearchRead(input.Legacy); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	read, err := server.service.ReadMissionObject(ctx, app.ResearchIDEReadRequest{
		MissionID:  missionID,
		ObjectKind: input.ObjectKind,
		ObjectID:   input.ObjectID,
		Offset:     input.Offset,
		MaxBytes:   input.MaxBytes,
		Limit:      input.Limit,
		Cursor:     input.Cursor,
		Legacy:     input.Legacy,
	})
	if err != nil {
		return errorFromErr(call.Name, missionID, err, []string{input.ObjectID})
	}
	return ToolResult{ToolName: call.Name, MissionID: missionID, Content: read}
}

func (server *Server) callResearchGrep(ctx context.Context, call ToolCall) ToolResult {
	var input researchGrepInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.enforceBoundMission(missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.enforceLegacyResearchRead(input.Legacy); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	var result app.ResearchIDEGrepResult
	var err error
	if input.Legacy {
		legacy, ok := server.service.(legacyResearchReader)
		if !ok {
			return errorResult(call.Name, missionID, "validation", "legacy research reader is not available", false, nil)
		}
		result, err = legacy.GrepMissionObjectsLegacy(ctx, missionID, input.Query, input.Limit, input.Cursor)
	} else {
		result, err = server.service.GrepMissionObjects(ctx, missionID, input.Query, input.Limit, input.Cursor)
	}
	if err != nil {
		return errorFromErr(call.Name, missionID, err, nil)
	}
	return ToolResult{ToolName: call.Name, MissionID: missionID, Content: result}
}

func (server *Server) callResearchReferences(ctx context.Context, call ToolCall) ToolResult {
	var input researchReferencesInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.enforceBoundMission(missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.enforceLegacyResearchRead(input.Legacy); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	var refs app.ResearchIDEReferences
	var err error
	if input.Legacy {
		legacy, ok := server.service.(legacyResearchReader)
		if !ok {
			return errorResult(call.Name, missionID, "validation", "legacy research reader is not available", false, nil)
		}
		refs, err = legacy.ListObjectReferencesLegacy(ctx, missionID, input.ObjectKind, input.ObjectID, input.Limit, input.Cursor)
	} else {
		refs, err = server.service.ListObjectReferences(ctx, missionID, input.ObjectKind, input.ObjectID, input.Limit, input.Cursor)
	}
	if err != nil {
		return errorFromErr(call.Name, missionID, err, []string{input.ObjectID})
	}
	return ToolResult{ToolName: call.Name, MissionID: missionID, Content: refs}
}

func (server *Server) enforceLegacyResearchRead(legacy bool) error {
	if legacy && !server.legacyResearchLoop {
		return fmt.Errorf("%w: legacy research reads require legacy research loop mode", app.ErrInvalidInput)
	}
	return nil
}

func (server *Server) callEvidencePropose(ctx context.Context, call ToolCall) ToolResult {
	var input evidenceProposeInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, producer, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := validateEvidenceProposeInput(input); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	proposalWrite := researchproposal.BuildProposalSubmitted(researchproposal.ProposalSubmittedRequest{
		ProposalID:                 input.ProposalID,
		EventID:                    input.ProposalEventID,
		MissionID:                  common.MissionID,
		Title:                      input.ProposalTitle,
		ObjectRefs:                 []app.ObjectRef{{ObjectKind: app.EvidenceRecordObjectKind, ObjectID: input.EvidenceID}},
		Producer:                   producer,
		IncludeObjectRefsInPayload: true,
	})
	result, err := server.service.CreateEvidenceProposal(ctx, app.CreateEvidenceProposalRequest{
		EvidenceEvent: researchproposal.BuildEvidenceProposedAppendRequest(researchproposal.EvidenceProposedEventRequest{
			EventID:    input.EventID,
			MissionID:  common.MissionID,
			EvidenceID: input.EvidenceID,
			ProposalID: input.ProposalID,
			Producer:   producer,
		}),
		Evidence: app.CreateEvidenceRecordRequest{
			EvidenceID:     input.EvidenceID,
			MissionID:      common.MissionID,
			State:          "proposed",
			Summary:        input.Summary,
			EvidenceType:   input.EvidenceType,
			SnapshotRefs:   input.SnapshotRefs,
			Confidence:     input.Confidence,
			Producer:       producer,
			CreatedEventID: input.EventID,
		},
		ProposalEvent: proposalWrite.Event,
		Proposal:      proposalWrite.Bundle,
	})
	if err != nil {
		return errorFromErr(call.Name, common.MissionID, err, []string{input.EvidenceID, input.ProposalID})
	}
	return proposalToolResult(
		call.Name,
		common.MissionID,
		result.Proposal.ProposalID,
		[]string{result.EvidenceEvent.EventID, result.ProposalEvent.EventID},
		result.Proposal.ObjectRefs,
	)
}

func (server *Server) callQuestionsPropose(ctx context.Context, call ToolCall) ToolResult {
	var input questionsProposeInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, producer, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := validateQuestionsProposeInput(input); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	proposalWrite := researchproposal.BuildProposalSubmitted(researchproposal.ProposalSubmittedRequest{
		ProposalID:                 input.ProposalID,
		EventID:                    input.ProposalEventID,
		MissionID:                  common.MissionID,
		Title:                      input.ProposalTitle,
		ObjectRefs:                 []app.ObjectRef{{ObjectKind: app.QuestionRecordObjectKind, ObjectID: input.QuestionID}},
		Producer:                   producer,
		IncludeObjectRefsInPayload: true,
	})
	result, err := server.service.CreateQuestionProposal(ctx, app.CreateQuestionProposalRequest{
		QuestionEvent: researchproposal.BuildQuestionProposedAppendRequest(researchproposal.QuestionProposedEventRequest{
			EventID:    input.EventID,
			MissionID:  common.MissionID,
			QuestionID: input.QuestionID,
			ProposalID: input.ProposalID,
			Producer:   producer,
		}),
		Question: app.CreateQuestionRecordRequest{
			QuestionID:         input.QuestionID,
			MissionID:          common.MissionID,
			State:              "open",
			Text:               input.Text,
			Priority:           input.Priority,
			Blocking:           input.Blocking,
			RelatedEvidenceIDs: input.RelatedEvidenceIDs,
			RelatedClaimIDs:    input.RelatedClaimIDs,
			CreatedEventID:     input.EventID,
		},
		ProposalEvent: proposalWrite.Event,
		Proposal:      proposalWrite.Bundle,
	})
	if err != nil {
		return errorFromErr(call.Name, common.MissionID, err, []string{input.QuestionID, input.ProposalID})
	}
	return proposalToolResult(
		call.Name,
		common.MissionID,
		result.Proposal.ProposalID,
		[]string{result.QuestionEvent.EventID, result.ProposalEvent.EventID},
		result.Proposal.ObjectRefs,
	)
}

func (server *Server) callClaimsPropose(ctx context.Context, call ToolCall) ToolResult {
	var input claimsProposeInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, producer, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := validateClaimsProposeInput(input); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	proposalWrite := researchproposal.BuildProposalSubmitted(researchproposal.ProposalSubmittedRequest{
		ProposalID:                 input.ProposalID,
		EventID:                    input.ProposalEventID,
		MissionID:                  common.MissionID,
		Title:                      input.ProposalTitle,
		ObjectRefs:                 []app.ObjectRef{{ObjectKind: app.ClaimRecordObjectKind, ObjectID: input.ClaimID}},
		Producer:                   producer,
		IncludeObjectRefsInPayload: true,
	})
	result, err := server.service.CreateClaimProposal(ctx, app.CreateClaimProposalRequest{
		ClaimEvent: researchproposal.BuildClaimProposedAppendRequest(researchproposal.ClaimProposedEventRequest{
			EventID:    input.EventID,
			MissionID:  common.MissionID,
			ClaimID:    input.ClaimID,
			ProposalID: input.ProposalID,
			Producer:   producer,
		}),
		Claim: app.CreateClaimRecordRequest{
			ClaimID:               input.ClaimID,
			MissionID:             common.MissionID,
			State:                 "proposed",
			Text:                  input.Text,
			ClaimType:             input.ClaimType,
			SupportingEvidenceIDs: input.SupportingEvidenceIDs,
			OpposingEvidenceIDs:   input.OpposingEvidenceIDs,
			DependsOnQuestionIDs:  input.DependsOnQuestionIDs,
			UserAssertionEventID:  input.UserAssertionEventID,
			Confidence:            input.Confidence,
			Approval:              app.Approval{State: "pending", Required: true},
			CreatedEventID:        input.EventID,
		},
		ProposalEvent: proposalWrite.Event,
		Proposal:      proposalWrite.Bundle,
	})
	if err != nil {
		return errorFromErr(call.Name, common.MissionID, err, []string{input.ClaimID, input.ProposalID})
	}
	return proposalToolResult(
		call.Name,
		common.MissionID,
		result.Proposal.ProposalID,
		[]string{result.ClaimEvent.EventID, result.ProposalEvent.EventID},
		result.Proposal.ObjectRefs,
	)
}

func (server *Server) callClaimConfidence(ctx context.Context, call ToolCall) ToolResult {
	var input claimConfidenceInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, producer, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := validateClaimConfidenceInput(input); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	event, err := server.service.UpdateClaimConfidence(ctx, app.UpdateClaimConfidenceRequest{
		EventID:          input.EventID,
		MissionID:        common.MissionID,
		ClaimID:          input.ClaimID,
		Confidence:       input.Confidence,
		BasisEvidenceIDs: input.BasisEvidenceIDs,
		Origin:           "agent",
		Producer:         producer,
		CausationEventID: input.CausationEventID,
		CorrelationID:    input.CorrelationID,
	})
	if err != nil {
		return errorFromErr(call.Name, common.MissionID, err, []string{input.ClaimID})
	}
	return ToolResult{
		ToolName:        call.Name,
		MissionID:       common.MissionID,
		CreatedEventIDs: []string{event.EventID},
		Content: map[string]any{
			"event":      event,
			"claim_id":   input.ClaimID,
			"confidence": input.Confidence,
		},
	}
}

func (server *Server) callProposalsSubmit(ctx context.Context, call ToolCall) ToolResult {
	var input proposalsSubmitInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	common, producer, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	if err := validateProposalsSubmitInput(input); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	proposalWrite := researchproposal.BuildProposalSubmitted(researchproposal.ProposalSubmittedRequest{
		ProposalID:                 input.ProposalID,
		EventID:                    input.EventID,
		MissionID:                  common.MissionID,
		Title:                      input.Title,
		ObjectRefs:                 input.ObjectRefs,
		Producer:                   producer,
		IncludeObjectRefsInPayload: true,
	})
	result, err := server.service.SubmitProposal(ctx, app.SubmitProposalRequest{
		ProposalEvent: proposalWrite.Event,
		Proposal:      proposalWrite.Bundle,
	})
	if err != nil {
		return errorFromErr(call.Name, common.MissionID, err, []string{input.ProposalID})
	}
	return proposalToolResult(call.Name, common.MissionID, result.Proposal.ProposalID, []string{result.ProposalEvent.EventID}, result.Proposal.ObjectRefs)
}

func proposalToolResult(toolName, missionID, proposalID string, eventIDs []string, refs []app.ObjectRef) ToolResult {
	return ToolResult{
		ToolName:             toolName,
		MissionID:            missionID,
		CreatedEventIDs:      eventIDs,
		ProposalID:           proposalID,
		CreatedRecords:       refs,
		RequiresUserApproval: true,
	}
}

func approvalRequiredResult(toolName, missionID, message string, related []string) ToolResult {
	result := errorResult(toolName, missionID, "approval_required", message, false, related)
	result.RequiresUserApproval = true
	return result
}

func isUserApprovalProducer(producer app.Producer) bool {
	producerType := strings.TrimSpace(producer.Type)
	return producerType == "user" || producerType == "steering_chat"
}

func normalizeMutatingInput(input commonMutatingInput) (commonMutatingInput, app.Producer, error) {
	input.MissionID = strings.TrimSpace(input.MissionID)
	input.SessionID = strings.TrimSpace(input.SessionID)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if err := validateID("mis_", input.MissionID); err != nil {
		return input, app.Producer{}, err
	}
	if err := validateID("ses_", input.SessionID); err != nil {
		return input, app.Producer{}, err
	}
	if input.IdempotencyKey == "" {
		return input, app.Producer{}, fmt.Errorf("%w: idempotency_key is required", app.ErrInvalidInput)
	}
	producer := app.Producer{
		Type: strings.TrimSpace(input.Producer.Type),
		ID:   strings.TrimSpace(input.Producer.ID),
	}
	if producer.Type == "" || producer.ID == "" {
		return input, app.Producer{}, fmt.Errorf("%w: producer type and id are required", app.ErrInvalidInput)
	}
	if producer.Type != "agent_session" || producer.ID != input.SessionID {
		return input, app.Producer{}, fmt.Errorf("%w: tool producer must be agent_session matching session_id", app.ErrInvalidInput)
	}
	return input, producer, nil
}
