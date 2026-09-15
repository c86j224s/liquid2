package articlepilot

import (
	"context"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/articleexperiment"
)

func TestExecutorRejectsContextualReportArm(t *testing.T) {
	executor := Executor{Bundle: articleexperiment.FixtureBundle{Fixture: articleexperiment.RealFixture{FixtureID: "M1"}}, Arm: articleexperiment.ArmBundle{Arm: articleexperiment.RealArm{ArmID: articleexperiment.ArmReport}}}
	var _ articleexperiment.Executor = executor
	output, err := executor.Execute(context.Background(), articleexperiment.ExecutionInput{Fixture: articleexperiment.ExecutionFixture{FixtureID: "M1"}, Arm: articleexperiment.ExecutionArm{ArmID: articleexperiment.ArmReport}})
	if err == nil || len(output.Attempts) != 1 || output.Attempts[0].Outcome != "failed" {
		t.Fatalf("output=%#v err=%v", output, err)
	}
}
