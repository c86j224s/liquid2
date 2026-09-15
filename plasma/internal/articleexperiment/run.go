package articleexperiment

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

const terminalTimeFormat = "2006-01-02T15:04:05.000000000Z07:00"

// Run executes exactly one preassigned synthetic pilot cell and writes one
// terminal manifest after every output receipt is durable.
func Run(ctx context.Context, config Config) (Result, error) {
	if config.Executor == nil {
		return Result{}, fmt.Errorf("%w: executor is required", producterror.ErrInvalidInput)
	}
	archiveRoot, repositoryRoot, err := prepareArchiveRoot(config.ArchiveRoot, config.RepositoryRoot)
	if err != nil {
		return Result{}, err
	}
	loaded, err := loadProtocol(archiveRoot, repositoryRoot, config.ProtocolPath, config.ProtocolSHA256)
	if err != nil {
		return Result{}, err
	}
	cell, _, _, err := findRunCell(loaded, strings.TrimSpace(config.RunID))
	if err != nil {
		return Result{}, err
	}
	return runLoadedCell(ctx, archiveRoot, repositoryRoot, loaded, cell, config.Executor, config.StartedAt)
}

func runLoadedCell(ctx context.Context, archiveRoot, repositoryRoot string, loaded loadedProtocol, cell RunCell, executor Executor, startedAt time.Time) (Result, error) {
	fixture := loaded.Fixtures[cell.FixtureID]
	arm := loaded.Arms[cell.ArmID]
	now := time.Now().UTC()
	started := startedAt.UTC()
	if startedAt.IsZero() {
		started = now
	} else if started.After(now) {
		return Result{}, fmt.Errorf("%w: started time is in the future", producterror.ErrInvalidInput)
	}
	runDir, err := prepareRunDir(archiveRoot, repositoryRoot, cell.RunID)
	if err != nil {
		return Result{}, err
	}
	runInfo, err := os.Stat(runDir)
	if err != nil {
		return Result{RunDir: runDir}, err
	}
	pending := PendingManifest{
		SchemaVersion: PendingSchemaVersion, ProtocolID: loaded.Protocol.ProtocolID, ProtocolSHA256: loaded.SHA256,
		FixtureID: cell.FixtureID, FixtureSHA256: fixture.SHA256, ArmID: cell.ArmID, ArmSHA256: arm.SHA256,
		RunID: cell.RunID, StartedAt: started.Format(terminalTimeFormat),
	}
	pendingPath := filepath.Join(runDir, "run.pending.json")
	if err := writeJSONExclusive(pendingPath, pending); err != nil {
		return Result{RunDir: runDir}, err
	}
	pendingRaw, err := readRegularFile(pendingPath, maxContractBytes)
	if err != nil {
		return Result{RunDir: runDir, PendingPath: pendingPath}, err
	}

	output, executeErr := executor.Execute(ctx, ExecutionInput{
		ProtocolID: loaded.Protocol.ProtocolID, ProtocolSHA256: loaded.SHA256,
		Fixture: executionFixture(fixture.Fixture), Inputs: cloneInputs(fixture.Inputs),
		Arm: executionArm(arm.Arm), ArmContract: cloneInput(arm.Contract), Cell: cell,
	})
	if err := validateRunDirectory(archiveRoot, repositoryRoot, runDir, runInfo); err != nil {
		return Result{RunDir: runDir, PendingPath: pendingPath}, err
	}
	failureClass := ""
	if receiptErr := validateAttempts(output.Attempts, executeErr != nil); receiptErr != nil {
		executeErr = receiptErr
		failureClass = "executor_contract_failed"
		output.Attempts = []AttemptReceipt{{Attempt: 1, Kind: "semantic", Outcome: "failed"}}
	} else if executeErr != nil {
		failureClass = "executor_failed"
	}
	attempts := append([]AttemptReceipt(nil), output.Attempts...)
	terminal := TerminalManifest{
		SchemaVersion: TerminalSchemaVersion, Pending: pending, PendingSHA256: bytesSHA256(pendingRaw), Attempts: attempts,
	}
	if executeErr != nil {
		terminal.Status = "failed"
		terminal.Failure = &FailureReceipt{Class: failureClass, Message: "synthetic arm executor failed"}
		if failureClass == "executor_failed" && len(output.Artifacts) != 0 {
			receipts, writeErr := writeArtifacts(runDir, output.Artifacts)
			if writeErr != nil {
				terminal.Failure = &FailureReceipt{Class: "artifact_failed", Message: "synthetic arm artifact storage failed"}
				executeErr = writeErr
			} else {
				terminal.PartialArtifacts = receipts
			}
		}
	} else {
		terminal.Status = "completed"
		receipts, writeErr := writeArtifacts(runDir, output.Artifacts)
		if writeErr != nil {
			terminal.Status = "failed"
			terminal.PartialArtifacts = receipts
			terminal.Failure = &FailureReceipt{Class: "artifact_failed", Message: "synthetic arm artifact storage failed"}
			executeErr = writeErr
		} else {
			terminal.Artifacts = receipts
		}
	}
	terminal.CompletedAt = time.Now().UTC().Format(terminalTimeFormat)
	terminalPath, terminalErr := writeTerminalAtomic(runDir, terminal)
	result := Result{RunDir: runDir, PendingPath: pendingPath, TerminalPath: terminalPath, Terminal: terminal}
	if terminalErr != nil {
		return result, terminalErr
	}
	validatedTerminal, validateErr := ValidateTerminal(runDir)
	if validateErr != nil {
		return result, validateErr
	}
	result.Terminal = validatedTerminal
	if executeErr != nil {
		return result, safeRunFailure(terminal, executeErr)
	}
	return result, nil
}

func validateRunDirectory(archiveRoot, repositoryRoot, runDir string, before os.FileInfo) error {
	resolved, err := canonicalExisting(runDir)
	if err != nil {
		return err
	}
	after, err := os.Stat(resolved)
	if err != nil {
		return err
	}
	if !after.IsDir() || !os.SameFile(before, after) || !pathInside(archiveRoot, resolved) || pathInside(repositoryRoot, resolved) {
		return fmt.Errorf("%w: run directory changed during execution", producterror.ErrConflict)
	}
	return nil
}

func safeRunFailure(terminal TerminalManifest, cause error) error {
	if terminal.Failure == nil {
		return fmt.Errorf("%w: synthetic run failed", producterror.ErrConflict)
	}
	switch terminal.Failure.Class {
	case "executor_contract_failed":
		return fmt.Errorf("%w: synthetic executor receipt rejected", producterror.ErrInvalidInput)
	case "artifact_failed":
		return fmt.Errorf("%w: synthetic artifact output rejected", producterror.ErrInvalidInput)
	default:
		if errors.Is(cause, context.Canceled) {
			return context.Canceled
		}
		if errors.Is(cause, context.DeadlineExceeded) {
			return context.DeadlineExceeded
		}
		return fmt.Errorf("%w: synthetic executor failed", producterror.ErrConflict)
	}
}

func findRunCell(loaded loadedProtocol, runID string) (RunCell, loadedFixture, loadedArm, error) {
	for _, cell := range loaded.Protocol.Runs {
		if cell.RunID == runID {
			return cell, loaded.Fixtures[cell.FixtureID], loaded.Arms[cell.ArmID], nil
		}
	}
	return RunCell{}, loadedFixture{}, loadedArm{}, fmt.Errorf("%w: run ID is not preassigned by protocol", producterror.ErrInvalidInput)
}
