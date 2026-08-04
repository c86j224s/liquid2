package agentexec

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/mcp"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestLongFormFinalizeBindingReachesCodexAndClaudeMCPConfigs(t *testing.T) {
	binding := reporting.LongFormFinalizeBinding{MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan", ArtifactID: "art_final", ToolSessionID: "ses_final", IdempotencyKey: "key", ProviderSessionID: "provider-1", AgentExecutor: "codex"}
	req := AgentRequest{MissionID: binding.MissionID, ToolSessionID: binding.ToolSessionID, AgentExecutor: "codex", ExtraMCPTools: []string{mcp.ToolReportLongFormFinalize}, LongFormFinalize: &binding}
	base := []string{"mcp", "-db", "/tmp/test.db"}
	codexArgs := codexMCPArgsForRequest(base, req)
	assertLongFormFinalizeArgs(t, codexArgs, binding)

	claude := ClaudeExecutor{MCPServer: ClaudeMCPServer{Name: "plasma", Command: "plasma", Args: base}}
	path, cleanup, err := claude.writeMCPConfig(req)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		MCPServers map[string]struct {
			Args []string `json:"args"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(content, &config); err != nil {
		t.Fatal(err)
	}
	assertLongFormFinalizeArgs(t, config.MCPServers["plasma"].Args, binding)
}

func TestFinalEditStageBindingReachesCodexAndClaudeMCPConfigs(t *testing.T) {
	binding := reporting.FinalEditStageBinding{MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan", Stage: reporting.FinalEditStageReader, SourceArtifactID: "art_source", EditedArtifactID: "art_reader", Filename: "report.md", ToolSessionID: "ses_reader", ProviderSessionID: "provider-reader", AgentExecutor: "codex"}
	req := AgentRequest{MissionID: binding.MissionID, ToolSessionID: binding.ToolSessionID, AgentExecutor: "codex", ExtraMCPTools: []string{mcp.ToolReportLongFormReaderEditStart}, FinalEditStage: &binding}
	base := []string{"mcp", "-db", "/tmp/test.db"}
	codexArgs := codexMCPArgsForRequest(base, req)
	assertFinalEditStageArgs(t, codexArgs, binding)

	claude := ClaudeExecutor{MCPServer: ClaudeMCPServer{Name: "plasma", Command: "plasma", Args: base}}
	path, cleanup, err := claude.writeMCPConfig(req)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		MCPServers map[string]struct {
			Args []string `json:"args"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(content, &config); err != nil {
		t.Fatal(err)
	}
	assertFinalEditStageArgs(t, config.MCPServers["plasma"].Args, binding)
}

func TestCorrectiveGateIncludesStageAndFinalBindings(t *testing.T) {
	final := reporting.LongFormFinalizeBinding{
		MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan", ArtifactID: "art_final",
		ToolSessionID: "ses_gate", ProviderSessionID: "provider-gate", AgentExecutor: "codex",
		CompositionStrategy: reporting.LongFormCompositionNarrativeEdit,
	}
	stage := reporting.FinalEditStageBinding{
		MissionID: final.MissionID, PendingEventID: final.PendingEventID, PlanEventID: final.PlanEventID,
		Stage: reporting.FinalEditStageGate, SourceArtifactID: "art_reader", EditedArtifactID: final.ArtifactID,
		ToolSessionID: final.ToolSessionID, ProviderSessionID: final.ProviderSessionID, AgentExecutor: final.AgentExecutor,
	}
	req := AgentRequest{
		MissionID: final.MissionID, ToolSessionID: final.ToolSessionID, AgentExecutor: final.AgentExecutor,
		ExtraMCPTools: []string{mcp.ToolReportLongFormEditStart}, LongFormFinalize: &final, FinalEditStage: &stage,
	}
	args := codexMCPArgsForRequest([]string{"mcp", "-db", "/tmp/test.db"}, req)
	assertLongFormFinalizeBindingEncoded(t, args, final)
	assertFinalEditStageArgs(t, args, stage)
}

func assertLongFormFinalizeArgs(t *testing.T, args []string, binding reporting.LongFormFinalizeBinding) {
	t.Helper()
	assertLongFormFinalizeBindingEncoded(t, args, binding)
	joined := strings.Join(args, "\n")
	if !strings.Contains(joined, mcp.ToolReportLongFormFinalize) || !strings.Contains(joined, "-agent-session-id\nses_final") {
		t.Fatalf("missing enabled tool or tool session: %#v", args)
	}
}

func assertLongFormFinalizeBindingEncoded(t *testing.T, args []string, binding reporting.LongFormFinalizeBinding) {
	t.Helper()
	index := slices.Index(args, "-report-long-form-finalize-binding-json")
	if index < 0 || index+1 >= len(args) {
		t.Fatalf("missing finalization binding flag: %#v", args)
	}
	var decoded reporting.LongFormFinalizeBinding
	if err := json.Unmarshal([]byte(args[index+1]), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.MissionID != binding.MissionID || decoded.ToolSessionID != binding.ToolSessionID || decoded.ProviderSessionID != binding.ProviderSessionID {
		t.Fatalf("binding changed in provider config: %#v", decoded)
	}
}

func assertFinalEditStageArgs(t *testing.T, args []string, binding reporting.FinalEditStageBinding) {
	t.Helper()
	index := slices.Index(args, "-report-final-edit-stage-binding-json")
	if index < 0 || index+1 >= len(args) {
		t.Fatalf("missing final edit stage binding flag: %#v", args)
	}
	var decoded reporting.FinalEditStageBinding
	if err := json.Unmarshal([]byte(args[index+1]), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.MissionID != binding.MissionID || decoded.ToolSessionID != binding.ToolSessionID || decoded.Stage != binding.Stage {
		t.Fatalf("stage binding changed in provider config: %#v", decoded)
	}
	wantTool := mcp.ToolReportLongFormReaderEditStart
	if binding.Stage == reporting.FinalEditStageStyle {
		wantTool = mcp.ToolReportLongFormStyleEditStart
	} else if binding.Stage == reporting.FinalEditStageGate {
		wantTool = mcp.ToolReportLongFormEditStart
	}
	joined := strings.Join(args, "\n")
	if !strings.Contains(joined, wantTool) || !strings.Contains(joined, "-agent-session-id\n"+binding.ToolSessionID) {
		t.Fatalf("missing enabled stage tool or tool session: %#v", args)
	}
}

func TestPartAssemblyBindingReachesCodexAndClaudeMCPConfigs(t *testing.T) {
	binding := reporting.PartAssemblyBinding{
		MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan", ToolSessionID: "ses_part",
		ProviderSessionID: "provider-1", PartIndex: 1, SectionCount: 3, AgentExecutor: "codex",
		Producer: app.Producer{Type: "agent_session", ID: "ses_part"},
	}
	req := AgentRequest{
		MissionID: binding.MissionID, ToolSessionID: binding.ToolSessionID, AgentExecutor: "codex",
		ExtraMCPTools: []string{mcp.ToolReportPartAssemblyStart, mcp.ToolReportPartAssemblySubmit},
		PartAssembly:  &binding,
	}
	base := []string{"mcp", "-db", "/tmp/test.db"}
	codexArgs := codexMCPArgsForRequest(base, req)
	assertPartAssemblyArgs(t, codexArgs, binding)

	claude := ClaudeExecutor{MCPServer: ClaudeMCPServer{Name: "plasma", Command: "plasma", Args: base}}
	path, cleanup, err := claude.writeMCPConfig(req)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		MCPServers map[string]struct {
			Args []string `json:"args"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(content, &config); err != nil {
		t.Fatal(err)
	}
	assertPartAssemblyArgs(t, config.MCPServers["plasma"].Args, binding)
}

func assertPartAssemblyArgs(t *testing.T, args []string, binding reporting.PartAssemblyBinding) {
	t.Helper()
	index := slices.Index(args, "-report-part-assembly-binding-json")
	if index < 0 || index+1 >= len(args) {
		t.Fatalf("missing part assembly binding flag: %#v", args)
	}
	var decoded reporting.PartAssemblyBinding
	if err := json.Unmarshal([]byte(args[index+1]), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.MissionID != binding.MissionID || decoded.ToolSessionID != binding.ToolSessionID || decoded.PartIndex != binding.PartIndex || decoded.SectionCount != binding.SectionCount {
		t.Fatalf("binding changed in provider config: %#v", decoded)
	}
	joined := strings.Join(args, "\n")
	if !strings.Contains(joined, mcp.ToolReportPartAssemblyStart) || !strings.Contains(joined, mcp.ToolReportPartAssemblySubmit) || !strings.Contains(joined, "-agent-session-id\nses_part") {
		t.Fatalf("missing enabled tool or tool session: %#v", args)
	}
}
