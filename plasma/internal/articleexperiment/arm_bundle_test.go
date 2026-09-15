package articleexperiment

import "testing"

func TestLoadArmBundleReturnsExactContractBytes(t *testing.T) {
	archive, repo, protocolPath, protocolSHA := writeRealProtocolFixture(t, func(*RealProtocol, map[string]*RealArm, map[string]*RealFixture, *BlindContract) {})
	loaded, err := LoadRealProtocol(archive, repo, protocolPath, protocolSHA)
	if err != nil {
		t.Fatal(err)
	}
	armRef := loaded.Protocol.Arms[2]
	bundle, err := LoadArmBundle(archive, repo, protocolPath, armRef.Path, armRef.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Arm.ArmID != ArmArticle || len(bundle.AuthorPrompt) == 0 || len(bundle.ReaderPrompt) == 0 || len(bundle.AuditorPrompt) == 0 || len(bundle.RepairPrompt) == 0 || len(bundle.Treatment) == 0 {
		t.Fatalf("bundle=%#v", bundle)
	}
}
