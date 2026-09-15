package reportdocument

import (
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"strings"
)

func ValidateReportBlockRefs(refs ReportBlockSourceRefs, records ScopeRecords) error {
	claimIDs := map[string]struct{}{}
	for _, claim := range records.Claims {
		claimIDs[claim.ClaimID] = struct{}{}
	}
	evidenceIDs := map[string]struct{}{}
	snapshotIDs := map[string]struct{}{}
	for _, evidence := range records.Evidence {
		evidenceIDs[evidence.EvidenceID] = struct{}{}
		for _, snapshotRef := range evidence.SnapshotRefs {
			snapshotIDs[snapshotRef.SnapshotID] = struct{}{}
		}
	}
	questionIDs := map[string]struct{}{}
	for _, question := range records.Questions {
		questionIDs[question.QuestionID] = struct{}{}
	}
	optionIDs := map[string]struct{}{}
	for _, option := range records.Options {
		optionIDs[option.OptionID] = struct{}{}
	}
	if err := requireRefsInScope("claim", refs.ClaimIDs, claimIDs); err != nil {
		return err
	}
	if err := requireRefsInScope("evidence", refs.EvidenceIDs, evidenceIDs); err != nil {
		return err
	}
	if err := requireRefsInScope("source snapshot", refs.SnapshotIDs, snapshotIDs); err != nil {
		return err
	}
	if err := requireRefsInScope("question", refs.QuestionIDs, questionIDs); err != nil {
		return err
	}
	if err := requireRefsInScope("option", refs.OptionIDs, optionIDs); err != nil {
		return err
	}
	return nil
}

func requireRefsInScope(kind string, refs []string, allowed map[string]struct{}) error {
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		if _, ok := allowed[ref]; !ok {
			return fmt.Errorf("%w: report block references out-of-scope %s %q", producterror.ErrInvalidInput, kind, ref)
		}
	}
	return nil
}

func validateReportBlock(block ReportBlock) error {
	switch block.BlockType {
	case "document":
		return nil
	case "title", "abstract", "paragraph", "quote", "evidence_summary", "unresolved_question", "option":
		_, err := blockText(block)
		return err
	case "heading":
		_, err := blockHeading(block)
		return err
	case "bullet_list":
		_, err := blockListItems(block)
		return err
	case "claim":
		var content struct {
			ClaimID      string `json:"claim_id"`
			RenderedText string `json:"rendered_text"`
		}
		if err := json.Unmarshal(block.Content, &content); err != nil {
			return err
		}
		if strings.TrimSpace(content.RenderedText) == "" {
			return fmt.Errorf("%w: claim block requires rendered text", producterror.ErrInvalidInput)
		}
		return nil
	default:
		return fmt.Errorf("%w: unsupported report block type", producterror.ErrInvalidInput)
	}
}

func blockText(block ReportBlock) (string, error) {
	var content struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(block.Content, &content); err != nil {
		return "", err
	}
	text := strings.TrimSpace(content.Text)
	if text == "" {
		return "", fmt.Errorf("%w: report block text is required", producterror.ErrInvalidInput)
	}
	return text, nil
}

func blockListItems(block ReportBlock) ([]string, error) {
	var content struct {
		Items []string `json:"items"`
	}
	if err := json.Unmarshal(block.Content, &content); err != nil {
		return nil, err
	}
	items := make([]string, 0, len(content.Items))
	for _, item := range content.Items {
		item = strings.TrimSpace(item)
		if item != "" {
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("%w: report list items are required", producterror.ErrInvalidInput)
	}
	return items, nil
}

type reportHeading struct {
	Level int
	Text  string
}

func blockHeading(block ReportBlock) (reportHeading, error) {
	var content reportHeading
	if err := json.Unmarshal(block.Content, &content); err != nil {
		return reportHeading{}, err
	}
	if content.Level <= 0 {
		content.Level = 2
	}
	if content.Level > 6 {
		content.Level = 6
	}
	content.Text = strings.TrimSpace(content.Text)
	if content.Text == "" {
		return reportHeading{}, fmt.Errorf("%w: report heading text is required", producterror.ErrInvalidInput)
	}
	return content, nil
}

var allowedReportBlockTypes = map[string]bool{
	"document":            true,
	"title":               true,
	"abstract":            true,
	"heading":             true,
	"paragraph":           true,
	"bullet_list":         true,
	"quote":               true,
	"claim":               true,
	"evidence_summary":    true,
	"unresolved_question": true,
	"option":              true,
}
