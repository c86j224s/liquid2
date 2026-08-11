package partassembly

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

type normalization struct {
	DropFirstLeadingHeadingTexts []string
	ConvertHeadingsBold          bool
	ForceHeadingLevel            int
	MaxHeadingLevel              int
	StripBoundaryRules           bool
}

// AssembleMarkdown는 immutable Section bodies와 connective assembly를 기존 framing으로 결합한다.
func AssembleMarkdown(part reporting.ReportPlanPart, drafts []SectionDraft, assembly reporting.PartAssembly, partIndex int) string {
	var out strings.Builder
	out.WriteString(partHeadingLine(part.Title, partIndex+1) + "\n\n")
	if intro := normalizeConnectiveMarkdown(assembly.Intro); intro != "" {
		out.WriteString(intro)
		out.WriteString("\n\n")
	}
	transitions := map[int]string{}
	for _, transition := range assembly.Transitions {
		transitions[transition.AfterSectionIndex] = normalizeConnectiveMarkdown(transition.Markdown)
	}
	for index, draft := range drafts {
		title := displayHeadingText(firstNonEmpty(draft.Title, fmt.Sprintf("Section %d", index+1)), partIndex+1, index+1)
		out.WriteString(fmt.Sprintf("## %d.%d %s\n\n", partIndex+1, index+1, title))
		out.WriteString(normalizeBodyMarkdown(draft.Markdown, draft.Title, partIndex+1, index+1))
		out.WriteString("\n\n")
		if transition := strings.TrimSpace(transitions[index+1]); transition != "" && index < len(drafts)-1 {
			out.WriteString(transition)
			out.WriteString("\n\n")
		}
	}
	if closing := normalizeConnectiveMarkdown(assembly.Closing); closing != "" {
		out.WriteString(closing)
		out.WriteString("\n")
	}
	return strings.TrimSpace(out.String()) + "\n"
}

var (
	markdownHeadingLineRE                = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*$`)
	markdownLeadingNumberingRE           = regexp.MustCompile(`^\d+(?:\.\d+)*\.?\s+`)
	markdownLeadingGlobalSectionNumberRE = regexp.MustCompile(`^\d+\.\s+`)
)

func normalizeBodyMarkdown(markdown string, sectionTitle string, partNumber int, sectionNumber int) string {
	return normalizeMarkdown(markdown, normalization{
		DropFirstLeadingHeadingTexts: []string{sectionTitle, fmt.Sprintf("%d.%d %s", partNumber, sectionNumber, sectionTitle)},
		ConvertHeadingsBold:          true,
		StripBoundaryRules:           true,
	})
}

func normalizeConnectiveMarkdown(markdown string) string {
	return normalizeMarkdown(markdown, normalization{ConvertHeadingsBold: true, StripBoundaryRules: true})
}

func normalizeMarkdown(markdown string, opts normalization) string {
	lines := strings.Split(strings.TrimSpace(markdown), "\n")
	if opts.StripBoundaryRules {
		lines = stripBoundaryHorizontalRules(lines)
	}
	out := make([]string, 0, len(lines))
	seenBody := false
	droppedFirstLeadingHeading := false
	dropFirstHeadingTexts := canonicalHeadingTextSet(opts.DropFirstLeadingHeadingTexts)
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
		if fenceMarker != "" || trimmed == "" {
			out = append(out, line)
			if trimmed != "" {
				seenBody = true
			}
			continue
		}
		matches := markdownHeadingLineRE.FindStringSubmatch(trimmed)
		if len(matches) == 3 {
			text := strings.TrimSpace(matches[2])
			canonicalText := canonicalHeadingText(text)
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
		}
		out = append(out, line)
		seenBody = true
		lastAdjacentHeadingText = ""
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
