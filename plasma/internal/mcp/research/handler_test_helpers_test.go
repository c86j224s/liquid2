package research

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
)

func callReadTool(handler *Handler, call wire.ToolCall) wire.ToolResult {
	switch {
	case strings.HasSuffix(call.Name, ".outline"):
		return handler.CallOutline(context.Background(), call)
	case strings.HasSuffix(call.Name, ".changes"):
		return handler.CallChanges(context.Background(), call)
	case strings.HasSuffix(call.Name, ".list"):
		return handler.CallList(context.Background(), call)
	case strings.HasSuffix(call.Name, ".read"):
		return handler.CallRead(context.Background(), call)
	case strings.HasSuffix(call.Name, ".grep"):
		return handler.CallGrep(context.Background(), call)
	default:
		return handler.CallReferences(context.Background(), call)
	}
}

func evidenceCall() wire.ToolCall {
	return mutationCall("plasma.evidence.propose", map[string]any{
		"evidence_id":       "evd_1",
		"event_id":          "evt_evidence",
		"proposal_id":       "prp_1",
		"proposal_event_id": "evt_proposal",
		"summary":           "Summary.",
		"evidence_type":     "quote",
		"snapshot_refs":     []map[string]any{{"snapshot_id": "src_1", "artifact_id": "art_1"}},
	})
}

func questionsCall() wire.ToolCall {
	return mutationCall("plasma.questions.propose", map[string]any{
		"question_id":       "qst_1",
		"event_id":          "evt_question",
		"proposal_id":       "prp_1",
		"proposal_event_id": "evt_proposal",
		"text":              "Question?",
	})
}

func claimsCall() wire.ToolCall {
	return mutationCall("plasma.claims.propose", map[string]any{
		"claim_id":                "clm_1",
		"event_id":                "evt_claim",
		"proposal_id":             "prp_1",
		"proposal_event_id":       "evt_proposal",
		"text":                    "Claim.",
		"supporting_evidence_ids": []string{"evd_1"},
	})
}

func confidenceCall() wire.ToolCall {
	return mutationCall("plasma.claim.confidence", map[string]any{
		"claim_id":   "clm_1",
		"event_id":   "evt_confidence",
		"confidence": map[string]any{"level": "medium", "rationale": "Basis."},
	})
}

func submitCall() wire.ToolCall {
	return mutationCall("plasma.proposals.submit", map[string]any{
		"proposal_id": "prp_1",
		"event_id":    "evt_submit",
		"object_refs": []map[string]any{{"object_kind": "evidence_record", "object_id": "evd_1"}},
	})
}

func mutationCall(name string, fields map[string]any) wire.ToolCall {
	fields["mission_id"] = "mis_1"
	fields["session_id"] = "ses_1"
	fields["idempotency_key"] = "idem_1"
	fields["producer"] = map[string]any{"type": "agent_session", "id": "ses_1"}
	encoded, err := json.Marshal(fields)
	if err != nil {
		panic(err)
	}
	return wire.ToolCall{Name: name, Arguments: encoded}
}

func mutationCallWithCommonField(call wire.ToolCall, field string, value any) wire.ToolCall {
	var fields map[string]any
	if err := json.Unmarshal(call.Arguments, &fields); err != nil {
		panic(err)
	}
	fields[field] = value
	encoded, err := json.Marshal(fields)
	if err != nil {
		panic(err)
	}
	return wire.ToolCall{Name: call.Name, Arguments: encoded}
}

type readerOnlyService struct{}

func (readerOnlyService) OutlineMission(context.Context, string) (app.ResearchIDEOutline, error) {
	return app.ResearchIDEOutline{}, nil
}

func (readerOnlyService) ListMissionChanges(context.Context, app.ResearchIDEChangesRequest) (app.ResearchIDEChanges, error) {
	return app.ResearchIDEChanges{}, nil
}

func (readerOnlyService) ListMissionObjects(context.Context, string, string, int, string) (app.ResearchIDEPage, error) {
	return app.ResearchIDEPage{}, nil
}

func (readerOnlyService) ReadMissionObject(context.Context, app.ResearchIDEReadRequest) (app.ResearchIDEObjectRead, error) {
	return app.ResearchIDEObjectRead{}, nil
}

func (readerOnlyService) GrepMissionObjects(context.Context, string, string, int, string) (app.ResearchIDEGrepResult, error) {
	return app.ResearchIDEGrepResult{}, nil
}

func (readerOnlyService) ListObjectReferences(context.Context, string, string, string, int, string) (app.ResearchIDEReferences, error) {
	return app.ResearchIDEReferences{}, nil
}

type countingProposalWriter struct {
	count int
}

func (writer *countingProposalWriter) CreateEvidenceProposal(context.Context, app.CreateEvidenceProposalRequest) (app.EvidenceProposalResult, error) {
	writer.count++
	return app.EvidenceProposalResult{}, nil
}

func (writer *countingProposalWriter) CreateQuestionProposal(context.Context, app.CreateQuestionProposalRequest) (app.QuestionProposalResult, error) {
	writer.count++
	return app.QuestionProposalResult{}, nil
}

func (writer *countingProposalWriter) CreateClaimProposal(context.Context, app.CreateClaimProposalRequest) (app.ClaimProposalResult, error) {
	writer.count++
	return app.ClaimProposalResult{}, nil
}

func (writer *countingProposalWriter) UpdateClaimConfidence(context.Context, app.UpdateClaimConfidenceRequest) (app.LedgerEvent, error) {
	writer.count++
	return app.LedgerEvent{}, nil
}

func (writer *countingProposalWriter) SubmitProposal(context.Context, app.SubmitProposalRequest) (app.SubmitProposalResult, error) {
	writer.count++
	return app.SubmitProposalResult{}, nil
}
