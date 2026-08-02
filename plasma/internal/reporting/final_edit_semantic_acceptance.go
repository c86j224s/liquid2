package reporting

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

const (
	FinalEditSemanticAcceptedEquivalent     = "accepted_equivalent"
	FinalEditSemanticRevertedToReader       = "reverted_to_reader"
	FinalEditSemanticRejectedRevertToReader = "rejected_revert_to_reader"
	FinalEditSemanticRepairedByGate         = "repaired_by_gate"
)

type FinalEditSemanticAcceptance struct {
	ParagraphOrdinal      int    `json:"paragraph_ordinal"`
	FinalParagraphOrdinal int    `json:"final_paragraph_ordinal"`
	Verdict               string `json:"verdict"`
}

type StoredFinalEditSemanticAcceptance struct {
	ParagraphOrdinal      int    `json:"paragraph_ordinal"`
	FinalParagraphOrdinal int    `json:"final_paragraph_ordinal"`
	Verdict               string `json:"verdict"`
	ReaderSHA256          string `json:"reader_sha256"`
	StyleSHA256           string `json:"style_sha256"`
	FinalSHA256           string `json:"final_sha256"`
}

type FinalEditSemanticAttestation struct {
	Records []StoredFinalEditSemanticAcceptance `json:"records,omitempty"`
	Digest  string                              `json:"digest,omitempty"`
	Count   int                                 `json:"count"`
}

type FinalEditSemanticComparisonParagraph struct {
	ParagraphOrdinal int    `json:"paragraph_ordinal"`
	ReaderSHA256     string `json:"reader_sha256"`
	StyleSHA256      string `json:"style_sha256"`
	ReaderText       string `json:"reader_text"`
	StyleText        string `json:"style_text"`
}

func FinalEditSemanticComparison(ctx context.Context, store FinalEditStageStore, stageBinding FinalEditStageBinding, finalMarkdown string) ([]FinalEditSemanticComparisonParagraph, error) {
	stageBinding = normalizeFinalEditStageBinding(stageBinding)
	events, err := store.ListEvents(ctx, stageBinding.MissionID)
	if err != nil {
		return nil, err
	}
	style, ok, err := finalEditStyleSubmissionForGate(ctx, store, events, stageBinding)
	if err != nil || !ok || !style.Changed {
		return nil, err
	}
	readerBlocks := markdownNonEmptyBlocks(string(style.SourceArtifact.Content))
	styleBlocks := markdownNonEmptyBlocks(string(style.Artifact.Content))
	if len(readerBlocks) != len(styleBlocks) {
		return nil, fmt.Errorf("%w: semantic comparison paragraph lineage is incomplete", app.ErrConflict)
	}
	out := []FinalEditSemanticComparisonParagraph{}
	for i := range readerBlocks {
		readerHash := contentSHA256([]byte(readerBlocks[i]))
		styleHash := contentSHA256([]byte(styleBlocks[i]))
		if readerHash == styleHash {
			continue
		}
		out = append(out, FinalEditSemanticComparisonParagraph{
			ParagraphOrdinal: i + 1,
			ReaderSHA256:     readerHash,
			StyleSHA256:      styleHash,
			ReaderText:       readerBlocks[i],
			StyleText:        styleBlocks[i],
		})
	}
	return out, nil
}

func ValidateFinalEditSemanticAcceptance(ctx context.Context, store FinalEditStageStore, stageBinding FinalEditStageBinding, finalMarkdown string, reviews []FinalEditSemanticAcceptance) (FinalEditSemanticAttestation, error) {
	stageBinding = normalizeFinalEditStageBinding(stageBinding)
	if stageBinding.Stage != FinalEditStageGate && stageBinding.Stage != FinalEditStageStyleSemanticValidation {
		return FinalEditSemanticAttestation{}, fmt.Errorf("%w: semantic acceptance requires a semantic validation stage", app.ErrInvalidInput)
	}
	events, err := store.ListEvents(ctx, stageBinding.MissionID)
	if err != nil {
		return FinalEditSemanticAttestation{}, err
	}
	style, ok, err := finalEditStyleSubmissionForGate(ctx, store, events, stageBinding)
	if err != nil {
		return FinalEditSemanticAttestation{}, err
	}
	if !ok || !style.Changed {
		if len(reviews) != 0 {
			return FinalEditSemanticAttestation{}, fmt.Errorf("%w: semantic acceptance review is foreign to unchanged style lineage", app.ErrConflict)
		}
		return FinalEditSemanticAttestation{}, nil
	}
	readerBlocks := markdownNonEmptyBlocks(string(style.SourceArtifact.Content))
	styleBlocks := markdownNonEmptyBlocks(string(style.Artifact.Content))
	finalBlocks := markdownNonEmptyBlocks(finalMarkdown)
	if len(readerBlocks) != len(styleBlocks) {
		return FinalEditSemanticAttestation{}, fmt.Errorf("%w: semantic acceptance paragraph lineage is incomplete", app.ErrConflict)
	}
	expected := map[int]StoredFinalEditSemanticAcceptance{}
	for i := range readerBlocks {
		readerHash := contentSHA256([]byte(readerBlocks[i]))
		styleHash := contentSHA256([]byte(styleBlocks[i]))
		if readerHash == styleHash {
			continue
		}
		expected[i+1] = StoredFinalEditSemanticAcceptance{
			ParagraphOrdinal: i + 1,
			ReaderSHA256:     readerHash,
			StyleSHA256:      styleHash,
		}
	}
	if len(reviews) != len(expected) {
		return FinalEditSemanticAttestation{}, fmt.Errorf("%w: semantic acceptance review count differs from changed paragraphs", app.ErrConflict)
	}
	records := make([]StoredFinalEditSemanticAcceptance, 0, len(reviews))
	seenSource := map[int]bool{}
	seenFinal := map[int]bool{}
	for _, review := range reviews {
		normalized, err := normalizeFinalEditSemanticAcceptanceInput(review)
		if err != nil {
			return FinalEditSemanticAttestation{}, err
		}
		if seenSource[normalized.ParagraphOrdinal] {
			return FinalEditSemanticAttestation{}, fmt.Errorf("%w: duplicate semantic acceptance paragraph review", app.ErrConflict)
		}
		seenSource[normalized.ParagraphOrdinal] = true
		if normalized.FinalParagraphOrdinal > len(finalBlocks) {
			return FinalEditSemanticAttestation{}, fmt.Errorf("%w: semantic acceptance final paragraph is outside the final manuscript", app.ErrConflict)
		}
		if seenFinal[normalized.FinalParagraphOrdinal] {
			return FinalEditSemanticAttestation{}, fmt.Errorf("%w: duplicate semantic acceptance final paragraph review", app.ErrConflict)
		}
		seenFinal[normalized.FinalParagraphOrdinal] = true
		want, ok := expected[normalized.ParagraphOrdinal]
		if !ok {
			return FinalEditSemanticAttestation{}, fmt.Errorf("%w: semantic acceptance review does not match durable lineage", app.ErrConflict)
		}
		finalBlock := strings.TrimSpace(finalBlocks[normalized.FinalParagraphOrdinal-1])
		if finalBlock == "" {
			return FinalEditSemanticAttestation{}, fmt.Errorf("%w: semantic acceptance final paragraph is empty", app.ErrConflict)
		}
		stored := StoredFinalEditSemanticAcceptance{
			ParagraphOrdinal:      normalized.ParagraphOrdinal,
			FinalParagraphOrdinal: normalized.FinalParagraphOrdinal,
			Verdict:               normalized.Verdict,
			ReaderSHA256:          want.ReaderSHA256,
			StyleSHA256:           want.StyleSHA256,
			FinalSHA256:           contentSHA256([]byte(finalBlock)),
		}
		if !semanticVerdictMatchesFinal(stored) {
			return FinalEditSemanticAttestation{}, fmt.Errorf("%w: semantic acceptance verdict is unresolved", app.ErrConflict)
		}
		records = append(records, stored)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].ParagraphOrdinal < records[j].ParagraphOrdinal })
	digest, err := finalEditSemanticAcceptanceDigest(records)
	if err != nil {
		return FinalEditSemanticAttestation{}, err
	}
	return FinalEditSemanticAttestation{Records: records, Digest: digest, Count: len(records)}, nil
}

func validateStoredFinalEditSemanticAcceptanceAgainstLineage(ctx context.Context, store FinalEditStageStore, stageBinding FinalEditStageBinding, finalMarkdown string, stored FinalEditSemanticAttestation) error {
	if stored.Count == 0 && len(stored.Records) == 0 && strings.TrimSpace(stored.Digest) == "" {
		return nil
	}
	reviews := make([]FinalEditSemanticAcceptance, 0, len(stored.Records))
	for _, record := range stored.Records {
		reviews = append(reviews, FinalEditSemanticAcceptance{
			ParagraphOrdinal:      record.ParagraphOrdinal,
			FinalParagraphOrdinal: record.FinalParagraphOrdinal,
			Verdict:               record.Verdict,
		})
	}
	derived, err := ValidateFinalEditSemanticAcceptance(ctx, store, stageBinding, finalMarkdown, reviews)
	if err != nil {
		return err
	}
	if derived.Count != stored.Count || derived.Digest != stored.Digest || !equalStoredFinalEditSemanticAcceptance(derived.Records, stored.Records) {
		return fmt.Errorf("%w: semantic acceptance does not match durable lineage", app.ErrConflict)
	}
	return nil
}

type finalEditStyleLineage struct {
	FinalEditStageResult
	SourceArtifact app.RawArtifact
}

func finalEditStyleSubmissionForGate(ctx context.Context, store FinalEditStageStore, events []app.LedgerEvent, gateBinding FinalEditStageBinding) (finalEditStyleLineage, bool, error) {
	plan, err := finalEditStagePlanForBinding(events, gateBinding)
	if err != nil {
		return finalEditStyleLineage{}, false, err
	}
	count := 0
	var found finalEditStyleLineage
	for _, event := range events {
		if event.EventType != FinalEditStyleSubmittedEventType {
			continue
		}
		binding, ok := finalEditStageBindingFromSubmittedEventForPipeline(event, plan.Pipeline)
		if !ok ||
			binding.MissionID != gateBinding.MissionID ||
			binding.PendingEventID != gateBinding.PendingEventID ||
			binding.PlanEventID != gateBinding.PlanEventID {
			continue
		}
		result, err := finalEditStageResultFromEvent(ctx, store, binding, event, true)
		if err != nil {
			return finalEditStyleLineage{}, false, err
		}
		if result.Artifact.ArtifactID != gateBinding.SourceArtifactID {
			continue
		}
		source, err := store.GetRawArtifact(ctx, binding.SourceArtifactID)
		if err != nil {
			return finalEditStyleLineage{}, false, err
		}
		found = finalEditStyleLineage{FinalEditStageResult: result, SourceArtifact: source}
		count++
	}
	if count > 1 {
		return finalEditStyleLineage{}, false, fmt.Errorf("%w: multiple style submissions match corrective gate lineage", app.ErrConflict)
	}
	return found, count == 1, nil
}

func BuildFinalEditStyleSemanticValidation(ctx context.Context, store FinalEditStageStore, stageBinding FinalEditStageBinding, reviews []FinalEditSemanticAcceptance) (string, FinalEditSemanticAttestation, error) {
	stageBinding = normalizeFinalEditStageBinding(stageBinding)
	if stageBinding.Stage != FinalEditStageStyleSemanticValidation {
		return "", FinalEditSemanticAttestation{}, fmt.Errorf("%w: style semantic validation requires its own stage", app.ErrInvalidInput)
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
		return "", FinalEditSemanticAttestation{}, fmt.Errorf("%w: style semantic validation requires style lineage", app.ErrConflict)
	}
	resolved, normalizedReviews, err := resolveFinalEditStyleMarkdown(string(style.SourceArtifact.Content), string(style.Artifact.Content), reviews)
	if err != nil {
		return "", FinalEditSemanticAttestation{}, err
	}
	attestation, err := ValidateFinalEditSemanticAcceptance(ctx, store, stageBinding, resolved, normalizedReviews)
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
		return "", nil, fmt.Errorf("%w: semantic validation paragraph lineage is incomplete", app.ErrConflict)
	}
	expected := map[int]bool{}
	for i := range readerBlocks {
		if contentSHA256([]byte(readerBlocks[i].Text)) != contentSHA256([]byte(styleBlocks[i].Text)) {
			expected[i+1] = true
		}
	}
	if len(reviews) != len(expected) {
		return "", nil, fmt.Errorf("%w: semantic validation review count differs from changed paragraphs", app.ErrConflict)
	}
	byOrdinal := map[int]FinalEditSemanticAcceptance{}
	for _, review := range reviews {
		review.Verdict = strings.TrimSpace(review.Verdict)
		if review.ParagraphOrdinal <= 0 || review.FinalParagraphOrdinal != 0 {
			return "", nil, fmt.Errorf("%w: semantic validation review is incomplete", app.ErrInvalidInput)
		}
		if !expected[review.ParagraphOrdinal] || byOrdinal[review.ParagraphOrdinal].ParagraphOrdinal != 0 {
			return "", nil, fmt.Errorf("%w: semantic validation review does not match durable lineage", app.ErrConflict)
		}
		switch review.Verdict {
		case FinalEditSemanticAcceptedEquivalent, FinalEditSemanticRejectedRevertToReader:
		default:
			return "", nil, fmt.Errorf("%w: unsupported semantic validation verdict", app.ErrInvalidInput)
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

type markdownBlockSpan struct {
	Start int
	End   int
	Text  string
}

func markdownNonEmptyBlockSpans(text string) []markdownBlockSpan {
	blocks := []markdownBlockSpan{}
	start := -1
	lastNonSpace := -1
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			j := i + 1
			for j < len(text) && (text[j] == ' ' || text[j] == '\t' || text[j] == '\r') {
				j++
			}
			if j < len(text) && text[j] == '\n' {
				if start >= 0 {
					blocks = append(blocks, markdownBlockSpan{Start: start, End: lastNonSpace + 1, Text: text[start : lastNonSpace+1]})
					start = -1
					lastNonSpace = -1
				}
				i = j
				continue
			}
		}
		if !isASCIISpace(text[i]) {
			if start < 0 {
				start = i
			}
			lastNonSpace = i
		}
	}
	if start >= 0 {
		blocks = append(blocks, markdownBlockSpan{Start: start, End: lastNonSpace + 1, Text: text[start : lastNonSpace+1]})
	}
	return blocks
}

func isASCIISpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func normalizeFinalEditSemanticAcceptanceInput(review FinalEditSemanticAcceptance) (FinalEditSemanticAcceptance, error) {
	review.Verdict = strings.TrimSpace(review.Verdict)
	if review.ParagraphOrdinal <= 0 || review.FinalParagraphOrdinal <= 0 {
		return FinalEditSemanticAcceptance{}, fmt.Errorf("%w: semantic acceptance review is incomplete", app.ErrInvalidInput)
	}
	switch review.Verdict {
	case FinalEditSemanticAcceptedEquivalent, FinalEditSemanticRevertedToReader, FinalEditSemanticRejectedRevertToReader, FinalEditSemanticRepairedByGate:
	default:
		return FinalEditSemanticAcceptance{}, fmt.Errorf("%w: unsupported semantic acceptance verdict", app.ErrInvalidInput)
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
		return StoredFinalEditSemanticAcceptance{}, fmt.Errorf("%w: semantic acceptance payload is incomplete", app.ErrInvalidInput)
	}
	switch review.Verdict {
	case FinalEditSemanticAcceptedEquivalent, FinalEditSemanticRevertedToReader, FinalEditSemanticRejectedRevertToReader, FinalEditSemanticRepairedByGate:
	default:
		return StoredFinalEditSemanticAcceptance{}, fmt.Errorf("%w: unsupported semantic acceptance verdict", app.ErrInvalidInput)
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
