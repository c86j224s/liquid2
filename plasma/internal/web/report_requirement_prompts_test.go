package web

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	plasmamcp "github.com/c86j224s/liquid2/plasma/internal/mcp"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestReportRequirementPromptOwnsDirectionAfterFixedOutline(t *testing.T) {
	plan := agentSectionalReportPlan{Parts: []agentReportPart{{Title: "Part", Sections: []agentReportSection{{Title: "One", Purpose: "first purpose"}, {Title: "Two", Purpose: "comparison purpose"}}}}}
	binding := reporting.ReportRequirementMapBinding{
		MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan", ToolSessionID: "ses_tool",
		IdempotencyKey: "rrk_1", AgentExecutor: "codex", Producer: app.Producer{Type: "agent_session", ID: "ses_tool"},
	}
	prompt := agentReportRequirementMapPrompt("Report", "include a comparison table", plan, []string{"evt_user", "evt_pending"}, binding)
	for _, expected := range []string{"include a comparison table", "fixed", "Section 1.2: Two", "indices are 1-based", "indexed title and purpose", "evt_user", "every eligible event", "current pending event must appear", "unmapped_reason", "REQUIREMENTS_MAPPED", plasmamcp.ToolReportRequirementsSubmit} {
		if !strings.Contains(strings.ToLower(prompt), strings.ToLower(expected)) {
			t.Fatalf("requirement prompt missing %q:\n%s", expected, prompt)
		}
	}
	if strings.Contains(prompt, "modify, replace, extend, or reorder") == false {
		t.Fatal("requirement prompt does not protect the fixed outline")
	}
}

func TestReportRequirementToolsExposeOnlyEligibleEventReadAndSubmit(t *testing.T) {
	tools := reportRequirementMCPTools()
	if len(tools) != 2 || tools[0] != plasmamcp.ToolResearchRead || tools[1] != plasmamcp.ToolReportRequirementsSubmit {
		t.Fatalf("unexpected report requirement tools: %#v", tools)
	}
}

func TestSectionPromptReceivesOnlyAssignedRequirements(t *testing.T) {
	assigned := []reporting.ReportRequirement{{RequirementID: "req_table", Instruction: "include a comparison table", SourceEventIDs: []string{"evt_pending"}, Owner: &reporting.ReportRequirementOwner{PartIndex: 1, SectionIndex: 1}}}
	prompt := agentSectionDraftPromptWithRequirements("Report", "mis_1", "ses_1", reportRigorProfiles["balanced"], agentSectionalReportPlan{}, agentReportPart{Title: "Part"}, agentReportSection{Title: "Section"}, 0, 0, "", assigned)
	for _, expected := range []string{"req_table", "include a comparison table", "not factual evidence", "Do not pull requirements assigned to other Sections"} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("section prompt missing %q:\n%s", expected, prompt)
		}
	}
}

func TestAgentProvidersCarryReportRequirementMCPContext(t *testing.T) {
	binding := reporting.ReportRequirementMapBinding{
		MissionID: "mis_1", PendingEventID: "evt_pending", PlanEventID: "evt_plan", ToolSessionID: "ses_tool",
		PreviousProviderSessionID: "ses_plan", IdempotencyKey: "rrk_1", AgentExecutor: "codex",
		Producer: app.Producer{Type: "agent_session", ID: "ses_tool"},
	}
	request := AgentRequest{
		MissionID: "mis_1", ToolSessionID: "ses_tool", AgentExecutor: "codex", ReplaceMCPTools: true,
		ExtraMCPTools:      []string{plasmamcp.ToolReportRequirementsSubmit},
		ReportRequirements: &binding,
	}
	assertReportRequirementMCPArgs(t, codexMCPArgsForRequest([]string{"mcp"}, request), binding)

	binding.AgentExecutor = "claude"
	request.AgentExecutor = "claude"
	claude := ClaudeExecutor{MCPServer: ClaudeMCPServer{Name: "plasma", Command: "/tmp/plasma", Args: []string{"mcp"}}}
	path, cleanup, err := claude.writeMCPConfig(request)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		MCPServers map[string]struct {
			Args []string `json:"args"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatal(err)
	}
	assertReportRequirementMCPArgs(t, config.MCPServers["plasma"].Args, binding)
}

func assertReportRequirementMCPArgs(t *testing.T, args []string, binding reporting.ReportRequirementMapBinding) {
	t.Helper()
	joined := strings.Join(args, "\n")
	if !strings.Contains(joined, plasmamcp.ToolReportRequirementsSubmit) {
		t.Fatalf("requirement submit tool missing from MCP args: %#v", args)
	}
	if strings.Contains(joined, "-report-plan-") {
		t.Fatalf("requirement mapper leaked report-plan flags into MCP args: %#v", args)
	}
	encoded := reportRequirementArgValue(t, args, "-report-requirements-binding-json")
	var got reporting.ReportRequirementMapBinding
	if err := json.Unmarshal([]byte(encoded), &got); err != nil {
		t.Fatal(err)
	}
	if got.MissionID != binding.MissionID || got.PendingEventID != binding.PendingEventID || got.PlanEventID != binding.PlanEventID || got.ToolSessionID != binding.ToolSessionID || got.PreviousProviderSessionID != binding.PreviousProviderSessionID || got.IdempotencyKey != binding.IdempotencyKey || got.AgentExecutor != binding.AgentExecutor || got.Producer != binding.Producer {
		t.Fatalf("requirement binding was not preserved: %#v", got)
	}
}

func reportRequirementArgValue(t *testing.T, args []string, flag string) string {
	t.Helper()
	for index := 0; index < len(args)-1; index++ {
		if args[index] == flag {
			return args[index+1]
		}
	}
	t.Fatalf("missing %s in args %#v", flag, args)
	return ""
}
