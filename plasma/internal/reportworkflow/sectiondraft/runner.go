package sectiondraft

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/longformutil"
)

const markdownMediaType = "text/markdown; charset=utf-8"

const (
	// EvidenceGapControlToken is the exact provider response that means the
	// Section cannot be supported by substantive evidence in the current scope.
	EvidenceGapControlToken = "SECTION_EVIDENCE_GAP"
	EvidenceGapReasonCode   = "inadequate_section_evidence"
	MaxEvidenceGapAttempts  = 2
)

// Run은 단일 Section writer를 실행하고 Section artifact/event를 기존 순서대로 저장한다.
func (runner Runner) Run(ctx context.Context, input Input) (Output, error) {
	input.Attempt = normalizeAttempt(input.Attempt)
	if input.Attempt > MaxEvidenceGapAttempts {
		return Output{}, longformutil.StageFailure("section", input.Base.PlanEvent.EventID, input.PartIndex+1, input.SectionIndex+1,
			fmt.Errorf("%w: section evidence attempt %d exceeds maximum %d", producterror.ErrInvalidInput, input.Attempt, MaxEvidenceGapAttempts))
	}
	if input.StartedEvent {
		if err := runner.appendStarted(ctx, input); err != nil {
			return Output{}, err
		}
	}
	started := time.Now()
	result, err := runner.Executor.Run(ctx, agentexec.AgentRequest{
		UserText: input.UserText, Prompt: Prompt(input), Model: input.Base.AgentModel,
		ReasoningEffort: input.Base.AgentReasoningEffort, MissionID: input.Base.MissionID,
		ToolSessionID: input.ToolSessionID, PreviousSessionID: input.PreviousSessionID,
		AgentExecutor: input.Base.AgentExecutor, MCPMode: input.Base.MCPMode,
		ExtraMCPTools: MCPTools(), ReplaceMCPTools: true,
	})
	durationMS := time.Since(started).Milliseconds()
	if err != nil {
		return Output{}, longformutil.StageFailure("section", input.Base.PlanEvent.EventID, input.PartIndex+1, input.SectionIndex+1,
			longformutil.AgentFailure(err, result, "report_section", durationMS, input.PreviousSessionID))
	}
	returnedSessionID := strings.TrimSpace(result.SessionID)
	validated, err := longformutil.ValidateSameSessionResult(result, input.PreviousSessionID)
	if err != nil {
		return Output{}, longformutil.StageFailure("section", input.Base.PlanEvent.EventID, input.PartIndex+1, input.SectionIndex+1,
			longformutil.AgentFailure(err, result, "report_section", durationMS, input.PreviousSessionID))
	}
	markdown := strings.TrimSpace(validated.Text)
	if markdown == EvidenceGapControlToken {
		gap := EvidenceGap{
			PartIndex: input.PartIndex + 1, SectionIndex: input.SectionIndex + 1, Attempt: input.Attempt,
			ReasonCode: EvidenceGapReasonCode, SessionID: validated.SessionID, ReturnedSessionID: returnedSessionID,
			PreviousSessionID: input.PreviousSessionID, ToolSessionID: input.ToolSessionID,
			SourceSessionID: input.SourceSessionID, DurationMS: durationMS,
		}
		if err := runner.appendEvidenceGap(ctx, input, gap, validated, returnedSessionID, durationMS); err != nil {
			return Output{}, longformutil.StageFailure("section", input.Base.PlanEvent.EventID, input.PartIndex+1, input.SectionIndex+1, err)
		}
		return Output{EvidenceGap: &gap, ReturnedSessionID: returnedSessionID, DurationMS: durationMS}, nil
	}
	if markdown == "" {
		return Output{}, longformutil.StageFailure("section", input.Base.PlanEvent.EventID, input.PartIndex+1, input.SectionIndex+1,
			longformutil.AgentFailure(fmt.Errorf("%w: section report agent returned empty Markdown", producterror.ErrInvalidInput), validated, "report_section", durationMS, input.PreviousSessionID))
	}
	artifact, err := runner.Service.CreateRawArtifact(ctx, artifact.CreateRequest{
		ArtifactID: runner.id("art"), MissionID: input.Base.MissionID, MediaType: markdownMediaType,
		Filename: longformutil.SafeFilename(fmt.Sprintf("%s part %02d section %02d", input.Base.Title, input.PartIndex+1, input.SectionIndex+1), ".md"),
		Producer: ledger.Producer{Type: "agent_session", ID: longformutil.FallbackSessionID(validated.SessionID, input.ToolSessionID)},
		Content:  []byte(markdown),
	})
	if err != nil {
		return Output{}, longformutil.StageFailure("section", input.Base.PlanEvent.EventID, input.PartIndex+1, input.SectionIndex+1, err)
	}
	wordCount := longformutil.WordCount(markdown)
	_, err = runner.Service.AppendEvent(ctx, reporting.BuildMarkdownReportSectionCreatedAppendRequest(reporting.MarkdownReportSectionCreatedEventRequest{
		MarkdownReportStageEventBase: runner.stageBase(input, artifact, validated, returnedSessionID, durationMS, input.CreatedText),
		PartIndex:                    input.PartIndex + 1,
		SectionIndex:                 input.SectionIndex + 1,
		WordCount:                    wordCount,
	}))
	if err != nil {
		return Output{}, longformutil.StageFailure("section", input.Base.PlanEvent.EventID, input.PartIndex+1, input.SectionIndex+1, err)
	}
	return Output{
		Draft:             Draft{Title: input.Section.Title, Markdown: markdown, ArtifactID: artifact.ArtifactID, WordCount: wordCount, SessionID: validated.SessionID},
		ReturnedSessionID: returnedSessionID, DurationMS: durationMS,
	}, nil
}

func (runner Runner) appendEvidenceGap(ctx context.Context, input Input, gap EvidenceGap, result agentexec.AgentResult, returnedSessionID string, durationMS int64) error {
	_, err := runner.Service.AppendEvent(ctx, reporting.BuildMarkdownReportSectionEvidenceGapAppendRequest(reporting.MarkdownReportSectionEvidenceGapEventRequest{
		EventID:                    runner.id("evt"),
		MissionID:                  input.Base.MissionID,
		PendingEventID:             input.Base.PendingEventID,
		PlanEventID:                input.Base.PlanEvent.EventID,
		PartIndex:                  gap.PartIndex,
		SectionIndex:               gap.SectionIndex,
		Attempt:                    gap.Attempt,
		ReasonCode:                 gap.ReasonCode,
		AgentExecutor:              input.Base.AgentExecutor,
		AgentSessionID:             result.SessionID,
		PreviousAgentSessionID:     input.PreviousSessionID,
		ReturnedAgentSessionID:     returnedSessionID,
		ToolSessionID:              input.ToolSessionID,
		SessionChainKind:           input.Base.SessionChainKind,
		PreReportResearchSessionID: input.Base.PreReportResearchSessionID,
		ReportPlanSessionID:        input.Base.ReportPlanSessionID,
		ReportSessionID:            result.SessionID,
		ForkSourceAgentSessionID:   input.SourceSessionID,
		DurationMS:                 durationMS,
		AgentUsage:                 result.Usage,
		AgentUsageSurface:          "report_section",
		AgentUsageDurationMS:       durationMS,
		AgentResumed:               result.Resumed,
		Producer:                   ledger.Producer{Type: "agent_session", ID: longformutil.FallbackSessionID(result.SessionID, input.ToolSessionID)},
	}))
	return err
}

func (runner Runner) appendStarted(ctx context.Context, input Input) error {
	sessionID := longformutil.FallbackSessionID(input.PreviousSessionID, input.ToolSessionID)
	_, err := runner.Service.AppendEvent(ctx, reporting.BuildMarkdownReportSectionStartedAppendRequest(reporting.MarkdownReportSectionStartedEventRequest{
		MarkdownReportStageEventBase: reporting.MarkdownReportStageEventBase{
			EventID: runner.id("evt"), MissionID: input.Base.MissionID, PendingEventID: input.Base.PendingEventID,
			PlanEventID: input.Base.PlanEvent.EventID, Title: input.Section.Title,
			AgentExecutor: input.Base.AgentExecutor, AgentModel: input.Base.AgentModel,
			AgentReasoningEffort: input.Base.AgentReasoningEffort, AgentSelectionSource: input.Base.AgentSelectionSource,
			AgentSessionID: sessionID, PreviousAgentSessionID: input.PreviousSessionID, ToolSessionID: input.ToolSessionID,
			ReportMode: reportexecution.ModeLongForm, ReportModeLabel: reportexecution.ModeLabel(reportexecution.ModeLongForm),
			ReportSessionPolicy: input.Base.ReportSessionPolicy, ReportSessionPolicySelection: input.Base.ReportSessionPolicySelection,
			PostReportHumanize: input.Base.PostReportHumanize, HumanizeEnabled: input.Base.PostReportHumanize != "disabled",
			GenerationGuidanceProfile: input.Base.GenerationGuidanceProfile, GenerationGuidanceSHA256: input.Base.GenerationGuidanceSHA256,
			SessionChainKind: input.Base.SessionChainKind, PreReportResearchSessionID: input.Base.PreReportResearchSessionID,
			ReportPlanSessionID: input.Base.ReportPlanSessionID, ReportSessionID: sessionID,
			ForkSourceAgentSessionID: input.SourceSessionID, CompositionStrategy: "sectional_preserve_markdown",
			AssemblyStrategy: "c4_normalized_section_headings", Text: "장문 리포트 섹션 Markdown 생성을 시작했습니다.",
			Producer: ledger.Producer{Type: "agent_session", ID: sessionID},
		},
		PartIndex: input.PartIndex + 1, SectionIndex: input.SectionIndex + 1,
	}))
	if err != nil {
		return longformutil.StageFailure("section", input.Base.PlanEvent.EventID, input.PartIndex+1, input.SectionIndex+1, err)
	}
	return nil
}

func (runner Runner) stageBase(input Input, artifact artifact.Raw, result agentexec.AgentResult, returnedSessionID string, durationMS int64, text string) reporting.MarkdownReportStageEventBase {
	return reporting.MarkdownReportStageEventBase{
		EventID: runner.id("evt"), MissionID: input.Base.MissionID, PendingEventID: input.Base.PendingEventID,
		PlanEventID: input.Base.PlanEvent.EventID, Title: input.Section.Title, Artifact: artifact,
		AgentExecutor: input.Base.AgentExecutor, AgentModel: input.Base.AgentModel,
		AgentReasoningEffort: input.Base.AgentReasoningEffort, AgentSelectionSource: input.Base.AgentSelectionSource,
		AgentSessionID: result.SessionID, PreviousAgentSessionID: input.PreviousSessionID,
		ReturnedAgentSessionID: returnedSessionID, ToolSessionID: input.ToolSessionID,
		ReportMode: reportexecution.ModeLongForm, ReportModeLabel: reportexecution.ModeLabel(reportexecution.ModeLongForm),
		ReportSessionPolicy: input.Base.ReportSessionPolicy, ReportSessionPolicySelection: input.Base.ReportSessionPolicySelection,
		PostReportHumanize: input.Base.PostReportHumanize, HumanizeEnabled: input.Base.PostReportHumanize != "disabled",
		GenerationGuidanceProfile: input.Base.GenerationGuidanceProfile, GenerationGuidanceSHA256: input.Base.GenerationGuidanceSHA256,
		SessionChainKind: input.Base.SessionChainKind, PreReportResearchSessionID: input.Base.PreReportResearchSessionID,
		ReportPlanSessionID: input.Base.ReportPlanSessionID, ReportSessionID: result.SessionID,
		ForkSourceAgentSessionID: input.SourceSessionID, CompositionStrategy: "sectional_preserve_markdown",
		AssemblyStrategy: "c4_normalized_section_headings", DurationMS: durationMS, Text: text,
		AgentUsage: result.Usage, AgentUsageSurface: "report_section", AgentUsageDurationMS: durationMS,
		AgentResumed: result.Resumed, Producer: ledger.Producer{Type: "agent_session", ID: longformutil.FallbackSessionID(result.SessionID, input.ToolSessionID)},
	}
}

func (runner Runner) id(prefix string) string {
	if runner.NewID != nil {
		return runner.NewID(prefix)
	}
	return prefix + "_missing"
}

func normalizeAttempt(attempt int) int {
	if attempt < 1 {
		return 1
	}
	return attempt
}
