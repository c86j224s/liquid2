package research

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
)

func TestDefinitionDigestsMatchPreExtractionContract(t *testing.T) {
	tests := []struct {
		name string
		defs []wire.ToolDefinition
		want string
	}{
		{
			name: "default research read definitions",
			defs: Definitions(false),
			want: "ae6414ff9aca4e9c4c1af36240b1bc65459704a0599e6e023f9e9e28499c6852",
		},
		{
			name: "legacy research read definitions",
			defs: Definitions(true),
			want: "2d7e8e7f023422bf606b2434ece089cc872ea361cfb13e6c622deb026451e90f",
		},
		{
			name: "legacy research mutation definitions",
			defs: LegacyMutationDefinitions(),
			want: "d2ab014353c40e3c00bd6531b0fbf0f802728597609ef7621c90b739b60a281f",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := definitionDigest(test.defs); got != test.want {
				t.Fatalf("definition digest = %s, want %s", got, test.want)
			}
		})
	}
}

func definitionDigest(defs []wire.ToolDefinition) string {
	hash := sha256.New()
	for _, def := range defs {
		hash.Write([]byte(def.Name))
		hash.Write([]byte{0})
		hash.Write([]byte(def.Description))
		hash.Write([]byte{0})
		hash.Write(def.InputSchema)
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}
