package confluencesource

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func TestConfluenceValidationAndConflictErrorsPreserveProductSentinels(t *testing.T) {
	validation := NewConfluenceValidationError("confluence_page_too_large", "safe validation message")
	if !errors.Is(validation, producterror.ErrInvalidInput) {
		t.Fatal("validation error does not preserve ErrInvalidInput")
	}
	conflict := NewConfluenceConflictError("confluence_version_changed", "safe conflict message")
	if !errors.Is(conflict, producterror.ErrConflict) {
		t.Fatal("conflict error does not preserve ErrConflict")
	}
	var got *ConfluenceError
	if !errors.As(fmt.Errorf("wrapped: %w", validation), &got) || got != validation {
		t.Fatal("errors.As did not recover the typed ConfluenceError")
	}
}

func TestConfluenceErrorDetailsRejectsTypedNil(t *testing.T) {
	var typedNil *ConfluenceError
	if details, ok := ConfluenceErrorDetails(typedNil); ok || details != nil {
		t.Fatalf("ConfluenceErrorDetails(typed nil) = %#v, %v; want nil, false", details, ok)
	}
	if got := ConfluenceSafeErrorMessage(typedNil); got != "Confluence 요청을 완료하지 못했습니다. 연결 상태와 권한을 확인하세요." {
		t.Fatalf("safe typed-nil message = %q", got)
	}
}

func TestConfluenceHTTPErrorsNormalizeAllHTTPClasses(t *testing.T) {
	tests := []struct {
		status   int
		category string
		code     string
		message  string
	}{
		{401, ConfluenceErrorCategoryAuth, ConfluenceErrorCodeUnauthorized, "Confluence 인증이 만료되었거나 유효하지 않습니다. 연결을 다시 인증하세요."},
		{403, ConfluenceErrorCategoryPermission, ConfluenceErrorCodeForbidden, "Confluence 권한 또는 OAuth scope가 부족합니다. 연결 권한과 페이지 접근 권한을 확인하세요."},
		{404, ConfluenceErrorCategoryNotFound, ConfluenceErrorCodeNotFound, "Confluence 사이트 또는 페이지를 찾을 수 없습니다. cloud id와 page id를 확인하세요."},
		{429, ConfluenceErrorCategoryRateLimited, ConfluenceErrorCodeRateLimited, "Confluence 요청이 제한되었습니다. 잠시 후 다시 시도하세요."},
		{400, ConfluenceErrorCategoryUpstream, ConfluenceErrorCodeUpstream, "Confluence 요청을 완료하지 못했습니다. 연결 상태와 권한을 확인하세요."},
		{500, ConfluenceErrorCategoryUpstream, ConfluenceErrorCodeUpstream, "Confluence 서비스가 요청을 처리하지 못했습니다. 잠시 후 다시 시도하세요."},
	}
	for _, test := range tests {
		err := NewConfluenceHTTPError(test.status, " 7 ", " GET /api/v2/pages/{page_id} ")
		if err.HTTPStatus != test.status || err.Category != test.category || err.Code != test.code || err.UserMessage != test.message {
			t.Fatalf("status %d normalized as %#v", test.status, err)
		}
		if err.RetryAfter != "7" || err.Operation != "GET /api/v2/pages/{page_id}" {
			t.Fatalf("status %d metadata = %#v", test.status, err)
		}
	}
}

func TestConfluenceJSONContainsOnlySafeFields(t *testing.T) {
	err := NewConfluenceTransportError("GET /api/v2/pages/{page_id}", errors.New("provider-secret"))
	encoded, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	text := string(encoded)
	for _, want := range []string{`"category":"confluence_upstream"`, `"code":"confluence_upstream_error"`, `"http_status":502`, `"operation":"GET /api/v2/pages/{page_id}"`, `"message":"Confluence 요청을 보내지 못했습니다. 연결 상태와 권한을 확인하세요."`} {
		if !containsJSONFragment(text, want) {
			t.Fatalf("JSON %q missing %q", text, want)
		}
	}
	if containsJSONFragment(text, "provider-secret") || containsJSONFragment(text, "cause") {
		t.Fatalf("JSON exposed private cause: %q", text)
	}
}

func containsJSONFragment(value, fragment string) bool {
	return len(value) >= len(fragment) && stringContains(value, fragment)
}

func stringContains(value, fragment string) bool {
	for i := 0; i+len(fragment) <= len(value); i++ {
		if value[i:i+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
