package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

type readOnlyValidationDraft struct {
	DraftID          string
	Stage            string
	MissionID        string
	SessionID        string
	PendingID        string
	PlanEventID      string
	SourceArtifactID string
	SourceSHA256     string
	PacketSHA256     string
	NextOffset       int
	Complete         bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type reportLongFormStyleSemanticValidationSubmitInput struct {
	CommonMutatingInput
	DraftID            string                                          `json:"draft_id"`
	PendingEventID     string                                          `json:"pending_event_id"`
	PlanEventID        string                                          `json:"plan_event_id"`
	SemanticAcceptance *[]reportLongFormStyleSemanticValidationVerdict `json:"semantic_acceptance"`
}

type reportLongFormStyleSemanticValidationVerdict struct {
	ParagraphOrdinal int    `json:"paragraph_ordinal"`
	Verdict          string `json:"verdict"`
}

type reportLongFormEvidenceGateSubmitInput struct {
	CommonMutatingInput
	DraftID        string                                    `json:"draft_id"`
	PendingEventID string                                    `json:"pending_event_id"`
	PlanEventID    string                                    `json:"plan_event_id"`
	GateFindings   *[]reportLongFormEvidenceGateFindingInput `json:"gate_findings"`
}

type reportLongFormEvidenceGateFindingInput struct {
	StatementSHA256 string   `json:"statement_sha256"`
	Classification  string   `json:"classification"`
	EvidenceIDs     []string `json:"evidence_ids"`
}

func (server *Server) callReportLongFormStyleSemanticValidationRead(ctx context.Context, call ToolCall) ToolResult {
	var input reportLongFormEditReadInput
	if err := decodeReportPlanJSON(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", "style semantic validation read arguments are invalid", false, nil)
	}
	missionID, sessionID, draftID := strings.TrimSpace(input.MissionID), strings.TrimSpace(input.SessionID), strings.TrimSpace(input.DraftID)
	if err := validateID("mis_", missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := validateID("ses_", sessionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := validateID("rfe_", draftID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	binding, err := server.requireFinalEditStageBinding(commonMutatingInput{MissionID: missionID, SessionID: sessionID}, reporting.FinalEditStageStyleSemanticValidation)
	if err != nil {
		return errorResult(call.Name, missionID, "binding", err.Error(), false, nil)
	}
	if _, _, err := reporting.StartFinalEditStage(ctx, server.service, newMCPID("evt"), binding); err != nil {
		return errorFromErr(call.Name, missionID, err, []string{draftID})
	}
	packetBytes, changedCount, sourceSHA, err := server.styleSemanticValidationPacket(ctx, binding)
	if err != nil {
		return errorFromErr(call.Name, missionID, err, []string{draftID})
	}
	content, offset, nextOffset, truncated, err := server.readReadOnlyValidationPacket(missionID, sessionID, draftID, binding, sourceSHA, packetBytes, input.Offset, input.MaxBytes)
	if err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	return ToolResult{ToolName: call.Name, MissionID: missionID, Content: map[string]any{
		"draft_id": draftID, "stage": binding.Stage, "content": content, "offset": offset, "next_offset": nextOffset,
		"content_length": len(packetBytes), "changed_paragraph_count": changedCount, "truncated": truncated,
	}}
}

func (server *Server) callReportLongFormStyleSemanticValidationSubmit(ctx context.Context, call ToolCall) ToolResult {
	var input reportLongFormStyleSemanticValidationSubmitInput
	if err := rejectUnexpectedReadOnlyValidationSubmitKeys(call.Arguments, reporting.FinalEditStageStyleSemanticValidation); err != nil {
		return errorResult(call.Name, "", "validation", err.Error(), false, nil)
	}
	if err := decodeReportPlanJSON(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", "style semantic validation submit arguments are invalid", false, nil)
	}
	common, _, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	binding, err := server.requireFinalEditStageBinding(common, reporting.FinalEditStageStyleSemanticValidation)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "binding", err.Error(), false, nil)
	}
	if input.PendingEventID != binding.PendingEventID || input.PlanEventID != binding.PlanEventID {
		return errorResult(call.Name, common.MissionID, "binding", "style semantic validation submit does not match the runner binding", false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if err := validateID("rfe_", draftID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	reviews, err := readOnlyStyleSemanticAcceptanceFromInput(input.SemanticAcceptance)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", "style semantic validation verdicts are invalid", false, []string{draftID})
	}
	packetBytes, _, sourceSHA, err := server.styleSemanticValidationPacket(ctx, binding)
	if err != nil {
		return errorFromErr(call.Name, common.MissionID, err, []string{draftID})
	}
	if err := server.requireReadOnlyValidationComplete(common.MissionID, common.SessionID, draftID, binding, sourceSHA, packetBytes); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	result, err := reporting.SubmitFinalEditStyleSemanticValidation(ctx, server.service, binding, newMCPID("evt"), reviews)
	if err != nil {
		return errorFromErr(call.Name, common.MissionID, err, []string{draftID})
	}
	return ToolResult{ToolName: call.Name, MissionID: common.MissionID, CreatedEventIDs: []string{result.Event.EventID}, Content: map[string]any{
		"draft_id": draftID, "stage": binding.Stage, "submitted": true, "artifact_id": result.Artifact.ArtifactID, "event_id": result.Event.EventID,
	}}
}

func (server *Server) callReportLongFormEvidenceGateRead(ctx context.Context, call ToolCall) ToolResult {
	var input reportLongFormEditReadInput
	if err := decodeReportPlanJSON(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", "evidence gate read arguments are invalid", false, nil)
	}
	missionID, sessionID, draftID := strings.TrimSpace(input.MissionID), strings.TrimSpace(input.SessionID), strings.TrimSpace(input.DraftID)
	if err := validateID("mis_", missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := validateID("ses_", sessionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := validateID("rfe_", draftID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	binding, err := server.requireFinalEditStageBinding(commonMutatingInput{MissionID: missionID, SessionID: sessionID}, reporting.FinalEditStageEvidenceGate)
	if err != nil {
		return errorResult(call.Name, missionID, "binding", err.Error(), false, nil)
	}
	if _, _, err := reporting.StartFinalEditStage(ctx, server.service, newMCPID("evt"), binding); err != nil {
		return errorFromErr(call.Name, missionID, err, []string{draftID})
	}
	packetBytes, passageCount, sourceSHA, err := server.evidenceGatePacket(ctx, binding)
	if err != nil {
		return errorFromErr(call.Name, missionID, err, []string{draftID})
	}
	content, offset, nextOffset, truncated, err := server.readReadOnlyValidationPacket(missionID, sessionID, draftID, binding, sourceSHA, packetBytes, input.Offset, input.MaxBytes)
	if err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	return ToolResult{ToolName: call.Name, MissionID: missionID, Content: map[string]any{
		"draft_id": draftID, "stage": binding.Stage, "content": content, "offset": offset, "next_offset": nextOffset,
		"content_length": len(packetBytes), "passage_count": passageCount, "truncated": truncated,
	}}
}

func (server *Server) callReportLongFormEvidenceGateSubmit(ctx context.Context, call ToolCall) ToolResult {
	var input reportLongFormEvidenceGateSubmitInput
	if err := rejectUnexpectedReadOnlyValidationSubmitKeys(call.Arguments, reporting.FinalEditStageEvidenceGate); err != nil {
		return errorResult(call.Name, "", "validation", err.Error(), false, nil)
	}
	if err := decodeReportPlanJSON(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", "evidence gate submit arguments are invalid", false, nil)
	}
	common, _, err := normalizeMutatingInput(input.CommonMutatingInput)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	binding, err := server.requireFinalEditStageBinding(common, reporting.FinalEditStageEvidenceGate)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "binding", err.Error(), false, nil)
	}
	if input.PendingEventID != binding.PendingEventID || input.PlanEventID != binding.PlanEventID {
		return errorResult(call.Name, common.MissionID, "binding", "evidence gate submit does not match the runner binding", false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if err := validateID("rfe_", draftID); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	findings, err := readOnlyEvidenceGateFindingsFromInput(input.GateFindings)
	if err != nil {
		return errorResult(call.Name, common.MissionID, "validation", "evidence gate findings are invalid", false, []string{draftID})
	}
	packetBytes, _, sourceSHA, err := server.evidenceGatePacket(ctx, binding)
	if err != nil {
		return errorFromErr(call.Name, common.MissionID, err, []string{draftID})
	}
	if err := server.requireReadOnlyValidationComplete(common.MissionID, common.SessionID, draftID, binding, sourceSHA, packetBytes); err != nil {
		return errorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	result, err := reporting.SubmitFinalEditEvidenceGate(ctx, server.service, reporting.FinalEditEvidenceGateSubmitRequest{
		StageBinding: binding, FinalBinding: server.longFormFinalizeBinding,
		StageEventID: newMCPID("evt"), CanonicalEventID: newMCPID("evt"), Findings: findings,
	})
	if err != nil {
		return errorFromErr(call.Name, common.MissionID, err, []string{draftID})
	}
	return ToolResult{ToolName: call.Name, MissionID: common.MissionID, CreatedEventIDs: []string{result.Event.EventID}, Content: map[string]any{
		"draft_id": draftID, "stage": binding.Stage, "submitted": true, "artifact_id": result.Artifact.ArtifactID, "event_id": result.Event.EventID,
	}}
}

func rejectUnexpectedReadOnlyValidationSubmitKeys(raw json.RawMessage, stage string) error {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return err
	}
	allowedTop := map[string]struct{}{
		"mission_id": {}, "session_id": {}, "idempotency_key": {}, "producer": {},
		"draft_id": {}, "pending_event_id": {}, "plan_event_id": {},
	}
	var listKey string
	allowedItem := map[string]struct{}{}
	switch stage {
	case reporting.FinalEditStageStyleSemanticValidation:
		listKey = "semantic_acceptance"
		allowedTop[listKey] = struct{}{}
		allowedItem = map[string]struct{}{"paragraph_ordinal": {}, "verdict": {}}
	case reporting.FinalEditStageEvidenceGate:
		listKey = "gate_findings"
		allowedTop[listKey] = struct{}{}
		allowedItem = map[string]struct{}{"statement_sha256": {}, "classification": {}, "evidence_ids": {}}
	default:
		return fmt.Errorf("%w: unsupported read-only validation stage", app.ErrInvalidInput)
	}
	for key := range top {
		if _, ok := allowedTop[key]; !ok {
			return fmt.Errorf("%w: read-only validation submit field %q is not allowed", app.ErrInvalidInput, key)
		}
	}
	rawItems, ok := top[listKey]
	if !ok || string(rawItems) == "null" {
		return nil
	}
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(rawItems, &items); err != nil {
		return nil
	}
	for _, item := range items {
		for key := range item {
			if _, ok := allowedItem[key]; !ok {
				return fmt.Errorf("%w: read-only validation submit field %q is not allowed", app.ErrInvalidInput, key)
			}
		}
	}
	return nil
}

func readOnlyStyleSemanticAcceptanceFromInput(input *[]reportLongFormStyleSemanticValidationVerdict) ([]reporting.FinalEditSemanticAcceptance, error) {
	if input == nil {
		return nil, nil
	}
	out := make([]reporting.FinalEditSemanticAcceptance, 0, len(*input))
	for _, item := range *input {
		out = append(out, reporting.FinalEditSemanticAcceptance{
			ParagraphOrdinal: item.ParagraphOrdinal,
			Verdict:          strings.TrimSpace(item.Verdict),
		})
	}
	return out, nil
}

func readOnlyEvidenceGateFindingsFromInput(input *[]reportLongFormEvidenceGateFindingInput) ([]reporting.FinalEditGateFinding, error) {
	if input == nil {
		return nil, fmt.Errorf("%w: evidence gate findings are required", app.ErrInvalidInput)
	}
	findings := make([]reporting.FinalEditGateFinding, 0, len(*input))
	for _, item := range *input {
		statementSHA := strings.TrimSpace(item.StatementSHA256)
		classification := strings.TrimSpace(item.Classification)
		if statementSHA == "" || classification == "" {
			return nil, fmt.Errorf("%w: evidence gate finding is incomplete", app.ErrInvalidInput)
		}
		evidenceIDs := make([]string, 0, len(item.EvidenceIDs))
		for _, evidenceID := range item.EvidenceIDs {
			if trimmed := strings.TrimSpace(evidenceID); trimmed != "" {
				evidenceIDs = append(evidenceIDs, trimmed)
			}
		}
		findings = append(findings, reporting.FinalEditGateFinding{
			StatementSHA256: statementSHA, Classification: classification, EvidenceIDs: evidenceIDs,
		})
	}
	return findings, nil
}

func (server *Server) styleSemanticValidationPacket(ctx context.Context, binding reporting.FinalEditStageBinding) ([]byte, int, string, error) {
	source, err := server.service.GetRawArtifact(ctx, binding.SourceArtifactID)
	if err != nil {
		return nil, 0, "", err
	}
	comparison, err := reporting.FinalEditSemanticComparison(ctx, server.service, binding, "")
	if err != nil {
		return nil, 0, "", err
	}
	packetBytes, err := json.MarshalIndent(comparison, "", "  ")
	if err != nil {
		return nil, 0, "", err
	}
	return packetBytes, len(comparison), source.SHA256, nil
}

func (server *Server) evidenceGatePacket(ctx context.Context, binding reporting.FinalEditStageBinding) ([]byte, int, string, error) {
	source, err := server.service.GetRawArtifact(ctx, binding.SourceArtifactID)
	if err != nil {
		return nil, 0, "", err
	}
	passages, err := reporting.FinalEditEvidenceGatePassages(string(source.Content))
	if err != nil {
		return nil, 0, "", err
	}
	packetBytes, err := json.MarshalIndent(map[string]any{
		"source_artifact_id": binding.SourceArtifactID,
		"source_sha256":      source.SHA256,
		"passages":           passages,
	}, "", "  ")
	if err != nil {
		return nil, 0, "", err
	}
	return packetBytes, len(passages), source.SHA256, nil
}

func (server *Server) readReadOnlyValidationPacket(missionID, sessionID, draftID string, binding reporting.FinalEditStageBinding, sourceSHA string, packet []byte, offset int, maxBytes int) (string, int, int, bool, error) {
	packetSHA := readOnlyValidationSHA(packet)
	server.mu.Lock()
	current := server.readOnlyValidationDrafts[draftID]
	if current == nil {
		if offset != 0 {
			server.mu.Unlock()
			return "", 0, 0, false, fmt.Errorf("%w: read-only validation reads must start at offset 0", app.ErrInvalidInput)
		}
		if len(server.readOnlyValidationDrafts) >= reportLongFormEditMaxDrafts {
			server.mu.Unlock()
			return "", 0, 0, false, fmt.Errorf("%w: too many in-process read-only validation drafts", app.ErrConflict)
		}
		now := nowUTC()
		current = &readOnlyValidationDraft{
			DraftID: draftID, Stage: binding.Stage, MissionID: missionID, SessionID: sessionID,
			PendingID: binding.PendingEventID, PlanEventID: binding.PlanEventID, SourceArtifactID: binding.SourceArtifactID,
			SourceSHA256: sourceSHA, PacketSHA256: packetSHA, CreatedAt: now, UpdatedAt: now,
		}
		server.readOnlyValidationDrafts[draftID] = current
	} else if err := current.validate(missionID, sessionID, binding, sourceSHA, packetSHA); err != nil {
		server.mu.Unlock()
		return "", 0, 0, false, err
	}
	expectedOffset := current.NextOffset
	server.mu.Unlock()
	if offset != expectedOffset {
		return "", 0, 0, false, fmt.Errorf("%w: read-only validation reads must use contiguous next_offset values", app.ErrInvalidInput)
	}
	content, actualOffset, nextOffset, truncated, err := boundedReportPatchContent(string(packet), offset, maxBytes)
	if err != nil {
		return "", 0, 0, false, err
	}
	server.mu.Lock()
	if current := server.readOnlyValidationDrafts[draftID]; current != nil {
		current.NextOffset = nextOffset
		current.Complete = !truncated
		current.UpdatedAt = nowUTC()
	}
	server.mu.Unlock()
	return content, actualOffset, nextOffset, truncated, nil
}

func (server *Server) requireReadOnlyValidationComplete(missionID, sessionID, draftID string, binding reporting.FinalEditStageBinding, sourceSHA string, packet []byte) error {
	packetSHA := readOnlyValidationSHA(packet)
	server.mu.Lock()
	defer server.mu.Unlock()
	current := server.readOnlyValidationDrafts[draftID]
	if current == nil {
		return fmt.Errorf("%w: read-only validation packet must be read to completion before submit", app.ErrConflict)
	}
	if err := current.validate(missionID, sessionID, binding, sourceSHA, packetSHA); err != nil {
		return err
	}
	if !current.Complete {
		return fmt.Errorf("%w: read-only validation packet must be read to completion before submit", app.ErrConflict)
	}
	return nil
}

func (draft *readOnlyValidationDraft) validate(missionID, sessionID string, binding reporting.FinalEditStageBinding, sourceSHA, packetSHA string) error {
	if draft.MissionID != missionID || draft.SessionID != sessionID || draft.Stage != binding.Stage ||
		draft.PendingID != binding.PendingEventID || draft.PlanEventID != binding.PlanEventID ||
		draft.SourceArtifactID != binding.SourceArtifactID || draft.SourceSHA256 != sourceSHA || draft.PacketSHA256 != packetSHA {
		return fmt.Errorf("%w: read-only validation draft differs from the bound stage packet", app.ErrConflict)
	}
	return nil
}

func readOnlyValidationSHA(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
