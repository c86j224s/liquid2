package articleexperiment

import (
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func TestLoadRealProtocolRejectsMalformedTypedFixtureArtifacts(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(*RealProtocol, map[string]*RealArm, map[string]*RealFixture, *BlindContract)
	}{
		{name: "claim count mismatch", edit: func(_ *RealProtocol, _ map[string]*RealArm, fixtures map[string]*RealFixture, _ *BlindContract) {
			fixtures["M1"].MaterialTruthClaims++
		}},
		{name: "connection count mismatch", edit: func(_ *RealProtocol, _ map[string]*RealArm, fixtures map[string]*RealFixture, _ *BlindContract) {
			fixtures["M2"].SupportedConnections++
		}},
		{name: "caveat count mismatch", edit: func(_ *RealProtocol, _ map[string]*RealArm, fixtures map[string]*RealFixture, _ *BlindContract) {
			fixtures["M3"].MaterialCaveats++
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			archive, repo, protocolPath, protocolSHA := writeRealProtocolFixture(t, test.edit)
			if _, err := LoadRealProtocol(archive, repo, protocolPath, protocolSHA); !errors.Is(err, producterror.ErrInvalidInput) {
				t.Fatalf("error = %v, want invalid input", err)
			}
		})
	}
}
