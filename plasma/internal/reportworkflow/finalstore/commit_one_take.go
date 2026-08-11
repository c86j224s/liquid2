package finalstore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

// CommitOneTake는 one_take direct draft 후보를 artifact, terminal event, report-run
// membership이 같은 저장 경계에 들어가도록 durable 저장한다.
func (runner Runner) CommitOneTake(ctx context.Context, input OneTakeInput) (Output, error) {
	if err := ctx.Err(); err != nil {
		return Output{}, err
	}
	if runner.Service == nil {
		return Output{}, fmt.Errorf("%w: finalstore service is required", producterror.ErrInvalidInput)
	}
	if err := validateBase(input.Base); err != nil {
		return Output{}, err
	}
	candidate := input.Candidate
	if err := validateOneTakeCandidate(candidate); err != nil {
		return Output{}, err
	}
	producer := ledger.Producer{Type: "agent_session", ID: fallbackSessionID(candidate.ReportSessionID, candidate.ToolSessionID)}
	raw, event, err := runner.Service.CreateRawArtifactWithEvent(ctx, artifact.CreateRequest{
		ArtifactID: candidate.ArtifactID, MissionID: input.Base.MissionID,
		MediaType: markdownMediaType, Filename: safeFilename(input.Base.Title, ".md"),
		Producer: producer, Content: []byte(candidate.Markdown),
	}, func(raw artifact.Raw) ledger.AppendRequest {
		return reporting.BuildMarkdownReportArtifactCreatedAppendRequest(reporting.MarkdownReportArtifactCreatedEventRequest{
			MarkdownReportEventBase: reporting.MarkdownReportEventBase{
				EventID: runner.id("evt"), MissionID: input.Base.MissionID, PendingEventID: input.Base.PendingEventID,
				Title: input.Base.Title, AgentExecutor: input.Base.AgentExecutor, AgentModel: input.Base.AgentModel,
				AgentReasoningEffort: input.Base.ReasoningEffort, AgentSelectionSource: input.Base.SelectionSource,
				AgentSessionID: candidate.ReportSessionID, PreviousAgentSessionID: candidate.PreviousSessionID,
				ReturnedAgentSessionID: candidate.ReturnedSessionID, ToolSessionID: candidate.ToolSessionID,
				MCPMode: input.Base.MCPMode, RigorLevel: input.Base.Rigor.Level, RigorLabel: input.Base.Rigor.Label,
				ReportMode: reportexecution.ModeOneTake, ReportModeLabel: reportexecution.ModeLabel(reportexecution.ModeOneTake),
				ReportSessionPolicy:          candidate.ReportSessionPolicy,
				ReportSessionPolicySelection: strings.TrimSpace(input.Base.PolicySelection),
				PostReportHumanize:           input.Base.PostHumanize, HumanizeEnabled: input.Base.PostHumanize != "disabled",
				GenerationGuidanceProfile: input.Base.GuidanceProfile, GenerationGuidanceSHA256: input.Base.GuidanceSHA256,
				SessionChainKind: "same_session_report", PreReportResearchSessionID: candidate.PreviousSessionID,
				ReportSessionID: candidate.ReportSessionID, CompositionStrategy: "one_take_markdown",
				DurationMS: time.Since(candidate.StartedAt).Milliseconds(),
				Text:       "빠른 Markdown 리포트 artifact를 생성했습니다.", AgentUsage: candidate.AgentUsage,
				AgentUsageSurface: "report_one_take", AgentUsageDurationMS: candidate.AgentDurationMS,
				AgentResumed: candidate.AgentResumed, Producer: producer,
			},
			Artifact: raw, PlanReviewState: "not_applicable", IncludePlanReview: false,
		})
	})
	if err != nil {
		return Output{}, err
	}
	if err := validateStoredArtifact(raw, input.Base.MissionID, candidate.ArtifactID, candidate.Markdown); err != nil {
		return Output{}, err
	}
	if err := validateStoredEvent(event, input.Base.MissionID, input.Base.PendingEventID, candidate.ArtifactID, ""); err != nil {
		return Output{}, err
	}
	return Output{Artifact: raw, Event: event, Markdown: candidate.Markdown, ReportSessionID: candidate.ReportSessionID}, nil
}
