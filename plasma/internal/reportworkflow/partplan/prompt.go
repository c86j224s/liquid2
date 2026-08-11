package partplan

import (
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/longformutil"
)

// Prompt는 한 Part의 Section 흐름을 미리 정리하는 private editorial brief prompt다.
func Prompt(input Input) string {
	requirements := reporting.ReportRequirementsForPart(input.Base.RequirementMap, input.PartIndex+1)
	prompt := fmt.Sprintf(`Take responsibility for the reading flow of one Part before its Sections are drafted.

Report title: %s
Part %d: %s

Overall report plan:
%s

Requirements assigned to this Part:
%s

Write a short private editorial brief that you can use again when the assembled Part returns for final authorship. State the Part's central reader question, the intended flow across its Sections, and one natural-sentence job for each Section. When one explanation clearly belongs in a single Section, name that home once; otherwise leave that point out.

Teach the later Section writers what a reader should understand and how the explanation should move. Keep this as useful working memory rather than a compliance checklist. Do not draft report paragraphs or add new researched facts. Return only the brief.`,
		input.Base.Title, input.PartIndex+1, input.Part.Title, longformutil.AnyJSON(input.Base.Plan), longformutil.AnyJSON(requirements))
	return reportprompt.WithLongFormDownstreamDirection(prompt, input.Base.DirectionHint)
}
