package reportfinaledit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	patchhandler "github.com/c86j224s/liquid2/plasma/internal/mcp/reportpatch"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

type ReadOnlyValidationDraft struct {
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

type ReportLongFormStyleSemanticValidationSubmitInput struct {
	wire.CommonMutatingInput
	DraftID            string                                          `json:"draft_id"`
	PendingEventID     string                                          `json:"pending_event_id"`
	PlanEventID        string                                          `json:"plan_event_id"`
	SemanticAcceptance *[]ReportLongFormStyleSemanticValidationVerdict `json:"semantic_acceptance"`
}

type ReportLongFormStyleSemanticValidationVerdict struct {
	ParagraphOrdinal int    `json:"paragraph_ordinal"`
	Verdict          string `json:"verdict"`
}

type ReportLongFormEvidenceGateSubmitInput struct {
	wire.CommonMutatingInput
	DraftID        string                                    `json:"draft_id"`
	PendingEventID string                                    `json:"pending_event_id"`
	PlanEventID    string                                    `json:"plan_event_id"`
	GateFindings   *[]ReportLongFormEvidenceGateFindingInput `json:"gate_findings"`
}

type ReportLongFormEvidenceGateFindingInput struct {
	StatementSHA256 string   `json:"statement_sha256"`
	Classification  string   `json:"classification"`
	EvidenceIDs     []string `json:"evidence_ids"`
}

func (server *Handler) CallReportLongFormStyleSemanticValidationRead(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input ReportLongFormEditReadInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", "style semantic validation read arguments are invalid", false, nil)
	}
	missionID, sessionID, draftID := strings.TrimSpace(input.MissionID), strings.TrimSpace(input.SessionID), strings.TrimSpace(input.DraftID)
	if err := server.ValidateID("mis_", missionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.ValidateID("ses_", sessionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.ValidateID("rfe_", draftID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	binding, err := server.RequireStageBinding(wire.CommonMutatingInput{MissionID: missionID, SessionID: sessionID}, reporting.FinalEditStageStyleSemanticValidation)
	if err != nil {
		return server.ErrorResult(call.Name, missionID, "binding", err.Error(), false, nil)
	}
	if _, _, err := reporting.StartFinalEditStage(ctx, server.Service, server.NewID("evt"), binding); err != nil {
		return server.ErrorFromErr(call.Name, missionID, err, []string{draftID})
	}
	packetBytes, changedCount, sourceSHA, err := server.styleSemanticValidationPacket(ctx, binding)
	if err != nil {
		return server.ErrorFromErr(call.Name, missionID, err, []string{draftID})
	}
	content, offset, nextOffset, truncated, err := server.readReadOnlyValidationPacket(missionID, sessionID, draftID, binding, sourceSHA, packetBytes, input.Offset, input.MaxBytes)
	if err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: missionID, Content: map[string]any{
		"draft_id": draftID, "stage": binding.Stage, "content": content, "offset": offset, "next_offset": nextOffset,
		"content_length": len(packetBytes), "changed_paragraph_count": changedCount, "truncated": truncated,
	}}
}

func (server *Handler) CallReportLongFormStyleSemanticValidationSubmit(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input ReportLongFormStyleSemanticValidationSubmitInput
	if err := rejectUnexpectedReadOnlyValidationSubmitKeys(call.Arguments, reporting.FinalEditStageStyleSemanticValidation); err != nil {
		return server.ErrorResult(call.Name, "", "validation", err.Error(), false, nil)
	}
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", "style semantic validation submit arguments are invalid", false, nil)
	}
	common, _, err := server.NormalizeInput(input.CommonMutatingInput)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil)
	}
	binding, err := server.RequireStageBinding(common, reporting.FinalEditStageStyleSemanticValidation)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "binding", err.Error(), false, nil)
	}
	if input.PendingEventID != binding.PendingEventID || input.PlanEventID != binding.PlanEventID {
		return server.ErrorResult(call.Name, common.MissionID, "binding", "style semantic validation submit does not match the runner binding", false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	if err := server.ValidateID("rfe_", draftID); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	reviews, err := readOnlyStyleSemanticAcceptanceFromInput(input.SemanticAcceptance)
	if err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", "style semantic validation verdicts are invalid", false, []string{draftID})
	}
	packetBytes, _, sourceSHA, err := server.styleSemanticValidationPacket(ctx, binding)
	if err != nil {
		return server.ErrorFromErr(call.Name, common.MissionID, err, []string{draftID})
	}
	if err := server.requireReadOnlyValidationComplete(common.MissionID, common.SessionID, draftID, binding, sourceSHA, packetBytes); err != nil {
		return server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID})
	}
	result, err := reporting.SubmitFinalEditStyleSemanticValidation(ctx, server.Service, binding, server.NewID("evt"), reviews)
	if err != nil {
		return server.ErrorFromErr(call.Name, common.MissionID, err, []string{draftID})
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: common.MissionID, CreatedEventIDs: []string{result.Event.EventID}, Content: map[string]any{
		"draft_id": draftID, "stage": binding.Stage, "submitted": true, "artifact_id": result.Artifact.ArtifactID, "event_id": result.Event.EventID,
	}}
}

func (server *Handler) CallReportLongFormEvidenceGateRead(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input ReportLongFormEditReadInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", "evidence gate read arguments are invalid", false, nil)
	}
	missionID, sessionID, draftID := strings.TrimSpace(input.MissionID), strings.TrimSpace(input.SessionID), strings.TrimSpace(input.DraftID)
	if err := server.ValidateID("mis_", missionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.ValidateID("ses_", sessionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.ValidateID("rfe_", draftID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID})
	}
	binding, err := server.RequireStageBinding(wire.CommonMutatingInput{MissionID: missionID, SessionID: sessionID}, reporting.FinalEditStageEvidenceGate)
	if err != nil {
		return server.withReadOnlyValidationContinuation(server.ErrorResult(call.Name, missionID, "binding", err.Error(), false, nil), draftID, server.StageBinding())
	}
	if _, _, err := reporting.StartFinalEditStage(ctx, server.Service, server.NewID("evt"), binding); err != nil {
		return server.ErrorFromErr(call.Name, missionID, err, []string{draftID})
	}
	packetBytes, passageCount, sourceSHA, err := server.evidenceGatePacket(ctx, binding)
	if err != nil {
		return server.ErrorFromErr(call.Name, missionID, err, []string{draftID})
	}
	content, offset, nextOffset, truncated, err := server.readReadOnlyValidationPacket(missionID, sessionID, draftID, binding, sourceSHA, packetBytes, input.Offset, input.MaxBytes)
	if err != nil {
		return server.withReadOnlyValidationContinuation(server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, []string{draftID}), draftID, binding)
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: missionID, Content: map[string]any{
		"draft_id": draftID, "stage": binding.Stage, "content": content, "offset": offset, "next_offset": nextOffset,
		"content_length": len(packetBytes), "passage_count": passageCount, "truncated": truncated,
	}}
}

func (server *Handler) CallReportLongFormEvidenceGateSubmit(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input ReportLongFormEvidenceGateSubmitInput
	if err := rejectUnexpectedReadOnlyValidationSubmitKeys(call.Arguments, reporting.FinalEditStageEvidenceGate); err != nil {
		return server.ErrorResult(call.Name, "", "validation", err.Error(), false, nil)
	}
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", "evidence gate submit arguments are invalid", false, nil)
	}
	draftID := strings.TrimSpace(input.DraftID)
	common, _, err := server.NormalizeInput(input.CommonMutatingInput)
	if err != nil {
		return server.withReadOnlyValidationContinuation(server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, nil), draftID, server.StageBinding())
	}
	binding, err := server.RequireStageBinding(common, reporting.FinalEditStageEvidenceGate)
	if err != nil {
		return server.withReadOnlyValidationContinuation(server.ErrorResult(call.Name, common.MissionID, "binding", err.Error(), false, nil), draftID, server.StageBinding())
	}
	if input.PendingEventID != binding.PendingEventID || input.PlanEventID != binding.PlanEventID {
		return server.withReadOnlyValidationContinuation(server.ErrorResult(call.Name, common.MissionID, "binding", "evidence gate submit does not match the runner binding", false, nil), draftID, binding)
	}
	if err := server.ValidateID("rfe_", draftID); err != nil {
		return server.withReadOnlyValidationContinuation(server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID}), draftID, binding)
	}
	findings, err := readOnlyEvidenceGateFindingsFromInput(input.GateFindings)
	if err != nil {
		return server.withReadOnlyValidationContinuation(server.ErrorResult(call.Name, common.MissionID, "validation", "evidence gate findings are invalid", false, []string{draftID}), draftID, binding)
	}
	packetBytes, _, sourceSHA, err := server.evidenceGatePacket(ctx, binding)
	if err != nil {
		return server.ErrorFromErr(call.Name, common.MissionID, err, []string{draftID})
	}
	if err := server.requireReadOnlyValidationComplete(common.MissionID, common.SessionID, draftID, binding, sourceSHA, packetBytes); err != nil {
		return server.withReadOnlyValidationContinuation(server.ErrorResult(call.Name, common.MissionID, "validation", err.Error(), false, []string{draftID}), draftID, binding)
	}
	result, err := reporting.SubmitFinalEditEvidenceGate(ctx, server.Service, reporting.FinalEditEvidenceGateSubmitRequest{
		StageBinding: binding, FinalBinding: server.FinalizeBinding(),
		StageEventID: server.NewID("evt"), CanonicalEventID: server.NewID("evt"), Findings: findings,
	})
	if err != nil {
		return server.withReadOnlyValidationContinuation(server.ErrorFromErr(call.Name, common.MissionID, err, []string{draftID}), draftID, binding)
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: common.MissionID, CreatedEventIDs: []string{result.Event.EventID}, Content: map[string]any{
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
		return fmt.Errorf("%w: unsupported read-only validation stage", producterror.ErrInvalidInput)
	}
	for key := range top {
		if _, ok := allowedTop[key]; !ok {
			return fmt.Errorf("%w: read-only validation submit field %q is not allowed", producterror.ErrInvalidInput, key)
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
				return fmt.Errorf("%w: read-only validation submit field %q is not allowed", producterror.ErrInvalidInput, key)
			}
		}
	}
	return nil
}

func readOnlyStyleSemanticAcceptanceFromInput(input *[]ReportLongFormStyleSemanticValidationVerdict) ([]reporting.FinalEditSemanticAcceptance, error) {
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

func readOnlyEvidenceGateFindingsFromInput(input *[]ReportLongFormEvidenceGateFindingInput) ([]reporting.FinalEditGateFinding, error) {
	if input == nil {
		return nil, fmt.Errorf("%w: evidence gate findings are required", producterror.ErrInvalidInput)
	}
	findings := make([]reporting.FinalEditGateFinding, 0, len(*input))
	for _, item := range *input {
		statementSHA := strings.TrimSpace(item.StatementSHA256)
		classification := strings.TrimSpace(item.Classification)
		if statementSHA == "" || classification == "" {
			return nil, fmt.Errorf("%w: evidence gate finding is incomplete", producterror.ErrInvalidInput)
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

func (server *Handler) styleSemanticValidationPacket(ctx context.Context, binding reporting.FinalEditStageBinding) ([]byte, int, string, error) {
	source, err := server.Service.GetRawArtifact(ctx, binding.SourceArtifactID)
	if err != nil {
		return nil, 0, "", err
	}
	comparison, err := reporting.FinalEditSemanticComparison(ctx, server.Service, binding, "")
	if err != nil {
		return nil, 0, "", err
	}
	packetBytes, err := json.MarshalIndent(comparison, "", "  ")
	if err != nil {
		return nil, 0, "", err
	}
	return packetBytes, len(comparison), source.SHA256, nil
}

func (server *Handler) evidenceGatePacket(ctx context.Context, binding reporting.FinalEditStageBinding) ([]byte, int, string, error) {
	source, err := server.Service.GetRawArtifact(ctx, binding.SourceArtifactID)
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

func (server *Handler) readReadOnlyValidationPacket(missionID, sessionID, draftID string, binding reporting.FinalEditStageBinding, sourceSHA string, packet []byte, offset int, maxBytes int) (string, int, int, bool, error) {
	packetSHA := readOnlyValidationSHA(packet)
	server.Mu.Lock()
	current := server.State.ValidationDrafts[draftID]
	if current == nil {
		if binding.Stage == reporting.FinalEditStageEvidenceGate {
			if active := server.activeReadOnlyValidationDraftLocked(missionID, sessionID, binding); active != nil {
				activeDraftID, activeNextOffset := active.DraftID, active.NextOffset
				server.Mu.Unlock()
				return "", 0, 0, false, fmt.Errorf("%w: evidence gate must continue draft_id %s at offset %d", producterror.ErrConflict, activeDraftID, activeNextOffset)
			}
		}
		if offset != 0 {
			server.Mu.Unlock()
			return "", 0, 0, false, fmt.Errorf("%w: read-only validation reads must start at offset 0", producterror.ErrInvalidInput)
		}
		if len(server.State.ValidationDrafts) >= reportLongFormEditMaxDrafts {
			server.Mu.Unlock()
			return "", 0, 0, false, fmt.Errorf("%w: too many in-process read-only validation drafts", producterror.ErrConflict)
		}
		now := nowUTC()
		current = &ReadOnlyValidationDraft{
			DraftID: draftID, Stage: binding.Stage, MissionID: missionID, SessionID: sessionID,
			PendingID: binding.PendingEventID, PlanEventID: binding.PlanEventID, SourceArtifactID: binding.SourceArtifactID,
			SourceSHA256: sourceSHA, PacketSHA256: packetSHA, CreatedAt: now, UpdatedAt: now,
		}
		server.State.ValidationDrafts[draftID] = current
	} else if err := current.validate(missionID, sessionID, binding, sourceSHA, packetSHA); err != nil {
		server.Mu.Unlock()
		return "", 0, 0, false, err
	}
	expectedOffset := current.NextOffset
	server.Mu.Unlock()
	if offset != expectedOffset {
		return "", 0, 0, false, fmt.Errorf("%w: read-only validation reads must use contiguous next_offset values; continue draft_id %s at offset %d", producterror.ErrInvalidInput, draftID, expectedOffset)
	}
	content, actualOffset, nextOffset, truncated, err := patchhandler.BoundedContent(string(packet), offset, maxBytes)
	if err != nil {
		return "", 0, 0, false, err
	}
	server.Mu.Lock()
	if current := server.State.ValidationDrafts[draftID]; current != nil {
		current.NextOffset = nextOffset
		current.Complete = !truncated
		current.UpdatedAt = nowUTC()
	}
	server.Mu.Unlock()
	return content, actualOffset, nextOffset, truncated, nil
}

func (server *Handler) requireReadOnlyValidationComplete(missionID, sessionID, draftID string, binding reporting.FinalEditStageBinding, sourceSHA string, packet []byte) error {
	packetSHA := readOnlyValidationSHA(packet)
	server.Mu.Lock()
	defer server.Mu.Unlock()
	current := server.State.ValidationDrafts[draftID]
	if current == nil {
		return fmt.Errorf("%w: read-only validation packet must be read to completion before submit", producterror.ErrConflict)
	}
	if err := current.validate(missionID, sessionID, binding, sourceSHA, packetSHA); err != nil {
		return err
	}
	if !current.Complete {
		return fmt.Errorf("%w: read-only validation packet must be read to completion before submit; continue draft_id %s at offset %d", producterror.ErrConflict, current.DraftID, current.NextOffset)
	}
	return nil
}

// withReadOnlyValidationContinuation keeps contract failures recoverable without
// weakening the complete, contiguous packet-read requirement.
func (server *Handler) withReadOnlyValidationContinuation(result wire.ToolResult, requestedDraftID string, binding reporting.FinalEditStageBinding) wire.ToolResult {
	server.Mu.Lock()
	defer server.Mu.Unlock()
	current := server.activeReadOnlyValidationDraftLocked(binding.MissionID, binding.ToolSessionID, binding)
	if current == nil {
		if server.ValidateID("rfe_", requestedDraftID) != nil || strings.TrimSpace(binding.ToolSessionID) == "" {
			return result
		}
		result.Content = readOnlyValidationContinuation(requestedDraftID, binding.ToolSessionID, 0, false)
		return result
	}
	result.Content = readOnlyValidationContinuation(current.DraftID, current.SessionID, current.NextOffset, current.Complete)
	return result
}

func (server *Handler) activeReadOnlyValidationDraftLocked(missionID, sessionID string, binding reporting.FinalEditStageBinding) *ReadOnlyValidationDraft {
	for _, draft := range server.State.ValidationDrafts {
		if draft.MissionID == missionID && draft.SessionID == sessionID && draft.Stage == binding.Stage &&
			draft.PendingID == binding.PendingEventID && draft.PlanEventID == binding.PlanEventID && draft.SourceArtifactID == binding.SourceArtifactID {
			return draft
		}
	}
	return nil
}

func readOnlyValidationContinuation(draftID, sessionID string, nextOffset int, complete bool) map[string]any {
	nextAction := "read"
	if complete {
		nextAction = "submit_once"
	}
	return map[string]any{
		"draft_id": draftID, "session_id": sessionID, "next_offset": nextOffset,
		"packet_complete": complete, "next_action": nextAction,
	}
}

func (draft *ReadOnlyValidationDraft) validate(missionID, sessionID string, binding reporting.FinalEditStageBinding, sourceSHA, packetSHA string) error {
	if draft.MissionID != missionID || draft.SessionID != sessionID || draft.Stage != binding.Stage ||
		draft.PendingID != binding.PendingEventID || draft.PlanEventID != binding.PlanEventID ||
		draft.SourceArtifactID != binding.SourceArtifactID || draft.SourceSHA256 != sourceSHA || draft.PacketSHA256 != packetSHA {
		return fmt.Errorf("%w: read-only validation draft differs from the bound stage packet", producterror.ErrConflict)
	}
	return nil
}

func readOnlyValidationSHA(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
