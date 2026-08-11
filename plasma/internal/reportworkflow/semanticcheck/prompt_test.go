package semanticcheck

import (
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/internal/finaledit"
)

func TestDirectionDoesNotEnterSemanticPrompt(t *testing.T) {
	input := finaledit.Input{Title: "Report", MissionID: "mis_semantic", DirectionHint: "DIRECTION_SENTINEL"}
	got := prompt(input, reporting.FinalEditStageBinding{Stage: Stage}, "draft_1", 1)
	if strings.Contains(got, input.DirectionHint) {
		t.Fatalf("semantic prompt leaked direction:\n%s", got)
	}
	if !strings.Contains(got, finaledit.StageSubmittedSentinel) {
		t.Fatalf("semantic prompt lost sentinel:\n%s", got)
	}
}
