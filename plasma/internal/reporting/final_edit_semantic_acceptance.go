package reporting

import (
	"context"
	"fmt"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"sort"
	"strings"
)

const (
	FinalEditSemanticAcceptedEquivalent     = "accepted_equivalent"
	FinalEditSemanticRevertedToReader       = "reverted_to_reader"
	FinalEditSemanticRejectedRevertToReader = "rejected_revert_to_reader"
	FinalEditSemanticRepairedByGate         = "repaired_by_gate"
)

// FinalEditSemanticAcceptance는 문체 편집 문단이 의미상 수용 가능한지에 대한 검토 결과다.
type FinalEditSemanticAcceptance struct {
	ParagraphOrdinal      int    `json:"paragraph_ordinal"`
	FinalParagraphOrdinal int    `json:"final_paragraph_ordinal"`
	Verdict               string `json:"verdict"`
}

// StoredFinalEditSemanticAcceptance는 의미 검토 결과와 비교 대상 문단 해시를 함께 저장한 record다.
type StoredFinalEditSemanticAcceptance struct {
	ParagraphOrdinal      int    `json:"paragraph_ordinal"`
	FinalParagraphOrdinal int    `json:"final_paragraph_ordinal"`
	Verdict               string `json:"verdict"`
	ReaderSHA256          string `json:"reader_sha256"`
	StyleSHA256           string `json:"style_sha256"`
	FinalSHA256           string `json:"final_sha256"`
}

// FinalEditSemanticAttestation는 의미 검토 record 묶음과 digest를 담는 제출 증명이다.
type FinalEditSemanticAttestation struct {
	Records []StoredFinalEditSemanticAcceptance `json:"records,omitempty"`
	Digest  string                              `json:"digest,omitempty"`
	Count   int                                 `json:"count"`
}

// FinalEditSemanticComparisonParagraph는 reader/style 문단을 한 쌍으로 비교하기 위한 입력이다.
type FinalEditSemanticComparisonParagraph struct {
	ParagraphOrdinal int    `json:"paragraph_ordinal"`
	ReaderSHA256     string `json:"reader_sha256"`
	StyleSHA256      string `json:"style_sha256"`
	ReaderText       string `json:"reader_text"`
	StyleText        string `json:"style_text"`
}

// FinalEditSemanticComparison는 reader/style artifact를 비교해 검증 대상 문단을 만든다.
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
		return nil, fmt.Errorf("%w: semantic comparison paragraph lineage is incomplete", producterror.ErrConflict)
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

// validateFinalEditSemanticAcceptance는 보고서 생성 파이프라인 계약을 검사한다. 제품 상태를 변경하지 않는 순수 검증 경계다.
func validateFinalEditSemanticAcceptance(ctx context.Context, store FinalEditStageStore, stageBinding FinalEditStageBinding, finalMarkdown string, reviews []FinalEditSemanticAcceptance) (FinalEditSemanticAttestation, error) {
	stageBinding = normalizeFinalEditStageBinding(stageBinding)
	if stageBinding.Stage != FinalEditStageGate && stageBinding.Stage != FinalEditStageStyleSemanticValidation {
		return FinalEditSemanticAttestation{}, fmt.Errorf("%w: semantic acceptance requires a semantic validation stage", producterror.ErrInvalidInput)
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
			return FinalEditSemanticAttestation{}, fmt.Errorf("%w: semantic acceptance review is foreign to unchanged style lineage", producterror.ErrConflict)
		}
		return FinalEditSemanticAttestation{}, nil
	}
	readerBlocks := markdownNonEmptyBlocks(string(style.SourceArtifact.Content))
	styleBlocks := markdownNonEmptyBlocks(string(style.Artifact.Content))
	finalBlocks := markdownNonEmptyBlocks(finalMarkdown)
	if len(readerBlocks) != len(styleBlocks) {
		return FinalEditSemanticAttestation{}, fmt.Errorf("%w: semantic acceptance paragraph lineage is incomplete", producterror.ErrConflict)
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
		return FinalEditSemanticAttestation{}, fmt.Errorf("%w: semantic acceptance review count differs from changed paragraphs", producterror.ErrConflict)
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
			return FinalEditSemanticAttestation{}, fmt.Errorf("%w: duplicate semantic acceptance paragraph review", producterror.ErrConflict)
		}
		seenSource[normalized.ParagraphOrdinal] = true
		if normalized.FinalParagraphOrdinal > len(finalBlocks) {
			return FinalEditSemanticAttestation{}, fmt.Errorf("%w: semantic acceptance final paragraph is outside the final manuscript", producterror.ErrConflict)
		}
		if seenFinal[normalized.FinalParagraphOrdinal] {
			return FinalEditSemanticAttestation{}, fmt.Errorf("%w: duplicate semantic acceptance final paragraph review", producterror.ErrConflict)
		}
		seenFinal[normalized.FinalParagraphOrdinal] = true
		want, ok := expected[normalized.ParagraphOrdinal]
		if !ok {
			return FinalEditSemanticAttestation{}, fmt.Errorf("%w: semantic acceptance review does not match durable lineage", producterror.ErrConflict)
		}
		finalBlock := strings.TrimSpace(finalBlocks[normalized.FinalParagraphOrdinal-1])
		if finalBlock == "" {
			return FinalEditSemanticAttestation{}, fmt.Errorf("%w: semantic acceptance final paragraph is empty", producterror.ErrConflict)
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
			return FinalEditSemanticAttestation{}, fmt.Errorf("%w: semantic acceptance verdict is unresolved", producterror.ErrConflict)
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
	derived, err := validateFinalEditSemanticAcceptance(ctx, store, stageBinding, finalMarkdown, reviews)
	if err != nil {
		return err
	}
	if derived.Count != stored.Count || derived.Digest != stored.Digest || !equalStoredFinalEditSemanticAcceptance(derived.Records, stored.Records) {
		return fmt.Errorf("%w: semantic acceptance does not match durable lineage", producterror.ErrConflict)
	}
	return nil
}

type finalEditStyleLineage struct {
	FinalEditStageResult
	SourceArtifact artifactcontract.Raw
}

func finalEditStyleSubmissionForGate(ctx context.Context, store FinalEditStageStore, events []ledger.Event, gateBinding FinalEditStageBinding) (finalEditStyleLineage, bool, error) {
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
		return finalEditStyleLineage{}, false, fmt.Errorf("%w: multiple style submissions match corrective gate lineage", producterror.ErrConflict)
	}
	return found, count == 1, nil
}

type markdownBlockSpan struct {
	Start int
	End   int
	Text  string
}
