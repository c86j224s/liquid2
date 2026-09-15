package articleexperiment

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type executorFunc func(ExecutionInput) (ExecutionOutput, error)

func (run executorFunc) Execute(_ context.Context, input ExecutionInput) (ExecutionOutput, error) {
	return run(input)
}

type fakeExecutor struct {
	calls  []ExecutionInput
	failAt string
}

func (executor *fakeExecutor) Execute(_ context.Context, input ExecutionInput) (ExecutionOutput, error) {
	executor.calls = append(executor.calls, input)
	attempts := []AttemptReceipt{{Attempt: 1, Kind: "semantic", Outcome: "completed"}}
	if input.Cell.RunID == executor.failAt {
		attempts[0].Outcome = "failed"
		return ExecutionOutput{Attempts: attempts}, errors.New("private provider detail")
	}
	body := []byte(input.Cell.FixtureID + "/" + input.Cell.ArmID + "\n")
	return ExecutionOutput{Attempts: attempts, Artifacts: []OutputArtifact{{Kind: "markdown", MediaType: "text/markdown; charset=utf-8", Filename: "article.md", Content: body}}}, nil
}

type protocolOptions struct {
	duplicateRunID bool
	symlinkFixture bool
}

func writeProtocolFixture(t *testing.T, options protocolOptions) (string, string, string) {
	t.Helper()
	archive := t.TempDir()
	if err := os.Chmod(archive, 0o700); err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	fixturesDir := filepath.Join(archive, "fixtures")
	armsDir := filepath.Join(archive, "arms")
	if err := os.MkdirAll(fixturesDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(armsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	fixtureRefs := make([]FileRef, 0, 3)
	for _, fixtureID := range requiredFixtureIDs {
		inputName := fixtureID + "-input.txt"
		input := []byte("original fixture " + fixtureID + "\n")
		if err := os.WriteFile(filepath.Join(fixturesDir, inputName), input, 0o600); err != nil {
			t.Fatal(err)
		}
		fixture := Fixture{SchemaVersion: FixtureSchemaVersion, FixtureID: fixtureID, ValueType: map[string]string{"M1": "method_how_to", "M2": "mechanism_insight", "M3": "discovery_context"}[fixtureID], Audience: "처음 읽는 독자", ReaderPromise: "핵심을 이해한다", Language: "ko", InputFiles: []FileRef{{ID: "source", Path: inputName, SHA256: bytesSHA256(input)}}}
		path := filepath.Join(fixturesDir, fixtureID+".json")
		writeTestJSON(t, path, fixture)
		if options.symlinkFixture && fixtureID == "M1" {
			realPath := filepath.Join(fixturesDir, "M1-real.json")
			if err := os.Rename(path, realPath); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(realPath, path); err != nil {
				t.Fatal(err)
			}
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		fixtureRefs = append(fixtureRefs, FileRef{ID: fixtureID, Path: filepath.ToSlash(filepath.Join("fixtures", fixtureID+".json")), SHA256: bytesSHA256(raw)})
	}
	armRefs := make([]FileRef, 0, 3)
	for _, armID := range requiredArmIDs {
		contract := []byte("contract/" + armID + "\n")
		if err := os.WriteFile(filepath.Join(armsDir, armID+".txt"), contract, 0o600); err != nil {
			t.Fatal(err)
		}
		arm := Arm{SchemaVersion: ArmSchemaVersion, ArmID: armID, Role: map[string]string{"R": "contextual_report_il", "E": "matched_control", "A": "article_narrative"}[armID], ContractPath: armID + ".txt", ContractSHA256: bytesSHA256(contract), ContextualOnly: armID == "R"}
		path := filepath.Join(armsDir, armID+".json")
		writeTestJSON(t, path, arm)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		armRefs = append(armRefs, FileRef{ID: armID, Path: filepath.ToSlash(filepath.Join("arms", armID+".json")), SHA256: bytesSHA256(raw)})
	}
	cells := make([]RunCell, 0, 9)
	for _, fixtureID := range requiredFixtureIDs {
		for _, armID := range requiredArmIDs {
			cells = append(cells, RunCell{FixtureID: fixtureID, ArmID: armID, RunID: "run-" + fixtureID + "-" + armID})
		}
	}
	if options.duplicateRunID {
		cells[1].RunID = cells[0].RunID
	}
	protocol := Protocol{SchemaVersion: ProtocolSchemaVersion, ProtocolID: "pilot-v1", CreatedAt: "2026-09-02T00:00:00Z", Fixtures: fixtureRefs, Arms: armRefs, Runs: cells}
	protocolPath := filepath.Join(archive, "protocol.json")
	writeTestJSON(t, protocolPath, protocol)
	return archive, repo, protocolPath
}

func writeTestJSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func matchedInputIdentity(input ExecutionInput) string {
	value := struct {
		Fixture ExecutionFixture
		Inputs  []InputArtifact
	}{Fixture: input.Fixture, Inputs: input.Inputs}
	raw, _ := json.Marshal(value)
	return bytesSHA256(raw)
}

func testFileSHA256(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return bytesSHA256(raw)
}
