package reportdocument

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"strings"
)

func renderReportMarkdown(blocks []ReportBlock) ([]byte, error) {
	var out bytes.Buffer
	footnotes := newReportFootnotes()
	for _, block := range blocks {
		switch block.BlockType {
		case "document":
			continue
		case "title":
			text, err := blockText(block)
			if err != nil {
				return nil, err
			}
			out.WriteString("# " + text + "\n\n")
		case "abstract", "paragraph":
			text, err := blockText(block)
			if err != nil {
				return nil, err
			}
			out.WriteString(text + footnotes.markdownMarker(block.SourceRefs) + "\n\n")
		case "heading":
			heading, err := blockHeading(block)
			if err != nil {
				return nil, err
			}
			out.WriteString(strings.Repeat("#", heading.Level) + " " + heading.Text + footnotes.markdownMarker(block.SourceRefs) + "\n\n")
		case "claim":
			var content struct {
				ClaimID      string `json:"claim_id"`
				RenderedText string `json:"rendered_text"`
			}
			if err := json.Unmarshal(block.Content, &content); err != nil {
				return nil, err
			}
			claimID := strings.TrimSpace(content.ClaimID)
			text := strings.TrimSpace(content.RenderedText)
			if claimID == "" || text == "" {
				return nil, fmt.Errorf("%w: claim block requires claim id and rendered text", producterror.ErrInvalidInput)
			}
			refs := block.SourceRefs
			if len(reportRefValues(refs)) == 0 {
				refs.ClaimIDs = []string{claimID}
			}
			out.WriteString("- " + text + footnotes.markdownMarker(refs) + "\n")
		case "evidence_summary":
			text, err := blockText(block)
			if err != nil {
				return nil, err
			}
			out.WriteString("- " + text + footnotes.markdownMarker(block.SourceRefs) + "\n")
		case "bullet_list":
			items, err := blockListItems(block)
			if err != nil {
				return nil, err
			}
			for index, item := range items {
				if index == len(items)-1 {
					item += footnotes.markdownMarker(block.SourceRefs)
				}
				out.WriteString("- " + item + "\n")
			}
		case "quote":
			text, err := blockText(block)
			if err != nil {
				return nil, err
			}
			lines := strings.Split(text, "\n")
			for index, line := range lines {
				if index == len(lines)-1 {
					line += footnotes.markdownMarker(block.SourceRefs)
				}
				out.WriteString("> " + line + "\n")
			}
		case "unresolved_question":
			text, err := blockText(block)
			if err != nil {
				return nil, err
			}
			out.WriteString("- " + text + footnotes.markdownMarker(block.SourceRefs) + "\n")
		case "option":
			text, err := blockText(block)
			if err != nil {
				return nil, err
			}
			out.WriteString("- " + text + footnotes.markdownMarker(block.SourceRefs) + "\n")
		default:
			return nil, fmt.Errorf("%w: unsupported report block type", producterror.ErrInvalidInput)
		}
		if block.BlockType == "claim" || block.BlockType == "evidence_summary" || block.BlockType == "bullet_list" || block.BlockType == "quote" || block.BlockType == "unresolved_question" || block.BlockType == "option" {
			out.WriteString("\n")
		}
	}
	footnotes.writeMarkdownDefinitions(&out)
	return bytes.TrimSpace(out.Bytes()), nil
}
