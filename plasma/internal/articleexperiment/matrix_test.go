package articleexperiment

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func TestRunMatrixCompletesAllCellsAndRetainsFailedArm(t *testing.T) {
	archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
	executor := &fakeExecutor{failAt: "run-M2-A"}
	result, err := RunMatrix(context.Background(), MatrixConfig{
		ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: testFileSHA256(t, protocolPath),
		ExpectedFailedRun: "run-M2-A", Executor: executor, StartedAt: time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Runs) != 9 || result.Completed != 8 || result.Failed != 1 || len(executor.calls) != 9 || result.ManifestPath == "" || !validSHA256(result.ManifestSHA256) {
		t.Fatalf("matrix result = %#v, calls=%d", result, len(executor.calls))
	}
	manifest, validateErr := ValidateMatrix(archive, repo, protocolPath, testFileSHA256(t, protocolPath), result.ManifestSHA256)
	if validateErr != nil || manifest.Completed != 8 || manifest.Failed != 1 || len(manifest.Runs) != 9 {
		t.Fatalf("matrix manifest = %#v, err=%v", manifest, validateErr)
	}
	if result.Runs[5].Terminal.Pending.RunID != "run-M2-A" || result.Runs[5].Terminal.Status != "failed" {
		t.Fatalf("failed cell was not retained: %#v", result.Runs[5].Terminal)
	}
}

func TestValidateMatrixRejectsChangedRunTerminal(t *testing.T) {
	archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
	result, err := RunMatrix(context.Background(), MatrixConfig{
		ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: testFileSHA256(t, protocolPath),
		ExpectedFailedRun: "run-M2-A", Executor: &fakeExecutor{failAt: "run-M2-A"},
	})
	if err != nil {
		t.Fatal(err)
	}
	terminalPath := result.Runs[0].TerminalPath
	raw, err := os.ReadFile(terminalPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(terminalPath, append(raw, ' '), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateMatrix(archive, repo, protocolPath, testFileSHA256(t, protocolPath), result.ManifestSHA256); !errors.Is(err, producterror.ErrConflict) {
		t.Fatalf("error = %v, want conflict", err)
	}
}

func TestValidateMatrixRejectsAlternateTerminalReceiptPath(t *testing.T) {
	archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
	result, err := RunMatrix(context.Background(), MatrixConfig{
		ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: testFileSHA256(t, protocolPath),
		ExpectedFailedRun: "run-M2-A", Executor: &fakeExecutor{failAt: "run-M2-A"},
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest := result.Manifest
	pendingPath := filepath.Join(archive, "runs", manifest.Runs[0].RunID, "run.pending.json")
	manifest.Runs[0].TerminalPath = filepath.ToSlash(filepath.Join("runs", manifest.Runs[0].RunID, "run.pending.json"))
	manifest.Runs[0].TerminalSHA256 = testFileSHA256(t, pendingPath)
	writeTestJSON(t, result.ManifestPath, manifest)
	matrixSHA := testFileSHA256(t, result.ManifestPath)
	if _, err := ValidateMatrix(archive, repo, protocolPath, testFileSHA256(t, protocolPath), matrixSHA); !errors.Is(err, producterror.ErrInvalidInput) {
		t.Fatalf("error = %v, want invalid input", err)
	}
}

func TestValidateMatrixRejectsMissingOrWrongExpectedFailure(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(*MatrixManifest)
	}{
		{name: "missing expected failure", edit: func(manifest *MatrixManifest) {
			manifest.ExpectedFailedRun = ""
		}},
		{name: "wrong expected failure", edit: func(manifest *MatrixManifest) {
			manifest.ExpectedFailedRun = "run-M1-A"
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
			result, err := RunMatrix(context.Background(), MatrixConfig{
				ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: testFileSHA256(t, protocolPath),
				ExpectedFailedRun: "run-M2-A", Executor: &fakeExecutor{failAt: "run-M2-A"},
			})
			if err != nil {
				t.Fatal(err)
			}
			manifest := result.Manifest
			test.edit(&manifest)
			writeTestJSON(t, result.ManifestPath, manifest)
			if _, err := ValidateMatrix(archive, repo, protocolPath, testFileSHA256(t, protocolPath), testFileSHA256(t, result.ManifestPath)); err == nil {
				t.Fatal("invalid expected failure was accepted")
			}
		})
	}
}

func TestValidateForcedFailureRejectsArtifactFailureAndIncompleteResult(t *testing.T) {
	makeResult := func() MatrixResult {
		result := MatrixResult{Completed: 8, Failed: 1, Runs: make([]Result, 9)}
		for index := range result.Runs {
			result.Runs[index].Terminal = TerminalManifest{Status: "completed", Pending: PendingManifest{RunID: fmt.Sprintf("run-%d", index)}}
		}
		result.Runs[0].Terminal = TerminalManifest{
			Status: "failed", Pending: PendingManifest{RunID: "run-M2-A"},
			Failure: &FailureReceipt{Class: "executor_failed", Message: "synthetic arm executor failed"},
		}
		return result
	}
	t.Run("artifact failure", func(t *testing.T) {
		result := makeResult()
		result.Runs[0].Terminal.Failure = &FailureReceipt{Class: "artifact_failed", Message: "synthetic arm artifact storage failed"}
		if err := ValidateForcedFailure(result, "run-M2-A"); !errors.Is(err, producterror.ErrConflict) {
			t.Fatalf("error = %v, want conflict", err)
		}
	})
	t.Run("incomplete result", func(t *testing.T) {
		result := makeResult()
		result.Runs[1].Terminal = TerminalManifest{}
		if err := ValidateForcedFailure(result, "run-M2-A"); !errors.Is(err, producterror.ErrConflict) {
			t.Fatalf("error = %v, want conflict", err)
		}
	})
}

func TestValidateMatrixRejectsInvalidCompletionTime(t *testing.T) {
	archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
	result, err := RunMatrix(context.Background(), MatrixConfig{
		ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: testFileSHA256(t, protocolPath),
		ExpectedFailedRun: "run-M2-A", Executor: &fakeExecutor{failAt: "run-M2-A"},
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest := result.Manifest
	manifest.CompletedAt = "not-a-time"
	writeTestJSON(t, result.ManifestPath, manifest)
	if _, err := ValidateMatrix(archive, repo, protocolPath, testFileSHA256(t, protocolPath), testFileSHA256(t, result.ManifestPath)); !errors.Is(err, producterror.ErrInvalidInput) {
		t.Fatalf("error = %v, want invalid input", err)
	}
}

func TestRunMatrixRejectsExistingRunBeforeExecution(t *testing.T) {
	archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
	if err := os.MkdirAll(filepath.Join(archive, "runs", "preexisting"), 0o700); err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{}
	_, err := RunMatrix(context.Background(), MatrixConfig{
		ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath,
		ProtocolSHA256: testFileSHA256(t, protocolPath), ExpectedFailedRun: "run-M2-A", Executor: executor,
	})
	if !errors.Is(err, producterror.ErrConflict) || len(executor.calls) != 0 {
		t.Fatalf("error=%v calls=%d", err, len(executor.calls))
	}
}

func TestValidateMatrixRejectsImpossibleChronology(t *testing.T) {
	archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
	result, err := RunMatrix(context.Background(), MatrixConfig{
		ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath,
		ProtocolSHA256: testFileSHA256(t, protocolPath), ExpectedFailedRun: "run-M2-A", Executor: &fakeExecutor{failAt: "run-M2-A"},
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest := result.Manifest
	manifest.CompletedAt = "2000-01-01T00:00:00.000000000Z"
	writeTestJSON(t, result.ManifestPath, manifest)
	if _, err := ValidateMatrix(archive, repo, protocolPath, testFileSHA256(t, protocolPath), testFileSHA256(t, result.ManifestPath)); !errors.Is(err, producterror.ErrConflict) {
		t.Fatalf("error = %v, want conflict", err)
	}
}

func TestRunRejectsUnassignedDuplicateAndExistingRunIDs(t *testing.T) {
	t.Run("unassigned", func(t *testing.T) {
		archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
		_, err := Run(context.Background(), Config{ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: testFileSHA256(t, protocolPath), RunID: "run-other", Executor: &fakeExecutor{}})
		if !errors.Is(err, producterror.ErrInvalidInput) {
			t.Fatalf("error = %v, want invalid input", err)
		}
	})
	t.Run("duplicate protocol run ID", func(t *testing.T) {
		archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{duplicateRunID: true})
		_, err := Run(context.Background(), Config{ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: testFileSHA256(t, protocolPath), RunID: "run-M1-R", Executor: &fakeExecutor{}})
		if !errors.Is(err, producterror.ErrInvalidInput) {
			t.Fatalf("error = %v, want invalid input", err)
		}
	})
	t.Run("existing run", func(t *testing.T) {
		archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
		config := Config{ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: testFileSHA256(t, protocolPath), RunID: "run-M1-R", Executor: &fakeExecutor{}}
		if _, err := Run(context.Background(), config); err != nil {
			t.Fatal(err)
		}
		if _, err := Run(context.Background(), config); !errors.Is(err, producterror.ErrConflict) {
			t.Fatalf("second run error = %v, want conflict", err)
		}
	})
}
