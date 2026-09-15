package articleexperiment

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

type loadedProtocol struct {
	Protocol Protocol
	SHA256   string
	Fixtures map[string]loadedFixture
	Arms     map[string]loadedArm
	Cells    map[string]RunCell
}

type loadedFixture struct {
	Fixture Fixture
	SHA256  string
	Inputs  []InputArtifact
}

type loadedArm struct {
	Arm      Arm
	SHA256   string
	Contract InputArtifact
}

func loadProtocol(archiveRoot, repositoryRoot, protocolPath, expectedSHA string) (loadedProtocol, error) {
	protocolPath, err := resolveInput(archiveRoot, repositoryRoot, archiveRoot, protocolPath)
	if err != nil {
		return loadedProtocol{}, err
	}
	protocol, raw, err := loadJSONFile[Protocol](protocolPath)
	if err != nil {
		return loadedProtocol{}, err
	}
	protocolSHA := bytesSHA256(raw)
	if !validSHA256(expectedSHA) || protocolSHA != expectedSHA {
		return loadedProtocol{}, fmt.Errorf("%w: protocol SHA-256 mismatch", producterror.ErrConflict)
	}
	if _, err := time.Parse(time.RFC3339Nano, protocol.CreatedAt); protocol.SchemaVersion != ProtocolSchemaVersion || !safeSlugPattern.MatchString(protocol.ProtocolID) || err != nil || len(protocol.Fixtures) != 3 || len(protocol.Arms) != 3 || len(protocol.Runs) != 9 {
		return loadedProtocol{}, fmt.Errorf("%w: protocol envelope is invalid", producterror.ErrInvalidInput)
	}
	loaded := loadedProtocol{Protocol: protocol, SHA256: protocolSHA, Fixtures: map[string]loadedFixture{}, Arms: map[string]loadedArm{}, Cells: map[string]RunCell{}}
	baseDir := filepath.Dir(protocolPath)
	protocolInputBytes := 0
	for _, ref := range protocol.Fixtures {
		path, err := resolveReference(archiveRoot, repositoryRoot, baseDir, ref.Path)
		if err != nil {
			return loadedProtocol{}, err
		}
		fixture, fixtureRaw, err := loadJSONFile[Fixture](path)
		if err != nil {
			return loadedProtocol{}, err
		}
		sha := bytesSHA256(fixtureRaw)
		if fixture.SchemaVersion != FixtureSchemaVersion || fixture.FixtureID != ref.ID || sha != ref.SHA256 || !validFixture(fixture) || loaded.Fixtures[ref.ID].SHA256 != "" {
			return loadedProtocol{}, fmt.Errorf("%w: fixture contract is invalid", producterror.ErrInvalidInput)
		}
		if len(fixture.InputFiles) > maxFixtureInputCount {
			return loadedProtocol{}, fmt.Errorf("%w: fixture input count exceeds ceiling", producterror.ErrInvalidInput)
		}
		seenInputIDs := map[string]bool{}
		seenInputFiles := []os.FileInfo{}
		inputs := make([]InputArtifact, 0, len(fixture.InputFiles))
		totalInputBytes := 0
		for _, input := range fixture.InputFiles {
			if !safeSlugPattern.MatchString(input.ID) || !validSHA256(input.SHA256) || seenInputIDs[input.ID] {
				return loadedProtocol{}, fmt.Errorf("%w: fixture input receipt is invalid", producterror.ErrInvalidInput)
			}
			inputPath, err := resolveReference(archiveRoot, repositoryRoot, filepath.Dir(path), input.Path)
			if err != nil {
				return loadedProtocol{}, err
			}
			inputInfo, err := os.Stat(inputPath)
			if err != nil {
				return loadedProtocol{}, err
			}
			for _, seenInfo := range seenInputFiles {
				if os.SameFile(seenInfo, inputInfo) {
					return loadedProtocol{}, fmt.Errorf("%w: fixture input file is duplicated", producterror.ErrInvalidInput)
				}
			}
			inputRaw, err := readRegularFile(inputPath, maxInputBytes)
			if err != nil {
				return loadedProtocol{}, err
			}
			if bytesSHA256(inputRaw) != input.SHA256 {
				return loadedProtocol{}, fmt.Errorf("%w: fixture input SHA-256 mismatch", producterror.ErrConflict)
			}
			totalInputBytes += len(inputRaw)
			protocolInputBytes += len(inputRaw)
			if totalInputBytes > maxFixtureInputBytes {
				return loadedProtocol{}, fmt.Errorf("%w: fixture input bytes exceed ceiling", producterror.ErrInvalidInput)
			}
			if protocolInputBytes > maxProtocolInputBytes {
				return loadedProtocol{}, fmt.Errorf("%w: protocol input bytes exceed ceiling", producterror.ErrInvalidInput)
			}
			seenInputIDs[input.ID] = true
			seenInputFiles = append(seenInputFiles, inputInfo)
			inputs = append(inputs, InputArtifact{ID: input.ID, SHA256: input.SHA256, Content: append([]byte(nil), inputRaw...)})
		}
		loaded.Fixtures[ref.ID] = loadedFixture{Fixture: fixture, SHA256: sha, Inputs: inputs}
	}
	for _, ref := range protocol.Arms {
		path, err := resolveReference(archiveRoot, repositoryRoot, baseDir, ref.Path)
		if err != nil {
			return loadedProtocol{}, err
		}
		arm, armRaw, err := loadJSONFile[Arm](path)
		if err != nil {
			return loadedProtocol{}, err
		}
		sha := bytesSHA256(armRaw)
		if arm.SchemaVersion != ArmSchemaVersion || arm.ArmID != ref.ID || sha != ref.SHA256 || !validArm(arm) || loaded.Arms[ref.ID].SHA256 != "" {
			return loadedProtocol{}, fmt.Errorf("%w: arm contract is invalid", producterror.ErrInvalidInput)
		}
		contractPath, err := resolveReference(archiveRoot, repositoryRoot, filepath.Dir(path), arm.ContractPath)
		if err != nil {
			return loadedProtocol{}, err
		}
		contractRaw, err := readRegularFile(contractPath, maxContractBytes)
		if err != nil {
			return loadedProtocol{}, err
		}
		if bytesSHA256(contractRaw) != arm.ContractSHA256 {
			return loadedProtocol{}, fmt.Errorf("%w: arm executable contract SHA-256 mismatch", producterror.ErrConflict)
		}
		loaded.Arms[ref.ID] = loadedArm{Arm: arm, SHA256: sha, Contract: InputArtifact{ID: "arm_contract", SHA256: arm.ContractSHA256, Content: contractRaw}}
	}
	if err := validateMatrix(&loaded); err != nil {
		return loadedProtocol{}, err
	}
	return loaded, nil
}

func validFixture(fixture Fixture) bool {
	if fixture.Language != "ko" || strings.TrimSpace(fixture.Audience) == "" || strings.TrimSpace(fixture.ReaderPromise) == "" || len(fixture.InputFiles) == 0 {
		return false
	}
	switch fixture.FixtureID {
	case "M1":
		return fixture.ValueType == "method_how_to"
	case "M2":
		return fixture.ValueType == "mechanism_insight"
	case "M3":
		return fixture.ValueType == "discovery_context"
	default:
		return false
	}
}

func validArm(arm Arm) bool {
	if strings.TrimSpace(arm.ContractPath) == "" || !validSHA256(arm.ContractSHA256) {
		return false
	}
	switch arm.ArmID {
	case ArmReport:
		return arm.Role == "contextual_report_il" && arm.ContextualOnly
	case ArmControl:
		return arm.Role == "matched_control" && !arm.ContextualOnly
	case ArmArticle:
		return arm.Role == "article_narrative" && !arm.ContextualOnly
	default:
		return false
	}
}

func validateMatrix(loaded *loadedProtocol) error {
	for _, id := range requiredFixtureIDs {
		if loaded.Fixtures[id].SHA256 == "" {
			return fmt.Errorf("%w: protocol fixture matrix is incomplete", producterror.ErrInvalidInput)
		}
	}
	seenArmContracts := map[string]bool{}
	for _, id := range requiredArmIDs {
		arm := loaded.Arms[id]
		if arm.SHA256 == "" || seenArmContracts[arm.Contract.SHA256] {
			return fmt.Errorf("%w: protocol arm matrix is incomplete or duplicated", producterror.ErrInvalidInput)
		}
		seenArmContracts[arm.Contract.SHA256] = true
	}
	seenRuns := map[string]bool{}
	for _, cell := range loaded.Protocol.Runs {
		key := cell.FixtureID + "/" + cell.ArmID
		if loaded.Fixtures[cell.FixtureID].SHA256 == "" || loaded.Arms[cell.ArmID].SHA256 == "" || !safeSlugPattern.MatchString(cell.RunID) || strings.Contains(cell.RunID, "..") || seenRuns[cell.RunID] || loaded.Cells[key].RunID != "" {
			return fmt.Errorf("%w: protocol run matrix is invalid", producterror.ErrInvalidInput)
		}
		seenRuns[cell.RunID] = true
		loaded.Cells[key] = cell
	}
	for _, fixtureID := range requiredFixtureIDs {
		for _, armID := range requiredArmIDs {
			if loaded.Cells[fixtureID+"/"+armID].RunID == "" {
				return fmt.Errorf("%w: protocol run matrix is incomplete", producterror.ErrInvalidInput)
			}
		}
	}
	return nil
}
