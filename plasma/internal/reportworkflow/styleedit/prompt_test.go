package styleedit

import (
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/finaledit"
)

func TestDirectionDoesNotEnterStylePrompt(t *testing.T) {
	input := finaledit.Input{Title: "Report", MissionID: "mis_style", DirectionHint: "DIRECTION_SENTINEL"}
	got := prompt(input, reporting.FinalEditStageBinding{Stage: Stage}, "draft_1", 1)
	if strings.Contains(got, input.DirectionHint) {
		t.Fatalf("style prompt leaked direction:\n%s", got)
	}
	if !strings.Contains(got, finaledit.StageSubmittedSentinel) {
		t.Fatalf("style prompt lost sentinel:\n%s", got)
	}
}
