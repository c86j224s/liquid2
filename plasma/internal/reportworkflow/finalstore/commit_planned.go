package finalstore

import (
	"context"
	"fmt"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

// CommitPlanned는 planned direct draft 후보를 기존 원자 저장 정책으로 durable 저장한다.
//
// 이 함수는 반드시 CreateRawArtifactWithEvent 한 번만 호출한다. event id는 atomic builder 안에서
// 할당하며, canonical plan이 예약한 artifact id를 그대로 사용한다.
func (runner Runner) CommitPlanned(ctx context.Context, input PlannedInput) (Output, error) {
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
	if err := validatePlannedCandidate(candidate); err != nil {
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
				AgentSessionID: candidate.ReportSessionID, PreviousAgentSessionID: candidate.ReportPlanSessionID,
				ReturnedAgentSessionID: candidate.ReturnedSessionID, ToolSessionID: candidate.ToolSessionID,
				MCPMode: input.Base.MCPMode, RigorLevel: input.Base.Rigor.Level, RigorLabel: input.Base.Rigor.Label,
				ReportMode: reportexecution.ModePlanned, ReportModeLabel: reportexecution.ModeLabel(reportexecution.ModePlanned),
				ReportSessionPolicy: input.Base.SessionPolicy, ReportSessionPolicySelection: input.Base.PolicySelection,
				PostReportHumanize: input.Base.PostHumanize, HumanizeEnabled: input.Base.PostHumanize != "disabled",
				GenerationGuidanceProfile: input.Base.GuidanceProfile, GenerationGuidanceSHA256: input.Base.GuidanceSHA256,
				SessionChainKind: candidate.SessionChainKind, PreReportResearchSessionID: candidate.PreReportResearchSessionID,
				ReportPlanSessionID: candidate.ReportPlanSessionID, ReportSessionID: candidate.ReportSessionID,
				ForkSourceAgentSessionID: candidate.ForkSourceSessionID, CompositionStrategy: "planned_markdown",
				DurationMS: time.Since(candidate.WorkflowStartedAt).Milliseconds(),
				Text:       "계획 기반 Markdown 리포트 artifact를 생성했습니다.", AgentUsage: candidate.AgentUsage,
				AgentUsageSurface: "report_markdown", AgentUsageDurationMS: candidate.AgentDurationMS,
				AgentResumed: candidate.AgentResumed, Producer: producer,
			},
			Artifact: raw, PlanEventID: candidate.PlanEventID, PlanToolSessionID: candidate.PlanToolSessionID,
			IncludePlanReview: true, PlanReviewRequired: false, PlanReviewState: "auto_accepted",
		})
	})
	if err != nil {
		return Output{}, err
	}
	if err := validateStoredArtifact(raw, input.Base.MissionID, candidate.ArtifactID, candidate.Markdown); err != nil {
		return Output{}, err
	}
	if err := validateStoredEvent(event, input.Base.MissionID, input.Base.PendingEventID, candidate.ArtifactID, candidate.PlanEventID); err != nil {
		return Output{}, err
	}
	return Output{Artifact: raw, Event: event, Markdown: candidate.Markdown, ReportSessionID: candidate.ReportSessionID}, nil
}
