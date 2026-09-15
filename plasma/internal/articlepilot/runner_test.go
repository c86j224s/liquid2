package articlepilot

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/articleexperiment"
)

type fakeProvider struct {
	requests []agentexec.AgentRequest
	results  []agentexec.AgentResult
}

func (provider *fakeProvider) Run(_ context.Context, request agentexec.AgentRequest) (agentexec.AgentResult, error) {
	provider.requests = append(provider.requests, request)
	return provider.results[len(provider.requests)-1], nil
}

func TestRunUsesToolsDisabledMatchedFlowWithoutRepair(t *testing.T) {
	author := testDocumentJSON(t, strings.Repeat("가", 7000))
	provider := &fakeProvider{results: []agentexec.AgentResult{
		providerResult(author, "author", false),
		providerResult(`{"needs_repair":false,"findings":[]}`, "reader", false),
		providerResult(`{"needs_repair":false,"findings":[]}`, "auditor", false),
	}}
	result, err := Run(context.Background(), testConfig(provider, articleexperiment.ArmArticle))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Document.Sections) != 2 || result.Audit.RepairApplied || !result.Audit.ConfirmationPassed || len(result.Usage.Calls) != 3 {
		t.Fatalf("result=%#v", result)
	}
	for _, request := range provider.requests {
		if !request.DisableTools || !request.IgnoreUserConfig || !request.ReplaceMCPTools || request.Model != "gpt-5.6-luna" || request.ReasoningEffort != "xhigh" {
			t.Fatalf("request=%#v", request)
		}
	}
}

func TestRunRepairsOnlyAfterDiagnosisUsingExactAuthorLineage(t *testing.T) {
	oldText := "고유한수정대상문장"
	body := oldText + strings.Repeat("가", 7000-len([]rune(oldText)))
	author := testDocumentJSON(t, body)
	newText := strings.Repeat("나", len([]rune(oldText)))
	repairJSON, _ := json.Marshal(repairBatch{Replacements: []replacement{{NodeID: "node_1", OldText: oldText, NewText: newText}}})
	provider := &fakeProvider{results: []agentexec.AgentResult{
		providerResult(author, "author", false),
		providerResult(`{"needs_repair":true,"findings":["opening"]}`, "reader", false),
		providerResult(`{"needs_repair":false,"findings":[]}`, "auditor", false),
		providerResult(string(repairJSON), "author", true),
		providerResult(`{"needs_repair":false,"findings":[]}`, "reader-confirmation", false),
		providerResult(`{"needs_repair":false,"findings":[]}`, "factual-confirmation", false),
	}}
	result, err := Run(context.Background(), testConfig(provider, articleexperiment.ArmControl))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Audit.RepairApplied || !strings.Contains(string(result.Manuscript), newText) || provider.requests[3].PreviousSessionID != "author" {
		t.Fatalf("result=%#v repair request=%#v", result, provider.requests[3])
	}
	if !strings.Contains(provider.requests[0].Prompt, "treatment-E") {
		t.Fatalf("control prompt missed frozen treatment: %q", provider.requests[0].Prompt)
	}
}

func testConfig(provider Provider, armID string) Config {
	clock := time.Unix(0, 0)
	return Config{
		ProtocolSHA256: strings.Repeat("a", 64),
		Bundle:         articleexperiment.FixtureBundle{Fixture: articleexperiment.RealFixture{FixtureID: "M1", Audience: "독자", ReaderPromise: "이해", Emphasis: "핵심", MinimumBodyCharacters: 7000, MaximumBodyCharacters: 11000, MaterialTruthClaims: 1}, SourceBody: []byte("source"), SourceCatalog: []byte(`{"catalog":true}`), Dossier: []byte(`{"dossier":true}`), ClaimInventory: []byte(`{"claims":[]}`), ClaimIDs: []string{"m1-c01"}},
		Arm:            articleexperiment.RealArm{ArmID: armID, Provider: articleexperiment.ProviderContract{Model: "gpt-5.6-luna", ReasoningEffort: "xhigh"}, Budget: articleexperiment.CellBudget{MaxProviderCalls: 6, MaxDurationMilliseconds: 1000, MaxSourceBytesPerCall: 64 << 10, MaxSourceBytesPerAttempt: 512 << 10, MaxInputTokens: 100000, MaxOutputTokens: 100000, MaxTotalTokens: 200000}},
		AuthorPrompt:   "author", ReaderPrompt: "reader", AuditorPrompt: "auditor", RepairPrompt: "repair", Treatment: []byte("treatment-" + armID), Provider: provider,
		Now: func() time.Time { clock = clock.Add(time.Millisecond); return clock },
	}
}

func testDocumentJSON(t *testing.T, body string) string {
	t.Helper()
	document := Document{SchemaVersion: DocumentSchemaVersion, Language: "ko", Title: Leaf{NodeID: "title", Text: "제목", NoFactualClaims: true}, Sections: []Section{{SectionID: "section_1", Heading: Leaf{NodeID: "heading_1", Text: "첫째", NoFactualClaims: true}, Nodes: []Leaf{{NodeID: "node_1", Text: body, ClaimIDs: []string{"m1-c01"}}}}, {SectionID: "section_2", Heading: Leaf{NodeID: "heading_2", Text: "둘째", NoFactualClaims: true}, Nodes: []Leaf{{NodeID: "node_2", Text: "마무리", NoFactualClaims: true}}}}}
	raw, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func providerResult(text, session string, resumed bool) agentexec.AgentResult {
	usage := agentusage.New("codex", "codex", "gpt-5.6-luna", "xhigh", "prompt").WithProviderUsage(agentusage.ProviderUsage{Scope: agentusage.UsageScopeCall, InputTokens: 10, OutputTokens: 5, TotalTokens: 15}, "test")
	return agentexec.AgentResult{Text: text, SessionID: session, Resumed: resumed, Usage: usage}
}
