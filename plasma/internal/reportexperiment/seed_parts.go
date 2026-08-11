package reportexperiment

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow"
)

type seedPartsInput struct {
	Service               Service
	Loaded                LoadedFixture
	Stem                  string
	MissionID             string
	PendingEventID        string
	PlanEventID           string
	PlanProviderSessionID string
	PreReportSessionID    string
	ForkSourceSessionID   string
	AgentModel            string
	ReasoningEffort       string
	GuidanceProfile       string
	GuidanceSHA256        string
	Producer              ledger.Producer
}

func createSeedParts(ctx context.Context, input seedPartsInput) ([]reportworkflow.PrefixPart, [][]reportworkflow.PrefixSection, []string, []string, int, error) {
	parts := make([]reportworkflow.PrefixPart, 0, len(input.Loaded.Parts))
	sections := make([][]reportworkflow.PrefixSection, 0, len(input.Loaded.Parts))
	partArtifactIDs := make([]string, 0, len(input.Loaded.Parts))
	sectionArtifactIDs := make([]string, 0, len(input.Loaded.Parts))
	sectionWordTotal := 0
	for _, part := range input.Loaded.Parts {
		partArtifactID := fmt.Sprintf("art_reportexperiment_%s_part_%02d", input.Stem, part.Spec.Index)
		artifact, err := createSeedArtifact(ctx, input.Service, input.MissionID, partArtifactID, fmt.Sprintf("part-%02d.md", part.Spec.Index), input.Producer, part.Content)
		if err != nil {
			return nil, nil, nil, nil, 0, err
		}
		stageBase := markdownStageBase(markdownStageBaseInput{
			MissionID: input.MissionID, PendingEventID: input.PendingEventID, PlanEventID: input.PlanEventID, Title: input.Loaded.Spec.ReportTitle,
			AgentModel: input.AgentModel, ReasoningEffort: input.ReasoningEffort, ToolSessionID: fmt.Sprintf("ses_reportexperiment_%s_part_%02d", input.Stem, part.Spec.Index),
			ProviderSessionID: input.PlanProviderSessionID, Rigor: input.Loaded.Spec.Rigor,
			PostReportHumanize: input.Loaded.Spec.PostReportHumanize, GuidanceProfile: input.GuidanceProfile, GuidanceSHA256: input.GuidanceSHA256,
			PreReportSessionID: input.PreReportSessionID, PlanSessionID: input.PlanProviderSessionID, ForkSourceSessionID: input.ForkSourceSessionID,
			Text:     "고정 reviewed Part artifact를 finalization 입력으로 기록했습니다.",
			Producer: input.Producer,
		})
		stageBase.EventID = fmt.Sprintf("evt_reportexperiment_%s_section_%02d", input.Stem, part.Spec.Index)
		stageBase.Artifact = artifact
		if _, err := input.Service.AppendEvent(ctx, reporting.BuildMarkdownReportSectionCreatedAppendRequest(reporting.MarkdownReportSectionCreatedEventRequest{
			MarkdownReportStageEventBase: stageBase,
			PartIndex:                    part.Spec.Index,
			SectionIndex:                 1,
			WordCount:                    part.WordCount,
		})); err != nil {
			return nil, nil, nil, nil, 0, err
		}
		stageBase.EventID = fmt.Sprintf("evt_reportexperiment_%s_part_%02d", input.Stem, part.Spec.Index)
		if _, err := input.Service.AppendEvent(ctx, reporting.BuildMarkdownReportPartCreatedAppendRequest(reporting.MarkdownReportPartCreatedEventRequest{
			MarkdownReportStageEventBase: stageBase,
			PartIndex:                    part.Spec.Index,
			SectionCount:                 1,
			WordCount:                    part.WordCount,
		})); err != nil {
			return nil, nil, nil, nil, 0, err
		}
		markdown := string(part.Content)
		sectionTitle := strings.TrimSpace(part.Spec.SectionTitle)
		if sectionTitle == "" {
			sectionTitle = strings.TrimSpace(part.Spec.Title)
		}
		parts = append(parts, reportworkflow.PrefixPart{Title: part.Spec.Title, Markdown: markdown, ArtifactID: partArtifactID, WordCount: part.WordCount})
		sections = append(sections, []reportworkflow.PrefixSection{{Title: sectionTitle, Markdown: markdown, ArtifactID: partArtifactID, WordCount: part.WordCount}})
		partArtifactIDs = append(partArtifactIDs, partArtifactID)
		sectionArtifactIDs = append(sectionArtifactIDs, partArtifactID)
		sectionWordTotal += part.WordCount
	}
	return parts, sections, partArtifactIDs, sectionArtifactIDs, sectionWordTotal, nil
}
