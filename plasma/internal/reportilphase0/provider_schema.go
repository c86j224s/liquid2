package reportilphase0

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

const (
	providerSchemaSourceSelection      = "il_source_selection"
	providerSchemaNarrative            = "il_narrative"
	providerSchemaDocument             = "il_document"
	providerSchemaAuthorEvidenceRepair = "il_author_evidence_repair"
	providerSchemaFlow                 = "il_flow"
)

var structuredOutputKeywords = map[string]struct{}{
	"$defs": {}, "$ref": {}, "$schema": {}, "additionalProperties": {}, "anyOf": {},
	"const": {}, "enum": {}, "items": {}, "maxItems": {}, "maxLength": {}, "maximum": {},
	"minItems": {}, "minLength": {}, "minimum": {}, "pattern": {}, "properties": {}, "required": {},
	"type": {},
}

func providerNarrativeSchemaBytes(contractID, documentID string) []byte {
	value, err := providerSchemaValue(providerSchemaNarrative)
	if err != nil {
		panic(fmt.Sprintf("build provider schema for %s: %v", providerSchemaNarrative, err))
	}
	properties := value["$defs"].(map[string]any)["narrative"].(map[string]any)["properties"].(map[string]any)
	properties["contract_id"] = map[string]any{"type": "string", "const": contractID}
	properties["document_id"] = map[string]any{"type": "string", "const": documentID}
	return marshalProviderSchema(providerSchemaNarrative, value)
}

func providerDocumentSchemaBytes(narrative Narrative, catalog reportilcontract.SourceCatalog) []byte {
	return marshalProviderSchema(providerSchemaDocument, providerDocumentDraftSchema(narrative, catalog))
}

func providerFlowSchemaBytes(document Document) []byte {
	return marshalProviderSchema(providerSchemaFlow, providerFlowDraftSchema(document))
}

func nonblankStringSchema() map[string]any {
	return map[string]any{"type": "string", "pattern": "^[\\s\\S]*\\S[\\s\\S]*$"}
}

func nonemptyStringSchema() map[string]any {
	return map[string]any{"type": "string", "pattern": "^[\\s\\S]+$"}
}

func marshalProviderSchema(stage string, value map[string]any) []byte {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("encode provider schema for %s: %v", stage, err))
	}
	return raw
}

func providerSchemaValue(stage string) (map[string]any, error) {
	switch stage {
	case providerSchemaNarrative:
		component, err := readSchemaObject("schemas/narrative-contract.experimental.v1.schema.json")
		if err != nil {
			return nil, err
		}
		return providerRootReference("narrative", component), nil
	default:
		return nil, fmt.Errorf("unknown provider schema stage %q", stage)
	}
}

func readSchemaObject(path string) (map[string]any, error) {
	raw, err := schemaFiles.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("decode schema %s: %w", path, err)
	}
	return value, nil
}

func nullableSchema(value any) map[string]any {
	return map[string]any{"anyOf": []any{value, map[string]any{"type": "null"}}}
}

func providerRootReference(name string, source map[string]any) map[string]any {
	component := cloneObject(source)
	defs := map[string]any{name: transformSchema(component)}
	if sourceDefs, ok := source["$defs"].(map[string]any); ok {
		for definitionName, definition := range sourceDefs {
			defs[definitionName] = transformSchema(cloneAny(definition))
		}
	}
	delete(defs[name].(map[string]any), "$defs")
	return map[string]any{
		"$ref":  "#/$defs/" + name,
		"$defs": defs,
	}
}

func transformSchema(value any) any {
	switch value := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(value))
		for key, nested := range value {
			switch key {
			case "$schema", "$id", "title", "contentEncoding", "minLength", "maxLength", "minProperties", "maxProperties", "uniqueItems":
				continue
			case "properties", "$defs":
				children, ok := nested.(map[string]any)
				if !ok {
					result[key] = transformSchema(nested)
					continue
				}
				transformed := make(map[string]any, len(children))
				for name, child := range children {
					transformed[name] = transformSchema(child)
				}
				result[key] = transformed
			default:
				result[key] = transformSchema(nested)
			}
		}
		if _, ok := result["const"]; ok {
			if _, ok := result["type"]; !ok {
				result["type"] = constType(result["const"])
			}
		}
		if _, ok := result["enum"]; ok {
			if _, hasType := result["type"]; !hasType {
				if values, ok := result["enum"].([]any); ok && len(values) > 0 {
					result["type"] = constType(values[0])
				}
			}
		}
		if properties, ok := result["properties"].(map[string]any); ok {
			originalRequired := map[string]bool{}
			if required, ok := result["required"].([]any); ok {
				for _, item := range required {
					if name, ok := item.(string); ok {
						originalRequired[name] = true
					}
				}
			}
			for name, nested := range properties {
				if !originalRequired[name] {
					properties[name] = map[string]any{
						"anyOf": []any{nested, map[string]any{"type": "null"}},
					}
				}
			}
			names := make([]string, 0, len(properties))
			for name := range properties {
				names = append(names, name)
			}
			sort.Strings(names)
			required := make([]any, len(names))
			for index, name := range names {
				required[index] = name
			}
			result["required"] = required
			result["additionalProperties"] = false
		}
		return result
	case []any:
		result := make([]any, len(value))
		for i, nested := range value {
			result[i] = transformSchema(nested)
		}
		return result
	default:
		return value
	}
}

func constType(value any) string {
	switch value.(type) {
	case bool:
		return "boolean"
	case string:
		return "string"
	case float64:
		return "number"
	case nil:
		return "null"
	default:
		return "string"
	}
}

func cloneObject(value map[string]any) map[string]any {
	return cloneAny(value).(map[string]any)
}

func cloneAny(value any) any {
	switch value := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(value))
		for key, nested := range value {
			result[key] = cloneAny(nested)
		}
		return result
	case []any:
		result := make([]any, len(value))
		for i, nested := range value {
			result[i] = cloneAny(nested)
		}
		return result
	default:
		return value
	}
}

func lintProviderSchema(raw []byte) error {
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return fmt.Errorf("decode provider schema: %w", err)
	}
	defs, ok := value["$defs"].(map[string]any)
	if !ok {
		return fmt.Errorf("structured output schema requires root $defs")
	}
	return lintProviderSchemaNode(value, defs, "#", true)
}

func lintProviderSchemaNode(value any, defs map[string]any, path string, schemaNode bool) error {
	switch value := value.(type) {
	case map[string]any:
		if schemaNode {
			for key := range value {
				if _, ok := structuredOutputKeywords[key]; !ok {
					return fmt.Errorf("unsupported structured output keyword at %s: %s", path, key)
				}
			}
		}
		if reference, ok := value["$ref"].(string); ok {
			const prefix = "#/$defs/"
			if len(reference) <= len(prefix) || reference[:len(prefix)] != prefix {
				return fmt.Errorf("unresolved structured output reference at %s: %s", path, reference)
			}
			if _, ok := defs[reference[len(prefix):]]; !ok {
				return fmt.Errorf("unresolved structured output reference at %s: %s", path, reference)
			}
		}
		if _, hasConst := value["const"]; hasConst {
			if _, hasType := value["type"]; !hasType {
				return fmt.Errorf("const-only structured output node at %s", path)
			}
		}
		if typeName, _ := value["type"].(string); typeName == "object" {
			if value["additionalProperties"] != false {
				return fmt.Errorf("structured output object must be closed at %s", path)
			}
		}
		if properties, ok := value["properties"].(map[string]any); ok {
			required, ok := value["required"].([]any)
			if !ok {
				return fmt.Errorf("structured output properties must all be required at %s", path)
			}
			requiredSet := map[string]bool{}
			for _, item := range required {
				name, ok := item.(string)
				if !ok {
					return fmt.Errorf("structured output required entry is not a string at %s", path)
				}
				requiredSet[name] = true
			}
			if len(requiredSet) != len(properties) {
				return fmt.Errorf("structured output properties must all be required at %s", path)
			}
			for name := range properties {
				if !requiredSet[name] {
					return fmt.Errorf("structured output property %q is not required at %s", name, path)
				}
			}
		}
		for key, nested := range value {
			switch key {
			case "properties", "$defs":
				children, ok := nested.(map[string]any)
				if !ok {
					continue
				}
				for name, child := range children {
					if err := lintProviderSchemaNode(child, defs, path+"/"+key+"/"+name, true); err != nil {
						return err
					}
				}
			case "required", "type", "enum", "const", "additionalProperties", "$ref", "$schema":
				continue
			default:
				if err := lintProviderSchemaNode(nested, defs, path+"/"+key, true); err != nil {
					return err
				}
			}
		}
	case []any:
		for index, nested := range value {
			if err := lintProviderSchemaNode(nested, defs, fmt.Sprintf("%s/%d", path, index), schemaNode); err != nil {
				return err
			}
		}
	}
	return nil
}
