package articleexperiment

import (
	"fmt"
	"path/filepath"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

// ArmBundle contains one validated real-pilot arm and the exact prompt and
// treatment bytes consumed by the minimal runner.
type ArmBundle struct {
	Arm           RealArm
	AuthorPrompt  []byte
	ReaderPrompt  []byte
	AuditorPrompt []byte
	RepairPrompt  []byte
	Treatment     []byte
	ToolPolicy    []byte
	OutputSchema  []byte
	Renderer      []byte
}

func LoadArmBundle(archiveRoot, repositoryRoot, protocolPath, armPath, expectedSHA string) (ArmBundle, error) {
	archiveRoot, repositoryRoot, err := prepareArchiveRoot(archiveRoot, repositoryRoot)
	if err != nil {
		return ArmBundle{}, err
	}
	protocolPath, err = resolveInput(archiveRoot, repositoryRoot, archiveRoot, protocolPath)
	if err != nil {
		return ArmBundle{}, err
	}
	armPath, err = resolveInput(archiveRoot, repositoryRoot, filepath.Dir(protocolPath), armPath)
	if err != nil {
		return ArmBundle{}, err
	}
	arm, raw, err := loadJSONFile[RealArm](armPath)
	if err != nil {
		return ArmBundle{}, err
	}
	if !validSHA256(expectedSHA) || bytesSHA256(raw) != expectedSHA {
		return ArmBundle{}, fmt.Errorf("%w: real arm SHA-256 mismatch", producterror.ErrConflict)
	}
	baseDir := filepath.Dir(armPath)
	if err := validateRealArm(archiveRoot, repositoryRoot, baseDir, arm); err != nil {
		return ArmBundle{}, err
	}
	load := func(ref FileRef) ([]byte, error) {
		_, content, err := loadRealReference(archiveRoot, repositoryRoot, baseDir, ref)
		return content, err
	}
	author, err := load(arm.Common.AuthorPrompt)
	if err != nil {
		return ArmBundle{}, err
	}
	reader, err := load(arm.Common.ReaderPrompt)
	if err != nil {
		return ArmBundle{}, err
	}
	auditor, err := load(arm.Common.AuditorPrompt)
	if err != nil {
		return ArmBundle{}, err
	}
	repair, err := load(arm.Common.RepairPrompt)
	if err != nil {
		return ArmBundle{}, err
	}
	treatment, err := load(arm.Treatment.Contract)
	if err != nil {
		return ArmBundle{}, err
	}
	toolPolicy, err := load(arm.Common.ToolPolicy)
	if err != nil {
		return ArmBundle{}, err
	}
	outputSchema, err := load(arm.Common.OutputSchema)
	if err != nil {
		return ArmBundle{}, err
	}
	renderer, err := load(arm.Common.Renderer)
	if err != nil {
		return ArmBundle{}, err
	}
	return ArmBundle{Arm: arm, AuthorPrompt: author, ReaderPrompt: reader, AuditorPrompt: auditor, RepairPrompt: repair, Treatment: treatment, ToolPolicy: toolPolicy, OutputSchema: outputSchema, Renderer: renderer}, nil
}
