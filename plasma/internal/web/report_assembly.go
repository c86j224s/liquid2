package web

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func agentReportAnyJSON(value any) string {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func sectionalDraftInventoryJSON(drafts []sectionalReportDraft) string {
	items := make([]map[string]any, 0, len(drafts))
	for index, draft := range drafts {
		items = append(items, map[string]any{
			"section_index": index + 1,
			"title":         draft.Title,
			"artifact_id":   draft.ArtifactID,
			"word_count":    draft.WordCount,
		})
	}
	return agentReportAnyJSON(items)
}

func sectionalPartInventoryJSON(parts []sectionalReportPartDraft) string {
	items := make([]map[string]any, 0, len(parts))
	for index, part := range parts {
		items = append(items, map[string]any{
			"part_index":  index + 1,
			"title":       part.Title,
			"artifact_id": part.ArtifactID,
			"word_count":  part.WordCount,
		})
	}
	return agentReportAnyJSON(items)
}

func assembleSectionalPartMarkdown(part agentReportPart, drafts []sectionalReportDraft, assembly agentPartAssembly, partIndex int) string {
	var out strings.Builder
	out.WriteString(fmt.Sprintf("# Part %d. %s\n\n", partIndex+1, firstNonEmpty(part.Title, fmt.Sprintf("Part %d", partIndex+1))))
	if intro := normalizeSectionalConnectiveMarkdown(assembly.Intro); intro != "" {
		out.WriteString(intro)
		out.WriteString("\n\n")
	}
	transitions := map[int]string{}
	for _, transition := range assembly.Transitions {
		transitions[transition.AfterSectionIndex] = normalizeSectionalConnectiveMarkdown(transition.Markdown)
	}
	for index, draft := range drafts {
		out.WriteString(fmt.Sprintf("## %d.%d %s\n\n", partIndex+1, index+1, firstNonEmpty(draft.Title, fmt.Sprintf("Section %d", index+1))))
		out.WriteString(normalizeSectionalBodyMarkdown(draft.Markdown, draft.Title, partIndex+1, index+1))
		out.WriteString("\n\n")
		if transition := strings.TrimSpace(transitions[index+1]); transition != "" && index < len(drafts)-1 {
			out.WriteString(transition)
			out.WriteString("\n\n")
		}
	}
	if closing := normalizeSectionalConnectiveMarkdown(assembly.Closing); closing != "" {
		out.WriteString(closing)
		out.WriteString("\n")
	}
	return strings.TrimSpace(out.String()) + "\n"
}

var (
	markdownHeadingLineRE      = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*$`)
	markdownLeadingNumberingRE = regexp.MustCompile(`^\d+(?:\.\d+)*\.?\s+`)
)

func normalizeSectionalBodyMarkdown(markdown string, sectionTitle string, partNumber int, sectionNumber int) string {
	return normalizeSectionalMarkdown(markdown, sectionalMarkdownNormalization{
		DropFirstLeadingHeadingTexts: []string{
			sectionTitle,
			fmt.Sprintf("%d.%d %s", partNumber, sectionNumber, sectionTitle),
		},
		ConvertHeadingsBold: true,
		StripBoundaryRules:  true,
	})
}

func normalizeSectionalConnectiveMarkdown(markdown string) string {
	return normalizeSectionalMarkdown(markdown, sectionalMarkdownNormalization{
		ConvertHeadingsBold: true,
		StripBoundaryRules:  true,
	})
}

func normalizeSectionalMarkdown(markdown string, opts sectionalMarkdownNormalization) string {
	lines := strings.Split(strings.TrimSpace(markdown), "\n")
	if opts.StripBoundaryRules {
		lines = stripBoundaryHorizontalRules(lines)
	}
	out := make([]string, 0, len(lines))
	seenBody := false
	droppedFirstLeadingHeading := false
	dropFirstHeadingTexts := canonicalMarkdownHeadingTextSet(opts.DropFirstLeadingHeadingTexts)
	lastAdjacentHeadingText := ""
	fenceMarker := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if marker, ok := markdownFenceMarker(trimmed); ok {
			if fenceMarker == "" {
				fenceMarker = marker
			} else if fenceMarker == marker {
				fenceMarker = ""
			}
			out = append(out, line)
			seenBody = true
			continue
		}
		if fenceMarker != "" {
			out = append(out, line)
			seenBody = true
			continue
		}
		if trimmed == "" {
			out = append(out, line)
			continue
		}
		matches := markdownHeadingLineRE.FindStringSubmatch(trimmed)
		if len(matches) == 3 {
			text := strings.TrimSpace(matches[2])
			canonicalText := canonicalMarkdownHeadingText(text)
			if len(dropFirstHeadingTexts) > 0 && !seenBody && !droppedFirstLeadingHeading && dropFirstHeadingTexts[canonicalText] {
				droppedFirstLeadingHeading = true
				lastAdjacentHeadingText = canonicalText
				continue
			}
			if canonicalText != "" && canonicalText == lastAdjacentHeadingText {
				continue
			}
			if opts.ConvertHeadingsBold {
				out = append(out, "**"+text+"**")
				seenBody = true
				lastAdjacentHeadingText = canonicalText
				continue
			}
			level := len(matches[1])
			if opts.ForceHeadingLevel > 0 {
				level = opts.ForceHeadingLevel
			}
			if opts.MaxHeadingLevel > 0 && level > opts.MaxHeadingLevel {
				level = opts.MaxHeadingLevel
			}
			out = append(out, strings.Repeat("#", level)+" "+text)
			seenBody = true
			lastAdjacentHeadingText = canonicalText
			continue
		}
		out = append(out, line)
		seenBody = true
		lastAdjacentHeadingText = ""
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func canonicalMarkdownHeadingText(text string) string {
	text = markdownLeadingNumberingRE.ReplaceAllString(strings.TrimSpace(text), "")
	return strings.ToLower(strings.Join(strings.Fields(text), " "))
}

func canonicalMarkdownHeadingTextSet(values []string) map[string]bool {
	set := map[string]bool{}
	for _, value := range values {
		canonical := canonicalMarkdownHeadingText(value)
		if canonical != "" {
			set[canonical] = true
		}
	}
	return set
}

func markdownFenceMarker(trimmed string) (string, bool) {
	for _, marker := range []string{"```", "~~~"} {
		if strings.HasPrefix(trimmed, marker) {
			return marker, true
		}
	}
	return "", false
}

func stripBoundaryHorizontalRules(lines []string) []string {
	start := 0
	for start < len(lines) && (strings.TrimSpace(lines[start]) == "" || isMarkdownHorizontalRule(lines[start])) {
		start++
	}
	end := len(lines)
	for end > start && (strings.TrimSpace(lines[end-1]) == "" || isMarkdownHorizontalRule(lines[end-1])) {
		end--
	}
	return lines[start:end]
}

func isMarkdownHorizontalRule(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 {
		return false
	}
	for _, marker := range []rune{'-', '*', '_'} {
		allMarker := true
		count := 0
		for _, r := range trimmed {
			if r == marker {
				count++
				continue
			}
			if r != ' ' && r != '\t' {
				allMarker = false
				break
			}
		}
		if allMarker && count >= 3 {
			return true
		}
	}
	return false
}

func reportWordCount(markdown string) int {
	return len(strings.Fields(markdown))
}

func fallbackSessionID(primary string, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return strings.TrimSpace(primary)
	}
	return strings.TrimSpace(fallback)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func reportRefViolations(blocks []app.ReportBlockDraftInput, claimIDs []string, evidenceIDs []string, records recordsResponse) []reportRefViolation {
	allowedClaims := stringSet(claimIDs)
	allowedEvidence := stringSet(evidenceIDs)
	claimStates := claimStateByID(records.Claims)
	evidenceStates := evidenceStateByID(records.Evidence)
	var violations []reportRefViolation
	for index, block := range blocks {
		blockType := strings.TrimSpace(block.BlockType)
		for _, id := range block.SourceRefs.ClaimIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			if _, ok := allowedClaims[id]; !ok {
				violations = append(violations, reportRefViolation{
					ObjectKind: "claim_record",
					ObjectID:   id,
					State:      recordState(claimStates[id]),
					Reason:     "claim is not approved for this report scope",
					BlockIndex: index,
					BlockType:  blockType,
				})
			}
		}
		for _, id := range block.SourceRefs.EvidenceIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			if _, ok := allowedEvidence[id]; !ok {
				violations = append(violations, reportRefViolation{
					ObjectKind: "evidence_record",
					ObjectID:   id,
					State:      recordState(evidenceStates[id]),
					Reason:     "evidence is not approved for this report scope",
					BlockIndex: index,
					BlockType:  blockType,
				})
			}
		}
	}
	return violations
}

func stringSet(ids []string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			set[id] = struct{}{}
		}
	}
	return set
}

func claimStateByID(claims []app.ClaimRecord) map[string]string {
	states := map[string]string{}
	for _, claim := range claims {
		states[strings.TrimSpace(claim.ClaimID)] = strings.TrimSpace(claim.State)
	}
	return states
}

func evidenceStateByID(evidence []app.EvidenceRecord) map[string]string {
	states := map[string]string{}
	for _, record := range evidence {
		states[strings.TrimSpace(record.EvidenceID)] = strings.TrimSpace(record.State)
	}
	return states
}

func recordState(state string) string {
	state = strings.TrimSpace(state)
	if state == "" {
		return "missing"
	}
	return state
}

func describeReportRefViolations(violations []reportRefViolation) string {
	parts := make([]string, 0, len(violations))
	for _, violation := range violations {
		parts = append(parts, fmt.Sprintf("%s %q is %s in block %d (%s)", violation.ObjectKind, violation.ObjectID, violation.State, violation.BlockIndex, violation.BlockType))
	}
	return strings.Join(parts, "; ")
}

func agentReportRepairPrompt(title string, missionID string, toolSessionID string, rigor reportRigorProfile, plan agentReportPlan, ast agentReportAST, claimIDs []string, evidenceIDs []string, violations []reportRefViolation) string {
	return fmt.Sprintf(`You are repairing a Plasma report AST before it is saved.

The previous AST is structurally valid, but some final refs point to records that are not approved for this report scope.
This is a correctable reference mistake, not a request to discard the article.

Repair rule:
- General research tools may inspect proposed, pending, rejected, or missing material as context.
- Final AST refs/source_refs must only contain approved claim_ids and approved evidence_ids listed below.
- Remove invalid refs, or replace them with approved refs that support the same text.
- If an invalid record contains a useful idea but it is not approved, you may mention it only as an unapproved candidate/background note and must not use that claim_id/evidence_id as a final ref.
- Keep the report Korean, coherent, and source-backed. Do not make it shorter just to avoid refs.
- You may call plasma.research.list, plasma.research.read, plasma.research.grep, and plasma.research.references again if you need approved replacement support.

Approved final ref scope:
%s

Invalid refs that caused repair:
%s

Evidence rigor still applies:
- Level: %s (%s)
- Meaning: %s
%s

User-visible generation plan:
%s

Original requested title:
%s

Plasma tool binding: use mission_id %s. If a tool requires session_id or producer, use session_id %s and producer {"type":"agent_session","id":"%s"}.

Return only the corrected AST JSON object with this shape:
{
  "title": "short report title",
  "summary": "executive summary paragraph",
  "blocks": [
    {"type": "heading", "level": 2, "text": "section title"},
    {"type": "paragraph", "text": "article paragraph", "refs": {"claim_ids": ["clm_..."], "evidence_ids": ["evd_..."]}},
    {"type": "bullet_list", "items": ["item"], "refs": {"evidence_ids": ["evd_..."]}},
    {"type": "quote", "text": "short callout"}
  ]
}

Original AST to repair:
%s
`, agentReportAnyJSON(app.ReportBlockSourceRefs{ClaimIDs: claimIDs, EvidenceIDs: evidenceIDs}), agentReportAnyJSON(violations), rigor.level, rigor.label, rigor.description, rigor.instructions, agentReportPlanJSON(plan), strings.TrimSpace(title), strings.TrimSpace(missionID), toolSessionID, toolSessionID, agentReportASTJSON(ast))
}

func agentReportBlocksToInputs(ast agentReportAST, fallbackTitle string) ([]app.ReportBlockDraftInput, error) {
	title := strings.TrimSpace(ast.Title)
	if title == "" {
		title = strings.TrimSpace(fallbackTitle)
	}
	if title == "" {
		title = "Mission report"
	}
	inputs := []app.ReportBlockDraftInput{{
		BlockType: "title",
		Content:   mustJSON(map[string]string{"text": title}),
	}}
	if summary := strings.TrimSpace(ast.Summary); summary != "" {
		inputs = append(inputs, app.ReportBlockDraftInput{
			BlockType: "abstract",
			Content:   mustJSON(map[string]string{"text": summary}),
		})
	}
	for _, block := range ast.Blocks {
		blockType := strings.TrimSpace(strings.ToLower(block.Type))
		if blockType == "list" {
			blockType = "bullet_list"
		}
		refs := block.SourceRefs
		if len(refs.ClaimIDs)+len(refs.EvidenceIDs)+len(refs.SnapshotIDs)+len(refs.QuestionIDs)+len(refs.OptionIDs) == 0 {
			refs = block.Refs
		}
		switch blockType {
		case "heading":
			text := strings.TrimSpace(block.Text)
			if text == "" {
				return nil, fmt.Errorf("%w: report heading text is required", app.ErrInvalidInput)
			}
			level := block.Level
			if level <= 0 {
				level = 2
			}
			inputs = append(inputs, app.ReportBlockDraftInput{
				BlockType:  "heading",
				Content:    mustJSON(map[string]any{"level": level, "text": text}),
				SourceRefs: refs,
			})
		case "paragraph":
			text := strings.TrimSpace(block.Text)
			if text == "" {
				return nil, fmt.Errorf("%w: report paragraph text is required", app.ErrInvalidInput)
			}
			inputs = append(inputs, app.ReportBlockDraftInput{
				BlockType:  "paragraph",
				Content:    mustJSON(map[string]string{"text": text}),
				SourceRefs: refs,
			})
		case "bullet_list":
			items := make([]string, 0, len(block.Items))
			for _, item := range block.Items {
				item = strings.TrimSpace(item)
				if item != "" {
					items = append(items, item)
				}
			}
			if len(items) == 0 {
				return nil, fmt.Errorf("%w: report list items are required", app.ErrInvalidInput)
			}
			inputs = append(inputs, app.ReportBlockDraftInput{
				BlockType:  "bullet_list",
				Content:    mustJSON(map[string][]string{"items": items}),
				SourceRefs: refs,
			})
		case "quote":
			text := strings.TrimSpace(block.Text)
			if text == "" {
				return nil, fmt.Errorf("%w: report quote text is required", app.ErrInvalidInput)
			}
			inputs = append(inputs, app.ReportBlockDraftInput{
				BlockType:  "quote",
				Content:    mustJSON(map[string]string{"text": text}),
				SourceRefs: refs,
			})
		case "":
			continue
		default:
			return nil, fmt.Errorf("%w: unsupported report AST block type %q", app.ErrInvalidInput, block.Type)
		}
	}
	if len(inputs) < 2 {
		return nil, fmt.Errorf("%w: report AST requires article content", app.ErrInvalidInput)
	}
	return inputs, nil
}
