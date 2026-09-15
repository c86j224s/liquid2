package mission

import (
	"encoding/json"

	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"github.com/c86j224s/liquid2/plasma/internal/mcptools"
)

// Definitions returns mission tools in the stable root list order.
func Definitions() []wire.ToolDefinition {
	return []wire.ToolDefinition{
		{Name: mcptools.ToolMissionGet, Description: "Read a Plasma mission projection.", InputSchema: schemaGet},
		{Name: mcptools.ToolMissionUpdate, Description: "Update supplied current mission metadata fields through the shared application service only when the user explicitly requests the edit.", InputSchema: schemaUpdate},
	}
}

var (
	schemaGet = objectSchema([]string{"mission_id"}, map[string]any{
		"mission_id": prefixedStringSchema("mis_"),
		"include":    arraySchema(stringSchema()),
	})
	schemaUpdate = missionUpdateSchema()
)

func missionUpdateSchema() json.RawMessage {
	properties := commonMutatingProperties()
	properties["producer"] = objectSchemaValue([]string{"type", "id"}, map[string]any{"type": map[string]any{"type": "string", "const": "user"}, "id": stringSchema()})
	properties["title"] = stringSchema()
	properties["objective"] = stringSchema()
	properties["scope"] = objectSchemaValue([]string{"included", "excluded"}, map[string]any{"included": arraySchema(stringSchema()), "excluded": arraySchema(stringSchema())})
	value := map[string]any{
		"type": "object", "additionalProperties": false,
		"required":   []string{"mission_id", "session_id", "idempotency_key", "producer"},
		"properties": properties,
		"anyOf":      []any{map[string]any{"required": []string{"title"}}, map[string]any{"required": []string{"objective"}}, map[string]any{"required": []string{"scope"}}},
	}
	encoded, _ := json.Marshal(value)
	return encoded
}

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

func commonMutatingProperties() map[string]any {
	return map[string]any{
		"mission_id":      prefixedStringSchema("mis_"),
		"session_id":      prefixedStringSchema("ses_"),
		"idempotency_key": stringSchema(),
		"producer": objectSchemaValue([]string{"type", "id"}, map[string]any{
			"type": map[string]any{"type": "string", "const": "agent_session"}, "id": prefixedStringSchema("ses_"),
		}),
	}
}

func objectSchemaValue(required []string, properties map[string]any) map[string]any {
	schema := map[string]any{"type": "object", "additionalProperties": false, "properties": properties}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func stringSchema() map[string]any { return map[string]any{"type": "string"} }
func prefixedStringSchema(prefix string) map[string]any {
	return map[string]any{"type": "string", "pattern": "^" + prefix}
}
func arraySchema(items any) map[string]any { return map[string]any{"type": "array", "items": items} }
