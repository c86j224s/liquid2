package reporting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"sort"
	"strings"
)

func validateV3StyleSemanticValidationReplay(ctx context.Context, store FinalEditStageStore, binding FinalEditStageBinding, source artifactcontract.Raw, artifact artifactcontract.Raw, operationCount int, changed bool, findings []StoredFinalEditGateFinding, semanticReview FinalEditSemanticAttestation) error {
	if operationCount != 0 || len(findings) != 0 {
		return fmt.Errorf("%w: style semantic validation replay must be a zero-operation verdict submission", producterror.ErrConflict)
	}
	records := semanticReview.Records
	for _, record := range records {
		if record.Verdict != FinalEditSemanticAcceptedEquivalent && record.Verdict != FinalEditSemanticRejectedRevertToReader {
			return fmt.Errorf("%w: style semantic validation replay verdict is invalid", producterror.ErrConflict)
		}
	}
	comparison, err := FinalEditSemanticComparison(ctx, store, binding, "")
	if err != nil {
		return err
	}
	if len(comparison) == 0 {
		if changed || artifact.ArtifactID != source.ArtifactID || artifact.SHA256 != source.SHA256 || string(artifact.Content) != string(source.Content) {
			return fmt.Errorf("%w: unchanged style semantic validation replay must reuse source artifact", producterror.ErrConflict)
		}
		if semanticReview.Count != 0 || len(records) != 0 || strings.TrimSpace(semanticReview.Digest) != "" {
			return fmt.Errorf("%w: unchanged style semantic validation replay cannot carry semantic acceptance", producterror.ErrConflict)
		}
		return nil
	}
	if semanticReview.Count != len(comparison) || len(records) != len(comparison) || strings.TrimSpace(semanticReview.Digest) == "" {
		return fmt.Errorf("%w: style semantic validation replay requires complete semantic acceptance", producterror.ErrConflict)
	}
	reviews := make([]FinalEditSemanticAcceptance, 0, len(records))
	for _, record := range records {
		reviews = append(reviews, FinalEditSemanticAcceptance{ParagraphOrdinal: record.ParagraphOrdinal, Verdict: record.Verdict})
	}
	resolved, attestation, err := buildFinalEditStyleSemanticValidation(ctx, store, binding, reviews)
	if err != nil {
		return err
	}
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return err
	}
	style, ok, err := finalEditStyleSubmissionForGate(ctx, store, events, binding)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: style semantic validation replay requires style lineage", producterror.ErrConflict)
	}
	if string(artifact.Content) != resolved || artifact.SHA256 != contentSHA256([]byte(resolved)) {
		return fmt.Errorf("%w: style semantic validation replay artifact differs from deterministic resolution", producterror.ErrConflict)
	}
	if !equalStoredFinalEditSemanticAcceptance(attestation.Records, records) || attestation.Digest != semanticReview.Digest || attestation.Count != semanticReview.Count {
		return fmt.Errorf("%w: style semantic validation replay semantic acceptance differs", producterror.ErrConflict)
	}
	if changed != (artifact.ArtifactID != source.ArtifactID) {
		return fmt.Errorf("%w: style semantic validation replay changed flag differs", producterror.ErrConflict)
	}
	switch {
	case resolved == string(source.Content):
		if changed || artifact.ArtifactID != source.ArtifactID {
			return fmt.Errorf("%w: style semantic validation replay must reuse source artifact", producterror.ErrConflict)
		}
	case resolved == string(style.SourceArtifact.Content):
		if !changed || artifact.ArtifactID != style.SourceArtifact.ArtifactID {
			return fmt.Errorf("%w: style semantic validation replay must reuse reader artifact", producterror.ErrConflict)
		}
	case !changed || artifact.ArtifactID != binding.EditedArtifactID:
		return fmt.Errorf("%w: style semantic validation replay artifact identity differs", producterror.ErrConflict)
	case artifact.Producer != (ledger.Producer{Type: "agent_session", ID: binding.ProviderSessionID}):
		return fmt.Errorf("%w: style semantic validation replay artifact producer differs", producterror.ErrConflict)
	}
	return nil
}

func decodeFinalEditSemanticAcceptancePayload(payload map[string]any) (FinalEditSemanticAttestation, error) {
	records, err := decodeStoredFinalEditSemanticAcceptancePayload(payload["semantic_acceptance"])
	if err != nil {
		return FinalEditSemanticAttestation{}, err
	}
	count, ok := payloadIntStrict(payload, "semantic_acceptance_count")
	if !ok {
		if len(records) == 0 && payload["semantic_acceptance_count"] == nil && payloadString(payload, "semantic_acceptance_digest") == "" {
			return FinalEditSemanticAttestation{}, nil
		}
		return FinalEditSemanticAttestation{}, fmt.Errorf("%w: semantic acceptance count is invalid", producterror.ErrConflict)
	}
	digest := payloadString(payload, "semantic_acceptance_digest")
	expected, err := finalEditSemanticAcceptanceDigest(records)
	if err != nil {
		return FinalEditSemanticAttestation{}, err
	}
	if count != len(records) || digest != expected {
		return FinalEditSemanticAttestation{}, fmt.Errorf("%w: semantic acceptance digest differs", producterror.ErrConflict)
	}
	return FinalEditSemanticAttestation{Records: records, Digest: digest, Count: count}, nil
}

func decodeStoredFinalEditSemanticAcceptancePayload(value any) ([]StoredFinalEditSemanticAcceptance, error) {
	if value == nil {
		return nil, nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("%w: semantic acceptance payload is invalid", producterror.ErrConflict)
	}
	var records []StoredFinalEditSemanticAcceptance
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&records); err != nil {
		return nil, fmt.Errorf("%w: semantic acceptance payload is invalid", producterror.ErrConflict)
	}
	if decoder.More() {
		return nil, fmt.Errorf("%w: semantic acceptance payload is invalid", producterror.ErrConflict)
	}
	out := make([]StoredFinalEditSemanticAcceptance, 0, len(records))
	seen := map[int]bool{}
	seenFinal := map[int]bool{}
	for _, record := range records {
		normalized, err := normalizeStoredFinalEditSemanticAcceptance(record)
		if err != nil {
			return nil, fmt.Errorf("%w: semantic acceptance payload is invalid", producterror.ErrConflict)
		}
		if seen[normalized.ParagraphOrdinal] {
			return nil, fmt.Errorf("%w: duplicate semantic acceptance payload", producterror.ErrConflict)
		}
		seen[normalized.ParagraphOrdinal] = true
		if seenFinal[normalized.FinalParagraphOrdinal] {
			return nil, fmt.Errorf("%w: duplicate semantic acceptance final paragraph payload", producterror.ErrConflict)
		}
		seenFinal[normalized.FinalParagraphOrdinal] = true
		if !semanticVerdictMatchesFinal(normalized) {
			return nil, fmt.Errorf("%w: semantic acceptance payload is unresolved", producterror.ErrConflict)
		}
		out = append(out, normalized)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ParagraphOrdinal < out[j].ParagraphOrdinal })
	return out, nil
}
