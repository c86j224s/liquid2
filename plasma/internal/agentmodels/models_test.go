package agentmodels

import "testing"

func TestResolveUsesTerraMediumDefault(t *testing.T) {
	metadata := Default()
	if metadata.Name != "gpt-5.6-terra" || metadata.Label != "GPT-5.6 Terra" {
		t.Fatalf("unexpected default metadata: %#v", metadata)
	}
	model, effort, err := Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	if model != "gpt-5.6-terra" || effort != "medium" {
		t.Fatalf("got %q / %q", model, effort)
	}
}

func TestCatalogPublishesModelDefaultReasoningEffort(t *testing.T) {
	for _, model := range Catalog() {
		if model.DefaultReasoningEffort != "medium" {
			t.Fatalf("model %q default = %q", model.Name, model.DefaultReasoningEffort)
		}
	}
}

func TestResolveValidatesModelCapabilities(t *testing.T) {
	for _, test := range []struct {
		model  string
		effort string
		valid  bool
	}{
		{"gpt-5.6-sol", "ultra", true},
		{"gpt-5.6-terra", "ultra", true},
		{"gpt-5.6-luna", "ultra", false},
		{"future-codex", "xhigh", true},
		{"future-codex", "ultra", false},
	} {
		_, _, err := Resolve(test.model, test.effort)
		if (err == nil) != test.valid {
			t.Errorf("Resolve(%q, %q) error = %v, valid = %v", test.model, test.effort, err, test.valid)
		}
	}
}

func TestResolveForSessionPreservesEmptyLegacyResume(t *testing.T) {
	model, effort, err := ResolveForSession("", "", "legacy-session")
	if err != nil {
		t.Fatal(err)
	}
	if model != "" || effort != "" {
		t.Fatalf("got %q / %q", model, effort)
	}
}
