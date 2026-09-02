package agentexec

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/agentcapability"
	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

func TestApplyResearchCapabilityProfile(t *testing.T) {
	profile := agentcapability.Research()
	codex, err := applyCapabilityProfile(AgentRequest{
		CapabilityProfile: profile.ID,
		ProfileRevision:   profile.Revision,
		CodexConfig:       []string{"features.apps=false"},
	}, "codex")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range codexResearchProfileConfig {
		if countString(codex.CodexConfig, want) != 1 {
			t.Fatalf("Codex config %q count = %d in %#v", want, countString(codex.CodexConfig, want), codex.CodexConfig)
		}
	}
	if codex.IgnoreUserConfig {
		t.Fatal("Codex research profile must preserve provider connection config")
	}

	claude, err := applyCapabilityProfile(AgentRequest{
		CapabilityProfile: profile.ID,
		ProfileRevision:   profile.Revision,
	}, "claude")
	if err != nil {
		t.Fatal(err)
	}
	if !claude.IgnoreUserConfig || claude.DisableTools {
		t.Fatalf("unexpected Claude research profile: %#v", claude)
	}
}

func TestApplyGoalDraftCapabilityProfileDisablesProductTools(t *testing.T) {
	profile := agentcapability.GoalDraft()
	for _, provider := range []string{"codex", "claude"} {
		t.Run(provider, func(t *testing.T) {
			req, err := applyCapabilityProfile(AgentRequest{
				CapabilityProfile: profile.ID,
				ProfileRevision:   profile.Revision,
				ExtraMCPTools:     []string{"plasma.sources.read"},
			}, provider)
			if err != nil {
				t.Fatal(err)
			}
			if !req.DisableTools || !req.EphemeralSession || !req.ReplaceMCPTools || len(req.ExtraMCPTools) != 0 {
				t.Fatalf("unexpected goal draft profile: %#v", req)
			}
			if provider == "claude" && !req.IgnoreUserConfig {
				t.Fatal("Claude goal draft must isolate user config")
			}
		})
	}
}

func TestReportILCapabilityProfileUsesExactIsolatedRequestFields(t *testing.T) {
	profile := agentcapability.ReportIL()
	req, err := applyCapabilityProfile(AgentRequest{
		CapabilityProfile: profile.ID,
		ProfileRevision:   profile.Revision,
		ExtraMCPTools:     []string{"plasma.sources.read"},
	}, "codex")
	if err != nil {
		t.Fatal(err)
	}
	if req.CapabilityProfile != agentcapability.ProfileReportILV1 || req.ProfileRevision != agentcapability.RevisionV1 || !req.DisableTools || !req.IgnoreUserConfig || !req.EphemeralSession || !req.ReplaceMCPTools || len(req.ExtraMCPTools) != 0 {
		t.Fatalf("unexpected report IL capability request: %#v", req)
	}
	wantConfig := append([]string(nil), codexResearchProfileConfig...)
	if len(req.CodexConfig) != len(wantConfig) {
		t.Fatalf("report IL Codex config length changed: %#v", req.CodexConfig)
	}
	for i, want := range wantConfig {
		if req.CodexConfig[i] != want {
			t.Fatalf("report IL Codex config[%d] = %q, want %q; full=%#v", i, req.CodexConfig[i], want, req.CodexConfig)
		}
	}
}

func TestReportILSourceCapabilityProfileRequiresCodexAndExactTools(t *testing.T) {
	catalog := testAgentSourceCatalog(t)
	binding := reportilcontract.SourceAccessBinding{
		PendingEventID: "evt_pending", Stage: "il_narrative", Attempt: 1,
		EditorialMemoryArtifactID: "art_editorial_memory", EditorialMemorySHA256: strings.Repeat("d", 64),
		Catalog: catalog,
	}
	profile := agentcapability.ReportILSource()
	request := AgentRequest{
		CapabilityProfile: profile.ID,
		ProfileRevision:   profile.Revision,
		ExtraMCPTools:     []string{reportilcontract.EditorialMemoryReadTool},
		ReportILSources:   &binding,
	}
	if _, err := applyCapabilityProfile(request, "claude"); err == nil {
		t.Fatal("non-Codex report IL source profile was accepted")
	}
	request.ReportILSources = nil
	if _, err := applyCapabilityProfile(request, "codex"); err == nil {
		t.Fatal("missing source binding was accepted")
	}
	request.ReportILSources = &binding
	bad := request
	bad.ExtraMCPTools = []string{"ambient.tool"}
	if _, err := applyCapabilityProfile(bad, "codex"); err == nil {
		t.Fatal("ambient report IL tool was accepted")
	}
	resolved, err := applyCapabilityProfile(request, "codex")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.DisableTools || !resolved.IgnoreUserConfig || !resolved.EphemeralSession || !resolved.ReplaceMCPTools || !reflect.DeepEqual(resolved.ExtraMCPTools, []string{reportilcontract.EditorialMemoryReadTool}) {
		t.Fatalf("report IL source profile widened capability: %#v", resolved)
	}
	for _, want := range codexReportILProfileConfig {
		if countString(resolved.CodexConfig, want) != 1 {
			t.Fatalf("report IL source config %q count = %d", want, countString(resolved.CodexConfig, want))
		}
	}
}

func TestReportILSourceCapabilityProfileAllowsEditorialMemoryExactQuoteRegistration(t *testing.T) {
	catalog := testAgentSourceCatalog(t)
	binding := reportilcontract.SourceAccessBinding{
		PendingEventID: "evt_pending", Stage: "il_editorial_memory", Attempt: 1,
		MaxCallBytes: reportilcontract.DefaultSourceReadMaxBytes,
		MaxReadBytes: reportilcontract.DefaultSourceAttemptReadBytes,
		Catalog:      catalog,
	}
	tools := []string{
		reportilcontract.SourceListTool,
		reportilcontract.SourceReadTool,
		reportilcontract.SourceQuoteRegisterTool,
		reportilcontract.EditorialMemoryStartTool,
		reportilcontract.EditorialMemoryAppendTool,
		reportilcontract.EditorialMemoryReadTool,
		reportilcontract.EditorialMemoryFinalizeTool,
	}
	profile := agentcapability.ReportILSource()
	resolved, err := applyCapabilityProfile(AgentRequest{
		CapabilityProfile: profile.ID,
		ProfileRevision:   profile.Revision,
		ExtraMCPTools:     tools,
		ReportILSources:   &binding,
	}, "codex")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.DisableTools || !resolved.IgnoreUserConfig || !resolved.EphemeralSession ||
		!resolved.ReplaceMCPTools || !reflect.DeepEqual(resolved.ExtraMCPTools, tools) {
		t.Fatalf("editorial-memory source profile changed exact tool surface: %#v", resolved)
	}
}

func TestReportILSourceCapabilityProfileRejectsRawSourceToolsDownstream(t *testing.T) {
	catalog := testAgentSourceCatalog(t)
	binding := reportilcontract.SourceAccessBinding{
		PendingEventID: "evt_pending", Stage: "il_narrative", Attempt: 1,
		EditorialMemoryArtifactID: "art_editorial_memory", EditorialMemorySHA256: strings.Repeat("d", 64),
		Catalog: catalog,
	}
	profile := agentcapability.ReportILSource()
	for _, tool := range []string{reportilcontract.SourceListTool, reportilcontract.SourceReadTool, reportilcontract.SourceQuoteRegisterTool} {
		_, err := applyCapabilityProfile(AgentRequest{
			CapabilityProfile: profile.ID,
			ProfileRevision:   profile.Revision,
			ExtraMCPTools: []string{
				reportilcontract.EditorialMemoryReadTool,
				tool,
			},
			ReportILSources: &binding,
		}, "codex")
		if err == nil {
			t.Fatalf("downstream report IL stage accepted raw source tool %q", tool)
		}
	}
}

func TestReportILSourceCapabilityProfilePreservesExplicitMemoryReadOnlyNarrative(t *testing.T) {
	catalog := testAgentSourceCatalog(t)
	binding := reportilcontract.SourceAccessBinding{
		PendingEventID: "evt_pending", Stage: "il_narrative", Attempt: 1,
		EditorialMemoryArtifactID: "art_editorial_memory", EditorialMemorySHA256: strings.Repeat("d", 64),
		Catalog: catalog,
	}
	profile := agentcapability.ReportILSource()
	resolved, err := applyCapabilityProfile(AgentRequest{
		CapabilityProfile: profile.ID,
		ProfileRevision:   profile.Revision,
		ExtraMCPTools: []string{
			reportilcontract.EditorialMemoryReadTool,
		},
		ReportILSources: &binding,
	}, "codex")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(resolved.ExtraMCPTools, []string{
		reportilcontract.EditorialMemoryReadTool,
	}) {
		t.Fatalf("explicit report-first source tools were widened: %#v", resolved.ExtraMCPTools)
	}
}

func TestReportUnverifiedCapabilityProfileUsesOnlyMissionSourceReads(t *testing.T) {
	profile := agentcapability.ReportUnverified()
	request := AgentRequest{
		CapabilityProfile: profile.ID,
		ProfileRevision:   profile.Revision,
		ExtraMCPTools:     []string{"ambient.tool"},
	}
	if _, err := applyCapabilityProfile(request, "claude"); err == nil {
		t.Fatal("non-Codex unverified report profile was accepted")
	}
	resolved, err := applyCapabilityProfile(request, "codex")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.DisableTools || !resolved.IgnoreUserConfig || !resolved.EphemeralSession || !resolved.PreserveResponseWhitespace || !resolved.ReplaceMCPTools || !reflect.DeepEqual(resolved.ExtraMCPTools, []string{mcptools.ToolSourcesList, mcptools.ToolSourcesRead}) || resolved.ReportILSources != nil {
		t.Fatalf("unverified report profile widened capability: %#v", resolved)
	}
	for _, want := range codexResearchProfileConfig {
		if countString(resolved.CodexConfig, want) != 1 {
			t.Fatalf("unverified report config %q count = %d", want, countString(resolved.CodexConfig, want))
		}
	}
}

func TestReportILSourceBindingReachesCodexMCPWithAmbientToolsReplaced(t *testing.T) {
	catalog := testAgentSourceCatalog(t)
	binding := reportilcontract.SourceAccessBinding{
		PendingEventID: "evt_pending", Stage: "il_flow", Attempt: 2,
		MaxCallBytes: reportilcontract.DefaultSourceReadMaxBytes, MaxReadBytes: reportilcontract.DefaultSourceAttemptReadBytes,
		Catalog: catalog,
	}
	req := AgentRequest{
		MissionID: "mis_agent_source", ToolSessionID: "ses_source_attempt", AgentExecutor: "codex",
		ReplaceMCPTools: true,
		ExtraMCPTools:   []string{reportilcontract.SourceListTool, reportilcontract.SourceReadTool},
		ReportILSources: &binding,
	}
	args := CodexMCPArgsForRequest([]string{"mcp", "-db", "/tmp/test.db", "-enabled-tool", "ambient.tool"}, req)
	joined := strings.Join(args, "\n")
	if strings.Contains(joined, "ambient.tool") || strings.Count(joined, "-enabled-tool") != 2 || strings.Count(joined, reportilcontract.SourceListTool) != 1 || strings.Count(joined, reportilcontract.SourceReadTool) != 1 {
		t.Fatalf("Codex MCP tool allowlist = %#v", args)
	}
	if !strings.Contains(joined, "-mission-id\nmis_agent_source") || !strings.Contains(joined, "-agent-session-id\nses_source_attempt") || !strings.Contains(joined, "-agent-executor\ncodex") {
		t.Fatalf("Codex MCP attempt binding = %#v", args)
	}
	encoded := argValueAfter(args, "-report-il-source-binding-json")
	if encoded == "" || !strings.Contains(encoded, catalog.SHA256) || !strings.Contains(encoded, `"stage":"il_flow"`) || !strings.Contains(encoded, `"attempt":2`) {
		t.Fatalf("Codex MCP source binding = %#v", args)
	}
}

func testAgentSourceCatalog(t *testing.T) reportilcontract.SourceCatalog {
	t.Helper()
	catalog, err := reportilcontract.SealSourceCatalog(reportilcontract.SourceCatalog{
		MissionID: "mis_agent_source",
		Sources: []reportilcontract.SourceCatalogEntry{{
			SourceKey: "source_001", SnapshotID: "src_agent_source",
			SnapshotReceipt: reportilcontract.SourceSnapshotReceipt("src_agent_source", strings.Repeat("a", 64)),
			ContentHash:     strings.Repeat("a", 64), RetrievalPolicy: "snapshot_only",
			Artifacts:      []reportilcontract.SourceCatalogArtifact{{ArtifactID: "art_agent_source", SHA256: strings.Repeat("a", 64), ByteSize: 5, MediaType: "text/plain"}},
			ReadableSHA256: strings.Repeat("b", 64), ReadableBytes: 5, Extraction: "stored_text",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func TestWithCapabilityProfilePreservesForkInterfacesAndOverridesRequests(t *testing.T) {
	delegate := &capabilityProfileTestExecutor{}
	profile := agentcapability.Research()
	executor := WithCapabilityProfile(delegate, profile)

	forker, canFork := executor.(AgentSessionForker)
	readiness, canCheckFork := executor.(AgentSessionForkReadiness)
	if !canFork || !canCheckFork {
		t.Fatalf("profile wrapper lost fork interfaces: fork=%t readiness=%t", canFork, canCheckFork)
	}
	if _, err := forker.ForkSession(context.Background(), "source-session"); err != nil {
		t.Fatal(err)
	}
	if err := readiness.CheckForkSession(context.Background(), "source-session"); err != nil {
		t.Fatal(err)
	}
	if _, err := executor.Run(context.Background(), AgentRequest{
		CapabilityProfile: agentcapability.ProfileLegacyV1,
		ProfileRevision:   agentcapability.RevisionV1,
	}); err != nil {
		t.Fatal(err)
	}
	if delegate.request.CapabilityProfile != profile.ID || delegate.request.ProfileRevision != profile.Revision {
		t.Fatalf("profile wrapper did not override request: %#v", delegate.request)
	}
}

type capabilityProfileTestExecutor struct {
	request AgentRequest
}

func (executor *capabilityProfileTestExecutor) Run(_ context.Context, req AgentRequest) (AgentResult, error) {
	executor.request = req
	return AgentResult{Text: "ok", SessionID: "session"}, nil
}

func (executor *capabilityProfileTestExecutor) ForkSession(_ context.Context, sourceSessionID string) (AgentSessionForkResult, error) {
	return AgentSessionForkResult{SessionID: "fork", SourceSessionID: sourceSessionID}, nil
}

func (executor *capabilityProfileTestExecutor) CheckForkSession(context.Context, string) error {
	return nil
}

func countString(values []string, target string) int {
	count := 0
	for _, value := range values {
		if value == target {
			count++
		}
	}
	return count
}
