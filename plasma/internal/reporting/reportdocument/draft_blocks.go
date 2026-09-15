package reportdocument

import (
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"strings"
)

func BuildDraftReportBlocks(
	versionID string,
	missionID string,
	title string,
	producer ledger.Producer,
	records ScopeRecords,
) ([]ReportBlock, error) {
	blockID := func(label string, index int) string {
		suffix := strings.TrimPrefix(versionID, "rvn_")
		if index == 0 {
			return "blk_" + suffix + "_" + label
		}
		return fmt.Sprintf("blk_%s_%s_%03d", suffix, label, index)
	}
	childBlocks := []ReportBlock{}
	addBlock := func(blockType string, order int, content any, refs ReportBlockSourceRefs) error {
		id := blockID(blockType, len(childBlocks)+1)
		if err := validateID("blk_", id); err != nil {
			return err
		}
		encoded, err := json.Marshal(content)
		if err != nil {
			return err
		}
		childBlocks = append(childBlocks, ReportBlock{
			SchemaVersion:   ReportBlockSchemaVersion,
			ObjectKind:      ReportBlockObjectKind,
			BlockID:         id,
			ReportVersionID: versionID,
			MissionID:       missionID,
			BlockType:       blockType,
			ParentBlockID:   blockID("root", 0),
			Order:           order,
			Content:         encoded,
			SourceRefs:      refs,
			Authorship:      ReportBlockAuthorship{Mode: "generated", Producer: producer},
			Approval:        Approval{State: "pending", Required: true},
		})
		return nil
	}

	if err := addBlock("title", 10, map[string]string{"text": title}, ReportBlockSourceRefs{}); err != nil {
		return nil, err
	}
	summary := fmt.Sprintf("Draft generated from %d claim(s), %d evidence record(s), %d question(s), and %d option(s).",
		len(records.Claims), len(records.Evidence), len(records.Questions), len(records.Options))
	if err := addBlock("abstract", 20, map[string]string{"text": summary}, ReportBlockSourceRefs{}); err != nil {
		return nil, err
	}
	order := 30
	if len(records.Claims) > 0 {
		if err := addBlock("heading", order, map[string]any{"level": 2, "text": "Claims"}, ReportBlockSourceRefs{}); err != nil {
			return nil, err
		}
		order += 10
		for _, claim := range records.Claims {
			refs := refsForClaim(claim, records.Evidence)
			if len(refs.EvidenceIDs) == 0 {
				return nil, fmt.Errorf("%w: generated claim block requires evidence links", producterror.ErrInvalidInput)
			}
			if err := addBlock("claim", order, map[string]string{
				"claim_id":      claim.ClaimID,
				"rendered_text": claim.Text,
			}, refs); err != nil {
				return nil, err
			}
			order += 10
		}
	}
	if len(records.Evidence) > 0 {
		if err := addBlock("heading", order, map[string]any{"level": 2, "text": "Evidence"}, ReportBlockSourceRefs{}); err != nil {
			return nil, err
		}
		order += 10
		for _, evidence := range records.Evidence {
			if err := addBlock("evidence_summary", order, map[string]any{
				"evidence_ids": []string{evidence.EvidenceID},
				"text":         evidence.Summary,
			}, refsForEvidence(evidence)); err != nil {
				return nil, err
			}
			order += 10
		}
	}
	if len(records.Questions) > 0 {
		if err := addBlock("heading", order, map[string]any{"level": 2, "text": "Open Questions"}, ReportBlockSourceRefs{}); err != nil {
			return nil, err
		}
		order += 10
		for _, question := range records.Questions {
			if err := addBlock("unresolved_question", order, map[string]string{
				"question_id": question.QuestionID,
				"text":        question.Text,
			}, ReportBlockSourceRefs{
				QuestionIDs: []string{question.QuestionID},
				EvidenceIDs: append([]string(nil), question.RelatedEvidenceIDs...),
				ClaimIDs:    append([]string(nil), question.RelatedClaimIDs...),
			}); err != nil {
				return nil, err
			}
			order += 10
		}
	}
	if len(records.Options) > 0 {
		if err := addBlock("heading", order, map[string]any{"level": 2, "text": "Options"}, ReportBlockSourceRefs{}); err != nil {
			return nil, err
		}
		order += 10
		for _, option := range records.Options {
			if err := addBlock("option", order, map[string]string{
				"option_id": option.OptionID,
				"text":      option.Title,
			}, ReportBlockSourceRefs{
				OptionIDs: []string{option.OptionID},
				ClaimIDs:  append([]string(nil), option.SupportingClaimIDs...),
			}); err != nil {
				return nil, err
			}
			order += 10
		}
	}

	childIDs := make([]string, 0, len(childBlocks))
	for _, block := range childBlocks {
		childIDs = append(childIDs, block.BlockID)
	}
	rootID := blockID("root", 0)
	rootContent, err := json.Marshal(map[string]any{"children": childIDs})
	if err != nil {
		return nil, err
	}
	root := ReportBlock{
		SchemaVersion:   ReportBlockSchemaVersion,
		ObjectKind:      ReportBlockObjectKind,
		BlockID:         rootID,
		ReportVersionID: versionID,
		MissionID:       missionID,
		BlockType:       "document",
		Order:           0,
		Content:         rootContent,
		SourceRefs:      ReportBlockSourceRefs{},
		Authorship:      ReportBlockAuthorship{Mode: "generated", Producer: producer},
		Approval:        Approval{State: "pending", Required: true},
	}
	return append([]ReportBlock{root}, childBlocks...), nil
}
