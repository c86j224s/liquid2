package reporting

import "testing"

func TestResolveModelSelection(t *testing.T) {
	tests := []struct {
		name   string
		input  ModelSelectionInput
		model  string
		effort string
		source string
		bad    bool
	}{
		{"request pair", ModelSelectionInput{Executor: "codex", RequestModel: " gpt-5.5 ", RequestReasoningEffort: " HIGH ", ReasoningEffortSupported: true}, "gpt-5.5", "high", AgentSelectionSourceExplicitRequest, false},
		{"explicit model uses its default", ModelSelectionInput{Executor: "codex", RequestModel: "gpt-5.5", SessionReasoningEffort: "high", ReasoningEffortSupported: true}, "gpt-5.5", "medium", AgentSelectionSourceExplicitRequest, false},
		{"effort with session model", ModelSelectionInput{Executor: "codex", RequestReasoningEffort: "low", SessionModel: "gpt-5.4", ReasoningEffortSupported: true}, "gpt-5.4", "low", AgentSelectionSourceExplicitRequest, false},
		{"session pair", ModelSelectionInput{Executor: "codex", SessionModel: "gpt-5.4", SessionReasoningEffort: "high", ReasoningEffortSupported: true}, "gpt-5.4", "high", AgentSelectionSourceMissionSession, false},
		{"provider pair", ModelSelectionInput{Executor: "codex", ProviderModel: "gpt-5.5", ProviderReasoningEffort: "low", ReasoningEffortSupported: true}, "gpt-5.5", "low", AgentSelectionSourceProviderDefault, false},
		{"partial session", ModelSelectionInput{Executor: "codex", SessionModel: "gpt-5.4", ProviderReasoningEffort: "medium", ReasoningEffortSupported: true}, "gpt-5.4", "medium", AgentSelectionSourceMissionSession, false},
		{"unknown codex", ModelSelectionInput{Executor: "codex", RequestModel: "future-codex", RequestReasoningEffort: "xhigh", ReasoningEffortSupported: true}, "future-codex", "xhigh", AgentSelectionSourceExplicitRequest, false},
		{"invalid known pair", ModelSelectionInput{Executor: "codex", RequestModel: "gpt-5.6-luna", RequestReasoningEffort: "ultra", ReasoningEffortSupported: true}, "", "", "", true},
		{"unsupported explicit effort", ModelSelectionInput{Executor: "claude", RequestModel: "claude-x", RequestReasoningEffort: "high"}, "", "", "", true},
		{"unsupported omitted effort", ModelSelectionInput{Executor: "claude", ProviderModel: "claude-x"}, "claude-x", "", AgentSelectionSourceProviderDefault, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ResolveModelSelection(test.input)
			if (err != nil) != test.bad {
				t.Fatalf("error = %v", err)
			}
			if !test.bad && (got.Model != test.model || got.ReasoningEffort != test.effort || got.Source != test.source) {
				t.Fatalf("got %#v", got)
			}
		})
	}
}
