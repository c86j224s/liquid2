package reporting

import (
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func decodeStoredFinalEditGateFindingsPayload(value any) ([]StoredFinalEditGateFinding, error) {
	return decodeLegacyStoredFinalEditGateFindingsPayload(value)
}

func decodeStoredFinalEditGateFindingsPayloadForStage(value any, pipeline string, stage string) ([]StoredFinalEditGateFinding, error) {
	if pipeline == FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 && stage == FinalEditStageEvidenceGate {
		return decodeEvidenceGateStoredFinalEditGateFindingsPayload(value)
	}
	return decodeLegacyStoredFinalEditGateFindingsPayload(value)
}

func decodeStoredFinalEditGateFindingsPayloadForPipeline(value any, pipeline string) ([]StoredFinalEditGateFinding, error) {
	if pipeline == FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 {
		return decodeEvidenceGateStoredFinalEditGateFindingsPayload(value)
	}
	return decodeLegacyStoredFinalEditGateFindingsPayload(value)
}

func decodeLegacyStoredFinalEditGateFindingsPayload(value any) ([]StoredFinalEditGateFinding, error) {
	if value == nil {
		return nil, nil
	}
	items, err := decodeStoredFinalEditGateFindingItems(value)
	if err != nil {
		return nil, err
	}
	out := make([]StoredFinalEditGateFinding, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		if !validStoredFinalEditStatementSHA256(item.StatementSHA256) || !finalEditGateClasses[item.Classification] || seen[item.StatementSHA256] {
			return nil, fmt.Errorf("%w: final edit gate finding payload is invalid", producterror.ErrConflict)
		}
		seen[item.StatementSHA256] = true
		if item.Classification == FinalEditGateClassUnverifiedExternalFact {
			if !validFinalEditRepairAction(item.RepairAction) {
				return nil, fmt.Errorf("%w: final edit gate finding repair action is invalid", producterror.ErrConflict)
			}
		} else if item.RepairAction != "" {
			return nil, fmt.Errorf("%w: final edit gate finding repair action is invalid", producterror.ErrConflict)
		}
		if len(item.EvidenceIDs) > 0 && item.RepairAction != FinalEditRepairAttachApprovedEvidence {
			return nil, fmt.Errorf("%w: final edit gate finding evidence refs are invalid", producterror.ErrConflict)
		}
		if item.RepairAction == FinalEditRepairAttachApprovedEvidence && len(item.EvidenceIDs) == 0 {
			return nil, fmt.Errorf("%w: final edit gate finding evidence refs are invalid", producterror.ErrConflict)
		}
		out = append(out, item)
	}
	return out, nil
}

func decodeEvidenceGateStoredFinalEditGateFindingsPayload(value any) ([]StoredFinalEditGateFinding, error) {
	if value == nil {
		return nil, nil
	}
	items, err := decodeStoredFinalEditGateFindingItems(value)
	if err != nil {
		return nil, err
	}
	out := make([]StoredFinalEditGateFinding, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		if !validStoredFinalEditStatementSHA256(item.StatementSHA256) || !finalEditGateClasses[item.Classification] || seen[item.StatementSHA256] {
			return nil, fmt.Errorf("%w: evidence gate finding payload is invalid", producterror.ErrConflict)
		}
		if item.RepairAction != "" {
			return nil, fmt.Errorf("%w: evidence gate finding repair action is invalid", producterror.ErrConflict)
		}
		seen[item.StatementSHA256] = true
		out = append(out, StoredFinalEditGateFinding{
			StatementSHA256: item.StatementSHA256,
			Classification:  item.Classification,
			EvidenceIDs:     item.EvidenceIDs,
		})
	}
	return out, nil
}

func decodeStoredFinalEditGateFindingItems(value any) ([]StoredFinalEditGateFinding, error) {
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("%w: final edit gate findings payload is invalid", producterror.ErrConflict)
	}
	out := make([]StoredFinalEditGateFinding, 0, len(items))
	for _, item := range items {
		raw, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%w: final edit gate finding payload is invalid", producterror.ErrConflict)
		}
		for key := range raw {
			switch key {
			case "statement_sha256", "classification", "repair_action", "evidence_ids":
			default:
				return nil, fmt.Errorf("%w: final edit gate finding contains unsupported field", producterror.ErrConflict)
			}
		}
		evidenceIDs, err := stringSlicePayload(raw["evidence_ids"])
		if err != nil {
			return nil, err
		}
		out = append(out, StoredFinalEditGateFinding{
			StatementSHA256: payloadString(raw, "statement_sha256"),
			Classification:  payloadString(raw, "classification"),
			RepairAction:    payloadString(raw, "repair_action"),
			EvidenceIDs:     evidenceIDs,
		})
	}
	return out, nil
}
