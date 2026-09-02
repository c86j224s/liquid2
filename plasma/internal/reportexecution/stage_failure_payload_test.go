package reportexecution

import (
	"errors"
	"strings"
	"testing"
)

func TestStageFailurePayloadPreservesNestedSafeMetadata(t *testing.T) {
	failure := NewStageFailure("il_reader", "", -1, -1, failurePayloadError{payload: map[string]any{
		"failed_surface":        "reader_process",
		"agent_session_id":      "session_1",
		"raw_provider_response": "secret",
	}})

	payload := failure.FailurePayload()
	if payload["failed_surface"] != "reader_process" || payload["agent_session_id"] != "session_1" {
		t.Fatalf("nested safe metadata was not preserved: %#v", payload)
	}
	if _, ok := payload["raw_provider_response"]; ok {
		t.Fatalf("unknown nested metadata passed the allowlist: %#v", payload)
	}
}

func TestStageFailurePayloadAddsBoundedStoreDiagnostic(t *testing.T) {
	failure := NewStageFailure("il_store", "", -1, -1, errors.New(strings.Repeat("x", 600)))

	payload := failure.FailurePayload()
	detail, ok := payload["internal_failure_detail"].(string)
	if !ok || len(detail) != 500 {
		t.Fatalf("bounded store diagnostic = %#v", payload)
	}
}

func TestStageFailurePayloadDoesNotCopyProviderErrorIntoStoreDiagnostic(t *testing.T) {
	failure := NewStageFailure("il_store", "", -1, -1, failurePayloadError{payload: map[string]any{
		"failed_surface": "artifact_store",
	}})

	payload := failure.FailurePayload()
	if payload["failed_surface"] != "artifact_store" {
		t.Fatalf("nested safe metadata was not preserved: %#v", payload)
	}
	if _, ok := payload["internal_failure_detail"]; ok {
		t.Fatalf("provider error entered store diagnostic: %#v", payload)
	}
}

func TestStageFailurePayloadOmitsOtherRawCauses(t *testing.T) {
	failure := NewStageFailure("il_reader", "", -1, -1, errors.New("private provider detail"))
	if payload := failure.FailurePayload(); payload != nil {
		t.Fatalf("non-store raw cause entered failure payload: %#v", payload)
	}
}
