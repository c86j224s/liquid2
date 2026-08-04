package reporting

import (
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

// FinalEditEvidenceGatePassage는 final edit evidence gate가 검토할 원문 passage와 인용 위치다.
type FinalEditEvidenceGatePassage struct {
	BlockOrdinal    int    `json:"block_ordinal"`
	StatementSHA256 string `json:"statement_sha256"`
	Text            string `json:"text"`
}

// FinalEditEvidenceGatePassages는 final edit evidence gate가 비교할 passage 목록을 만든다.
func FinalEditEvidenceGatePassages(markdown string) ([]FinalEditEvidenceGatePassage, error) {
	blocks := markdownNonEmptyBlockSpans(markdown)
	if len(blocks) == 0 {
		return nil, fmt.Errorf("%w: evidence gate report has no judgeable passages", app.ErrConflict)
	}
	passages := make([]FinalEditEvidenceGatePassage, 0, len(blocks))
	for i, block := range blocks {
		passages = append(passages, FinalEditEvidenceGatePassage{
			BlockOrdinal:    i + 1,
			StatementSHA256: contentSHA256([]byte(block.Text)),
			Text:            block.Text,
		})
	}
	return passages, nil
}

func validateFinalEditEvidenceGateFindingStatementsInSource(markdown string, findings []StoredFinalEditGateFinding) error {
	if len(findings) == 0 {
		return nil
	}
	passages, err := FinalEditEvidenceGatePassages(markdown)
	if err != nil {
		return err
	}
	allowed := make(map[string]bool, len(passages))
	for _, passage := range passages {
		allowed[passage.StatementSHA256] = true
	}
	for _, finding := range findings {
		if !allowed[finding.StatementSHA256] {
			return fmt.Errorf("%w: evidence gate statement hash is not present in the bound source artifact", app.ErrInvalidInput)
		}
	}
	return nil
}
