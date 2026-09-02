package startuprecovery

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Step is one named startup reconciliation operation.
type Step struct {
	Name string
	Run  func(context.Context) (int, error)
}

// Result records the outcome of one startup reconciliation operation.
type Result struct {
	Name    string
	Changed int
	Err     error
}

// Run executes all steps in declared order. Errors are returned both on their
// individual results and as a named aggregate error.
func Run(ctx context.Context, steps []Step) ([]Result, error) {
	results := make([]Result, 0, len(steps))
	var stepErrors []error
	for index, step := range steps {
		name := strings.TrimSpace(step.Name)
		result := Result{Name: step.Name}
		if name == "" {
			result.Err = fmt.Errorf("startup recovery step %d has an empty name", index)
		} else if step.Run == nil {
			result.Err = fmt.Errorf("startup recovery step %q has a nil Run function", name)
		} else {
			result.Changed, result.Err = step.Run(ctx)
		}
		if result.Err != nil {
			stepErrors = append(stepErrors, fmt.Errorf("startup recovery step %q: %w", name, result.Err))
		}
		results = append(results, result)
	}
	return results, errors.Join(stepErrors...)
}
