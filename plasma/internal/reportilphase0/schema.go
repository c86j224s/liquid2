package reportilphase0

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed schemas/*.json
var schemaFiles embed.FS

const (
	narrativeSchemaURL    = "urn:plasma:report-narrative:experimental:v1"
	flowResponseSchemaURL = "urn:plasma:report-flow-response:experimental:v2"
)

type schemas struct {
	narrative    *jsonschema.Schema
	document     *jsonschema.Schema
	attestation  *jsonschema.Schema
	flowResponse *jsonschema.Schema
}

func loadSchemas() (schemas, error) {
	narrative, err := compileSchema("schemas/narrative-contract.experimental.v1.schema.json", narrativeSchemaURL)
	if err != nil {
		return schemas{}, err
	}
	document, err := compileSchema("schemas/report-il.experimental.v2.schema.json", "urn:plasma:report-il:experimental:v2")
	if err != nil {
		return schemas{}, err
	}
	attestation, err := compileSchema("schemas/flow-attestation.experimental.v2.schema.json", "urn:plasma:report-flow-attestation:experimental:v2")
	if err != nil {
		return schemas{}, err
	}
	flowResponse, err := compileSchema("schemas/report-flow-response.experimental.v2.schema.json", flowResponseSchemaURL)
	if err != nil {
		return schemas{}, err
	}
	return schemas{narrative: narrative, document: document, attestation: attestation, flowResponse: flowResponse}, nil
}

func compileSchema(path, resourceURL string) (*jsonschema.Schema, error) {
	raw, err := schemaFiles.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("decode embedded schema %s: %w", path, err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()
	compiler.AssertContent()
	if err := compiler.AddResource(resourceURL, value); err != nil {
		return nil, fmt.Errorf("add embedded schema %s: %w", path, err)
	}
	compiled, err := compiler.Compile(resourceURL)
	if err != nil {
		return nil, fmt.Errorf("compile embedded schema %s: %w", path, err)
	}
	return compiled, nil
}

func decodeStrictJSON[T any](raw []byte) (T, error) {
	var value T
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("trailing JSON value")
		}
		return value, err
	}
	return value, nil
}

func schemaValue(value any) (any, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, err
	}
	return generic, nil
}
