package missionrecovery

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// FailurePolicy는 복구 step 오류가 plan을 중단하는지 결정한다.
type FailurePolicy string

const (
	// FailFast는 step이 오류를 반환하면 plan을 중단한다.
	FailFast FailurePolicy = "fail_fast"
	// BestEffort는 step 오류를 결과에 기록하고 다음 step을 계속 실행한다.
	BestEffort FailurePolicy = "best_effort"
)

const (
	// StepReportCompletion은 report completion 복구 step의 literal 이름이다.
	StepReportCompletion = "report_completion"
	// StepWorkflowState는 workflow state reconciliation step의 literal 이름이다.
	StepWorkflowState = "workflow_state"
	// StepReportDraft는 report draft 복구 step의 literal 이름이다.
	StepReportDraft = "report_draft"
	// StepDesignedReportExport는 designed report export step의 literal 이름이다.
	StepDesignedReportExport = "designed_report_export"
)

// Step은 이름, failure policy, 복구 callback을 가진 하나의 고정 plan step이다.
type Step struct {
	Name   string
	Policy FailurePolicy
	Run    func(context.Context) error
}

// ReportLock은 report lock을 획득하고 release 함수를 반환한다.
type ReportLock func() func()

// Plan은 두 pre-lock step과 하나의 report-lock scope 안의 두 report step으로
// 구성된 고정 mission recovery plan이다.
type Plan struct {
	PreLock           [2]Step
	AcquireReportLock ReportLock
	Report            [2]Step
}

// Result는 실행된 step의 exact name과 callback 오류를 보존한다.
type Result struct {
	Name string
	Err  error
}

// StepError는 fail-fast로 중단된 step의 이름과 원래 오류를 함께 보존한다.
type StepError struct {
	Name string
	Err  error
}

// Error는 error interface를 구현한다.
func (err *StepError) Error() string {
	return fmt.Sprintf("mission recovery step %q failed: %v", err.Name, err.Err)
}

// Unwrap는 errors.Is와 errors.As가 callback 오류를 찾을 수 있게 한다.
func (err *StepError) Unwrap() error {
	return err.Err
}

// ValidationError는 recovery plan의 모든 invalid 항목을 보고한다.
type ValidationError struct {
	Problems []error
}

// Error는 error interface를 구현한다.
func (err *ValidationError) Error() string {
	if len(err.Problems) == 0 {
		return "mission recovery plan is invalid"
	}
	return fmt.Sprintf("mission recovery plan is invalid: %v", errors.Join(err.Problems...))
}

// Unwrap는 errors.Is와 errors.As가 개별 validation 오류를 찾을 수 있게 한다.
func (err *ValidationError) Unwrap() []error {
	return err.Problems
}

var expectedSteps = [...]struct {
	name   string
	policy FailurePolicy
}{
	{name: StepReportCompletion, policy: FailFast},
	{name: StepWorkflowState, policy: BestEffort},
	{name: StepReportDraft, policy: FailFast},
	{name: StepDesignedReportExport, policy: FailFast},
}

var expectedStepNames = [...]string{
	StepReportCompletion,
	StepWorkflowState,
	StepReportDraft,
	StepDesignedReportExport,
}

// Validate는 callback이나 lock을 사용하지 않고 고정 plan 전체를 검증한다.
func Validate(plan Plan) error {
	steps := [...]Step{
		plan.PreLock[0],
		plan.PreLock[1],
		plan.Report[0],
		plan.Report[1],
	}
	problems := make([]error, 0)
	seen := make(map[string]int, len(steps))
	for index, step := range steps {
		name := strings.TrimSpace(step.Name)
		if name == "" {
			problems = append(problems, fmt.Errorf("step %d has an empty name", index))
		}
		if previous, ok := seen[step.Name]; ok {
			problems = append(problems, fmt.Errorf("step %d duplicates step %d name %q", index, previous, step.Name))
		} else {
			seen[step.Name] = index
		}
		if step.Name != expectedStepNames[index] {
			problems = append(problems, fmt.Errorf("step %d has name %q, want %q", index, step.Name, expectedStepNames[index]))
		}
		if step.Run == nil {
			problems = append(problems, fmt.Errorf("step %d (%q) has a nil callback", index, step.Name))
		}
		if step.Policy != FailFast && step.Policy != BestEffort {
			problems = append(problems, fmt.Errorf("step %d (%q) has invalid failure policy %q", index, step.Name, step.Policy))
		} else if step.Policy != expectedSteps[index].policy {
			problems = append(problems, fmt.Errorf("step %d (%q) has failure policy %q, want %q", index, step.Name, step.Policy, expectedSteps[index].policy))
		}
	}
	if plan.AcquireReportLock == nil {
		problems = append(problems, errors.New("report lock acquisition is nil"))
	}
	if len(problems) != 0 {
		return &ValidationError{Problems: problems}
	}
	return nil
}

// Run은 고정 mission recovery plan을 먼저 검증한 뒤 순서대로 실행한다. Best-effort
// 오류는 results에 남기되 반환 오류로 만들지 않는다. Fail-fast 오류는 이름이 있는
// StepError로 반환하고 뒤의 step을 실행하지 않는다.
func Run(ctx context.Context, plan Plan) ([]Result, error) {
	if err := Validate(plan); err != nil {
		return nil, err
	}
	results := make([]Result, 0, len(expectedStepNames))
	execute := func(step Step) error {
		err := step.Run(ctx)
		results = append(results, Result{Name: step.Name, Err: err})
		if err != nil && step.Policy == FailFast {
			return &StepError{Name: step.Name, Err: err}
		}
		return nil
	}
	for _, step := range plan.PreLock {
		if err := execute(step); err != nil {
			return results, err
		}
	}

	release := plan.AcquireReportLock()
	if release == nil {
		return results, errors.New("mission recovery report lock returned a nil release function")
	}
	defer release()
	for _, step := range plan.Report {
		if err := execute(step); err != nil {
			return results, err
		}
	}
	return results, nil
}
