package articleexperiment

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func validateAttempts(attempts []AttemptReceipt, failed bool) error {
	if len(attempts) < 1 || len(attempts) > 2 {
		return fmt.Errorf("%w: executor attempt receipt count is invalid", producterror.ErrInvalidInput)
	}
	for index, attempt := range attempts {
		if attempt.Attempt != index+1 || index == 0 && attempt.Kind != "semantic" || index == 1 && attempt.Kind != "technical_retry" {
			return fmt.Errorf("%w: executor attempt receipt is invalid", producterror.ErrInvalidInput)
		}
		wantOutcome := "completed"
		if len(attempts) == 2 && index == 0 || failed && index == len(attempts)-1 {
			wantOutcome = "failed"
		}
		if attempt.Outcome != wantOutcome {
			return fmt.Errorf("%w: executor attempt outcome differs", producterror.ErrInvalidInput)
		}
	}
	return nil
}

func writeArtifacts(runDir string, artifacts []OutputArtifact) ([]ArtifactReceipt, error) {
	if len(artifacts) == 0 || len(artifacts) > maxOutputArtifactCount {
		return nil, fmt.Errorf("%w: completed output artifact count is invalid", producterror.ErrInvalidInput)
	}
	seen := map[string]bool{}
	totalBytes := 0
	for _, artifact := range artifacts {
		filename := strings.TrimSpace(artifact.Filename)
		filenameKey := strings.ToLower(filename)
		if strings.TrimSpace(artifact.Kind) == "" || strings.TrimSpace(artifact.MediaType) == "" || filename == "" || filename != filepath.Base(filename) || reservedRunFilename(filename) || seen[filenameKey] || len(artifact.Content) == 0 || len(artifact.Content) > maxOutputBytes {
			return nil, fmt.Errorf("%w: output artifact is invalid", producterror.ErrInvalidInput)
		}
		totalBytes += len(artifact.Content)
		if totalBytes > maxOutputTotalBytes {
			return nil, fmt.Errorf("%w: output artifact bytes exceed ceiling", producterror.ErrInvalidInput)
		}
		seen[filenameKey] = true
	}
	receipts := make([]ArtifactReceipt, 0, len(artifacts))
	for _, artifact := range artifacts {
		filename := strings.TrimSpace(artifact.Filename)
		path := filepath.Join(runDir, filename)
		if err := writeExclusive(path, artifact.Content); err != nil {
			sortArtifactReceipts(receipts)
			return receipts, err
		}
		receipts = append(receipts, ArtifactReceipt{Kind: artifact.Kind, MediaType: artifact.MediaType, Filename: filename, SHA256: bytesSHA256(artifact.Content), ByteSize: len(artifact.Content)})
	}
	sortArtifactReceipts(receipts)
	return receipts, nil
}

// ValidateTerminal reloads a run only when the terminal manifest and every
// receipted artifact still match the immutable pending identity.
func ValidateTerminal(runDir string) (TerminalManifest, error) {
	runDir, err := canonicalExisting(runDir)
	if err != nil {
		return TerminalManifest{}, err
	}
	terminal, _, err := loadJSONFile[TerminalManifest](filepath.Join(runDir, "run.terminal.json"))
	if err != nil {
		return TerminalManifest{}, err
	}
	startedAt, startErr := time.Parse(terminalTimeFormat, terminal.Pending.StartedAt)
	completedAt, completedErr := time.Parse(terminalTimeFormat, terminal.CompletedAt)
	if terminal.SchemaVersion != TerminalSchemaVersion || !validTerminalStatus(terminal.Status) || startErr != nil || completedErr != nil || completedAt.Before(startedAt) || terminal.Pending.SchemaVersion != PendingSchemaVersion || !safeSlugPattern.MatchString(terminal.Pending.ProtocolID) || !safeSlugPattern.MatchString(terminal.Pending.FixtureID) || !safeSlugPattern.MatchString(terminal.Pending.ArmID) || terminal.Pending.RunID != filepath.Base(runDir) || !validSHA256(terminal.Pending.ProtocolSHA256) || !validSHA256(terminal.Pending.FixtureSHA256) || !validSHA256(terminal.Pending.ArmSHA256) {
		return TerminalManifest{}, fmt.Errorf("%w: terminal manifest is invalid", producterror.ErrInvalidInput)
	}
	pending, pendingRaw, err := loadJSONFile[PendingManifest](filepath.Join(runDir, "run.pending.json"))
	if err != nil || pending != terminal.Pending || !validSHA256(terminal.PendingSHA256) || bytesSHA256(pendingRaw) != terminal.PendingSHA256 {
		return TerminalManifest{}, fmt.Errorf("%w: terminal pending receipt differs", producterror.ErrConflict)
	}
	providerFailed := terminal.Status == "failed" && terminal.Failure != nil && (terminal.Failure.Class == "executor_failed" || terminal.Failure.Class == "executor_contract_failed")
	if err := validateAttempts(terminal.Attempts, providerFailed); err != nil {
		return TerminalManifest{}, err
	}
	if err := validateTerminalShape(terminal); err != nil {
		return TerminalManifest{}, err
	}
	seenArtifacts := map[string]string{}
	artifactCount := len(terminal.Artifacts) + len(terminal.PartialArtifacts)
	if artifactCount > maxOutputArtifactCount {
		return TerminalManifest{}, fmt.Errorf("%w: terminal artifact count exceeds ceiling", producterror.ErrInvalidInput)
	}
	totalArtifactBytes := 0
	for _, receipts := range [][]ArtifactReceipt{terminal.Artifacts, terminal.PartialArtifacts} {
		for _, receipt := range receipts {
			totalArtifactBytes += receipt.ByteSize
			if totalArtifactBytes > maxOutputTotalBytes {
				return TerminalManifest{}, fmt.Errorf("%w: terminal artifact bytes exceed ceiling", producterror.ErrInvalidInput)
			}
			if err := validateArtifactReceipt(runDir, receipt, seenArtifacts); err != nil {
				return TerminalManifest{}, err
			}
		}
	}
	if err := rejectUnreceiptedFiles(runDir, seenArtifacts); err != nil {
		return TerminalManifest{}, err
	}
	return terminal, nil
}

func reservedRunFilename(filename string) bool {
	switch strings.ToLower(filename) {
	case "run.pending.json", "run.terminal.json", ".run.terminal.json.tmp":
		return true
	default:
		return false
	}
}

func validTerminalStatus(status string) bool {
	return status == "completed" || status == "failed"
}

func validateTerminalShape(terminal TerminalManifest) error {
	if terminal.Status == "completed" && (terminal.Failure != nil || len(terminal.Artifacts) == 0 || len(terminal.PartialArtifacts) != 0) {
		return fmt.Errorf("%w: completed terminal shape is invalid", producterror.ErrInvalidInput)
	}
	if terminal.Status == "failed" && (terminal.Failure == nil || !validFailureReceipt(*terminal.Failure) || len(terminal.Artifacts) != 0) {
		return fmt.Errorf("%w: failed terminal shape is invalid", producterror.ErrInvalidInput)
	}
	return nil
}

func validFailureReceipt(receipt FailureReceipt) bool {
	switch receipt.Class {
	case "executor_failed", "executor_contract_failed":
		return receipt.Message == "synthetic arm executor failed" || receipt.Message == "real pilot executor failed"
	case "artifact_failed":
		return receipt.Message == "synthetic arm artifact storage failed" || receipt.Message == "real pilot artifact storage failed"
	default:
		return false
	}
}

func validateArtifactReceipt(runDir string, receipt ArtifactReceipt, seen map[string]string) error {
	filenameKey := strings.ToLower(receipt.Filename)
	if strings.TrimSpace(receipt.Kind) == "" || strings.TrimSpace(receipt.MediaType) == "" || receipt.Filename != filepath.Base(receipt.Filename) || reservedRunFilename(receipt.Filename) || !validSHA256(receipt.SHA256) || receipt.ByteSize <= 0 || seen[filenameKey] != "" {
		return fmt.Errorf("%w: terminal artifact receipt is invalid", producterror.ErrInvalidInput)
	}
	seen[filenameKey] = receipt.Filename
	raw, err := readRegularFile(filepath.Join(runDir, receipt.Filename), maxOutputBytes)
	if err != nil || len(raw) != receipt.ByteSize || bytesSHA256(raw) != receipt.SHA256 {
		return fmt.Errorf("%w: terminal artifact receipt differs", producterror.ErrConflict)
	}
	return nil
}

func rejectUnreceiptedFiles(runDir string, artifacts map[string]string) error {
	entries, err := os.ReadDir(runDir)
	if err != nil {
		return err
	}
	allowedExact := map[string]bool{"run.pending.json": true, "run.terminal.json": true}
	for _, filename := range artifacts {
		allowedExact[filename] = true
	}
	for _, entry := range entries {
		if allowedExact[entry.Name()] {
			continue
		}
		if entry.Name() == ".run.terminal.json.tmp" {
			temporary, temporaryErr := os.Lstat(filepath.Join(runDir, entry.Name()))
			final, finalErr := os.Lstat(filepath.Join(runDir, "run.terminal.json"))
			if temporaryErr == nil && finalErr == nil && temporary.Mode().IsRegular() && final.Mode().IsRegular() && os.SameFile(temporary, final) {
				continue
			}
		}
		return fmt.Errorf("%w: run contains an unreceipted file", producterror.ErrConflict)
	}
	return nil
}

func sortArtifactReceipts(receipts []ArtifactReceipt) {
	for i := 1; i < len(receipts); i++ {
		for j := i; j > 0 && receipts[j].Filename < receipts[j-1].Filename; j-- {
			receipts[j], receipts[j-1] = receipts[j-1], receipts[j]
		}
	}
}
