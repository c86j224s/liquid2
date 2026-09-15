package articleexperiment

import "testing"

func TestLoadFixtureBundleReturnsValidatedBytes(t *testing.T) {
	archive, repo, protocolPath, protocolSHA := writeRealProtocolFixture(t, func(*RealProtocol, map[string]*RealArm, map[string]*RealFixture, *BlindContract) {})
	loaded, err := LoadRealProtocol(archive, repo, protocolPath, protocolSHA)
	if err != nil {
		t.Fatal(err)
	}
	fixtureRef := loaded.Protocol.Fixtures[0]
	bundle, err := LoadFixtureBundle(archive, repo, protocolPath, fixtureRef.Path, fixtureRef.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Fixture.FixtureID != "M1" || len(bundle.SourceBody) == 0 || len(bundle.SourceCatalog) == 0 || len(bundle.Dossier) == 0 || len(bundle.ClaimInventory) == 0 {
		t.Fatalf("bundle=%#v", bundle)
	}
}
