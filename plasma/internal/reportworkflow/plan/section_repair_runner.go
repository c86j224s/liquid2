package plan

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

const maxSectionPlanRepairResponseBytes = 64 * 1024

type sectionPlanRepairResponse struct {
	Replacements []reporting.ReportSectionPlanReplacement `json:"replacements"`
}

// RunLongFormSectionRepair resumes the original planner once, validates that
// only terminal gap coordinates changed, and records the amendment durably.
func (runner Runner) RunLongFormSectionRepair(ctx context.Context, input LongFormSectionRepairInput) (LongFormSectionRepairOutput, error) {
	gaps, err := normalizeRepairCoordinates(input.Plan.Plan, input.Gaps)
	if err != nil || runner.RepairStore == nil {
		return LongFormSectionRepairOutput{}, fmt.Errorf("%w: Section plan repair input is invalid", producterror.ErrInvalidInput)
	}
	input.Gaps = gaps
	toolSessionID := runner.id("ses")
	started := time.Now()
	result, runErr := runner.Executor.Run(ctx, agentexec.AgentRequest{
		UserText: "repair unsupported sections in long-form report plan",
		Prompt:   LongFormSectionRepairPrompt(input), Model: input.Plan.AgentModel,
		ReasoningEffort: input.Plan.AgentReasoningEffort, MissionID: input.Request.MissionID,
		ToolSessionID: toolSessionID, PreviousSessionID: input.Plan.ReportPlanSessionID,
		AgentExecutor: input.Plan.AgentExecutor, MCPMode: input.Plan.MCPMode,
		ExtraMCPTools: ResearchMCPTools(), ReplaceMCPTools: true,
	})
	durationMS := time.Since(started).Milliseconds()
	if runErr == nil {
		result, runErr = validateSameSessionResult(result, input.Plan.ReportPlanSessionID)
	}
	if runErr != nil {
		return LongFormSectionRepairOutput{}, reportAgentFailure(runErr, result, "report_plan_repair", durationMS, input.Plan.ReportPlanSessionID)
	}
	base := reporting.MarkdownReportEventBase{
		EventID: runner.id("evt"), MissionID: input.Request.MissionID, PendingEventID: input.Request.PendingEventID,
		Title: input.Request.Title, AgentExecutor: input.Plan.AgentExecutor, AgentModel: input.Plan.AgentModel,
		AgentReasoningEffort: input.Plan.AgentReasoningEffort, AgentSelectionSource: input.Plan.AgentSelectionSource,
		AgentSessionID: result.SessionID, PreviousAgentSessionID: input.Plan.ReportPlanSessionID,
		ReturnedAgentSessionID: result.SessionID, ToolSessionID: toolSessionID, MCPMode: input.Plan.MCPMode,
		RigorLevel: input.Request.Rigor.Level, RigorLabel: input.Request.Rigor.Label,
		ReportMode: reportexecution.ModeLongForm, ReportModeLabel: reportexecution.ModeLabel(reportexecution.ModeLongForm),
		ReportSessionPolicy: input.Plan.ReportSessionPolicy, ReportSessionPolicySelection: input.Plan.ReportSessionPolicySelection,
		PostReportHumanize: input.Request.PostReportHumanize, HumanizeEnabled: input.Request.PostReportHumanize != "disabled",
		GenerationGuidanceProfile: input.Plan.GenerationGuidanceProfile,
		GenerationGuidanceSHA256:  input.Plan.GenerationGuidanceSHA256,
		SessionChainKind:          input.Plan.SessionChainKind, PreReportResearchSessionID: input.Plan.PreReportResearchSessionID,
		ReportPlanSessionID: input.Plan.ReportPlanSessionID, ReportSessionID: result.SessionID,
		ForkSourceAgentSessionID: input.Plan.ForkSourceSessionID, CompositionStrategy: "sectional_preserve_markdown",
		DurationMS: durationMS, AgentUsage: result.Usage, AgentUsageSurface: "report_plan_repair",
		AgentUsageDurationMS: durationMS, AgentResumed: result.Resumed,
		Producer: ledger.Producer{Type: "agent_session", ID: result.SessionID},
	}
	text := strings.TrimSpace(result.Text)
	if text == SectionPlanUnrepairableControlToken {
		finalized, err := reporting.FinalizeLongFormSectionPlanUnrepairable(context.WithoutCancel(ctx), runner.RepairStore, input.Plan.Plan, reporting.LongFormSectionPlanRepairEventRequest{
			MarkdownReportEventBase: base, PlanEventID: input.Plan.Event.EventID, Coordinates: gaps,
		})
		if err != nil {
			return LongFormSectionRepairOutput{}, err
		}
		if finalized.Unrepairable {
			return LongFormSectionRepairOutput{}, fmt.Errorf("%w: planner found no supportable replacement Section", producterror.ErrConflict)
		}
		return sectionPlanRepairOutput(finalized), nil
	}
	response, err := decodeSectionPlanRepairResponse(text)
	if err != nil || !sameCoordinateSet(response.Replacements, gaps) {
		return LongFormSectionRepairOutput{}, fmt.Errorf("%w: planner returned an invalid Section repair", producterror.ErrInvalidInput)
	}
	refs := make([]reporting.ReportPlanSourceRefs, 0, len(response.Replacements))
	for _, replacement := range response.Replacements {
		refs = append(refs, replacement.Section.TargetRefs)
	}
	durableCtx := context.WithoutCancel(ctx)
	if err := runner.RepairStore.ValidateReportPlanRefs(durableCtx, input.Request.MissionID, refs); err != nil {
		return LongFormSectionRepairOutput{}, fmt.Errorf("%w: planner returned invalid Section references", producterror.ErrInvalidInput)
	}
	finalized, err := reporting.FinalizeLongFormSectionPlanRepair(durableCtx, runner.RepairStore, input.Plan.Plan, reporting.LongFormSectionPlanRepairEventRequest{
		MarkdownReportEventBase: base, PlanEventID: input.Plan.Event.EventID, Replacements: response.Replacements,
	})
	if err != nil {
		return LongFormSectionRepairOutput{}, err
	}
	if finalized.Unrepairable {
		return LongFormSectionRepairOutput{}, fmt.Errorf("%w: planner previously found no supportable replacement Section", producterror.ErrConflict)
	}
	return sectionPlanRepairOutput(finalized), nil
}

func sectionPlanRepairOutput(result reporting.LongFormSectionPlanRepairResult) LongFormSectionRepairOutput {
	return LongFormSectionRepairOutput{Plan: result.Plan, Event: result.Event, Replacements: result.Replacements}
}

func decodeSectionPlanRepairResponse(text string) (sectionPlanRepairResponse, error) {
	if text == "" || len([]byte(text)) > maxSectionPlanRepairResponseBytes {
		return sectionPlanRepairResponse{}, fmt.Errorf("%w: Section plan repair response is empty or too large", producterror.ErrInvalidInput)
	}
	decoder := json.NewDecoder(bytes.NewBufferString(text))
	decoder.DisallowUnknownFields()
	var response sectionPlanRepairResponse
	if err := decoder.Decode(&response); err != nil {
		return sectionPlanRepairResponse{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return sectionPlanRepairResponse{}, fmt.Errorf("%w: Section plan repair response has trailing data", producterror.ErrInvalidInput)
	}
	return response, nil
}

func normalizeRepairCoordinates(plan reporting.SectionalReportPlan, values []reporting.ReportSectionCoordinate) ([]reporting.ReportSectionCoordinate, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("%w: terminal Section gaps are required", producterror.ErrInvalidInput)
	}
	result := append([]reporting.ReportSectionCoordinate(nil), values...)
	sort.Slice(result, func(i, j int) bool {
		if result[i].PartIndex == result[j].PartIndex {
			return result[i].SectionIndex < result[j].SectionIndex
		}
		return result[i].PartIndex < result[j].PartIndex
	})
	for index, coordinate := range result {
		if coordinate.PartIndex < 1 || coordinate.PartIndex > len(plan.Parts) || coordinate.SectionIndex < 1 || coordinate.SectionIndex > len(plan.Parts[coordinate.PartIndex-1].Sections) ||
			index > 0 && coordinate == result[index-1] {
			return nil, fmt.Errorf("%w: terminal Section gap coordinate is invalid", producterror.ErrInvalidInput)
		}
	}
	return result, nil
}

func sameCoordinateSet(values []reporting.ReportSectionPlanReplacement, gaps []reporting.ReportSectionCoordinate) bool {
	coordinates := make([]reporting.ReportSectionCoordinate, len(values))
	for index, value := range values {
		coordinates[index] = value.ReportSectionCoordinate
	}
	normalized, err := normalizeRepairCoordinatesFromValues(coordinates)
	if err != nil || len(normalized) != len(gaps) {
		return false
	}
	for index := range gaps {
		if normalized[index] != gaps[index] {
			return false
		}
	}
	return true
}

func normalizeRepairCoordinatesFromValues(values []reporting.ReportSectionCoordinate) ([]reporting.ReportSectionCoordinate, error) {
	result := append([]reporting.ReportSectionCoordinate(nil), values...)
	sort.Slice(result, func(i, j int) bool {
		if result[i].PartIndex == result[j].PartIndex {
			return result[i].SectionIndex < result[j].SectionIndex
		}
		return result[i].PartIndex < result[j].PartIndex
	})
	for index := range result {
		if result[index].PartIndex < 1 || result[index].SectionIndex < 1 || index > 0 && result[index] == result[index-1] {
			return nil, fmt.Errorf("%w: replacement Section coordinate is invalid", producterror.ErrInvalidInput)
		}
	}
	return result, nil
}
