package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestStdioResourcesListAcceptsAbsentAndEmptyParams(t *testing.T) {
	server := NewServer(&fakeMCPService{})
	for _, test := range []struct {
		name   string
		params json.RawMessage
	}{
		{name: "absent"},
		{name: "null", params: json.RawMessage(`null`)},
		{name: "empty object", params: json.RawMessage(`{}`)},
		{name: "empty cursor", params: json.RawMessage(`{"cursor":""}`)},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := handleRPC(context.Background(), server, rpcMessage{
				ID:     json.RawMessage(`1`),
				Method: "resources/list",
				Params: test.params,
			})
			if response.Error != nil {
				t.Fatalf("resources/list failed: %#v", response.Error)
			}
			result := mustMarshalForTest(t, response.Result)
			if strings.Count(result, `"uri":"plasma://docs/mcp/`) != 5 {
				t.Fatalf("resources/list did not return the five static docs: %s", result)
			}
			if strings.Contains(result, ".ko.md") {
				t.Fatalf("resources/list exposed Korean counterpart files: %s", result)
			}
		})
	}
}

func TestStdioResourcesListRejectsMalformedParamsAndCursor(t *testing.T) {
	server := NewServer(&fakeMCPService{})
	for _, test := range []struct {
		name   string
		params json.RawMessage
	}{
		{name: "string params", params: json.RawMessage(`"invalid"`)},
		{name: "array params", params: json.RawMessage(`[]`)},
		{name: "unknown-only object", params: json.RawMessage(`{"limit":1}`)},
		{name: "cursor plus unknown field", params: json.RawMessage(`{"cursor":"","limit":1}`)},
		{name: "case-variant cursor field", params: json.RawMessage(`{"Cursor":""}`)},
		{name: "null cursor", params: json.RawMessage(`{"cursor":null}`)},
		{name: "numeric cursor", params: json.RawMessage(`{"cursor":1}`)},
		{name: "object cursor", params: json.RawMessage(`{"cursor":{}}`)},
		{name: "array cursor", params: json.RawMessage(`{"cursor":[]}`)},
		{name: "whitespace cursor", params: json.RawMessage(`{"cursor":" "}`)},
		{name: "non-empty cursor", params: json.RawMessage(`{"cursor":"next"}`)},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := handleRPC(context.Background(), server, rpcMessage{
				ID:     json.RawMessage(`1`),
				Method: "resources/list",
				Params: test.params,
			})
			if response.Error == nil || response.Error.Code != -32602 || response.Error.Message != "invalid params" {
				t.Fatalf("resources/list did not reject invalid params: %#v", response)
			}
		})
	}
}
