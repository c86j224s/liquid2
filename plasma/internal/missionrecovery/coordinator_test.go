package missionrecovery

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func validPlan(calls *[]string) Plan {
	step := func(name string, policy FailurePolicy) Step {
		return Step{
			Name:   name,
			Policy: policy,
			Run: func(context.Context) error {
				*calls = append(*calls, name)
				return nil
			},
		}
	}
	return Plan{
		PreLock: [2]Step{
			step(StepReportCompletion, FailFast),
			step(StepWorkflowState, BestEffort),
		},
		AcquireReportLock: func() func() {
			*calls = append(*calls, "acquire_lock")
			return func() { *calls = append(*calls, "release_lock") }
		},
		Report: [2]Step{
			step(StepReportDraft, FailFast),
			step(StepDesignedReportExport, FailFast),
		},
	}
}

func TestRunPreservesFixedOrderAndResults(t *testing.T) {
	var calls []string
	results, err := Run(context.Background(), validPlan(&calls))
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{
		StepReportCompletion,
		StepWorkflowState,
		"acquire_lock",
		StepReportDraft,
		StepDesignedReportExport,
		"release_lock",
	}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
	wantResults := []Result{
		{Name: StepReportCompletion},
		{Name: StepWorkflowState},
		{Name: StepReportDraft},
		{Name: StepDesignedReportExport},
	}
	if !reflect.DeepEqual(results, wantResults) {
		t.Fatalf("results = %#v, want %#v", results, wantResults)
	}
}

func TestRunValidatesBeforeCallbacksOrLock(t *testing.T) {
	var calls []string
	plan := validPlan(&calls)
	plan.Report[1].Name = "wrong"
	plan.AcquireReportLock = func() func() {
		calls = append(calls, "acquire_lock")
		return func() { calls = append(calls, "release_lock") }
	}
	results, err := Run(context.Background(), plan)
	if err == nil {
		t.Fatal("Run returned nil error for invalid plan")
	}
	if results != nil {
		t.Fatalf("results = %#v, want nil", results)
	}
	if len(calls) != 0 {
		t.Fatalf("callbacks or lock ran before validation: %v", calls)
	}
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error = %v, want ValidationError", err)
	}
}

func TestValidateRejectsDuplicateEmptyAndWrongNames(t *testing.T) {
	cases := []struct {
		name string
		edit func(*Plan)
		want []string
	}{
		{
			name: "duplicate",
			edit: func(plan *Plan) { plan.PreLock[1].Name = StepReportCompletion },
			want: []string{"duplicates", "want"},
		},
		{
			name: "empty",
			edit: func(plan *Plan) { plan.Report[0].Name = "  " },
			want: []string{"empty", "want"},
		},
		{
			name: "wrong order",
			edit: func(plan *Plan) {
				plan.PreLock[0].Name = StepWorkflowState
				plan.PreLock[1].Name = StepReportCompletion
			},
			want: []string{"step 0", "step 1"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls []string
			plan := validPlan(&calls)
			tc.edit(&plan)
			err := Validate(plan)
			if err == nil {
				t.Fatal("Validate returned nil")
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("error = %v, want substring %q", err, want)
				}
			}
		})
	}
}

func TestValidateRejectsStepPolicyMismatchBeforeExecution(t *testing.T) {
	cases := []struct {
		name string
		edit func(*Plan)
		want string
	}{
		{
			name: "workflow state must be best effort",
			edit: func(plan *Plan) { plan.PreLock[1].Policy = FailFast },
			want: "workflow_state",
		},
		{
			name: "report draft must be fail fast",
			edit: func(plan *Plan) { plan.Report[0].Policy = BestEffort },
			want: "report_draft",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls []string
			plan := validPlan(&calls)
			tc.edit(&plan)
			_, err := Run(context.Background(), plan)
			if err == nil {
				t.Fatal("Run returned nil for policy mismatch")
			}
			if len(calls) != 0 {
				t.Fatalf("callbacks or lock ran before policy validation: %v", calls)
			}
			if !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "failure policy") {
				t.Fatalf("error = %v, want named policy mismatch", err)
			}
		})
	}
}

func TestValidateRejectsNilCallbackInvalidPolicyAndNilLock(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Plan)
		want string
	}{
		{name: "nil callback", edit: func(plan *Plan) { plan.PreLock[0].Run = nil }, want: "nil callback"},
		{name: "invalid policy", edit: func(plan *Plan) { plan.PreLock[1].Policy = FailurePolicy("retry") }, want: "invalid failure policy"},
		{name: "nil lock", edit: func(plan *Plan) { plan.AcquireReportLock = nil }, want: "lock acquisition is nil"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var calls []string
			plan := validPlan(&calls)
			tc.edit(&plan)
			err := Validate(plan)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestRunBestEffortKeepsErrorAndContinues(t *testing.T) {
	var calls []string
	plan := validPlan(&calls)
	workflowErr := errors.New("workflow reconciliation failed")
	plan.PreLock[1].Run = func(context.Context) error {
		calls = append(calls, StepWorkflowState)
		return workflowErr
	}
	results, err := Run(context.Background(), plan)
	if err != nil {
		t.Fatalf("Run returned best-effort error: %v", err)
	}
	if !reflect.DeepEqual(calls, []string{
		StepReportCompletion,
		StepWorkflowState,
		"acquire_lock",
		StepReportDraft,
		StepDesignedReportExport,
		"release_lock",
	}) {
		t.Fatalf("calls = %v", calls)
	}
	if results[1].Name != StepWorkflowState || results[1].Err != workflowErr {
		t.Fatalf("workflow result = %#v", results[1])
	}
}

func TestRunFailFastBeforeLock(t *testing.T) {
	var calls []string
	plan := validPlan(&calls)
	completionErr := errors.New("completion failed")
	plan.PreLock[0].Run = func(context.Context) error {
		calls = append(calls, StepReportCompletion)
		return completionErr
	}
	results, err := Run(context.Background(), plan)
	if !errors.Is(err, completionErr) {
		t.Fatalf("error = %v, want completion error", err)
	}
	var stepErr *StepError
	if !errors.As(err, &stepErr) || stepErr.Name != StepReportCompletion {
		t.Fatalf("error = %v, want named StepError", err)
	}
	if !reflect.DeepEqual(calls, []string{StepReportCompletion}) {
		t.Fatalf("calls = %v, lock or later step ran", calls)
	}
	if !reflect.DeepEqual(results, []Result{{Name: StepReportCompletion, Err: completionErr}}) {
		t.Fatalf("results = %#v", results)
	}
}

func TestRunFailFastUnderLockReleases(t *testing.T) {
	var calls []string
	plan := validPlan(&calls)
	draftErr := errors.New("draft failed")
	plan.Report[0].Run = func(context.Context) error {
		calls = append(calls, StepReportDraft)
		return draftErr
	}
	results, err := Run(context.Background(), plan)
	if !errors.Is(err, draftErr) {
		t.Fatalf("error = %v, want draft error", err)
	}
	if !reflect.DeepEqual(calls, []string{
		StepReportCompletion,
		StepWorkflowState,
		"acquire_lock",
		StepReportDraft,
		"release_lock",
	}) {
		t.Fatalf("calls = %v, want lock release after failure", calls)
	}
	if !reflect.DeepEqual(results, []Result{
		{Name: StepReportCompletion},
		{Name: StepWorkflowState},
		{Name: StepReportDraft, Err: draftErr},
	}) {
		t.Fatalf("results = %#v", results)
	}
}

func TestRunAcquiresAndReleasesReportLockExactlyOnce(t *testing.T) {
	var acquires, releases int
	var calls []string
	plan := validPlan(&calls)
	plan.AcquireReportLock = func() func() {
		acquires++
		return func() { releases++ }
	}
	if _, err := Run(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	if acquires != 1 || releases != 1 {
		t.Fatalf("acquires=%d releases=%d, want one each", acquires, releases)
	}

	plan.Report[1].Run = func(context.Context) error { return errors.New("export failed") }
	if _, err := Run(context.Background(), plan); err == nil {
		t.Fatal("Run returned nil error for report failure")
	}
	if acquires != 2 || releases != 2 {
		t.Fatalf("after error acquires=%d releases=%d, want two each", acquires, releases)
	}
}

func TestRunRejectsNilReleaseAfterPreLockExecution(t *testing.T) {
	var calls []string
	plan := validPlan(&calls)
	plan.AcquireReportLock = func() func() { return nil }
	_, err := Run(context.Background(), plan)
	if err == nil || !strings.Contains(err.Error(), "nil release") {
		t.Fatalf("error = %v, want nil release error", err)
	}
	if !reflect.DeepEqual(calls, []string{StepReportCompletion, StepWorkflowState}) {
		t.Fatalf("calls = %v, report steps ran unexpectedly", calls)
	}
}
