package reportexperiment

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
)

func TestRunPropagatesSuccessfulPathCloseErrorBeforeManifest(t *testing.T) {
	ctx := context.Background()
	archive := t.TempDir()
	repo := t.TempDir()
	fixturePath := writeTestFixture(t, archive, "close-error")
	closeErr := errors.New("close failed")
	executor := &fakeV3Executor{}
	_, err := Run(ctx, Config{
		ArchiveRoot: archive, FixturePath: fixturePath, RunID: "close-error", RepositoryRoot: repo,
		AgentModel: "gpt-test", AgentReasoningEffort: "medium",
		ServiceFactory: func(ctx context.Context, dbPath string) (ServiceHandle, error) {
			handle, err := sqliteServiceFactory(ctx, dbPath)
			if err != nil {
				return ServiceHandle{}, err
			}
			return ServiceHandle{Service: handle.Service, Close: func() error {
				if handle.Close != nil {
					_ = handle.Close()
				}
				return closeErr
			}}, nil
		},
		BinaryPair: BinaryPair{
			Experiment: BinaryMetadata{Path: "/tmp/plasma-report-experiment", SHA256: strings.Repeat("a", 64)},
			PlasmaMCP:  BinaryMetadata{Path: "/tmp/plasma", SHA256: strings.Repeat("b", 64)},
			Codex:      BinaryMetadata{Path: "/tmp/codex", SHA256: strings.Repeat("c", 64)},
		},
		ExecutorFactory: func(_ context.Context, input ExecutorContext) (agentexec.AgentExecutor, error) {
			executor.store = input.Service
			return executor, nil
		},
	})
	if !errors.Is(err, closeErr) {
		t.Fatalf("Run error = %v, want close error", err)
	}
	if _, statErr := os.Stat(filepath.Join(archive, "runs", "close-error", "manifest.json")); !os.IsNotExist(statErr) {
		t.Fatalf("manifest should not be written after close failure: %v", statErr)
	}
}
