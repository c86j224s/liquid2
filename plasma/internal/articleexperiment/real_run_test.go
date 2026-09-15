package articleexperiment

import (
	"context"
	"strings"
	"testing"
)

func TestRunRealCellRejectsMissingPreflightAndReportArm(t *testing.T) {
	archive, repo, protocolPath, protocolSHA := writeRealProtocolFixture(t, func(*RealProtocol, map[string]*RealArm, map[string]*RealFixture, *BlindContract) {})
	config := RealRunConfig{ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: protocolSHA, RunID: "real-M1-A", ExecutorFactory: func(FixtureBundle, ArmBundle) (Executor, error) {
		return executorFunc(func(ExecutionInput) (ExecutionOutput, error) { return ExecutionOutput{}, nil }), nil
	}}
	if _, err := RunRealCell(context.Background(), config); err == nil {
		t.Fatal("missing live lock was accepted")
	}
	config.RunID = "real-M1-R"
	if _, err := RunRealCell(context.Background(), config); err == nil || !strings.Contains(err.Error(), "live comment") {
		t.Fatalf("R preflight error=%v", err)
	}
}
