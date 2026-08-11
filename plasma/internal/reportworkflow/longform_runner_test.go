package reportworkflow

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/plan"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/sectiondraft"
)

func TestRunLongFormPrefixObservesSerialStages(t *testing.T) {
	planValue := reporting.SectionalReportPlan{
		Summary: "Explain one point.",
		Parts: []reporting.ReportPlanPart{{
			Title: "Core Part",
			Sections: []reporting.ReportPlanSection{{
				Title: "Core Section", Purpose: "Explain the mechanism.",
			}},
		}},
	}
	requirementMap, err := reporting.NormalizeReportRequirementMap(reporting.ReportRequirementMap{ReviewedEventIDs: []string{"evt_pending"}}, planValue)
	if err != nil {
		t.Fatal(err)
	}
	requirementHash, requirementJSON, err := reporting.ReportRequirementMapHash(requirementMap)
	if err != nil {
		t.Fatal(err)
	}
	service := &workflowService{
		events: []ledger.Event{{
			EventID: "evt_pending", MissionID: "mis_1", EventType: "report.draft.pending",
			Producer: ledger.Producer{Type: "user", ID: "plasma-ui"},
			Payload:  mustWorkflowJSON(map[string]any{"origin_pending_event_id": "evt_pending", "retry_strategy": "initial"}),
		}},
		selection: reporting.ReportPlanSubmissionSelection{
			EventID: "evt_submitted", ArgumentsHash: "args", PlanHash: "plan_hash",
			Plan: mustWorkflowJSON(planValue),
		},
		reqSelect: app.ReportRequirementMapSelection{
			Event:              ledger.Event{EventID: "evt_requirements", MissionID: "mis_1", EventType: reporting.ReportRequirementsMappedEventType},
			RequirementMapHash: requirementHash,
			RequirementMap:     requirementJSON,
		},
	}
	executor := &workflowExecutor{results: []agentexec.AgentResult{
		{Text: reporting.ReportPlanSubmittedSentinel, SessionID: "plan-session-1"},
		{Text: reporting.ReportRequirementsMappedSentinel, SessionID: "plan-session-1"},
		{Text: "# Core Section\n\nBody.", SessionID: "plan-session-1"},
		{Text: `{"intro":"Intro","transitions":[],"closing":"Close"}`, SessionID: "plan-session-1"},
	}}
	observer := &workflowObserver{}
	runner := workflowRunner(service, executor, observer)

	out, err := runner.RunLongFormPrefix(context.Background(), DraftInput{
		MissionID: "mis_1", PendingEventID: "evt_pending", Title: "Reader Report",
		AgentExecutor: "codex", AgentModel: "gpt-test", AgentReasoningEffort: "medium",
		MCPMode: "auto", Rigor: reportprompt.RigorProfile{Level: "balanced", Label: "균형형"},
		ReportMode: reportexecution.ModeLongForm, ExecutionStrategy: "serial",
		ReportSessionPolicy:          reportexecution.SessionPolicyFreshSession,
		ReportSessionPolicySelection: reportexecution.SessionPolicySelectionAutoFreshSession,
		PostReportHumanize:           "disabled",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Parts) != 1 || out.Parts[0].ArtifactID == "" || out.FinalTail != FinalTailLegacy {
		t.Fatalf("unexpected prefix output: %#v", out)
	}
	assertObservedNodes(t, observer, []string{
		NodePlan, NodePlan,
		NodeRequirements, NodeRequirements,
		NodeSectionDraft, NodeSectionDraft,
		NodePartAssembly, NodePartAssembly,
	})
}

func TestRunLongFormPrefixSerialRetriesEvidenceGapInSameSession(t *testing.T) {
	planValue := workflowSingleSectionPlan()
	service := workflowRecoveredPrefixService("evt_pending", planValue)
	executor := &evidenceGapRetryWorkflowExecutor{sectionTexts: []string{
		sectiondraft.EvidenceGapControlToken,
		"# Core Section\n\nSubstantive replacement-backed body.",
	}}
	runner := workflowRunner(service, executor, &workflowObserver{})

	_, err := runner.RunLongFormPrefix(context.Background(), workflowLongFormInput("evt_pending", "serial"))
	if err != nil {
		t.Fatal(err)
	}
	if len(executor.sectionRequests) != 2 {
		t.Fatalf("section provider calls = %d, want 2", len(executor.sectionRequests))
	}
	first, second := executor.sectionRequests[0], executor.sectionRequests[1]
	if first.ToolSessionID == "" || first.ToolSessionID != second.ToolSessionID || first.PreviousSessionID != second.PreviousSessionID {
		t.Fatalf("section retry did not reuse session binding: first=%#v second=%#v", first, second)
	}
	if countWorkflowEvents(service.events, sectiondraft.EvidenceGapEventType) != 1 || countWorkflowEvents(service.events, sectiondraft.CreatedEventType) != 1 {
		t.Fatalf("unexpected section event counts: %#v", service.events)
	}
	gap := workflowEventPayload(t, latestWorkflowEvent(service.events, sectiondraft.EvidenceGapEventType))
	if gap["attempt_number"] != float64(1) || gap["reason_code"] != sectiondraft.EvidenceGapReasonCode {
		t.Fatalf("unexpected gap payload: %#v", gap)
	}
}

func TestRunLongFormPrefixFanoutRetriesEvidenceGapInSameTaskSession(t *testing.T) {
	planValue := workflowSingleSectionPlan()
	service := workflowRecoveredPrefixService("evt_pending", planValue)
	executor := &evidenceGapRetryWorkflowExecutor{sectionTexts: []string{
		sectiondraft.EvidenceGapControlToken,
		"# Core Section\n\nFanout retry body.",
	}}
	runner := workflowRunner(service, executor, &workflowObserver{})

	_, err := runner.RunLongFormPrefix(context.Background(), workflowLongFormInput("evt_pending", "section_fanout"))
	if err != nil {
		t.Fatal(err)
	}
	if len(executor.sectionRequests) != 2 || len(executor.forks) != 2 {
		t.Fatalf("section calls=%d forks=%d, want 2 section calls and the existing section+part forks", len(executor.sectionRequests), len(executor.forks))
	}
	first, second := executor.sectionRequests[0], executor.sectionRequests[1]
	if first.ToolSessionID == "" || first.ToolSessionID != second.ToolSessionID || first.PreviousSessionID == "" || first.PreviousSessionID != second.PreviousSessionID {
		t.Fatalf("fanout retry did not stay in same task/session: first=%#v second=%#v", first, second)
	}
}

func TestRunLongFormPrefixRecoveredEvidenceGapAttemptOneInvokesAttemptTwo(t *testing.T) {
	planValue := workflowSingleSectionPlan()
	service := workflowRecoveredPrefixService("evt_pending", planValue)
	service.events = append(service.events, workflowSectionGapEvent("evt_pending", 1, "provider-plan", "ses_gap"))
	executor := &evidenceGapRetryWorkflowExecutor{sectionTexts: []string{"# Core Section\n\nRecovered retry body."}}
	runner := workflowRunner(service, executor, &workflowObserver{})

	_, err := runner.RunLongFormPrefix(context.Background(), workflowLongFormInput("evt_pending", "serial"))
	if err != nil {
		t.Fatal(err)
	}
	if len(executor.sectionRequests) != 1 {
		t.Fatalf("section provider calls = %d, want only recovered attempt 2", len(executor.sectionRequests))
	}
	req := executor.sectionRequests[0]
	if req.ToolSessionID != "ses_gap" || req.PreviousSessionID != "provider-plan" || !strings.Contains(req.Prompt, "Section evidence attempt: 2 of 2") {
		t.Fatalf("recovered attempt 2 binding mismatch: %#v", req)
	}
}

func TestRunLongFormPrefixSerialRejectsRecoveredEvidenceGapSourceMismatch(t *testing.T) {
	planValue := workflowSingleSectionPlan()
	service := workflowRecoveredPrefixService("evt_pending", planValue)
	service.events = append(service.events, workflowSectionGapEventWithSourceID("evt_gap", "evt_pending", 1, "provider-plan", "ses_gap", "unexpected-source"))
	executor := &evidenceGapRetryWorkflowExecutor{sectionTexts: []string{"# Should not run"}}
	runner := workflowRunner(service, executor, &workflowObserver{})

	_, err := runner.RunLongFormPrefix(context.Background(), workflowLongFormInput("evt_pending", "serial"))
	if err == nil || !errors.Is(err, producterror.ErrConflict) {
		t.Fatalf("err = %v, want conflict for serial source mismatch", err)
	}
	if len(executor.requests) != 0 {
		t.Fatalf("provider calls = %d, want 0", len(executor.requests))
	}
}

func TestRunLongFormPrefixFanoutRejectsRecoveredEvidenceGapSourceMismatch(t *testing.T) {
	planValue := workflowSingleSectionPlan()
	service := workflowRecoveredPrefixService("evt_pending", planValue)
	service.events = append(service.events, workflowSectionGapEventWithSourceID("evt_gap", "evt_pending", 1, "fork-session-A", "ses_gap", "unexpected-source"))
	executor := &evidenceGapRetryWorkflowExecutor{sectionTexts: []string{"# Should not run"}}
	runner := workflowRunner(service, executor, &workflowObserver{})

	_, err := runner.RunLongFormPrefix(context.Background(), workflowLongFormInput("evt_pending", "section_fanout"))
	if err == nil || !errors.Is(err, producterror.ErrConflict) {
		t.Fatalf("err = %v, want conflict for fanout source mismatch", err)
	}
	if len(executor.requests) != 0 || len(executor.forks) != 0 {
		t.Fatalf("provider calls=%d forks=%d, want none", len(executor.requests), len(executor.forks))
	}
}

func TestRunLongFormPrefixRecoveredEvidenceGapAttemptTwoInvokesOneBoundedRepair(t *testing.T) {
	planValue := workflowSingleSectionPlan()
	service := workflowRecoveredPrefixService("evt_pending", planValue)
	service.events = append(service.events,
		workflowSectionGapEventWithID("evt_gap_1", "evt_pending", 1, "provider-plan", "ses_gap"),
		workflowSectionGapEventWithID("evt_gap_2", "evt_pending", 2, "provider-plan", "ses_gap"),
	)
	executor := &evidenceGapRetryWorkflowExecutor{repairText: plan.SectionPlanUnrepairableControlToken}
	runner := workflowRunner(service, executor, &workflowObserver{})

	_, err := runner.RunLongFormPrefix(context.Background(), workflowLongFormInput("evt_pending", "serial"))
	var stage *reportexecution.StageFailureError
	if err == nil || !errors.As(err, &stage) || stage.Kind != "section" || !errors.Is(err, producterror.ErrConflict) {
		t.Fatalf("err = %v, stage = %#v, want section conflict", err, stage)
	}
	if len(executor.repairRequests) != 1 || len(executor.sectionRequests) != 0 {
		t.Fatalf("repair calls=%d section calls=%d, want one repair and no Section rerun", len(executor.repairRequests), len(executor.sectionRequests))
	}
	if countWorkflowEvents(service.events, reporting.LongFormSectionPlanRepairCompletedEventType) != 1 {
		t.Fatalf("unrepairable outcome was not recorded: %#v", service.events)
	}
	service.events = append(service.events,
		ledger.Event{EventID: "evt_root_failed", MissionID: "mis_1", EventType: "report.draft.failed", Payload: mustWorkflowJSON(map[string]any{
			"pending_event_id": "evt_pending", "kind": "report_draft_failed",
		})},
		ledger.Event{EventID: "evt_retry", MissionID: "mis_1", EventType: "report.draft.pending", Producer: ledger.Producer{Type: "user", ID: "plasma-ui"}, Payload: mustWorkflowJSON(map[string]any{
			"origin_pending_event_id": "evt_pending", "retry_of_pending_event_id": "evt_pending", "retry_strategy": "resume_failed", "attempt_number": 2,
		})},
	)
	_, err = runner.RunLongFormPrefix(context.Background(), workflowLongFormInput("evt_retry", "serial"))
	if err == nil || !errors.Is(err, producterror.ErrConflict) || len(executor.repairRequests) != 1 {
		t.Fatalf("resume_failed retried a consumed repair round: err=%v repair calls=%d", err, len(executor.repairRequests))
	}
}

func TestRunLongFormPrefixRepairsTerminalGapAndStartsReplacementAtAttemptOne(t *testing.T) {
	planValue := workflowSingleSectionPlan()
	service := workflowRecoveredPrefixService("evt_pending", planValue)
	service.events = append(service.events,
		workflowSectionGapEventWithID("evt_gap_1", "evt_pending", 1, "provider-plan", "ses_gap"),
		workflowSectionGapEventWithID("evt_gap_2", "evt_pending", 2, "provider-plan", "ses_gap"),
	)
	executor := &evidenceGapRetryWorkflowExecutor{
		repairText:   workflowSectionRepairResponse(1, 1, "Supported Replacement", "Explain a source-backed mechanism."),
		sectionTexts: []string{"# Supported Replacement\n\nSubstantive body."},
	}
	runner := workflowRunner(service, executor, &workflowObserver{})

	out, err := runner.RunLongFormPrefix(context.Background(), workflowLongFormInput("evt_pending", "serial"))
	if err != nil {
		t.Fatal(err)
	}
	if len(executor.repairRequests) != 1 || len(executor.sectionRequests) != 1 {
		t.Fatalf("repair calls=%d section calls=%d, want one each", len(executor.repairRequests), len(executor.sectionRequests))
	}
	request := executor.sectionRequests[0]
	if !strings.Contains(request.Prompt, "Section evidence attempt: 1 of 2") || !strings.Contains(request.Prompt, "Supported Replacement") {
		t.Fatalf("replacement Section did not receive a fresh bounded attempt:\n%s", request.Prompt)
	}
	if got := out.Plan.Parts[0].Sections[0].Title; got != "Supported Replacement" {
		t.Fatalf("effective plan title = %q", got)
	}
	if countWorkflowEvents(service.events, reporting.LongFormSectionPlanRepairCompletedEventType) != 1 || countWorkflowEvents(service.events, sectiondraft.CreatedEventType) != 1 {
		t.Fatalf("unexpected repair/Section event counts: %#v", service.events)
	}
}

func TestRunLongFormPrefixRejectsInvalidReplacementReferencesBeforeRepairEvent(t *testing.T) {
	planValue := workflowSingleSectionPlan()
	service := workflowRecoveredPrefixService("evt_pending", planValue)
	service.events = append(service.events,
		workflowSectionGapEventWithID("evt_gap_1", "evt_pending", 1, "provider-plan", "ses_gap"),
		workflowSectionGapEventWithID("evt_gap_2", "evt_pending", 2, "provider-plan", "ses_gap"),
	)
	service.validateRefsErr = errors.New("invalid source reference")
	repairText := string(mustWorkflowJSON(map[string]any{"replacements": []any{map[string]any{
		"part_index": 1, "section_index": 1,
		"section": map[string]any{
			"title": "Invalid Replacement", "purpose": "Explain an unsupported reference.",
			"target_refs": map[string]any{"snapshot_ids": []string{"snap_other_mission"}},
		},
	}}}))
	executor := &evidenceGapRetryWorkflowExecutor{repairText: repairText}
	runner := workflowRunner(service, executor, &workflowObserver{})

	_, err := runner.RunLongFormPrefix(context.Background(), workflowLongFormInput("evt_pending", "serial"))
	if err == nil || !errors.Is(err, producterror.ErrInvalidInput) {
		t.Fatalf("err = %v, want invalid input", err)
	}
	if service.validatedRefsMission != "mis_1" || len(service.validatedRefs) != 1 ||
		len(service.validatedRefs[0].SnapshotIDs) != 1 || service.validatedRefs[0].SnapshotIDs[0] != "snap_other_mission" {
		t.Fatalf("replacement refs were not validated in the report mission: mission=%q refs=%#v", service.validatedRefsMission, service.validatedRefs)
	}
	if countWorkflowEvents(service.events, reporting.LongFormSectionPlanRepairCompletedEventType) != 0 || len(executor.sectionRequests) != 0 {
		t.Fatalf("invalid replacement was recorded or drafted: events=%#v section calls=%d", service.events, len(executor.sectionRequests))
	}
}

func TestRunLongFormPrefixFanoutPreservesSuccessfulSectionDuringPlanRepair(t *testing.T) {
	planValue := reporting.SectionalReportPlan{Summary: "Explain two points.", Parts: []reporting.ReportPlanPart{{
		Title: "Core Part", Purpose: "Explain the core.", Sections: []reporting.ReportPlanSection{
			{Title: "Completed Section", Purpose: "Explain the supported point."},
			{Title: "Unsupported Section", Purpose: "Explain the unsupported point."},
		},
	}}}
	service := workflowRecoveredPrefixService("evt_pending", planValue)
	service.events = append(service.events,
		workflowSectionCreatedEventAt("evt_completed", "evt_pending", "art_completed", "completed-session", 1, 1, "Completed Section"),
		workflowSectionGapEventAt("evt_gap_1", "evt_pending", 1, "failed-session", "ses_gap", "", 1, 2),
		workflowSectionGapEventAt("evt_gap_2", "evt_pending", 2, "failed-session", "ses_gap", "", 1, 2),
	)
	service.artifacts = append(service.artifacts, artifact.Raw{
		ArtifactID: "art_completed", MissionID: "mis_1", MediaType: "text/markdown; charset=utf-8",
		Content: []byte("# Completed Section\n\nKeep this body."),
	})
	executor := &evidenceGapRetryWorkflowExecutor{
		repairText:   workflowSectionRepairResponse(1, 2, "Replacement Section", "Explain a supported second point."),
		sectionTexts: []string{"# Replacement Section\n\nNew body."},
	}
	runner := workflowRunner(service, executor, &workflowObserver{})

	out, err := runner.RunLongFormPrefix(context.Background(), workflowLongFormInput("evt_pending", "section_fanout"))
	if err != nil {
		t.Fatal(err)
	}
	if len(executor.sectionRequests) != 1 || !strings.Contains(executor.sectionRequests[0].Prompt, "Replacement Section") {
		t.Fatalf("only the replacement Section should run: %#v", executor.sectionRequests)
	}
	if got := out.Sections[0][0].ArtifactID; got != "art_completed" {
		t.Fatalf("completed Section artifact changed: %q", got)
	}
	if countWorkflowEvents(service.events, sectiondraft.CreatedEventType) != 2 {
		t.Fatalf("completed Section was rewritten: %#v", service.events)
	}
}

func TestRunLongFormPrefixFanoutRepairsAllTerminalGapsInOneRound(t *testing.T) {
	planValue := reporting.SectionalReportPlan{Summary: "Explain three points.", Parts: []reporting.ReportPlanPart{{
		Title: "Core Part", Purpose: "Explain the core.", Sections: []reporting.ReportPlanSection{
			{Title: "Completed Section", Purpose: "Explain the supported point."},
			{Title: "Unsupported A", Purpose: "Explain the first unsupported point."},
			{Title: "Unsupported B", Purpose: "Explain the second unsupported point."},
		},
	}}}
	service := workflowRecoveredPrefixService("evt_pending", planValue)
	service.events = append(service.events,
		workflowSectionCreatedEventAt("evt_completed", "evt_pending", "art_completed", "completed-session", 1, 1, "Completed Section"),
		workflowSectionGapEventAt("evt_gap_a1", "evt_pending", 1, "failed-a", "ses_a", "", 1, 2),
		workflowSectionGapEventAt("evt_gap_a2", "evt_pending", 2, "failed-a", "ses_a", "", 1, 2),
		workflowSectionGapEventAt("evt_gap_b1", "evt_pending", 1, "failed-b", "ses_b", "", 1, 3),
		workflowSectionGapEventAt("evt_gap_b2", "evt_pending", 2, "failed-b", "ses_b", "", 1, 3),
	)
	service.artifacts = append(service.artifacts, artifact.Raw{
		ArtifactID: "art_completed", MissionID: "mis_1", MediaType: "text/markdown; charset=utf-8",
		Content: []byte("# Completed Section\n\nKeep this body."),
	})
	repairText := string(mustWorkflowJSON(map[string]any{"replacements": []any{
		map[string]any{"part_index": 1, "section_index": 2, "section": map[string]any{"title": "Replacement A", "purpose": "Explain supported point A."}},
		map[string]any{"part_index": 1, "section_index": 3, "section": map[string]any{"title": "Replacement B", "purpose": "Explain supported point B."}},
	}}))
	executor := &evidenceGapRetryWorkflowExecutor{
		repairText: repairText, sectionTexts: []string{"Substantive body A.", "Substantive body B."},
	}
	runner := workflowRunner(service, executor, &workflowObserver{})

	out, err := runner.RunLongFormPrefix(context.Background(), workflowLongFormInput("evt_pending", "section_fanout"))
	if err != nil {
		t.Fatal(err)
	}
	if len(executor.repairRequests) != 1 || len(executor.sectionRequests) != 2 {
		t.Fatalf("repair calls=%d section calls=%d, want one repair and two replacements", len(executor.repairRequests), len(executor.sectionRequests))
	}
	prompts := executor.sectionRequests[0].Prompt + "\n" + executor.sectionRequests[1].Prompt
	if !strings.Contains(prompts, "Replacement A") || !strings.Contains(prompts, "Replacement B") {
		t.Fatalf("replacement prompts are incomplete:\n%s", prompts)
	}
	if out.Sections[0][0].ArtifactID != "art_completed" || countWorkflowEvents(service.events, sectiondraft.CreatedEventType) != 3 {
		t.Fatalf("fanout did not preserve one success and create two replacements: %#v", service.events)
	}
	if countWorkflowEvents(service.events, reporting.LongFormSectionPlanRepairCompletedEventType) != 1 {
		t.Fatalf("multiple repair events were recorded: %#v", service.events)
	}
}

func TestRunLongFormPrefixDoesNotRepairTwiceWhenReplacementStillLacksEvidence(t *testing.T) {
	planValue := workflowSingleSectionPlan()
	service := workflowRecoveredPrefixService("evt_pending", planValue)
	executor := &evidenceGapRetryWorkflowExecutor{
		repairText: workflowSectionRepairResponse(1, 1, "Replacement Section", "Explain a supported mechanism."),
		sectionTexts: []string{
			sectiondraft.EvidenceGapControlToken, sectiondraft.EvidenceGapControlToken,
			sectiondraft.EvidenceGapControlToken, sectiondraft.EvidenceGapControlToken,
		},
	}
	runner := workflowRunner(service, executor, &workflowObserver{})

	_, err := runner.RunLongFormPrefix(context.Background(), workflowLongFormInput("evt_pending", "serial"))
	if err == nil || !errors.Is(err, producterror.ErrConflict) {
		t.Fatalf("err = %v, want final Section conflict", err)
	}
	if len(executor.repairRequests) != 1 || len(executor.sectionRequests) != 4 {
		t.Fatalf("repair calls=%d section calls=%d, want one repair and four bounded attempts", len(executor.repairRequests), len(executor.sectionRequests))
	}
	if countWorkflowEvents(service.events, reporting.LongFormSectionPlanRepairCompletedEventType) != 1 || countWorkflowEvents(service.events, sectiondraft.EvidenceGapEventType) != 4 {
		t.Fatalf("unexpected bounded repair history: %#v", service.events)
	}
}

func TestRunLongFormPrefixResumeFailedRecoversExistingSectionPlanRepair(t *testing.T) {
	planValue := workflowSingleSectionPlan()
	service := workflowRecoveredPrefixService("evt_root", planValue)
	service.events = append(service.events,
		workflowSectionGapEventWithID("evt_gap_1", "evt_root", 1, "provider-plan", "ses_gap"),
		workflowSectionGapEventWithID("evt_gap_2", "evt_root", 2, "provider-plan", "ses_gap"),
		workflowSectionPlanRepairEvent("evt_root", reporting.ReportSectionPlanReplacement{
			ReportSectionCoordinate: reporting.ReportSectionCoordinate{PartIndex: 1, SectionIndex: 1},
			Section:                 reporting.ReportPlanSection{Title: "Recovered Replacement", Purpose: "Explain supported facts."},
		}),
		ledger.Event{EventID: "evt_root_failed", MissionID: "mis_1", EventType: "report.draft.failed", Payload: mustWorkflowJSON(map[string]any{
			"pending_event_id": "evt_root", "kind": "report_draft_failed",
		})},
		ledger.Event{EventID: "evt_retry", MissionID: "mis_1", EventType: "report.draft.pending", Producer: ledger.Producer{Type: "user", ID: "plasma-ui"}, Payload: mustWorkflowJSON(map[string]any{
			"origin_pending_event_id": "evt_root", "retry_of_pending_event_id": "evt_root", "retry_strategy": "resume_failed", "attempt_number": 2,
		})},
	)
	executor := &evidenceGapRetryWorkflowExecutor{sectionTexts: []string{"# Recovered Replacement\n\nBody."}}
	runner := workflowRunner(service, executor, &workflowObserver{})

	out, err := runner.RunLongFormPrefix(context.Background(), workflowLongFormInput("evt_retry", "serial"))
	if err != nil {
		t.Fatal(err)
	}
	if len(executor.repairRequests) != 0 || len(executor.sectionRequests) != 1 {
		t.Fatalf("existing repair should be reused: repair=%d section=%d", len(executor.repairRequests), len(executor.sectionRequests))
	}
	if !strings.Contains(executor.sectionRequests[0].Prompt, "Recovered Replacement") || !strings.Contains(executor.sectionRequests[0].Prompt, "Section evidence attempt: 1 of 2") {
		t.Fatalf("recovered effective plan was not used:\n%s", executor.sectionRequests[0].Prompt)
	}
	if out.Plan.Parts[0].Sections[0].Title != "Recovered Replacement" {
		t.Fatalf("unexpected effective plan: %#v", out.Plan)
	}
}

func TestRunLongFormPrefixRejectsRecoveredOrphanEvidenceGapAttemptTwo(t *testing.T) {
	planValue := workflowSingleSectionPlan()
	service := workflowRecoveredPrefixService("evt_pending", planValue)
	service.events = append(service.events, workflowSectionGapEvent("evt_pending", 2, "provider-plan", "ses_gap"))
	executor := &evidenceGapRetryWorkflowExecutor{}
	runner := workflowRunner(service, executor, &workflowObserver{})

	_, err := runner.RunLongFormPrefix(context.Background(), workflowLongFormInput("evt_pending", "serial"))
	if err == nil || !errors.Is(err, producterror.ErrConflict) {
		t.Fatalf("err = %v, want conflict for orphan attempt 2", err)
	}
	if len(executor.requests) != 0 {
		t.Fatalf("provider calls = %d, want 0", len(executor.requests))
	}
}

func TestRunLongFormPrefixRejectsRecoveredDuplicateEvidenceGapAttempt(t *testing.T) {
	planValue := workflowSingleSectionPlan()
	service := workflowRecoveredPrefixService("evt_pending", planValue)
	service.events = append(service.events,
		workflowSectionGapEventWithID("evt_gap_1", "evt_pending", 1, "provider-plan", "ses_gap"),
		workflowSectionGapEventWithID("evt_gap_1_dup", "evt_pending", 1, "provider-plan", "ses_gap"),
	)
	executor := &evidenceGapRetryWorkflowExecutor{}
	runner := workflowRunner(service, executor, &workflowObserver{})

	_, err := runner.RunLongFormPrefix(context.Background(), workflowLongFormInput("evt_pending", "serial"))
	if err == nil || !errors.Is(err, producterror.ErrConflict) {
		t.Fatalf("err = %v, want conflict for duplicate attempt", err)
	}
	if len(executor.requests) != 0 {
		t.Fatalf("provider calls = %d, want 0", len(executor.requests))
	}
}

func TestRunLongFormPrefixRejectsRecoveredConflictingEvidenceGapBinding(t *testing.T) {
	planValue := workflowSingleSectionPlan()
	service := workflowRecoveredPrefixService("evt_pending", planValue)
	service.events = append(service.events,
		workflowSectionGapEventWithID("evt_gap_1", "evt_pending", 1, "provider-plan", "ses_gap"),
		workflowSectionGapEventWithID("evt_gap_2", "evt_pending", 2, "provider-plan", "ses_other"),
	)
	executor := &evidenceGapRetryWorkflowExecutor{}
	runner := workflowRunner(service, executor, &workflowObserver{})

	_, err := runner.RunLongFormPrefix(context.Background(), workflowLongFormInput("evt_pending", "serial"))
	if err == nil || !errors.Is(err, producterror.ErrConflict) {
		t.Fatalf("err = %v, want conflict for changed retry binding", err)
	}
	if len(executor.requests) != 0 {
		t.Fatalf("provider calls = %d, want 0", len(executor.requests))
	}
}

func TestRunLongFormPrefixRejectsRecoveredConflictingEvidenceGapSourceLineage(t *testing.T) {
	planValue := workflowSingleSectionPlan()
	service := workflowRecoveredPrefixService("evt_pending", planValue)
	service.events = append(service.events,
		workflowSectionGapEventWithSourceID("evt_gap_1", "evt_pending", 1, "provider-plan", "ses_gap", "source-A"),
		workflowSectionGapEventWithSourceID("evt_gap_2", "evt_pending", 2, "provider-plan", "ses_gap", "source-B"),
	)
	executor := &evidenceGapRetryWorkflowExecutor{}
	runner := workflowRunner(service, executor, &workflowObserver{})

	_, err := runner.RunLongFormPrefix(context.Background(), workflowLongFormInput("evt_pending", "serial"))
	if err == nil || !errors.Is(err, producterror.ErrConflict) {
		t.Fatalf("err = %v, want conflict for changed source lineage", err)
	}
	if len(executor.requests) != 0 {
		t.Fatalf("provider calls = %d, want 0", len(executor.requests))
	}
}

func TestRunLongFormPrefixRecoveredCreatedSectionOverridesPriorGap(t *testing.T) {
	planValue := workflowSingleSectionPlan()
	service := workflowRecoveredPrefixService("evt_pending", planValue)
	service.events = append(service.events,
		workflowSectionGapEventWithID("evt_gap_1", "evt_pending", 1, "provider-plan", "ses_gap"),
		workflowSectionCreatedEvent("evt_pending", "art_section", "provider-plan"),
		workflowSectionGapEventWithID("evt_gap_2", "evt_pending", 2, "provider-plan", "ses_gap"),
	)
	service.artifacts = append(service.artifacts, artifact.Raw{
		ArtifactID: "art_section", MissionID: "mis_1", MediaType: "text/markdown; charset=utf-8",
		Content: []byte("# Core Section\n\nAlready created."),
	})
	executor := &evidenceGapRetryWorkflowExecutor{}
	runner := workflowRunner(service, executor, &workflowObserver{})

	_, err := runner.RunLongFormPrefix(context.Background(), workflowLongFormInput("evt_pending", "serial"))
	if err != nil {
		t.Fatal(err)
	}
	if len(executor.sectionRequests) != 0 || len(executor.partRequests) != 1 {
		t.Fatalf("created Section was not treated as complete: section=%d part=%d", len(executor.sectionRequests), len(executor.partRequests))
	}
}

func TestRunLongFormPrefixNewPendingDoesNotInheritPriorGapBudget(t *testing.T) {
	planValue := workflowSingleSectionPlan()
	service := workflowRecoveredPrefixService("evt_root", planValue)
	service.events = append(service.events,
		workflowSectionGapEvent("evt_root", 2, "provider-plan", "ses_gap"),
		ledger.Event{
			EventID: "evt_root_failed", MissionID: "mis_1", EventType: "report.draft.failed",
			Payload: mustWorkflowJSON(map[string]any{"pending_event_id": "evt_root", "kind": "report_draft_failed"}),
		},
		ledger.Event{
			EventID: "evt_retry", MissionID: "mis_1", EventType: "report.draft.pending",
			Producer: ledger.Producer{Type: "user", ID: "plasma-ui"},
			Payload: mustWorkflowJSON(map[string]any{
				"origin_pending_event_id": "evt_root", "retry_of_pending_event_id": "evt_root",
				"retry_strategy": "resume_failed", "attempt_number": 2,
			}),
		},
	)
	executor := &evidenceGapRetryWorkflowExecutor{sectionTexts: []string{"# Core Section\n\nFresh retry body."}}
	runner := workflowRunner(service, executor, &workflowObserver{})

	_, err := runner.RunLongFormPrefix(context.Background(), workflowLongFormInput("evt_retry", "serial"))
	if err != nil {
		t.Fatal(err)
	}
	if len(executor.sectionRequests) != 1 || !strings.Contains(executor.sectionRequests[0].Prompt, "Section evidence attempt: 1 of 2") {
		t.Fatalf("new pending inherited old gap budget: %#v", executor.sectionRequests)
	}
}

type evidenceGapRetryWorkflowExecutor struct {
	mu              sync.Mutex
	sectionTexts    []string
	requests        []agentexec.AgentRequest
	sectionRequests []agentexec.AgentRequest
	repairRequests  []agentexec.AgentRequest
	partRequests    []agentexec.AgentRequest
	repairText      string
	forks           []string
	forkSeq         int
}

func (executor *evidenceGapRetryWorkflowExecutor) Run(_ context.Context, req agentexec.AgentRequest) (agentexec.AgentResult, error) {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	executor.requests = append(executor.requests, req)
	switch {
	case req.UserText == "repair unsupported sections in long-form report plan":
		executor.repairRequests = append(executor.repairRequests, req)
		if executor.repairText == "" {
			return agentexec.AgentResult{SessionID: req.PreviousSessionID}, workflowErr("unexpected Section plan repair request")
		}
		return agentexec.AgentResult{Text: executor.repairText, SessionID: req.PreviousSessionID}, nil
	case strings.HasPrefix(req.UserText, "draft section"):
		executor.sectionRequests = append(executor.sectionRequests, req)
		if len(executor.sectionTexts) == 0 {
			return agentexec.AgentResult{SessionID: req.PreviousSessionID}, workflowErr("unexpected section request")
		}
		text := executor.sectionTexts[0]
		executor.sectionTexts = executor.sectionTexts[1:]
		return agentexec.AgentResult{Text: text, SessionID: req.PreviousSessionID}, nil
	case strings.HasPrefix(req.UserText, "assemble part"):
		executor.partRequests = append(executor.partRequests, req)
		return agentexec.AgentResult{Text: `{"intro":"Intro","transitions":[],"closing":"Close"}`, SessionID: req.PreviousSessionID}, nil
	default:
		return agentexec.AgentResult{SessionID: req.PreviousSessionID}, workflowErr("unexpected request: " + req.UserText)
	}
}

func (executor *evidenceGapRetryWorkflowExecutor) ForkSession(_ context.Context, sourceSessionID string) (agentexec.AgentSessionForkResult, error) {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	executor.forkSeq++
	sessionID := "fork-session-" + string(rune('A'+executor.forkSeq-1))
	executor.forks = append(executor.forks, sessionID)
	return agentexec.AgentSessionForkResult{SessionID: sessionID, SourceSessionID: sourceSessionID}, nil
}

func workflowSingleSectionPlan() reporting.SectionalReportPlan {
	return reporting.SectionalReportPlan{
		Summary: "Explain one point.",
		Parts: []reporting.ReportPlanPart{{
			Title: "Core Part", Purpose: "Explain the core.",
			Sections: []reporting.ReportPlanSection{{
				Title: "Core Section", Purpose: "Explain the mechanism directly.",
			}},
		}},
	}
}

func workflowSectionRepairResponse(partIndex, sectionIndex int, title, purpose string) string {
	return string(mustWorkflowJSON(map[string]any{"replacements": []any{map[string]any{
		"part_index": partIndex, "section_index": sectionIndex,
		"section": map[string]any{"title": title, "purpose": purpose},
	}}}))
}

func workflowSectionPlanRepairEvent(pendingID string, replacement reporting.ReportSectionPlanReplacement) ledger.Event {
	req := reporting.BuildLongFormSectionPlanRepairCompletedAppendRequest(reporting.LongFormSectionPlanRepairEventRequest{
		MarkdownReportEventBase: reporting.MarkdownReportEventBase{
			EventID: "evt_section_plan_repair", MissionID: "mis_1", PendingEventID: pendingID, Title: "Reader Report",
			AgentExecutor: "codex", AgentModel: "gpt-test", AgentReasoningEffort: "medium",
			AgentSessionID: "provider-plan", PreviousAgentSessionID: "provider-plan", ReturnedAgentSessionID: "provider-plan",
			ToolSessionID: "ses_plan_repair", MCPMode: "auto", ReportMode: reportexecution.ModeLongForm,
			ReportSessionPolicy:          reportexecution.SessionPolicyFreshSession,
			ReportSessionPolicySelection: reportexecution.SessionPolicySelectionAutoFreshSession,
			SessionChainKind:             "fresh_session_report", ReportPlanSessionID: "provider-plan", ReportSessionID: "provider-plan",
			CompositionStrategy: "sectional_preserve_markdown", Producer: ledger.Producer{Type: "agent_session", ID: "provider-plan"},
		},
		PlanEventID: "evt_plan", Coordinates: []reporting.ReportSectionCoordinate{replacement.ReportSectionCoordinate},
		Replacements: []reporting.ReportSectionPlanReplacement{replacement},
	})
	return ledger.Event{EventID: req.EventID, MissionID: req.MissionID, EventType: req.EventType, Producer: req.Producer, CausationEventID: req.CausationEventID, CorrelationID: req.CorrelationID, Payload: req.Payload}
}

func workflowLongFormInput(pendingID string, strategy string) DraftInput {
	input := draftInput(reportexecution.ModeLongForm)
	input.PendingEventID = pendingID
	input.ExecutionStrategy = strategy
	input.Title = "Reader Report"
	input.GenerationGuidanceProfile = ""
	return input
}

func workflowRecoveredPrefixService(pendingID string, planValue reporting.SectionalReportPlan) *workflowService {
	return &workflowService{events: []ledger.Event{
		workflowPendingEvent(pendingID),
		workflowPlanEvent(pendingID, planValue),
		workflowRequirementMapEvent(pendingID, planValue),
	}}
}

func workflowPendingEvent(pendingID string) ledger.Event {
	return ledger.Event{
		EventID: pendingID, MissionID: "mis_1", EventType: "report.draft.pending",
		Producer: ledger.Producer{Type: "user", ID: "plasma-ui"},
		Payload:  mustWorkflowJSON(map[string]any{"origin_pending_event_id": pendingID, "retry_strategy": "initial", "attempt_number": 1}),
	}
}

func workflowPlanEvent(pendingID string, planValue reporting.SectionalReportPlan) ledger.Event {
	return ledger.Event{
		EventID: "evt_plan", MissionID: "mis_1", EventType: "report.plan.created",
		Producer: ledger.Producer{Type: "agent_session", ID: "provider-plan"},
		Payload: mustWorkflowJSON(map[string]any{
			"pending_event_id": pendingID, "report_mode": reportexecution.ModeLongForm,
			"kind":        "sectional_markdown_report_plan",
			"artifact_id": "art_plan", "agent_session_id": "provider-plan",
			"agent_executor": "codex", "agent_model": "gpt-test", "agent_reasoning_effort": "medium",
			"mcp_mode": "auto", "report_session_policy": reportexecution.SessionPolicyFreshSession,
			"report_session_policy_selection": reportexecution.SessionPolicySelectionAutoFreshSession,
			"session_chain_kind":              "fresh_session_report", "report_plan_session_id": "provider-plan",
			"post_report_humanize": "disabled", "plan": planValue,
		}),
	}
}

func workflowRequirementMapEvent(pendingID string, planValue reporting.SectionalReportPlan) ledger.Event {
	requirementMap, err := reporting.NormalizeReportRequirementMap(reporting.ReportRequirementMap{ReviewedEventIDs: []string{pendingID}}, planValue)
	if err != nil {
		panic(err)
	}
	return ledger.Event{
		EventID: "evt_requirements", MissionID: "mis_1", EventType: reporting.ReportRequirementsMappedEventType,
		Producer: ledger.Producer{Type: "agent_session", ID: "provider-plan"},
		Payload: mustWorkflowJSON(map[string]any{
			"pending_event_id": pendingID, "plan_event_id": "evt_plan", "requirement_map": requirementMap,
		}),
	}
}

func workflowSectionGapEvent(pendingID string, attempt int, providerSessionID string, toolSessionID string) ledger.Event {
	return workflowSectionGapEventWithID("evt_gap", pendingID, attempt, providerSessionID, toolSessionID)
}

func workflowSectionGapEventWithID(eventID string, pendingID string, attempt int, providerSessionID string, toolSessionID string) ledger.Event {
	return workflowSectionGapEventWithSourceID(eventID, pendingID, attempt, providerSessionID, toolSessionID, "")
}

func workflowSectionGapEventWithSourceID(eventID string, pendingID string, attempt int, providerSessionID string, toolSessionID string, sourceSessionID string) ledger.Event {
	return workflowSectionGapEventAt(eventID, pendingID, attempt, providerSessionID, toolSessionID, sourceSessionID, 1, 1)
}

func workflowSectionGapEventAt(eventID string, pendingID string, attempt int, providerSessionID string, toolSessionID string, sourceSessionID string, partIndex, sectionIndex int) ledger.Event {
	payload := map[string]any{
		"pending_event_id": pendingID, "plan_event_id": "evt_plan",
		"part_index": partIndex, "section_index": sectionIndex, "attempt_number": attempt,
		"reason_code":    sectiondraft.EvidenceGapReasonCode,
		"agent_executor": "codex", "agent_session_id": providerSessionID,
		"previous_agent_session_id": providerSessionID, "returned_agent_session_id": providerSessionID,
		"tool_session_id": toolSessionID, "session_chain_kind": "fresh_session_report",
		"report_plan_session_id": "provider-plan", "report_session_id": providerSessionID,
		"duration_ms": float64(1),
	}
	if sourceSessionID != "" {
		payload["fork_source_agent_session_id"] = sourceSessionID
	}
	return ledger.Event{
		EventID: eventID, MissionID: "mis_1", EventType: sectiondraft.EvidenceGapEventType,
		Producer: ledger.Producer{Type: "agent_session", ID: providerSessionID},
		Payload:  mustWorkflowJSON(payload),
	}
}

func workflowSectionCreatedEvent(pendingID string, artifactID string, providerSessionID string) ledger.Event {
	return workflowSectionCreatedEventAt("evt_section_created", pendingID, artifactID, providerSessionID, 1, 1, "Core Section")
}

func workflowSectionCreatedEventAt(eventID, pendingID, artifactID, providerSessionID string, partIndex, sectionIndex int, title string) ledger.Event {
	return ledger.Event{
		EventID: eventID, MissionID: "mis_1", EventType: sectiondraft.CreatedEventType,
		Producer: ledger.Producer{Type: "agent_session", ID: providerSessionID},
		Payload: mustWorkflowJSON(map[string]any{
			"pending_event_id": pendingID, "plan_event_id": "evt_plan", "artifact_id": artifactID,
			"title": title, "agent_session_id": providerSessionID,
			"part_index": partIndex, "section_index": sectionIndex, "word_count": 4,
		}),
	}
}

func countWorkflowEvents(events []ledger.Event, eventType string) int {
	count := 0
	for _, event := range events {
		if event.EventType == eventType {
			count++
		}
	}
	return count
}

func latestWorkflowEvent(events []ledger.Event, eventType string) ledger.Event {
	for index := len(events) - 1; index >= 0; index-- {
		if events[index].EventType == eventType {
			return events[index]
		}
	}
	return ledger.Event{}
}

func workflowEventPayload(t *testing.T, event ledger.Event) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestRunLongFormPrefixRejectsRecoveredUnsupportedFinalPipeline(t *testing.T) {
	planValue := reporting.SectionalReportPlan{
		Summary: "Explain one point.",
		Parts: []reporting.ReportPlanPart{{
			Title: "Core Part",
			Sections: []reporting.ReportPlanSection{{
				Title: "Core Section", Purpose: "Explain the mechanism.",
			}},
		}},
	}
	service := &workflowService{events: []ledger.Event{
		{
			EventID: "evt_pending", MissionID: "mis_1", EventType: "report.draft.pending",
			Producer: ledger.Producer{Type: "user", ID: "plasma-ui"},
			Payload:  mustWorkflowJSON(map[string]any{"origin_pending_event_id": "evt_pending", "retry_strategy": "initial"}),
		},
		{
			EventID: "evt_plan", MissionID: "mis_1", EventType: "report.plan.created",
			Producer: ledger.Producer{Type: "agent_session", ID: "plan-session-1"},
			Payload: mustWorkflowJSON(map[string]any{
				"pending_event_id": "evt_pending", "report_mode": reportexecution.ModeLongForm,
				"artifact_id": "art_plan", "report_plan_session_id": "plan-session-1",
				"final_edit_pipeline": "reader_style_gate_future", "post_report_humanize": "disabled",
				"plan": planValue,
			}),
		},
	}}
	executor := &workflowExecutor{}
	observer := &workflowObserver{}
	runner := workflowRunner(service, executor, observer)

	_, err := runner.RunLongFormPrefix(context.Background(), DraftInput{
		MissionID: "mis_1", PendingEventID: "evt_pending", Title: "Reader Report",
		ReportMode: reportexecution.ModeLongForm, ExecutionStrategy: "serial",
		AgentExecutor: "codex", AgentModel: "gpt-test", MCPMode: "auto",
		ReportSessionPolicy: reportexecution.SessionPolicyFreshSession,
		PostReportHumanize:  "disabled",
	})
	if !errors.Is(err, producterror.ErrConflict) {
		t.Fatalf("err = %v, want conflict", err)
	}
	if len(executor.requests) != 0 {
		t.Fatalf("provider requests = %d, want 0", len(executor.requests))
	}
	assertObservedNodes(t, observer, []string{NodePlan, NodePlan})
}

func TestRunLongFormPrefixIgnoresOutOfPlanCreatedEventBeforeRequirements(t *testing.T) {
	planValue := reporting.SectionalReportPlan{
		Summary: "Explain one point.",
		Parts: []reporting.ReportPlanPart{{
			Title: "Core Part",
			Sections: []reporting.ReportPlanSection{{
				Title: "Core Section", Purpose: "Explain the mechanism.",
			}},
		}},
	}
	requirementMap, err := reporting.NormalizeReportRequirementMap(reporting.ReportRequirementMap{ReviewedEventIDs: []string{"evt_pending"}}, planValue)
	if err != nil {
		t.Fatal(err)
	}
	requirementHash, requirementJSON, err := reporting.ReportRequirementMapHash(requirementMap)
	if err != nil {
		t.Fatal(err)
	}
	service := &workflowService{
		events: []ledger.Event{
			{
				EventID: "evt_pending", MissionID: "mis_1", EventType: "report.draft.pending",
				Producer: ledger.Producer{Type: "user", ID: "plasma-ui"},
				Payload:  mustWorkflowJSON(map[string]any{"origin_pending_event_id": "evt_pending", "retry_strategy": "initial"}),
			},
			{
				EventID: "evt_plan", MissionID: "mis_1", EventType: "report.plan.created",
				Producer: ledger.Producer{Type: "agent_session", ID: "plan-session-1"},
				Payload: mustWorkflowJSON(map[string]any{
					"pending_event_id": "evt_pending", "report_mode": reportexecution.ModeLongForm,
					"artifact_id": "art_plan", "report_plan_session_id": "plan-session-1",
					"post_report_humanize": "disabled", "plan": planValue,
				}),
			},
			{
				EventID: "evt_section_out_of_plan", MissionID: "mis_1", EventType: "report.section.created",
				Producer: ledger.Producer{Type: "agent_session", ID: "provider-section"},
				Payload: mustWorkflowJSON(map[string]any{
					"pending_event_id": "evt_pending", "plan_event_id": "evt_plan",
					"artifact_id": "art_section_out_of_plan", "title": "Outside",
					"agent_session_id": "provider-section", "part_index": 2, "section_index": 1,
				}),
			},
		},
		artifacts: []artifact.Raw{{
			ArtifactID: "art_section_out_of_plan", MissionID: "mis_1",
			MediaType: "text/markdown; charset=utf-8", Content: []byte("# Outside\n\nValid but outside the plan."),
		}},
		reqSelect: app.ReportRequirementMapSelection{
			Event:              ledger.Event{EventID: "evt_requirements", MissionID: "mis_1", EventType: reporting.ReportRequirementsMappedEventType},
			RequirementMapHash: requirementHash,
			RequirementMap:     requirementJSON,
		},
	}
	executor := &workflowExecutor{results: []agentexec.AgentResult{
		{Text: reporting.ReportRequirementsMappedSentinel, SessionID: "plan-session-1"},
		{Text: "# Core Section\n\nBody.", SessionID: "plan-session-1"},
		{Text: `{"intro":"Intro","transitions":[],"closing":"Close"}`, SessionID: "plan-session-1"},
	}}
	runner := workflowRunner(service, executor, &workflowObserver{})

	_, err = runner.RunLongFormPrefix(context.Background(), DraftInput{
		MissionID: "mis_1", PendingEventID: "evt_pending", Title: "Reader Report",
		AgentExecutor: "codex", AgentModel: "gpt-test", AgentReasoningEffort: "medium",
		MCPMode: "auto", Rigor: reportprompt.RigorProfile{Level: "balanced", Label: "균형형"},
		ReportMode: reportexecution.ModeLongForm, ExecutionStrategy: "serial",
		ReportSessionPolicy:          reportexecution.SessionPolicyFreshSession,
		ReportSessionPolicySelection: reportexecution.SessionPolicySelectionAutoFreshSession,
		PostReportHumanize:           "disabled",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(executor.requests) == 0 || !strings.HasPrefix(executor.requests[0].UserText, "map explicit report requirements") {
		t.Fatalf("requirements mapping was skipped: %#v", executor.requests)
	}
}

func TestRunLongFormPrefixWrapsPlanAndRequirementAgentFailures(t *testing.T) {
	planValue := reporting.SectionalReportPlan{
		Summary: "Explain one point.",
		Parts: []reporting.ReportPlanPart{{Title: "Core Part", Sections: []reporting.ReportPlanSection{{
			Title: "Core Section", Purpose: "Explain the mechanism.",
		}}}},
	}
	requirementMap, err := reporting.NormalizeReportRequirementMap(reporting.ReportRequirementMap{ReviewedEventIDs: []string{"evt_pending"}}, planValue)
	if err != nil {
		t.Fatal(err)
	}
	requirementHash, requirementJSON, err := reporting.ReportRequirementMapHash(requirementMap)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name        string
		executor    *workflowExecutor
		wantStage   string
		wantSurface string
	}{
		{
			name:        "plan provider",
			executor:    &workflowExecutor{results: []agentexec.AgentResult{{Text: "partial", SessionID: "plan-session-1"}}, errs: []error{workflowErr("plan boom")}},
			wantStage:   "plan",
			wantSurface: "report_plan",
		},
		{
			name: "requirements provider",
			executor: &workflowExecutor{
				results: []agentexec.AgentResult{
					{Text: reporting.ReportPlanSubmittedSentinel, SessionID: "plan-session-1"},
					{Text: "partial", SessionID: "plan-session-1"},
				},
				errs: []error{nil, workflowErr("requirements boom")},
			},
			wantStage:   "requirements",
			wantSurface: "report_requirements",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			service := &workflowService{
				events: []ledger.Event{{
					EventID: "evt_pending", MissionID: "mis_1", EventType: "report.draft.pending",
					Producer: ledger.Producer{Type: "user", ID: "plasma-ui"},
					Payload:  mustWorkflowJSON(map[string]any{"origin_pending_event_id": "evt_pending", "retry_strategy": "initial"}),
				}},
				selection: reporting.ReportPlanSubmissionSelection{
					EventID: "evt_submitted", ArgumentsHash: "args", PlanHash: "plan_hash",
					Plan: mustWorkflowJSON(planValue),
				},
				reqSelect: app.ReportRequirementMapSelection{
					Event:              ledger.Event{EventID: "evt_requirements", MissionID: "mis_1", EventType: reporting.ReportRequirementsMappedEventType},
					RequirementMapHash: requirementHash,
					RequirementMap:     requirementJSON,
				},
			}
			runner := workflowRunner(service, tc.executor, &workflowObserver{})
			_, err := runner.RunLongFormPrefix(context.Background(), DraftInput{
				MissionID: "mis_1", PendingEventID: "evt_pending", Title: "Reader Report",
				AgentExecutor: "codex", AgentModel: "gpt-test", AgentReasoningEffort: "medium",
				MCPMode: "auto", Rigor: reportprompt.RigorProfile{Level: "balanced", Label: "균형형"},
				ReportMode: reportexecution.ModeLongForm, ExecutionStrategy: "serial",
				ReportSessionPolicy:          reportexecution.SessionPolicyFreshSession,
				ReportSessionPolicySelection: reportexecution.SessionPolicySelectionAutoFreshSession,
				PostReportHumanize:           "disabled",
			})
			if err == nil {
				t.Fatal("RunLongFormPrefix error = nil")
			}
			var stage *reportexecution.StageFailureError
			if !errors.As(err, &stage) || stage.Kind != tc.wantStage {
				t.Fatalf("stage failure = %#v, err %v", stage, err)
			}
			var payload reportexecution.FailurePayloadProvider
			if !errors.As(err, &payload) {
				t.Fatalf("failure payload missing from %v", err)
			}
			if got := payload.FailurePayload()["failed_surface"]; got != tc.wantSurface {
				t.Fatalf("failed_surface = %v, want %s", got, tc.wantSurface)
			}
		})
	}
}

func TestRunLongFormPrefixSectionFanoutPartPlansAreRaceFree(t *testing.T) {
	planValue := reporting.SectionalReportPlan{
		Summary: "Explain three parts.",
		Parts: []reporting.ReportPlanPart{
			{Title: "Part One", Sections: []reporting.ReportPlanSection{{Title: "S1", Purpose: "Draft one."}}},
			{Title: "Part Two", Sections: []reporting.ReportPlanSection{{Title: "S2", Purpose: "Draft two."}}},
			{Title: "Part Three", Sections: []reporting.ReportPlanSection{{Title: "S3", Purpose: "Draft three."}}},
		},
	}
	requirementMap, err := reporting.NormalizeReportRequirementMap(reporting.ReportRequirementMap{ReviewedEventIDs: []string{"evt_pending"}}, planValue)
	if err != nil {
		t.Fatal(err)
	}
	requirementHash, requirementJSON, err := reporting.ReportRequirementMapHash(requirementMap)
	if err != nil {
		t.Fatal(err)
	}
	service := &workflowService{
		events: []ledger.Event{{
			EventID: "evt_pending", MissionID: "mis_1", EventType: "report.draft.pending",
			Producer: ledger.Producer{Type: "user", ID: "plasma-ui"},
			Payload:  mustWorkflowJSON(map[string]any{"origin_pending_event_id": "evt_pending", "retry_strategy": "initial"}),
		}},
		selection: reporting.ReportPlanSubmissionSelection{
			EventID: "evt_submitted", ArgumentsHash: "args", PlanHash: "plan_hash",
			Plan: mustWorkflowJSON(planValue),
		},
		reqSelect: app.ReportRequirementMapSelection{
			Event:              ledger.Event{EventID: "evt_requirements", MissionID: "mis_1", EventType: reporting.ReportRequirementsMappedEventType},
			RequirementMapHash: requirementHash,
			RequirementMap:     requirementJSON,
		},
	}
	executor := &fanoutPartPlanRaceExecutor{}
	runner := workflowRunner(service, executor, &workflowObserver{})
	_, err = runner.RunLongFormPrefix(context.Background(), DraftInput{
		MissionID: "mis_1", PendingEventID: "evt_pending", Title: "Reader Report",
		AgentExecutor: "codex", AgentModel: "gpt-test", AgentReasoningEffort: "medium",
		MCPMode: "auto", Rigor: reportprompt.RigorProfile{Level: "balanced", Label: "균형형"},
		ReportMode: reportexecution.ModeLongForm, ExecutionStrategy: "section_fanout",
		ReportSessionPolicy:          reportexecution.SessionPolicyFreshSession,
		ReportSessionPolicySelection: reportexecution.SessionPolicySelectionAutoFreshSession,
		PostReportHumanize:           "disabled",
		GenerationGuidanceProfile:    reportprompt.ProfilePartConnectiveEconomyVoice,
	})
	var stage *reportexecution.StageFailureError
	if err == nil || !errors.As(err, &stage) || stage.Kind != "section" {
		t.Fatalf("RunLongFormPrefix err = %v, stage = %#v, want section failure", err, stage)
	}
	if got := atomic.LoadInt32(&executor.partPlanCalls); got != 3 {
		t.Fatalf("part plan calls = %d, want 3", got)
	}
	if got := atomic.LoadInt32(&executor.maxPartPlanInflight); got < 2 {
		t.Fatalf("max part plan concurrency = %d, want at least 2", got)
	}
}

type fanoutPartPlanRaceExecutor struct {
	mu                  sync.Mutex
	forkSeq             int
	partPlanCalls       int32
	partPlanInflight    int32
	maxPartPlanInflight int32
}

func (executor *fanoutPartPlanRaceExecutor) Run(_ context.Context, req agentexec.AgentRequest) (agentexec.AgentResult, error) {
	switch {
	case strings.HasPrefix(req.UserText, "plan section-fanout"):
		return agentexec.AgentResult{Text: reporting.ReportPlanSubmittedSentinel, SessionID: "plan-session-1"}, nil
	case strings.HasPrefix(req.UserText, "map explicit report requirements"):
		return agentexec.AgentResult{Text: reporting.ReportRequirementsMappedSentinel, SessionID: "plan-session-1"}, nil
	case strings.HasPrefix(req.UserText, "plan the reading flow for Part"):
		atomic.AddInt32(&executor.partPlanCalls, 1)
		inflight := atomic.AddInt32(&executor.partPlanInflight, 1)
		for {
			current := atomic.LoadInt32(&executor.maxPartPlanInflight)
			if inflight <= current || atomic.CompareAndSwapInt32(&executor.maxPartPlanInflight, current, inflight) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		atomic.AddInt32(&executor.partPlanInflight, -1)
		return agentexec.AgentResult{Text: "Use a distinct reading arc for this Part.", SessionID: req.PreviousSessionID}, nil
	case strings.HasPrefix(req.UserText, "draft section"):
		return agentexec.AgentResult{SessionID: req.PreviousSessionID}, workflowErr("stop after part planning")
	default:
		return agentexec.AgentResult{SessionID: req.PreviousSessionID}, workflowErr("unexpected request: " + req.UserText)
	}
}

func (executor *fanoutPartPlanRaceExecutor) ForkSession(_ context.Context, sourceSessionID string) (agentexec.AgentSessionForkResult, error) {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	executor.forkSeq++
	return agentexec.AgentSessionForkResult{SessionID: "fork-session-" + string(rune('A'+executor.forkSeq)), SourceSessionID: sourceSessionID}, nil
}
