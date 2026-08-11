package legacyfinalize

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/finaledit"
)

const GateSubmittedSentinel = "REPORT_FINALIZED"

// Runner는 legacy finalizer provider 실행과 durable replay를 수행한다.
type Runner struct {
	Store reporting.FinalEditStageStore
	NewID finaledit.IDGenerator
}

// Part는 legacy finalizer prompt에 노출되는 ordered Part inventory다.
type Part struct {
	Title      string
	Markdown   string
	ArtifactID string
	WordCount  int
}

// Input은 legacy finalization binding과 prompt에 필요한 frozen request metadata다.
type Input struct {
	MissionID                    string
	Title                        string
	DirectionHint                string
	FinalUserText                string
	ExecutorName                 string
	AgentModel                   string
	AgentReasoningEffort         string
	AgentSelectionSource         string
	MCPMode                      string
	Rigor                        reportprompt.RigorProfile
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	PostReportHumanize           string
	GenerationGuidanceProfile    string
	GenerationGuidanceSHA256     string
	PendingEventID               string
	ArtifactID                   string
	PlanEvent                    ledger.Event
	Plan                         reporting.SectionalReportPlan
	RequirementMap               reporting.ReportRequirementMap
	Parts                        []Part
	PartArtifactIDs              []string
	SectionArtifactIDs           []string
	SectionWordTotal             int
	SessionChainKind             string
	PreReportResearchSessionID   string
	ReportPlanSessionID          string
	FinalSessionID               string
	FinalForkSourceID            string
	StartedAt                    time.Time
}

// Output은 canonical finalization과 provider 결과를 함께 반환한다.
type Output struct {
	Artifact    artifact.Raw
	Event       ledger.Event
	Markdown    string
	AgentResult agentexec.AgentResult
	Binding     reporting.LongFormFinalizeBinding
}

// Run은 legacy finalizer의 기존 prompt/tool/session/retry/replay 계약을 실행한다.
func (runner Runner) Run(ctx context.Context, input Input, executor agentexec.AgentExecutor) (Output, error) {
	core := finaledit.Runner{Store: runner.Store, NewID: runner.NewID}
	toolSessionID := core.ID("ses")
	binding := reporting.LongFormFinalizeBinding{
		MissionID: input.MissionID, PendingEventID: input.PendingEventID, PlanEventID: input.PlanEvent.EventID, ArtifactID: input.ArtifactID,
		Filename: finaledit.SafeFilename(input.Title, ".md"), Title: input.Title, ToolSessionID: toolSessionID,
		IdempotencyKey:    "report-long-form-finalize:" + input.PendingEventID + ":" + input.PlanEvent.EventID,
		ProviderSessionID: input.FinalSessionID, PreviousProviderSessionID: input.FinalSessionID,
		PartArtifactIDs: input.PartArtifactIDs, SectionArtifactIDs: input.SectionArtifactIDs, SectionWordCount: input.SectionWordTotal,
		CompositionStrategy: reportprompt.LongFormCompositionStrategy(input.GenerationGuidanceProfile),
		AgentExecutor:       input.ExecutorName, AgentModel: input.AgentModel, AgentReasoningEffort: input.AgentReasoningEffort, AgentSelectionSource: input.AgentSelectionSource,
		MCPMode: input.MCPMode, RigorLevel: input.Rigor.Level, RigorLabel: input.Rigor.Label,
		ReportSessionPolicy: input.ReportSessionPolicy, ReportSessionPolicySelection: input.ReportSessionPolicySelection,
		PostReportHumanize: input.PostReportHumanize, GenerationGuidanceProfile: input.GenerationGuidanceProfile, GenerationGuidanceSHA256: input.GenerationGuidanceSHA256,
		SessionChainKind: input.SessionChainKind, PreReportResearchSessionID: input.PreReportResearchSessionID, ReportPlanSessionID: input.ReportPlanSessionID,
		ForkSourceAgentSessionID: input.FinalForkSourceID, PlanToolSessionID: finaledit.ReportEventString(input.PlanEvent, "tool_session_id"), StartedAt: input.StartedAt,
		Producer: ledger.Producer{Type: "agent_session", ID: input.FinalSessionID},
	}
	var finalResult agentexec.AgentResult
	var finalization reporting.LongFormFinalizeResult
	var hint reporting.LongFormFinalizationHint
	canonical := false
	finalUserText := strings.TrimSpace(input.FinalUserText)
	if finalUserText == "" {
		finalUserText = "finalize section-fanout long-form markdown report"
	}
	for attempt := 1; attempt <= 2; attempt++ {
		attemptStarted := time.Now()
		prompt := PromptWithRequirements(input, binding, attempt, canonical, hint)
		if !canonical {
			prompt = reportprompt.WithLongFormDownstreamDirection(prompt, input.DirectionHint)
		}
		result, runErr := executor.Run(ctx, agentexec.AgentRequest{
			UserText: finalUserText,
			Prompt:   prompt,
			Model:    input.AgentModel, ReasoningEffort: input.AgentReasoningEffort, MissionID: input.MissionID, ToolSessionID: toolSessionID,
			PreviousSessionID: input.FinalSessionID, AgentExecutor: input.ExecutorName, MCPMode: input.MCPMode,
			ExtraMCPTools: MCPTools(input.GenerationGuidanceProfile), ReplaceMCPTools: true, LongFormFinalize: &binding,
		})
		durationMS := time.Since(attemptStarted).Milliseconds()
		logFinalObservation(input.MissionID, input.PendingEventID, input.PlanEvent.EventID, attempt, input.FinalSessionID, result, durationMS)
		if runErr == nil {
			result, runErr = finaledit.ValidatedSameSessionResult(result, input.FinalSessionID)
		}
		if runErr == nil {
			finalResult = result
		}
		loaded, exists, loadErr := reporting.LoadLongFormFinalization(context.WithoutCancel(ctx), runner.Store, binding)
		if loadErr != nil {
			return Output{}, finaledit.StageFailure("final", input.PlanEvent.EventID, loadErr)
		}
		canonical = exists
		if exists {
			finalization = loaded
		}
		if runErr == nil && canonical && result.Text == GateSubmittedSentinel {
			return outputFromFinal(finalization, finalResult, binding), nil
		}
		if attempt == 1 {
			hint = reporting.RecoverLongFormFinalizationHint(result.Text)
			continue
		}
		cause := runErr
		if cause == nil {
			cause = fmt.Errorf("%w: finalization acknowledgement was not exact", producterror.ErrConflict)
		}
		if canonical {
			return outputFromFinal(finalization, finalResult, binding), nil
		}
		return Output{}, finaledit.StageFailure("final", input.PlanEvent.EventID, finaledit.AgentFailure(cause, result, "report_frame", durationMS, input.FinalSessionID))
	}
	return outputFromFinal(finalization, finalResult, binding), nil
}

func outputFromFinal(final reporting.LongFormFinalizeResult, result agentexec.AgentResult, binding reporting.LongFormFinalizeBinding) Output {
	return Output{Artifact: final.Artifact, Event: final.Event, Markdown: string(final.Artifact.Content), AgentResult: result, Binding: binding}
}

func logFinalObservation(missionID, pendingEventID, planEventID string, attempt int, boundSessionID string, result agentexec.AgentResult, durationMS int64) {
	returnedSessionID := strings.TrimSpace(result.SessionID)
	inputTokens, outputTokens, totalTokens := 0, 0, 0
	usageAvailable := result.Usage.ProviderUsage != nil
	if usageAvailable {
		inputTokens = result.Usage.ProviderUsage.InputTokens
		outputTokens = result.Usage.ProviderUsage.OutputTokens
		totalTokens = result.Usage.ProviderUsage.TotalTokens
		if totalTokens == 0 {
			totalTokens = inputTokens + outputTokens
		}
	}
	log.Printf("report_long_form_final_observed mission_id=%q pending_event_id=%q plan_event_id=%q attempt_count=%d returned_session_present=%t returned_session_matches_bound=%t usage_available=%t input_tokens=%d output_tokens=%d total_tokens=%d resumed=%t duration_ms=%d", missionID, pendingEventID, planEventID, attempt, returnedSessionID != "", returnedSessionID != "" && returnedSessionID == strings.TrimSpace(boundSessionID), usageAvailable, inputTokens, outputTokens, totalTokens, result.Resumed, durationMS)
}

// ForkSession은 legacy section_fanout finalizer의 기존 session fork 오류 문자열을 보존한다.
func ForkSession(ctx context.Context, forker agentexec.AgentSessionForker, sourceSessionID string) (string, string, error) {
	sourceSessionID = strings.TrimSpace(sourceSessionID)
	if sourceSessionID == "" {
		return "", "", fmt.Errorf("%w: section fanout requires a report plan provider session", producterror.ErrConflict)
	}
	fork, err := forker.ForkSession(ctx, sourceSessionID)
	if err != nil {
		return "", "", fmt.Errorf("section fanout session fork failed: %w", err)
	}
	if strings.TrimSpace(fork.SessionID) == "" {
		return "", "", fmt.Errorf("%w: section fanout session fork returned an empty session", producterror.ErrConflict)
	}
	return strings.TrimSpace(fork.SessionID), strings.TrimSpace(fork.SourceSessionID), nil
}
