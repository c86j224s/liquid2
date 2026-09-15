package workflow

import (
	"encoding/json"

	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
)

// Definitions returns workflow tools in the established root list order.
func Definitions() []wire.ToolDefinition {
	return []wire.ToolDefinition{
		{Name: mcptools.ToolWorkflowStart, Description: "Request a bounded Plasma workflow run for the bound mission. This queues work and does not call the provider inside the MCP tool.", InputSchema: schemaWorkflowStart},
		{Name: mcptools.ToolWorkflowStatus, Description: "Read shared workflow run status from the mission ledger projection.", InputSchema: schemaWorkflowStatus},
		{Name: mcptools.ToolWorkflowStop, Description: "Request that a bounded workflow run stop before the next step.", InputSchema: schemaWorkflowStop},
	}
}

var (
	schemaWorkflowStart  = objectSchema([]string{"mission_id", "instruction"}, workflowStartProperties())
	schemaWorkflowStatus = objectSchema([]string{"mission_id"}, workflowStatusProperties())
	schemaWorkflowStop   = objectSchema([]string{"mission_id", "workflow_run_id"}, workflowStopProperties())
)

func objectSchema(required []string, properties map[string]any) json.RawMessage {
	schema := map[string]any{"type": "object", "additionalProperties": false, "properties": properties}
	if len(required) > 0 {
		schema["required"] = required
	}
	encoded, err := json.Marshal(schema)
	if err != nil {
		panic(err)
	}
	return encoded
}

func workflowStartProperties() map[string]any {
	return map[string]any{
		"mission_id":                   prefixedStringSchema("mis_"),
		"instruction":                  stringSchema(),
		"workflow_run_id":              prefixedStringSchema("wfr_"),
		"step_instruction_mode":        enumSchema("layered"),
		"user_instruction_raw":         stringSchema(),
		"run_goal":                     stringSchema(),
		"agent_executor":               stringSchema(),
		"mcp_mode":                     enumSchema("auto", "explicit"),
		"max_steps":                    map[string]any{"type": "integer", "minimum": 1, "maximum": 20},
		"max_duration_ms":              map[string]any{"type": "integer", "minimum": 0, "maximum": 86400000},
		"stop_condition":               stringSchema(),
		"start_after_event_id":         prefixedStringSchema("evt_"),
		"requested_by_tool_session_id": prefixedStringSchema("ses_"),
	}
}

func workflowStatusProperties() map[string]any {
	return map[string]any{
		"mission_id":      prefixedStringSchema("mis_"),
		"workflow_run_id": prefixedStringSchema("wfr_"),
	}
}

func workflowStopProperties() map[string]any {
	return map[string]any{
		"mission_id":      prefixedStringSchema("mis_"),
		"workflow_run_id": prefixedStringSchema("wfr_"),
		"reason":          stringSchema(),
	}
}

func stringSchema() map[string]any { return map[string]any{"type": "string"} }

func enumSchema(values ...string) map[string]any {
	enum := make([]any, 0, len(values))
	for _, value := range values {
		enum = append(enum, value)
	}
	return map[string]any{"type": "string", "enum": enum}
}

func prefixedStringSchema(prefix string) map[string]any {
	return map[string]any{"type": "string", "pattern": "^" + prefix}
}
