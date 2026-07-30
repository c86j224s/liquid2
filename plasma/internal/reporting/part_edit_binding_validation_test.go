package reporting_test

import (
	"errors"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestValidatePartEditBindingRejectsIncompleteDurableContract(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*reporting.PartEditBinding)
	}{
		{name: "missing previous provider", mutate: func(value *reporting.PartEditBinding) { value.PreviousProviderSessionID = "" }},
		{name: "missing mcp mode", mutate: func(value *reporting.PartEditBinding) { value.MCPMode = "" }},
		{name: "missing session policy", mutate: func(value *reporting.PartEditBinding) { value.ReportSessionPolicy = "" }},
		{name: "missing guidance profile", mutate: func(value *reporting.PartEditBinding) { value.GenerationGuidanceProfile = "" }},
		{name: "missing session chain", mutate: func(value *reporting.PartEditBinding) { value.SessionChainKind = "" }},
		{name: "missing report plan session", mutate: func(value *reporting.PartEditBinding) { value.ReportPlanSessionID = "" }},
		{name: "missing fork source", mutate: func(value *reporting.PartEditBinding) { value.ForkSourceAgentSessionID = "" }},
		{name: "requirement event without hash", mutate: func(value *reporting.PartEditBinding) {
			value.RequirementMapEventID = "evt_requirements"
			value.RequirementMapHash = ""
		}},
		{name: "requirement hash without event", mutate: func(value *reporting.PartEditBinding) {
			value.RequirementMapEventID = ""
			value.RequirementMapHash = "hash"
		}},
		{name: "previous provider differs", mutate: func(value *reporting.PartEditBinding) { value.PreviousProviderSessionID = "provider-other" }},
		{name: "provider is report plan", mutate: func(value *reporting.PartEditBinding) {
			value.ProviderSessionID = value.ReportPlanSessionID
			value.PreviousProviderSessionID = value.ProviderSessionID
		}},
		{name: "fork source differs", mutate: func(value *reporting.PartEditBinding) { value.ForkSourceAgentSessionID = "provider-other" }},
		{name: "idempotency differs", mutate: func(value *reporting.PartEditBinding) { value.IdempotencyKey = "part-edit-key" }},
		{name: "edited artifact is source", mutate: func(value *reporting.PartEditBinding) { value.EditedArtifactID = value.SourceArtifactID }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			binding := partEditBinding()
			tc.mutate(&binding)
			if err := reporting.ValidatePartEditBinding(binding); !errors.Is(err, app.ErrInvalidInput) {
				t.Fatalf("error=%v, want invalid input", err)
			}
		})
	}
}

func TestValidatePartEditBindingPreservesOptionalNormalizedMetadata(t *testing.T) {
	binding := partEditBinding()
	binding.AgentExecutor = " CODEX "
	binding.AgentModel = ""
	binding.AgentReasoningEffort = ""
	binding.AgentSelectionSource = ""
	binding.ReportSessionPolicySelection = ""
	binding.GenerationGuidanceSHA256 = ""
	binding.RequirementMapEventID = ""
	binding.RequirementMapHash = ""
	if err := reporting.ValidatePartEditBinding(binding); err != nil {
		t.Fatalf("optional normalized metadata rejected: %v", err)
	}
}
