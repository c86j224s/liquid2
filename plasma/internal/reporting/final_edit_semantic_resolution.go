package reporting

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"strings"
)

// buildFinalEditStyleSemanticValidation는 보고서 생성 파이프라인에서 사용할 구조화된 값을 조립한다. 저장이나 외부 호출은 수행하지 않는다.
func buildFinalEditStyleSemanticValidation(ctx context.Context, store FinalEditStageStore, stageBinding FinalEditStageBinding, reviews []FinalEditSemanticAcceptance) (string, FinalEditSemanticAttestation, error) {
	stageBinding = normalizeFinalEditStageBinding(stageBinding)
	if stageBinding.Stage != FinalEditStageStyleSemanticValidation {
		return "", FinalEditSemanticAttestation{}, fmt.Errorf("%w: style semantic validation requires its own stage", producterror.ErrInvalidInput)
	}
	events, err := store.ListEvents(ctx, stageBinding.MissionID)
	if err != nil {
		return "", FinalEditSemanticAttestation{}, err
	}
	style, ok, err := finalEditStyleSubmissionForGate(ctx, store, events, stageBinding)
	if err != nil {
		return "", FinalEditSemanticAttestation{}, err
	}
	if !ok {
		return "", FinalEditSemanticAttestation{}, fmt.Errorf("%w: style semantic validation requires style lineage", producterror.ErrConflict)
	}
	resolved, normalizedReviews, err := resolveFinalEditStyleMarkdown(string(style.SourceArtifact.Content), string(style.Artifact.Content), reviews)
	if err != nil {
		return "", FinalEditSemanticAttestation{}, err
	}
	attestation, err := validateFinalEditSemanticAcceptance(ctx, store, stageBinding, resolved, normalizedReviews)
	if err != nil {
		return "", FinalEditSemanticAttestation{}, err
	}
	return resolved, attestation, nil
}

func resolveFinalEditStyleMarkdown(readerMarkdown string, styleMarkdown string, reviews []FinalEditSemanticAcceptance) (string, []FinalEditSemanticAcceptance, error) {
	if err := ValidateFinalEditStyleMarkdown(readerMarkdown, styleMarkdown); err != nil {
		return "", nil, err
	}
	readerBlocks := markdownNonEmptyBlockSpans(readerMarkdown)
	styleBlocks := markdownNonEmptyBlockSpans(styleMarkdown)
	if len(readerBlocks) != len(styleBlocks) {
		return "", nil, fmt.Errorf("%w: semantic validation paragraph lineage is incomplete", producterror.ErrConflict)
	}
	expected := map[int]bool{}
	for i := range readerBlocks {
		if contentSHA256([]byte(readerBlocks[i].Text)) != contentSHA256([]byte(styleBlocks[i].Text)) {
			expected[i+1] = true
		}
	}
	if len(reviews) != len(expected) {
		return "", nil, fmt.Errorf("%w: semantic validation review count differs from changed paragraphs", producterror.ErrConflict)
	}
	byOrdinal := map[int]FinalEditSemanticAcceptance{}
	for _, review := range reviews {
		review.Verdict = strings.TrimSpace(review.Verdict)
		if review.ParagraphOrdinal <= 0 || review.FinalParagraphOrdinal != 0 {
			return "", nil, fmt.Errorf("%w: semantic validation review is incomplete", producterror.ErrInvalidInput)
		}
		if !expected[review.ParagraphOrdinal] || byOrdinal[review.ParagraphOrdinal].ParagraphOrdinal != 0 {
			return "", nil, fmt.Errorf("%w: semantic validation review does not match durable lineage", producterror.ErrConflict)
		}
		switch review.Verdict {
		case FinalEditSemanticAcceptedEquivalent, FinalEditSemanticRejectedRevertToReader:
		default:
			return "", nil, fmt.Errorf("%w: unsupported semantic validation verdict", producterror.ErrInvalidInput)
		}
		byOrdinal[review.ParagraphOrdinal] = review
	}
	var out strings.Builder
	cursor := 0
	normalized := make([]FinalEditSemanticAcceptance, 0, len(reviews))
	for i, styleBlock := range styleBlocks {
		out.WriteString(styleMarkdown[cursor:styleBlock.Start])
		ordinal := i + 1
		review, changed := byOrdinal[ordinal]
		switch {
		case !changed:
			out.WriteString(styleMarkdown[styleBlock.Start:styleBlock.End])
		case review.Verdict == FinalEditSemanticAcceptedEquivalent:
			out.WriteString(styleBlock.Text)
			review.FinalParagraphOrdinal = ordinal
			normalized = append(normalized, review)
		case review.Verdict == FinalEditSemanticRejectedRevertToReader:
			out.WriteString(readerBlocks[i].Text)
			review.FinalParagraphOrdinal = ordinal
			normalized = append(normalized, review)
		}
		cursor = styleBlock.End
	}
	out.WriteString(styleMarkdown[cursor:])
	resolved := out.String()
	if err := ValidateFinalEditStyleMarkdown(readerMarkdown, resolved); err != nil {
		return "", nil, err
	}
	return resolved, normalized, nil
}

func normalizeFinalEditSemanticAcceptanceInput(review FinalEditSemanticAcceptance) (FinalEditSemanticAcceptance, error) {
	review.Verdict = strings.TrimSpace(review.Verdict)
	if review.ParagraphOrdinal <= 0 || review.FinalParagraphOrdinal <= 0 {
		return FinalEditSemanticAcceptance{}, fmt.Errorf("%w: semantic acceptance review is incomplete", producterror.ErrInvalidInput)
	}
	switch review.Verdict {
	case FinalEditSemanticAcceptedEquivalent, FinalEditSemanticRevertedToReader, FinalEditSemanticRejectedRevertToReader, FinalEditSemanticRepairedByGate:
	default:
		return FinalEditSemanticAcceptance{}, fmt.Errorf("%w: unsupported semantic acceptance verdict", producterror.ErrInvalidInput)
	}
	return review, nil
}

func normalizeStoredFinalEditSemanticAcceptance(review StoredFinalEditSemanticAcceptance) (StoredFinalEditSemanticAcceptance, error) {
	review.Verdict = strings.TrimSpace(review.Verdict)
	review.ReaderSHA256 = strings.TrimSpace(review.ReaderSHA256)
	review.StyleSHA256 = strings.TrimSpace(review.StyleSHA256)
	review.FinalSHA256 = strings.TrimSpace(review.FinalSHA256)
	if review.ParagraphOrdinal <= 0 || review.FinalParagraphOrdinal <= 0 ||
		!validStoredFinalEditStatementSHA256(review.ReaderSHA256) ||
		!validStoredFinalEditStatementSHA256(review.StyleSHA256) ||
		!validStoredFinalEditStatementSHA256(review.FinalSHA256) {
		return StoredFinalEditSemanticAcceptance{}, fmt.Errorf("%w: semantic acceptance payload is incomplete", producterror.ErrInvalidInput)
	}
	switch review.Verdict {
	case FinalEditSemanticAcceptedEquivalent, FinalEditSemanticRevertedToReader, FinalEditSemanticRejectedRevertToReader, FinalEditSemanticRepairedByGate:
	default:
		return StoredFinalEditSemanticAcceptance{}, fmt.Errorf("%w: unsupported semantic acceptance verdict", producterror.ErrInvalidInput)
	}
	return review, nil
}

func semanticVerdictMatchesFinal(review StoredFinalEditSemanticAcceptance) bool {
	switch review.Verdict {
	case FinalEditSemanticAcceptedEquivalent:
		return review.FinalSHA256 == review.StyleSHA256
	case FinalEditSemanticRevertedToReader, FinalEditSemanticRejectedRevertToReader:
		return review.FinalSHA256 == review.ReaderSHA256
	case FinalEditSemanticRepairedByGate:
		return review.FinalSHA256 != review.StyleSHA256 && review.FinalSHA256 != review.ReaderSHA256
	default:
		return false
	}
}

func finalEditSemanticAcceptanceDigest(records []StoredFinalEditSemanticAcceptance) (string, error) {
	if len(records) == 0 {
		return "", nil
	}
	encoded, err := json.Marshal(records)
	if err != nil {
		return "", err
	}
	return contentSHA256(encoded), nil
}

func equalStoredFinalEditSemanticAcceptance(left, right []StoredFinalEditSemanticAcceptance) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
