package articleexperiment

import (
	"fmt"
	"path/filepath"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

// FixtureBundle contains validated immutable fixture bytes needed by the
// minimal archive-local provider adapter.
type FixtureBundle struct {
	Fixture        RealFixture
	SourceBody     []byte
	SourceCatalog  []byte
	Dossier        []byte
	ClaimInventory []byte
	ClaimIDs       []string
}

// LoadFixtureBundle reuses the typed fixture validator and returns exact bytes
// without exposing mutable repository or product state.
func LoadFixtureBundle(archiveRoot, repositoryRoot, protocolPath, fixturePath, expectedSHA string) (FixtureBundle, error) {
	archiveRoot, repositoryRoot, err := prepareArchiveRoot(archiveRoot, repositoryRoot)
	if err != nil {
		return FixtureBundle{}, err
	}
	protocolPath, err = resolveInput(archiveRoot, repositoryRoot, archiveRoot, protocolPath)
	if err != nil {
		return FixtureBundle{}, err
	}
	fixturePath, err = resolveInput(archiveRoot, repositoryRoot, filepath.Dir(protocolPath), fixturePath)
	if err != nil {
		return FixtureBundle{}, err
	}
	fixture, raw, err := loadJSONFile[RealFixture](fixturePath)
	if err != nil {
		return FixtureBundle{}, err
	}
	if !validSHA256(expectedSHA) || bytesSHA256(raw) != expectedSHA {
		return FixtureBundle{}, fmt.Errorf("%w: real fixture SHA-256 mismatch", producterror.ErrConflict)
	}
	baseDir := filepath.Dir(fixturePath)
	if err := validateRealFixture(archiveRoot, repositoryRoot, baseDir, fixture); err != nil {
		return FixtureBundle{}, err
	}
	catalogPath, catalogRaw, err := loadRealReference(archiveRoot, repositoryRoot, baseDir, fixture.SourceCatalog)
	if err != nil {
		return FixtureBundle{}, err
	}
	catalog, _, err := decodeJSONBytes[FixtureSourceCatalog](filepath.Base(catalogPath), catalogRaw)
	if err != nil || len(catalog.Sources) != 1 {
		return FixtureBundle{}, fmt.Errorf("%w: fixture source catalog is invalid", producterror.ErrInvalidInput)
	}
	_, sourceBody, err := loadRealReference(archiveRoot, repositoryRoot, baseDir, catalog.Sources[0].Content)
	if err != nil {
		return FixtureBundle{}, err
	}
	_, dossier, err := loadRealReference(archiveRoot, repositoryRoot, baseDir, fixture.Dossier)
	if err != nil {
		return FixtureBundle{}, err
	}
	_, claims, err := loadRealReference(archiveRoot, repositoryRoot, baseDir, fixture.ClaimInventory)
	if err != nil {
		return FixtureBundle{}, err
	}
	inventory, _, err := decodeJSONBytes[FixtureClaimInventory]("claim-inventory.json", claims)
	if err != nil {
		return FixtureBundle{}, err
	}
	claimIDs := make([]string, 0, len(inventory.Claims))
	for _, claim := range inventory.Claims {
		claimIDs = append(claimIDs, claim.ClaimID)
	}
	return FixtureBundle{
		Fixture: fixture, SourceBody: append([]byte(nil), sourceBody...),
		SourceCatalog: append([]byte(nil), catalogRaw...), Dossier: append([]byte(nil), dossier...),
		ClaimInventory: append([]byte(nil), claims...), ClaimIDs: claimIDs,
	}, nil
}
