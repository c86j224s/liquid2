package reportdocument

import (
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/researchrecords"
	"strings"
)

func BuildReportBlocksFromDraftInputs(
	versionID string,
	missionID string,
	producer ledger.Producer,
	inputs []ReportBlockDraftInput,
) ([]ReportBlock, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("%w: report draft requires blocks", producterror.ErrInvalidInput)
	}
	blockID := func(label string, index int) string {
		suffix := strings.TrimPrefix(versionID, "rvn_")
		if index == 0 {
			return "blk_" + suffix + "_" + label
		}
		return fmt.Sprintf("blk_%s_%s_%03d", suffix, label, index)
	}
	childBlocks := make([]ReportBlock, 0, len(inputs))
	for index, input := range inputs {
		blockType := strings.TrimSpace(input.BlockType)
		if !allowedReportBlockTypes[blockType] || blockType == "document" {
			return nil, fmt.Errorf("%w: unsupported report block type", producterror.ErrInvalidInput)
		}
		content := append(json.RawMessage(nil), input.Content...)
		if len(content) == 0 {
			content = json.RawMessage(`{}`)
		}
		block := ReportBlock{
			SchemaVersion:   ReportBlockSchemaVersion,
			ObjectKind:      ReportBlockObjectKind,
			BlockID:         blockID(blockType, index+1),
			ReportVersionID: versionID,
			MissionID:       missionID,
			BlockType:       blockType,
			ParentBlockID:   blockID("root", 0),
			Order:           (index + 1) * 10,
			Content:         content,
			SourceRefs:      input.SourceRefs,
			Authorship:      ReportBlockAuthorship{Mode: "generated", Producer: producer},
			Approval:        Approval{State: "pending", Required: true},
		}
		if err := validateReportBlock(block); err != nil {
			return nil, err
		}
		childBlocks = append(childBlocks, block)
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

func refsForClaim(claim researchrecords.ClaimRecord, evidence []researchrecords.EvidenceRecord) ReportBlockSourceRefs {
	refs := ReportBlockSourceRefs{ClaimIDs: []string{claim.ClaimID}}
	evidenceIDs := append(append([]string{}, claim.SupportingEvidenceIDs...), claim.OpposingEvidenceIDs...)
	for _, evidenceID := range evidenceIDs {
		addUnique(&refs.EvidenceIDs, evidenceID)
		for _, record := range evidence {
			if record.EvidenceID == evidenceID {
				for _, snapshotRef := range record.SnapshotRefs {
					addUnique(&refs.SnapshotIDs, snapshotRef.SnapshotID)
				}
			}
		}
	}
	return refs
}

func refsForEvidence(evidence researchrecords.EvidenceRecord) ReportBlockSourceRefs {
	refs := ReportBlockSourceRefs{EvidenceIDs: []string{evidence.EvidenceID}}
	for _, snapshotRef := range evidence.SnapshotRefs {
		addUnique(&refs.SnapshotIDs, snapshotRef.SnapshotID)
	}
	return refs
}

func RenderReportExport(version ReportVersion, blocks []ReportBlock, target string) ([]byte, string, string, error) {
	switch target {
	case ReportExportTargetJSONAST:
		export := ReportASTExport{
			SchemaVersion: "plasma.report_ast_export.v1",
			ObjectKind:    "report_ast_export",
			Version:       version,
			Blocks:        blocks,
		}
		content, err := json.MarshalIndent(export, "", "  ")
		if err != nil {
			return nil, "", "", err
		}
		content = append(content, '\n')
		return content, "application/json", version.ReportVersionID + ".json", nil
	case ReportExportTargetMarkdown:
		content, err := renderReportMarkdown(blocks)
		if err != nil {
			return nil, "", "", err
		}
		return content, "text/markdown", version.ReportVersionID + ".md", nil
	case ReportExportTargetHTML:
		content, err := renderReportHTML(blocks)
		if err != nil {
			return nil, "", "", err
		}
		return content, "text/html; charset=utf-8", version.ReportVersionID + ".html", nil
	default:
		return nil, "", "", fmt.Errorf("%w: unsupported report export target", producterror.ErrInvalidInput)
	}
}
