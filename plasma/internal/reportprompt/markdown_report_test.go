package reportprompt

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestMarkdownReportPromptSHACharacterization(t *testing.T) {
	rigor := RigorProfile{
		Level:        "balanced",
		Label:        "균형형",
		Description:  "검증된 사실을 중심에 두고, 유용한 약한 신호는 맥락과 한계를 붙여 사용합니다.",
		Instructions: "- Anchor the main storyline on source-backed facts.\n- Keep weak signals caveated.",
	}
	plan := reporting.ReportPlan{
		Summary: "Use source-backed material.",
		Sections: []reporting.ReportPlanSection{{
			Title:   "Section",
			Purpose: "Cover evidence.",
		}},
	}
	cases := []struct {
		name string
		text string
		sha  string
	}{
		{
			name: "one_take",
			text: OneTakeMarkdownReportPrompt("Quick", "mis_1", "ses_1", rigor, ProfileNarrativeContract),
			sha:  "aba0b9a959310fb7757826ccf683e91e5a6057c7dd6fd83a31eba8aab3232123",
		},
		{
			name: "planned_plan",
			text: MarkdownReportPlanPrompt("Report", "mis_1", "ses_1", "evt_1", "key_1", rigor, ProfileNarrativeContract),
			sha:  "8d5540eebf3d7fbd581e27eecce070fb5a628599c30b64a9ff5c1ecedcffc93a",
		},
		{
			name: "planned_write",
			text: PlannedMarkdownReportPrompt("Report", "mis_1", "ses_1", rigor, plan, ProfileNarrativeContract),
			sha:  "963be3d29e377e1a2611abb9f1c4c19c821beaa090368d5bf6be48ec852b19e6",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sum := sha256.Sum256([]byte(tc.text))
			got := hex.EncodeToString(sum[:])
			if got != tc.sha {
				t.Fatalf("prompt SHA changed: got %s want %s", got, tc.sha)
			}
		})
	}
}
