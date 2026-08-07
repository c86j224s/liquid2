package research

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
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
			want: "047e9536c469c1f7793b0b0ef48fb6b22816e79172aa660cbfab01dcf05773eb",
		},
		{
			name: "legacy research read definitions",
			defs: Definitions(true),
			want: "a561235b7d0f3e2d5c0eab278dfefff9eb14014c3666e6f999ae4a6e9e36080f",
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

func TestDefinitionsExposeExactLiteralGrepDescription(t *testing.T) {
	tests := []struct {
		name   string
		legacy bool
		want   string
	}{
		{
			name:   "default",
			legacy: false,
			want:   "Find candidate snippets using case-insensitive literal substring search across non-report mission research discovery objects. The entire query must occur contiguously; use one short exact word or phrase per call and split separate concepts into separate searches. Matches are candidates, not evidence or sources until read and referenced.",
		},
		{
			name:   "legacy",
			legacy: true,
			want:   "Find candidate snippets using case-insensitive literal substring search across mission ledger objects. The entire query must occur contiguously; use one short exact word or phrase per call and split separate concepts into separate searches. Matches are candidates, not evidence or sources until read and referenced.",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			legacy := test.legacy
			want := test.want
			found := false
			for _, def := range Definitions(legacy) {
				if def.Name == mcptools.ToolResearchGrep {
					found = true
					if def.Description != want {
						t.Fatalf("grep description = %q, want %q", def.Description, want)
					}
					break
				}
			}
			if !found {
				t.Fatalf("missing grep definition for legacy=%v", legacy)
			}
		})
	}
}

func TestDefinitionsExposeReportBoundaryReferencesDescription(t *testing.T) {
	const want = "Follow forward and backward references between pinned sources, non-report raw artifacts, and non-report ledger events."
	found := false
	for _, def := range Definitions(false) {
		if def.Name == mcptools.ToolResearchRefs {
			found = true
			if def.Description != want {
				t.Fatalf("references description = %q, want %q", def.Description, want)
			}
			break
		}
	}
	if !found {
		t.Fatalf("missing references definition")
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
