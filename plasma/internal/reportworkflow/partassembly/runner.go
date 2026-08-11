package partassembly

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
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/longformutil"
)

const markdownMediaType = "text/markdown; charset=utf-8"

// Run은 단일 Part assembly agent를 실행하고 assembled Part artifact/event를 저장한다.
func (runner Runner) Run(ctx context.Context, input Input) (Output, error) {
	started := time.Now()
	assembly, result, returnedSessionID, err := runner.runAgent(ctx, input)
	durationMS := time.Since(started).Milliseconds()
	if err != nil {
		return Output{}, longformutil.StageFailure("part", input.Base.PlanEvent.EventID, input.PartIndex+1, 0,
			longformutil.AgentFailure(err, result, "report_part", durationMS, input.PreviousSessionID))
	}
	partMarkdown := AssembleMarkdown(input.Part, input.Sections, assembly, input.PartIndex)
	raw, err := runner.Service.CreateRawArtifact(ctx, artifact.CreateRequest{
		ArtifactID: runner.id("art"), MissionID: input.Base.MissionID, MediaType: markdownMediaType,
		Filename: longformutil.SafeFilename(fmt.Sprintf("%s part %02d", input.Base.Title, input.PartIndex+1), ".md"),
		Producer: ledger.Producer{Type: "agent_session", ID: longformutil.FallbackSessionID(result.SessionID, input.ToolSessionID)},
		Content:  []byte(partMarkdown),
	})
	if err != nil {
		return Output{}, longformutil.StageFailure("part", input.Base.PlanEvent.EventID, input.PartIndex+1, 0, err)
	}
	wordCount := longformutil.WordCount(partMarkdown)
	_, err = runner.Service.AppendEvent(ctx, reporting.BuildMarkdownReportPartCreatedAppendRequest(reporting.MarkdownReportPartCreatedEventRequest{
		MarkdownReportStageEventBase: reporting.MarkdownReportStageEventBase{
			EventID: runner.id("evt"), MissionID: input.Base.MissionID, PendingEventID: input.Base.PendingEventID,
			PlanEventID: input.Base.PlanEvent.EventID, Title: input.Part.Title, Artifact: raw,
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
			ForkSourceAgentSessionID: input.ForkSourceSessionID, CompositionStrategy: "sectional_preserve_markdown",
			AssemblyStrategy: "c4_normalized_section_headings", DurationMS: durationMS,
			Text:       "장문 리포트 파트 Markdown을 보존 조립했습니다.",
			AgentUsage: result.Usage, AgentUsageSurface: "report_part", AgentUsageDurationMS: durationMS,
			AgentResumed: result.Resumed, Producer: ledger.Producer{Type: "agent_session", ID: longformutil.FallbackSessionID(result.SessionID, input.ToolSessionID)},
		},
		PartIndex: input.PartIndex + 1, SectionCount: len(input.Sections), WordCount: wordCount,
	}))
	if err != nil {
		return Output{}, longformutil.StageFailure("part", input.Base.PlanEvent.EventID, input.PartIndex+1, 0, err)
	}
	return Output{Draft: PartDraft{Title: input.Part.Title, Markdown: partMarkdown, ArtifactID: raw.ArtifactID, WordCount: wordCount, SessionID: result.SessionID}, ReturnedSessionID: returnedSessionID, DurationMS: durationMS}, nil
}

func (runner Runner) runAgent(ctx context.Context, input Input) (reporting.PartAssembly, agentexec.AgentResult, string, error) {
	agentReq := agentexec.AgentRequest{
		UserText: fmt.Sprintf("assemble part %d for sectional long-form markdown report", input.PartIndex+1),
		Prompt:   Prompt(input), Model: input.Base.AgentModel, ReasoningEffort: input.Base.AgentReasoningEffort,
		MissionID: input.Base.MissionID, ToolSessionID: input.ToolSessionID, PreviousSessionID: input.PreviousSessionID,
		AgentExecutor: input.Base.AgentExecutor, MCPMode: input.Base.MCPMode,
		ExtraMCPTools: ReadMCPTools(), ReplaceMCPTools: true,
	}
	var binding reporting.PartAssemblyBinding
	if UseEditTools(input.Base.GenerationGuidanceProfile) {
		binding = partAssemblyBinding(input)
		agentReq.Prompt = EditToolsPrompt(input, binding, runner.id("rpa"))
		agentReq.ExtraMCPTools = MCPTools(input.Base.GenerationGuidanceProfile)
		agentReq.PartAssembly = &binding
	}
	agentReq.Prompt = reportprompt.WithLongFormDownstreamDirection(agentReq.Prompt, input.Base.DirectionHint)
	result, err := runner.Executor.Run(ctx, agentReq)
	if err != nil {
		return reporting.PartAssembly{}, result, "", err
	}
	returnedSessionID := strings.TrimSpace(result.SessionID)
	result, err = longformutil.ValidateSameSessionResult(result, input.PreviousSessionID)
	if err != nil {
		return reporting.PartAssembly{}, result, returnedSessionID, err
	}
	if !UseEditTools(input.Base.GenerationGuidanceProfile) {
		assembly, parseErr := ParseAgentPartAssembly(result.Text)
		return assembly, result, returnedSessionID, parseErr
	}
	if strings.TrimSpace(result.Text) != reporting.PartAssemblySubmittedSentinel {
		return reporting.PartAssembly{}, result, returnedSessionID, fmt.Errorf("%w: part assembly agent did not confirm MCP submission", producterror.ErrInvalidInput)
	}
	submission, exists, err := reporting.LoadPartAssemblySubmission(context.WithoutCancel(ctx), runner.Service, binding)
	if err != nil {
		return reporting.PartAssembly{}, result, returnedSessionID, err
	}
	if !exists {
		return reporting.PartAssembly{}, result, returnedSessionID, fmt.Errorf("%w: part assembly MCP submission is missing", producterror.ErrConflict)
	}
	return submission.Assembly, result, returnedSessionID, nil
}

// RunAgent는 Web의 기존 단위 테스트 호환 지점에서 provider 실행 계약만 재사용한다.
// Part artifact 저장은 Run만 수행하며 이 함수는 durable MCP replay 검증까지만 책임진다.
func (runner Runner) RunAgent(ctx context.Context, input Input) (reporting.PartAssembly, agentexec.AgentResult, string, error) {
	return runner.runAgent(ctx, input)
}

// Binding은 Part assembly MCP 도구가 사용할 durable identity 계약을 만든다.
// root는 session lineage를 이미 확정한 Input만 넘겨야 하며 caller가 정책을 선택하지 않는다.
func Binding(input Input) reporting.PartAssemblyBinding {
	return partAssemblyBinding(input)
}

func partAssemblyBinding(input Input) reporting.PartAssemblyBinding {
	return reporting.PartAssemblyBinding{
		MissionID: input.Base.MissionID, PendingEventID: input.Base.PendingEventID,
		PlanEventID: input.Base.PlanEvent.EventID, ToolSessionID: input.ToolSessionID,
		ProviderSessionID: input.PreviousSessionID, PreviousProviderSessionID: input.PreviousSessionID,
		PartIndex: input.PartIndex + 1, SectionCount: len(input.Sections), SectionArtifactIDs: sectionArtifactIDs(input.Sections),
		AgentExecutor: input.Base.AgentExecutor, AgentModel: input.Base.AgentModel,
		AgentReasoningEffort: input.Base.AgentReasoningEffort, AgentSelectionSource: input.Base.AgentSelectionSource,
		MCPMode: input.Base.MCPMode, ReportSessionPolicy: input.Base.ReportSessionPolicy,
		ReportSessionPolicySelection: input.Base.ReportSessionPolicySelection, PostReportHumanize: input.Base.PostReportHumanize,
		GenerationGuidanceProfile: input.Base.GenerationGuidanceProfile, GenerationGuidanceSHA256: input.Base.GenerationGuidanceSHA256,
		SessionChainKind: input.Base.SessionChainKind, PreReportResearchSessionID: input.Base.PreReportResearchSessionID,
		ReportPlanSessionID: input.Base.ReportPlanSessionID, ForkSourceAgentSessionID: input.ForkSourceSessionID,
		Producer: ledger.Producer{Type: "agent_session", ID: input.ToolSessionID},
	}
}

func sectionArtifactIDs(drafts []SectionDraft) []string {
	ids := make([]string, len(drafts))
	for index, draft := range drafts {
		ids[index] = strings.TrimSpace(draft.ArtifactID)
	}
	return ids
}

func (runner Runner) id(prefix string) string {
	if runner.NewID != nil {
		return runner.NewID(prefix)
	}
	return prefix + "_missing"
}
