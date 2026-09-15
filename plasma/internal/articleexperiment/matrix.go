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

// RunMatrix executes the nine frozen cells in protocol order. Arm failures are
// preserved as terminal results and do not erase the remaining preflight cells.
func RunMatrix(ctx context.Context, config MatrixConfig) (MatrixResult, error) {
	if config.Executor == nil {
		return MatrixResult{}, fmt.Errorf("%w: executor is required", producterror.ErrInvalidInput)
	}
	archiveRoot, repositoryRoot, err := prepareArchiveRoot(config.ArchiveRoot, config.RepositoryRoot)
	if err != nil {
		return MatrixResult{}, err
	}
	loaded, err := loadProtocol(archiveRoot, repositoryRoot, config.ProtocolPath, config.ProtocolSHA256)
	if err != nil {
		return MatrixResult{}, err
	}
	if _, _, _, err := findRunCell(loaded, config.ExpectedFailedRun); err != nil {
		return MatrixResult{}, fmt.Errorf("%w: expected failed run is not preassigned", producterror.ErrInvalidInput)
	}
	manifestPath := filepath.Join(archiveRoot, "matrix-"+loaded.Protocol.ProtocolID+".terminal.json")
	if _, err := os.Lstat(manifestPath); err == nil {
		return MatrixResult{}, fmt.Errorf("%w: matrix terminal already exists", producterror.ErrConflict)
	} else if !os.IsNotExist(err) {
		return MatrixResult{}, err
	}
	if err := validateFreshRunSet(archiveRoot, loaded.Protocol.Runs); err != nil {
		return MatrixResult{}, err
	}
	result := MatrixResult{Runs: make([]Result, 0, len(loaded.Protocol.Runs))}
	receipts := make([]MatrixRunReceipt, 0, len(loaded.Protocol.Runs))
	for _, cell := range loaded.Protocol.Runs {
		run, runErr := runLoadedCell(ctx, archiveRoot, repositoryRoot, loaded, cell, config.Executor, config.StartedAt)
		if runErr != nil && (run.TerminalPath == "" || run.Terminal.Status != "failed" || errors.Is(runErr, context.Canceled) || errors.Is(runErr, context.DeadlineExceeded)) {
			return result, runErr
		}
		if run.TerminalPath == "" {
			return result, fmt.Errorf("%w: run ended without terminal receipt", producterror.ErrConflict)
		}
		terminal := run.Terminal
		terminalRaw, err := readRegularFile(run.TerminalPath, maxContractBytes)
		if err != nil {
			return result, err
		}
		result.Runs = append(result.Runs, run)
		receipts = append(receipts, MatrixRunReceipt{
			FixtureID: cell.FixtureID, ArmID: cell.ArmID, RunID: cell.RunID,
			Status: terminal.Status, TerminalPath: filepath.ToSlash(filepath.Join("runs", cell.RunID, "run.terminal.json")),
			TerminalSHA256: bytesSHA256(terminalRaw),
		})
		if terminal.Status == "completed" {
			result.Completed++
		} else {
			result.Failed++
		}
	}
	if len(result.Runs) != 9 || result.Completed+result.Failed != 9 {
		return result, fmt.Errorf("%w: matrix terminal count is incomplete", producterror.ErrConflict)
	}
	if err := ValidateForcedFailure(result, config.ExpectedFailedRun); err != nil {
		return result, err
	}
	result.Manifest = MatrixManifest{
		SchemaVersion: MatrixSchemaVersion, ProtocolID: loaded.Protocol.ProtocolID, ProtocolSHA256: loaded.SHA256,
		ExpectedFailedRun: config.ExpectedFailedRun, Runs: receipts, Completed: result.Completed, Failed: result.Failed,
		CompletedAt: time.Now().UTC().Format(terminalTimeFormat),
	}
	writtenPath, err := writeJSONAtomic(archiveRoot, ".matrix-"+loaded.Protocol.ProtocolID+".tmp", filepath.Base(manifestPath), result.Manifest)
	if err != nil {
		return result, err
	}
	result.ManifestPath = writtenPath
	manifestRaw, err := readRegularFile(writtenPath, maxContractBytes)
	if err != nil {
		return result, err
	}
	result.ManifestSHA256 = bytesSHA256(manifestRaw)
	validatedManifest, err := ValidateMatrix(archiveRoot, repositoryRoot, config.ProtocolPath, loaded.SHA256, result.ManifestSHA256)
	if err != nil {
		return result, err
	}
	result.Manifest = validatedManifest
	return result, nil
}

func validateFreshRunSet(archiveRoot string, cells []RunCell) error {
	runsRoot := filepath.Join(archiveRoot, "runs")
	if info, err := os.Lstat(runsRoot); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: runs root is invalid", producterror.ErrInvalidInput)
		}
		entries, err := os.ReadDir(runsRoot)
		if err != nil {
			return err
		}
		if len(entries) != 0 {
			return fmt.Errorf("%w: matrix archive already contains runs", producterror.ErrConflict)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	for _, cell := range cells {
		if !safeSlugPattern.MatchString(cell.RunID) || strings.Contains(cell.RunID, "..") {
			return fmt.Errorf("%w: matrix run ID is unsafe", producterror.ErrInvalidInput)
		}
	}
	return nil
}

// ValidateMatrix reloads the matrix receipt and every bound run terminal.
func ValidateMatrix(archiveRoot, repositoryRoot, protocolPath, protocolSHA, matrixSHA string) (MatrixManifest, error) {
	archiveRoot, err := canonicalExistingDirectory(archiveRoot)
	if err != nil {
		return MatrixManifest{}, err
	}
	archiveInfo, err := os.Stat(archiveRoot)
	if err != nil {
		return MatrixManifest{}, err
	}
	if archiveInfo.Mode().Perm()&0o077 != 0 {
		return MatrixManifest{}, fmt.Errorf("%w: archive root must not grant group or other permissions", producterror.ErrInvalidInput)
	}
	repositoryRoot, err = canonicalExistingDirectory(repositoryRoot)
	if err != nil {
		return MatrixManifest{}, err
	}
	if pathInside(repositoryRoot, archiveRoot) {
		return MatrixManifest{}, fmt.Errorf("%w: archive root resolved inside the repository", producterror.ErrInvalidInput)
	}
	loaded, err := loadProtocol(archiveRoot, repositoryRoot, protocolPath, protocolSHA)
	if err != nil {
		return MatrixManifest{}, err
	}
	manifestPath := filepath.Join(archiveRoot, "matrix-"+loaded.Protocol.ProtocolID+".terminal.json")
	manifest, manifestRaw, err := loadJSONFile[MatrixManifest](manifestPath)
	if err != nil {
		return MatrixManifest{}, err
	}
	if !validSHA256(matrixSHA) || bytesSHA256(manifestRaw) != matrixSHA {
		return MatrixManifest{}, fmt.Errorf("%w: matrix manifest SHA-256 mismatch", producterror.ErrConflict)
	}
	matrixCompletedAt, completedErr := time.Parse(terminalTimeFormat, manifest.CompletedAt)
	if manifest.SchemaVersion != MatrixSchemaVersion || manifest.ProtocolID != loaded.Protocol.ProtocolID || manifest.ProtocolSHA256 != loaded.SHA256 || !safeSlugPattern.MatchString(manifest.ExpectedFailedRun) || len(manifest.Runs) != 9 || manifest.Completed != 8 || manifest.Failed != 1 || completedErr != nil {
		return MatrixManifest{}, fmt.Errorf("%w: matrix terminal is invalid", producterror.ErrInvalidInput)
	}
	if _, _, _, err := findRunCell(loaded, manifest.ExpectedFailedRun); err != nil {
		return MatrixManifest{}, fmt.Errorf("%w: matrix expected failure is not preassigned", producterror.ErrInvalidInput)
	}
	seen := map[string]bool{}
	completed, failed := 0, 0
	latestRunCompletion := time.Time{}
	for index, receipt := range manifest.Runs {
		cell := loaded.Protocol.Runs[index]
		expectedTerminalPath := filepath.ToSlash(filepath.Join("runs", cell.RunID, "run.terminal.json"))
		if receipt.FixtureID != cell.FixtureID || receipt.ArmID != cell.ArmID || receipt.RunID != cell.RunID || receipt.TerminalPath != expectedTerminalPath || seen[receipt.RunID] || !validSHA256(receipt.TerminalSHA256) {
			return MatrixManifest{}, fmt.Errorf("%w: matrix run receipt is invalid", producterror.ErrInvalidInput)
		}
		seen[receipt.RunID] = true
		terminalPath, err := resolveReference(archiveRoot, repositoryRoot, archiveRoot, receipt.TerminalPath)
		if err != nil {
			return MatrixManifest{}, err
		}
		raw, err := readRegularFile(terminalPath, maxContractBytes)
		if err != nil || bytesSHA256(raw) != receipt.TerminalSHA256 {
			return MatrixManifest{}, fmt.Errorf("%w: matrix terminal receipt differs", producterror.ErrConflict)
		}
		terminal, err := ValidateTerminal(filepath.Dir(terminalPath))
		if err != nil {
			return MatrixManifest{}, err
		}
		runCompletedAt, runCompletedErr := time.Parse(terminalTimeFormat, terminal.CompletedAt)
		if runCompletedErr != nil {
			return MatrixManifest{}, fmt.Errorf("%w: run completion time is invalid", producterror.ErrInvalidInput)
		}
		if runCompletedAt.After(latestRunCompletion) {
			latestRunCompletion = runCompletedAt
		}
		fixture := loaded.Fixtures[cell.FixtureID]
		arm := loaded.Arms[cell.ArmID]
		if terminal.Status != receipt.Status || terminal.Pending.ProtocolID != loaded.Protocol.ProtocolID || terminal.Pending.ProtocolSHA256 != loaded.SHA256 || terminal.Pending.FixtureID != cell.FixtureID || terminal.Pending.FixtureSHA256 != fixture.SHA256 || terminal.Pending.ArmID != cell.ArmID || terminal.Pending.ArmSHA256 != arm.SHA256 || terminal.Pending.RunID != cell.RunID {
			return MatrixManifest{}, fmt.Errorf("%w: matrix run terminal differs", producterror.ErrConflict)
		}
		if receipt.Status == "completed" {
			if receipt.RunID == manifest.ExpectedFailedRun {
				return MatrixManifest{}, fmt.Errorf("%w: expected failed run completed", producterror.ErrConflict)
			}
			completed++
		} else if receipt.Status == "failed" {
			if receipt.RunID != manifest.ExpectedFailedRun || terminal.Failure == nil || terminal.Failure.Class != "executor_failed" {
				return MatrixManifest{}, fmt.Errorf("%w: retained failure differs", producterror.ErrConflict)
			}
			failed++
		} else {
			return MatrixManifest{}, fmt.Errorf("%w: matrix run status is invalid", producterror.ErrInvalidInput)
		}
	}
	if completed != manifest.Completed || failed != manifest.Failed || matrixCompletedAt.Before(latestRunCompletion) {
		return MatrixManifest{}, fmt.Errorf("%w: matrix counts or chronology differ", producterror.ErrConflict)
	}
	runsRoot := filepath.Join(archiveRoot, "runs")
	entries, err := os.ReadDir(runsRoot)
	if err != nil {
		return MatrixManifest{}, err
	}
	if len(entries) != len(seen) {
		return MatrixManifest{}, fmt.Errorf("%w: archive contains an unregistered run", producterror.ErrConflict)
	}
	for _, entry := range entries {
		if !entry.IsDir() || !seen[entry.Name()] {
			return MatrixManifest{}, fmt.Errorf("%w: archive contains an unregistered run", producterror.ErrConflict)
		}
	}
	return manifest, nil
}

// ValidateForcedFailure confirms that a synthetic preflight retained exactly
// the requested preassigned failed cell and no other failed cell.
func ValidateForcedFailure(result MatrixResult, runID string) error {
	if !safeSlugPattern.MatchString(runID) || len(result.Runs) != 9 || result.Completed != 8 || result.Failed != 1 {
		return fmt.Errorf("%w: forced failure result is invalid", producterror.ErrConflict)
	}
	failedRun := ""
	completed := 0
	seenRunIDs := map[string]bool{}
	for _, run := range result.Runs {
		currentRunID := run.Terminal.Pending.RunID
		if !safeSlugPattern.MatchString(currentRunID) || seenRunIDs[currentRunID] {
			return fmt.Errorf("%w: forced failure run identity is invalid", producterror.ErrConflict)
		}
		seenRunIDs[currentRunID] = true
		switch run.Terminal.Status {
		case "completed":
			completed++
		case "failed":
			if failedRun != "" || run.Terminal.Failure == nil || run.Terminal.Failure.Class != "executor_failed" {
				return fmt.Errorf("%w: forced failure class is invalid", producterror.ErrConflict)
			}
			failedRun = currentRunID
		default:
			return fmt.Errorf("%w: forced failure status is invalid", producterror.ErrConflict)
		}
	}
	if completed != result.Completed || failedRun != runID {
		return fmt.Errorf("%w: requested forced failure was not retained", producterror.ErrConflict)
	}
	return nil
}

func executionFixture(fixture Fixture) ExecutionFixture {
	return ExecutionFixture{
		FixtureID: fixture.FixtureID, ValueType: fixture.ValueType,
		Audience: fixture.Audience, ReaderPromise: fixture.ReaderPromise,
		Emphasis: fixture.Emphasis, Language: fixture.Language,
	}
}

func executionArm(arm Arm) ExecutionArm {
	return ExecutionArm{
		ArmID: arm.ArmID, Role: arm.Role,
		ContractSHA256: arm.ContractSHA256, ContextualOnly: arm.ContextualOnly,
	}
}

func cloneInputs(inputs []InputArtifact) []InputArtifact {
	cloned := make([]InputArtifact, len(inputs))
	for index, input := range inputs {
		cloned[index] = cloneInput(input)
	}
	return cloned
}

func cloneInput(input InputArtifact) InputArtifact {
	return InputArtifact{ID: input.ID, SHA256: input.SHA256, Content: append([]byte(nil), input.Content...)}
}
