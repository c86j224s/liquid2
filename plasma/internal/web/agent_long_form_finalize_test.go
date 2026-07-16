package web

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"

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

func assertLongFormFinalizeArgs(t *testing.T, args []string, binding reporting.LongFormFinalizeBinding) {
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
	joined := strings.Join(args, "\n")
	if !strings.Contains(joined, mcp.ToolReportLongFormFinalize) || !strings.Contains(joined, "-agent-session-id\nses_final") {
		t.Fatalf("missing enabled tool or tool session: %#v", args)
	}
}
