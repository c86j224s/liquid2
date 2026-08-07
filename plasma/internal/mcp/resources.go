package mcp

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/mcpdocs"
)

const resourceNotFoundCode = -32002

type mcpResourceDefinition struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MIMEType    string `json:"mimeType,omitempty"`
}

type mcpResourceContent struct {
	URI      string `json:"uri"`
	MIMEType string `json:"mimeType,omitempty"`
	Text     string `json:"text"`
}

type resourceReadParams struct {
	URI string `json:"uri"`
}

// listMCPResources adapts the static public documentation catalog to the MCP
// resources/list wire shape. It must not inspect mission or runtime state.
func listMCPResources(id json.RawMessage, params json.RawMessage) rpcResponse {
	if !validResourceListParams(params) {
		return rpcFailure(id, -32602, "invalid params")
	}
	docs := mcpdocs.List()
	resources := make([]mcpResourceDefinition, 0, len(docs))
	for _, doc := range docs {
		resources = append(resources, mcpResourceDefinition{
			URI:         doc.URI,
			Name:        doc.Name,
			Description: doc.Description,
			MIMEType:    doc.MIMEType,
		})
	}
	return rpcResult(id, map[string]any{"resources": resources})
}

func validResourceListParams(params json.RawMessage) bool {
	if len(params) == 0 {
		return true
	}
	if strings.TrimSpace(string(params)) == "null" {
		return true
	}
	var input map[string]json.RawMessage
	if err := json.Unmarshal(params, &input); err != nil {
		return false
	}
	if len(input) == 0 {
		return true
	}
	cursor, ok := input["cursor"]
	if !ok || len(input) != 1 {
		return false
	}
	return strings.TrimSpace(string(cursor)) == `""`
}

// readMCPResource adapts one static public Markdown document to the MCP
// resources/read wire shape. Unknown resources use MCP's resource-not-found
// server error code, while malformed params remain JSON-RPC invalid params.
func readMCPResource(id json.RawMessage, params json.RawMessage) rpcResponse {
	var input resourceReadParams
	if len(params) == 0 || json.Unmarshal(params, &input) != nil {
		return rpcFailure(id, -32602, "invalid params")
	}
	uri := strings.TrimSpace(input.URI)
	if uri == "" {
		return rpcFailure(id, -32602, "invalid params")
	}
	if !validResourceURI(uri) {
		return rpcFailure(id, -32602, "invalid resource uri")
	}
	document, ok := mcpdocs.Read(uri)
	if !ok {
		return rpcFailure(id, resourceNotFoundCode, "resource not found")
	}
	return rpcResult(id, map[string]any{"contents": []mcpResourceContent{{
		URI:      document.URI,
		MIMEType: document.MIMEType,
		Text:     document.Text,
	}}})
}

func validResourceURI(raw string) bool {
	parsed, err := url.Parse(raw)
	return err == nil && parsed.Scheme != ""
}
