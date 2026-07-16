package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/version"
)

const defaultMCPProtocolVersion = "2024-11-05"

type rpcMessage struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type promptGetParams struct {
	Name string `json:"name"`
}

// ServeStdio exposes a Plasma MCP server over newline-delimited JSON-RPC stdio.
func ServeStdio(ctx context.Context, input io.Reader, output io.Writer, server *Server) error {
	if server == nil {
		return fmt.Errorf("mcp server is required")
	}
	decoder := json.NewDecoder(input)
	encoder := json.NewEncoder(output)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		var message rpcMessage
		if err := decoder.Decode(&message); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if !message.hasID() {
			continue
		}
		if id, ok := message.validRequestEnvelope(); !ok {
			if err := encoder.Encode(rpcFailure(id, -32600, "invalid request")); err != nil {
				return err
			}
			continue
		}
		response := handleRPC(ctx, server, message)
		if err := encoder.Encode(response); err != nil {
			return err
		}
	}
}

func (message rpcMessage) hasID() bool {
	trimmed := strings.TrimSpace(string(message.ID))
	return trimmed != "" && trimmed != "null"
}

func (message rpcMessage) validRequestEnvelope() (json.RawMessage, bool) {
	if !validRPCID(message.ID) {
		return json.RawMessage(`null`), false
	}
	if message.JSONRPC != "2.0" {
		return message.ID, false
	}
	return message.ID, true
}

func validRPCID(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return false
	}
	var value any
	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.UseNumber()
	if decoder.Decode(&value) != nil {
		return false
	}
	switch value.(type) {
	case string, json.Number:
		return true
	default:
		return false
	}
}

func handleRPC(ctx context.Context, server *Server, message rpcMessage) rpcResponse {
	switch message.Method {
	case "initialize":
		return rpcResult(message.ID, server.initializeResult(message.Params))
	case "ping":
		return rpcResult(message.ID, map[string]any{})
	case "tools/list":
		return rpcResult(message.ID, map[string]any{"tools": mcpTools(server.ListTools())})
	case "tools/call":
		return rpcResult(message.ID, callMCPTool(ctx, server, message.Params))
	case "resources/list":
		return rpcResult(message.ID, map[string]any{"resources": []any{}})
	case "prompts/list":
		return rpcResult(message.ID, map[string]any{"prompts": []map[string]string{{
			"name":        "plasma.research.workflow",
			"description": "Short Plasma research workflow guidance without mission data.",
		}}})
	case "prompts/get":
		return rpcResult(message.ID, getPrompt(message.Params))
	default:
		return rpcFailure(message.ID, -32601, "method not found")
	}
}

func (server *Server) initializeResult(params json.RawMessage) map[string]any {
	protocolVersion := defaultMCPProtocolVersion
	var input struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	if len(params) > 0 {
		_ = json.Unmarshal(params, &input)
	}
	if strings.TrimSpace(input.ProtocolVersion) != "" {
		protocolVersion = strings.TrimSpace(input.ProtocolVersion)
	}
	return map[string]any{
		"protocolVersion": protocolVersion,
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    "plasma",
			"version": version.Version,
		},
		"instructions": server.instructions(),
	}
}

func (server *Server) instructions() string {
	instructions := "Plasma tools are mission-scoped. Sources are original pinned materials; agent answers, controller outputs, and reports are results or artifacts, not sources. Start with plasma.research.outline, narrow with plasma.research.list or plasma.research.grep, confirm with plasma.research.read, and use plasma.research.references when relationships matter. Grep matches and connector search results are candidates for reading and judgment; they are not accepted sources, evidence, or saved knowledge. Source snapshot creation remains user-review gated by the application."
	instructions += " Sources may be snapshot_only pinned artifacts or live_reference local_path sources. Local path sources must be attached from configured allowlisted roots by root_id and relative_path only; absolute filesystem paths are not valid tool inputs. Reading a live local_path source records a bounded source.observed event with metadata, not a new source body snapshot."
	instructions += " PDF sources are original documents; read them through Plasma tools, which return bounded extracted text and extraction metadata rather than raw PDF bytes."
	instructions += " When a result or report depends on live local_path material, cite the observation_event_id plus observed_at, relative_path, sha256, and git metadata when available; do not cite only the source id."
	instructions += " Soft-removed and superseded sources are hidden from plasma.sources.list by default and should not be used for new reading or reporting unless explicitly requested for audit/history review."
	instructions += " In the C1 default loop, use MCP/source read tools to answer the user directly. When you find original material worth user review, use plasma.sources.candidates.propose to record it as a review candidate only. When proposing a plasma.sources.search result, copy source_uri into url and title into title so connector names such as Confluence page titles are preserved. Do not create evidence, claim, confidence, or proposal records unless the server was explicitly started for the legacy research loop."
	instructions += " Workflow control tools can request, inspect, or stop bounded mission workflow runs; start queues ledger work for the current user turn and bound agent executor, and does not invoke the provider inside the MCP call."
	instructions += " Mission metadata editing is user-owned: call plasma.mission.update only for an explicit user request, never as autonomous cleanup or steering. Plasma-spawned research agents do not receive this tool by default."
	instructions += " Tool calls made through a mission-bound server are logged as mcp.tool.called ledger events for user-visible debugging and report-generation trace review."
	binding := server.binding
	if binding.MissionID != "" {
		instructions += " This MCP server is bound to mission_id " + binding.MissionID + "."
	}
	if binding.AgentSessionID != "" {
		instructions += " Use session_id " + binding.AgentSessionID + " only when a read or explicitly enabled legacy tool requires a session-bound producer."
	}
	if binding.CurrentUserEventID != "" {
		instructions += " Workflow start requests from this MCP server are deferred until turn.user event " + binding.CurrentUserEventID + " has a terminal agent response."
	}
	if binding.AgentExecutor != "" {
		instructions += " Workflow start requests must use the bound agent_executor " + binding.AgentExecutor + " or omit agent_executor."
	}
	return instructions
}

func getPrompt(params json.RawMessage) any {
	var input promptGetParams
	if len(params) > 0 {
		_ = json.Unmarshal(params, &input)
	}
	if strings.TrimSpace(input.Name) != "plasma.research.workflow" {
		return map[string]any{"description": "Unknown prompt.", "messages": []any{}}
	}
	return map[string]any{
		"description": "Short Plasma research workflow guidance without mission data.",
		"messages": []map[string]any{{
			"role": "user",
			"content": map[string]string{
				"type": "text",
				"text": "Use Plasma as a mission ledger. Start with plasma.research.outline, list or grep candidates, read bounded objects or source chunks, and use references to follow source, raw artifact, observation, and ledger-event links. Treat grep results as candidates only. Do not treat agent results as sources, and do not rely on prompt-packed mission data. Local path sources must come from configured roots by root_id and relative_path, never arbitrary absolute paths. For live local_path material, ground claims in source.observed metadata such as observation_event_id, observed_at, relative_path, sha256, and git state when available. If you find new original material worth user review, call plasma.sources.candidates.propose; this records a review candidate only and does not create a source snapshot. When proposing a plasma.sources.search result, copy source_uri into url and title into title.",
			},
		}},
	}
}

func mcpTools(tools []ToolDefinition) []mcpToolDefinition {
	output := make([]mcpToolDefinition, 0, len(tools))
	for _, tool := range tools {
		output = append(output, mcpToolDefinition{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		})
	}
	return output
}

func callMCPTool(ctx context.Context, server *Server, params json.RawMessage) map[string]any {
	var input toolCallParams
	if len(params) > 0 && json.Unmarshal(params, &input) != nil {
		return mcpToolResult(errorResult("", "", "protocol", "tools/call params are invalid", false, nil))
	}
	if len(input.Arguments) == 0 {
		input.Arguments = json.RawMessage(`{}`)
	}
	return mcpToolResult(server.Call(ctx, ToolCall{Name: input.Name, Arguments: input.Arguments}))
}

func mcpToolResult(result ToolResult) map[string]any {
	encoded, err := json.Marshal(result)
	if err != nil {
		encoded = []byte(`{"error":{"error_kind":"internal","message":"failed to encode tool result","retryable":false}}`)
	}
	return map[string]any{
		"content": []map[string]string{{
			"type": "text",
			"text": string(encoded),
		}},
		"isError": result.Error != nil,
	}
}

func rpcResult(id json.RawMessage, result any) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Result: result}
}

func rpcFailure(id json.RawMessage, code int, message string) rpcResponse {
	return rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: message},
	}
}
