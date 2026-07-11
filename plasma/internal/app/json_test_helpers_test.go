package app

import (
	"encoding/json"
	"reflect"
	"testing"
)

func assertJSONPayloadIncludes(t *testing.T, raw []byte, want map[string]any) {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}
	for key, value := range want {
		if !reflect.DeepEqual(got[key], value) {
			t.Fatalf("payload key %q mismatch: got %#v want %#v in %#v", key, got[key], value, got)
		}
	}
}
