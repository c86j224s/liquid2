package app

import (
	"errors"
	"fmt"
	"strings"
)

const (
	ConfluenceErrorCategoryAuth        = "confluence_auth"
	ConfluenceErrorCategoryPermission  = "confluence_permission"
	ConfluenceErrorCategoryNotFound    = "confluence_not_found"
	ConfluenceErrorCategoryRateLimited = "confluence_rate_limited"
	ConfluenceErrorCategoryValidation  = "confluence_validation"
	ConfluenceErrorCategoryConflict    = "confluence_conflict"
	ConfluenceErrorCategoryUpstream    = "confluence_upstream"

	ConfluenceErrorCodeUnauthorized  = "confluence_unauthorized"
	ConfluenceErrorCodeForbidden     = "confluence_forbidden"
	ConfluenceErrorCodeNotFound      = "confluence_not_found"
	ConfluenceErrorCodeRateLimited   = "confluence_rate_limited"
	ConfluenceErrorCodeVersionDrift  = "confluence_version_changed"
	ConfluenceErrorCodeCloudMismatch = "confluence_cloud_mismatch"
	ConfluenceErrorCodePageMismatch  = "confluence_page_mismatch"
	ConfluenceErrorCodeTooLarge      = "confluence_page_too_large"
	ConfluenceErrorCodeTokenExpired  = "confluence_token_expired"
	ConfluenceErrorCodeRevoked       = "confluence_connection_revoked"
	ConfluenceErrorCodeUpstream      = "confluence_upstream_error"
)

// ConfluenceError는 애플리케이션 서비스 계층에서 호출자에게 안정적으로 노출하는 오류 타입이다. 사용자 메시지는 안전한 필드만 사용해야 한다.
type ConfluenceError struct {
	Category    string `json:"category"`
	Code        string `json:"code"`
	HTTPStatus  int    `json:"http_status,omitempty"`
	RetryAfter  string `json:"retry_after,omitempty"`
	Operation   string `json:"operation,omitempty"`
	UserMessage string `json:"message"`
	cause       error
}

// Error는 호출자에게 노출 가능한 안정적인 오류 문자열을 반환하며, 민감한 원문이나 provider 응답을 포함하지 않아야 한다.
func (err *ConfluenceError) Error() string {
	if err == nil {
		return ""
	}
	if strings.TrimSpace(err.UserMessage) != "" {
		return strings.TrimSpace(err.UserMessage)
	}
	if strings.TrimSpace(err.Code) != "" {
		return strings.TrimSpace(err.Code)
	}
	return "Confluence 요청을 완료하지 못했습니다."
}

// Unwrap은 상위 계층이 오류 원인을 검사할 수 있게 내부 오류를 돌려주되, 사용자 노출 메시지의 안전성 계약은 바꾸지 않는다.
func (err *ConfluenceError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.cause
}

// NewConfluenceValidationError는 사용자 입력 문제를 안정 오류 code와 HTTP 400으로 감싼다.
func NewConfluenceValidationError(code string, message string) *ConfluenceError {
	return &ConfluenceError{
		Category:    ConfluenceErrorCategoryValidation,
		Code:        strings.TrimSpace(code),
		HTTPStatus:  400,
		UserMessage: strings.TrimSpace(message),
		cause:       ErrInvalidInput,
	}
}

// NewConfluenceConflictError는 상태 충돌을 안정 오류 code와 HTTP 409로 감싼다.
func NewConfluenceConflictError(code string, message string) *ConfluenceError {
	return &ConfluenceError{
		Category:    ConfluenceErrorCategoryConflict,
		Code:        strings.TrimSpace(code),
		HTTPStatus:  409,
		UserMessage: strings.TrimSpace(message),
		cause:       ErrConflict,
	}
}

// NewConfluenceHTTPError는 upstream HTTP status를 사용자 행동이 가능한 오류 범주로 정규화한다.
func NewConfluenceHTTPError(status int, retryAfter string, operation string) *ConfluenceError {
	category := ConfluenceErrorCategoryUpstream
	code := ConfluenceErrorCodeUpstream
	message := "Confluence 요청을 완료하지 못했습니다. 연결 상태와 권한을 확인하세요."
	switch status {
	case 401:
		category = ConfluenceErrorCategoryAuth
		code = ConfluenceErrorCodeUnauthorized
		message = "Confluence 인증이 만료되었거나 유효하지 않습니다. 연결을 다시 인증하세요."
	case 403:
		category = ConfluenceErrorCategoryPermission
		code = ConfluenceErrorCodeForbidden
		message = "Confluence 권한 또는 OAuth scope가 부족합니다. 연결 권한과 페이지 접근 권한을 확인하세요."
	case 404:
		category = ConfluenceErrorCategoryNotFound
		code = ConfluenceErrorCodeNotFound
		message = "Confluence 사이트 또는 페이지를 찾을 수 없습니다. cloud id와 page id를 확인하세요."
	case 429:
		category = ConfluenceErrorCategoryRateLimited
		code = ConfluenceErrorCodeRateLimited
		message = "Confluence 요청이 제한되었습니다. 잠시 후 다시 시도하세요."
	}
	if status >= 500 {
		message = "Confluence 서비스가 요청을 처리하지 못했습니다. 잠시 후 다시 시도하세요."
	}
	return &ConfluenceError{
		Category:    category,
		Code:        code,
		HTTPStatus:  status,
		RetryAfter:  strings.TrimSpace(retryAfter),
		Operation:   strings.TrimSpace(operation),
		UserMessage: message,
	}
}

// NewConfluenceTransportError는 요청 전송 실패를 upstream 오류 범주로 감싸고 원인은 Unwrap에 남긴다.
func NewConfluenceTransportError(operation string, cause error) *ConfluenceError {
	return &ConfluenceError{
		Category:    ConfluenceErrorCategoryUpstream,
		Code:        ConfluenceErrorCodeUpstream,
		HTTPStatus:  502,
		Operation:   strings.TrimSpace(operation),
		UserMessage: "Confluence 요청을 보내지 못했습니다. 연결 상태와 권한을 확인하세요.",
		cause:       cause,
	}
}

// ConfluenceErrorDetails는 Confluence 오류에서 사용자에게 안전한 상세 필드만 추출한다.
func ConfluenceErrorDetails(err error) (*ConfluenceError, bool) {
	var confluenceErr *ConfluenceError
	if errors.As(err, &confluenceErr) && confluenceErr != nil {
		return confluenceErr, true
	}
	return nil, false
}

// ConfluenceErrorStatus는 Confluence 오류를 HTTP 상태 코드로 변환한다.
func ConfluenceErrorStatus(err error) int {
	if confluenceErr, ok := ConfluenceErrorDetails(err); ok && confluenceErr.HTTPStatus > 0 {
		return confluenceErr.HTTPStatus
	}
	if errors.Is(err, ErrInvalidInput) {
		return 400
	}
	if errors.Is(err, ErrConflict) {
		return 409
	}
	return 500
}

// ConfluenceSafeErrorMessage는 Confluence 오류의 사용자 노출 문구를 안전한 형태로 반환한다.
func ConfluenceSafeErrorMessage(err error) string {
	if confluenceErr, ok := ConfluenceErrorDetails(err); ok {
		return confluenceErr.Error()
	}
	return "Confluence 요청을 완료하지 못했습니다. 연결 상태와 권한을 확인하세요."
}

// ConfluenceHTTPErrorString은 Confluence HTTP 오류의 짧은 진단 문자열을 만든다.
func ConfluenceHTTPErrorString(status int, operation string) string {
	operation = strings.TrimSpace(operation)
	if operation == "" {
		return fmt.Sprintf("confluence connector request returned %d", status)
	}
	return fmt.Sprintf("confluence connector %s returned %d", operation, status)
}
