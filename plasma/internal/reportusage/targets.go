package reportusage

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

type Target struct {
	EventType                string
	Surface                  string
	AgentSessionID           string
	PreviousAgentSessionID   string
	ForkSourceAgentSessionID string
	AgentExecutor            string
	AgentModel               string
	AgentReasoningEffort     string
}

type targetSpec struct {
	surface string
	stage   string
}

var targetSpecs = map[string]targetSpec{
	"report.requirements.mapped":                            {surface: "report_requirements"},
	"report.part.edited":                                    {surface: "report_part_edit"},
	"report.final_edit.writer.submitted":                    {surface: "report_final_write", stage: "final_write"},
	"report.final_edit.reader.submitted":                    {surface: "report_reader_edit", stage: "reader_edit"},
	"report.final_edit.style.submitted":                     {surface: "report_style_edit", stage: "style_edit"},
	"report.final_edit.gate.submitted":                      {surface: "report_corrective_gate", stage: "corrective_gate"},
	"report.final_edit.style_semantic_validation.submitted": {surface: "report_style_semantic_validation", stage: "style_semantic_validation"},
	"report.final_edit.evidence_gate.submitted":             {surface: "report_evidence_gate", stage: "evidence_gate"},
}

func TargetForEvent(event ledger.Event) (Target, bool, error) {
	eventType := strings.TrimSpace(event.EventType)
	spec, ok := targetSpecs[eventType]
	if !ok {
		return Target{}, false, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return Target{}, false, err
	}
	if spec.stage != "" && stringField(payload, "stage") != spec.stage {
		return Target{}, false, fmt.Errorf("report usage target %s has invalid stage", event.EventID)
	}
	target := Target{
		EventType: eventType, Surface: spec.surface,
		AgentSessionID:           stringField(payload, "provider_session_id"),
		PreviousAgentSessionID:   stringField(payload, "previous_provider_session_id"),
		ForkSourceAgentSessionID: stringField(payload, "fork_source_agent_session_id"),
		AgentExecutor:            stringField(payload, "agent_executor"), AgentModel: stringField(payload, "agent_model"),
		AgentReasoningEffort: stringField(payload, "agent_reasoning_effort"),
	}
	if eventType == "report.requirements.mapped" {
		previous := stringField(payload, "previous_provider_session_id")
		target.AgentSessionID, target.PreviousAgentSessionID = previous, previous
	} else if spec.stage != "" {
		target.PreviousAgentSessionID = target.AgentSessionID
	}
	return target, true, nil
}

func stringField(payload map[string]any, key string) string {
	value, _ := payload[key].(string)
	return strings.TrimSpace(value)
}
