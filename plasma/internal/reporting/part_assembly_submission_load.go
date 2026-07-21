package reporting

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func LoadPartAssemblySubmission(ctx context.Context, store PartAssemblySubmissionStore, binding PartAssemblyBinding) (PartAssemblySubmission, bool, error) {
	binding = normalizePartAssemblyBinding(binding)
	if err := ValidatePartAssemblyBinding(binding); err != nil {
		return PartAssemblySubmission{}, false, err
	}
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return PartAssemblySubmission{}, false, err
	}
	var found PartAssemblySubmission
	count := 0
	for _, event := range events {
		if event.EventType != PartAssemblySubmittedEventType {
			continue
		}
		var payload partAssemblySubmittedPayload
		if json.Unmarshal(event.Payload, &payload) != nil {
			continue
		}
		if !partAssemblySubmissionMatches(payload, binding) {
			continue
		}
		count++
		found = PartAssemblySubmission{Event: event, Binding: binding, Assembly: normalizePartAssembly(payload.Assembly, binding.SectionCount)}
	}
	if count == 0 {
		return PartAssemblySubmission{}, false, nil
	}
	if count != 1 {
		return PartAssemblySubmission{}, false, fmt.Errorf("%w: multiple part assembly submissions match binding", app.ErrConflict)
	}
	return found, true, nil
}

func ValidatePartAssemblyBinding(value PartAssemblyBinding) error {
	value = normalizePartAssemblyBinding(value)
	if value.MissionID == "" || value.PendingEventID == "" || value.PlanEventID == "" || value.ToolSessionID == "" || value.AgentExecutor == "" || value.PartIndex < 1 || value.SectionCount < 1 {
		return fmt.Errorf("%w: part assembly binding is incomplete", app.ErrInvalidInput)
	}
	if value.Producer.Type != "agent_session" || value.Producer.ID != value.ToolSessionID {
		return fmt.Errorf("%w: part assembly producer binding mismatch", app.ErrInvalidInput)
	}
	return nil
}

func normalizePartAssemblyBinding(value PartAssemblyBinding) PartAssemblyBinding {
	value.MissionID = strings.TrimSpace(value.MissionID)
	value.PendingEventID = strings.TrimSpace(value.PendingEventID)
	value.PlanEventID = strings.TrimSpace(value.PlanEventID)
	value.ToolSessionID = strings.TrimSpace(value.ToolSessionID)
	value.ProviderSessionID = strings.TrimSpace(value.ProviderSessionID)
	value.PreviousProviderSessionID = strings.TrimSpace(value.PreviousProviderSessionID)
	value.AgentExecutor = strings.TrimSpace(value.AgentExecutor)
	value.AgentModel = strings.TrimSpace(value.AgentModel)
	value.AgentReasoningEffort = strings.TrimSpace(value.AgentReasoningEffort)
	value.AgentSelectionSource = strings.TrimSpace(value.AgentSelectionSource)
	value.MCPMode = strings.TrimSpace(value.MCPMode)
	value.ReportSessionPolicy = strings.TrimSpace(value.ReportSessionPolicy)
	value.ReportSessionPolicySelection = strings.TrimSpace(value.ReportSessionPolicySelection)
	value.PostReportHumanize = strings.TrimSpace(value.PostReportHumanize)
	value.GenerationGuidanceProfile = strings.TrimSpace(value.GenerationGuidanceProfile)
	value.GenerationGuidanceSHA256 = strings.TrimSpace(value.GenerationGuidanceSHA256)
	value.SessionChainKind = strings.TrimSpace(value.SessionChainKind)
	value.PreReportResearchSessionID = strings.TrimSpace(value.PreReportResearchSessionID)
	value.ReportPlanSessionID = strings.TrimSpace(value.ReportPlanSessionID)
	value.ForkSourceAgentSessionID = strings.TrimSpace(value.ForkSourceAgentSessionID)
	value.Producer.Type = strings.TrimSpace(value.Producer.Type)
	value.Producer.ID = strings.TrimSpace(value.Producer.ID)
	return value
}

func normalizePartAssembly(value PartAssembly, sectionCount int) PartAssembly {
	value.Intro = strings.TrimSpace(value.Intro)
	value.Closing = strings.TrimSpace(value.Closing)
	transitions := make([]PartTransition, 0, len(value.Transitions))
	seen := map[int]bool{}
	for _, transition := range value.Transitions {
		transition.Markdown = strings.TrimSpace(transition.Markdown)
		if transition.AfterSectionIndex <= 0 || transition.AfterSectionIndex >= sectionCount || transition.Markdown == "" || seen[transition.AfterSectionIndex] {
			continue
		}
		seen[transition.AfterSectionIndex] = true
		transitions = append(transitions, transition)
	}
	value.Transitions = transitions
	return value
}

func partAssemblySubmissionMatches(payload partAssemblySubmittedPayload, binding PartAssemblyBinding) bool {
	return payload.Kind == PartAssemblySubmittedKind &&
		payload.PendingEventID == binding.PendingEventID &&
		payload.PlanEventID == binding.PlanEventID &&
		payload.ToolSessionID == binding.ToolSessionID &&
		payload.PartIndex == binding.PartIndex &&
		payload.SectionCount == binding.SectionCount &&
		payload.AgentExecutor == binding.AgentExecutor &&
		payload.AgentModel == binding.AgentModel &&
		payload.AgentReasoningEffort == binding.AgentReasoningEffort
}
