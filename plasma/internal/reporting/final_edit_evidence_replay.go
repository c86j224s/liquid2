package reporting

import (
	"context"
	"fmt"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"strings"
)

func validateV3EvidenceGateReplay(source artifactcontract.Raw, artifact artifactcontract.Raw, operationCount int, changed bool, findings []StoredFinalEditGateFinding, semanticReview FinalEditSemanticAttestation) error {
	if operationCount != 0 || changed {
		return fmt.Errorf("%w: evidence gate replay must be an unchanged zero-operation submission", producterror.ErrConflict)
	}
	if artifact.ArtifactID != source.ArtifactID || artifact.SHA256 != source.SHA256 || string(artifact.Content) != string(source.Content) {
		return fmt.Errorf("%w: evidence gate replay must reuse the exact source artifact", producterror.ErrConflict)
	}
	if semanticReview.Count != 0 || len(semanticReview.Records) != 0 || strings.TrimSpace(semanticReview.Digest) != "" {
		return fmt.Errorf("%w: evidence gate replay cannot carry semantic acceptance", producterror.ErrConflict)
	}
	return validateFinalEditEvidenceGateFindingStatementsInSource(string(source.Content), findings)
}

func validateStoredFinalEditGateFindingEvidence(ctx context.Context, store LongFormFinalizationStore, missionID string, findings []StoredFinalEditGateFinding) error {
	requiresEvidence := false
	for _, finding := range findings {
		requiresEvidence = requiresEvidence || len(finding.EvidenceIDs) > 0
	}
	if !requiresEvidence {
		return nil
	}
	validator, ok := store.(finalEditEvidenceStore)
	if !ok || validator == nil {
		return fmt.Errorf("%w: final edit gate evidence validator is required", producterror.ErrConflict)
	}
	missionID = strings.TrimSpace(missionID)
	for _, finding := range findings {
		for _, evidenceID := range finding.EvidenceIDs {
			record, err := validator.GetEvidenceRecord(ctx, evidenceID)
			if err != nil || record.MissionID != missionID || strings.TrimSpace(record.State) != "approved" {
				return fmt.Errorf("%w: final edit gate evidence ref is not approved", producterror.ErrConflict)
			}
		}
	}
	return nil
}
