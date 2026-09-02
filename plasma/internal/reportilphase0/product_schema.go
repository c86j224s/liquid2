package reportilphase0

import (
	"encoding/json"
	"fmt"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

func productDocumentSchemaText() string {
	var response struct {
		Properties struct {
			Document json.RawMessage `json:"document"`
		} `json:"properties"`
	}
	raw, err := schemaFiles.ReadFile("schemas/report-flow-response.experimental.v2.schema.json")
	if err != nil {
		panic(fmt.Sprintf("read product flow response schema: %v", err))
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		panic(fmt.Sprintf("decode product flow response schema: %v", err))
	}
	if len(response.Properties.Document) == 0 {
		panic("product flow response schema has no document property")
	}
	return string(response.Properties.Document)
}

func compileProductDocumentSchema() (*jsonschema.Schema, error) {
	var response struct {
		Properties struct {
			Document json.RawMessage `json:"document"`
		} `json:"properties"`
	}
	raw, err := schemaFiles.ReadFile("schemas/report-flow-response.experimental.v2.schema.json")
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, err
	}
	return compileProductSchema(string(response.Properties.Document), "urn:plasma:report-il-product:v1")
}

func compileProductSchema(raw, resource string) (*jsonschema.Schema, error) {
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, err
	}
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()
	compiler.AssertContent()
	if err := compiler.AddResource(resource, value); err != nil {
		return nil, err
	}
	return compiler.Compile(resource)
}

func validateProductDocumentShape(value Document) error {
	if len(value.Extensions) != 0 {
		return fmt.Errorf("product document excludes extensions")
	}
	if len(value.Assets) > maxProductFigures {
		return fmt.Errorf("product document exceeds the figure asset ceiling")
	}
	providerShape := value
	providerShape.Assets = nil
	providerShape.Blocks = make([]Block, 0, len(value.Blocks))
	figures := 0
	for _, block := range value.Blocks {
		if block.Kind == "raw" || len(block.Extension) != 0 {
			return fmt.Errorf("product document excludes raw and extension blocks")
		}
		if block.Kind == "figure" {
			figures++
			if block.Figure == nil {
				return fmt.Errorf("product figure is incomplete")
			}
			continue
		}
		if block.Figure != nil {
			return fmt.Errorf("product document has a figure payload on a non-figure block")
		}
		providerShape.Blocks = append(providerShape.Blocks, block)
	}
	if figures != len(value.Assets) || figures > maxProductFigures {
		return fmt.Errorf("product figures and assets must be one-to-one within the ceiling")
	}
	compiled, err := compileProductDocumentSchema()
	if err != nil {
		return err
	}
	generic, err := schemaValue(providerShape)
	if err != nil {
		return err
	}
	return compiled.Validate(generic)
}
