package articleexperiment

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func TestRunPreservesSuccessFailureAndMatchedInputs(t *testing.T) {
	archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
	executor := &fakeExecutor{failAt: "run-M2-A"}
	seenInputByCell := map[string]string{}
	seenArmContract := map[string]string{}
	for _, fixtureID := range requiredFixtureIDs {
		for _, armID := range requiredArmIDs {
			runID := "run-" + fixtureID + "-" + armID
			result, err := Run(context.Background(), Config{
				ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: testFileSHA256(t, protocolPath),
				RunID: runID, Executor: executor, StartedAt: time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC),
			})
			if runID == executor.failAt {
				if err == nil || strings.Contains(err.Error(), "private") || result.Terminal.Status != "failed" || result.Terminal.Failure == nil || strings.Contains(result.Terminal.Failure.Message, "private") {
					t.Fatalf("failed run result = %#v, err=%v", result, err)
				}
			} else if err != nil || result.Terminal.Status != "completed" {
				t.Fatalf("Run(%s): result=%#v err=%v", runID, result, err)
			}
			terminal, validateErr := ValidateTerminal(result.RunDir)
			if validateErr != nil || terminal.Status != result.Terminal.Status {
				t.Fatalf("ValidateTerminal(%s): %#v, %v", runID, terminal, validateErr)
			}
			call := executor.calls[len(executor.calls)-1]
			if call.Cell.RunID != runID || call.Cell.FixtureID != fixtureID || call.Cell.ArmID != armID || call.Fixture.FixtureID != fixtureID || call.Arm.ArmID != armID || call.ProtocolID != "pilot-v1" || call.ProtocolSHA256 != testFileSHA256(t, protocolPath) || call.Arm.ContextualOnly != (armID == "R") {
				t.Fatalf("executor received wrong cell identity: %#v", call)
			}
			seenInputByCell[fixtureID+"/"+armID] = matchedInputIdentity(call)
			seenArmContract[armID] = call.ArmContract.SHA256
			expectedInput := []byte("original fixture " + fixtureID + "\n")
			if len(call.Inputs) != 1 || call.Inputs[0].ID != "source" || call.Inputs[0].SHA256 != bytesSHA256(expectedInput) || string(call.Inputs[0].Content) != string(expectedInput) {
				t.Fatalf("executor received wrong fixture bytes: %#v", call.Inputs)
			}
			expectedContract := []byte("contract/" + armID + "\n")
			if call.Arm.ContractSHA256 != call.ArmContract.SHA256 || call.ArmContract.SHA256 != bytesSHA256(expectedContract) || string(call.ArmContract.Content) != string(expectedContract) {
				t.Fatalf("executor received wrong arm contract: arm=%#v contract=%#v", call.Arm, call.ArmContract)
			}
			if terminal.Pending.ProtocolID != call.ProtocolID || terminal.Pending.ProtocolSHA256 != call.ProtocolSHA256 || terminal.Pending.FixtureID != call.Fixture.FixtureID || terminal.Pending.ArmID != call.Arm.ArmID {
				t.Fatalf("terminal identity differs from execution input: %#v vs %#v", terminal.Pending, call)
			}
		}
	}
	if len(executor.calls) != 9 {
		t.Fatalf("executor calls = %d, want 9", len(executor.calls))
	}
	for _, fixtureID := range requiredFixtureIDs {
		if seenInputByCell[fixtureID+"/E"] != seenInputByCell[fixtureID+"/A"] {
			t.Fatalf("%s E/A fixture inputs differ", fixtureID)
		}
	}
	if seenArmContract["E"] == seenArmContract["A"] || !validSHA256(seenArmContract["E"]) || !validSHA256(seenArmContract["A"]) {
		t.Fatalf("E/A arm contracts must be distinct valid receipts: %#v", seenArmContract)
	}
}

func TestRunRejectsFutureStartBeforeConsumingRun(t *testing.T) {
	archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
	config := Config{
		ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath,
		ProtocolSHA256: testFileSHA256(t, protocolPath), RunID: "run-M1-A",
		Executor: &fakeExecutor{}, StartedAt: time.Now().Add(time.Hour),
	}
	if _, err := Run(context.Background(), config); !errors.Is(err, producterror.ErrInvalidInput) {
		t.Fatalf("error = %v, want invalid input", err)
	}
	if _, err := os.Stat(filepath.Join(archive, "runs")); !os.IsNotExist(err) {
		t.Fatalf("future start consumed run directory: %v", err)
	}
	config.StartedAt = time.Time{}
	if _, err := Run(context.Background(), config); err != nil {
		t.Fatalf("valid retry failed: %v", err)
	}
}

func TestRunPersistsCanceledExecutorAsFailedTerminal(t *testing.T) {
	archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
	executor := executorFunc(func(ExecutionInput) (ExecutionOutput, error) {
		return ExecutionOutput{Attempts: []AttemptReceipt{{Attempt: 1, Kind: "semantic", Outcome: "failed"}}}, context.Canceled
	})
	result, err := Run(context.Background(), Config{
		ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath,
		ProtocolSHA256: testFileSHA256(t, protocolPath), RunID: "run-M1-A", Executor: executor,
	})
	if !errors.Is(err, context.Canceled) || result.Terminal.Status != "failed" || result.TerminalPath == "" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if _, validateErr := ValidateTerminal(result.RunDir); validateErr != nil {
		t.Fatalf("canceled terminal did not validate: %v", validateErr)
	}
}

func TestRunRetainsArtifactsReturnedWithExecutorFailure(t *testing.T) {
	archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
	executor := executorFunc(func(ExecutionInput) (ExecutionOutput, error) {
		return ExecutionOutput{
			Attempts:  []AttemptReceipt{{Attempt: 1, Kind: "semantic", Outcome: "failed"}},
			Artifacts: []OutputArtifact{{Kind: "markdown", MediaType: "text/markdown", Filename: "partial.md", Content: []byte("partial")}},
		}, errors.New("private provider detail")
	})
	result, err := Run(context.Background(), Config{ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: testFileSHA256(t, protocolPath), RunID: "run-M1-A", Executor: executor})
	if err == nil || strings.Contains(err.Error(), "private") || result.Terminal.Status != "failed" || len(result.Terminal.PartialArtifacts) != 1 || result.Terminal.PartialArtifacts[0].Filename != "partial.md" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if _, validateErr := ValidateTerminal(result.RunDir); validateErr != nil {
		t.Fatalf("failed terminal with partial evidence did not validate: %v", validateErr)
	}
}

func TestRunRejectsMalformedAttemptReceiptsWithCanonicalFailure(t *testing.T) {
	archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
	executor := executorFunc(func(ExecutionInput) (ExecutionOutput, error) {
		return ExecutionOutput{Artifacts: []OutputArtifact{{Kind: "markdown", MediaType: "text/markdown", Filename: "article.md", Content: []byte("body")}}}, nil
	})
	result, err := Run(context.Background(), Config{ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: testFileSHA256(t, protocolPath), RunID: "run-M1-A", Executor: executor})
	if !errors.Is(err, producterror.ErrInvalidInput) || result.Terminal.Status != "failed" || result.Terminal.Failure == nil || result.Terminal.Failure.Class != "executor_contract_failed" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if _, validateErr := ValidateTerminal(result.RunDir); validateErr != nil {
		t.Fatalf("canonical failed terminal did not validate: %v", validateErr)
	}
	if _, statErr := os.Stat(filepath.Join(result.RunDir, "article.md")); !os.IsNotExist(statErr) {
		t.Fatalf("malformed executor wrote an artifact: %v", statErr)
	}
}

func TestRunRejectsWhitespaceMetadataAndCaseFoldedReservedFilename(t *testing.T) {
	for _, test := range []struct {
		name     string
		artifact OutputArtifact
	}{
		{name: "whitespace kind", artifact: OutputArtifact{Kind: " ", MediaType: "text/plain", Filename: "article.md", Content: []byte("body")}},
		{name: "whitespace media type", artifact: OutputArtifact{Kind: "markdown", MediaType: " ", Filename: "article.md", Content: []byte("body")}},
		{name: "case-folded reserved filename", artifact: OutputArtifact{Kind: "invalid", MediaType: "application/json", Filename: "RUN.TERMINAL.JSON", Content: []byte("{}")}},
	} {
		t.Run(test.name, func(t *testing.T) {
			archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
			executor := executorFunc(func(ExecutionInput) (ExecutionOutput, error) {
				return ExecutionOutput{Attempts: []AttemptReceipt{{Attempt: 1, Kind: "semantic", Outcome: "completed"}}, Artifacts: []OutputArtifact{test.artifact}}, nil
			})
			result, err := Run(context.Background(), Config{ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: testFileSHA256(t, protocolPath), RunID: "run-M1-A", Executor: executor})
			if !errors.Is(err, producterror.ErrInvalidInput) || result.Terminal.Status != "failed" || result.Terminal.Failure == nil || result.Terminal.Failure.Class != "artifact_failed" {
				t.Fatalf("result=%#v err=%v", result, err)
			}
			if _, validateErr := ValidateTerminal(result.RunDir); validateErr != nil {
				t.Fatalf("artifact rejection terminal did not validate: %v", validateErr)
			}
		})
	}
}

func TestRunRejectsHarnessOwnedArtifactFilename(t *testing.T) {
	archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
	executor := executorFunc(func(ExecutionInput) (ExecutionOutput, error) {
		return ExecutionOutput{
			Attempts:  []AttemptReceipt{{Attempt: 1, Kind: "semantic", Outcome: "completed"}},
			Artifacts: []OutputArtifact{{Kind: "invalid", MediaType: "application/json", Filename: "run.terminal.json", Content: []byte("{}")}},
		}, nil
	})
	result, err := Run(context.Background(), Config{ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: testFileSHA256(t, protocolPath), RunID: "run-M1-A", Executor: executor})
	if !errors.Is(err, producterror.ErrInvalidInput) || result.Terminal.Status != "failed" || result.Terminal.Failure == nil || result.Terminal.Failure.Class != "artifact_failed" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if _, validateErr := ValidateTerminal(result.RunDir); validateErr != nil {
		t.Fatalf("reserved-name failure terminal did not validate: %v", validateErr)
	}
}

func TestRunRejectsInvalidArtifactBatchBeforeWriting(t *testing.T) {
	archive, repo, protocolPath := writeProtocolFixture(t, protocolOptions{})
	executor := executorFunc(func(ExecutionInput) (ExecutionOutput, error) {
		return ExecutionOutput{
			Attempts: []AttemptReceipt{{Attempt: 1, Kind: "semantic", Outcome: "completed"}},
			Artifacts: []OutputArtifact{
				{Kind: "markdown", MediaType: "text/markdown", Filename: "article.md", Content: []byte("body")},
				{Kind: "invalid", MediaType: "text/plain", Filename: "../escape", Content: []byte("bad")},
			},
		}, nil
	})
	result, err := Run(context.Background(), Config{ArchiveRoot: archive, RepositoryRoot: repo, ProtocolPath: protocolPath, ProtocolSHA256: testFileSHA256(t, protocolPath), RunID: "run-M1-A", Executor: executor})
	if !errors.Is(err, producterror.ErrInvalidInput) || result.Terminal.Status != "failed" || len(result.Terminal.PartialArtifacts) != 0 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if _, statErr := os.Stat(filepath.Join(result.RunDir, "article.md")); !os.IsNotExist(statErr) {
		t.Fatalf("invalid batch wrote an artifact: %v", statErr)
	}
	if _, validateErr := ValidateTerminal(result.RunDir); validateErr != nil {
		t.Fatalf("artifact failure terminal did not validate: %v", validateErr)
	}
}
