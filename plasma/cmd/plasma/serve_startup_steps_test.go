package main

import (
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestServeStartupRecoveryStepsOrderAndNames(t *testing.T) {
	steps := serveStartupRecoverySteps(app.NewService(nil))
	if len(steps) != 2 {
		t.Fatalf("step count=%d", len(steps))
	}
	if steps[0].Name != "source_candidate_staging" || steps[1].Name != "report_completion" {
		t.Fatalf("unexpected steps: %#v", steps)
	}
	if steps[0].Run == nil || steps[1].Run == nil {
		t.Fatal("startup recovery step has nil runner")
	}
}
