package research

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"github.com/c86j224s/liquid2/plasma/internal/researchproposal"
)

func (handler *Handler) CallEvidencePropose(ctx context.Context, call wire.ToolCall) wire.ToolResult {
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
	if result, ok := handler.mutationReadyResult(call.Name, common.MissionID); !ok {
		return result
	}
	proposalWrite := researchproposal.BuildProposalSubmitted(researchproposal.ProposalSubmittedRequest{
		ProposalID: input.ProposalID, EventID: input.ProposalEventID, MissionID: common.MissionID,
		Title: input.ProposalTitle, ObjectRefs: []app.ObjectRef{{ObjectKind: app.EvidenceRecordObjectKind, ObjectID: input.EvidenceID}},
		Producer: producer, IncludeObjectRefsInPayload: true,
	})
	result, err := handler.proposalWriter.CreateEvidenceProposal(ctx, app.CreateEvidenceProposalRequest{
		EvidenceEvent: researchproposal.BuildEvidenceProposedAppendRequest(researchproposal.EvidenceProposedEventRequest{
			EventID: input.EventID, MissionID: common.MissionID, EvidenceID: input.EvidenceID, ProposalID: input.ProposalID, Producer: producer,
		}),
		Evidence: app.CreateEvidenceRecordRequest{
			EvidenceID: input.EvidenceID, MissionID: common.MissionID, State: "proposed", Summary: input.Summary,
			EvidenceType: input.EvidenceType, SnapshotRefs: input.SnapshotRefs, Confidence: input.Confidence, Producer: producer, CreatedEventID: input.EventID,
		},
		ProposalEvent: proposalWrite.Event, Proposal: proposalWrite.Bundle,
	})
	if err != nil {
		return errorFromErr(call.Name, common.MissionID, err, []string{input.EvidenceID, input.ProposalID})
	}
	return proposalToolResult(call.Name, common.MissionID, result.Proposal.ProposalID, []string{result.EvidenceEvent.EventID, result.ProposalEvent.EventID}, result.Proposal.ObjectRefs)
}

func (handler *Handler) CallQuestionsPropose(ctx context.Context, call wire.ToolCall) wire.ToolResult {
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
	if result, ok := handler.mutationReadyResult(call.Name, common.MissionID); !ok {
		return result
	}
	proposalWrite := researchproposal.BuildProposalSubmitted(researchproposal.ProposalSubmittedRequest{
		ProposalID: input.ProposalID, EventID: input.ProposalEventID, MissionID: common.MissionID,
		Title: input.ProposalTitle, ObjectRefs: []app.ObjectRef{{ObjectKind: app.QuestionRecordObjectKind, ObjectID: input.QuestionID}},
		Producer: producer, IncludeObjectRefsInPayload: true,
	})
	result, err := handler.proposalWriter.CreateQuestionProposal(ctx, app.CreateQuestionProposalRequest{
		QuestionEvent: researchproposal.BuildQuestionProposedAppendRequest(researchproposal.QuestionProposedEventRequest{
			EventID: input.EventID, MissionID: common.MissionID, QuestionID: input.QuestionID, ProposalID: input.ProposalID, Producer: producer,
		}),
		Question: app.CreateQuestionRecordRequest{
			QuestionID: input.QuestionID, MissionID: common.MissionID, State: "open", Text: input.Text, Priority: input.Priority,
			Blocking: input.Blocking, RelatedEvidenceIDs: input.RelatedEvidenceIDs, RelatedClaimIDs: input.RelatedClaimIDs, CreatedEventID: input.EventID,
		},
		ProposalEvent: proposalWrite.Event, Proposal: proposalWrite.Bundle,
	})
	if err != nil {
		return errorFromErr(call.Name, common.MissionID, err, []string{input.QuestionID, input.ProposalID})
	}
	return proposalToolResult(call.Name, common.MissionID, result.Proposal.ProposalID, []string{result.QuestionEvent.EventID, result.ProposalEvent.EventID}, result.Proposal.ObjectRefs)
}

func (handler *Handler) CallClaimsPropose(ctx context.Context, call wire.ToolCall) wire.ToolResult {
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
	if result, ok := handler.mutationReadyResult(call.Name, common.MissionID); !ok {
		return result
	}
	proposalWrite := researchproposal.BuildProposalSubmitted(researchproposal.ProposalSubmittedRequest{
		ProposalID: input.ProposalID, EventID: input.ProposalEventID, MissionID: common.MissionID,
		Title: input.ProposalTitle, ObjectRefs: []app.ObjectRef{{ObjectKind: app.ClaimRecordObjectKind, ObjectID: input.ClaimID}},
		Producer: producer, IncludeObjectRefsInPayload: true,
	})
	result, err := handler.proposalWriter.CreateClaimProposal(ctx, app.CreateClaimProposalRequest{
		ClaimEvent: researchproposal.BuildClaimProposedAppendRequest(researchproposal.ClaimProposedEventRequest{
			EventID: input.EventID, MissionID: common.MissionID, ClaimID: input.ClaimID, ProposalID: input.ProposalID, Producer: producer,
		}),
		Claim: app.CreateClaimRecordRequest{
			ClaimID: input.ClaimID, MissionID: common.MissionID, State: "proposed", Text: input.Text, ClaimType: input.ClaimType,
			SupportingEvidenceIDs: input.SupportingEvidenceIDs, OpposingEvidenceIDs: input.OpposingEvidenceIDs,
			DependsOnQuestionIDs: input.DependsOnQuestionIDs, UserAssertionEventID: input.UserAssertionEventID,
			Confidence: input.Confidence, Approval: app.Approval{State: "pending", Required: true}, CreatedEventID: input.EventID,
		},
		ProposalEvent: proposalWrite.Event, Proposal: proposalWrite.Bundle,
	})
	if err != nil {
		return errorFromErr(call.Name, common.MissionID, err, []string{input.ClaimID, input.ProposalID})
	}
	return proposalToolResult(call.Name, common.MissionID, result.Proposal.ProposalID, []string{result.ClaimEvent.EventID, result.ProposalEvent.EventID}, result.Proposal.ObjectRefs)
}

func (handler *Handler) CallClaimConfidence(ctx context.Context, call wire.ToolCall) wire.ToolResult {
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
	if result, ok := handler.mutationReadyResult(call.Name, common.MissionID); !ok {
		return result
	}
	event, err := handler.proposalWriter.UpdateClaimConfidence(ctx, app.UpdateClaimConfidenceRequest{
		EventID: input.EventID, MissionID: common.MissionID, ClaimID: input.ClaimID, Confidence: input.Confidence,
		BasisEvidenceIDs: input.BasisEvidenceIDs, Origin: "agent", Producer: producer,
		CausationEventID: input.CausationEventID, CorrelationID: input.CorrelationID,
	})
	if err != nil {
		return errorFromErr(call.Name, common.MissionID, err, []string{input.ClaimID})
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: common.MissionID, CreatedEventIDs: []string{event.EventID}, Content: map[string]any{"event": event, "claim_id": input.ClaimID, "confidence": input.Confidence}}
}

func (handler *Handler) CallProposalsSubmit(ctx context.Context, call wire.ToolCall) wire.ToolResult {
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
	if result, ok := handler.mutationReadyResult(call.Name, common.MissionID); !ok {
		return result
	}
	proposalWrite := researchproposal.BuildProposalSubmitted(researchproposal.ProposalSubmittedRequest{
		ProposalID: input.ProposalID, EventID: input.EventID, MissionID: common.MissionID,
		Title: input.Title, ObjectRefs: input.ObjectRefs, Producer: producer, IncludeObjectRefsInPayload: true,
	})
	result, err := handler.proposalWriter.SubmitProposal(ctx, app.SubmitProposalRequest{ProposalEvent: proposalWrite.Event, Proposal: proposalWrite.Bundle})
	if err != nil {
		return errorFromErr(call.Name, common.MissionID, err, []string{input.ProposalID})
	}
	return proposalToolResult(call.Name, common.MissionID, result.Proposal.ProposalID, []string{result.ProposalEvent.EventID}, result.Proposal.ObjectRefs)
}

func proposalToolResult(toolName, missionID, proposalID string, eventIDs []string, refs []app.ObjectRef) wire.ToolResult {
	return wire.ToolResult{ToolName: toolName, MissionID: missionID, CreatedEventIDs: eventIDs, ProposalID: proposalID, CreatedRecords: refs, RequiresUserApproval: true}
}
