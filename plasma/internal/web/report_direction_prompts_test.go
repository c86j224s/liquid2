package web

import (
	"os"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestReportDirectionPromptAllowlist(t *testing.T) {
	hint := "DIRECTION_SENTINEL"
	allowed := withReportDirection("base prompt", hint)
	if !strings.Contains(allowed, reporting.DirectionAdvisory) || !strings.Contains(allowed, hint) {
		t.Fatalf("allowed prompt = %q", allowed)
	}
	for name, prompt := range map[string]string{
		"patch": AgentReportPatchPrompt("t", "mis_1", "ses_1", "evt_1", "art_1", "edit", reporting.PatchRequest{}),
		"part":  agentPartAssemblyPrompt("t", "mis_1", "ses_1", reportRigorProfiles["balanced"], agentSectionalReportPlan{}, agentReportPart{}, nil, 0, ""),
		"final": agentLongFormFinalizePrompt("t", "mis_1", reportRigorProfiles["balanced"], agentSectionalReportPlan{}, nil, "", reporting.LongFormFinalizeBinding{ToolSessionID: "ses_1", PendingEventID: "evt_1", PlanEventID: "evt_2", IdempotencyKey: "key"}, 1, false, reporting.LongFormFinalizationHint{}),
	} {
		if strings.Contains(prompt, hint) || strings.Contains(prompt, reporting.DirectionAdvisory) {
			t.Fatalf("%s leaked direction", name)
		}
	}
	routes, err := os.ReadFile("report_routes.go")
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(string(routes), "withReportDirection("); count != 5 {
		t.Fatalf("expected five allowlisted prompt call sites, got %d", count)
	}
}
