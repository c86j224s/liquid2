package reportworkflow

import (
	"context"
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/directdraft"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/evidencecheck"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/finalstore"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/finalwrite"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/legacyfinalize"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partassembly"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partedit"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partplan"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/plan"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/readeredit"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/reportassembly"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/requirements"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/sectiondraft"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/semanticcheck"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/styleedit"
)

// NewRunner는 제품 고정 stage runner들을 조립한다.
func NewRunner(config RunnerConfig) Runner {
	if config.Service == nil {
		panic("reportworkflow: service is required")
	}
	return Runner{
		service:         config.Service,
		finalEditStore:  config.Service,
		humanizeService: config.Service,
		executor:        config.Executor,
		newID:           config.NewID,
		planRunner: plan.Runner{
			Service: config.Service, RepairStore: config.Service, Lifecycle: config.Lifecycle, Executor: config.Executor,
			NewID: config.NewID, LatestSessionID: config.LatestSessionID,
		},
		directDraftRunner: directdraft.Runner{
			Executor: config.Executor, NewID: config.NewID, LatestSessionID: config.LatestSessionID,
		},
		finalStoreRunner: finalstore.Runner{
			Service: config.Service, GateReader: finalstore.ReportingGateReader{Store: config.Service}, NewID: config.NewID,
		},
		requirementsRunner: requirements.Runner{
			Service: config.Service, Lifecycle: config.Lifecycle, Executor: config.Executor,
		},
		partPlanRunner: partplan.Runner{
			Service: config.Service, Executor: config.Executor, NewID: config.NewID,
		},
		sectionDraftRunner: sectiondraft.Runner{
			Service: config.Service, Executor: config.Executor, NewID: config.NewID,
		},
		partAssemblyRunner: partassembly.Runner{
			Service: config.Service, Executor: config.Executor, NewID: config.NewID,
		},
		partEditRunner: partedit.Runner{
			Service: config.Service, Executor: config.Executor, NewID: config.NewID,
		},
		reportAssemblyRunner: reportassembly.Runner{
			Store: config.Service, NewID: config.NewID,
		},
		finalWriteRunner: finalwrite.Runner{
			Store: config.Service, NewID: config.NewID,
		},
		readerEditRunner: readeredit.Runner{
			Store: config.Service, NewID: config.NewID,
		},
		styleEditRunner: styleedit.Runner{
			Store: config.Service, NewID: config.NewID,
		},
		semanticCheckRunner: semanticcheck.Runner{
			Store: config.Service, NewID: config.NewID,
		},
		evidenceCheckRunner: evidencecheck.Runner{
			Store: config.Service, NewID: config.NewID,
		},
		legacyFinalizeRunner: legacyfinalize.Runner{
			Store: config.Service, NewID: config.NewID,
		},
	}
}

// WithObserver는 ledger payload를 만들지 않는 내용 없는 node 관측 hook을 붙인다.
func (runner Runner) WithObserver(observer Observer) Runner {
	runner.observer = observer
	return runner
}

// RunDraft는 one_take와 planned Markdown report graph를 같은 typed entrypoint로 실행한다.
func (runner Runner) RunDraft(ctx context.Context, input DraftInput) (DraftOutput, error) {
	family, err := SelectFamily(input.ReportMode, input.ExecutionStrategy)
	if err != nil {
		return DraftOutput{}, err
	}
	switch family {
	case FamilyOneTake:
		return runner.runOneTake(ctx, input)
	case FamilyPlanned:
		return runner.runPlanned(ctx, input)
	default:
		return DraftOutput{}, fmt.Errorf("%w: %s report workflow must use RunLongForm", producterror.ErrInvalidInput, family)
	}
}

func (runner Runner) runOneTake(ctx context.Context, input DraftInput) (DraftOutput, error) {
	done := runner.observeStart(NodeDirectDraft)
	out, err := runner.directDraftRunner.RunOneTake(ctx, directdraftInput(input))
	done(err, false)
	if err != nil {
		return DraftOutput{}, err
	}
	doneStore := runner.observeStart(NodeFinalStore)
	stored, err := runner.finalStoreRunner.CommitOneTake(ctx, finalstoreOneTakeInput(input, out))
	doneStore(err, false)
	if err != nil {
		return DraftOutput{}, err
	}
	return draftOutput(stored), nil
}

func (runner Runner) runPlanned(ctx context.Context, input DraftInput) (DraftOutput, error) {
	donePlan := runner.observeStart(NodePlan)
	planOut, err := runner.planRunner.RunMarkdown(ctx, planInput(input))
	donePlan(err, planOut.Recovered)
	if err != nil {
		return DraftOutput{}, err
	}
	doneDraft := runner.observeStart(NodeDirectDraft)
	out, err := runner.directDraftRunner.RunPlanned(ctx, directdraft.PlannedInput{
		BaseInput:                  directdraftInput(input),
		Plan:                       planOut.Plan,
		PlanEventID:                planOut.Event.EventID,
		PlanToolSessionID:          planOut.PlanToolSessionID,
		ArtifactID:                 planOut.ArtifactID,
		ReportPlanSessionID:        planOut.ReportPlanSessionID,
		SessionChainKind:           planOut.SessionChainKind,
		PreReportResearchSessionID: planOut.PreReportResearchSessionID,
		ForkSourceSessionID:        planOut.ForkSourceSessionID,
		WorkflowStartedAt:          planOut.StartedAt,
	})
	doneDraft(err, false)
	if err != nil {
		return DraftOutput{}, err
	}
	doneStore := runner.observeStart(NodeFinalStore)
	stored, err := runner.finalStoreRunner.CommitPlanned(ctx, finalstorePlannedInput(input, out))
	doneStore(err, false)
	if err != nil {
		return DraftOutput{}, err
	}
	return draftOutput(stored), nil
}

func planInput(input DraftInput) plan.Input {
	return plan.Input{
		MissionID: input.MissionID, PendingEventID: input.PendingEventID,
		Title: input.Title, DirectionHint: input.DirectionHint,
		AgentExecutor: input.AgentExecutor, AgentModel: input.AgentModel,
		AgentReasoningEffort: input.AgentReasoningEffort, AgentSelectionSource: input.AgentSelectionSource,
		MCPMode: input.MCPMode, Rigor: input.Rigor,
		ReportSessionPolicy: input.ReportSessionPolicy, ReportSessionPolicySelection: input.ReportSessionPolicySelection,
		PostReportHumanize:        input.PostReportHumanize,
		GenerationGuidanceProfile: input.GenerationGuidanceProfile, GenerationGuidanceSHA256: input.GenerationGuidanceSHA256,
	}
}

func directdraftInput(input DraftInput) directdraft.BaseInput {
	return directdraft.BaseInput{
		MissionID: input.MissionID, PendingEventID: input.PendingEventID,
		Title: input.Title, DirectionHint: input.DirectionHint,
		AgentExecutor: input.AgentExecutor, AgentModel: input.AgentModel,
		AgentReasoningEffort: input.AgentReasoningEffort, AgentSelectionSource: input.AgentSelectionSource,
		MCPMode: input.MCPMode, Rigor: input.Rigor,
		ReportSessionPolicy: input.ReportSessionPolicy, ReportSessionPolicySelection: input.ReportSessionPolicySelection,
		PostReportHumanize:        input.PostReportHumanize,
		GenerationGuidanceProfile: input.GenerationGuidanceProfile, GenerationGuidanceSHA256: input.GenerationGuidanceSHA256,
	}
}

func finalstoreBaseInput(input DraftInput) finalstore.BaseInput {
	return finalstore.BaseInput{
		MissionID: input.MissionID, PendingEventID: input.PendingEventID, Title: input.Title,
		AgentExecutor: input.AgentExecutor, AgentModel: input.AgentModel, ReasoningEffort: input.AgentReasoningEffort,
		SelectionSource: input.AgentSelectionSource, MCPMode: input.MCPMode, Rigor: input.Rigor,
		SessionPolicy: input.ReportSessionPolicy, PolicySelection: input.ReportSessionPolicySelection,
		PostHumanize: input.PostReportHumanize, GuidanceProfile: input.GenerationGuidanceProfile,
		GuidanceSHA256: input.GenerationGuidanceSHA256,
	}
}

func finalstoreOneTakeInput(input DraftInput, out directdraft.OneTakeCandidate) finalstore.OneTakeInput {
	return finalstore.OneTakeInput{
		Base: finalstoreBaseInput(input),
		Candidate: finalstore.OneTakeCandidate{
			ArtifactID: out.ArtifactID, ToolSessionID: out.ToolSessionID,
			PreviousSessionID: out.PreviousSessionID, ReturnedSessionID: out.ReturnedSessionID,
			ReportSessionID: out.ReportSessionID, ReportSessionPolicy: out.ReportSessionPolicy,
			Markdown: out.Markdown, StartedAt: out.StartedAt, AgentDurationMS: out.AgentDurationMS,
			AgentUsage: out.AgentUsage, AgentResumed: out.AgentResumed,
		},
	}
}

func finalstorePlannedInput(input DraftInput, out directdraft.PlannedCandidate) finalstore.PlannedInput {
	return finalstore.PlannedInput{
		Base: finalstoreBaseInput(input),
		Candidate: finalstore.PlannedCandidate{
			ArtifactID: out.ArtifactID, ToolSessionID: out.ToolSessionID,
			PlanEventID: out.PlanEventID, PlanToolSessionID: out.PlanToolSessionID,
			ReportPlanSessionID: out.ReportPlanSessionID, SessionChainKind: out.SessionChainKind,
			PreReportResearchSessionID: out.PreReportResearchSessionID, ForkSourceSessionID: out.ForkSourceSessionID,
			ReturnedSessionID: out.ReturnedSessionID, ReportSessionID: out.ReportSessionID,
			Markdown: out.Markdown, WorkflowStartedAt: out.WorkflowStartedAt,
			AgentDurationMS: out.AgentDurationMS, AgentUsage: out.AgentUsage, AgentResumed: out.AgentResumed,
		},
	}
}

func draftOutput(out finalstore.Output) DraftOutput {
	return DraftOutput{Artifact: out.Artifact, Event: out.Event, Markdown: out.Markdown, ReportSessionID: out.ReportSessionID}
}
