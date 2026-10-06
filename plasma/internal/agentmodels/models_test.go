package agentmodels

import "testing"

func TestResolveUsesLunaXHighDefault(t *testing.T) {
	metadata := Default()
	if metadata.Name != "gpt-5.6-luna" || metadata.Label != "GPT-5.6 Luna" {
		t.Fatalf("unexpected default metadata: %#v", metadata)
	}
	model, effort, err := Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	if model != "gpt-5.6-luna" || effort != "xhigh" {
		t.Fatalf("got %q / %q", model, effort)
	}
}

func TestCatalogPublishesModelDefaultReasoningEffort(t *testing.T) {
	for _, model := range Catalog() {
		want := "medium"
		if model.Name == "gpt-5.6-luna" {
			want = "xhigh"
		}
		if model.DefaultReasoningEffort != want {
			t.Fatalf("model %q default = %q", model.Name, model.DefaultReasoningEffort)
		}
	}
}

func TestResolveKeepsExplicitGPT55DefaultAtMedium(t *testing.T) {
	model, effort, err := Resolve("gpt-5.5", "")
	if err != nil {
		t.Fatal(err)
	}
	if model != "gpt-5.5" || effort != "medium" {
		t.Fatalf("got %q / %q", model, effort)
	}
}

func TestResolveKeepsUnknownModelFallbackAtMedium(t *testing.T) {
	model, effort, err := Resolve("future-codex", "")
	if err != nil {
		t.Fatal(err)
	}
	if model != "future-codex" || effort != "medium" {
		t.Fatalf("got %q / %q", model, effort)
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

func TestCatalogPublishesGPT6Models(t *testing.T) {
	want := map[string]struct {
		label          string
		ultraSupported bool
	}{
		"gpt-6-astra": {"GPT-6 Astra", true},
		"gpt-6-sol":   {"GPT-6 Sol", true},
		"gpt-6-luna":  {"GPT-6 Luna", false},
	}
	found := map[string]bool{}
	for _, model := range Catalog() {
		expect, ok := want[model.Name]
		if !ok {
			continue
		}
		found[model.Name] = true
		if model.Label != expect.label {
			t.Fatalf("model %q label = %q, want %q", model.Name, model.Label, expect.label)
		}
		if contains(model.ReasoningEfforts, "ultra") != expect.ultraSupported {
			t.Fatalf("model %q ultra support = %v, want %v", model.Name, contains(model.ReasoningEfforts, "ultra"), expect.ultraSupported)
		}
	}
	for name := range want {
		if !found[name] {
			t.Fatalf("catalog missing model %q", name)
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
